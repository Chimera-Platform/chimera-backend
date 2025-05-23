package services

import (
	"backend/models"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobManager handles background jobs
type JobManager struct {
	jobs map[string]*models.Job
	mu   sync.RWMutex
}

// NewJobManager creates a new job manager
func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*models.Job),
	}
}

// CreateJob creates a new job
func (m *JobManager) CreateJob(userID, jobType string, data map[string]interface{}) *models.Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	job := &models.Job{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      jobType,
		Status:    models.JobStatusPending,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Data:      data,
	}

	m.jobs[job.ID] = job
	log.Printf("Created job %s of type %s for user %s", job.ID, jobType, userID)
	return job
}

// GetJob gets a job by ID
func (m *JobManager) GetJob(jobID string) (*models.Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[jobID]
	return job, exists
}

// UpdateJobStatus updates a job's status
func (m *JobManager) UpdateJobStatus(jobID string, status models.JobStatus) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return false
	}

	job.Status = status
	job.UpdatedAt = time.Now().Unix()
	log.Printf("Updated job %s status to %s", jobID, status)
	return true
}

// CompleteJob marks a job as completed with a result
func (m *JobManager) CompleteJob(jobID string, result *models.Image) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return false
	}

	job.Status = models.JobStatusCompleted
	job.Result = result
	job.UpdatedAt = time.Now().Unix()
	log.Printf("Completed job %s with result %s", jobID, result.URL)
	return true
}

// FailJob marks a job as failed with an error
func (m *JobManager) FailJob(jobID string, errorMsg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return false
	}

	job.Status = models.JobStatusFailed
	job.Error = errorMsg
	job.UpdatedAt = time.Now().Unix()
	log.Printf("Failed job %s with error: %s", jobID, errorMsg)
	return true
}

// CleanupOldJobs removes jobs older than the specified duration
func (m *JobManager) CleanupOldJobs(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	cutoff := now - int64(maxAge.Seconds())
	count := 0

	for id, job := range m.jobs {
		if job.CreatedAt < cutoff {
			delete(m.jobs, id)
			count++
		}
	}

	if count > 0 {
		log.Printf("Cleaned up %d old jobs", count)
	}
	return count
}

// StartCleanupRoutine starts a routine to periodically clean up old jobs
func (m *JobManager) StartCleanupRoutine(interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			m.CleanupOldJobs(maxAge)
		}
	}()
	log.Printf("Started job cleanup routine (interval: %v, max age: %v)", interval, maxAge)
} 