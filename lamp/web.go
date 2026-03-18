package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:embed web/*
var webAssets embed.FS

type apiError struct {
	Error string `json:"error"`
}

type statusResponse struct {
	On         bool `json:"on"`
	Brightness int  `json:"brightness"`
	ColorTemp  int  `json:"color_temp"`
}

func runWebServer(lamp *Lamp, addr string) error {
	mux := http.NewServeMux()
	staticFS, err := fs.Sub(webAssets, "web")
	if err != nil {
		return fmt.Errorf("failed to init web assets: %w", err)
	}
	staticHandler := http.FileServer(http.FS(staticFS))

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		s, err := lamp.GetStatus()
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, statusResponse{
			On:         s.On,
			Brightness: s.Brightness,
			ColorTemp:  s.ColorTemp,
		})
	})

	mux.HandleFunc("/api/power", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req struct {
			On bool `json:"on"`
		}
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}
		var err error
		if req.On {
			err = lamp.TurnOn()
		} else {
			err = lamp.TurnOff()
		}
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	mux.HandleFunc("/api/brightness", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req struct {
			Value int `json:"value"`
		}
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}
		if err := lamp.SetBrightness(req.Value); err != nil {
			writeJSONError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	mux.HandleFunc("/api/color_temp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req struct {
			Value int `json:"value"`
		}
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}
		if err := lamp.SetColorTemp(req.Value); err != nil {
			writeJSONError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	mux.HandleFunc("/api/scene", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req struct {
			Name       string `json:"name"`
			Brightness *int   `json:"brightness"`
		}
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}
		if err := lamp.SetScene(req.Name, req.Brightness); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	mux.Handle("/", staticHandler)

	log.Printf("Web 控制台已启动: %s", localURL(addr))
	log.Printf("如需局域网访问，请使用: %s", lanURLHint(addr))
	return http.ListenAndServe(addr, loggingMiddleware(mux))
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("无效 JSON 请求体: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return fmt.Errorf("无效 JSON 请求体: 仅允许一个 JSON 对象")
	}
	return nil
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, apiError{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func localURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "http://127.0.0.1:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	if strings.HasPrefix(addr, "localhost:") || strings.HasPrefix(addr, "127.0.0.1:") {
		return "http://" + addr
	}
	return "http://" + addr
}

func lanURLHint(addr string) string {
	port := "8080"
	if strings.Contains(addr, ":") {
		parts := strings.Split(addr, ":")
		port = parts[len(parts)-1]
	}
	return "http://<本机IP>:" + port
}
