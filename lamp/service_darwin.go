//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const launchAgentLabel = "com.huangke.mylamp"

var launchAgentTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>{{.Label}}</string>

  <key>ProgramArguments</key>
  <array>
    <string>{{.Program}}</string>
    <string>serve</string>
    <string>{{.Addr}}</string>
  </array>

  <key>WorkingDirectory</key>
  <string>{{.WorkingDir}}</string>

  <key>EnvironmentVariables</key>
  <dict>
    <key>LAMP_IP</key>
    <string>{{.LampIP}}</string>
    <key>LAMP_TOKEN</key>
    <string>{{.LampToken}}</string>
{{- if .LampDebug }}
    <key>LAMP_DEBUG</key>
    <string>{{.LampDebug}}</string>
{{- end }}
{{- if .LampWebAddr }}
    <key>LAMP_WEB_ADDR</key>
    <string>{{.LampWebAddr}}</string>
{{- end }}
  </dict>

  <key>RunAtLoad</key>
  <true/>

  <key>KeepAlive</key>
  <true/>

  <key>StandardOutPath</key>
  <string>{{.StdoutPath}}</string>

  <key>StandardErrorPath</key>
  <string>{{.StderrPath}}</string>
</dict>
</plist>
`))

type launchAgentConfig struct {
	Label       string
	Program     string
	Addr        string
	WorkingDir  string
	LampIP      string
	LampToken   string
	LampDebug   string
	LampWebAddr string
	StdoutPath  string
	StderrPath  string
}

func runServiceCommand(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: lamp service <start|stop|restart|status>")
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "start":
		return serviceStart()
	case "stop":
		return serviceStop()
	case "restart":
		return serviceRestart()
	case "status":
		return serviceStatus()
	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

func serviceStart() error {
	if err := ensureLaunchAgent(); err != nil {
		return err
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	plist := launchAgentPath()
	_ = runLaunchctl("bootout", "gui/"+uid, plist)
	if err := runLaunchctl("bootstrap", "gui/"+uid, plist); err != nil {
		return err
	}
	if err := runLaunchctl("kickstart", "-k", "gui/"+uid+"/"+launchAgentLabel); err != nil {
		return err
	}

	fmt.Println("service started")
	fmt.Printf("Web UI: %s\n", localURL(serviceAddress()))
	return nil
}

func serviceStop() error {
	uid := fmt.Sprintf("%d", os.Getuid())
	plist := launchAgentPath()
	err := runLaunchctl("bootout", "gui/"+uid, plist)
	if err != nil && !isLaunchctlNotLoaded(err) {
		return err
	}
	fmt.Println("service stopped")
	return nil
}

func serviceRestart() error {
	if err := serviceStop(); err != nil {
		return err
	}
	return serviceStart()
}

func serviceStatus() error {
	uid := fmt.Sprintf("%d", os.Getuid())
	out, err := exec.Command("launchctl", "print", "gui/"+uid+"/"+launchAgentLabel).CombinedOutput()
	if err != nil {
		if isLaunchctlNotLoadedOutput(string(out)) {
			fmt.Println("service status: stopped")
			return nil
		}
		return fmt.Errorf("launchctl print failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	state := "loaded"
	text := string(out)
	if strings.Contains(text, "state = running") {
		state = "running"
	}

	fmt.Printf("service status: %s\n", state)
	fmt.Printf("plist: %s\n", launchAgentPath())
	fmt.Printf("Web UI: %s\n", localURL(serviceAddress()))
	return nil
}

func ensureLaunchAgent() error {
	cfg, err := currentLaunchAgentConfig()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(launchAgentPath()), 0o755); err != nil {
		return fmt.Errorf("create launch agent directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.StdoutPath), 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	var buf bytes.Buffer
	if err := launchAgentTemplate.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("render launch agent: %w", err)
	}
	if err := os.WriteFile(launchAgentPath(), buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write launch agent: %w", err)
	}
	return nil
}

func currentLaunchAgentConfig() (launchAgentConfig, error) {
	ip := strings.TrimSpace(os.Getenv("LAMP_IP"))
	token := strings.TrimSpace(os.Getenv("LAMP_TOKEN"))
	if ip == "" || token == "" {
		return launchAgentConfig{}, fmt.Errorf("LAMP_IP and LAMP_TOKEN are required for service start")
	}

	exe, err := os.Executable()
	if err != nil {
		return launchAgentConfig{}, fmt.Errorf("resolve executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return launchAgentConfig{}, fmt.Errorf("resolve executable symlink: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return launchAgentConfig{}, fmt.Errorf("resolve working directory: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return launchAgentConfig{}, fmt.Errorf("resolve home directory: %w", err)
	}

	addr := serviceAddress()
	return launchAgentConfig{
		Label:       launchAgentLabel,
		Program:     exe,
		Addr:        addr,
		WorkingDir:  wd,
		LampIP:      ip,
		LampToken:   token,
		LampDebug:   strings.TrimSpace(os.Getenv("LAMP_DEBUG")),
		LampWebAddr: strings.TrimSpace(os.Getenv("LAMP_WEB_ADDR")),
		StdoutPath:  filepath.Join(home, "Library", "Logs", "mylamp.log"),
		StderrPath:  filepath.Join(home, "Library", "Logs", "mylamp-error.log"),
	}, nil
}

func serviceAddress() string {
	addr := strings.TrimSpace(os.Getenv("LAMP_WEB_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	return addr
}

func launchAgentPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", "Library", "LaunchAgents", launchAgentLabel+".plist")
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
}

func runLaunchctl(args ...string) error {
	out, err := exec.Command("launchctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func isLaunchctlNotLoaded(err error) bool {
	return isLaunchctlNotLoadedOutput(err.Error())
}

func isLaunchctlNotLoadedOutput(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "could not find service") ||
		strings.Contains(text, "input/output error") ||
		strings.Contains(text, "service is disabled")
}
