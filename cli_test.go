package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
)

// captureStdout redirects os.Stdout for the duration of fn and returns everything written to it.
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
	defer ctx.close()

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
	defer installedCtx.close()

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
	// cliDoctor only errors (triggering os.Exit) if newCLIContext fails, which shouldn't happen here.
	captureStdout(t, func() {
		if !runCLI([]string{"doctor"}) {
			t.Error("expected runCLI([\"doctor\"]) to return true")
		}
	})
}

func TestCLIUv_Dispatch(t *testing.T) {
	if err := cliUv(nil); err == nil {
		t.Error("expected usage error with no subcommand, got nil")
	}
	if err := cliUv([]string{"frobnicate"}); err == nil {
		t.Error("expected error for unknown uv subcommand, got nil")
	}
}

func TestCLIUvPython_Dispatch(t *testing.T) {
	if err := cliUvPython(nil); err == nil {
		t.Error("expected usage error with no subcommand, got nil")
	}
	if err := cliUvPython([]string{"install"}); err == nil {
		t.Error("expected error for 'install' with no version argument, got nil")
	}
	if err := cliUvPython([]string{"frobnicate"}); err == nil {
		t.Error("expected error for unknown uv python subcommand, got nil")
	}
}

func TestCLIUvVenv_Dispatch(t *testing.T) {
	if err := cliUvVenv(nil); err == nil {
		t.Error("expected usage error with no subcommand, got nil")
	}
	if err := cliUvVenv([]string{"frobnicate"}); err == nil {
		t.Error("expected error for unknown uv venv subcommand, got nil")
	}
}

func TestCLIUvVenvCreate_UsageError(t *testing.T) {
	if err := cliUvVenvCreate(nil); err == nil {
		t.Error("expected usage error with no path argument, got nil")
	}
}

// ensureUvAvailableForTest installs uv if it isn't already available, skipping the calling test if that's not possible (e.g. no network).
func ensureUvAvailableForTest(t *testing.T, ctx *cliContext) {
	t.Helper()

	if ctx.uvService.IsUvAvailable() {
		return
	}
	if err := cliUvInstall(); err != nil {
		t.Skipf("uv could not be installed (no network access?): %v", err)
	}
	if !ctx.uvService.IsUvAvailable() {
		t.Skip("uv still not available after install attempt")
	}
}

func TestCLIUvActions_Integration(t *testing.T) {
	ctx, err := newCLIContext()
	if err != nil {
		t.Fatalf("newCLIContext error: %v", err)
	}
	defer ctx.db.Close()

	ensureUvAvailableForTest(t, ctx)

	if err := cliUvInstall(); err != nil {
		t.Fatalf("cliUvInstall error: %v", err)
	}

	const pythonVersion = "3.12"
	if err := cliUvPythonInstall(pythonVersion); err != nil {
		t.Skipf("could not install Python %s via uv (no network access?): %v", pythonVersion, err)
	}

	if err := cliUvPythonList(); err != nil {
		t.Errorf("cliUvPythonList error: %v", err)
	}

	venvPath := filepath.Join(t.TempDir(), "cli-uv-venv-test")
	if err := cliUvVenvCreate([]string{"--python", pythonVersion, venvPath}); err != nil {
		t.Fatalf("cliUvVenvCreate error: %v", err)
	}

	venvs, err := ctx.envService.GetVirtualEnvironments()
	if err != nil {
		t.Fatalf("GetVirtualEnvironments error: %v", err)
	}
	var created *services.VirtualEnvironment
	for i := range venvs {
		if strings.Contains(venvs[i].Path, venvPath) {
			created = &venvs[i]
			break
		}
	}
	if created == nil {
		t.Fatal("created uv venv not found via GetVirtualEnvironments()")
	}
	if created.Source != "uv" {
		t.Errorf("venv Source = %q, want \"uv\"", created.Source)
	}

	if err := ctx.envService.DeleteVirtualEnvironment(created.ID); err != nil {
		t.Errorf("failed to clean up test venv DB entry: %v", err)
	}
}

