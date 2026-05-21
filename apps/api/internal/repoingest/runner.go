package repoingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codeatlas/apps/api/internal/indexer"
	"codeatlas/apps/api/internal/ingestprogress"
	"codeatlas/apps/api/internal/jobqueue"
)

// Runner executes queued ingestion jobs.
type Runner struct {
	svc *Service
}

func NewRunner(svc *Service) *Runner {
	return &Runner{svc: svc}
}

func (r *Runner) RunJob(ctx context.Context, job *jobqueue.Job) error {
	repoID, err := jobqueue.ParseRepositoryID(job)
	if err != nil {
		return err
	}
	repo, err := r.svc.store.GetByID(ctx, repoID)
	if err != nil {
		return err
	}

	meta := decodeJobMetadata(job.Metadata)

	if meta.Reindex {
		return r.svc.runReindexJob(ctx, job, repo, meta)
	}
	return r.svc.runIngestJob(ctx, job, repo, meta)
}

type jobMetadata struct {
	SourceType   SourceType `json:"sourceType"`
	SourceURL    string     `json:"sourceUrl"`
	Branch       string     `json:"branch"`
	DisplayName  string     `json:"displayName"`
	ZIPPath      string     `json:"zipPath,omitempty"`
	Reindex      bool       `json:"reindex,omitempty"`
	SkipPrepare  bool       `json:"skipPrepare,omitempty"`
}

func decodeJobMetadata(raw map[string]any) jobMetadata {
	if raw == nil {
		return jobMetadata{}
	}
	b, _ := json.Marshal(raw)
	var meta jobMetadata
	_ = json.Unmarshal(b, &meta)
	return meta
}

func (s *Service) runIngestJob(ctx context.Context, job *jobqueue.Job, repo Repository, meta jobMetadata) error {
	req := CreateRequest{
		SourceType:  meta.SourceType,
		SourceURL:   meta.SourceURL,
		Branch:      meta.Branch,
		DisplayName: meta.DisplayName,
		ZIPPath:     meta.ZIPPath,
	}
	src, ok := s.sources[req.SourceType]
	if !ok {
		return fmt.Errorf("unsupported source type: %s", req.SourceType)
	}
	return s.executeIngestion(ctx, job, repo, req, src, false)
}

func (s *Service) runReindexJob(ctx context.Context, job *jobqueue.Job, repo Repository, meta jobMetadata) error {
	src, ok := s.sources[repo.SourceType]
	if !ok {
		return fmt.Errorf("unsupported source type: %s", repo.SourceType)
	}
	req := CreateRequest{
		SourceType:  repo.SourceType,
		SourceURL:   repo.SourceURL,
		Branch:      repo.Branch,
		DisplayName: repo.Name,
	}
	return s.executeIngestion(ctx, job, repo, req, src, meta.SkipPrepare)
}

func (s *Service) executeIngestion(
	ctx context.Context,
	job *jobqueue.Job,
	repo Repository,
	req CreateRequest,
	src Source,
	skipPrepare bool,
) error {
	durations := ingestprogress.NewStepDurations()
	publish := func(repoStatus Status, pct float64, files, symbols, edges int) {
		s.publishJobProgress(ctx, job, repo.ID, repoStatus, pct, files, symbols, edges, durations)
	}

	stepStatus := StatusCloning
	if req.SourceType == SourceZIP {
		stepStatus = StatusExtracting
	}
	if err := s.store.UpdateStatus(ctx, repo.ID, stepStatus, ""); err != nil {
		return err
	}
	currentStep := ingestprogress.RepoStatusToCurrentStep(string(stepStatus))
	durations.Start(currentStep)
	publish(stepStatus, 5, 0, 0, 0)

	if !skipPrepare {
		stageStart := time.Now()
		if err := src.Prepare(ctx, req, repo.WorkspacePath); err != nil {
			_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
			s.publishFailed(ctx, job, repo.ID, currentStep, durations, err.Error())
			_ = os.RemoveAll(repo.WorkspacePath)
			return err
		}
		durations.Complete(currentStep)
		prepMs := time.Since(stageStart).Milliseconds()
		stageLabel := "cloneDurationMs"
		if req.SourceType == SourceZIP {
			stageLabel = "extractDurationMs"
		}
		_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
			Stage:           stepStatus,
			ProgressPercent: 20,
			StageMetadata:   map[string]any{stageLabel: prepMs},
		})
		indexStep := ingestprogress.StepIndexWorkspace
		durations.Start(indexStep)
		durations.Complete(indexStep)
		publish(StatusParsing, 22, 0, 0, 0)
	} else {
		_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
			Stage:           stepStatus,
			ProgressPercent: 20,
			StageMetadata:   map[string]any{"reindexReuseWorkspace": true},
		})
		publish(StatusParsing, 22, 0, 0, 0)
	}

	parseStep := ingestprogress.StepParseSources
	durations.Start(parseStep)

	lastProgress := time.Now()
	report := func(evt indexer.ProgressEvent) {
		now := time.Now()
		if now.Sub(lastProgress) < 500*time.Millisecond && evt.Progress < 100 {
			return
		}
		lastProgress = now
		stage := Status(evt.Stage)
		if stage == "" {
			stage = StatusParsing
		}
		progress := 20 + evt.Progress*0.75
		if stage == StatusGeneratingEmbeddings {
			if evt.Progress > 100 {
				progress = 85
			}
		}
		_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
			Stage:             stage,
			ProgressPercent:   progress,
			FilesIndexed:      evt.Files,
			SymbolsIndexed:    evt.Symbols,
			EdgesIndexed:      evt.Edges,
			EmbeddingsIndexed: evt.Embeddings,
			StageMetadata:     evt.Metadata,
		})
		publish(stage, progress, evt.Files, evt.Symbols, evt.Edges)
	}

	if err := s.store.UpdateStatus(ctx, repo.ID, StatusParsing, ""); err != nil {
		return err
	}

	res, err := s.indexer.Run(ctx, indexer.Request{
		RepositoryPath: repo.WorkspacePath,
		RepositoryName: repo.Name,
		OnProgress:     report,
	})
	durations.Complete(parseStep)
	durations.Complete(ingestprogress.StepBuildDependencyGraph)
	durations.Complete(ingestprogress.StepSemanticEmbeddings)

	if err != nil {
		_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
		s.publishFailed(ctx, job, repo.ID, parseStep, durations, err.Error())
		_ = os.RemoveAll(repo.WorkspacePath)
		return err
	}

	_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
		Stage:             StatusReady,
		ProgressPercent:   100,
		FilesIndexed:      maxInt(res.Files, 1),
		SymbolsIndexed:    res.Symbols,
		EdgesIndexed:      res.FileDependencies,
		EmbeddingsIndexed: res.Embeddings,
	})
	if err := s.store.UpdateStatus(ctx, repo.ID, StatusReady, ""); err != nil {
		return err
	}
	s.publishComplete(ctx, job, repo.ID, durations, res.Files, res.Symbols, res.FileDependencies)
	s.logger.Info("repository_ingestion_ready", "repository_id", repo.ID, "job_id", job.ID)
	s.runSocioEnrichment(repo.ID)
	s.syncCodeowners(context.Background(), repo)
	s.runDriftValidation(repo.ID)
	return nil
}

