package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const helpText = `用法：
  lamp 开
  lamp 关
  lamp 状态
  lamp 亮度 <1-100>
  lamp 色温 <2700-5100>
  lamp <场景> [亮度1-100]

场景：暖白 / 自然 / 冷白 / 阅读 / 睡前`

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "错误："+format+"\n", args...)
	os.Exit(1)
}

func main() {
	// Load .env: cwd first, then the fixed project directory
	loadDotEnv(".env")
	loadDotEnv("/Users/huangke/Developer/mylamp/.env")

	ip := os.Getenv("LAMP_IP")
	token := os.Getenv("LAMP_TOKEN")
	if ip == "" || token == "" {
		missing := []string{}
		if ip == "" {
			missing = append(missing, "LAMP_IP")
		}
		if token == "" {
			missing = append(missing, "LAMP_TOKEN")
		}
		fatal(".env 文件中缺少配置项：%s", strings.Join(missing, ", "))
	}

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println(helpText)
		os.Exit(0)
	}

	lamp, err := NewLamp(ip, token)
	if err != nil {
		fatal("%v", err)
	}

	cmd := args[0]

	switch cmd {
	case "开":
		if err := lamp.TurnOn(); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		fmt.Println("台灯已开")

	case "关":
		if err := lamp.TurnOff(); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		fmt.Println("台灯已关")

	case "状态":
		s, err := lamp.GetStatus()
		if err != nil {
			fatal("台灯控制失败：%v", err)
		}
		onStr := "关"
		if s.On {
			onStr = "开"
		}
		fmt.Printf("状态：%s\n亮度：%d%%\n色温：%dK\n", onStr, s.Brightness, s.ColorTemp)

	case "亮度":
		if len(args) < 2 {
			fatal("请指定亮度值，例如：lamp 亮度 80")
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			fatal("亮度必须是整数，收到：%s", args[1])
		}
		if err := lamp.SetBrightness(v); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		if v < 1 {
			v = 1
		} else if v > 100 {
			v = 100
		}
		fmt.Printf("亮度已设置：%d%%\n", v)

	case "色温":
		if len(args) < 2 {
			fatal("请指定色温值，例如：lamp 色温 4000")
		}
		k, err := strconv.Atoi(args[1])
		if err != nil {
			fatal("色温必须是整数，收到：%s", args[1])
		}
		if err := lamp.SetColorTemp(k); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		if k < 2700 {
			k = 2700
		} else if k > 5100 {
			k = 5100
		}
		fmt.Printf("色温已设置：%dK\n", k)

	default:
		// 场景模式
		if _, ok := scenes[cmd]; !ok {
			fmt.Fprintf(os.Stderr, "错误：未知命令 '%s'\n\n%s\n", cmd, helpText)
			os.Exit(1)
		}
		var brightnessPtr *int
		if len(args) > 1 {
			v, err := strconv.Atoi(args[1])
			if err != nil {
				fatal("亮度必须是整数，收到：%s", args[1])
			}
			brightnessPtr = &v
		}
		if err := lamp.SetScene(cmd, brightnessPtr); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		msg := fmt.Sprintf("场景：%s", cmd)
		if brightnessPtr != nil {
			msg += fmt.Sprintf("，亮度：%d%%", *brightnessPtr)
		}
		fmt.Println(msg)
	}
}