func TestCoercePluginParams(t *testing.T) {
	plugin := &models.PluginV2{
		Definition: models.PluginDefinition{
			Inputs: []models.PluginInputV2{
				{Name: "count", Type: models.PluginInputTypeNumber},
				{Name: "enabled", Type: models.PluginInputTypeBoolean},
				{Name: "tags", Type: models.PluginInputTypeMultiSelect},
				{Name: "label", Type: models.PluginInputTypeText},
			},
		},
	}

	coerced, err := coercePluginParams(plugin, map[string]interface{}{
		"count":      "5",
		"enabled":    "true",
		"tags":       "a, b,c",
		"label":      "hello",
		"unknown":    "passthrough",
		"already_ok": 3.5, // not a string; must pass through unchanged
	})
	if err != nil {
		t.Fatalf("coercePluginParams error: %v", err)
	}

	if v, ok := coerced["count"].(float64); !ok || v != 5 {
		t.Errorf("count = %#v, want float64(5)", coerced["count"])
	}
	if v, ok := coerced["enabled"].(bool); !ok || v != true {
		t.Errorf("enabled = %#v, want bool(true)", coerced["enabled"])
	}
	tags, ok := coerced["tags"].([]interface{})
	if !ok || len(tags) != 3 || tags[0] != "a" || tags[1] != "b" || tags[2] != "c" {
		t.Errorf("tags = %#v, want [a b c]", coerced["tags"])
	}
	if v, ok := coerced["label"].(string); !ok || v != "hello" {
		t.Errorf("label = %#v, want string(\"hello\")", coerced["label"])
	}
	if v, ok := coerced["unknown"].(string); !ok || v != "passthrough" {
		t.Errorf("unknown = %#v, want string(\"passthrough\")", coerced["unknown"])
	}
	if v, ok := coerced["already_ok"].(float64); !ok || v != 3.5 {
		t.Errorf("already_ok = %#v, want float64(3.5) unchanged", coerced["already_ok"])
	}
}

func TestCoercePluginParams_Errors(t *testing.T) {
	plugin := &models.PluginV2{
		Definition: models.PluginDefinition{
			Inputs: []models.PluginInputV2{
				{Name: "count", Type: models.PluginInputTypeNumber},
				{Name: "enabled", Type: models.PluginInputTypeBoolean},
			},
		},
	}

	if _, err := coercePluginParams(plugin, map[string]interface{}{"count": "not-a-number"}); err == nil {
		t.Error("expected error coercing a non-numeric string for a number input")
	}
	if _, err := coercePluginParams(plugin, map[string]interface{}{"enabled": "not-a-bool"}); err == nil {
		t.Error("expected error coercing a non-boolean string for a boolean input")
	}
}

func TestFindPlugin_NotFound(t *testing.T) {
	ctx, err := newCLIContext()
	if err != nil {
		t.Fatalf("newCLIContext error: %v", err)
	}
	defer ctx.close()

	if _, err := findPlugin(ctx.pluginLoaderV2, "definitely-not-a-real-plugin-id"); err == nil {
		t.Error("expected error for a nonexistent plugin reference, got nil")
	}
}

func TestCLIJob_Dispatch(t *testing.T) {
	if err := cliJob(nil); err == nil {
		t.Error("expected usage error with no subcommand, got nil")
	}
	if err := cliJob([]string{"frobnicate"}); err == nil {
		t.Error("expected error for unknown job subcommand, got nil")
	}
}

func TestCLIJobRun_UsageErrors(t *testing.T) {
	if err := cliJobRun(nil); err == nil {
		t.Error("expected usage error with no plugin id and no --file, got nil")
	}

	// Flags after the positional plugin id aren't parsed as flags by Go's flag package; this must surface a clear error, not a confusing one.
	if err := cliJobRun([]string{"some-plugin", "--param", "x=1"}); err == nil {
		t.Error("expected an error for flags placed after the positional plugin id, got nil")
	} else if !strings.Contains(err.Error(), "flags must come before") {
		t.Errorf("expected a flag-order hint in the error, got: %v", err)
	}

	if err := cliJobRun([]string{"--file", filepath.Join(t.TempDir(), "missing.json")}); err == nil {
		t.Error("expected error reading a nonexistent --file, got nil")
	}

	if err := cliJobRun([]string{"definitely-not-a-real-plugin-id"}); err == nil {
		t.Error("expected error for an unknown plugin reference, got nil")
	}
}

func TestCLIJobList(t *testing.T) {
	if err := cliJobList(); err != nil {
		t.Fatalf("cliJobList error: %v", err)
	}
}

func TestCLIJobStatus_NotFound(t *testing.T) {
	if err := cliJobStatus([]string{"not-a-real-job-id"}); err == nil {
		t.Error("expected error for a nonexistent job id, got nil")
	}
}

