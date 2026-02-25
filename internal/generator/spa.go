package generator

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/internal/parser"
	"github.com/noatgnu/cauldron-go/internal/webrpkg"
)

//go:embed templates/spa/*
var spaTemplates embed.FS

type SPAConfig struct {
	PluginPath      string
	OutputDir       string
	CheckOnly       bool
	SkipCheck       bool
	NoBuild         bool
	PyodideVersion  string
	WebRVersion     string
	GithubAction    bool
	ExamplesDir     string
	SharedLibPath   string
	SpaTemplatePath string
}

type SPAGenerator struct {
	config     SPAConfig
	definition *models.PluginDefinition
	pluginDir  string
}

type EnvironmentData struct {
	Production           bool
	Runtime              string
	PyodideVersion       string
	PyodidePackagesJSON  string
	WebrVersion          string
	WebrPackagesJSON     string
	PluginDefinitionJSON string
	PluginScript         string
	PluginModulesJSON    string
	HasExample           bool
	ExampleBasePath      string
}

type PackageJSONData struct {
	PluginID            string
	IsPython            bool
	IsR                 bool
	PyodideVersion      string
	PyodidePackagesJSON string
	WebrPackagesJSON    string
	SharedLibPath       string
}

type AngularJSONData struct {
	PluginID string
}

type IndexHTMLData struct {
	PluginName string
}

func NewSPAGenerator(config SPAConfig) *SPAGenerator {
	return &SPAGenerator{
		config: config,
	}
}

func (g *SPAGenerator) getPluginDir() string {
	info, err := os.Stat(g.config.PluginPath)
	if err != nil {
		return filepath.Dir(g.config.PluginPath)
	}
	if info.IsDir() {
		return g.config.PluginPath
	}
	return filepath.Dir(g.config.PluginPath)
}

func (g *SPAGenerator) isRPlugin() bool {
	if g.definition == nil {
		return false
	}
	return g.definition.Runtime.HasEnvironment("r") && !g.definition.Runtime.HasEnvironment("python")
}

func (g *SPAGenerator) isPythonPlugin() bool {
	if g.definition == nil {
		return false
	}
	return g.definition.Runtime.HasEnvironment("python")
}

func (g *SPAGenerator) CheckCompatibility() (bool, []string) {
	definition, err := parser.ParsePlugin(g.config.PluginPath)
	if err != nil {
		return false, []string{fmt.Sprintf("Failed to parse plugin: %v", err)}
	}

	pluginDir := g.getPluginDir()
	compat := CheckPyodideCompatibility(definition, pluginDir)

	return compat.Compatible, compat.Issues
}

