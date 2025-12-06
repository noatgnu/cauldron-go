//go:build !windows
// +build !windows

package services

import (
	"os/exec"
)

func hideConsoleWindow(cmd *exec.Cmd) {
}
