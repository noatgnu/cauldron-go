package services

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/models"
)

const pluginsDirSettingKey = "pluginsDirLastKnown"

// MigrateThirdPartyPlugins carries over plugin folders from a previous version's plugins directory that are missing from the current one, since each portable install/upgrade gets its own folder.
func MigrateThirdPartyPlugins(db *DatabaseService, currentPluginsDir string) {
	previousDir, err := db.GetSetting(pluginsDirSettingKey)
	if err != nil {
		log.Printf("[PluginDirMigration] Failed to read last known plugins dir: %v", err)
	}

	if previousDir != "" && previousDir != currentPluginsDir {
		copyMissingPluginFolders(previousDir, currentPluginsDir)
	}

	if err := db.SaveSetting(pluginsDirSettingKey, currentPluginsDir); err != nil {
		log.Printf("[PluginDirMigration] Failed to record current plugins dir: %v", err)
	}
}

// copyMissingPluginFolders copies every top-level folder in previousDir not already present in currentDir, so freshly-bundled built-ins are never overwritten.
func copyMissingPluginFolders(previousDir, currentDir string) {
	entries, err := os.ReadDir(previousDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		dest := filepath.Join(currentDir, entry.Name())
		if _, err := os.Stat(dest); err == nil {
			continue
		}

		src := filepath.Join(previousDir, entry.Name())
		if err := copyDir(src, dest); err != nil {
			log.Printf("[PluginDirMigration] Failed to carry over plugin %q: %v", entry.Name(), err)
			os.RemoveAll(dest)
			continue
		}
		log.Printf("[PluginDirMigration] Carried over plugin %q from previous installation", entry.Name())
	}
}

// ReconcileOrphanedPluginData reattaches CustomEnvVar rows from a stale PluginRegistry row (its folder no longer exists, e.g. after a migration) to the current row for the same plugin string ID, then removes the stale row. Call after LoadPlugins so migrated plugins already have fresh rows.
func ReconcileOrphanedPluginData(db *DatabaseService) {
	entries, err := db.GetAllPluginRegistryEntries()
	if err != nil {
		log.Printf("[PluginDirMigration] Failed to list plugin registry entries: %v", err)
		return
	}

	currentIDByPluginID := make(map[string]uint)
	var stale []models.PluginRegistry
	for _, entry := range entries {
		if _, err := os.Stat(entry.FolderPath); err == nil {
			currentIDByPluginID[entry.PluginID] = entry.ID
		} else {
			stale = append(stale, entry)
		}
	}

	for _, s := range stale {
		newID, ok := currentIDByPluginID[s.PluginID]
		if !ok || newID == s.ID {
			continue
		}

		if err := db.GetDB().Model(&CustomEnvVar{}).Where("plugin_id = ?", s.ID).Update("plugin_id", newID).Error; err != nil {
			log.Printf("[PluginDirMigration] Failed to reattach env vars for %q: %v", s.PluginID, err)
			continue
		}
		if err := db.GetDB().Delete(&models.PluginRegistry{}, s.ID).Error; err != nil {
			log.Printf("[PluginDirMigration] Failed to remove stale registry row for %q: %v", s.PluginID, err)
			continue
		}
		log.Printf("[PluginDirMigration] Reattached env vars and cleaned up stale registry row for %q", s.PluginID)
	}
}