func (g *SPAGenerator) Generate() error {
	definition, err := parser.ParsePlugin(g.config.PluginPath)
	if err != nil {
		return fmt.Errorf("failed to parse plugin: %w", err)
	}

	g.definition = definition
	g.pluginDir = g.getPluginDir()

	if !g.config.SkipCheck {
		if g.isRPlugin() {
			compat := CheckWebRCompatibility(definition, g.pluginDir)
			if !compat.Compatible {
				fmt.Println("Warning: Plugin has compatibility issues with WebR:")
				for _, issue := range compat.Issues {
					fmt.Printf("  - %s\n", issue)
				}
				fmt.Println("Continuing with generation anyway...")
			}

			if len(compat.MaybeSupport) > 0 {
				fmt.Println("Note: The following R packages may or may not be available in WebR:")
				for _, pkg := range compat.MaybeSupport {
					fmt.Printf("  - %s\n", pkg)
				}
			}
		} else {
			compat := CheckPyodideCompatibility(definition, g.pluginDir)
			if !compat.Compatible {
				fmt.Println("Warning: Plugin has compatibility issues with Pyodide:")
				for _, issue := range compat.Issues {
					fmt.Printf("  - %s\n", issue)
				}
				fmt.Println("Continuing with generation anyway...")
			}

			if len(compat.MaybeSupport) > 0 {
				fmt.Println("Note: The following packages may or may not be available in Pyodide:")
				for _, pkg := range compat.MaybeSupport {
					fmt.Printf("  - %s\n", pkg)
				}
			}
		}
	}

	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := g.copyTemplate(); err != nil {
		return fmt.Errorf("failed to copy template: %w", err)
	}

	if err := g.generateEnvironment(); err != nil {
		return fmt.Errorf("failed to generate environment.ts: %w", err)
	}

	if err := g.generatePackageJSON(); err != nil {
		return fmt.Errorf("failed to generate package.json: %w", err)
	}

	if err := g.generateAngularJSON(); err != nil {
		return fmt.Errorf("failed to generate angular.json: %w", err)
	}

	if err := g.generateIndexHTML(); err != nil {
		return fmt.Errorf("failed to generate index.html: %w", err)
	}

	if g.isPythonPlugin() {
		if err := g.generateLockScript(); err != nil {
			return fmt.Errorf("failed to generate lock script: %w", err)
		}
	}

	if err := g.copyExampleFiles(); err != nil {
		return fmt.Errorf("failed to copy example files: %w", err)
	}

	if g.config.GithubAction {
		if err := g.generateGithubWorkflow(); err != nil {
			return fmt.Errorf("failed to generate GitHub workflow: %w", err)
		}
	}

	if !g.config.NoBuild {
		fmt.Println("Running npm install...")
		cmd := exec.Command("npm", "install")
		cmd.Dir = g.config.OutputDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm install failed: %w", err)
		}

		fmt.Println("Building Angular app...")
		cmd = exec.Command("npm", "run", "build")
		cmd.Dir = g.config.OutputDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ng build failed: %w", err)
		}
	}

	return nil
}

func (g *SPAGenerator) copyTemplate() error {
	templatePath := g.config.SpaTemplatePath
	if templatePath == "" {
		execPath, err := os.Executable()
		if err == nil {
			templatePath = filepath.Join(filepath.Dir(execPath), "spa-template")
		}
	}

	if templatePath == "" || !dirExists(templatePath) {
		cwd, _ := os.Getwd()
		templatePath = filepath.Join(cwd, "spa-template")
	}

	if !dirExists(templatePath) {
		return fmt.Errorf("spa-template not found at %s", templatePath)
	}

	return copyDir(templatePath, g.config.OutputDir)
}

