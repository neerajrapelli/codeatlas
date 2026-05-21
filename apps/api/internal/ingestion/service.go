package ingestion

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"codeatlas/apps/api/internal/github"
	"codeatlas/apps/api/internal/socio"

	"github.com/google/uuid"
)

// Service runs socio-technical ingestion phases (graph enrichment).
type Service struct {
	store  *socio.Store
	github *github.Client
	logger *slog.Logger
	maxCommitPages  int
	maxPRPages      int
	maxCommitDetail int
}

func NewService(store *socio.Store, gh *github.Client, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store: store, github: gh, logger: logger,
		maxCommitPages: 30, maxPRPages: 20, maxCommitDetail: 400,
	}
}

// RunPhase1GitHubHistory syncs GitHub history into the graph (Phase 1).
func (s *Service) RunPhase1GitHubHistory(ctx context.Context, repositoryID int64) error {
	ref, err := s.store.GetRepository(ctx, repositoryID)
	if err != nil {
		return err
	}
	if ref.SourceType != "github" {
		runID, _ := s.store.StartIngestionRun(ctx, repositoryID, socio.PhaseGitHubHistory)
		_ = s.store.UpdateRunProgress(ctx, runID, 100, socio.StatusSkipped, "socio-technical sync requires GitHub source")
		s.logger.Info("socio_ingestion_skipped", "repository_id", repositoryID, "source_type", ref.SourceType)
		return nil
	}
	if !s.github.Enabled() {
		runID, _ := s.store.StartIngestionRun(ctx, repositoryID, socio.PhaseGitHubHistory)
		_ = s.store.UpdateRunProgress(ctx, runID, 0, socio.StatusSkipped, "GITHUB_TOKEN not configured")
		return nil
	}
	owner, name, err := github.ParseRepoURL(ref.SourceURL)
	if err != nil {
		runID, _ := s.store.StartIngestionRun(ctx, repositoryID, socio.PhaseGitHubHistory)
		_ = s.store.UpdateRunProgress(ctx, runID, 0, socio.StatusFailed, err.Error())
		return err
	}

	runID, err := s.store.StartIngestionRun(ctx, repositoryID, socio.PhaseGitHubHistory)
	if err != nil {
		return err
	}

	pathIndex, err := s.store.FilePathIndex(ctx, repositoryID)
	if err != nil {
		_ = s.store.UpdateRunProgress(ctx, runID, 0, socio.StatusFailed, err.Error())
		return err
	}

	contributorCache := make(map[string]uuid.UUID)
	ensureContributor := func(ctx context.Context, u *github.User) (*uuid.UUID, error) {
		if u == nil || u.Login == "" {
			return nil, nil
		}
		key := strconv.FormatInt(u.ID, 10)
		if id, ok := contributorCache[key]; ok {
			return &id, nil
		}
		id, err := s.store.UpsertContributor(ctx, repositoryID, key, u.Login, u.Name, u.Avatar)
		if err != nil {
			return nil, err
		}
		contributorCache[key] = id
		return &id, nil
	}

	step := func(name string, fn func(context.Context) (int, error)) error {
		start := time.Now()
		_ = s.store.RecordStep(ctx, runID, name, socio.StatusRunning, 0, 0, nil)
		n, err := fn(ctx)
		dur := time.Since(start).Milliseconds()
		if err != nil {
			_ = s.store.RecordStep(ctx, runID, name, socio.StatusFailed, dur, n, map[string]any{"error": err.Error()})
			_ = s.store.UpdateRunProgress(ctx, runID, 0, socio.StatusFailed, err.Error())
			return err
		}
		_ = s.store.RecordStep(ctx, runID, name, socio.StatusCompleted, dur, n, nil)
		return nil
	}

	if err := step(socio.StepResolveRepo, func(ctx context.Context) (int, error) {
		return len(pathIndex), nil
	}); err != nil {
		return err
	}
	_ = s.store.UpdateRunProgress(ctx, runID, 5, socio.StatusRunning, "")

	if err := step(socio.StepSyncCommits, func(ctx context.Context) (int, error) {
		since := time.Now().AddDate(0, 0, -365)
		list, err := s.github.ListCommits(ctx, owner, name, since, s.maxCommitPages)
		if err != nil {
			return 0, err
		}
		linked := 0
		detailBudget := s.maxCommitDetail
		for i, item := range list {
			if i > 0 && i%40 == 0 {
				pct := 5 + float64(i)/float64(max(len(list), 1))*60
				_ = s.store.UpdateRunProgress(ctx, runID, pct, socio.StatusRunning, "")
			}
			var authorID *uuid.UUID
			if item.Author != nil {
				authorID, _ = ensureContributor(ctx, item.Author)
			}
			cid, err := s.store.UpsertCommit(ctx, repositoryID, item.SHA, authorID, item.Date, item.Message, 0, 0)
			if err != nil {
				continue
			}
			if detailBudget <= 0 {
				continue
			}
			detailBudget--
			detail, err := s.github.GetCommit(ctx, owner, name, item.SHA)
			if err != nil || detail == nil {
				continue
			}
			add, del := 0, 0
			for _, f := range detail.Files {
				add += f.Additions
				del += f.Deletions
				fileID, ok := resolveFile(pathIndex, f.Path)
				if !ok {
					continue
				}
				_ = s.store.UpsertCommitFile(ctx, cid, fileID, socio.MapChangeKind(f.Status), f.Additions, f.Deletions)
				linked++
			}
			_, _ = s.store.UpsertCommit(ctx, repositoryID, item.SHA, authorID, item.Date, item.Message, add, del)
		}
		return linked, nil
	}); err != nil {
		return err
	}
	_ = s.store.UpdateRunProgress(ctx, runID, 70, socio.StatusRunning, "")

	if err := step(socio.StepSyncPullRequests, func(ctx context.Context) (int, error) {
		prs, err := s.github.ListPullRequests(ctx, owner, name, s.maxPRPages)
		if err != nil {
			return 0, err
		}
		linked := 0
		for _, pr := range prs {
			var authorID *uuid.UUID
			if pr.User != nil {
				authorID, _ = ensureContributor(ctx, pr.User)
			}
			prID, err := s.store.UpsertPullRequest(ctx, repositoryID, pr.Number, pr.Title, pr.State, authorID,
				pr.CreatedAt, pr.MergedAt, pr.ClosedAt, pr.Additions, pr.Deletions, pr.ChangedFiles)
			if err != nil {
				continue
			}
			files, err := s.github.ListPRFiles(ctx, owner, name, pr.Number)
			if err != nil {
				continue
			}
			for _, f := range files {
				fileID, ok := resolveFile(pathIndex, f.Path)
				if !ok {
					continue
				}
				_ = s.store.UpsertPRFile(ctx, prID, fileID, socio.MapChangeKind(f.Status), f.Additions, f.Deletions)
				linked++
			}
		}
		return linked, nil
	}); err != nil {
		return err
	}
	_ = s.store.UpdateRunProgress(ctx, runID, 85, socio.StatusRunning, "")

	if err := step(socio.StepComputeMetrics, func(ctx context.Context) (int, error) {
		metrics, owners, err := socio.ComputeFileMetrics(ctx, s.store.Pool(), repositoryID)
		if err != nil {
			return 0, err
		}
		if err := s.store.ReplaceFileMetrics(ctx, repositoryID, metrics); err != nil {
			return 0, err
		}
		if err := s.store.ReplaceContributorOwnership(ctx, repositoryID, owners); err != nil {
			return 0, err
		}
		return len(metrics), nil
	}); err != nil {
		return err
	}

	status := socio.StatusCompleted
	if len(pathIndex) == 0 {
		status = socio.StatusPartial
	}
	_ = s.store.UpdateRunProgress(ctx, runID, 100, status, "")
	s.logger.Info("socio_phase1_complete", "repository_id", repositoryID, "owner", owner, "repo", name)
	return nil
}

func resolveFile(index map[string]int64, path string) (int64, bool) {
	if id, ok := index[path]; ok {
		return id, true
	}
	if id, ok := index[socio.NormalizePath(path)]; ok {
		return id, true
	}
	return 0, false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
