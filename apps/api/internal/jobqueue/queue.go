package jobqueue

import (
	"context"
	"time"

	"codeatlas/apps/api/internal/ingestprogress"
)

// JobQueue persists and claims ingestion work.
type JobQueue interface {
	Enqueue(ctx context.Context, repositoryID string, phase int, metadata map[string]any) (string, error)
	Dequeue(ctx context.Context) (*Job, error)
	UpdateProgress(ctx context.Context, jobID string, event ingestprogress.StreamEvent) error
	Complete(ctx context.Context, jobID string) error
	Fail(ctx context.Context, jobID string, errMsg string) error
	GetStatus(ctx context.Context, repositoryID string) (*Job, error)
	GetLatestForRepository(ctx context.Context, repositoryID int64) (*Job, error)
	GetByID(ctx context.Context, jobID string) (*Job, error)
}

type Job struct {
	ID           string
	RepositoryID string
	Phase        int
	Status       string
	CurrentStep  string
	Progress     ingestprogress.StreamEvent
	Metadata     map[string]any
	ErrorMsg     string
	QueuedAt     time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}