func (g *SPAGenerator) generateEnvironment() error {
	pyodidePackages := []string{}
	webrPackages := []map[string]string{}

	if g.isRPlugin() {
		basePackages := getRRequiredPackages(g.definition, g.pluginDir)
		fmt.Printf("  Resolving R package dependencies for: %v\n", basePackages)

		resolver, err := webrpkg.NewResolver(g.config.WebRVersion)
		if err != nil {
			return fmt.Errorf("failed to create WebR package resolver: %w", err)
		}

		fmt.Printf("  Using R version: %s\n", resolver.GetRVersion())
		result, err := resolver.Resolve(basePackages)
		if err != nil {
			return fmt.Errorf("failed to resolve package dependencies: %w", err)
		}

		fmt.Printf("  Resolved %d packages (including dependencies)\n", len(result.InstallOrder))

		unavailableSet := make(map[string]bool)
		for _, unavail := range result.Unavailable {
			unavailableSet[unavail] = true
		}

		var unavailableRequired []string
		for _, reqPkg := range basePackages {
			if unavailableSet[reqPkg] {
				unavailableRequired = append(unavailableRequired, reqPkg)
			}
		}

		if len(unavailableRequired) > 0 {
			fmt.Printf("\n  Error: The following required packages cannot be installed in WebR:\n")
			for _, pkg := range unavailableRequired {
				fmt.Printf("    - %s\n", pkg)
			}
			fmt.Printf("\n  These packages (or their dependencies) do not have WebR/WASM binaries available.\n")
			fmt.Printf("  This plugin cannot run in a browser-based SPA.\n")
			fmt.Printf("  Use the desktop CauldronGO application instead.\n\n")
			return fmt.Errorf("cannot generate WebR SPA: %d required packages unavailable: %v", len(unavailableRequired), unavailableRequired)
		}

		for _, pkgName := range result.InstallOrder {
			for _, pkgInfo := range result.ResolvedPkgs {
				if pkgInfo.Name == pkgName && pkgInfo.Available {
					webrPackages = append(webrPackages, map[string]string{
						"name": pkgInfo.Name,
						"repo": pkgInfo.Repository,
					})
					break
				}
			}
		}
	} else {
		pyodidePackages = getRequiredPackages(g.definition, g.pluginDir)
	}

	pyodidePackagesJSON, _ := json.Marshal(pyodidePackages)
	webrPackagesJSON, _ := json.Marshal(webrPackages)

	defCopy := *g.definition

	if defCopy.Plots != nil {
		plotsCopy := make([]models.PluginPlot, len(defCopy.Plots))
		for i, plot := range defCopy.Plots {
			plotsCopy[i] = plot
			if plotsCopy[i].Customization == nil {
				plotsCopy[i].Customization = []models.PlotCustomization{}
			}
		}
		defCopy.Plots = plotsCopy
	}

	if defCopy.Example != nil && defCopy.Example.Values != nil {
		newValues := make(map[string]interface{})
		for k, v := range defCopy.Example.Values {
			if strVal, ok := v.(string); ok {
				if strings.HasPrefix(strVal, "examples/") {
					newValues[k] = strings.TrimPrefix(strVal, "examples/")
				} else if strings.HasPrefix(strVal, "examples\\") {
					newValues[k] = strings.TrimPrefix(strVal, "examples\\")
				} else {
					newValues[k] = v
				}
			} else {
				newValues[k] = v
			}
		}
		defCopy.Example = &models.ExampleData{
			Enabled: defCopy.Example.Enabled,
			Values:  newValues,
		}
	}

	definitionJSON, err := json.MarshalIndent(&defCopy, "  ", "  ")
	if err != nil {
		return err
	}

	pluginScript, err := g.readPluginScript()
	if err != nil {
		return err
	}

	pluginModules, err := g.findLocalModules()
	if err != nil {
		return err
	}
	pluginModulesJSON, _ := json.Marshal(pluginModules)

	runtime := "pyodide"
	if g.isRPlugin() {
		runtime = "webr"
	}

	data := EnvironmentData{
		Production:           true,
		Runtime:              runtime,
		PyodideVersion:       g.config.PyodideVersion,
		PyodidePackagesJSON:  string(pyodidePackagesJSON),
		WebrVersion:          g.config.WebRVersion,
		WebrPackagesJSON:     string(webrPackagesJSON),
		PluginDefinitionJSON: string(definitionJSON),
		PluginScript:         pluginScript,
		PluginModulesJSON:    string(pluginModulesJSON),
		HasExample:           g.hasExampleInputs(),
		ExampleBasePath:      "assets/examples/",
	}

	tmpl, err := template.ParseFS(spaTemplates, "templates/spa/environment.ts.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse environment template: %w", err)
	}

	envPath := filepath.Join(g.config.OutputDir, "src", "environments", "environment.ts")
	if err := os.MkdirAll(filepath.Dir(envPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(envPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func (g *SPAGenerator) generatePackageJSON() error {
	pyodidePackages := []string{}
	webrPackages := []map[string]string{}

	if g.isRPlugin() {
		basePackages := getRRequiredPackages(g.definition, g.pluginDir)
		for _, pkg := range basePackages {
			webrPackages = append(webrPackages, map[string]string{"name": pkg, "repo": ""})
		}
	} else {
		pyodidePackages = getRequiredPackages(g.definition, g.pluginDir)
	}

	pyodidePackagesJSON, _ := json.Marshal(pyodidePackages)
	webrPackagesJSON, _ := json.Marshal(webrPackages)

	sharedLibPath := g.config.SharedLibPath
	if sharedLibPath == "" {
		sharedLibPath = "../cauldron-go/shared-lib/dist/cauldron-forms"
	}

	data := PackageJSONData{
		PluginID:            g.definition.Plugin.ID,
		IsPython:            g.isPythonPlugin(),
		IsR:                 g.isRPlugin(),
		PyodideVersion:      g.config.PyodideVersion,
		PyodidePackagesJSON: string(pyodidePackagesJSON),
		WebrPackagesJSON:    string(webrPackagesJSON),
		SharedLibPath:       sharedLibPath,
	}

	tmpl, err := template.ParseFS(spaTemplates, "templates/spa/package.json.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse package.json template: %w", err)
	}

	pkgPath := filepath.Join(g.config.OutputDir, "package.json")
	f, err := os.Create(pkgPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func (g *SPAGenerator) generateAngularJSON() error {
	data := AngularJSONData{
		PluginID: g.definition.Plugin.ID,
	}

	tmpl, err := template.ParseFS(spaTemplates, "templates/spa/angular.json.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse angular.json template: %w", err)
	}

	angularPath := filepath.Join(g.config.OutputDir, "angular.json")
	f, err := os.Create(angularPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func (g *SPAGenerator) generateIndexHTML() error {
	data := IndexHTMLData{
		PluginName: g.definition.Plugin.Name,
	}

	tmpl, err := template.ParseFS(spaTemplates, "templates/spa/index.html.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse index.html template: %w", err)
	}

	htmlPath := filepath.Join(g.config.OutputDir, "src", "index.html")
	f, err := os.Create(htmlPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func (g *SPAGenerator) generateLockScript() error {
	packages := getRequiredPackages(g.definition, g.pluginDir)
	packagesJSON, _ := json.Marshal(packages)

	data := struct {
		PyodidePackagesJSON string
	}{
		PyodidePackagesJSON: string(packagesJSON),
	}

	tmpl, err := template.ParseFS(spaTemplates, "templates/spa/generate-lock.mjs.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse generate-lock.mjs template: %w", err)
	}

	scriptsDir := filepath.Join(g.config.OutputDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return err
	}

	lockPath := filepath.Join(scriptsDir, "generate-lock.mjs")
	f, err := os.Create(lockPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func (g *SPAGenerator) readPluginScript() (string, error) {
	scriptPath := g.definition.Runtime.GetEntrypoint()
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(g.pluginDir, scriptPath)
	}

	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read plugin script: %w", err)
	}

	escapedContent := strings.ReplaceAll(string(scriptContent), "\\", "\\\\")
	escapedContent = strings.ReplaceAll(escapedContent, "`", "\\`")
	escapedContent = strings.ReplaceAll(escapedContent, "${", "\\${")

	return escapedContent, nil
}

func (g *SPAGenerator) findLocalModules() (map[string]string, error) {
	if g.isRPlugin() {
		return g.findLocalRModules()
	}
	return g.findLocalPythonModules()
}

func (g *SPAGenerator) findLocalPythonModules() (map[string]string, error) {
	modules := make(map[string]string)

	entrypointPath := g.definition.Runtime.GetEntrypoint()
	if !filepath.IsAbs(entrypointPath) {
		entrypointPath = filepath.Join(g.pluginDir, entrypointPath)
	}
	entrypointPath, _ = filepath.Abs(entrypointPath)

	err := filepath.WalkDir(g.pluginDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "__pycache__" || d.Name() == "venv" || d.Name() == ".venv" || d.Name() == "examples" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) == ".py" {
			absPath, _ := filepath.Abs(path)
			if absPath == entrypointPath {
				return nil
			}

			relPath, err := filepath.Rel(g.pluginDir, path)
			if err != nil {
				return nil
			}

			moduleName := strings.TrimSuffix(relPath, ".py")
			moduleName = strings.ReplaceAll(moduleName, "\\", "/")

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			escapedContent := strings.ReplaceAll(string(content), "\\", "\\\\")
			escapedContent = strings.ReplaceAll(escapedContent, "`", "\\`")
			escapedContent = strings.ReplaceAll(escapedContent, "${", "\\${")
			modules[moduleName] = escapedContent
		}
		return nil
	})

	return modules, err
}

func extractLocalImports(content string) []string {
	var imports []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "from ") && strings.Contains(line, " import ") {
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 2 {
				moduleName := parts[1]
				if !strings.Contains(moduleName, ".") {
					imports = append(imports, moduleName)
				}
			}
		} else if strings.HasPrefix(line, "import ") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) >= 2 {
				moduleName := strings.Split(parts[1], ",")[0]
				moduleName = strings.Split(moduleName, " ")[0]
				moduleName = strings.TrimSpace(moduleName)
				if !strings.Contains(moduleName, ".") {
					imports = append(imports, moduleName)
				}
			}
		}
	}

	return imports
}

