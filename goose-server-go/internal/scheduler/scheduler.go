package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// ScheduledJob represents a scheduled recipe execution job
type ScheduledJob struct {
	ID               string     `json:"id"`
	Source           string     `json:"source"`                      // Path to recipe file
	Cron             string     `json:"cron"`                        // Cron expression
	LastRun          *time.Time `json:"last_run,omitempty"`          // Last execution time
	CurrentlyRunning bool       `json:"currently_running"`           // Is job currently executing
	Paused           bool       `json:"paused"`                      // Is job paused
	CurrentSessionID *string    `json:"current_session_id,omitempty"` // Current execution session
	ProcessStartTime *time.Time `json:"process_start_time,omitempty"` // Current execution start time
}

// SchedulerError represents scheduler-specific errors
type SchedulerError struct {
	Code    string
	Message string
	Err     error
}

func (e *SchedulerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Error codes
const (
	ErrJobExists     = "JOB_EXISTS"
	ErrJobNotFound   = "JOB_NOT_FOUND"
	ErrStorageError  = "STORAGE_ERROR"
	ErrRecipeError   = "RECIPE_ERROR"
	ErrCronError     = "CRON_ERROR"
	ErrInternalError = "INTERNAL_ERROR"
)

// JobExecutor is a function that executes a scheduled job
type JobExecutor func(ctx context.Context, job *ScheduledJob) (sessionID string, err error)

// Scheduler manages scheduled recipe executions
type Scheduler struct {
	cron           *cron.Cron
	jobs           map[string]*jobEntry
	storagePath    string
	scheduledDir   string
	executor       JobExecutor
	runningTasks   map[string]context.CancelFunc
	mu             sync.RWMutex
}

// jobEntry holds the job and its cron entry ID
type jobEntry struct {
	job     *ScheduledJob
	entryID cron.EntryID
}

// NewScheduler creates a new scheduler
func NewScheduler(dataDir string, executor JobExecutor) (*Scheduler, error) {
	s := &Scheduler{
		cron:           cron.New(cron.WithSeconds()),
		jobs:           make(map[string]*jobEntry),
		storagePath:    filepath.Join(dataDir, "schedules.json"),
		scheduledDir:   filepath.Join(dataDir, "scheduled_recipes"),
		executor:       executor,
		runningTasks:   make(map[string]context.CancelFunc),
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0755); err != nil {
		return nil, &SchedulerError{Code: ErrStorageError, Message: "failed to create storage directory", Err: err}
	}
	if err := os.MkdirAll(s.scheduledDir, 0755); err != nil {
		return nil, &SchedulerError{Code: ErrStorageError, Message: "failed to create scheduled recipes directory", Err: err}
	}

	// Load existing jobs
	if err := s.loadFromStorage(); err != nil {
		// Log but don't fail - storage might not exist yet
		_ = err
	}

	// Start the cron scheduler
	s.cron.Start()

	return s, nil
}

// AddJob adds a new scheduled job
func (s *Scheduler) AddJob(job *ScheduledJob, copyRecipe bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if job already exists
	if _, exists := s.jobs[job.ID]; exists {
		return &SchedulerError{Code: ErrJobExists, Message: fmt.Sprintf("job %s already exists", job.ID)}
	}

	// Validate cron expression
	cronExpr := normalizeCron(job.Cron)
	// Use parser that accepts 6-field format (with seconds) since we use cron.WithSeconds()
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(cronExpr); err != nil {
		return &SchedulerError{Code: ErrCronError, Message: "invalid cron expression", Err: err}
	}

	// Copy recipe file if requested
	if copyRecipe {
		destPath, err := s.copyRecipeFile(job.Source)
		if err != nil {
			return err
		}
		job.Source = destPath
	}

	// Verify recipe file exists
	if _, err := os.Stat(job.Source); os.IsNotExist(err) {
		return &SchedulerError{Code: ErrRecipeError, Message: "recipe file not found", Err: err}
	}

	// Create cron entry
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeJob(job.ID)
	})
	if err != nil {
		return &SchedulerError{Code: ErrCronError, Message: "failed to schedule job", Err: err}
	}

	s.jobs[job.ID] = &jobEntry{
		job:     job,
		entryID: entryID,
	}

	return s.persist()
}

