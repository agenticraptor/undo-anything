//go:build windows

package daemon

import (
	"os"
	"syscall"
)

// detachAttr requests a detached process group on Windows.
func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000008} // DETACHED_PROCESS
}

// isAlive reports whether a process with the given PID exists. On Windows
// os.FindProcess only succeeds for live processes.
func isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.FindProcess(pid)
	return err == nil
}

// terminate stops the process. Windows has no SIGTERM, so we kill it.
func terminate(proc *os.Process) error {
	return proc.Kill()
}
