package repoingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"codeatlas/apps/api/internal/driftdetector"
	"codeatlas/apps/api/internal/teams"
	"codeatlas/apps/api/internal/indexer"
	"codeatlas/apps/api/internal/tenant"
	"codeatlas/apps/api/internal/ingestprogress"
	"codeatlas/apps/api/internal/ingestion"
	"codeatlas/apps/api/internal/jobqueue"
	"codeatlas/apps/api/internal/vcsauth"
)

type Service struct {
	workspaceRoot string
	store         *Store
	indexer       *indexer.Service
	socioIngest   *ingestion.Service
	queue         jobqueue.JobQueue
	progressBus   ingestprogress.EventBus
	driftEngine   *driftdetector.Engine
	teamsSvc      *teams.Service
	sources       map[SourceType]Source
	repoSources   *vcsauth.Store
	logger             *slog.Logger
	parseWorkers       int
	embeddingsEnabled  bool
}

func NewService(
	workspaceRoot string,
	store *Store,
	idx *indexer.Service,
	socioIngest *ingestion.Service,
	queue jobqueue.JobQueue,
	progressBus ingestprogress.EventBus,
	driftEngine *driftdetector.Engine,
	teamsSvc *teams.Service,
	logger *slog.Logger,
	zipMaxBytes int64,
	zipMaxFiles int,
	parseWorkers int,
	cloneResolver CloneResolver,
	repoSources *vcsauth.Store,
	embeddingsEnabled bool,
) *Service {
	svc := &Service{
		workspaceRoot:       workspaceRoot,
		store:               store,
		indexer:             idx,
		socioIngest:         socioIngest,
		queue:               queue,
		progressBus:         progressBus,
		driftEngine:         driftEngine,
		teamsSvc:            teamsSvc,
		logger:              logger,
		parseWorkers:        parseWorkers,
		embeddingsEnabled:   embeddingsEnabled,
		sources: map[SourceType]Source{
			SourceGitHub:    NewGitSource(SourceGitHub),
			SourceGitLab:    NewGitSource(SourceGitLab),
			SourceBitbucket: NewGitSource(SourceBitbucket),
			SourceZIP:       NewZIPSource(zipMaxBytes, zipMaxFiles),
		},
		repoSources: repoSources,
	}
	for _, st := range []SourceType{SourceGitHub, SourceGitLab, SourceBitbucket} {
		if g, ok := svc.sources[st].(*GitSource); ok {
			g.SetCloneResolver(cloneResolver)
		}
	}
	return svc
}

func (s *Service) Ingest(ctx context.Context, req CreateRequest) (Repository, error) {
	src, ok := s.sources[req.SourceType]
	if !ok {
		return Repository{}, fmt.Errorf("unsupported source type: %s", req.SourceType)
	}
	if req.SourceType != SourceZIP {
		if err := validateGitIngestRequest(req); err != nil {
			return Repository{}, err
		}
	}

	workspacePath, err := s.prepareWorkspacePath(req.SourceType)
	if err != nil {
		return Repository{}, err
	}
	repoName := req.DisplayName
	if repoName == "" {
		repoName = deriveName(req)
	}

	repo, err := s.store.Create(ctx, Repository{
		Name:          repoName,
		SourceType:    req.SourceType,
		SourceURL:     req.SourceURL,
		Branch:        req.Branch,
		WorkspacePath: workspacePath,
		Status:        StatusQueued,
	})
	if err != nil {
		return Repository{}, err
	}

	stepStatus := StatusCloning
	if req.SourceType == SourceZIP {
		stepStatus = StatusExtracting
	}
	if err := s.store.UpdateStatus(ctx, repo.ID, stepStatus, ""); err != nil {
		return repo, err
	}

	if err := src.Prepare(ctx, req, workspacePath); err != nil {
		_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
		_ = os.RemoveAll(workspacePath)
		return repo, err
	}

	if err := s.store.UpdateStatus(ctx, repo.ID, StatusParsing, ""); err != nil {
		_ = os.RemoveAll(workspacePath)
		return repo, err
	}

	_, err = s.indexer.Run(ctx, indexer.Request{
		RepositoryID:   repo.ID,
		RepositoryPath: workspacePath,
		RepositoryName: repoName,
		TenantID:       tenant.Normalize(repo.TenantID),
		ParseWorkers:   s.parseWorkers,
	})
	if err != nil {
		_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
		_ = os.RemoveAll(workspacePath)
		return repo, err
	}

	repo.Status = StatusReady
	if err := s.store.UpdateStatus(ctx, repo.ID, StatusReady, ""); err != nil {
		return repo, err
	}
	s.logger.Info("repository_ingestion_ready", "repository_id", repo.ID, "source_type", repo.SourceType, "workspace", workspacePath)
	s.runSocioEnrichment(repo.ID)
	s.runDriftValidation(repo.ID)
	return repo, nil
}

