package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type JobQueueService struct {
	ctx            context.Context
	db             *DatabaseService
	jobs           map[string]*models.Job
	queue          chan *models.Job
	workers        int
	mu             sync.RWMutex
	wg             sync.WaitGroup
	scriptExecutor *ScriptExecutor
	pluginLoader   *PluginLoaderV2
	settingsServ   *SettingsService
	paused         bool
	stopImmediate  bool
	currentJobID   string
	cancelFuncs    map[string]context.CancelFunc
	shutdownChan   chan struct{}
	wailsApp       *application.App
}

// GenerateJobOutputDir builds a pluginID+timestamp+random output dir name so same-second concurrent runs of the same plugin never collide.
func GenerateJobOutputDir(baseDir, pluginID string) string {
	return filepath.Join(baseDir, fmt.Sprintf("%s_%s_%s",
		pluginID,
		time.Now().Format("20060102_150405"),
		uuid.New().String()[:8]))
}

func getScriptName(plugin *models.PluginV2) string {
	if plugin.Definition.Runtime.IsDockerRuntime() {
		return plugin.Definition.Runtime.GetEntrypoint()
	}
	return filepath.Base(plugin.ScriptPath)
}

func NewJobQueueService(ctx context.Context, db *DatabaseService) *JobQueueService {
	return newJobQueueServiceInternal(db, ctx)
}

func newJobQueueServiceInternal(db *DatabaseService, ctx context.Context) *JobQueueService {
	if ctx == nil {
		ctx = context.Background()
	}
	service := &JobQueueService{
		ctx:          ctx,
		db:           db,
		jobs:         make(map[string]*models.Job),
		queue:        make(chan *models.Job, 100),
		workers:      2,
		cancelFuncs:  make(map[string]context.CancelFunc),
		shutdownChan: make(chan struct{}),
	}

	service.loadFromDatabase()

	for i := 0; i < service.workers; i++ {
		service.wg.Add(1)
		go service.worker()
	}

	return service
}

func (j *JobQueueService) SetScriptExecutor(scriptExecutor *ScriptExecutor) {
	j.scriptExecutor = scriptExecutor
}

func (j *JobQueueService) SetPluginLoader(pluginLoader *PluginLoaderV2) {
	j.pluginLoader = pluginLoader
}

