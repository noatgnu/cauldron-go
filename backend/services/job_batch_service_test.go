package services

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestCreateBatch_PersistsRow(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	batchService := NewBatchService(db, jobQueue)

	batch, err := batchService.CreateBatch("My Batch", "cv-plot", "1.0.0", 3)
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if batch.ID == "" {
		t.Fatal("expected CreateBatch to generate a non-empty ID")
	}

	fetched, err := batchService.GetBatch(batch.ID)
	if err != nil {
		t.Fatalf("GetBatch failed: %v", err)
	}
	if fetched.Label != "My Batch" || fetched.PluginID != "cv-plot" || fetched.ExpectedCount != 3 {
		t.Errorf("fetched batch does not match created batch: %+v", fetched)
	}
}

func TestGetBatchStatus_DerivesCountsFromChildJobs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	batchService := NewBatchService(db, jobQueue)

	batch, err := batchService.CreateBatch("Status Batch", "normalization", "1.0.0", 3)
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}

	// Seed jobs directly in the DB (bypassing the real worker pool) so their status
	// is only ever changed explicitly by this test, not raced by a worker actually
	// processing them (a job with no Args completes almost instantly).
	var jobIDs []string
	for i := 0; i < 3; i++ {
		job := &models.Job{
			ID:        fmt.Sprintf("status-batch-job-%d", i),
			Type:      "normalization",
			Name:      "Job",
			Status:    models.JobStatusPending,
			Args:      []string{},
			BatchID:   batch.ID,
			CreatedAt: time.Now(),
		}
		if err := db.GetDB().Create(job).Error; err != nil {
			t.Fatalf("Failed to seed job %d: %v", i, err)
		}
		jobIDs = append(jobIDs, job.ID)
	}

	statusBefore, err := batchService.GetBatchStatus(batch.ID)
	if err != nil {
		t.Fatalf("GetBatchStatus failed: %v", err)
	}
	if statusBefore.TotalJobs != 3 || statusBefore.PendingCount != 3 {
		t.Fatalf("expected 3 pending jobs before any status change, got total=%d pending=%d", statusBefore.TotalJobs, statusBefore.PendingCount)
	}

	if err := db.GetDB().Model(&models.Job{}).Where("id = ?", jobIDs[0]).Update("status", models.JobStatusCompleted).Error; err != nil {
		t.Fatalf("Failed to update job status: %v", err)
	}
	if err := db.GetDB().Model(&models.Job{}).Where("id = ?", jobIDs[1]).Update("status", models.JobStatusFailed).Error; err != nil {
		t.Fatalf("Failed to update job status: %v", err)
	}

	statusAfter, err := batchService.GetBatchStatus(batch.ID)
	if err != nil {
		t.Fatalf("GetBatchStatus failed: %v", err)
	}
	if statusAfter.CompletedCount != 1 || statusAfter.FailedCount != 1 || statusAfter.PendingCount != 1 {
		t.Errorf("expected counts to reflect the mutated statuses (completed=1, failed=1, pending=1), got completed=%d failed=%d pending=%d",
			statusAfter.CompletedCount, statusAfter.FailedCount, statusAfter.PendingCount)
	}
}

func TestDeleteBatch_CascadesAndCancelsInProgress(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	batchService := NewBatchService(db, jobQueue)

	batch, err := batchService.CreateBatch("Delete Batch", "normalization", "1.0.0", 2)
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}

	jobID1, err := jobQueue.CreateJobWithEnvironmentsAndBatch(
		"normalization", "Job 1", "", []string{}, map[string]interface{}{}, "", "", "", "", "", "", batch.ID,
	)
	if err != nil {
		t.Fatalf("Failed to create job 1: %v", err)
	}
	jobID2, err := jobQueue.CreateJobWithEnvironmentsAndBatch(
		"normalization", "Job 2", "", []string{}, map[string]interface{}{}, "", "", "", "", "", "", batch.ID,
	)
	if err != nil {
		t.Fatalf("Failed to create job 2: %v", err)
	}

	if _, err := jobQueue.UpdateJob(jobID1, func(job *models.Job) {
		job.Status = models.JobStatusInProgress
	}); err != nil {
		t.Fatalf("Failed to mark job 1 in_progress: %v", err)
	}

	cancelled := false
	_, cancel := context.WithCancel(context.Background())
	jobQueue.RegisterJobCancelFunc(jobID1, func() {
		cancelled = true
		cancel()
	})

	if err := batchService.DeleteBatch(batch.ID); err != nil {
		t.Fatalf("DeleteBatch failed: %v", err)
	}

	if !cancelled {
		t.Error("expected the in-progress job's cancel function to be invoked by DeleteBatch")
	}

	if _, err := jobQueue.GetJob(jobID1); err == nil {
		t.Error("expected job 1 to be deleted")
	}
	if _, err := jobQueue.GetJob(jobID2); err == nil {
		t.Error("expected job 2 to be deleted")
	}
	if _, err := batchService.GetBatch(batch.ID); err == nil {
		t.Error("expected the batch row itself to be deleted")
	}
}