// RemoveJob removes a scheduled job
func (s *Scheduler) RemoveJob(id string, removeRecipe bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.jobs[id]
	if !exists {
		return &SchedulerError{Code: ErrJobNotFound, Message: fmt.Sprintf("job %s not found", id)}
	}

	// Cancel if running
	if cancel, running := s.runningTasks[id]; running {
		cancel()
		delete(s.runningTasks, id)
	}

	// Remove from cron
	s.cron.Remove(entry.entryID)

	// Remove recipe file if in scheduled directory
	if removeRecipe {
		if filepath.Dir(entry.job.Source) == s.scheduledDir {
			os.Remove(entry.job.Source)
		}
	}

	delete(s.jobs, id)

	return s.persist()
}

// UpdateCron updates the cron expression for a job
func (s *Scheduler) UpdateCron(id string, newCron string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.jobs[id]
	if !exists {
		return &SchedulerError{Code: ErrJobNotFound, Message: fmt.Sprintf("job %s not found", id)}
	}

	if entry.job.CurrentlyRunning {
		return &SchedulerError{Code: ErrInternalError, Message: "cannot update running job"}
	}

	// Validate new cron expression
	cronExpr := normalizeCron(newCron)
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(cronExpr); err != nil {
		return &SchedulerError{Code: ErrCronError, Message: "invalid cron expression", Err: err}
	}

	// Remove old cron entry
	s.cron.Remove(entry.entryID)

	// Create new cron entry
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeJob(id)
	})
	if err != nil {
		return &SchedulerError{Code: ErrCronError, Message: "failed to reschedule job", Err: err}
	}

	entry.entryID = entryID
	entry.job.Cron = newCron

	return s.persist()
}

// ListJobs returns all scheduled jobs
func (s *Scheduler) ListJobs() []*ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, entry := range s.jobs {
		jobs = append(jobs, entry.job)
	}
	return jobs
}

// GetJob returns a specific job
func (s *Scheduler) GetJob(id string) (*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.jobs[id]
	if !exists {
		return nil, &SchedulerError{Code: ErrJobNotFound, Message: fmt.Sprintf("job %s not found", id)}
	}
	return entry.job, nil
}

// PauseJob pauses a scheduled job
func (s *Scheduler) PauseJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.jobs[id]
	if !exists {
		return &SchedulerError{Code: ErrJobNotFound, Message: fmt.Sprintf("job %s not found", id)}
	}

	if entry.job.CurrentlyRunning {
		return &SchedulerError{Code: ErrInternalError, Message: "cannot pause running job"}
	}

	entry.job.Paused = true
	return s.persist()
}

// UnpauseJob resumes a paused job
func (s *Scheduler) UnpauseJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.jobs[id]
	if !exists {
		return &SchedulerError{Code: ErrJobNotFound, Message: fmt.Sprintf("job %s not found", id)}
	}

	entry.job.Paused = false
	return s.persist()
}

// RunNow triggers immediate execution of a job
func (s *Scheduler) RunNow(id string) (string, error) {
	s.mu.RLock()
	entry, exists := s.jobs[id]
	s.mu.RUnlock()

	if !exists {
		return "", &SchedulerError{Code: ErrJobNotFound, Message: fmt.Sprintf("job %s not found", id)}
	}

	if entry.job.CurrentlyRunning {
		return "", &SchedulerError{Code: ErrInternalError, Message: "job is already running"}
	}

	// Execute synchronously
	sessionID, err := s.doExecuteJob(id)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

// KillRunningJob cancels a running job
func (s *Scheduler) KillRunningJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancel, running := s.runningTasks[id]
	if !running {
		return &SchedulerError{Code: ErrInternalError, Message: "job is not running"}
	}

	cancel()
	delete(s.runningTasks, id)

	// Update job state
	if entry, exists := s.jobs[id]; exists {
		entry.job.CurrentlyRunning = false
		entry.job.CurrentSessionID = nil
		entry.job.ProcessStartTime = nil
	}

	return s.persist()
}

// GetRunningJobInfo returns info about a running job
func (s *Scheduler) GetRunningJobInfo(id string) (sessionID *string, startTime *time.Time, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.jobs[id]
	if !exists {
		return nil, nil, &SchedulerError{Code: ErrJobNotFound, Message: fmt.Sprintf("job %s not found", id)}
	}

	if !entry.job.CurrentlyRunning {
		return nil, nil, nil
	}

	return entry.job.CurrentSessionID, entry.job.ProcessStartTime, nil
}

// executeJob is called by cron scheduler
func (s *Scheduler) executeJob(id string) {
	s.mu.RLock()
	entry, exists := s.jobs[id]
	s.mu.RUnlock()

	if !exists {
		return
	}

	if entry.job.Paused || entry.job.CurrentlyRunning {
		return
	}

	_, _ = s.doExecuteJob(id)
}

