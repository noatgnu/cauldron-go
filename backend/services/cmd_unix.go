//go:build !windows
// +build !windows

package services

import (
	"os/exec"
	"syscall"
)

func hideConsoleWindow(cmd *exec.Cmd) {
}

func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
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
