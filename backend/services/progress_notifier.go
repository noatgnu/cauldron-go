package services

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ProgressType string

const (
	ProgressTypeDownload ProgressType = "download"
	ProgressTypeInstall  ProgressType = "install"
	ProgressTypeScript   ProgressType = "script"
	ProgressTypeExtract  ProgressType = "extract"
	ProgressTypeAnalysis ProgressType = "analysis"
	ProgressTypeGeneric  ProgressType = "generic"
)

type ProgressNotification struct {
	Type       ProgressType           `json:"type"`
	ID         string                 `json:"id"`
	Message    string                 `json:"message"`
	Percentage float64                `json:"percentage"`
	Status     string                 `json:"status"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

type ProgressNotifier struct {
	ctx context.Context
}

func NewProgressNotifier(ctx context.Context) *ProgressNotifier {
	return &ProgressNotifier{
		ctx: ctx,
	}
}

func (p *ProgressNotifier) Emit(notification ProgressNotification) {
	if p.ctx.Value("wails-test") != nil {
		return
	}
	runtime.EventsEmit(p.ctx, "progress", notification)
}

func (p *ProgressNotifier) EmitProgress(progressType ProgressType, id string, message string, percentage float64) {
	p.Emit(ProgressNotification{
		Type:       progressType,
		ID:         id,
		Message:    message,
		Percentage: percentage,
		Status:     "in_progress",
	})
}

func (p *ProgressNotifier) EmitStart(progressType ProgressType, id string, message string) {
	p.Emit(ProgressNotification{
		Type:       progressType,
		ID:         id,
		Message:    message,
		Percentage: 0,
		Status:     "started",
	})
}

func (p *ProgressNotifier) EmitComplete(progressType ProgressType, id string, message string) {
	p.Emit(ProgressNotification{
		Type:       progressType,
		ID:         id,
		Message:    message,
		Percentage: 100,
		Status:     "completed",
	})
}

func (p *ProgressNotifier) EmitError(progressType ProgressType, id string, message string, errorMsg string) {
	p.Emit(ProgressNotification{
		Type:       progressType,
		ID:         id,
		Message:    message,
		Percentage: 0,
		Status:     "error",
		Data: map[string]interface{}{
			"error": errorMsg,
		},
	})
}

func (p *ProgressNotifier) EmitWithData(progressType ProgressType, id string, message string, percentage float64, data map[string]interface{}) {
	p.Emit(ProgressNotification{
		Type:       progressType,
		ID:         id,
		Message:    message,
		Percentage: percentage,
		Status:     "in_progress",
		Data:       data,
	})
}
