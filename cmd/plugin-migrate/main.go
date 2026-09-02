package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

// RenameOp renames a named element (input, env var, or output) in place.
type RenameOp struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// NameOp identifies a named element to remove.
type NameOp struct {
	Name string `yaml:"name"`
}

// AddInputOp adds a new input or env var definition, optionally positioned after an existing one.
type AddInputOp struct {
	models.PluginInputV2 `yaml:",inline"`
	InsertAfter          string `yaml:"insertAfter,omitempty"`
}

// PackageOp adds or removes a single package requirement.
type PackageOp struct {
	Name string `yaml:"name"`
}

// RequirementOp sets the Python and/or R version constraint.
type RequirementOp struct {
	Python string `yaml:"python,omitempty"`
	R      string `yaml:"r,omitempty"`
}

// Operation is a single declarative change; exactly one field should be set.
type Operation struct {
	RenameInput    *RenameOp      `yaml:"renameInput,omitempty"`
	RemoveInput    *NameOp        `yaml:"removeInput,omitempty"`
	AddInput       *AddInputOp    `yaml:"addInput,omitempty"`
	RenameEnvVar   *RenameOp      `yaml:"renameEnvVar,omitempty"`
	RemoveEnvVar   *NameOp        `yaml:"removeEnvVar,omitempty"`
	AddEnvVar      *AddInputOp    `yaml:"addEnvVar,omitempty"`
	RenameOutput   *RenameOp      `yaml:"renameOutput,omitempty"`
	RemoveOutput   *NameOp        `yaml:"removeOutput,omitempty"`
	SetRequirement *RequirementOp `yaml:"setRequirement,omitempty"`
	AddPackage     *PackageOp     `yaml:"addPackage,omitempty"`
	RemovePackage  *PackageOp     `yaml:"removePackage,omitempty"`
}

// MigrationFile describes one schema version step for a plugin.
type MigrationFile struct {
	SchemaVersion int         `yaml:"schemaVersion"`
	Description   string      `yaml:"description,omitempty"`
	Operations    []Operation `yaml:"operations"`

	path string
}

var migrationFileNamePattern = regexp.MustCompile(`^(\d{4})_[a-z0-9_]+\.yaml$`)

func loadPluginDefinition(path string) (*models.PluginDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var def models.PluginDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &def, nil
}

func savePluginDefinition(path string, def *models.PluginDefinition) error {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(def); err != nil {
		return fmt.Errorf("failed to encode %s: %w", path, err)
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(buf.String()), 0644)
}

