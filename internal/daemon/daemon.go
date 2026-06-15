// Package daemon manages the background watcher process: starting it detached,
// tracking it via a pidfile, stopping it, and installing it as an OS service
// (launchd on macOS, a systemd user unit on Linux).
package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/agenticraptor/undo-anything/internal/store"
)

// ErrAlreadyRunning is returned by Start when a live daemon already exists.
var ErrAlreadyRunning = errors.New("watcher already running")

// ErrNotRunning is returned by Stop when no live daemon exists.
var ErrNotRunning = errors.New("watcher is not running")

const (
	pidFile = "daemon.pid"
	logFile = "daemon.log"
)

func pidPath(s *store.Store) string { return filepath.Join(s.Dir, pidFile) }

// LogPath returns the path the background daemon logs to.
func LogPath(s *store.Store) string { return filepath.Join(s.Dir, logFile) }

// Status reports the running daemon's PID, or ok=false if none is alive. A
// stale pidfile (process gone) is cleaned up as a side effect.
func Status(s *store.Store) (pid int, ok bool) {
	pid, err := readPID(s)
	if err != nil {
		return 0, false
	}
	if !isAlive(pid) {
		_ = os.Remove(pidPath(s))
		return 0, false
	}
	return pid, true
}

// Start launches the watcher as a detached background process and records its
// PID. It returns the new PID.
func Start(s *store.Store) (int, error) {
	if pid, ok := Status(s); ok {
		return pid, ErrAlreadyRunning
	}
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	logf, err := os.OpenFile(LogPath(s), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer logf.Close()

	cmd := exec.Command(exe, "watch", s.Root, "--quiet")
	cmd.Dir = s.Root
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = detachAttr()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	if err := writePID(s, pid); err != nil {
		_ = cmd.Process.Kill()
		return 0, err
	}
	_ = cmd.Process.Release()
	return pid, nil
}

// Stop terminates the running daemon.
func Stop(s *store.Store) error {
	pid, ok := Status(s)
	if !ok {
		return ErrNotRunning
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := terminate(proc); err != nil {
		return err
	}
	return os.Remove(pidPath(s))
}

func readPID(s *store.Store) (int, error) {
	data, err := os.ReadFile(pidPath(s))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func writePID(s *store.Store, pid int) error {
	return os.WriteFile(pidPath(s), []byte(fmt.Sprintf("%d\n", pid)), 0o644)
}