func (s *Service) Enqueue(ctx context.Context, req CreateRequest, tenantID string) (Repository, string, error) {
	if _, ok := s.sources[req.SourceType]; !ok {
		return Repository{}, "", fmt.Errorf("unsupported source type: %s", req.SourceType)
	}
	if req.SourceType != SourceZIP {
		if err := validateGitIngestRequest(req); err != nil {
			return Repository{}, "", err
		}
	}
	if s.queue == nil {
		return Repository{}, "", fmt.Errorf("ingestion job queue unavailable")
	}

	workspacePath, err := s.prepareWorkspacePath(req.SourceType)
	if err != nil {
		return Repository{}, "", err
	}
	repoName := req.DisplayName
	if repoName == "" {
		repoName = deriveName(req)
	}
	if tenantID == "" {
		tenantID = "default"
	}

	repo, err := s.store.Create(ctx, Repository{
		Name:          repoName,
		TenantID:      tenantID,
		SourceType:    req.SourceType,
		SourceURL:     req.SourceURL,
		Branch:        req.Branch,
		WorkspacePath: workspacePath,
		Status:        StatusQueued,
	})
	if err != nil {
		return Repository{}, "", err
	}

	meta := map[string]any{
		"sourceType":  req.SourceType,
		"sourceUrl":   req.SourceURL,
		"branch":      req.Branch,
		"displayName": req.DisplayName,
		"tenantId":    tenantID,
	}
	if req.UserSubject != "" {
		meta["userSubject"] = req.UserSubject
	}
	if req.ZIPPath != "" {
		meta["zipPath"] = req.ZIPPath
	}
	if req.ProviderTokenID != nil {
		meta["providerTokenId"] = req.ProviderTokenID.String()
	}
	if req.ExternalRepoID != "" {
		meta["externalRepoId"] = req.ExternalRepoID
	}
	if req.ExternalRepoFullName != "" {
		meta["externalRepoFullName"] = req.ExternalRepoFullName
	}
	jobID, err := s.queue.Enqueue(ctx, strconv.FormatInt(repo.ID, 10), 1, meta)
	if err != nil {
		return Repository{}, "", err
	}
	if s.repoSources != nil {
		_ = s.repoSources.UpsertRepoSource(ctx, repo.ID, string(req.SourceType), req.ExternalRepoID, req.ExternalRepoFullName, req.ProviderTokenID, map[string]any{
			"branch": req.Branch,
		})
	}
	if s.progressBus != nil {
		s.progressBus.Publish(repo.ID, ingestprogress.NewQueuedEvent(repo.ID, 1))
	}
	s.logger.Info("repository_ingestion_queued", "repository_id", repo.ID, "job_id", jobID)
	return repo, jobID, nil
}