// TestCLIJobRun_Integration exercises job run/wait/query end to end using this repo's built-in cv-plot plugin, instead of installing over the network.
func TestCLIJobRun_Integration(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd error: %v", err)
	}

	ctx, err := newCLIContextWithPluginsDir(filepath.Join(cwd, "plugins"))
	if err != nil {
		t.Fatalf("newCLIContextWithPluginsDir error: %v", err)
	}
	defer ctx.close()

	plugin, err := findPlugin(ctx.pluginLoaderV2, "cv-plot")
	if err != nil {
		t.Fatalf("findPlugin error: %v", err)
	}

	params, err := coercePluginParams(plugin, map[string]interface{}{
		"annotation_file":     filepath.Join(cwd, "examples", "diann", "annotation.txt"),
		"log_file_path":       filepath.Join(cwd, "examples", "diann", "Reports.log.txt"),
		"report_pr_file_path": filepath.Join(cwd, "examples", "diann", "Reports.pr_matrix.tsv"),
		"report_pg_file_path": filepath.Join(cwd, "examples", "diann", "Reports.pg_matrix.tsv"),
		"intensity_col":       "Intensity",
	})
	if err != nil {
		t.Fatalf("coercePluginParams error: %v", err)
	}

	jobID, err := executePluginJob(ctx.pluginExecutor, ctx.jobQueue, ctx.settings, plugin, params)
	if err != nil {
		t.Fatalf("executePluginJob error: %v", err)
	}

	job, err := waitForJob(ctx, jobID, 150*time.Second)
	if err != nil {
		t.Fatalf("waitForJob error: %v", err)
	}
	if job.OutputPath != "" {
		defer os.RemoveAll(job.OutputPath)
	}

	if job.Status != models.JobStatusCompleted {
		t.Errorf("expected job to complete, got status=%s error=%q", job.Status, job.Error)
	}

	if err := cliJobStatus([]string{jobID}); err != nil {
		t.Errorf("cliJobStatus error: %v", err)
	}
}

func TestCLIPluginList(t *testing.T) {
	if err := cliPluginList(); err != nil {
		t.Fatalf("cliPluginList error: %v", err)
	}
}

func TestCLIPluginInputs(t *testing.T) {
	repoURL := "https://github.com/noatgnu/diann-curtainptm-converter-plugin"

	setupCtx, err := newCLIContext()
	if err != nil {
		t.Fatalf("newCLIContext error: %v", err)
	}
	_ = setupCtx.pluginInstaller.UninstallPlugin(repoURL, services.UninstallOptions{
		DeleteJobHistory:   true,
		DeleteEnvironments: true,
	})
	setupCtx.close()

	if err := cliPluginInstall([]string{repoURL}); err != nil {
		t.Fatalf("cliPluginInstall error: %v", err)
	}
	defer func() {
		if err := cliPluginUninstall([]string{"--delete-job-history", "--delete-environments", repoURL}); err != nil {
			t.Errorf("cleanup: cliPluginUninstall error: %v", err)
		}
	}()

	if err := cliPluginInputs([]string{"diann-curtainptm-converter"}); err != nil {
		t.Errorf("cliPluginInputs error: %v", err)
	}

	jsonOutput := captureStdout(t, func() {
		if err := cliPluginInputs([]string{"--json", "diann-curtainptm-converter"}); err != nil {
			t.Errorf("cliPluginInputs --json error: %v", err)
		}
	})
	var inputs []models.PluginInputV2
	if err := json.Unmarshal([]byte(jsonOutput), &inputs); err != nil {
		t.Errorf("cliPluginInputs --json output did not parse as JSON: %v\noutput: %s", err, jsonOutput)
	}
	if len(inputs) == 0 {
		t.Error("expected at least one input in --json output")
	}
}

func TestCLIPluginInputs_UsageErrors(t *testing.T) {
	if err := cliPluginInputs(nil); err == nil {
		t.Error("expected usage error with no plugin id, got nil")
	}
	if err := cliPluginInputs([]string{"some-plugin", "extra-arg"}); err == nil {
		t.Error("expected error for unexpected extra arguments, got nil")
	}
	if err := cliPluginInputs([]string{"definitely-not-a-real-plugin-id"}); err == nil {
		t.Error("expected error for an unknown plugin reference, got nil")
	}
}
