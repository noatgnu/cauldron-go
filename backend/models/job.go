package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusInProgress JobStatus = "in_progress"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type StringArray []string

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		*s = []string{}
		return nil
	}

	return json.Unmarshal(bytes, s)
}

func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

type JSONMap map[string]interface{}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		*j = make(map[string]interface{})
		return nil
	}

	return json.Unmarshal(bytes, j)
}

func (j JSONMap) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return json.Marshal(j)
}

type Job struct {
	ID             string      `gorm:"primaryKey" json:"id"`
	Type           string      `gorm:"not null" json:"type"`
	Name           string      `gorm:"not null" json:"name"`
	Status         JobStatus   `gorm:"not null;default:pending" json:"status"`
	Progress       float64     `gorm:"default:0" json:"progress"`
	Command        string      `gorm:"not null" json:"command"`
	Args           StringArray `gorm:"type:text" json:"args"`
	Parameters     JSONMap     `gorm:"type:text" json:"parameters"`
	PythonEnvPath  string      `json:"pythonEnvPath,omitempty"`
	PythonEnvType  string      `json:"pythonEnvType,omitempty"`
	REnvPath       string      `json:"rEnvPath,omitempty"`
	REnvType       string      `json:"rEnvType,omitempty"`
	OutputPath     string      `json:"outputPath"`
	TerminalOutput StringArray `gorm:"type:text" json:"terminalOutput"`
	CreatedAt      time.Time   `gorm:"not null" json:"createdAt"`
	StartedAt      *time.Time  `json:"startedAt,omitempty"`
	CompletedAt    *time.Time  `json:"completedAt,omitempty"`
	Error          string      `json:"error,omitempty"`
}

type JobRequest struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	InputFiles []string               `json:"inputFiles"`
	Parameters map[string]interface{} `json:"parameters"`
}
