// Package pluginmigrations defines the declarative plugin.yaml migration format shared by the plugin-migrate CLI and the app's runtime CustomEnvVar reconciliation.
package pluginmigrations

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

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

	// Path is the file this migration was loaded from; not part of the YAML itself.
	Path string `yaml:"-"`
}

// FileNamePattern is the required migrations/NNNN_slug.yaml naming convention.
var FileNamePattern = regexp.MustCompile(`^(\d{4})_[a-z0-9_]+\.yaml$`)

// LoadMigrations reads migrations/NNNN_*.yaml files in dir, sorted by filename, requiring schemaVersion to increase by 1 with no gaps; a missing dir just means no migrations exist yet.
func LoadMigrations(dir string) ([]MigrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", dir, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !FileNamePattern.MatchString(e.Name()) {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var migrations []MigrationFile
	expected := 1
	for _, name := range names {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		var m MigrationFile
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		m.Path = path
		if m.SchemaVersion != expected {
			return nil, fmt.Errorf("%s: expected schemaVersion %d (migrations must increase by 1 with no gaps), got %d", name, expected, m.SchemaVersion)
		}
		migrations = append(migrations, m)
		expected++
	}
	return migrations, nil
}
