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

const (
	serviceLabel      = "com.huangke.mylamp"
	servicePlistPath  = "/Library/LaunchDaemons/com.huangke.mylamp.plist"
	serviceStdoutPath = "/Library/Logs/mylamp.log"
	serviceStderrPath = "/Library/Logs/mylamp-error.log"
)

var launchDaemonTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
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

type launchDaemonConfig struct {
	Label      string
	Program    string
	Addr       string
	WorkingDir string
	StdoutPath string
	StderrPath string
}

func runServiceCommand(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: sudo lamp service <start|stop|restart|status>")
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
	if err := ensureRootPrivileges(); err != nil {
		return err
	}
	if err := ensureServiceConfigFile(); err != nil {
		return err
	}
	if err := ensureLaunchDaemon(); err != nil {
		return err
	}

	_ = runLaunchctl("bootout", "system", servicePlistPath)
	if err := runLaunchctl("bootstrap", "system", servicePlistPath); err != nil {
		return err
	}
	if err := runLaunchctl("enable", "system/"+serviceLabel); err != nil && !isLaunchctlAlreadyEnabled(err) {
		return err
	}
	if err := runLaunchctl("kickstart", "-k", "system/"+serviceLabel); err != nil {
		return err
	}

	fmt.Println("service started")
	fmt.Printf("config: %s\n", systemConfigPath)
	fmt.Printf("plist: %s\n", servicePlistPath)
	fmt.Printf("Web UI: %s\n", localURL(serviceAddress()))
	return nil
}

func serviceStop() error {
	if err := ensureRootPrivileges(); err != nil {
		return err
	}

	err := runLaunchctl("bootout", "system", servicePlistPath)
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
	out, err := exec.Command("launchctl", "print", "system/"+serviceLabel).CombinedOutput()
	if err != nil {
		if isLaunchctlNotLoadedOutput(string(out)) {
			fmt.Println("service status: stopped")
			fmt.Printf("plist: %s\n", servicePlistPath)
			fmt.Printf("config: %s\n", systemConfigPath)
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
	fmt.Printf("plist: %s\n", servicePlistPath)
	fmt.Printf("config: %s\n", systemConfigPath)
	fmt.Printf("stdout: %s\n", serviceStdoutPath)
	fmt.Printf("stderr: %s\n", serviceStderrPath)
	fmt.Printf("Web UI: %s\n", localURL(serviceAddress()))
	return nil
}

func ensureLaunchDaemon() error {
	cfg, err := currentLaunchDaemonConfig()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(servicePlistPath), 0o755); err != nil {
		return fmt.Errorf("create launch daemon directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.StdoutPath), 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	var buf bytes.Buffer
	if err := launchDaemonTemplate.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("render launch daemon: %w", err)
	}
	if err := os.WriteFile(servicePlistPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write launch daemon: %w", err)
	}
	return nil
}

func currentLaunchDaemonConfig() (launchDaemonConfig, error) {
	exe, err := os.Executable()
	if err != nil {
		return launchDaemonConfig{}, fmt.Errorf("resolve executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return launchDaemonConfig{}, fmt.Errorf("resolve executable symlink: %w", err)
	}

	addr := serviceAddress()
	return launchDaemonConfig{
		Label:      serviceLabel,
		Program:    exe,
		Addr:       addr,
		WorkingDir: filepath.Dir(exe),
		StdoutPath: serviceStdoutPath,
		StderrPath: serviceStderrPath,
	}, nil
}

func ensureRootPrivileges() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command must be run with sudo because it manages %s", servicePlistPath)
	}
	return nil
}

func ensureServiceConfigFile() error {
	if _, err := os.Stat(systemConfigPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing config file: %s", systemConfigPath)
		}
		return fmt.Errorf("read config file %s: %w", systemConfigPath, err)
	}
	return nil
}

func serviceAddress() string {
	addr := strings.TrimSpace(os.Getenv("LAMP_WEB_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	return addr
}

func runLaunchctl(args ...string) error {
	out, err := exec.Command("launchctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func isLaunchctlAlreadyEnabled(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "service already enabled")
}

func isLaunchctlNotLoaded(err error) bool {
	return isLaunchctlNotLoadedOutput(err.Error())
}

func isLaunchctlNotLoadedOutput(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "could not find service") ||
		strings.Contains(text, "input/output error") ||
		strings.Contains(text, "service is disabled") ||
		strings.Contains(text, "service is not loaded")
}