func (g *SPAGenerator) findLocalRModules() (map[string]string, error) {
	modules := make(map[string]string)

	entrypointPath := g.definition.Runtime.GetEntrypoint()
	if !filepath.IsAbs(entrypointPath) {
		entrypointPath = filepath.Join(g.pluginDir, entrypointPath)
	}
	entrypointPath, _ = filepath.Abs(entrypointPath)

	err := filepath.WalkDir(g.pluginDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == ".Rproj.user" || d.Name() == "examples" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) == ".R" || filepath.Ext(path) == ".r" {
			absPath, _ := filepath.Abs(path)
			if absPath == entrypointPath {
				return nil
			}

			relPath, err := filepath.Rel(g.pluginDir, path)
			if err != nil {
				return nil
			}

			moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
			moduleName = strings.ReplaceAll(moduleName, "\\", "/")

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			escapedContent := strings.ReplaceAll(string(content), "\\", "\\\\")
			escapedContent = strings.ReplaceAll(escapedContent, "`", "\\`")
			escapedContent = strings.ReplaceAll(escapedContent, "${", "\\${")
			modules[moduleName] = escapedContent
		}
		return nil
	})

	return modules, err
}

func extractRSourceCalls(content string) []string {
	var sources []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "source(") || strings.Contains(line, " source(") {
			start := strings.Index(line, "source(")
			if start >= 0 {
				rest := line[start+7:]
				if idx := strings.Index(rest, ")"); idx > 0 {
					arg := rest[:idx]
					arg = strings.Trim(arg, "\"' ")
					if strings.HasSuffix(arg, ".R") {
						sources = append(sources, arg)
					}
				}
			}
		}
	}

	return sources
}

