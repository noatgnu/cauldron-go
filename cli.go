package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
)

// runCLI runs a recognized CLI subcommand headlessly and returns true; unrecognized/empty args fall through to the normal GUI launch.
func runCLI(args []string) bool {
	if len(args) == 0 {
		return false
	}

	verbose := false
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--verbose" {
			verbose = true
			continue
		}
		filtered = append(filtered, a)
	}
	if len(filtered) == 0 {
		return false
	}

	switch filtered[0] {
	case "version", "--version", "-v":
		cliVersion()
		return true
	case "plugin":
		// Backend services log via the standard `log` package; quiet it by default so command output isn't drowned out.
		if !verbose {
			log.SetOutput(io.Discard)
		}
		if err := cliPlugin(filtered[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return true
	case "doctor":
		if !verbose {
			log.SetOutput(io.Discard)
		}
		if err := cliDoctor(); err != nil {
			os.Exit(1)
		}
		return true
	case "uv":
		if !verbose {
			log.SetOutput(io.Discard)
		}
		if err := cliUv(filtered[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return true
	case "job":
		if !verbose {
			log.SetOutput(io.Discard)
		}
		if err := cliJob(filtered[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return true
	case "db":
		if !verbose {
			log.SetOutput(io.Discard)
		}
		if err := cliDb(filtered[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return true
	default:
		return false
	}
}

func cliVersion() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("cauldron (version info unavailable)")
		return
	}

	revision := "unknown"
	dirty := ""
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}

	fmt.Printf("cauldron %s%s (built with %s)\n", revision, dirty, info.GoVersion)
}

// cliContext bundles backend services usable headlessly; every constructor here accepts a nil *application.App and degrades gracefully.
type cliContext struct {
	db                 *services.DatabaseService
	settings           *services.SettingsService
	envService         *services.EnvironmentService
	uvService          *services.UvService
	dockerImageBuilder *services.DockerImageBuilder
	pluginLoaderV2     *services.PluginLoaderV2
	pluginInstaller    *services.PluginInstaller
	pluginExecutor     *services.PluginExecutor
	jobQueue           *services.JobQueueService
	backupService      *services.BackupService
}

// close shuts down the job queue (killing any in-flight subprocess first, same as App.Shutdown) before closing the database.
func (c *cliContext) close() {
	if c.jobQueue != nil {
		c.jobQueue.Shutdown()
	}
	c.db.Close()
}

func newCLIContext() (*cliContext, error) {
	return newCLIContextWithPluginsDir("")
}

// newCLIContextWithPluginsDir is newCLIContext with an explicit plugins directory, used by tests to point at the repo's built-in plugins/ folder.
func newCLIContextWithPluginsDir(pluginsDir string) (*cliContext, error) {
	userDataPath, err := getUserDataPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get user data path: %w", err)
	}

	db, err := services.NewDatabaseServiceV3(userDataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	settings := services.NewSettingsServiceV3(db)
	fileService := services.NewFileServiceV3(nil)
	envService := services.NewEnvironmentServiceV3(db, settings, services.NewProgressNotifierV3(nil))
	uvService := services.NewUvServiceV3(fileService, db, settings, nil)
	gitAuthService := services.NewGitAuthService(db)
	dockerImageBuilder := services.NewDockerImageBuilder(db)

	pluginLoaderV2 := services.NewPluginLoaderV2(pluginsDir, db, dockerImageBuilder)
	if err := pluginLoaderV2.LoadPlugins(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load plugins: %v\n", err)
	}

	if pluginsDir == "" {
		exePath, _ := os.Executable()
		pluginsDir = filepath.Join(filepath.Dir(exePath), "plugins")
	}
	pluginInstaller := services.NewPluginInstallerV3(pluginsDir, db, pluginLoaderV2, gitAuthService, nil)
	pluginExecutor := services.NewPluginExecutor()

	jobQueue := services.NewJobQueueServiceV3(db, nil)
	scriptExecutor := services.NewScriptExecutor(settings, db)
	scriptExecutor.SetUpdateCallback(func(jobID string, update models.Job) {
		job, err := jobQueue.GetJob(jobID)
		if err != nil {
			return
		}
		if update.Status != "" {
			job.Status = update.Status
		}
		if update.Progress > 0 {
			job.Progress = update.Progress
		}
		if update.Error != "" {
			job.Error = update.Error
		}
		if update.OutputPath != "" {
			job.OutputPath = update.OutputPath
		}
		if update.Status == "completed" {
			now := time.Now()
			job.CompletedAt = &now
		}
		db.GetDB().Model(job).Select("*").Updates(job)
	})
	scriptExecutor.SetOutputCallback(func(jobID string, line string) {
		job, err := jobQueue.GetJob(jobID)
		if err != nil {
			return
		}
		job.TerminalOutput = append(job.TerminalOutput, line)
		const maxLines = 100
		if len(job.TerminalOutput) > maxLines {
			job.TerminalOutput = job.TerminalOutput[len(job.TerminalOutput)-maxLines:]
		}
		db.GetDB().Model(job).Select("*").Updates(job)
	})
	scriptExecutor.SetPluginLoader(pluginLoaderV2)
	jobQueue.SetScriptExecutor(scriptExecutor)
	jobQueue.SetPluginLoader(pluginLoaderV2)

	return &cliContext{
		db:                 db,
		settings:           settings,
		envService:         envService,
		uvService:          uvService,
		dockerImageBuilder: dockerImageBuilder,
		pluginLoaderV2:     pluginLoaderV2,
		pluginInstaller:    pluginInstaller,
		pluginExecutor:     pluginExecutor,
		jobQueue:           jobQueue,
		backupService:      services.NewBackupService(db),
	}, nil
}

func cliPlugin(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cauldron plugin install <repo-url> [--ref <commit>] [--registry <source>] | cauldron plugin install --file <path> | cauldron plugin uninstall <repo-url> | cauldron plugin list | cauldron plugin inputs [--json] <plugin-id>")
	}

	switch args[0] {
	case "install":
		return cliPluginInstall(args[1:])
	case "uninstall":
		return cliPluginUninstall(args[1:])
	case "list":
		return cliPluginList()
	case "inputs":
		return cliPluginInputs(args[1:])
	default:
		return fmt.Errorf("unknown plugin subcommand: %s", args[0])
	}
}

func cliPluginList() error {
	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	plugins := ctx.pluginLoaderV2.GetAllPlugins()
	if len(plugins) == 0 {
		fmt.Println("No plugins installed.")
		return nil
	}
	for _, p := range plugins {
		fmt.Printf("%-4d %-32s %s\n", p.ID, p.Definition.Plugin.ID, p.Definition.Plugin.Name)
	}
	return nil
}

// cliPluginInputs prints a plugin's parameter schema, for building --param/--params-json values for "cauldron job run".
func cliPluginInputs(args []string) error {
	fs := flag.NewFlagSet("plugin inputs", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "output the raw input schema as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: cauldron plugin inputs [--json] <plugin-id>")
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("unexpected extra arguments %v -- flags must come before the plugin id", fs.Args()[1:])
	}

	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	plugin, err := findPlugin(ctx.pluginLoaderV2, fs.Arg(0))
	if err != nil {
		return err
	}

	if *asJSON {
		data, err := json.MarshalIndent(plugin.Definition.Inputs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal inputs: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Plugin: %s (%s)\n", plugin.Definition.Plugin.Name, plugin.Definition.Plugin.ID)
	if plugin.Definition.Plugin.Description != "" {
		fmt.Println(plugin.Definition.Plugin.Description)
	}
	fmt.Println()

	if len(plugin.Definition.Inputs) == 0 {
		fmt.Println("This plugin takes no inputs.")
		return nil
	}

	for _, input := range plugin.Definition.Inputs {
		required := ""
		if input.Required {
			required = " (required)"
		}
		fmt.Printf("--param %s=<%s>%s\n", input.Name, input.Type, required)
		if input.Label != "" && input.Label != input.Name {
			fmt.Printf("    label:       %s\n", input.Label)
		}
		if input.Description != "" {
			fmt.Printf("    description: %s\n", input.Description)
		}
		if input.Default != nil {
			fmt.Printf("    default:     %v\n", input.Default)
		}
		if input.Placeholder != "" {
			fmt.Printf("    example:     %s\n", input.Placeholder)
		}
		if len(input.Options) > 0 {
			opts := make([]string, len(input.Options))
			for i, o := range input.Options {
				opts[i] = o.Value
			}
			fmt.Printf("    options:     %s\n", strings.Join(opts, ", "))
		}
		if input.Min != nil {
			fmt.Printf("    min:         %v\n", *input.Min)
		}
		if input.Max != nil {
			fmt.Printf("    max:         %v\n", *input.Max)
		}
		fmt.Println()
	}
	return nil
}

func cliPluginUninstall(args []string) error {
	fs := flag.NewFlagSet("plugin uninstall", flag.ContinueOnError)
	deleteJobHistory := fs.Bool("delete-job-history", false, "also delete job history for this plugin")
	deleteEnvironments := fs.Bool("delete-environments", false, "also delete environments bound to this plugin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: cauldron plugin uninstall <repo-url> [--delete-job-history] [--delete-environments]")
	}

	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	repoURL := fs.Arg(0)
	fmt.Printf("Uninstalling %s...\n", repoURL)
	if err := ctx.pluginInstaller.UninstallPlugin(repoURL, services.UninstallOptions{
		DeleteJobHistory:   *deleteJobHistory,
		DeleteEnvironments: *deleteEnvironments,
	}); err != nil {
		return fmt.Errorf("uninstall failed: %w", err)
	}
	fmt.Println("  OK")
	return nil
}

func cliPluginInstall(args []string) error {
	fs := flag.NewFlagSet("plugin install", flag.ContinueOnError)
	ref := fs.String("ref", "", "commit hash / ref to install")
	registry := fs.String("registry", "", "registry source label")
	file := fs.String("file", "", "path to a file with one repository URL per line, for batch install")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var repoURLs []string
	if *file != "" {
		lines, err := readNonEmptyLines(*file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", *file, err)
		}
		repoURLs = lines
	} else {
		if fs.NArg() == 0 {
			return fmt.Errorf("usage: cauldron plugin install <repo-url> [--ref <commit>] [--registry <source>]")
		}
		repoURLs = []string{fs.Arg(0)}
	}

	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	var registrySource *string
	if *registry != "" {
		registrySource = registry
	}

	failed := 0
	for _, repoURL := range repoURLs {
		fmt.Printf("Installing %s...\n", repoURL)
		pluginID, err := ctx.pluginInstaller.InstallPlugin(repoURL, *ref, registrySource, func(status string) {
			fmt.Printf("  %s\n", status)
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  FAILED: %v\n", err)
			failed++
			continue
		}
		fmt.Printf("  OK (plugin id: %s)\n", pluginID)
	}

	if failed > 0 {
		return fmt.Errorf("%d/%d plugin(s) failed to install", failed, len(repoURLs))
	}
	return nil
}

func readNonEmptyLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

func cliDoctor() error {
	fmt.Println("Cauldron Doctor")
	fmt.Println("===============")

	ctx, err := newCLIContext()
	if err != nil {
		fmt.Printf("[FAIL]    Startup                  %v\n", err)
		return err
	}
	defer ctx.close()

	report := func(ok bool, label string, detail string) {
		status := "OK"
		if !ok {
			status = "MISSING"
		}
		fmt.Printf("[%-7s] %-24s %s\n", status, label, detail)
	}

	report(true, "Database", "opened successfully")

	pyEnvs, err := ctx.envService.DetectPythonEnvironments()
	if err != nil || len(pyEnvs) == 0 {
		report(false, "Python", "no Python environment detected")
	} else {
		report(true, "Python", fmt.Sprintf("%d environment(s) detected (%s)", len(pyEnvs), pyEnvs[0].Path))
	}

	rEnvs, err := ctx.envService.DetectREnvironments()
	if err != nil || len(rEnvs) == 0 {
		report(false, "R", "no R environment detected")
	} else {
		report(true, "R", fmt.Sprintf("%d environment(s) detected (%s)", len(rEnvs), rEnvs[0].Path))
	}

	if ctx.uvService.IsUvAvailable() {
		if uvPath, err := ctx.uvService.GetUvPath(); err == nil {
			report(true, "uv", uvPath)
		} else if systemPath, err := exec.LookPath("uv"); err == nil {
			report(true, "uv", systemPath+" (system PATH)")
		} else {
			report(true, "uv", "available")
		}
	} else {
		report(false, "uv", "not installed (install from Settings > Python)")
	}

	if err := ctx.dockerImageBuilder.CheckDockerAvailable(); err != nil {
		report(false, "Docker", err.Error())
	} else {
		report(true, "Docker", "daemon reachable")
	}

	plugins := ctx.pluginLoaderV2.GetAllPlugins()
	report(true, "Plugins", fmt.Sprintf("%d loaded", len(plugins)))

	return nil
}

func cliUv(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cauldron uv install | cauldron uv python install <version> | cauldron uv python list | cauldron uv venv create <path> [--python <version>]")
	}

	switch args[0] {
	case "install":
		return cliUvInstall()
	case "python":
		return cliUvPython(args[1:])
	case "venv":
		return cliUvVenv(args[1:])
	default:
		return fmt.Errorf("unknown uv subcommand: %s", args[0])
	}
}

func cliUvInstall() error {
	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	if ctx.uvService.IsUvAvailable() {
		path, _ := ctx.uvService.GetUvPath()
		if path == "" {
			path = "(system PATH)"
		}
		fmt.Printf("uv is already available: %s\n", path)
		return nil
	}

	fmt.Println("Downloading uv...")
	if err := ctx.uvService.DownloadUv(); err != nil {
		return fmt.Errorf("failed to install uv: %w", err)
	}
	path, _ := ctx.uvService.GetUvPath()
	fmt.Printf("OK: uv installed at %s\n", path)
	return nil
}

func cliUvPython(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cauldron uv python install <version> | cauldron uv python list")
	}

	switch args[0] {
	case "install":
		if len(args) < 2 {
			return fmt.Errorf("usage: cauldron uv python install <version>")
		}
		return cliUvPythonInstall(args[1])
	case "list":
		return cliUvPythonList()
	default:
		return fmt.Errorf("unknown uv python subcommand: %s", args[0])
	}
}

func cliUvPythonInstall(version string) error {
	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	if !ctx.uvService.IsUvAvailable() {
		return fmt.Errorf("uv is not installed; run 'cauldron uv install' first")
	}

	fmt.Printf("Installing Python %s via uv...\n", version)
	if err := ctx.uvService.InstallUvPythonVersion(version); err != nil {
		return fmt.Errorf("failed to install Python %s: %w", version, err)
	}
	fmt.Println("  OK")
	return nil
}

func cliUvPythonList() error {
	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	if !ctx.uvService.IsUvAvailable() {
		return fmt.Errorf("uv is not installed; run 'cauldron uv install' first")
	}

	versions, err := ctx.uvService.ListUvManagedPythons()
	if err != nil {
		return fmt.Errorf("failed to list uv-managed Python versions: %w", err)
	}
	if len(versions) == 0 {
		fmt.Println("No Python versions installed via uv.")
		return nil
	}
	for _, v := range versions {
		fmt.Printf("%-12s %-14s %s\n", v.Version, v.Implementation, v.Path)
	}
	return nil
}

func cliUvVenv(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cauldron uv venv create <path> [--python <version>]")
	}

	switch args[0] {
	case "create":
		return cliUvVenvCreate(args[1:])
	default:
		return fmt.Errorf("unknown uv venv subcommand: %s", args[0])
	}
}

func cliUvVenvCreate(args []string) error {
	fs := flag.NewFlagSet("uv venv create", flag.ContinueOnError)
	pythonVersion := fs.String("python", "", "Python version to use (uv-managed)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: cauldron uv venv create <path> [--python <version>]")
	}
	venvPath := fs.Arg(0)

	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	if !ctx.uvService.IsUvAvailable() {
		return fmt.Errorf("uv is not installed; run 'cauldron uv install' first")
	}

	fmt.Printf("Creating venv at %s...\n", venvPath)
	if err := ctx.uvService.CreateUvVirtualEnv(*pythonVersion, venvPath, "", ""); err != nil {
		return fmt.Errorf("failed to create venv: %w", err)
	}
	fmt.Println("  OK")
	return nil
}

