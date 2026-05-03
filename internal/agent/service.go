package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
)

const systemdUnit = `[Unit]
Description=remote-cli agent
After=network.target

[Service]
ExecStart={{.Bin}} run
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.remote-cli.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.Bin}}</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
</dict>
</plist>
`

func selfBin() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("find executable: %w", err)
	}
	return filepath.EvalSymlinks(bin)
}

func renderTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ── Linux systemd ─────────────────────────────────────────────────────────────

func systemdUnitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", "remote-cli.service")
}

func installSystemd(bin string) error {
	content, err := renderTemplate(systemdUnit, struct{ Bin string }{bin})
	if err != nil {
		return err
	}
	path := systemdUnitPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}
	for _, args := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "remote-cli"},
		{"--user", "start", "remote-cli"},
	} {
		if out, err := exec.Command("systemctl", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl %v: %v\n%s", args, err, out)
		}
	}
	fmt.Printf("installed: %s\nservice enabled and started\n", path)
	return nil
}

func uninstallSystemd() error {
	exec.Command("systemctl", "--user", "stop", "remote-cli").Run()    //nolint
	exec.Command("systemctl", "--user", "disable", "remote-cli").Run() //nolint
	path := systemdUnitPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	exec.Command("systemctl", "--user", "daemon-reload").Run() //nolint
	fmt.Println("service removed")
	return nil
}

func systemdCtl(args ...string) error {
	out, err := exec.Command("systemctl", append([]string{"--user"}, args...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", out)
	}
	return nil
}

func systemdLogs() error {
	cmd := exec.Command("journalctl", "--user", "-u", "remote-cli", "-f", "--no-pager")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ── macOS launchd ─────────────────────────────────────────────────────────────

func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.remote-cli.agent.plist")
}

func launchdLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Logs", "remote-cli.log")
}

func installLaunchd(bin string) error {
	content, err := renderTemplate(launchdPlist, struct {
		Bin     string
		LogPath string
	}{bin, launchdLogPath()})
	if err != nil {
		return err
	}
	path := launchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}
	out, err := exec.Command("launchctl", "load", "-w", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %v\n%s", err, out)
	}
	fmt.Printf("installed: %s\nservice loaded and started\n", path)
	return nil
}

func uninstallLaunchd() error {
	path := launchdPlistPath()
	exec.Command("launchctl", "unload", "-w", path).Run() //nolint
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Println("service removed")
	return nil
}

func launchdLogs() error {
	cmd := exec.Command("tail", "-f", launchdLogPath())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ── Public API ────────────────────────────────────────────────────────────────

func ServiceInstall() error {
	bin, err := selfBin()
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		return installSystemd(bin)
	case "darwin":
		return installLaunchd(bin)
	default:
		return fmt.Errorf("service management not supported on %s — run 'remote-cli run' manually", runtime.GOOS)
	}
}

func ServiceUninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallSystemd()
	case "darwin":
		return uninstallLaunchd()
	default:
		return fmt.Errorf("service management not supported on %s", runtime.GOOS)
	}
}

func ServiceStart() error {
	switch runtime.GOOS {
	case "linux":
		return systemdCtl("start", "remote-cli")
	case "darwin":
		out, err := exec.Command("launchctl", "start", "com.remote-cli.agent").CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", out)
		}
		return nil
	default:
		return fmt.Errorf("service management not supported on %s", runtime.GOOS)
	}
}

func ServiceStop() error {
	switch runtime.GOOS {
	case "linux":
		return systemdCtl("stop", "remote-cli")
	case "darwin":
		out, err := exec.Command("launchctl", "stop", "com.remote-cli.agent").CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", out)
		}
		return nil
	default:
		return fmt.Errorf("service management not supported on %s", runtime.GOOS)
	}
}

func ServiceLogs() error {
	switch runtime.GOOS {
	case "linux":
		return systemdLogs()
	case "darwin":
		return launchdLogs()
	default:
		return fmt.Errorf("service management not supported on %s", runtime.GOOS)
	}
}
