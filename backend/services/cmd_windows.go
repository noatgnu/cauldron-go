//go:build windows
// +build windows

package services

import (
	"os/exec"
	"syscall"
)

const (
	createNoWindow        = 0x08000000
	createNewProcessGroup = 0x00000200
	idlePriorityClass     = 0x00000040
)

func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow | createNewProcessGroup,
	}
}

// Windows has no post-start priority API without a process handle, so this sets it pre-start via CreationFlags.
func lowerJobProcessPriority(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= idlePriorityClass
}

// No-op on Windows; already applied above at process creation.
func lowerJobProcessPriorityAfterStart(cmd *exec.Cmd) {
}