func (s *Service) processInBackground(repo Repository, req CreateRequest, src Source) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Minute)
	defer cancel()
	start := time.Now()

	stepStatus := StatusCloning
	if req.SourceType == SourceZIP {
		stepStatus = StatusExtracting
	}
	if err := s.store.UpdateStatus(ctx, repo.ID, stepStatus, ""); err != nil {
		s.logger.Error("repository_status_update_failed", "repository_id", repo.ID, "error", err)
		return
	}

	stageStart := time.Now()
	if err := src.Prepare(ctx, req, repo.WorkspacePath); err != nil {
		_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
		_ = os.RemoveAll(repo.WorkspacePath)
		s.logger.Error("repository_prepare_failed", "repository_id", repo.ID, "error", err)
		return
	}

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

	lastProgress := time.Now()
	var lastStage Status
	report := func(evt indexer.ProgressEvent) {
		now := time.Now()
		stage := Status(evt.Stage)
		if stage == "" {
			stage = StatusParsing
		}
		stageChanged := stage != lastStage
		if now.Sub(lastProgress) < 500*time.Millisecond && evt.Progress < 100 &&
			stage != StatusGeneratingEmbeddings && stage != StatusBuildingGraph && !stageChanged {
			return
		}
		lastProgress = now
		lastStage = stage
		progress := mapIndexerProgress(stage, evt, s.embeddingsEnabled)
		_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
			Stage:             stage,
			ProgressPercent:   progress,
			FilesIndexed:      evt.Files,
			SymbolsIndexed:    evt.Symbols,
			EdgesIndexed:      evt.Edges,
			EmbeddingsIndexed: evt.Embeddings,
			StageMetadata:     evt.Metadata,
		})
		s.logger.Info("repository_stage_progress",
			"repository_id", repo.ID,
			"stage", evt.Stage,
			"progress", progress,
			"files", evt.Files,
			"symbols", evt.Symbols,
			"edges", evt.Edges,
			"embeddings", evt.Embeddings,
		)
	}

	if err := s.store.UpdateStatus(ctx, repo.ID, StatusParsing, ""); err != nil {
		s.logger.Error("repository_status_update_failed", "repository_id", repo.ID, "error", err)
		return
	}

	res, err := s.indexer.Run(ctx, indexer.Request{
		RepositoryID:   repo.ID,
		RepositoryPath: repo.WorkspacePath,
		RepositoryName: repo.Name,
		TenantID:       tenant.Normalize(repo.TenantID),
		ParseWorkers:   s.parseWorkers,
		OnProgress:     report,
	})
	if err != nil {
		_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
		_ = os.RemoveAll(repo.WorkspacePath)
		s.logger.Error("repository_indexing_failed", "repository_id", repo.ID, "error", err)
		return
	}

	_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
		Stage:             StatusReady,
		ProgressPercent:   100,
		FilesIndexed:      maxInt(res.Files, 1),
		SymbolsIndexed:    res.Symbols,
		EdgesIndexed:      res.FileDependencies,
		EmbeddingsIndexed: res.Embeddings,
		StageMetadata: map[string]any{
			"totalDurationMs": time.Since(start).Milliseconds(),
			"indexDurationMs": res.Duration.Milliseconds(),
		},
	})
	if err := s.store.UpdateStatus(ctx, repo.ID, StatusReady, ""); err != nil {
		s.logger.Error("repository_status_update_failed", "repository_id", repo.ID, "error", err)
		return
	}
	s.logger.Info("repository_ingestion_ready", "repository_id", repo.ID, "source_type", repo.SourceType, "workspace", repo.WorkspacePath)
	s.runSocioEnrichment(repo.ID)
	s.runDriftValidation(repo.ID)
}

func (s *Service) runSocioEnrichment(repositoryID int64) {
	if s.socioIngest == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		if err := s.socioIngest.RunPhase1GitHubHistory(ctx, repositoryID); err != nil {
			s.logger.Error("socio_ingestion_failed", "repository_id", repositoryID, "error", err)
			return
		}
		if err := s.socioIngest.RunPhase2EngineeringMemory(ctx, repositoryID); err != nil {
			s.logger.Error("engineering_memory_failed", "repository_id", repositoryID, "error", err)
		}
	}()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *Service) ListRecent(ctx context.Context, tenantID string, limit int) ([]Repository, error) {
	return s.store.ListRecent(ctx, tenantID, limit)
}

func (s *Service) GetProgress(ctx context.Context, repositoryID int64) (ProgressResponse, error) {
	return s.store.GetProgress(ctx, repositoryID)
}

// GetByID returns a single repository row.
func (s *Service) GetByID(ctx context.Context, id int64) (Repository, error) {
	return s.store.GetByID(ctx, id)
}

// Delete removes workspace files, deletes the repository row, and cascades indexed data.
func (s *Service) Delete(ctx context.Context, id int64) (Repository, error) {
	repo, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Repository{}, err
	}
	// Guard already verified tenant access; delete by primary key so legacy tenant_id
	// values on the row cannot cause a silent 0-row delete.
	if err := s.store.DeleteRepository(ctx, id, ""); err != nil {
		return Repository{}, err
	}
	if repo.WorkspacePath != "" {
		path := repo.WorkspacePath
		go func() { _ = os.RemoveAll(path) }()
	}
	return repo, nil
}