// loadMigrations reads every migrations/NNNN_*.yaml file, sorted by filename, validating that
// schemaVersion strictly increases by 1 per file so the chain has no gaps or reordering.
func loadMigrations(migrationsDir string) ([]MigrationFile, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", migrationsDir, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !migrationFileNamePattern.MatchString(e.Name()) {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var migrations []MigrationFile
	expected := 1
	for _, name := range names {
		path := filepath.Join(migrationsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		var m MigrationFile
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		m.path = path
		if m.SchemaVersion != expected {
			return nil, fmt.Errorf("%s: expected schemaVersion %d (migrations must increase by 1 with no gaps), got %d", name, expected, m.SchemaVersion)
		}
		migrations = append(migrations, m)
		expected++
	}
	return migrations, nil
}

func applyOperation(def *models.PluginDefinition, op Operation) error {
	switch {
	case op.RenameInput != nil:
		return renameInput(def, op.RenameInput.From, op.RenameInput.To)
	case op.RemoveInput != nil:
		return removeInput(def, op.RemoveInput.Name)
	case op.AddInput != nil:
		return addInput(def, op.AddInput)
	case op.RenameEnvVar != nil:
		return renameEnvVar(def, op.RenameEnvVar.From, op.RenameEnvVar.To)
	case op.RemoveEnvVar != nil:
		return removeEnvVar(def, op.RemoveEnvVar.Name)
	case op.AddEnvVar != nil:
		def.Execution.EnvVariables = append(def.Execution.EnvVariables, op.AddEnvVar.PluginInputV2)
		return nil
	case op.RenameOutput != nil:
		return renameOutput(def, op.RenameOutput.From, op.RenameOutput.To)
	case op.RemoveOutput != nil:
		return removeOutput(def, op.RemoveOutput.Name)
	case op.SetRequirement != nil:
		if op.SetRequirement.Python != "" {
			def.Execution.Requirements.Python = op.SetRequirement.Python
		}
		if op.SetRequirement.R != "" {
			def.Execution.Requirements.R = op.SetRequirement.R
		}
		return nil
	case op.AddPackage != nil:
		addPackage(def, op.AddPackage.Name)
		return nil
	case op.RemovePackage != nil:
		removePackage(def, op.RemovePackage.Name)
		return nil
	default:
		return fmt.Errorf("migration operation has no recognized action set")
	}
}

func renameInput(def *models.PluginDefinition, from, to string) error {
	found := false
	for i := range def.Inputs {
		if def.Inputs[i].Name == from {
			def.Inputs[i].Name = to
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("renameInput: no input named %q", from)
	}
	if v, ok := def.Execution.ArgsMapping[from]; ok {
		delete(def.Execution.ArgsMapping, from)
		def.Execution.ArgsMapping[to] = v
	}
	if def.Example != nil {
		if v, ok := def.Example.Values[from]; ok {
			delete(def.Example.Values, from)
			def.Example.Values[to] = v
		}
	}
	return nil
}

func removeInput(def *models.PluginDefinition, name string) error {
	idx := -1
	for i := range def.Inputs {
		if def.Inputs[i].Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("removeInput: no input named %q", name)
	}
	def.Inputs = append(def.Inputs[:idx], def.Inputs[idx+1:]...)
	delete(def.Execution.ArgsMapping, name)
	if def.Example != nil {
		delete(def.Example.Values, name)
	}
	return nil
}

func addInput(def *models.PluginDefinition, op *AddInputOp) error {
	for i := range def.Inputs {
		if def.Inputs[i].Name == op.Name {
			return fmt.Errorf("addInput: input named %q already exists", op.Name)
		}
	}
	if op.InsertAfter == "" {
		def.Inputs = append(def.Inputs, op.PluginInputV2)
		return nil
	}
	for i := range def.Inputs {
		if def.Inputs[i].Name == op.InsertAfter {
			def.Inputs = append(def.Inputs[:i+1], append([]models.PluginInputV2{op.PluginInputV2}, def.Inputs[i+1:]...)...)
			return nil
		}
	}
	return fmt.Errorf("addInput: insertAfter target %q not found", op.InsertAfter)
}

func renameEnvVar(def *models.PluginDefinition, from, to string) error {
	for i := range def.Execution.EnvVariables {
		if def.Execution.EnvVariables[i].Name == from {
			def.Execution.EnvVariables[i].Name = to
			return nil
		}
	}
	return fmt.Errorf("renameEnvVar: no env var named %q", from)
}

func removeEnvVar(def *models.PluginDefinition, name string) error {
	for i := range def.Execution.EnvVariables {
		if def.Execution.EnvVariables[i].Name == name {
			def.Execution.EnvVariables = append(def.Execution.EnvVariables[:i], def.Execution.EnvVariables[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("removeEnvVar: no env var named %q", name)
}

func renameOutput(def *models.PluginDefinition, from, to string) error {
	for i := range def.Outputs {
		if def.Outputs[i].Name == from {
			def.Outputs[i].Name = to
			return nil
		}
	}
	return fmt.Errorf("renameOutput: no output named %q", from)
}

func removeOutput(def *models.PluginDefinition, name string) error {
	for i := range def.Outputs {
		if def.Outputs[i].Name == name {
			def.Outputs = append(def.Outputs[:i], def.Outputs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("removeOutput: no output named %q", name)
}

// packageBaseName strips a pip/CRAN-style version specifier (e.g. "numpy>=1.24.0" -> "numpy").
func packageBaseName(pkg string) string {
	for _, sep := range []string{">=", "<=", "==", "~=", ">", "<", "="} {
		if idx := strings.Index(pkg, sep); idx != -1 {
			return pkg[:idx]
		}
	}
	return pkg
}

// addPackage upserts by base name: an existing entry has its version specifier replaced,
// so "addPackage: numpy>=1.26.0" also serves as the way to bump an already-required package.
func addPackage(def *models.PluginDefinition, pkg string) {
	base := packageBaseName(pkg)
	for i, existing := range def.Execution.Requirements.Packages {
		if packageBaseName(existing) == base {
			def.Execution.Requirements.Packages[i] = pkg
			return
		}
	}
	def.Execution.Requirements.Packages = append(def.Execution.Requirements.Packages, pkg)
}

func removePackage(def *models.PluginDefinition, pkg string) {
	base := packageBaseName(pkg)
	var kept []string
	for _, existing := range def.Execution.Requirements.Packages {
		if packageBaseName(existing) != base {
			kept = append(kept, existing)
		}
	}
	def.Execution.Requirements.Packages = kept
}

func nextMigrationNumber(migrationsDir string) (int, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, err
	}
	max := 0
	for _, e := range entries {
		if e.IsDir() || !migrationFileNamePattern.MatchString(e.Name()) {
			continue
		}
		n, _ := strconv.Atoi(e.Name()[:4])
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

func cmdNew(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: plugin-migrate new <plugin-dir> <slug>")
	}
	pluginDir, slug := args[0], args[1]
	migrationsDir := filepath.Join(pluginDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return err
	}

	n, err := nextMigrationNumber(migrationsDir)
	if err != nil {
		return err
	}

	safeSlug := regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(strings.ToLower(slug), "_")
	fileName := fmt.Sprintf("%04d_%s.yaml", n, safeSlug)
	path := filepath.Join(migrationsDir, fileName)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}

	template := fmt.Sprintf(`schemaVersion: %d
description: ""
operations:
  # - renameInput: { from: old_name, to: new_name }
  # - addInput: { name: new_field, label: "New Field", type: text, default: "", insertAfter: some_existing_input }
  # - removeInput: { name: obsolete_field }
  # - renameEnvVar: { from: OLD_KEY, to: NEW_KEY }
  # - addPackage: { name: "numpy>=1.24.0" }
  # - setRequirement: { python: ">=3.11" }
`, n)
	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		return err
	}
	fmt.Printf("Created %s\n", path)
	return nil
}

func cmdApply(args []string, dryRun bool) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: plugin-migrate apply <plugin-dir>")
	}
	pluginDir := args[0]
	pluginYamlPath := filepath.Join(pluginDir, "plugin.yaml")
	migrationsDir := filepath.Join(pluginDir, "migrations")

	def, err := loadPluginDefinition(pluginYamlPath)
	if err != nil {
		return err
	}

	migrations, err := loadMigrations(migrationsDir)
	if err != nil {
		return err
	}

	if def.Plugin.SchemaVersion > len(migrations) {
		return fmt.Errorf("%s records schemaVersion %d but only %d migration file(s) exist in %s", def.Plugin.ID, def.Plugin.SchemaVersion, len(migrations), migrationsDir)
	}

	pending := migrations[def.Plugin.SchemaVersion:]
	if len(pending) == 0 {
		fmt.Printf("%s is already at schemaVersion %d, nothing to apply\n", def.Plugin.ID, def.Plugin.SchemaVersion)
		return nil
	}

	for _, m := range pending {
		for i, op := range m.Operations {
			if err := applyOperation(def, op); err != nil {
				return fmt.Errorf("%s operation %d: %w", filepath.Base(m.path), i+1, err)
			}
		}
		def.Plugin.SchemaVersion = m.SchemaVersion
		verb := "Applied"
		if dryRun {
			verb = "Would apply"
		}
		desc := m.Description
		if desc != "" {
			desc = ": " + desc
		}
		fmt.Printf("%s %s (schemaVersion %d)%s\n", verb, filepath.Base(m.path), m.SchemaVersion, desc)
	}

	if dryRun {
		fmt.Printf("\nDry run only -- %s was not modified. Re-run without -dry-run to write changes.\n", pluginYamlPath)
		return nil
	}

	if err := savePluginDefinition(pluginYamlPath, def); err != nil {
		return err
	}
	fmt.Printf("\n%s updated to schemaVersion %d\n", pluginYamlPath, def.Plugin.SchemaVersion)
	return nil
}

func printUsage() {
	fmt.Println("Usage: plugin-migrate <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  new <plugin-dir> <slug>    Scaffold a new migration file in <plugin-dir>/migrations/")
	fmt.Println("  apply <plugin-dir>         Apply pending migrations to <plugin-dir>/plugin.yaml")
	fmt.Println("  diff <plugin-dir>          Preview pending migrations without writing changes")
	fmt.Println()
	fmt.Println("Migrations run on the plugin developer's own machine and produce a finished")
	fmt.Println("plugin.yaml to commit and publish -- end users never execute migration logic.")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "new":
		err = cmdNew(os.Args[2:])
	case "apply":
		err = cmdApply(os.Args[2:], false)
	case "diff":
		err = cmdApply(os.Args[2:], true)
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
