//go:build !windows
// +build !windows

package services

import (
	"os/exec"
	"syscall"
)

func hideConsoleWindow(cmd *exec.Cmd) {
}

// No-op on Unix; niceness needs the PID, so it's set after Start instead.
func lowerJobProcessPriority(cmd *exec.Cmd) {
}

// Best-effort: lowers child niceness so a CPU-heavy plugin can't starve the GUI thread.
func lowerJobProcessPriorityAfterStart(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Setpriority(syscall.PRIO_PROCESS, cmd.Process.Pid, 19)
}
