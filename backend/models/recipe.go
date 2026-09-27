package models

import "time"

type Recipe struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Label       string    `gorm:"not null" json:"label"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"not null" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"not null" json:"updatedAt"`
}

type RecipeStage struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	RecipeID      string  `gorm:"index;not null" json:"recipeId"`
	StageIndex    int     `gorm:"not null" json:"stageIndex"`
	PluginID      string  `gorm:"not null" json:"pluginId"`
	PluginVersion string  `json:"pluginVersion,omitempty"`
	Params        JSONMap `gorm:"type:text" json:"params"`
	Bindings      JSONMap `gorm:"type:text" json:"bindings"`

	Repository   string  `json:"repository,omitempty"`
	CommitHash   string  `json:"commitHash,omitempty"`
	Requirements JSONMap `gorm:"type:text" json:"requirements,omitempty"`
}

type RecipeStageBinding struct {
	Stage  int    `json:"stage"`
	Output string `json:"output"`
}
