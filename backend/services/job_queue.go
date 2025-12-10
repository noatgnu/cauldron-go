package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type JobQueueService struct {
	ctx           context.Context
	db            *DatabaseService
	jobs          map[string]*models.Job
	queue         chan *models.Job
	workers       int
	mu            sync.RWMutex
	wg            sync.WaitGroup
	pythonRunner  *PythonRunner
	rRunner       *RRunner
	directRunner  *DirectRunner
	settingsServ  *SettingsService
	paused        bool
	stopImmediate bool
	currentJobID  string
	cancelFunc    context.CancelFunc
}

func NewJobQueueService(ctx context.Context, db *DatabaseService) *JobQueueService {
	service := &JobQueueService{
		ctx:     ctx,
		db:      db,
		jobs:    make(map[string]*models.Job),
		queue:   make(chan *models.Job, 100),
		workers: 2,
	}

	service.loadFromDatabase()

	for i := 0; i < service.workers; i++ {
		service.wg.Add(1)
		go service.worker()
	}

	return service
}

func (j *JobQueueService) SetRunners(pythonRunner *PythonRunner, rRunner *RRunner, directRunner *DirectRunner, settings *SettingsService) {
	j.pythonRunner = pythonRunner
	j.rRunner = rRunner
	j.directRunner = directRunner
	j.settingsServ = settings
}

