package services

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

const BackupFormatVersion = 1

// PluginBackupEntry captures one PluginRegistry row in a form that survives reinstall (row IDs are not stable across restores).
type PluginBackupEntry struct {
	PluginID      string  `json:"pluginId"`
	Name          string  `json:"name"`
	Repository    string  `json:"repository"`
	InstallSource string  `json:"installSource"`
	UpdatePolicy  string  `json:"updatePolicy"`
	PinnedVersion *string `json:"pinnedVersion,omitempty"`
	Enabled       bool    `json:"enabled"`
}

// CustomEnvVarBackupEntry keys on the plugin's stable string ID rather than its PluginRegistry row ID, which is reassigned on reinstall. PluginID is empty for global vars.
type CustomEnvVarBackupEntry struct {
	PluginID string `json:"pluginId"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

// BackupData is the on-disk backup file format: settings + plugin registry, optionally including per-plugin/global secret env vars.
type BackupData struct {
	Version       int                       `json:"version"`
	CreatedAt     time.Time                 `json:"createdAt"`
	Settings      map[string]string         `json:"settings"`
	Plugins       []PluginBackupEntry       `json:"plugins"`
	CustomEnvVars []CustomEnvVarBackupEntry `json:"customEnvVars,omitempty"`
}

// RestoreResult summarizes what a restore actually did, for CLI/GUI reporting.
type RestoreResult struct {
	SettingsRestored int               `json:"settingsRestored"`
	PluginsInstalled []string          `json:"pluginsInstalled"`
	PluginsSkipped   []string          `json:"pluginsSkipped"`
	PluginsFailed    map[string]string `json:"pluginsFailed"`
	EnvVarsRestored  int               `json:"envVarsRestored"`
}

// BackupSummary is the safe-to-display subset of a BackupData -- counts only, never secret values -- for GUI confirmation before a restore.
type BackupSummary struct {
	CreatedAt       time.Time `json:"createdAt"`
	SettingsCount   int       `json:"settingsCount"`
	PluginsCount    int       `json:"pluginsCount"`
	EnvVarsCount    int       `json:"envVarsCount"`
	IncludesSecrets bool      `json:"includesSecrets"`
}

func (d *BackupData) Summary() BackupSummary {
	return BackupSummary{
		CreatedAt:       d.CreatedAt,
		SettingsCount:   len(d.Settings),
		PluginsCount:    len(d.Plugins),
		EnvVarsCount:    len(d.CustomEnvVars),
		IncludesSecrets: len(d.CustomEnvVars) > 0,
	}
}

type BackupService struct {
	db *DatabaseService
}

func NewBackupService(db *DatabaseService) *BackupService {
	return &BackupService{db: db}
}

// CreateBackup gathers settings and installed-plugin metadata; CustomEnvVars (which may hold secrets such as API keys) are only included when includeSecrets is true.
func (b *BackupService) CreateBackup(includeSecrets bool) (*BackupData, error) {
	settings, err := b.db.GetAllSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	registryEntries, err := b.db.GetAllPluginRegistryEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin registry: %w", err)
	}

	plugins := make([]PluginBackupEntry, 0, len(registryEntries))
	pluginIDByRegistryID := make(map[uint]string, len(registryEntries))
	for _, entry := range registryEntries {
		pluginIDByRegistryID[entry.ID] = entry.PluginID
		plugins = append(plugins, PluginBackupEntry{
			PluginID:      entry.PluginID,
			Name:          entry.Name,
			Repository:    entry.Repository,
			InstallSource: entry.InstallSource,
			UpdatePolicy:  entry.UpdatePolicy,
			PinnedVersion: entry.PinnedVersion,
			Enabled:       entry.Enabled,
		})
	}

	data := &BackupData{
		Version:   BackupFormatVersion,
		CreatedAt: time.Now(),
		Settings:  settings,
		Plugins:   plugins,
	}

	if includeSecrets {
		envVars, err := b.allCustomEnvVars(pluginIDByRegistryID)
		if err != nil {
			return nil, fmt.Errorf("failed to read custom env vars: %w", err)
		}
		data.CustomEnvVars = envVars
	}

	return data, nil
}

func (b *BackupService) allCustomEnvVars(pluginIDByRegistryID map[uint]string) ([]CustomEnvVarBackupEntry, error) {
	var raw []CustomEnvVar
	if err := b.db.GetDB().Find(&raw).Error; err != nil {
		return nil, err
	}

	entries := make([]CustomEnvVarBackupEntry, 0, len(raw))
	for _, v := range raw {
		pluginID := ""
		if v.PluginID != 0 {
			id, ok := pluginIDByRegistryID[v.PluginID]
			if !ok {
				continue
			}
			pluginID = id
		}
		entries = append(entries, CustomEnvVarBackupEntry{PluginID: pluginID, Key: v.Key, Value: v.Value})
	}
	return entries, nil
}

// WriteBackupFile writes data as indented JSON with 0600 permissions since it may contain secrets.
func WriteBackupFile(path string, data *BackupData) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func ReadBackupFile(path string) (*BackupData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var data BackupData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to parse backup file: %w", err)
	}
	if data.Version != BackupFormatVersion {
		return nil, fmt.Errorf("unsupported backup format version %d (expected %d)", data.Version, BackupFormatVersion)
	}
	return &data, nil
}

// RestoreBackup applies settings directly, reinstalls remote-sourced plugins that are missing (builtin plugins are skipped, they load automatically), and restores enabled state + custom env vars for plugins present after restore.
func (b *BackupService) RestoreBackup(data *BackupData, installer *PluginInstaller, progress func(string)) (*RestoreResult, error) {
	result := &RestoreResult{PluginsFailed: make(map[string]string)}

	for key, value := range data.Settings {
		if err := b.db.SaveSetting(key, value); err != nil {
			return result, fmt.Errorf("failed to restore setting %q: %w", key, err)
		}
		result.SettingsRestored++
	}

	for _, p := range data.Plugins {
		existing, err := b.db.GetPluginRegistryByPluginID(p.PluginID)
		if err != nil {
			return result, fmt.Errorf("failed to look up plugin %q: %w", p.PluginID, err)
		}

		if existing == nil {
			if p.InstallSource != "remote" || p.Repository == "" {
				result.PluginsSkipped = append(result.PluginsSkipped, p.PluginID)
				continue
			}
			if progress != nil {
				progress(fmt.Sprintf("Installing plugin %s (%s)...", p.PluginID, p.Repository))
			}
			if _, err := installer.InstallPlugin(p.Repository, "", nil, progress); err != nil {
				result.PluginsFailed[p.PluginID] = err.Error()
				continue
			}
			existing, err = b.db.GetPluginRegistryByPluginID(p.PluginID)
			if err != nil || existing == nil {
				result.PluginsFailed[p.PluginID] = "installed but could not be re-read from the registry"
				continue
			}
			result.PluginsInstalled = append(result.PluginsInstalled, p.PluginID)
		}

		if existing.Enabled != p.Enabled {
			if err := b.db.GetDB().Model(&models.PluginRegistry{}).Where("id = ?", existing.ID).Update("enabled", p.Enabled).Error; err != nil {
				return result, fmt.Errorf("failed to restore enabled state for %q: %w", p.PluginID, err)
			}
		}
	}

	for _, v := range data.CustomEnvVars {
		var registryID uint
		if v.PluginID != "" {
			entry, err := b.db.GetPluginRegistryByPluginID(v.PluginID)
			if err != nil {
				return result, fmt.Errorf("failed to look up plugin %q for env var %q: %w", v.PluginID, v.Key, err)
			}
			if entry == nil {
				// Plugin wasn't restored (skipped or failed above) -- its env vars have nothing to attach to.
				continue
			}
			registryID = entry.ID
		}
		if err := b.db.SaveCustomEnvVar(CustomEnvVar{PluginID: registryID, Key: v.Key, Value: v.Value}); err != nil {
			return result, fmt.Errorf("failed to restore env var %q: %w", v.Key, err)
		}
		result.EnvVarsRestored++
	}

	return result, nil
}
