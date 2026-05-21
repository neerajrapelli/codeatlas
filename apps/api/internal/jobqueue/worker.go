package jobqueue

import (
	"context"
	"log/slog"
	"strconv"
	"time"
)

// IngestRunner executes a claimed ingestion job.
type IngestRunner interface {
	RunJob(ctx context.Context, job *Job) error
}

// StartWorker polls the queue and runs ingestion jobs.
func StartWorker(ctx context.Context, q JobQueue, runner IngestRunner, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("ingestion_worker_stopped")
				return
			case <-ticker.C:
				job, err := q.Dequeue(ctx)
				if err != nil {
					logger.Error("ingestion_dequeue_failed", "error", err)
					continue
				}
				if job == nil {
					continue
				}
				logger.Info("ingestion_job_started", "job_id", job.ID, "repository_id", job.RepositoryID)
				if err := runner.RunJob(ctx, job); err != nil {
					_ = q.Fail(ctx, job.ID, err.Error())
					logger.Error("ingestion_job_failed", "job_id", job.ID, "error", err)
					continue
				}
				_ = q.Complete(ctx, job.ID)
				logger.Info("ingestion_job_complete", "job_id", job.ID, "repository_id", job.RepositoryID)
			}
		}
	}()
}

// ParseRepositoryID helper.
func ParseRepositoryID(job *Job) (int64, error) {
	return strconv.ParseInt(job.RepositoryID, 10, 64)
}
