package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/services"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read pipe: %v", err)
	}
	return buf.String()
}

func TestCLIVersion(t *testing.T) {
	output := captureStdout(t, cliVersion)

	if !strings.Contains(output, "cauldron") {
		t.Errorf("expected version output to mention \"cauldron\", got: %q", output)
	}
	if !strings.Contains(output, "built with") {
		t.Errorf("expected version output to mention Go version, got: %q", output)
	}
}

func TestReadNonEmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "repos.txt")
	content := "https://github.com/example/one\n\n# a comment\n  \nhttps://github.com/example/two\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	lines, err := readNonEmptyLines(path)
	if err != nil {
		t.Fatalf("readNonEmptyLines error: %v", err)
	}

	want := []string{"https://github.com/example/one", "https://github.com/example/two"}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d: %v", len(lines), len(want), lines)
	}
	for i, line := range lines {
		if line != want[i] {
			t.Errorf("line %d = %q, want %q", i, line, want[i])
		}
	}
}

func TestReadNonEmptyLines_MissingFile(t *testing.T) {
	if _, err := readNonEmptyLines(filepath.Join(t.TempDir(), "does-not-exist.txt")); err == nil {
		t.Error("expected error reading a nonexistent file, got nil")
	}
}

func TestNewCLIContext(t *testing.T) {
	ctx, err := newCLIContext()
	if err != nil {
		t.Fatalf("newCLIContext error: %v", err)
	}
	defer ctx.db.Close()

	if ctx.db == nil {
		t.Error("db is nil")
	}
	if ctx.settings == nil {
		t.Error("settings is nil")
	}
	if ctx.envService == nil {
		t.Error("envService is nil")
	}
	if ctx.uvService == nil {
		t.Error("uvService is nil")
	}
	if ctx.dockerImageBuilder == nil {
		t.Error("dockerImageBuilder is nil")
	}
	if ctx.pluginLoaderV2 == nil {
		t.Error("pluginLoaderV2 is nil")
	}
	if ctx.pluginInstaller == nil {
		t.Error("pluginInstaller is nil")
	}
}

func TestCLIPluginInstall_UsageErrors(t *testing.T) {
	if err := cliPluginInstall(nil); err == nil {
		t.Error("expected usage error with no repo URL and no --file, got nil")
	}

	missing := filepath.Join(t.TempDir(), "does-not-exist.txt")
	if err := cliPluginInstall([]string{"--file", missing}); err == nil {
		t.Error("expected error reading a nonexistent --file, got nil")
	}
}

func TestCLIPlugin_Dispatch(t *testing.T) {
	if err := cliPlugin(nil); err == nil {
		t.Error("expected usage error with no subcommand, got nil")
	}
	if err := cliPlugin([]string{"frobnicate"}); err == nil {
		t.Error("expected error for unknown plugin subcommand, got nil")
	}
}

func TestCLIPluginInstallAndUninstall(t *testing.T) {
	repoURL := "https://github.com/noatgnu/diann-curtainptm-converter-plugin"

	ctx, err := newCLIContext()
	if err != nil {
		t.Fatalf("newCLIContext error: %v", err)
	}
	defer ctx.db.Close()

	// Best-effort pre-cleanup in case a previous run left this installed.
	_ = ctx.pluginInstaller.UninstallPlugin(repoURL, services.UninstallOptions{
		DeleteJobHistory:   true,
		DeleteEnvironments: true,
	})

	if err := cliPluginInstall([]string{repoURL}); err != nil {
		t.Fatalf("cliPluginInstall error: %v", err)
	}

	installedCtx, err := newCLIContext()
	if err != nil {
		t.Fatalf("newCLIContext error: %v", err)
	}
	defer installedCtx.db.Close()

	found := false
	for _, p := range installedCtx.pluginLoaderV2.GetAllPlugins() {
		if p.Repository == repoURL {
			found = true
			break
		}
	}
	if !found {
		t.Error("installed plugin not found via pluginLoaderV2.GetAllPlugins()")
	}

	if err := cliPluginUninstall([]string{"--delete-job-history", "--delete-environments", repoURL}); err != nil {
		t.Fatalf("cliPluginUninstall error: %v", err)
	}
}

func TestCLIDoctor(t *testing.T) {
	if err := cliDoctor(); err != nil {
		t.Fatalf("cliDoctor error: %v", err)
	}
}

func TestRunCLI_FallThrough(t *testing.T) {
	if runCLI(nil) {
		t.Error("expected runCLI(nil) to fall through (return false)")
	}
	if runCLI([]string{"some-unrecognized-arg"}) {
		t.Error("expected an unrecognized subcommand to fall through (return false)")
	}
	if runCLI([]string{"--verbose"}) {
		t.Error("expected --verbose with no subcommand to fall through (return false)")
	}
}

func TestRunCLI_Version(t *testing.T) {
	output := captureStdout(t, func() {
		if !runCLI([]string{"version"}) {
			t.Error("expected runCLI([\"version\"]) to return true")
		}
	})
	if !strings.Contains(output, "cauldron") {
		t.Errorf("expected version output to mention \"cauldron\", got: %q", output)
	}
}

func TestRunCLI_Doctor(t *testing.T) {
	// cliDoctor only errors (triggering os.Exit) if newCLIContext fails,
	// which shouldn't happen in a normal test environment.
	captureStdout(t, func() {
		if !runCLI([]string{"doctor"}) {
			t.Error("expected runCLI([\"doctor\"]) to return true")
		}
	})
}