func (g *SPAGenerator) hasExampleInputs() bool {
	if g.definition.Example == nil || !g.definition.Example.Enabled {
		return false
	}

	if g.definition.Example.Values == nil {
		return false
	}

	fileInputs := make(map[string]bool)
	for _, input := range g.definition.Inputs {
		if input.Type == "file" {
			fileInputs[input.Name] = true
		}
	}

	for key, value := range g.definition.Example.Values {
		if fileInputs[key] {
			if strVal, ok := value.(string); ok && strVal != "" {
				return true
			}
		}
	}
	return false
}

func (g *SPAGenerator) copyExampleFiles() error {
	if g.definition.Example == nil || !g.definition.Example.Enabled {
		return nil
	}

	examplesDir := filepath.Join(g.config.OutputDir, "src", "assets", "examples")

	for _, value := range g.definition.Example.Values {
		strVal, ok := value.(string)
		if !ok {
			continue
		}

		srcPath := g.resolveExampleFilePath(strVal)
		if srcPath == "" {
			continue
		}

		normalizedPath := strVal
		if strings.HasPrefix(strVal, "examples/") || strings.HasPrefix(strVal, "examples\\") {
			normalizedPath = strVal[9:]
		}

		dstPath := filepath.Join(examplesDir, normalizedPath)

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		srcContent, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}

		if err := os.WriteFile(dstPath, srcContent, 0644); err != nil {
			return err
		}
	}

	return nil
}

