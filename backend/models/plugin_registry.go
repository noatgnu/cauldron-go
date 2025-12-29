package models

import (
	"time"

	"gorm.io/gorm"
)

type PluginRegistry struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	PluginID       string         `gorm:"not null;index" json:"pluginId"`
	Name           string         `gorm:"not null" json:"name"`
	Version        string         `json:"version"`
	Repository     string         `json:"repository"`
	CommitHash     string         `json:"commitHash"`
	FolderPath     string         `gorm:"uniqueIndex;not null" json:"folderPath"`
	InstallSource  string         `gorm:"default:'builtin'" json:"installSource"`
	RegistrySource *string        `json:"registrySource,omitempty"`
	UpdatePolicy   string         `gorm:"default:'auto'" json:"updatePolicy"`
	PinnedVersion  *string        `json:"pinnedVersion,omitempty"`
	InstalledAt    time.Time      `json:"installedAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (PluginRegistry) TableName() string {
	return "plugin_registry"
}