func (j *JobQueueService) worker() {
	defer j.wg.Done()

	for {
		// Check if paused before trying to pull from queue
		j.mu.RLock()
		isPaused := j.paused
		j.mu.RUnlock()

		if isPaused {
			// Don't consume from queue while paused
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Try to get a job from the queue with timeout
		select {
		case job, ok := <-j.queue:
			if !ok {
				// Queue channel closed
				return
			}

			// Double-check paused status after receiving job
			j.mu.RLock()
			isPaused = j.paused
			j.mu.RUnlock()

			if isPaused {
				// Queue was paused while we were getting the job
				// Put it back as pending for ResumeQueue to pick up
				log.Printf("[worker] Queue paused after receiving job, marking as pending: %s", job.ID)
				job.Status = models.JobStatusPending
				if err := j.db.GetDB().Model(job).Select("*").Updates(job).Error; err != nil {
					log.Printf("[worker] Failed to save job status: %v", err)
				}
				continue
			}

			// Process the job
			j.processJob(job)

		case <-j.shutdownChan:
			// Shutdown signal received
			return

		case <-time.After(1 * time.Second):
			// Timeout - just loop back to check paused status
			continue
		}
	}
}

func (j *JobQueueService) CreateJob(jobType string, name string, command string, args []string) (string, error) {
	return j.CreateJobWithParameters(jobType, name, command, args, make(map[string]interface{}), "", "")
}

func (j *JobQueueService) CreateJobWithEnvironments(jobType string, name string, command string, args []string, parameters map[string]interface{}, pluginVersion string, pluginCommitHash string, pythonPath string, pythonEnvType string, rPath string, rEnvType string) (string, error) {
	job := &models.Job{
		ID:               uuid.New().String(),
		Type:             jobType,
		Name:             name,
		Status:           models.JobStatusPending,
		Progress:         0,
		Command:          command,
		Args:             args,
		Parameters:       parameters,
		PythonEnvPath:    pythonPath,
		PythonEnvType:    pythonEnvType,
		REnvPath:         rPath,
		REnvType:         rEnvType,
		TerminalOutput:   []string{},
		PluginVersion:    pluginVersion,
		PluginCommitHash: pluginCommitHash,
		CreatedAt:        time.Now(),
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

func (j *JobQueueService) CreateJobWithParameters(jobType string, name string, command string, args []string, parameters map[string]interface{}, pluginVersion string, pluginCommitHash string) (string, error) {
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

	return j.CreateJobWithEnvironments(jobType, name, command, args, parameters, pluginVersion, pluginCommitHash, pythonPath, pythonEnvType, rPath, rEnvType)
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

	j.db.GetDB().Model(job).Select("*").Updates(job)
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
		j.db.GetDB().Model(job).Select("*").Updates(job)
		j.emitJobUpdate(job)
		return
	}

	if err := j.ValidateJobEnvironment(job); err != nil {
		completedTime := time.Now()
		job.CompletedAt = &completedTime
		job.Status = models.JobStatusFailed
		job.Error = err.Error()
		j.db.GetDB().Model(job).Select("*").Updates(job)
		j.emitJobUpdate(job)
		return
	}

	if len(job.Args) == 0 {
		completedTime := time.Now()
		job.CompletedAt = &completedTime
		job.Status = models.JobStatusCompleted
		job.Progress = 100
		j.db.GetDB().Model(job).Select("*").Updates(job)
		j.emitJobUpdate(job)
		return
	}

	var err error

	var pluginID uint
	var isPluginV2Job bool
	if pid, ok := job.Parameters["pluginId"].(uint); ok && pid > 0 {
		pluginID = pid
		isPluginV2Job = true
	} else if pid, ok := job.Parameters["pluginId"].(float64); ok && pid > 0 {
		pluginID = uint(pid)
		isPluginV2Job = true
	}

	if isPluginV2Job && j.scriptExecutor != nil && j.pluginLoader != nil {
		log.Printf("[processJob] Processing plugin v2 job %s (pluginId: %d)", job.ID, pluginID)

		plugin, err := j.pluginLoader.GetPlugin(pluginID)
		if err != nil {
			completedTime := time.Now()
			job.CompletedAt = &completedTime
			job.Status = models.JobStatusFailed
			job.Error = fmt.Sprintf("Failed to load plugin: %v", err)
			j.db.GetDB().Model(job).Select("*").Updates(job)
			j.emitJobUpdate(job)
			return
		}

		log.Printf("[processJob] Loaded plugin: Name=%s, ID=%s, FolderPath=%s",
			plugin.Definition.Plugin.Name, plugin.Definition.Plugin.ID, plugin.FolderPath)

		var outputDir string
		if od, ok := job.Parameters["outputDir"].(string); ok {
			outputDir = od
		}

		config := ScriptConfig{
			PluginID:     pluginID,
			Type:         plugin.Definition.Plugin.ID,
			Environments: plugin.Definition.Runtime.GetEnvironments(),
			ScriptName:   getScriptName(plugin),
			Args:         job.Args[1:],
			OutputDir:    outputDir,
			FolderPath:   plugin.FolderPath,
		}

		log.Printf("[processJob] Created ScriptConfig with Type='%s' for plugin binding lookup", config.Type)

		jobCtx, cancel := context.WithCancel(j.ctx)
		defer cancel()
		j.RegisterJobCancelFunc(job.ID, cancel)
		defer j.UnregisterJobCancelFunc(job.ID)

		envs := config.Environments
		if len(envs) == 0 {
			completedTime := time.Now()
			job.CompletedAt = &completedTime
			job.Status = models.JobStatusFailed
			job.Error = "No runtime environments specified"
			j.db.GetDB().Model(job).Select("*").Updates(job)
			j.emitJobUpdate(job)
			return
		}

		primaryEnv := envs[0]
		log.Printf("[processJob] Executing plugin v2 job with primary environment: %s", primaryEnv)

		switch primaryEnv {
		case "python":
			err = j.scriptExecutor.ExecutePythonScript(jobCtx, job.ID, config)
		case "r":
			err = j.scriptExecutor.ExecuteRScript(jobCtx, job.ID, config)
		case "julia":
			err = fmt.Errorf("julia runtime not yet implemented")
		case "node":
			err = fmt.Errorf("node runtime not yet implemented")
		case "docker":
			err = j.scriptExecutor.ExecuteDockerScript(jobCtx, job.ID, config)
		case "direct":
			err = j.scriptExecutor.ExecuteDirectScript(jobCtx, job.ID, config)
		default:
			err = fmt.Errorf("unsupported primary environment: %s", primaryEnv)
		}
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

	j.mu.RLock()
	_, stillExists := j.jobs[job.ID]
	j.mu.RUnlock()
	if !stillExists {
		return
	}

	j.db.GetDB().Model(job).Select("*").Updates(job)
	j.emitJobUpdate(job)
}

func (j *JobQueueService) emitJobUpdate(job *models.Job) {
	if j.ctx != nil && j.ctx.Value("wails-test") != nil {
		return
	}
	j.emitEvent("job:update", job)
}

func (j *JobQueueService) emitEvent(name string, data interface{}) {
	if j.wailsApp != nil && j.wailsApp.Event != nil {
		j.wailsApp.Event.Emit(name, data)
	}
}

func (j *JobQueueService) SetWailsApp(wailsApp *application.App) {
	j.wailsApp = wailsApp
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

	newArgs := make([]string, len(originalJob.Args))
	copy(newArgs, originalJob.Args)

	newParameters := make(map[string]interface{}, len(originalJob.Parameters))
	for k, v := range originalJob.Parameters {
		newParameters[k] = v
	}

	if oldOutputDir, ok := originalJob.Parameters["outputDir"].(string); ok && oldOutputDir != "" {
		newOutputDir := GenerateJobOutputDir(filepath.Dir(oldOutputDir), originalJob.Type)

		// Prefer the plugin's real output flag over guessing, so a miss never falls back to silently overwriting the original run's output.
		candidateFlags := []string{j.pluginOutputFlag(originalJob.Type), "--output_folder", "--output_dir", "-o"}
		replaced := false
		for _, flag := range candidateFlags {
			if flag == "" || replaced {
				continue
			}
			for i := 0; i < len(newArgs)-1; i++ {
				if newArgs[i] == flag && newArgs[i+1] == oldOutputDir {
					newArgs[i+1] = newOutputDir
					replaced = true
					break
				}
			}
		}

		if !replaced {
			return "", fmt.Errorf("could not locate the output directory argument to rewrite for rerun; refusing to reuse %s and overwrite its results", oldOutputDir)
		}

		newParameters["outputDir"] = newOutputDir
	}

	newJob := &models.Job{
		ID:               uuid.New().String(),
		Type:             originalJob.Type,
		Name:             originalJob.Name + " (Rerun)",
		Status:           models.JobStatusPending,
		Progress:         0,
		Command:          originalJob.Command,
		Args:             newArgs,
		Parameters:       newParameters,
		PythonEnvPath:    newPythonPath,
		PythonEnvType:    newPythonType,
		REnvPath:         newRPath,
		REnvType:         newRType,
		TerminalOutput:   []string{},
		PluginVersion:    originalJob.PluginVersion,
		PluginCommitHash: originalJob.PluginCommitHash,
		CreatedAt:        time.Now(),
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

// pluginOutputFlag looks up the CLI flag a plugin uses for its output directory (plugin.yaml's execution.outputDir).
func (j *JobQueueService) pluginOutputFlag(pluginStringID string) string {
	if j.pluginLoader == nil {
		return ""
	}
	plugin, err := j.pluginLoader.GetPluginByStringID(pluginStringID)
	if err != nil {
		return ""
	}
	return plugin.Definition.Execution.OutputDir
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

	// Load all jobs into memory
	j.mu.Lock()
	for i := range jobs {
		j.jobs[jobs[i].ID] = &jobs[i]
	}
	j.mu.Unlock()

	// Find and queue pending jobs
	var pendingJobs []*models.Job
	for i := range jobs {
		if jobs[i].Status == models.JobStatusPending {
			pendingJobs = append(pendingJobs, &jobs[i])
		}
	}

	if len(pendingJobs) > 0 {
		log.Printf("[loadFromDatabase] Found %d pending jobs, queueing them...", len(pendingJobs))

		// Queue pending jobs in background to avoid blocking startup
		go func() {
			for _, job := range pendingJobs {
				j.queue <- job
				log.Printf("[loadFromDatabase] Queued pending job: %s - %s", job.ID, job.Name)
			}
			log.Printf("[loadFromDatabase] Finished queueing %d pending jobs", len(pendingJobs))
		}()
	}

	log.Println("[loadFromDatabase] Complete")
	return nil
}

func (j *JobQueueService) Shutdown() {
	log.Println("[Shutdown] Closing job queue...")

	// Signal all workers to shut down immediately
	close(j.shutdownChan)

	// Workers only check shutdownChan between jobs, so kill in-flight subprocesses first or a blocked worker would outlive the app as an orphan.
	if j.scriptExecutor != nil {
		log.Println("[Shutdown] Killing any in-flight job subprocesses...")
		j.scriptExecutor.KillAllJobs()
	}

	// Close the queue channel
	close(j.queue)

	log.Println("[Shutdown] Waiting for workers to finish (max 10 seconds)...")
	done := make(chan struct{})
	go func() {
		j.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[Shutdown] All workers finished gracefully")
	case <-time.After(10 * time.Second):
		log.Println("[Shutdown] WARNING: Timeout waiting for workers, forcing shutdown")
	}
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

	j.db.GetDB().Model(job).Select("*").Updates(job)
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

	j.db.GetDB().Model(job).Select("*").Updates(job)
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

	j.db.GetDB().Model(job).Select("*").Updates(job)
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

	if j.ctx == nil || j.ctx.Value("wails-test") == nil {
		j.emitEvent("queue:status", map[string]interface{}{
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
	log.Println("[StopQueueImmediate] Queue stopped immediately - stopping ALL in-progress jobs")

	// Force kill all running processes in ScriptExecutor
	if j.scriptExecutor != nil {
		log.Println("[StopQueueImmediate] Killing all running processes...")
		j.scriptExecutor.KillAllJobs()
	}

	var inProgressJobs []*models.Job
	j.db.GetDB().Where("status = ?", models.JobStatusInProgress).Find(&inProgressJobs)

	log.Printf("[StopQueueImmediate] Found %d in-progress jobs to stop", len(inProgressJobs))

	for _, job := range inProgressJobs {
		log.Printf("[StopQueueImmediate] Stopping job %s", job.ID)

		job.Status = models.JobStatusPending
		job.StartedAt = nil
		job.CompletedAt = nil
		job.Error = ""
		j.db.GetDB().Model(job).Select("*").Updates(job)

		if cancelFunc, exists := j.cancelFuncs[job.ID]; exists {
			log.Printf("[StopQueueImmediate] Calling cancel function for job %s", job.ID)
			cancelFunc()
			delete(j.cancelFuncs, job.ID)
		}

		if j.ctx == nil || j.ctx.Value("wails-test") == nil {
			j.emitEvent("job:update", map[string]interface{}{
				"jobId":       job.ID,
				"status":      job.Status,
				"startedAt":   nil,
				"completedAt": nil,
				"error":       "",
			})
		}
	}

	j.currentJobID = ""
	log.Println("[StopQueueImmediate] Cleared currentJobID and stopped all jobs")

	if j.ctx == nil || j.ctx.Value("wails-test") == nil {
		j.emitEvent("queue:status", map[string]interface{}{
			"paused":        true,
			"stopImmediate": true,
		})
	}

	return nil
}

func (j *JobQueueService) RegisterJobCancelFunc(jobID string, cancelFunc context.CancelFunc) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cancelFuncs[jobID] = cancelFunc
	log.Printf("[RegisterJobCancelFunc] Registered cancel function for job %s", jobID)
}

func (j *JobQueueService) UnregisterJobCancelFunc(jobID string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	delete(j.cancelFuncs, jobID)
	log.Printf("[UnregisterJobCancelFunc] Unregistered cancel function for job %s", jobID)
}

func (j *JobQueueService) ResumeQueue() error {
	j.mu.Lock()
	paused := j.paused
	j.mu.Unlock()

	if !paused {
		return fmt.Errorf("queue is not paused")
	}

	j.mu.Lock()
	j.paused = false
	j.stopImmediate = false
	j.mu.Unlock()

	log.Println("[ResumeQueue] Queue resumed - processing will continue")

	if j.ctx == nil || j.ctx.Value("wails-test") == nil {
		j.emitEvent("queue:status", map[string]interface{}{
			"paused":        false,
			"stopImmediate": false,
		})
	}

	var pendingJobs []*models.Job
	j.db.GetDB().Where("status = ?", models.JobStatusPending).Order("created_at ASC").Find(&pendingJobs)

	log.Printf("[ResumeQueue] Found %d pending jobs to requeue", len(pendingJobs))

	if len(pendingJobs) == 0 {
		log.Println("[ResumeQueue] No pending jobs to requeue")
		return nil
	}

	// Add jobs to memory map first
	j.mu.Lock()
	for _, job := range pendingJobs {
		if _, exists := j.jobs[job.ID]; !exists {
			j.jobs[job.ID] = job
		}
	}
	j.mu.Unlock()

	// Send jobs to queue in a goroutine to avoid blocking while holding lock
	// Use blocking sends so workers can process them when ready
	go func() {
		for _, job := range pendingJobs {
			// Check if we should stop (queue was paused again)
			j.mu.RLock()
			isPaused := j.paused
			j.mu.RUnlock()

			if isPaused {
				log.Println("[ResumeQueue] Queue paused again, stopping requeue")
				return
			}

			// Blocking send - waits for worker to be ready
			j.queue <- job
			log.Printf("[ResumeQueue] Requeued pending job: %s - %s", job.ID, job.Name)
		}
		log.Println("[ResumeQueue] Finished requeueing all pending jobs")
	}()

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

func (j *JobQueueService) ProcessPendingJobs() error {
	log.Println("[ProcessPendingJobs] Manually processing pending jobs")

	var pendingJobs []*models.Job
	if err := j.db.GetDB().Where("status = ?", models.JobStatusPending).Order("created_at ASC").Find(&pendingJobs).Error; err != nil {
		return fmt.Errorf("failed to get pending jobs: %v", err)
	}

	log.Printf("[ProcessPendingJobs] Found %d pending jobs", len(pendingJobs))

	if len(pendingJobs) == 0 {
		log.Println("[ProcessPendingJobs] No pending jobs to process")
		return nil
	}

	// Add jobs to memory map first
	j.mu.Lock()
	for _, job := range pendingJobs {
		if _, exists := j.jobs[job.ID]; !exists {
			j.jobs[job.ID] = job
		}
	}
	j.mu.Unlock()

	// Send jobs to queue in a goroutine using blocking sends
	go func() {
		for _, job := range pendingJobs {
			// Blocking send - waits for worker to be ready
			j.queue <- job
			log.Printf("[ProcessPendingJobs] Queued pending job: %s - %s", job.ID, job.Name)
		}
		log.Println("[ProcessPendingJobs] Finished processing all pending jobs")
	}()

	return nil
}