// ErrZIPWorkspaceMissing is returned when a ZIP-sourced repo no longer has extracted files on disk.
var ErrZIPWorkspaceMissing = errors.New("zip workspace missing; upload the archive again to re-index")

// Reindex clears indexed rows for the repository and runs ingestion again using stored source metadata.
func (s *Service) Reindex(ctx context.Context, id int64) error {
	repo, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

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

	skipPrepare := false
	if repo.SourceType == SourceZIP {
		entries, err := os.ReadDir(repo.WorkspacePath)
		if err != nil || len(entries) == 0 {
			return ErrZIPWorkspaceMissing
		}
		skipPrepare = true
	}

	if err := s.store.DeleteFilesForRepository(ctx, id); err != nil {
		return err
	}
	_ = s.store.UpdateProgress(ctx, id, ProgressUpdate{
		Stage:             StatusQueued,
		ProgressPercent:   0,
		FilesIndexed:      0,
		SymbolsIndexed:    0,
		EdgesIndexed:      0,
		EmbeddingsIndexed: 0,
		StageMetadata:     map[string]any{},
	})

	meta := map[string]any{
		"sourceType":  repo.SourceType,
		"sourceUrl":   repo.SourceURL,
		"branch":      repo.Branch,
		"displayName": repo.Name,
		"reindex":     true,
		"skipPrepare": skipPrepare,
	}
	if s.queue != nil {
		jobID, err := s.queue.Enqueue(ctx, strconv.FormatInt(id, 10), 1, meta)
		if err != nil {
			return err
		}
		if s.progressBus != nil {
			s.progressBus.Publish(id, ingestprogress.NewQueuedEvent(id, 1))
		}
		s.logger.Info("repository_reindex_queued", "repository_id", id, "job_id", jobID)
		return nil
	}
	go s.processReindexBackground(repo, req, src, skipPrepare)
	return nil
}

// GetIngestionStreamEvent returns the latest SSE payload for a repository.
func (s *Service) GetIngestionStreamEvent(ctx context.Context, repositoryID int64) (ingestprogress.StreamEvent, error) {
	if s.queue != nil {
		job, err := s.queue.GetStatus(ctx, strconv.FormatInt(repositoryID, 10))
		if err != nil {
			return ingestprogress.StreamEvent{}, err
		}
		if job != nil {
			ev := job.Progress
			if s.progressBus != nil {
				s.progressBus.Publish(repositoryID, ev)
			}
			return ev, nil
		}
	}
	if s.progressBus != nil {
		if ev, ok := s.progressBus.Latest(repositoryID); ok {
			return ev, nil
		}
	}
	progress, err := s.store.GetProgress(ctx, repositoryID)
	if err != nil {
		return ingestprogress.StreamEvent{}, err
	}
	return ingestprogress.BuildEvent(
		1,
		string(progress.Status),
		ingestprogress.CurrentStepForProgress(string(progress.Stage), progress.ProgressPercent, s.embeddingsEnabled),
		progress.Metrics.FilesIndexed,
		progress.Metrics.FilesIndexed,
		progress.ProgressPercent,
		nil,
	), nil
}

