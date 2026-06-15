//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// detachAttr starts the child in its own session so it survives the parent
// shell exiting.
func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// isAlive reports whether a process with the given PID exists, using the
// classic signal-0 liveness probe.
func isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// terminate asks the process to shut down gracefully.
func terminate(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}