// jobSpec describes one plugin invocation: a single --param/--params-json call, or one entry of a --file batch (a JSON array of jobSpec).
type jobSpec struct {
	Plugin string                 `json:"plugin"`
	Params map[string]interface{} `json:"params"`
}

// stringSliceFlag implements flag.Value to accept a repeatable string flag, used for --param key=value.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ",") }
func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func cliJob(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cauldron job run [--param key=value]... <plugin-id> | cauldron job run --file <path> | cauldron job list | cauldron job status <job-id>")
	}

	switch args[0] {
	case "run":
		return cliJobRun(args[1:])
	case "list":
		return cliJobList()
	case "status":
		return cliJobStatus(args[1:])
	default:
		return fmt.Errorf("unknown job subcommand: %s", args[0])
	}
}

func cliJobRun(args []string) error {
	fs := flag.NewFlagSet("job run", flag.ContinueOnError)
	var paramFlags stringSliceFlag
	fs.Var(&paramFlags, "param", "a key=value parameter (repeatable); coerced to the plugin input's declared type")
	paramsJSON := fs.String("params-json", "", "parameters as a raw JSON object")
	file := fs.String("file", "", "path to a JSON array of {\"plugin\":..., \"params\":{...}} job specs, for batch processing")
	timeout := fs.Duration("timeout", 30*time.Minute, "max time to wait for each job to finish")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var specs []jobSpec
	if *file != "" {
		data, err := os.ReadFile(*file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", *file, err)
		}
		if err := json.Unmarshal(data, &specs); err != nil {
			return fmt.Errorf("failed to parse %s: %w", *file, err)
		}
	} else {
		if fs.NArg() == 0 {
			return fmt.Errorf("usage: cauldron job run [--param key=value]... [--params-json '{...}'] <plugin-id>")
		}
		if fs.NArg() > 1 {
			// flag.Parse stops recognizing flags at the first positional argument, so a misplaced --param lands here instead of erroring cleanly.
			return fmt.Errorf("unexpected extra arguments %v -- flags must come before the plugin id, e.g. cauldron job run --param key=value cv-plot", fs.Args()[1:])
		}

		params := map[string]interface{}{}
		if *paramsJSON != "" {
			if err := json.Unmarshal([]byte(*paramsJSON), &params); err != nil {
				return fmt.Errorf("invalid --params-json: %w", err)
			}
		}
		for _, kv := range paramFlags {
			key, value, ok := strings.Cut(kv, "=")
			if !ok {
				return fmt.Errorf("invalid --param %q, expected key=value", kv)
			}
			params[key] = value
		}

		specs = []jobSpec{{Plugin: fs.Arg(0), Params: params}}
	}
	if len(specs) == 0 {
		return fmt.Errorf("no jobs to run")
	}

	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	var jobIDs []string
	for _, spec := range specs {
		plugin, err := findPlugin(ctx.pluginLoaderV2, spec.Plugin)
		if err != nil {
			return fmt.Errorf("plugin %q: %w", spec.Plugin, err)
		}

		coerced, err := coercePluginParams(plugin, spec.Params)
		if err != nil {
			return fmt.Errorf("plugin %q: %w", spec.Plugin, err)
		}

		jobID, err := executePluginJob(ctx.pluginExecutor, ctx.jobQueue, ctx.settings, plugin, coerced)
		if err != nil {
			return fmt.Errorf("failed to start job for plugin %q: %w", spec.Plugin, err)
		}
		fmt.Printf("Queued job %s for plugin %s (%s)\n", jobID, plugin.Definition.Plugin.ID, plugin.Definition.Plugin.Name)
		jobIDs = append(jobIDs, jobID)
	}

	failed := 0
	for _, jobID := range jobIDs {
		job, err := waitForJob(ctx, jobID, *timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", jobID, err)
			failed++
			continue
		}
		fmt.Printf("  %s: %s", jobID, job.Status)
		if job.OutputPath != "" {
			fmt.Printf(" (output: %s)", job.OutputPath)
		}
		fmt.Println()
		if job.Status != models.JobStatusCompleted {
			if job.Error != "" {
				fmt.Fprintf(os.Stderr, "    error: %s\n", job.Error)
			}
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d/%d job(s) did not complete successfully", failed, len(jobIDs))
	}
	return nil
}

// findPlugin resolves a CLI-supplied plugin reference: a numeric database ID, or the plugin's string id/name from its plugin.yaml.
func findPlugin(loader *services.PluginLoaderV2, ref string) (*models.PluginV2, error) {
	if id, err := strconv.ParseUint(ref, 10, 64); err == nil {
		if plugin, err := loader.GetPlugin(uint(id)); err == nil {
			return plugin, nil
		}
	}
	for _, p := range loader.GetAllPlugins() {
		if p.Definition.Plugin.ID == ref || p.Definition.Plugin.Name == ref {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no plugin found matching %q", ref)
}

// coercePluginParams converts --param key=value strings into the Go type the plugin's declared input expects (float64/bool/[]interface{}); values already typed via --params-json or --file pass through unchanged.
func coercePluginParams(plugin *models.PluginV2, params map[string]interface{}) (map[string]interface{}, error) {
	coerced := make(map[string]interface{}, len(params))
	for key, value := range params {
		strValue, isString := value.(string)
		if !isString {
			coerced[key] = value
			continue
		}

		var input *models.PluginInputV2
		for i := range plugin.Definition.Inputs {
			if plugin.Definition.Inputs[i].Name == key {
				input = &plugin.Definition.Inputs[i]
				break
			}
		}
		if input == nil {
			coerced[key] = strValue
			continue
		}

		switch input.Type {
		case models.PluginInputTypeNumber:
			f, err := strconv.ParseFloat(strValue, 64)
			if err != nil {
				return nil, fmt.Errorf("--param %s: expected a number, got %q", key, strValue)
			}
			coerced[key] = f
		case models.PluginInputTypeBoolean:
			b, err := strconv.ParseBool(strValue)
			if err != nil {
				return nil, fmt.Errorf("--param %s: expected true/false, got %q", key, strValue)
			}
			coerced[key] = b
		case models.PluginInputTypeMultiSelect:
			parts := strings.Split(strValue, ",")
			arr := make([]interface{}, len(parts))
			for i, p := range parts {
				arr[i] = strings.TrimSpace(p)
			}
			coerced[key] = arr
		default:
			coerced[key] = strValue
		}
	}
	return coerced, nil
}

// waitForJob polls a job until it reaches a terminal status, printing new output lines as they appear, since the process must stay alive for the queue's workers to run it at all.
func waitForJob(ctx *cliContext, jobID string, timeout time.Duration) (*models.Job, error) {
	deadline := time.Now().Add(timeout)
	printedLines := 0

	for {
		job, err := ctx.jobQueue.GetJob(jobID)
		if err != nil {
			return nil, fmt.Errorf("failed to look up job: %w", err)
		}

		if len(job.TerminalOutput) > printedLines {
			for _, line := range job.TerminalOutput[printedLines:] {
				fmt.Println("   ", line)
			}
			printedLines = len(job.TerminalOutput)
		}

		if job.Status == models.JobStatusCompleted || job.Status == models.JobStatusFailed {
			return job, nil
		}

		if time.Now().After(deadline) {
			return job, fmt.Errorf("timed out waiting for job to finish (status: %s)", job.Status)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func cliJobList() error {
	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	jobs := ctx.jobQueue.GetAllJobs()
	if len(jobs) == 0 {
		fmt.Println("No jobs found.")
		return nil
	}
	for _, job := range jobs {
		fmt.Printf("%-38s %-24s %-12s %s\n", job.ID, job.Name, job.Status, job.CreatedAt.Format(time.RFC3339))
	}
	return nil
}

func cliJobStatus(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cauldron job status <job-id>")
	}

	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	job, err := ctx.jobQueue.GetJob(args[0])
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	fmt.Printf("ID:       %s\n", job.ID)
	fmt.Printf("Name:     %s\n", job.Name)
	fmt.Printf("Status:   %s\n", job.Status)
	fmt.Printf("Progress: %.0f%%\n", job.Progress)
	fmt.Printf("Created:  %s\n", job.CreatedAt.Format(time.RFC3339))
	if job.OutputPath != "" {
		fmt.Printf("Output:   %s\n", job.OutputPath)
	}
	if job.Error != "" {
		fmt.Printf("Error:    %s\n", job.Error)
	}
	if len(job.TerminalOutput) > 0 {
		fmt.Println("Output log:")
		for _, line := range job.TerminalOutput {
			fmt.Println("  " + line)
		}
	}
	return nil
}

func cliDb(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cauldron db backup <path> [--include-secrets] | cauldron db restore <path>")
	}

	switch args[0] {
	case "backup":
		return cliDbBackup(args[1:])
	case "restore":
		return cliDbRestore(args[1:])
	default:
		return fmt.Errorf("unknown db subcommand: %s", args[0])
	}
}

// cliDbBackup writes settings + installed-plugin metadata to a JSON file. Custom env vars (which may hold secrets like API keys) are excluded unless --include-secrets is passed.
func cliDbBackup(args []string) error {
	fs := flag.NewFlagSet("db backup", flag.ContinueOnError)
	includeSecrets := fs.Bool("include-secrets", false, "also back up custom environment variables (may contain secrets)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: cauldron db backup <path> [--include-secrets]")
	}
	path := fs.Arg(0)

	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	data, err := ctx.backupService.CreateBackup(*includeSecrets)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	if err := services.WriteBackupFile(path, data); err != nil {
		return err
	}

	fmt.Printf("Backup written to %s\n", path)
	fmt.Printf("  %d setting(s), %d plugin(s)", len(data.Settings), len(data.Plugins))
	if *includeSecrets {
		fmt.Printf(", %d env var(s)", len(data.CustomEnvVars))
	}
	fmt.Println()
	if !*includeSecrets {
		fmt.Println("  (custom environment variables were not included; pass --include-secrets to include them)")
	} else {
		fmt.Println("  WARNING: this file contains secrets in cleartext -- store it securely")
	}
	return nil
}

// cliDbRestore applies settings, reinstalls missing remote plugins, and restores enabled state + env vars from a backup file.
func cliDbRestore(args []string) error {
	fs := flag.NewFlagSet("db restore", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: cauldron db restore <path>")
	}
	path := fs.Arg(0)

	data, err := services.ReadBackupFile(path)
	if err != nil {
		return err
	}

	ctx, err := newCLIContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer ctx.close()

	result, err := ctx.backupService.RestoreBackup(data, ctx.pluginInstaller, func(status string) {
		fmt.Println("  " + status)
	})
	if err != nil {
		return fmt.Errorf("restore failed partway through: %w", err)
	}

	fmt.Printf("Settings restored: %d\n", result.SettingsRestored)
	fmt.Printf("Plugins installed: %d\n", len(result.PluginsInstalled))
	for _, id := range result.PluginsInstalled {
		fmt.Printf("  + %s\n", id)
	}
	if len(result.PluginsSkipped) > 0 {
		fmt.Printf("Plugins skipped (builtin or missing repository): %d\n", len(result.PluginsSkipped))
		for _, id := range result.PluginsSkipped {
			fmt.Printf("  - %s\n", id)
		}
	}
	if len(result.PluginsFailed) > 0 {
		fmt.Printf("Plugins failed: %d\n", len(result.PluginsFailed))
		for id, msg := range result.PluginsFailed {
			fmt.Printf("  ! %s: %s\n", id, msg)
		}
	}
	if result.EnvVarsRestored > 0 {
		fmt.Printf("Env vars restored: %d\n", result.EnvVarsRestored)
	}

	if len(result.PluginsFailed) > 0 {
		return fmt.Errorf("%d plugin(s) failed to restore", len(result.PluginsFailed))
	}
	return nil
}
