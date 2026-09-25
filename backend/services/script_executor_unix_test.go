//go:build !windows
// +build !windows

package services

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCancelJob_KillsBackgroundedGrandchild(t *testing.T) {
	executor, _ := createTestScriptExecutor(t)

	pluginDir := t.TempDir()
	pidFile := filepath.Join(pluginDir, "grandchild.pid")
	scriptPath := filepath.Join(pluginDir, "run.sh")
	script := "#!/bin/sh\nsleep 30 &\necho $! > \"$1\"\nwait\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	config := ScriptConfig{
		PluginID:   1,
		Type:       "test-plugin",
		ScriptName: "run.sh",
		Args:       []string{pidFile},
		OutputDir:  t.TempDir(),
		FolderPath: pluginDir,
	}

	jobID := "test-job-process-group"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- executor.ExecuteDirectScript(ctx, jobID, config)
	}()

	var grandchildPID int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidFile)
		if err == nil && strings.TrimSpace(string(data)) != "" {
			pid, convErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if convErr == nil {
				grandchildPID = pid
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if grandchildPID == 0 {
		t.Fatal("timed out waiting for the backgrounded grandchild's PID to be written")
	}

	if err := executor.CancelJob(jobID); err != nil {
		t.Fatalf("CancelJob error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ExecuteDirectScript to return after cancellation")
	}

	time.Sleep(200 * time.Millisecond)
	if err := syscall.Kill(grandchildPID, 0); err == nil {
		syscall.Kill(grandchildPID, syscall.SIGKILL)
		t.Errorf("expected grandchild PID %d to be dead after CancelJob, but it is still alive", grandchildPID)
	}
}
