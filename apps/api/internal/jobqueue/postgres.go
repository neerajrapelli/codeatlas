package jobqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"codeatlas/apps/api/internal/ingestprogress"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresQueue struct {
	pool *pgxpool.Pool
}

func NewPostgresQueue(pool *pgxpool.Pool) *PostgresQueue {
	return &PostgresQueue{pool: pool}
}

func (q *PostgresQueue) Enqueue(ctx context.Context, repositoryID string, phase int, metadata map[string]any) (string, error) {
	repoID, err := strconv.ParseInt(repositoryID, 10, 64)
	if err != nil || repoID <= 0 {
		return "", fmt.Errorf("invalid repository id")
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metaBytes, _ := json.Marshal(metadata)
	initial := ingestprogress.NewQueuedEvent(repoID, phase)
	progBytes, _ := json.Marshal(initial)

	var jobID string
	err = q.pool.QueryRow(ctx, `
		INSERT INTO ingestion_jobs (repository_id, phase, status, current_step, progress_json, metadata)
		VALUES ($1, $2, 'queued', $3, $4::jsonb, $5::jsonb)
		RETURNING id::text
	`, repoID, phase, ingestprogress.StepCloneRepository, progBytes, metaBytes).Scan(&jobID)
	return jobID, err
}

func (q *PostgresQueue) Dequeue(ctx context.Context) (*Job, error) {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var job Job
	var progJSON []byte
	var metaJSON []byte
	var startedAt *time.Time
	var completedAt *time.Time
	var errMsg *string

	err = tx.QueryRow(ctx, `
		SELECT id::text, repository_id::text, phase, status::text, current_step,
		       progress_json, metadata, error_msg, queued_at, started_at, completed_at
		FROM ingestion_jobs
		WHERE status = 'queued'
		ORDER BY queued_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&job.ID, &job.RepositoryID, &job.Phase, &job.Status, &job.CurrentStep,
		&progJSON, &metaJSON, &errMsg, &job.QueuedAt, &startedAt, &completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if errMsg != nil {
		job.ErrorMsg = *errMsg
	}
	job.StartedAt = startedAt
	job.CompletedAt = completedAt
	_ = json.Unmarshal(progJSON, &job.Progress)
	_ = json.Unmarshal(metaJSON, &job.Metadata)

	_, err = tx.Exec(ctx, `
		UPDATE ingestion_jobs
		SET status = 'running', started_at = COALESCE(started_at, now()), updated_at = now()
		WHERE id = $1::uuid
	`, job.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	job.Status = "running"
	return &job, nil
}

func (q *PostgresQueue) UpdateProgress(ctx context.Context, jobID string, event ingestprogress.StreamEvent) error {
	progBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = q.pool.Exec(ctx, `
		UPDATE ingestion_jobs
		SET progress_json = $2::jsonb,
		    current_step = $3,
		    updated_at = now()
		WHERE id = $1::uuid
	`, jobID, progBytes, event.CurrentStep)
	return err
}

func (q *PostgresQueue) Complete(ctx context.Context, jobID string) error {
	ev := ingestprogress.StreamEvent{Status: ingestprogress.StatusComplete}
	_ = q.UpdateProgress(ctx, jobID, ev)
	_, err := q.pool.Exec(ctx, `
		UPDATE ingestion_jobs
		SET status = 'complete', completed_at = now(), updated_at = now()
		WHERE id = $1::uuid
	`, jobID)
	return err
}

func (q *PostgresQueue) Fail(ctx context.Context, jobID string, errMsg string) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE ingestion_jobs
		SET status = 'failed', error_msg = $2, completed_at = now(), updated_at = now()
		WHERE id = $1::uuid
	`, jobID, errMsg)
	return err
}

func (q *PostgresQueue) GetStatus(ctx context.Context, repositoryID string) (*Job, error) {
	repoID, err := strconv.ParseInt(repositoryID, 10, 64)
	if err != nil {
		return nil, err
	}
	return q.GetLatestForRepository(ctx, repoID)
}

func (q *PostgresQueue) GetLatestForRepository(ctx context.Context, repositoryID int64) (*Job, error) {
	var job Job
	var progJSON []byte
	var metaJSON []byte
	var errMsg *string
	var startedAt, completedAt *time.Time

	err := q.pool.QueryRow(ctx, `
		SELECT id::text, repository_id::text, phase, status::text, current_step,
		       progress_json, metadata, error_msg, queued_at, started_at, completed_at
		FROM ingestion_jobs
		WHERE repository_id = $1
		ORDER BY queued_at DESC
		LIMIT 1
	`, repositoryID).Scan(&job.ID, &job.RepositoryID, &job.Phase, &job.Status, &job.CurrentStep,
		&progJSON, &metaJSON, &errMsg, &job.QueuedAt, &startedAt, &completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if errMsg != nil {
		job.ErrorMsg = *errMsg
	}
	job.StartedAt = startedAt
	job.CompletedAt = completedAt
	_ = json.Unmarshal(progJSON, &job.Progress)
	_ = json.Unmarshal(metaJSON, &job.Metadata)
	return &job, nil
}