func (g *SPAGenerator) resolveExampleFilePath(filePath string) string {
	if filepath.IsAbs(filePath) {
		if _, err := os.Stat(filePath); err == nil {
			return filePath
		}
		return ""
	}

	normalizedPath := filePath
	if strings.HasPrefix(filePath, "examples/") || strings.HasPrefix(filePath, "examples\\") {
		normalizedPath = filePath[9:]
	}

	pluginExamplesPath := filepath.Join(g.pluginDir, "examples", normalizedPath)
	if _, err := os.Stat(pluginExamplesPath); err == nil {
		return pluginExamplesPath
	}

	if g.config.ExamplesDir != "" {
		for _, dir := range strings.Split(g.config.ExamplesDir, ":") {
			dir = strings.TrimSpace(dir)
			if dir == "" {
				continue
			}
			configExamplesPath := filepath.Join(dir, normalizedPath)
			if _, err := os.Stat(configExamplesPath); err == nil {
				return configExamplesPath
			}
		}
	}

	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		execExamplesPath := filepath.Join(execDir, "examples", normalizedPath)
		if _, err := os.Stat(execExamplesPath); err == nil {
			return execExamplesPath
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		cwdExamplesPath := filepath.Join(cwd, "examples", normalizedPath)
		if _, err := os.Stat(cwdExamplesPath); err == nil {
			return cwdExamplesPath
		}
	}

	return ""
}

func (g *SPAGenerator) generateGithubWorkflow() error {
	workflowDir := filepath.Join(g.config.OutputDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return err
	}

	hasExample := g.definition.Example != nil && g.definition.Example.Enabled && g.hasExampleInputs()
	integrationTestStep := ""
	if hasExample {
		integrationTestStep = `
      - name: Run integration tests with example
        run: npm run test:ci:integration
        continue-on-error: false
        env:
          CHROME_BIN: /usr/bin/google-chrome
`
	}

	content := `name: Deploy to GitHub Pages

on:
  push:
    branches: [ main, master ]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Run unit tests
        run: npm run test:ci
        env:
          CHROME_BIN: /usr/bin/google-chrome
` + integrationTestStep + `
      - name: Generate lock file and build
        run: npm run build:ci -- --base-href /${{ github.event.repository.name }}/

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: './dist/browser'

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
`
	return os.WriteFile(filepath.Join(workflowDir, "deploy.yml"), []byte(content), 0644)
}

func GeneratePluginWorkflow(pluginDir string, pyodideVersion string) error {
	return GeneratePluginWorkflowWithVersions(pluginDir, pyodideVersion, "0.5.0")
}

