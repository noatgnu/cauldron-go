package models

import "time"

type JobChain struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	RecipeID  string    `json:"recipeId,omitempty"`
	Label     string    `gorm:"not null" json:"label"`
	CreatedAt time.Time `gorm:"not null" json:"createdAt"`
}

type JobChainStage struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	ChainID       string  `gorm:"index;not null" json:"chainId"`
	StageIndex    int     `gorm:"not null" json:"stageIndex"`
	PluginID      uint    `gorm:"not null" json:"pluginId"`
	PluginVersion string  `json:"pluginVersion,omitempty"`
	Params        JSONMap `gorm:"type:text" json:"params"`
	Bindings      JSONMap `gorm:"type:text" json:"bindings"`
	JobID         string  `json:"jobId,omitempty"`
}
