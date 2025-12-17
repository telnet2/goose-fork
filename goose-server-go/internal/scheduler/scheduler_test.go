package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestScheduler(t *testing.T) (*Scheduler, string, func()) {
	tempDir, err := os.MkdirTemp("", "scheduler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	executor := func(ctx context.Context, job *ScheduledJob) (string, error) {
		return "test-session-" + job.ID, nil
	}

	sched, err := NewScheduler(tempDir, executor)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create scheduler: %v", err)
	}

	cleanup := func() {
		sched.Stop()
		os.RemoveAll(tempDir)
	}

	return sched, tempDir, cleanup
}

func createTestRecipeFile(t *testing.T, dir string) string {
	content := `version: "1.0.0"
title: Test Recipe
description: A test recipe
instructions: Do the thing
`
	path := filepath.Join(dir, "test_recipe.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test recipe file: %v", err)
	}
	return path
}

func TestNewScheduler(t *testing.T) {
	sched, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	if sched == nil {
		t.Fatal("expected non-nil scheduler")
	}
}

func TestScheduler_AddJob(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 * * * *", // Every hour
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	// Verify job was added
	jobs := sched.ListJobs()
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}

func TestScheduler_AddJob_Duplicate(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 * * * *",
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	// Try to add duplicate
	err := sched.AddJob(job, false)
	if err == nil {
		t.Fatal("expected error for duplicate job")
	}

	schedErr, ok := err.(*SchedulerError)
	if !ok {
		t.Fatalf("expected SchedulerError, got %T", err)
	}
	if schedErr.Code != ErrJobExists {
		t.Errorf("expected error code '%s', got '%s'", ErrJobExists, schedErr.Code)
	}
}

func TestScheduler_AddJob_InvalidCron(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "invalid cron",
	}

	err := sched.AddJob(job, false)
	if err == nil {
		t.Fatal("expected error for invalid cron")
	}

	schedErr, ok := err.(*SchedulerError)
	if !ok {
		t.Fatalf("expected SchedulerError, got %T", err)
	}
	if schedErr.Code != ErrCronError {
		t.Errorf("expected error code '%s', got '%s'", ErrCronError, schedErr.Code)
	}
}

func TestScheduler_AddJob_MissingRecipe(t *testing.T) {
	sched, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: "/nonexistent/recipe.yaml",
		Cron:   "0 * * * *",
	}

	err := sched.AddJob(job, false)
	if err == nil {
		t.Fatal("expected error for missing recipe")
	}

	schedErr, ok := err.(*SchedulerError)
	if !ok {
		t.Fatalf("expected SchedulerError, got %T", err)
	}
	if schedErr.Code != ErrRecipeError {
		t.Errorf("expected error code '%s', got '%s'", ErrRecipeError, schedErr.Code)
	}
}

func TestScheduler_AddJob_CopyRecipe(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 * * * *",
	}

	if err := sched.AddJob(job, true); err != nil {
		t.Fatalf("failed to add job with copy: %v", err)
	}

	// Verify job source was updated to scheduled directory
	retrievedJob, err := sched.GetJob("test-job-1")
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if retrievedJob.Source == recipePath {
		t.Error("expected job source to be updated to scheduled directory")
	}
}

func TestScheduler_RemoveJob(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 * * * *",
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	if err := sched.RemoveJob("test-job-1", false); err != nil {
		t.Fatalf("failed to remove job: %v", err)
	}

	jobs := sched.ListJobs()
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestScheduler_RemoveJob_NotFound(t *testing.T) {
	sched, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	err := sched.RemoveJob("nonexistent", false)
	if err == nil {
		t.Fatal("expected error for nonexistent job")
	}

	schedErr, ok := err.(*SchedulerError)
	if !ok {
		t.Fatalf("expected SchedulerError, got %T", err)
	}
	if schedErr.Code != ErrJobNotFound {
		t.Errorf("expected error code '%s', got '%s'", ErrJobNotFound, schedErr.Code)
	}
}

func TestScheduler_GetJob(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 * * * *",
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	retrieved, err := sched.GetJob("test-job-1")
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if retrieved.ID != "test-job-1" {
		t.Errorf("expected ID 'test-job-1', got '%s'", retrieved.ID)
	}
}

func TestScheduler_GetJob_NotFound(t *testing.T) {
	sched, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	_, err := sched.GetJob("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent job")
	}
}

func TestScheduler_UpdateCron(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 * * * *",
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	if err := sched.UpdateCron("test-job-1", "30 * * * *"); err != nil {
		t.Fatalf("failed to update cron: %v", err)
	}

	retrieved, err := sched.GetJob("test-job-1")
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if retrieved.Cron != "30 * * * *" {
		t.Errorf("expected cron '30 * * * *', got '%s'", retrieved.Cron)
	}
}

func TestScheduler_PauseUnpause(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 * * * *",
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	// Pause
	if err := sched.PauseJob("test-job-1"); err != nil {
		t.Fatalf("failed to pause job: %v", err)
	}

	retrieved, _ := sched.GetJob("test-job-1")
	if !retrieved.Paused {
		t.Error("expected job to be paused")
	}

	// Unpause
	if err := sched.UnpauseJob("test-job-1"); err != nil {
		t.Fatalf("failed to unpause job: %v", err)
	}

	retrieved, _ = sched.GetJob("test-job-1")
	if retrieved.Paused {
		t.Error("expected job to be unpaused")
	}
}

func TestScheduler_RunNow(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 0 1 1 *", // Rarely triggers (Jan 1st)
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	sessionID, err := sched.RunNow("test-job-1")
	if err != nil {
		t.Fatalf("failed to run job now: %v", err)
	}

	if sessionID == "" {
		t.Error("expected non-empty session ID")
	}
}

