package models

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Job represents a background job
type Job struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Type      string    `json:"type"` // "inpaint", "generate", etc.
	Status    JobStatus `json:"status"`
	Result    *Image    `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt int64     `json:"createdAt"`
	UpdatedAt int64     `json:"updatedAt"`
	// Job-specific data
	Data map[string]interface{} `json:"data"`
}

// JobResponse is the response for job-related endpoints
type JobResponse struct {
	Success bool  `json:"success"`
	Job     *Job  `json:"job,omitempty"`
	Error   string `json:"error,omitempty"`
}

// JobStatusResponse is the response for job status endpoint
type JobStatusResponse struct {
	Success bool  `json:"success"`
	Job     *Job  `json:"job,omitempty"`
	Error   string `json:"error,omitempty"`
} 