func (s *Service) processReindexBackground(repo Repository, req CreateRequest, src Source, skipPrepare bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Minute)
	defer cancel()
	start := time.Now()

	stepStatus := StatusCloning
	if req.SourceType == SourceZIP {
		stepStatus = StatusExtracting
	}
	if err := s.store.UpdateStatus(ctx, repo.ID, stepStatus, ""); err != nil {
		s.logger.Error("repository_status_update_failed", "repository_id", repo.ID, "error", err)
		return
	}

	if !skipPrepare {
		_ = os.RemoveAll(repo.WorkspacePath)
		if err := os.MkdirAll(repo.WorkspacePath, 0o755); err != nil {
			_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
			s.logger.Error("repository_workspace_mkdir_failed", "repository_id", repo.ID, "error", err)
			return
		}

		stageStart := time.Now()
		if err := src.Prepare(ctx, req, repo.WorkspacePath); err != nil {
			_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
			s.logger.Error("repository_prepare_failed", "repository_id", repo.ID, "error", err)
			return
		}

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
	} else {
		_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
			Stage:           stepStatus,
			ProgressPercent: 20,
			StageMetadata:   map[string]any{"reindexReuseWorkspace": true},
		})
	}

	lastProgress := time.Now()
	var lastStage Status
	report := func(evt indexer.ProgressEvent) {
		now := time.Now()
		stage := Status(evt.Stage)
		if stage == "" {
			stage = StatusParsing
		}
		stageChanged := stage != lastStage
		if now.Sub(lastProgress) < 500*time.Millisecond && evt.Progress < 100 &&
			stage != StatusGeneratingEmbeddings && stage != StatusBuildingGraph && !stageChanged {
			return
		}
		lastProgress = now
		lastStage = stage
		progress := mapIndexerProgress(stage, evt, s.embeddingsEnabled)
		_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
			Stage:             stage,
			ProgressPercent:   progress,
			FilesIndexed:      evt.Files,
			SymbolsIndexed:    evt.Symbols,
			EdgesIndexed:      evt.Edges,
			EmbeddingsIndexed: evt.Embeddings,
			StageMetadata:     evt.Metadata,
		})
		s.logger.Info("repository_stage_progress",
			"repository_id", repo.ID,
			"stage", evt.Stage,
			"progress", progress,
			"files", evt.Files,
			"symbols", evt.Symbols,
			"edges", evt.Edges,
			"embeddings", evt.Embeddings,
		)
	}

	if err := s.store.UpdateStatus(ctx, repo.ID, StatusParsing, ""); err != nil {
		s.logger.Error("repository_status_update_failed", "repository_id", repo.ID, "error", err)
		return
	}

	res, err := s.indexer.Run(ctx, indexer.Request{
		RepositoryID:   repo.ID,
		RepositoryPath: repo.WorkspacePath,
		RepositoryName: repo.Name,
		TenantID:       tenant.Normalize(repo.TenantID),
		ParseWorkers:   s.parseWorkers,
		OnProgress:     report,
	})
	if err != nil {
		_ = s.store.UpdateStatus(ctx, repo.ID, StatusFailed, err.Error())
		_ = os.RemoveAll(repo.WorkspacePath)
		s.logger.Error("repository_indexing_failed", "repository_id", repo.ID, "error", err)
		return
	}

	_ = s.store.UpdateProgress(ctx, repo.ID, ProgressUpdate{
		Stage:             StatusReady,
		ProgressPercent:   100,
		FilesIndexed:      maxInt(res.Files, 1),
		SymbolsIndexed:    res.Symbols,
		EdgesIndexed:      res.FileDependencies,
		EmbeddingsIndexed: res.Embeddings,
		StageMetadata: map[string]any{
			"totalDurationMs": time.Since(start).Milliseconds(),
			"indexDurationMs": res.Duration.Milliseconds(),
		},
	})
	if err := s.store.UpdateStatus(ctx, repo.ID, StatusReady, ""); err != nil {
		s.logger.Error("repository_status_update_failed", "repository_id", repo.ID, "error", err)
		return
	}
	s.logger.Info("repository_reindex_ready", "repository_id", repo.ID, "source_type", repo.SourceType, "workspace", repo.WorkspacePath)
	s.runSocioEnrichment(repo.ID)
	s.runDriftValidation(repo.ID)
}

func (s *Service) prepareWorkspacePath(sourceType SourceType) (string, error) {
	if err := os.MkdirAll(s.workspaceRoot, 0o755); err != nil {
		return "", fmt.Errorf("mkdir workspace root: %w", err)
	}
	workspacePath := filepath.Join(s.workspaceRoot, workspaceName(sourceType))
	absWorkspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}
	if err := os.MkdirAll(absWorkspacePath, 0o755); err != nil {
		return "", fmt.Errorf("mkdir workspace path: %w", err)
	}
	return absWorkspacePath, nil
}

func validateGitIngestRequest(req CreateRequest) error {
	if err := ValidateGitSourceURL(req.SourceType, req.SourceURL); err != nil {
		return err
	}
	return ValidateGitBranch(req.Branch)
}

func deriveName(req CreateRequest) string {
	if req.SourceType == SourceZIP {
		base := filepath.Base(req.ZIPPath)
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
	parts := strings.Split(strings.Trim(req.SourceURL, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "repository"
}