func GeneratePluginWorkflowWithVersions(pluginDir string, pyodideVersion string, webrVersion string) error {
	pluginPath := filepath.Join(pluginDir, "plugin.yaml")
	definition, err := parser.ParsePlugin(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to parse plugin: %w", err)
	}

	isR := definition.Runtime.HasEnvironment("r") && !definition.Runtime.HasEnvironment("python")
	hasExample := definition.Example != nil && definition.Example.Enabled && hasExampleFileInputs(definition)

	workflowDir := filepath.Join(pluginDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return err
	}

	integrationTestStep := ""
	if hasExample {
		integrationTestStep = `
      - name: Run integration tests with example
        working-directory: spa
        run: npm run test:ci:integration
        continue-on-error: false
        env:
          CHROME_BIN: /usr/bin/google-chrome
`
	}

	var content string
	if isR {
		content = `name: Build SPA and Deploy to GitHub Pages

on:
  push:
    branches: [ main, master ]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: true

env:
  CAULDRON_VERSION: "master"
  WEBR_VERSION: "` + webrVersion + `"

jobs:
  build-spa:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout plugin
        uses: actions/checkout@v4

      - name: Checkout CauldronGO
        uses: actions/checkout@v4
        with:
          repository: noatgnu/cauldron-go
          ref: ${{ env.CAULDRON_VERSION }}
          path: cauldron-go

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Build shared-lib
        working-directory: cauldron-go/shared-lib
        run: |
          npm install
          npm run build

      - name: Build plugin-to-spa
        working-directory: cauldron-go
        run: go build -o ../plugin-to-spa ./cmd/plugin-to-spa/

      - name: Generate SPA
        run: |
          ./plugin-to-spa \
            --plugin ./plugin.yaml \
            --output ./spa \
            --skip-check \
            --no-build \
            --webr-version ${{ env.WEBR_VERSION }} \
            --spa-template ./cauldron-go/spa-template \
            --shared-lib-path /home/runner/work/${{ github.event.repository.name }}/${{ github.event.repository.name }}/cauldron-go/shared-lib/dist/cauldron-forms \
            --examples-dir ./examples:./cauldron-go/examples

      - name: Install SPA dependencies
        working-directory: spa
        run: npm install

      - name: Run unit tests
        working-directory: spa
        run: npm run test:ci
        env:
          CHROME_BIN: /usr/bin/google-chrome
` + integrationTestStep + `
      - name: Build SPA
        working-directory: spa
        run: npm run build:ci -- --base-href /${{ github.event.repository.name }}/

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: './spa/dist/browser'

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build-spa
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
`
	} else {
		content = `name: Build SPA and Deploy to GitHub Pages

on:
  push:
    branches: [ main, master ]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: true

env:
  CAULDRON_VERSION: "master"
  PYODIDE_VERSION: "` + pyodideVersion + `"

jobs:
  build-spa:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout plugin
        uses: actions/checkout@v4

      - name: Checkout CauldronGO
        uses: actions/checkout@v4
        with:
          repository: noatgnu/cauldron-go
          ref: ${{ env.CAULDRON_VERSION }}
          path: cauldron-go

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Build shared-lib
        working-directory: cauldron-go/shared-lib
        run: |
          npm install
          npm run build

      - name: Build plugin-to-spa
        working-directory: cauldron-go
        run: go build -o ../plugin-to-spa ./cmd/plugin-to-spa/

      - name: Generate SPA
        run: |
          ./plugin-to-spa \
            --plugin ./plugin.yaml \
            --output ./spa \
            --skip-check \
            --no-build \
            --pyodide-version ${{ env.PYODIDE_VERSION }} \
            --spa-template ./cauldron-go/spa-template \
            --shared-lib-path /home/runner/work/${{ github.event.repository.name }}/${{ github.event.repository.name }}/cauldron-go/shared-lib/dist/cauldron-forms \
            --examples-dir ./examples:./cauldron-go/examples

      - name: Install SPA dependencies
        working-directory: spa
        run: npm install

      - name: Run unit tests
        working-directory: spa
        run: npm run test:ci
        env:
          CHROME_BIN: /usr/bin/google-chrome
` + integrationTestStep + `
      - name: Generate lock file and build SPA
        working-directory: spa
        run: npm run build:ci -- --base-href /${{ github.event.repository.name }}/

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: './spa/dist/browser'

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build-spa
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
`
	}
	return os.WriteFile(filepath.Join(workflowDir, "deploy-spa.yml"), []byte(content), 0644)
}

func hasExampleFileInputs(definition *models.PluginDefinition) bool {
	if definition.Example == nil || definition.Example.Values == nil {
		return false
	}

	fileInputs := make(map[string]bool)
	for _, input := range definition.Inputs {
		if input.Type == "file" {
			fileInputs[input.Name] = true
		}
	}

	for key, value := range definition.Example.Values {
		if fileInputs[key] {
			if strVal, ok := value.(string); ok && strVal != "" {
				return true
			}
		}
	}
	return false
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if relPath == "node_modules" || strings.HasPrefix(relPath, "node_modules/") ||
			relPath == "dist" || strings.HasPrefix(relPath, "dist/") ||
			relPath == ".angular" || strings.HasPrefix(relPath, ".angular/") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func toJSONArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	result, _ := json.Marshal(items)
	return string(result)
}
