package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/noatgnu/cauldron-go/backend/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func createTestDB(t *testing.T) *DatabaseService {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	service := &DatabaseService{
		ctx: context.Background(),
		db:  db,
	}

	if err := service.autoMigrate(); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return service
}

func TestDatabaseCreation(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	t.Log("✓ Database created and migrated successfully")

	// Test Settings table
	t.Run("Settings Table", func(t *testing.T) {
		err := db.SaveSetting("test_key", "test_value")
		if err != nil {
			t.Fatalf("Failed to save setting: %v", err)
		}

		value, err := db.GetSetting("test_key")
		if err != nil {
			t.Fatalf("Failed to get setting: %v", err)
		}

		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", value)
		}

		t.Log("✓ Settings table works")
	})

	// Test ImportedFile table
	t.Run("ImportedFile Table", func(t *testing.T) {
		file := &ImportedFile{
			Name:       "test.csv",
			Path:       "/tmp/test.csv",
			Size:       1024,
			ImportedAt: 1234567890,
			FileType:   "csv",
			Preview:    "10 rows, 5 columns",
		}

		err := db.GetDB().Create(file).Error
		if err != nil {
			t.Fatalf("Failed to create imported file: %v", err)
		}

		var retrieved ImportedFile
		err = db.GetDB().First(&retrieved, file.ID).Error
		if err != nil {
			t.Fatalf("Failed to retrieve imported file: %v", err)
		}

		if retrieved.Name != "test.csv" {
			t.Errorf("Expected 'test.csv', got '%s'", retrieved.Name)
		}

		files, err := db.GetImportedFiles()
		if err != nil {
			t.Fatalf("Failed to get imported files: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}

		t.Log("✓ ImportedFile table works")
	})

	// Test Job table
	t.Run("Job Table", func(t *testing.T) {
		job := &models.Job{
			ID:             uuid.New().String(),
			Type:           "pca",
			Name:           "Test PCA",
			Status:         models.JobStatusPending,
			Progress:       0,
			Command:        "python",
			Args:           []string{"pca.py"},
			TerminalOutput: []string{},
		}

		err := db.GetDB().Create(job).Error
		if err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		var retrieved models.Job
		err = db.GetDB().First(&retrieved, "id = ?", job.ID).Error
		if err != nil {
			t.Fatalf("Failed to retrieve job: %v", err)
		}

		if retrieved.Name != "Test PCA" {
			t.Errorf("Expected 'Test PCA', got '%s'", retrieved.Name)
		}

		if retrieved.Type != "pca" {
			t.Errorf("Expected 'pca', got '%s'", retrieved.Type)
		}

		t.Log("✓ Job table works")
	})

	t.Log("\n=== All database tables created and functional ===")
}

func TestDatabaseAutoMigration(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Verify all tables exist by checking if we can query them
	tables := []string{"settings", "imported_files", "jobs"}
	for _, table := range tables {
		var count int64
		err := db.GetDB().Table(table).Count(&count).Error
		if err != nil {
			t.Errorf("Table '%s' does not exist or is not accessible: %v", table, err)
		} else {
			t.Logf("✓ Table '%s' exists (count: %d)", table, count)
		}
	}
}

func TestJobQueueService(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	ctx := context.Background()

	jobQueue := NewJobQueueService(ctx, db)
	defer jobQueue.Shutdown()

	// Create a job
	jobID, err := jobQueue.CreateJob("test", "Test Job", "python", []string{"test.py"})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	t.Logf("✓ Created job with ID: %s", jobID)

	// Retrieve the job
	job, err := jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Name != "Test Job" {
		t.Errorf("Expected 'Test Job', got '%s'", job.Name)
	}

	t.Log("✓ Retrieved job successfully")

	// Get all jobs
	jobs := jobQueue.GetAllJobs()
	if len(jobs) == 0 {
		t.Error("Expected at least 1 job")
	}

	t.Logf("✓ GetAllJobs returned %d job(s)", len(jobs))

	// Delete the job
	err = jobQueue.DeleteJob(jobID)
	if err != nil {
		t.Fatalf("Failed to delete job: %v", err)
	}

	t.Log("✓ Deleted job successfully")

	// Verify job is deleted
	_, err = jobQueue.GetJob(jobID)
	if err == nil {
		t.Error("Expected error when getting deleted job, got nil")
	}

	t.Log("✓ Job deleted from database")
}
