package services

import (
	"errors"
	"log"
	"path/filepath"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/internal/pluginmigrations"
)

// LargeMigrationOperationThreshold flags a pending migration set as "large" past this many total operations, requiring explicit confirmation before ApplyPendingEnvVarMigration will proceed.
const LargeMigrationOperationThreshold = 50

// ErrLargeMigrationRequiresConfirmation is returned by ApplyPendingEnvVarMigration when the pending set is large and confirmedLarge was not set.
var ErrLargeMigrationRequiresConfirmation = errors.New("pending migration is unusually large and requires explicit confirmation")

// EnvVarKeyChange is one saved CustomEnvVar key that a plugin update's migrations renamed or removed.
type EnvVarKeyChange struct {
	From    string `json:"from"`
	To      string `json:"to"` // empty when Removed is true
	Removed bool   `json:"removed"`
}

// PendingEnvVarMigration summarizes what ApplyPendingEnvVarMigration would do, plus a Large flag when the pending migration set is unusually big and warrants a second confirmation.
type PendingEnvVarMigration struct {
	Changes         []EnvVarKeyChange `json:"changes"`
	TotalOperations int               `json:"totalOperations"`
	Large           bool              `json:"large"`
}

// PluginMigrationService reconciles saved CustomEnvVar values against a plugin's declared env-var renames/removals, only via explicit user action, never automatically or by running plugin code.
type PluginMigrationService struct {
	db *DatabaseService
}

func NewPluginMigrationService(db *DatabaseService) *PluginMigrationService {
	return &PluginMigrationService{db: db}
}

// pendingEnvVarChanges resolves, for each saved CustomEnvVar key, whether pending migrations renamed or removed it, following multi-step rename chains through correctly.
func (s *PluginMigrationService) pendingEnvVarChanges(registryID uint, pluginFolderPath string, lastApplied, current int) (*PendingEnvVarMigration, error) {
	result := &PendingEnvVarMigration{}
	if current <= lastApplied {
		return result, nil
	}

	migrations, err := pluginmigrations.LoadMigrations(filepath.Join(pluginFolderPath, "migrations"))
	if err != nil {
		return nil, err
	}
	if current > len(migrations) {
		current = len(migrations)
	}
	if current <= lastApplied {
		return result, nil
	}
	pending := migrations[lastApplied:current]

	for _, m := range pending {
		result.TotalOperations += len(m.Operations)
	}
	result.Large = result.TotalOperations > LargeMigrationOperationThreshold

	existing, err := s.db.GetCustomEnvVars(registryID)
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		return result, nil
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

	for original, currentKey := range effective {
		if removed[original] {
			result.Changes = append(result.Changes, EnvVarKeyChange{From: original, Removed: true})
			continue
		}
		if currentKey != original {
			result.Changes = append(result.Changes, EnvVarKeyChange{From: original, To: currentKey})
		}
	}
	return result, nil
}

// DetectPendingEnvVarMigration is a read-only preview of what ApplyPendingEnvVarMigration would do.
func (s *PluginMigrationService) DetectPendingEnvVarMigration(registry *models.PluginRegistry, currentSchemaVersion int) (*PendingEnvVarMigration, error) {
	return s.pendingEnvVarChanges(registry.ID, registry.FolderPath, registry.LastAppliedSchemaVersion, currentSchemaVersion)
}

// ApplyPendingEnvVarMigration renames/removes saved CustomEnvVar entries, then records the new schemaVersion so this isn't offered again. A large pending set (see LargeMigrationOperationThreshold) is rejected unless confirmedLarge is true.
func (s *PluginMigrationService) ApplyPendingEnvVarMigration(registry *models.PluginRegistry, currentSchemaVersion int, confirmedLarge bool) error {
	pending, err := s.pendingEnvVarChanges(registry.ID, registry.FolderPath, registry.LastAppliedSchemaVersion, currentSchemaVersion)
	if err != nil {
		return err
	}
	if pending.Large && !confirmedLarge {
		return ErrLargeMigrationRequiresConfirmation
	}

	for _, c := range pending.Changes {
		if c.Removed {
			if err := s.db.DeleteCustomEnvVarByKey(registry.ID, c.From); err != nil {
				return err
			}
			log.Printf("[PluginMigration] Removed saved env var %q for plugin %q (registry ID %d)", c.From, registry.PluginID, registry.ID)
			continue
		}
		renamed, err := s.db.RenameCustomEnvVarKey(registry.ID, c.From, c.To)
		if err != nil {
			return err
		}
		if renamed {
			log.Printf("[PluginMigration] Renamed saved env var %q -> %q for plugin %q (registry ID %d)", c.From, c.To, registry.PluginID, registry.ID)
		} else {
			log.Printf("[PluginMigration] Dropped saved env var %q for plugin %q (registry ID %d): destination key %q already had a saved value", c.From, registry.PluginID, registry.ID, c.To)
		}
	}

	if err := s.db.GetDB().Model(&models.PluginRegistry{}).Where("id = ?", registry.ID).Update("last_applied_schema_version", currentSchemaVersion).Error; err != nil {
		return err
	}
	registry.LastAppliedSchemaVersion = currentSchemaVersion
	log.Printf("[PluginMigration] Plugin %q (registry ID %d) reconciled to schemaVersion %d (%d change(s), %d total operation(s))", registry.PluginID, registry.ID, currentSchemaVersion, len(pending.Changes), pending.TotalOperations)
	return nil
}
