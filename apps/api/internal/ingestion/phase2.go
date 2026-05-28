package ingestion

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"codeatlas/apps/api/internal/archintel"
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
	since, _ := s.store.LatestPhaseCompletedAt(ctx, repositoryID, socio.PhaseEngineering)
	if since == nil {
		fallback := time.Now().AddDate(0, -3, 0)
		since = &fallback
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
		issues, err := s.github.ListIssues(ctx, owner, name, 4)
		if err != nil {
			return 0, err
		}
		n := 0
		for _, iss := range issues {
			if since != nil && !iss.CreatedAt.IsZero() && iss.CreatedAt.Before(*since) {
				continue
			}
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
				if since != nil && !cmt.CreatedAt.IsZero() && cmt.CreatedAt.Before(*since) {
					continue
				}
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
	_ = s.store.UpdateRunProgress(ctx, runID, 60, socio.StatusRunning, "")

	if err := step(socio.StepSyncPRReviews, func(ctx context.Context) (int, error) {
		prs, err := s.store.ListPullRequestRefs(ctx, repositoryID, 40)
		if err != nil {
			return 0, err
		}
		n := 0
		for _, pr := range prs {
			reviews, err := s.github.ListPRReviews(ctx, owner, name, pr.Number)
			if err != nil {
				continue
			}
			for _, rev := range reviews {
				if since != nil && !rev.CreatedAt.IsZero() && rev.CreatedAt.Before(*since) {
					continue
				}
				var reviewerID *uuid.UUID
				if rev.User != nil {
					reviewerID, _ = ensureContributor(ctx, rev.User)
				}
				_ = s.store.UpsertPRReview(ctx, pr.ID, reviewerID, rev.State, rev.CreatedAt)
				sources = append(sources, textSource{
					text: rev.Body, sourceKind: "pr_review", sourceID: &pr.ID,
					label: fmt.Sprintf("PR #%d review", pr.Number),
				})
				n++
			}
		}
		return n, nil
	}); err != nil {
		return err
	}
	_ = s.store.UpdateRunProgress(ctx, runID, 75, socio.StatusRunning, "")

	if err := step(socio.StepSyncDiscussions, func(ctx context.Context) (int, error) {
		discussions, err := s.github.ListDiscussions(ctx, owner, name, 4)
		if err != nil {
			return 0, err
		}
		n := 0
		for _, d := range discussions {
			if since != nil && !d.CreatedAt.IsZero() && d.CreatedAt.Before(*since) {
				continue
			}
			body := strings.TrimSpace(d.Body)
			title := strings.TrimSpace(d.Title)
			if title == "" && body == "" {
				continue
			}
			author := ""
			if d.User != nil {
				author = d.User.Login
			}
			moduleHints := make([]string, 0, 4)
			for _, draft := range signals.ExtractFromText(title + "\n" + body) {
				for _, p := range draft.FilePaths {
					moduleHints = append(moduleHints, p)
				}
			}
			sourceID := fmt.Sprintf("%d", d.Number)
			_ = s.store.UpsertDiscussionDocument(
				ctx,
				repositoryID,
				"github_discussion",
				sourceID,
				title,
				author,
				body,
				moduleHints,
				[]string{author},
				&d.CreatedAt,
				map[string]any{"state": d.State},
			)
			sources = append(sources, textSource{
				text: title + "\n" + body, sourceKind: "discussion", sourceID: nil,
				label: fmt.Sprintf("Discussion #%d", d.Number),
			})
			n++
		}
		return n, nil
	}); err != nil {
		return err
	}
	s.logger.Info("socio_phase2_discussions_synced", "repository_id", repositoryID, "since", since)
	_ = s.store.UpdateRunProgress(ctx, runID, 85, socio.StatusRunning, "")

	decisionStore := archintel.NewStore(s.store.Pool())
	discussionAnalyzer := archintel.NewAnalyzer(false)

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
			decisions := discussionAnalyzer.AnalyzeDiscussion(archintel.DiscussionInput{
				SourceKind: src.sourceKind,
				Title:      src.label,
				Body:       src.text,
			})
			for _, dec := range decisions {
				decID, err := decisionStore.UpsertDecision(ctx, repositoryID, dec, src.sourceKind, src.label)
				if err != nil {
					continue
				}
				eventType := "proposed"
				if dec.Status == archintel.DecisionAccepted {
					eventType = "accepted"
				} else if dec.Status == archintel.DecisionRejected {
					eventType = "rejected"
				}
				_ = decisionStore.InsertDecisionEvent(
					ctx,
					repositoryID,
					decID,
					eventType,
					dec.Summary,
					"",
					time.Now().UTC(),
					map[string]any{"sourceKind": src.sourceKind, "sourceLabel": src.label},
				)
			}
		}
		return inserted, nil
	}); err != nil {
		return err
	}
	s.logger.Info("socio_phase2_signals_extracted", "repository_id", repositoryID, "source_count", len(sources))

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
