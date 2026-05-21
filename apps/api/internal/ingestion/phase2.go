package ingestion

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"codeatlas/apps/api/internal/github"
	"codeatlas/apps/api/internal/signals"
	"codeatlas/apps/api/internal/socio"

	"github.com/google/uuid"
)

// RunPhase2EngineeringMemory syncs issues/PR discussions and extracts architecture_signals.
func (s *Service) RunPhase2EngineeringMemory(ctx context.Context, repositoryID int64) error {
	ref, err := s.store.GetRepository(ctx, repositoryID)
	if err != nil {
		return err
	}
	if ref.SourceType != "github" {
		runID, _ := s.store.StartIngestionRun(ctx, repositoryID, socio.PhaseEngineering)
		_ = s.store.UpdateRunProgress(ctx, runID, 100, socio.StatusSkipped, "engineering memory requires GitHub source")
		return nil
	}
	if !s.github.Enabled() {
		runID, _ := s.store.StartIngestionRun(ctx, repositoryID, socio.PhaseEngineering)
		_ = s.store.UpdateRunProgress(ctx, runID, 0, socio.StatusSkipped, "GITHUB_TOKEN not configured")
		return nil
	}
	owner, name, err := github.ParseRepoURL(ref.SourceURL)
	if err != nil {
		runID, _ := s.store.StartIngestionRun(ctx, repositoryID, socio.PhaseEngineering)
		_ = s.store.UpdateRunProgress(ctx, runID, 0, socio.StatusFailed, err.Error())
		return err
	}

	runID, err := s.store.StartIngestionRun(ctx, repositoryID, socio.PhaseEngineering)
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

	type textSource struct {
		text       string
		sourceKind string
		sourceID   *uuid.UUID
		label      string
	}

	var sources []textSource

	if err := step(socio.StepSyncIssues, func(ctx context.Context) (int, error) {
		issues, err := s.github.ListIssues(ctx, owner, name, 2)
		if err != nil {
			return 0, err
		}
		n := 0
		for _, iss := range issues {
			var authorID *uuid.UUID
			if iss.User != nil {
				authorID, _ = ensureContributor(ctx, iss.User)
			}
			issueID, err := s.store.UpsertIssue(ctx, repositoryID, iss.Number, iss.Title, iss.State, authorID, iss.CreatedAt, nil)
			if err != nil {
				continue
			}
			n++
			combined := iss.Title + "\n" + iss.Body
			for _, d := range signals.ExtractFromText(combined) {
				for _, path := range d.FilePaths {
					if fid, ok := resolveFile(pathIndex, path); ok {
						_ = s.store.UpsertIssueFileRef(ctx, issueID, fid, "mentioned")
					}
				}
			}
			sources = append(sources, textSource{
				text: combined, sourceKind: "issue", sourceID: &issueID,
				label: fmt.Sprintf("Issue #%d", iss.Number),
			})
		}
		return n, nil
	}); err != nil {
		return err
	}
	_ = s.store.UpdateRunProgress(ctx, runID, 35, socio.StatusRunning, "")

	if err := step(socio.StepSyncPRDiscussions, func(ctx context.Context) (int, error) {
		prs, err := s.store.ListPullRequestRefs(ctx, repositoryID, 40)
		if err != nil {
			return 0, err
		}
		n := 0
		for _, pr := range prs {
			sources = append(sources, textSource{
				text: pr.Title, sourceKind: "pull_request", sourceID: &pr.ID,
				label: fmt.Sprintf("PR #%d", pr.Number),
			})
			comments, err := s.github.ListIssueComments(ctx, owner, name, pr.Number)
			if err != nil {
				continue
			}
			for _, cmt := range comments {
				var authorID *uuid.UUID
				if cmt.User != nil {
					authorID, _ = ensureContributor(ctx, cmt.User)
				}
				ext := strconv.FormatInt(cmt.ID, 10)
				_ = s.store.UpsertPRComment(ctx, pr.ID, authorID, cmt.Body, cmt.CreatedAt, ext)
				sources = append(sources, textSource{
					text: cmt.Body, sourceKind: "pr_comment", sourceID: &pr.ID,
					label: fmt.Sprintf("PR #%d comment", pr.Number),
				})
				n++
			}
		}
		return n, nil
	}); err != nil {
		return err
	}
	_ = s.store.UpdateRunProgress(ctx, runID, 70, socio.StatusRunning, "")

	if err := step(socio.StepExtractSignals, func(ctx context.Context) (int, error) {
		if err := s.store.ClearArchitectureSignals(ctx, repositoryID); err != nil {
			return 0, err
		}
		inserted := 0
		seen := make(map[string]struct{})
		for _, src := range sources {
			drafts := signals.ExtractFromText(src.text)
			if len(drafts) == 0 {
				continue
			}
			for _, d := range drafts {
				fileIDs := fileIDsForDraft(pathIndex, d.FilePaths)
				targets := fileIDs
				if len(targets) == 0 {
					targets = []int64{0}
				}
				for _, fid := range targets {
					var fileID *int64
					if fid > 0 {
						fileID = &fid
					}
					dedupe := fmt.Sprintf("%s|%v|%s|%s", d.SignalType, fileID, src.sourceKind, d.Summary)
					if _, ok := seen[dedupe]; ok {
						continue
					}
					seen[dedupe] = struct{}{}
					meta := map[string]any{"sourceLabel": src.label}
					if err := s.store.InsertArchitectureSignal(ctx, repositoryID, fileID, d.SignalType, d.Summary, d.Confidence, src.sourceKind, src.sourceID, meta); err != nil {
						continue
					}
					inserted++
				}
			}
		}
		return inserted, nil
	}); err != nil {
		return err
	}

	status := socio.StatusCompleted
	if len(sources) == 0 {
		status = socio.StatusPartial
	}
	_ = s.store.UpdateRunProgress(ctx, runID, 100, status, "")
	s.logger.Info("socio_phase2_complete", "repository_id", repositoryID)
	return nil
}

func fileIDsForDraft(pathIndex map[string]int64, paths []string) []int64 {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[int64]struct{})
	var ids []int64
	for _, p := range paths {
		if id, ok := pathIndex[p]; ok {
			if _, dup := seen[id]; !dup {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	return ids
}