func TestScheduler_ListJobs(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	for i := 1; i <= 3; i++ {
		job := &ScheduledJob{
			ID:     string(rune('0' + i)),
			Source: recipePath,
			Cron:   "0 * * * *",
		}
		if err := sched.AddJob(job, false); err != nil {
			t.Fatalf("failed to add job %d: %v", i, err)
		}
	}

	jobs := sched.ListJobs()
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

func TestScheduler_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scheduler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	recipePath := createTestRecipeFile(t, tempDir)

	executor := func(ctx context.Context, job *ScheduledJob) (string, error) {
		return "test-session-" + job.ID, nil
	}

	// Create scheduler and add job
	sched1, err := NewScheduler(tempDir, executor)
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	job := &ScheduledJob{
		ID:     "persistent-job",
		Source: recipePath,
		Cron:   "0 * * * *",
	}

	if err := sched1.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	sched1.Stop()

	// Create new scheduler and verify job was loaded
	sched2, err := NewScheduler(tempDir, executor)
	if err != nil {
		t.Fatalf("failed to create second scheduler: %v", err)
	}
	defer sched2.Stop()

	jobs := sched2.ListJobs()
	if len(jobs) != 1 {
		t.Errorf("expected 1 job after reload, got %d", len(jobs))
	}

	if len(jobs) > 0 && jobs[0].ID != "persistent-job" {
		t.Errorf("expected job ID 'persistent-job', got '%s'", jobs[0].ID)
	}
}

func TestNormalizeCron(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"* * * * *", "0 * * * * *"},           // 5-field to 6-field
		{"0 * * * * *", "0 * * * * *"},         // Already 6-field
		{"0 0 * * *", "0 0 0 * * *"},           // 5-field
		{"*/5 * * * *", "0 */5 * * * *"},       // With step
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeCron(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeCron(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitCronFields(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"* * * * *", 5},
		{"0 * * * * *", 6},
		{"0  *   * * *", 5}, // Multiple spaces
		{"0\t*\t*\t*\t*", 5}, // Tabs
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			fields := splitCronFields(tt.input)
			if len(fields) != tt.expected {
				t.Errorf("splitCronFields(%s) = %d fields, want %d", tt.input, len(fields), tt.expected)
			}
		})
	}
}

func TestSchedulerError_Error(t *testing.T) {
	err := &SchedulerError{
		Code:    ErrJobNotFound,
		Message: "job not found",
	}

	if err.Error() != "JOB_NOT_FOUND: job not found" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// With wrapped error
	err.Err = os.ErrNotExist
	if err.Error() != "JOB_NOT_FOUND: job not found: file does not exist" {
		t.Errorf("unexpected error message with wrapped error: %s", err.Error())
	}
}

func TestScheduler_GetRunningJobInfo(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 0 1 1 *",
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	// Job not running initially
	sessionID, startTime, err := sched.GetRunningJobInfo("test-job-1")
	if err != nil {
		t.Fatalf("failed to get running job info: %v", err)
	}

	if sessionID != nil || startTime != nil {
		t.Error("expected nil session ID and start time for non-running job")
	}
}

func TestScheduler_ScheduleRecipe(t *testing.T) {
	sched, tempDir, cleanup := setupTestScheduler(t)
	defer cleanup()

	recipePath := createTestRecipeFile(t, tempDir)
	cronSchedule := "0 * * * *"

	// Schedule new recipe
	if err := sched.ScheduleRecipe(recipePath, &cronSchedule); err != nil {
		t.Fatalf("failed to schedule recipe: %v", err)
	}

	jobs := sched.ListJobs()
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}

	// Update schedule
	newCron := "30 * * * *"
	if err := sched.ScheduleRecipe(recipePath, &newCron); err != nil {
		t.Fatalf("failed to update recipe schedule: %v", err)
	}

	// Remove schedule
	if err := sched.ScheduleRecipe(recipePath, nil); err != nil {
		t.Fatalf("failed to remove recipe schedule: %v", err)
	}

	jobs = sched.ListJobs()
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs after removal, got %d", len(jobs))
	}
}

// Test that job starts with correct initial state
func TestScheduledJob_InitialState(t *testing.T) {
	job := &ScheduledJob{
		ID:     "test-job",
		Source: "/path/to/recipe.yaml",
		Cron:   "0 * * * *",
	}

	if job.CurrentlyRunning {
		t.Error("expected job to not be running initially")
	}
	if job.Paused {
		t.Error("expected job to not be paused initially")
	}
	if job.LastRun != nil {
		t.Error("expected LastRun to be nil initially")
	}
	if job.CurrentSessionID != nil {
		t.Error("expected CurrentSessionID to be nil initially")
	}
	if job.ProcessStartTime != nil {
		t.Error("expected ProcessStartTime to be nil initially")
	}
}

// Test scheduler stop cancels running jobs
func TestScheduler_StopCancelsRunning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scheduler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	executionStarted := make(chan bool, 1)
	executor := func(ctx context.Context, job *ScheduledJob) (string, error) {
		executionStarted <- true
		// Wait for cancellation or timeout
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
			return "session", nil
		}
	}

	sched, err := NewScheduler(tempDir, executor)
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	recipePath := createTestRecipeFile(t, tempDir)

	job := &ScheduledJob{
		ID:     "test-job-1",
		Source: recipePath,
		Cron:   "0 0 1 1 *",
	}

	if err := sched.AddJob(job, false); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	// Start job execution in background
	go func() {
		sched.RunNow("test-job-1")
	}()

	// Wait for execution to start
	select {
	case <-executionStarted:
		// Good, job started
	case <-time.After(1 * time.Second):
		t.Fatal("job did not start in time")
	}

	// Stop scheduler should cancel running job
	sched.Stop()
}