func (s *Service) runDriftValidation(repositoryID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if s.driftEngine != nil {
			if _, err := s.driftEngine.ValidateAll(ctx, repositoryID); err != nil {
				s.logger.Warn("drift_validation_failed", "repository_id", repositoryID, "error", err)
			}
		}
	}()
}

func (s *Service) syncCodeowners(ctx context.Context, repo Repository) {
	if s.teamsSvc == nil {
		return
	}
	candidates := []string{
		filepath.Join(repo.WorkspacePath, "CODEOWNERS"),
		filepath.Join(repo.WorkspacePath, ".github", "CODEOWNERS"),
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := s.teamsSvc.UpsertFromCodeowners(ctx, repo.ID, string(b)); err != nil {
			s.logger.Warn("codeowners_sync_failed", "repository_id", repo.ID, "error", err)
		}
		return
	}
}

func (s *Service) publishJobProgress(
	ctx context.Context,
	job *jobqueue.Job,
	repoID int64,
	repoStatus Status,
	pct float64,
	files, symbols, edges int,
	durations *ingestprogress.StepDurations,
) {
	current := ingestprogress.RepoStatusToCurrentStep(string(repoStatus))
	if repoStatus == StatusParsing && pct < 28 {
		current = ingestprogress.StepIndexWorkspace
	}
	total := files
	if total < 1 && pct > 0 {
		total = int(float64(files) / (pct / 100))
		if total < files {
			total = files + 100
		}
	}
	ev := ingestprogress.BuildEvent(job.Phase, string(repoStatus), current, files, total, pct, durations.Snapshot())
	if repoStatus == StatusQueued {
		ev.Status = ingestprogress.StatusQueued
	}
	if repoStatus == StatusReady {
		ev.Status = ingestprogress.StatusComplete
	}
	if repoStatus == StatusFailed {
		ev.Status = ingestprogress.StatusFailed
	}
	if s.broadcaster != nil {
		s.broadcaster.Publish(repoID, ev)
	}
	if s.queue != nil && job != nil {
		_ = s.queue.UpdateProgress(ctx, job.ID, ev)
	}
}

func (s *Service) publishComplete(ctx context.Context, job *jobqueue.Job, repoID int64, d *ingestprogress.StepDurations, files, symbols, edges int) {
	ev := ingestprogress.BuildEvent(1, string(StatusReady), ingestprogress.StepSemanticEmbeddings, files, files, 100, d.Snapshot())
	ev.Status = ingestprogress.StatusComplete
	if s.broadcaster != nil {
		s.broadcaster.Publish(repoID, ev)
	}
	if s.queue != nil && job != nil {
		_ = s.queue.UpdateProgress(ctx, job.ID, ev)
	}
}

func (s *Service) publishFailed(ctx context.Context, job *jobqueue.Job, repoID int64, step string, d *ingestprogress.StepDurations, msg string) {
	ev := ingestprogress.BuildEvent(1, string(StatusFailed), step, 0, 0, 0, d.Snapshot())
	ev.Status = ingestprogress.StatusFailed
	if s.broadcaster != nil {
		s.broadcaster.Publish(repoID, ev)
	}
	if s.queue != nil && job != nil {
		_ = s.queue.UpdateProgress(ctx, job.ID, ev)
	}
}
