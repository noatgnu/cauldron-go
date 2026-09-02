package services

import (
	"path/filepath"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/internal/pluginmigrations"
)

// EnvVarKeyChange is one saved CustomEnvVar key that a plugin update's migrations renamed or removed.
type EnvVarKeyChange struct {
	From    string `json:"from"`
	To      string `json:"to"` // empty when Removed is true
	Removed bool   `json:"removed"`
}

// PluginMigrationService reconciles saved CustomEnvVar values against a plugin's declared env-var renames/removals, only via explicit user action, never automatically or by running plugin code.
type PluginMigrationService struct {
	db *DatabaseService
}

func NewPluginMigrationService(db *DatabaseService) *PluginMigrationService {
	return &PluginMigrationService{db: db}
}

// pendingEnvVarChanges resolves, for each saved CustomEnvVar key, whether pending migrations renamed or removed it, following multi-step rename chains through correctly.
func (s *PluginMigrationService) pendingEnvVarChanges(registryID uint, pluginFolderPath string, lastApplied, current int) ([]EnvVarKeyChange, error) {
	if current <= lastApplied {
		return nil, nil
	}

	migrations, err := pluginmigrations.LoadMigrations(filepath.Join(pluginFolderPath, "migrations"))
	if err != nil {
		return nil, err
	}
	if current > len(migrations) {
		current = len(migrations)
	}
	if current <= lastApplied {
		return nil, nil
	}
	pending := migrations[lastApplied:current]

	existing, err := s.db.GetCustomEnvVars(registryID)
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		return nil, nil
	}

	effective := make(map[string]string, len(existing))
	removed := make(map[string]bool, len(existing))
	for _, e := range existing {
		effective[e.Key] = e.Key
	}

	for _, m := range pending {
		for _, op := range m.Operations {
			switch {
			case op.RenameEnvVar != nil:
				for original, currentKey := range effective {
					if currentKey == op.RenameEnvVar.From {
						effective[original] = op.RenameEnvVar.To
					}
				}
			case op.RemoveEnvVar != nil:
				for original, currentKey := range effective {
					if currentKey == op.RemoveEnvVar.Name {
						removed[original] = true
					}
				}
			}
		}
	}

	var changes []EnvVarKeyChange
	for original, currentKey := range effective {
		if removed[original] {
			changes = append(changes, EnvVarKeyChange{From: original, Removed: true})
			continue
		}
		if currentKey != original {
			changes = append(changes, EnvVarKeyChange{From: original, To: currentKey})
		}
	}
	return changes, nil
}

// DetectPendingEnvVarMigration is a read-only preview of what ApplyPendingEnvVarMigration would do.
func (s *PluginMigrationService) DetectPendingEnvVarMigration(registry *models.PluginRegistry, currentSchemaVersion int) ([]EnvVarKeyChange, error) {
	return s.pendingEnvVarChanges(registry.ID, registry.FolderPath, registry.LastAppliedSchemaVersion, currentSchemaVersion)
}

// ApplyPendingEnvVarMigration renames/removes saved CustomEnvVar entries, then records the new schemaVersion so this isn't offered again.
func (s *PluginMigrationService) ApplyPendingEnvVarMigration(registry *models.PluginRegistry, currentSchemaVersion int) error {
	changes, err := s.pendingEnvVarChanges(registry.ID, registry.FolderPath, registry.LastAppliedSchemaVersion, currentSchemaVersion)
	if err != nil {
		return err
	}

	for _, c := range changes {
		if c.Removed {
			if err := s.db.DeleteCustomEnvVarByKey(registry.ID, c.From); err != nil {
				return err
			}
			continue
		}
		if err := s.db.RenameCustomEnvVarKey(registry.ID, c.From, c.To); err != nil {
			return err
		}
	}

	if err := s.db.GetDB().Model(&models.PluginRegistry{}).Where("id = ?", registry.ID).Update("last_applied_schema_version", currentSchemaVersion).Error; err != nil {
		return err
	}
	registry.LastAppliedSchemaVersion = currentSchemaVersion
	return nil
}
