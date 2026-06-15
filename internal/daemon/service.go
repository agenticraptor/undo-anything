package daemon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/agenticraptor/undo-anything/internal/store"
)

// ServiceInfo describes an installed (or to-be-installed) OS service.
type ServiceInfo struct {
	Kind  string // "launchd", "systemd", or "unsupported"
	Label string
	Path  string
}

// Label returns a stable, per-folder service label derived from the root path.
func Label(s *store.Store) string {
	sum := sha256.Sum256([]byte(s.Root))
	return "undo-anything-" + hex.EncodeToString(sum[:])[:8]
}

// Install registers the watcher as a user-level OS service that starts on login
// and restarts on crash. It returns details and a best-effort activation error
// (the files are always written even if activation needs a manual step).
func Install(s *store.Store) (ServiceInfo, error) {
	exe, err := os.Executable()
	if err != nil {
		return ServiceInfo{}, err
	}
	switch runtime.GOOS {
	case "darwin":
		return installLaunchd(s, exe)
	case "linux":
		return installSystemd(s, exe)
	default:
		return ServiceInfo{Kind: "unsupported"}, fmt.Errorf("service install is not supported on %s; run `ua daemon start` instead", runtime.GOOS)
	}
}

// Uninstall removes a previously installed service.
func Uninstall(s *store.Store) (ServiceInfo, error) {
	switch runtime.GOOS {
	case "darwin":
		return uninstallLaunchd(s)
	case "linux":
		return uninstallSystemd(s)
	default:
		return ServiceInfo{Kind: "unsupported"}, fmt.Errorf("service install is not supported on %s", runtime.GOOS)
	}
}

func installLaunchd(s *store.Store, exe string) (ServiceInfo, error) {
	label := Label(s)
	dir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ServiceInfo{}, err
	}
	path := filepath.Join(dir, label+".plist")
	plist := fmt.Sprintf(launchdTemplate,
		label,
		plistEscape(exe), plistEscape(s.Root),
		plistEscape(LogPath(s)), plistEscape(LogPath(s)), plistEscape(s.Root))
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return ServiceInfo{}, err
	}
	info := ServiceInfo{Kind: "launchd", Label: label, Path: path}
	if _, err := exec.LookPath("launchctl"); err == nil {
		_ = exec.Command("launchctl", "unload", path).Run()
		err = exec.Command("launchctl", "load", path).Run()
		return info, err
	}
	return info, nil
}

func uninstallLaunchd(s *store.Store) (ServiceInfo, error) {
	label := Label(s)
	path := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", label+".plist")
	if _, err := exec.LookPath("launchctl"); err == nil {
		_ = exec.Command("launchctl", "unload", path).Run()
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		err = nil
	}
	return ServiceInfo{Kind: "launchd", Label: label, Path: path}, err
}

func installSystemd(s *store.Store, exe string) (ServiceInfo, error) {
	label := Label(s)
	dir := filepath.Join(configHome(), "systemd", "user")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ServiceInfo{}, err
	}
	path := filepath.Join(dir, label+".service")
	unit := fmt.Sprintf(systemdTemplate, s.Root, exe, s.Root, s.Root)
	if err := os.WriteFile(path, []byte(unit), 0o644); err != nil {
		return ServiceInfo{}, err
	}
	info := ServiceInfo{Kind: "systemd", Label: label, Path: path}
	if _, err := exec.LookPath("systemctl"); err == nil {
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		err = exec.Command("systemctl", "--user", "enable", "--now", label+".service").Run()
		return info, err
	}
	return info, nil
}

func uninstallSystemd(s *store.Store) (ServiceInfo, error) {
	label := Label(s)
	path := filepath.Join(configHome(), "systemd", "user", label+".service")
	if _, err := exec.LookPath("systemctl"); err == nil {
		_ = exec.Command("systemctl", "--user", "disable", "--now", label+".service").Run()
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		err = nil
	}
	return ServiceInfo{Kind: "systemd", Label: label, Path: path}, err
}

// plistEscape escapes the five XML predefined entities so a path containing
// `&`, `<`, `>`, or quotes can't produce a malformed launchd plist.
func plistEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}

func configHome() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return x
	}
	return filepath.Join(os.Getenv("HOME"), ".config")
}

const launchdTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>watch</string>
        <string>%s</string>
        <string>--quiet</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>WorkingDirectory</key>
    <string>%s</string>
</dict>
</plist>
`

const systemdTemplate = `[Unit]
Description=undo-anything watcher for %s
After=default.target

[Service]
Type=simple
ExecStart=%s watch %s --quiet
WorkingDirectory=%s
Restart=on-failure
RestartSec=3

[Install]
WantedBy=default.target
`