// doExecuteJob performs the actual job execution
func (s *Scheduler) doExecuteJob(id string) (string, error) {
	s.mu.Lock()
	entry, exists := s.jobs[id]
	if !exists {
		s.mu.Unlock()
		return "", &SchedulerError{Code: ErrJobNotFound, Message: fmt.Sprintf("job %s not found", id)}
	}

	// Mark as running
	now := time.Now()
	entry.job.CurrentlyRunning = true
	entry.job.ProcessStartTime = &now

	// Create cancellation context
	ctx, cancel := context.WithCancel(context.Background())
	s.runningTasks[id] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		entry.job.CurrentlyRunning = false
		entry.job.LastRun = &now
		entry.job.ProcessStartTime = nil
		delete(s.runningTasks, id)
		s.persist()
		s.mu.Unlock()
	}()

	// Execute the job
	if s.executor == nil {
		return "", &SchedulerError{Code: ErrInternalError, Message: "no executor configured"}
	}

	sessionID, err := s.executor(ctx, entry.job)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	entry.job.CurrentSessionID = &sessionID
	s.mu.Unlock()

	return sessionID, nil
}

// copyRecipeFile copies a recipe to the scheduled recipes directory
func (s *Scheduler) copyRecipeFile(source string) (string, error) {
	content, err := os.ReadFile(source)
	if err != nil {
		return "", &SchedulerError{Code: ErrRecipeError, Message: "failed to read recipe file", Err: err}
	}

	filename := filepath.Base(source)
	destPath := filepath.Join(s.scheduledDir, filename)

	// Handle naming conflicts
	ext := filepath.Ext(destPath)
	nameWithoutExt := destPath[:len(destPath)-len(ext)]
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		destPath = fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		counter++
	}

	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return "", &SchedulerError{Code: ErrStorageError, Message: "failed to copy recipe file", Err: err}
	}

	return destPath, nil
}

// loadFromStorage loads jobs from the storage file
func (s *Scheduler) loadFromStorage() error {
	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var jobs []*ScheduledJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return err
	}

	for _, job := range jobs {
		// Verify recipe file exists
		if _, err := os.Stat(job.Source); os.IsNotExist(err) {
			// Skip jobs with missing recipe files
			continue
		}

		// Reset running state on load
		job.CurrentlyRunning = false
		job.CurrentSessionID = nil
		job.ProcessStartTime = nil

		// Create cron entry
		cronExpr := normalizeCron(job.Cron)
		entryID, err := s.cron.AddFunc(cronExpr, func() {
			s.executeJob(job.ID)
		})
		if err != nil {
			continue
		}

		s.jobs[job.ID] = &jobEntry{
			job:     job,
			entryID: entryID,
		}
	}

	return nil
}

// persist saves jobs to the storage file
func (s *Scheduler) persist() error {
	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, entry := range s.jobs {
		jobs = append(jobs, entry.job)
	}

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return &SchedulerError{Code: ErrStorageError, Message: "failed to marshal jobs", Err: err}
	}

	if err := os.WriteFile(s.storagePath, data, 0644); err != nil {
		return &SchedulerError{Code: ErrStorageError, Message: "failed to write storage file", Err: err}
	}

	return nil
}

// normalizeCron converts 5-field cron to 6-field by prepending "0"
func normalizeCron(expr string) string {
	fields := len(splitCronFields(expr))
	if fields == 5 {
		return "0 " + expr
	}
	return expr
}

// splitCronFields splits a cron expression into fields
func splitCronFields(expr string) []string {
	var fields []string
	var current string
	for _, c := range expr {
		if c == ' ' || c == '\t' {
			if current != "" {
				fields = append(fields, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		fields = append(fields, current)
	}
	return fields
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel all running tasks
	for id, cancel := range s.runningTasks {
		cancel()
		delete(s.runningTasks, id)
	}

	// Stop cron scheduler
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// ScheduleRecipe schedules or updates a recipe schedule
func (s *Scheduler) ScheduleRecipe(recipePath string, cronSchedule *string) error {
	// Generate ID from recipe path
	id := filepath.Base(recipePath)

	if cronSchedule == nil || *cronSchedule == "" {
		// Remove schedule
		return s.RemoveJob(id, false)
	}

	// Check if job exists
	if _, err := s.GetJob(id); err == nil {
		// Update existing
		return s.UpdateCron(id, *cronSchedule)
	}

	// Create new job
	job := &ScheduledJob{
		ID:     id,
		Source: recipePath,
		Cron:   *cronSchedule,
	}

	return s.AddJob(job, true)
}
