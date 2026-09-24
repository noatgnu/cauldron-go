package services

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/noatgnu/cauldron-go/backend/models"
)

type BatchStatus struct {
	BatchID         string        `json:"batchId"`
	Label           string        `json:"label"`
	PluginID        string        `json:"pluginId"`
	PluginVersion   string        `json:"pluginVersion,omitempty"`
	ExpectedCount   int           `json:"expectedCount"`
	TotalJobs       int           `json:"totalJobs"`
	PendingCount    int           `json:"pendingCount"`
	InProgressCount int           `json:"inProgressCount"`
	CompletedCount  int           `json:"completedCount"`
	FailedCount     int           `json:"failedCount"`
	CreatedAt       time.Time     `json:"createdAt"`
	Jobs            []*models.Job `json:"jobs"`
}

type BatchService struct {
	db       *DatabaseService
	jobQueue *JobQueueService
}

func NewBatchService(db *DatabaseService, jobQueue *JobQueueService) *BatchService {
	return &BatchService{db: db, jobQueue: jobQueue}
}

func (b *BatchService) CreateBatch(label, pluginID, pluginVersion string, expectedCount int) (*models.JobBatch, error) {
	batch := &models.JobBatch{
		ID:            uuid.New().String(),
		Label:         label,
		PluginID:      pluginID,
		PluginVersion: pluginVersion,
		ExpectedCount: expectedCount,
		CreatedAt:     time.Now(),
	}
	if err := b.db.GetDB().Create(batch).Error; err != nil {
		return nil, err
	}
	return batch, nil
}

func (b *BatchService) GetBatch(id string) (*models.JobBatch, error) {
	var batch models.JobBatch
	if err := b.db.GetDB().First(&batch, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

func (b *BatchService) GetAllBatches(limit, offset int) ([]*models.JobBatch, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var batches []*models.JobBatch
	err := b.db.GetDB().Order("created_at DESC").Limit(limit).Offset(offset).Find(&batches).Error
	return batches, err
}

func (b *BatchService) GetBatchStatus(id string) (*BatchStatus, error) {
	batch, err := b.GetBatch(id)
	if err != nil {
		return nil, err
	}
	jobs := b.jobQueue.GetJobsByBatchID(id)

	status := &BatchStatus{
		BatchID:       batch.ID,
		Label:         batch.Label,
		PluginID:      batch.PluginID,
		PluginVersion: batch.PluginVersion,
		ExpectedCount: batch.ExpectedCount,
		CreatedAt:     batch.CreatedAt,
		Jobs:          jobs,
		TotalJobs:     len(jobs),
	}
	for _, jb := range jobs {
		switch jb.Status {
		case models.JobStatusPending:
			status.PendingCount++
		case models.JobStatusInProgress:
			status.InProgressCount++
		case models.JobStatusCompleted:
			status.CompletedCount++
		case models.JobStatusFailed:
			status.FailedCount++
		}
	}
	return status, nil
}

func (b *BatchService) DeleteBatch(id string) error {
	jobs := b.jobQueue.GetJobsByBatchID(id)
	for _, jb := range jobs {
		if err := b.jobQueue.DeleteJob(jb.ID); err != nil {
			log.Printf("[DeleteBatch] failed to delete job %s in batch %s: %v", jb.ID, id, err)
		}
	}
	return b.db.GetDB().Delete(&models.JobBatch{}, "id = ?", id).Error
}
