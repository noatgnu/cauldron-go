package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/services"
)

// runCLI checks args (os.Args[1:]) for a recognized CLI subcommand and, if
// found, runs it headlessly (no GUI, no window) and returns true. Any
// unrecognized or empty args falls through so main() proceeds with the
// normal desktop GUI launch.
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
		// Backend services log through the standard `log` package; quiet
		// it by default so command output isn't drowned out.
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

// cliContext bundles the subset of backend services usable headlessly, i.e.
// without a running Wails application/window. Every service constructor
// here accepts a nil *application.App and degrades gracefully (progress
// notifications and dialogs are simply skipped).
type cliContext struct {
	db                 *services.DatabaseService
	settings           *services.SettingsService
	envService         *services.EnvironmentService
	uvService          *services.UvService
	dockerImageBuilder *services.DockerImageBuilder
	pluginLoaderV2     *services.PluginLoaderV2
	pluginInstaller    *services.PluginInstaller
}

func newCLIContext() (*cliContext, error) {
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

	pluginLoaderV2 := services.NewPluginLoaderV2("", db, dockerImageBuilder)
	if err := pluginLoaderV2.LoadPlugins(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load plugins: %v\n", err)
	}

	exePath, _ := os.Executable()
	pluginsDir := filepath.Join(filepath.Dir(exePath), "plugins")
	pluginInstaller := services.NewPluginInstallerV3(pluginsDir, db, pluginLoaderV2, gitAuthService, nil)

	return &cliContext{
		db:                 db,
		settings:           settings,
		envService:         envService,
		uvService:          uvService,
		dockerImageBuilder: dockerImageBuilder,
		pluginLoaderV2:     pluginLoaderV2,
		pluginInstaller:    pluginInstaller,
	}, nil
}

func cliPlugin(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cauldron plugin install <repo-url> [--ref <commit>] [--registry <source>] | cauldron plugin install --file <path>")
	}

	switch args[0] {
	case "install":
		return cliPluginInstall(args[1:])
	case "uninstall":
		return cliPluginUninstall(args[1:])
	default:
		return fmt.Errorf("unknown plugin subcommand: %s", args[0])
	}
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
	defer ctx.db.Close()

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
	defer ctx.db.Close()

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
	defer ctx.db.Close()

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