func (j *JobQueueService) worker() {
	defer j.wg.Done()

	for job := range j.queue {
		j.mu.RLock()
		isPaused := j.paused
		j.mu.RUnlock()

		if isPaused {
			log.Printf("[worker] Queue is paused, requeueing job: %s", job.ID)
			j.mu.Lock()
			j.queue <- job
			j.mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		j.processJob(job)
	}
}

func (j *JobQueueService) CreateJob(jobType string, name string, command string, args []string) (string, error) {
	return j.CreateJobWithParameters(jobType, name, command, args, make(map[string]interface{}))
}

func (j *JobQueueService) CreateJobWithParameters(jobType string, name string, command string, args []string, parameters map[string]interface{}) (string, error) {
	pythonPath := ""
	pythonEnvType := ""
	rPath := ""
	rEnvType := ""

	if command == "python" {
		pythonEnv, err := j.db.GetActivePythonEnvironment()
		if err == nil && pythonEnv != nil {
			pythonPath = pythonEnv.Path
			pythonEnvType = pythonEnv.Type
		}
	} else if command == "r" {
		rEnv, err := j.db.GetActiveREnvironment()
		if err == nil && rEnv != nil {
			rPath = rEnv.Path
			rEnvType = rEnv.Type
		}
	} else {
		pythonEnv, err := j.db.GetActivePythonEnvironment()
		if err == nil && pythonEnv != nil {
			pythonPath = pythonEnv.Path
			pythonEnvType = pythonEnv.Type
		}

		rEnv, err := j.db.GetActiveREnvironment()
		if err == nil && rEnv != nil {
			rPath = rEnv.Path
			rEnvType = rEnv.Type
		}
	}

	job := &models.Job{
		ID:             uuid.New().String(),
		Type:           jobType,
		Name:           name,
		Status:         models.JobStatusPending,
		Progress:       0,
		Command:        command,
		Args:           args,
		Parameters:     parameters,
		PythonEnvPath:  pythonPath,
		PythonEnvType:  pythonEnvType,
		REnvPath:       rPath,
		REnvType:       rEnvType,
		TerminalOutput: []string{},
		CreatedAt:      time.Now(),
	}

	j.mu.Lock()
	j.jobs[job.ID] = job
	j.mu.Unlock()

	if err := j.db.GetDB().Create(job).Error; err != nil {
		return "", err
	}

	j.queue <- job
	j.emitJobUpdate(job)

	return job.ID, nil
}

func (j *JobQueueService) GetJob(id string) (*models.Job, error) {
	j.mu.RLock()
	job, ok := j.jobs[id]
	j.mu.RUnlock()

	if ok {
		return job, nil
	}

	var dbJob models.Job
	if err := j.db.GetDB().First(&dbJob, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	j.mu.Lock()
	j.jobs[id] = &dbJob
	j.mu.Unlock()

	return &dbJob, nil
}

func (j *JobQueueService) GetAllJobs() []*models.Job {
	var jobs []*models.Job

	if j.db == nil || j.db.GetDB() == nil {
		log.Println("[GetAllJobs] ERROR: Database is nil!")
		return jobs
	}

	log.Println("[GetAllJobs] Starting database query...")
	result := j.db.GetDB().Order("created_at DESC").Limit(100).Find(&jobs)

	if result.Error != nil {
		log.Printf("[GetAllJobs] ERROR: Database query failed: %v\n", result.Error)
		return jobs
	}

	log.Printf("[GetAllJobs] SUCCESS: Found %d jobs\n", len(jobs))
	return jobs
}

func (j *JobQueueService) DeleteJob(id string) error {
	j.mu.Lock()
	delete(j.jobs, id)
	j.mu.Unlock()

	return j.db.GetDB().Delete(&models.Job{}, "id = ?", id).Error
}

func (j *JobQueueService) ValidateJobEnvironment(job *models.Job) error {
	if job.PythonEnvPath != "" {
		envs, err := j.db.GetPythonEnvironments()
		if err != nil {
			return fmt.Errorf("failed to get Python environments: %v", err)
		}

		found := false
		for _, env := range envs {
			if env.Path == job.PythonEnvPath {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Python environment not found: %s (type: %s). Please select a new environment", job.PythonEnvPath, job.PythonEnvType)
		}
	}

	if job.REnvPath != "" {
		log.Printf("[ValidateJobEnvironment] Validating R environment: %s (type: %s)", job.REnvPath, job.REnvType)

		if _, err := os.Stat(job.REnvPath); os.IsNotExist(err) {
			log.Printf("[ValidateJobEnvironment] ERROR: R environment file does not exist: %s", job.REnvPath)
			return fmt.Errorf("R environment not found: %s (type: %s). Please select a new environment", job.REnvPath, job.REnvType)
		}

		envs, err := j.db.GetREnvironments()
		if err != nil {
			log.Printf("[ValidateJobEnvironment] ERROR: Failed to get R environments from DB: %v", err)
			return fmt.Errorf("failed to get R environments: %v", err)
		}

		log.Printf("[ValidateJobEnvironment] Found %d R environments in DB", len(envs))
		for i, env := range envs {
			log.Printf("[ValidateJobEnvironment] DB R env %d: Path=%s, Type=%s", i, env.Path, env.Type)
		}

		found := false
		for _, env := range envs {
			if env.Path == job.REnvPath {
				found = true
				break
			}
		}

		if !found {
			log.Printf("[ValidateJobEnvironment] ERROR: R environment path not found in DB")
			return fmt.Errorf("R environment not found: %s (type: %s). Please select a new environment", job.REnvPath, job.REnvType)
		}
		log.Printf("[ValidateJobEnvironment] R environment validated successfully")
	}

	return nil
}

func (j *JobQueueService) processJob(job *models.Job) {
	j.mu.Lock()
	j.currentJobID = job.ID
	j.mu.Unlock()

	defer func() {
		j.mu.Lock()
		j.currentJobID = ""
		j.mu.Unlock()
	}()

	now := time.Now()
	job.StartedAt = &now
	job.Status = models.JobStatusInProgress

	j.db.GetDB().Save(job)
	j.emitJobUpdate(job)

	j.mu.RLock()
	shouldStopImmediate := j.stopImmediate
	j.mu.RUnlock()

	if shouldStopImmediate {
		log.Printf("[processJob] Immediate stop requested, canceling job: %s", job.ID)
		completedTime := time.Now()
		job.CompletedAt = &completedTime
		job.Status = models.JobStatusFailed
		job.Error = "Job stopped by user request"
		j.db.GetDB().Save(job)
		j.emitJobUpdate(job)
		return
	}

	if err := j.ValidateJobEnvironment(job); err != nil {
		completedTime := time.Now()
		job.CompletedAt = &completedTime
		job.Status = models.JobStatusFailed
		job.Error = err.Error()
		j.db.GetDB().Save(job)
		j.emitJobUpdate(job)
		return
	}

	if len(job.Args) == 0 {
		completedTime := time.Now()
		job.CompletedAt = &completedTime
		job.Status = models.JobStatusCompleted
		job.Progress = 100
		j.db.GetDB().Save(job)
		j.emitJobUpdate(job)
		return
	}

	var err error
	outputCallback := func(line string) {
		job.TerminalOutput = append(job.TerminalOutput, line)
		j.db.GetDB().Save(job)
		if j.ctx.Value("wails-test") == nil {
			runtime.EventsEmit(j.ctx, "job:output", map[string]interface{}{
				"jobId":  job.ID,
				"output": line,
			})
		}
	}

	if job.Command == "r" {
		if j.rRunner == nil {
			completedTime := time.Now()
			job.CompletedAt = &completedTime
			job.Status = models.JobStatusFailed
			job.Error = "R runner not initialized"
			j.db.GetDB().Save(job)
			j.emitJobUpdate(job)
			return
		}
		err = j.rRunner.ExecuteScript(job.Args[0], job.Args[1:], outputCallback)
	} else if job.Command == "direct" {
		if j.directRunner == nil {
			completedTime := time.Now()
			job.CompletedAt = &completedTime
			job.Status = models.JobStatusFailed
			job.Error = "Direct runner not initialized"
			j.db.GetDB().Save(job)
			j.emitJobUpdate(job)
			return
		}
		var workingDir string
		if outputDir, ok := job.Parameters["outputDir"].(string); ok {
			workingDir = outputDir
		}
		err = j.directRunner.ExecuteProgram(job.Args[0], job.Args[1:], workingDir, outputCallback)
	} else {
		if j.pythonRunner == nil {
			completedTime := time.Now()
			job.CompletedAt = &completedTime
			job.Status = models.JobStatusFailed
			job.Error = "Python runner not initialized"
			j.db.GetDB().Save(job)
			j.emitJobUpdate(job)
			return
		}
		err = j.pythonRunner.ExecuteScript(job.Args[0], job.Args[1:], outputCallback)
	}

	completedTime := time.Now()
	job.CompletedAt = &completedTime

	if err != nil {
		job.Status = models.JobStatusFailed
		job.Error = err.Error()
	} else {
		job.Status = models.JobStatusCompleted
		job.Progress = 100
	}

	if outputDir, ok := job.Parameters["outputDir"].(string); ok && outputDir != "" {
		job.OutputPath = outputDir
	}

	j.db.GetDB().Save(job)
	j.emitJobUpdate(job)
}

func (j *JobQueueService) emitJobUpdate(job *models.Job) {
	// Skip events in test mode
	if j.ctx.Value("wails-test") != nil {
		return
	}
	runtime.EventsEmit(j.ctx, "job:update", job)
}

func (j *JobQueueService) RerunJob(jobID string, useSameEnvironment bool, pythonEnvPath string, rEnvPath string) (string, error) {
	originalJob, err := j.GetJob(jobID)
	if err != nil {
		return "", fmt.Errorf("failed to get original job: %v", err)
	}

	var newPythonPath, newPythonType, newRPath, newRType string

	if useSameEnvironment {
		newPythonPath = originalJob.PythonEnvPath
		newPythonType = originalJob.PythonEnvType
		newRPath = originalJob.REnvPath
		newRType = originalJob.REnvType
	} else {
		if pythonEnvPath != "" {
			newPythonPath = pythonEnvPath
			pythonEnv, err := j.db.GetPythonEnvironments()
			if err == nil {
				for _, env := range pythonEnv {
					if env.Path == pythonEnvPath {
						newPythonType = env.Type
						break
					}
				}
			}
		}

		if rEnvPath != "" {
			newRPath = rEnvPath
			rEnv, err := j.db.GetREnvironments()
			if err == nil {
				for _, env := range rEnv {
					if env.Path == rEnvPath {
						newRType = env.Type
						break
					}
				}
			}
		}
	}

	newJob := &models.Job{
		ID:             uuid.New().String(),
		Type:           originalJob.Type,
		Name:           originalJob.Name + " (Rerun)",
		Status:         models.JobStatusPending,
		Progress:       0,
		Command:        originalJob.Command,
		Args:           originalJob.Args,
		Parameters:     originalJob.Parameters,
		PythonEnvPath:  newPythonPath,
		PythonEnvType:  newPythonType,
		REnvPath:       newRPath,
		REnvType:       newRType,
		TerminalOutput: []string{},
		CreatedAt:      time.Now(),
	}

	j.mu.Lock()
	j.jobs[newJob.ID] = newJob
	j.mu.Unlock()

	if err := j.db.GetDB().Create(newJob).Error; err != nil {
		return "", err
	}

	j.queue <- newJob
	j.emitJobUpdate(newJob)

	return newJob.ID, nil
}

func (j *JobQueueService) loadFromDatabase() error {
	log.Println("[loadFromDatabase] Starting...")

	if j.db == nil || j.db.GetDB() == nil {
		log.Println("[loadFromDatabase] ERROR: Database is nil!")
		return fmt.Errorf("database not initialized")
	}

	var jobs []models.Job
	log.Println("[loadFromDatabase] Querying database...")
	result := j.db.GetDB().Order("created_at DESC").Limit(100).Find(&jobs)

	if result.Error != nil {
		log.Printf("[loadFromDatabase] ERROR: %v\n", result.Error)
		return result.Error
	}

	log.Printf("[loadFromDatabase] Found %d jobs in database\n", len(jobs))

	j.mu.Lock()
	for i := range jobs {
		j.jobs[jobs[i].ID] = &jobs[i]
	}
	j.mu.Unlock()

	log.Println("[loadFromDatabase] Complete")
	return nil
}

func (j *JobQueueService) Shutdown() {
	close(j.queue)
	j.wg.Wait()
}

func (j *JobQueueService) UpdateJobProgress(id string, progress float64, output string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	job, ok := j.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Progress = progress
	if output != "" {
		job.TerminalOutput = append(job.TerminalOutput, output)
	}

	j.db.GetDB().Save(job)
	j.emitJobUpdate(job)

	return nil
}

func (j *JobQueueService) FailJob(id string, errorMsg string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	job, ok := j.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	completedTime := time.Now()
	job.CompletedAt = &completedTime
	job.Status = models.JobStatusFailed
	job.Error = errorMsg

	j.db.GetDB().Save(job)
	j.emitJobUpdate(job)

	return nil
}

func (j *JobQueueService) CompleteJob(id string, outputPath string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	job, ok := j.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	completedTime := time.Now()
	job.CompletedAt = &completedTime
	job.Status = models.JobStatusCompleted
	job.Progress = 100
	job.OutputPath = outputPath

	j.db.GetDB().Save(job)
	j.emitJobUpdate(job)

	return nil
}

func (j *JobQueueService) GetJobsByStatus(status models.JobStatus) []*models.Job {
	var jobs []*models.Job
	j.db.GetDB().Where("status = ?", status).Order("created_at DESC").Find(&jobs)
	return jobs
}

func (j *JobQueueService) SearchJobs(query string) []*models.Job {
	var jobs []*models.Job
	query = strings.ToLower(query)
	searchPattern := "%" + query + "%"

	j.db.GetDB().
		Where("LOWER(name) LIKE ? OR LOWER(type) LIKE ?", searchPattern, searchPattern).
		Order("created_at DESC").
		Find(&jobs)

	return jobs
}

func (j *JobQueueService) RequeueJob(job *models.Job) {
	j.mu.Lock()
	j.jobs[job.ID] = job
	j.mu.Unlock()

	j.queue <- job
	j.emitJobUpdate(job)
}

func (j *JobQueueService) PauseQueue() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.paused {
		return fmt.Errorf("queue is already paused")
	}

	j.paused = true
	log.Println("[PauseQueue] Queue paused - will finish current job then stop processing")

	if j.ctx.Value("wails-test") == nil {
		runtime.EventsEmit(j.ctx, "queue:status", map[string]interface{}{
			"paused":        true,
			"stopImmediate": false,
		})
	}

	return nil
}

func (j *JobQueueService) StopQueueImmediate() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.paused = true
	j.stopImmediate = true
	log.Println("[StopQueueImmediate] Queue stopped immediately - canceling current job and pausing queue")

	if j.currentJobID != "" {
		log.Printf("[StopQueueImmediate] Marking current job %s for cancellation", j.currentJobID)
	}

	if j.ctx.Value("wails-test") == nil {
		runtime.EventsEmit(j.ctx, "queue:status", map[string]interface{}{
			"paused":        true,
			"stopImmediate": true,
		})
	}

	return nil
}

func (j *JobQueueService) ResumeQueue() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.paused {
		return fmt.Errorf("queue is not paused")
	}

	j.paused = false
	j.stopImmediate = false
	log.Println("[ResumeQueue] Queue resumed - processing will continue")

	if j.ctx.Value("wails-test") == nil {
		runtime.EventsEmit(j.ctx, "queue:status", map[string]interface{}{
			"paused":        false,
			"stopImmediate": false,
		})
	}

	var pendingJobs []*models.Job
	j.db.GetDB().Where("status = ?", models.JobStatusPending).Order("created_at ASC").Find(&pendingJobs)

	for _, job := range pendingJobs {
		if _, exists := j.jobs[job.ID]; !exists {
			j.jobs[job.ID] = job
			j.queue <- job
			log.Printf("[ResumeQueue] Requeued pending job: %s - %s", job.ID, job.Name)
		}
	}

	return nil
}

func (j *JobQueueService) GetQueueStatus() map[string]interface{} {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var pendingCount int64
	var inProgressCount int64

	j.db.GetDB().Model(&models.Job{}).Where("status = ?", models.JobStatusPending).Count(&pendingCount)
	j.db.GetDB().Model(&models.Job{}).Where("status = ?", models.JobStatusInProgress).Count(&inProgressCount)

	return map[string]interface{}{
		"paused":          j.paused,
		"stopImmediate":   j.stopImmediate,
		"currentJobID":    j.currentJobID,
		"pendingCount":    pendingCount,
		"inProgressCount": inProgressCount,
		"queueLength":     len(j.queue),
	}
}
