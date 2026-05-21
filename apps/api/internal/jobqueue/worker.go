package jobqueue

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// IngestRunner executes a claimed ingestion job.
type IngestRunner interface {
	RunJob(ctx context.Context, job *Job) error
}

// StartWorker polls the queue and runs ingestion jobs with bounded concurrency.
func StartWorker(ctx context.Context, q JobQueue, runner IngestRunner, concurrency int, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	if concurrency < 1 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				wg.Wait()
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
				sem <- struct{}{}
				wg.Add(1)
				go func(j *Job) {
					defer wg.Done()
					defer func() { <-sem }()
					logger.Info("ingestion_job_started", "job_id", j.ID, "repository_id", j.RepositoryID)
					if err := runner.RunJob(ctx, j); err != nil {
						_ = q.Fail(ctx, j.ID, err.Error())
						logger.Error("ingestion_job_failed", "job_id", j.ID, "error", err)
						return
					}
					_ = q.Complete(ctx, j.ID)
					logger.Info("ingestion_job_complete", "job_id", j.ID, "repository_id", j.RepositoryID)
				}(job)
			}
		}
	}()
}

// ParseRepositoryID helper.
func ParseRepositoryID(job *Job) (int64, error) {
	return strconv.ParseInt(job.RepositoryID, 10, 64)
}
