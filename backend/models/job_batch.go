package models

import "time"

type JobBatch struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	Label         string    `gorm:"not null" json:"label"`
	PluginID      string    `json:"pluginId"`
	PluginVersion string    `json:"pluginVersion,omitempty"`
	ExpectedCount int       `gorm:"not null" json:"expectedCount"`
	CreatedAt     time.Time `gorm:"not null" json:"createdAt"`
}
