package archintel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type QueryService struct {
	pool *pgxpool.Pool
}

func NewQueryService(pool *pgxpool.Pool) *QueryService {
	return &QueryService{pool: pool}
}

func (q *QueryService) ListTimeline(ctx context.Context, repositoryID int64, limit int) ([]TimelineEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := q.pool.Query(ctx, `
		SELECT e.id::text, e.event_at, e.event_type, d.title, e.summary, e.decision_id::text, d.affected_modules, d.affected_files, d.participants, d.source_kind, d.source_ref
		FROM architecture_decision_events e
		JOIN architecture_decisions d ON d.id = e.decision_id
		WHERE e.repository_id=$1
		ORDER BY e.event_at DESC
		LIMIT $2
	`, repositoryID, limit)
	if err != nil {
		return nil, fmt.Errorf("list timeline: %w", err)
	}
	defer rows.Close()
	var out []TimelineEntry
	for rows.Next() {
		var item TimelineEntry
		var modulesRaw, filesRaw, participantsRaw []byte
		var sourceKind string
		if err := rows.Scan(
			&item.ID, &item.OccurredAt, &item.Kind, &item.Title, &item.Summary, &item.DecisionID,
			&modulesRaw, &filesRaw, &participantsRaw, &sourceKind, &item.EvidenceRef,
		); err != nil {
			return nil, err
		}
		item.EvidenceKind = EvidenceKind(sourceKind)
		_ = json.Unmarshal(modulesRaw, &item.RelatedModules)
		_ = json.Unmarshal(filesRaw, &item.RelatedFiles)
		_ = json.Unmarshal(participantsRaw, &item.Participants)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *QueryService) ListDecisions(ctx context.Context, repositoryID int64, limit int) ([]DecisionRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := q.pool.Query(ctx, `
		SELECT id::text, repository_id, title, summary, status, confidence, tradeoffs, affected_modules, affected_files, participants, created_at, updated_at
		FROM architecture_decisions
		WHERE repository_id=$1
		ORDER BY updated_at DESC
		LIMIT $2
	`, repositoryID, limit)
	if err != nil {
		return nil, fmt.Errorf("list decisions: %w", err)
	}
	defer rows.Close()
	var out []DecisionRecord
	for rows.Next() {
		var rec DecisionRecord
		var tradeoffsRaw, modulesRaw, filesRaw, participantsRaw []byte
		var status string
		if err := rows.Scan(
			&rec.ID, &rec.RepositoryID, &rec.Title, &rec.Summary, &status, &rec.Confidence,
			&tradeoffsRaw, &modulesRaw, &filesRaw, &participantsRaw, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rec.Status = DecisionStatus(status)
		_ = json.Unmarshal(tradeoffsRaw, &rec.Tradeoffs)
		_ = json.Unmarshal(modulesRaw, &rec.AffectedModules)
		_ = json.Unmarshal(filesRaw, &rec.AffectedFiles)
		_ = json.Unmarshal(participantsRaw, &rec.Participants)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (q *QueryService) Search(ctx context.Context, repositoryID int64, query string, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 30
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	rows, err := q.pool.Query(ctx, `
		WITH q AS (SELECT websearch_to_tsquery('english', $2) AS tsq)
		SELECT
		  d.id::text,
		  d.source_kind,
		  d.title,
		  d.summary,
		  ts_rank(COALESCE(doc.search_document, to_tsvector('english', d.title || ' ' || d.summary)), q.tsq) AS keyword_score,
		  d.updated_at,
		  d.affected_modules,
		  d.participants
		FROM architecture_decisions d
		LEFT JOIN discussion_documents doc
		  ON doc.repository_id=d.repository_id
		 AND doc.source_kind=d.source_kind
		 AND doc.source_id=d.source_ref
		CROSS JOIN q
		WHERE d.repository_id=$1
		  AND (
		    COALESCE(doc.search_document, to_tsvector('english', d.title || ' ' || d.summary)) @@ q.tsq
		    OR d.title ILIKE '%' || $2 || '%'
		    OR d.summary ILIKE '%' || $2 || '%'
		  )
		ORDER BY keyword_score DESC, d.updated_at DESC
		LIMIT $3
	`, repositoryID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search architecture: %w", err)
	}
	defer rows.Close()
	now := time.Now().UTC()
	var out []SearchHit
	for rows.Next() {
		var hit SearchHit
		var updated time.Time
		var modulesRaw, participantsRaw []byte
		if err := rows.Scan(&hit.ID, &hit.Kind, &hit.Title, &hit.Summary, &hit.KeywordScore, &updated, &modulesRaw, &participantsRaw); err != nil {
			return nil, err
		}
		hit.VectorScore = 0
		hit.ModuleBoost = 0
		hit.MaintainerBoost = 0
		ageHours := now.Sub(updated).Hours()
		if ageHours < 0 {
			ageHours = 0
		}
		hit.RecencyBoost = 1 / (1 + ageHours/168) // 1-week half-life style decay
		hit.Score = hit.KeywordScore + hit.RecencyBoost
		_ = json.Unmarshal(modulesRaw, &hit.MatchedModules)
		_ = json.Unmarshal(participantsRaw, &hit.Participants)
		t := updated
		hit.OccurredAt = &t
		hit.EvidenceKind = EvidenceKind(hit.Kind)
		out = append(out, hit)
	}
	return out, rows.Err()
}

func (q *QueryService) ListPRInsights(ctx context.Context, repositoryID int64, limit int) ([]PRInsight, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := q.pool.Query(ctx, `
		SELECT pr.id, pr.external_number, pr.title, COALESCE(c.login,''),
		       COUNT(DISTINCT rc.id)::int AS comments,
		       COUNT(DISTINCT rv.id)::int AS reviews,
		       COALESCE(MAX(pr.closed_at), MAX(pr.created_at))::text
		FROM pull_requests pr
		LEFT JOIN contributors c ON c.id = pr.author_contributor_id
		LEFT JOIN pr_comments rc ON rc.pull_request_id = pr.id
		LEFT JOIN pr_reviews rv ON rv.pull_request_id = pr.id
		WHERE pr.repository_id = $1
		GROUP BY pr.id, pr.external_number, pr.title, c.login
		ORDER BY pr.external_number DESC
		LIMIT $2
	`, repositoryID, limit)
	if err != nil {
		return nil, fmt.Errorf("list pr insights: %w", err)
	}
	defer rows.Close()
	out := make([]PRInsight, 0, limit)
	for rows.Next() {
		var item PRInsight
		var comments, reviews int
		if err := rows.Scan(&item.PullRequestID, &item.Number, &item.Title, &item.Author, &comments, &reviews, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.ReviewDisagreeCt = reviews
		item.Summary = fmt.Sprintf("%d comments, %d reviews", comments, reviews)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *QueryService) ListMaintainerInfluence(ctx context.Context, repositoryID int64, limit int) ([]MaintainerInfluence, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := q.pool.Query(ctx, `
		SELECT c.login, COALESCE(c.display_name,''), COUNT(DISTINCT d.id)::int
		FROM contributors c
		JOIN architecture_decisions d
		  ON d.repository_id = c.repository_id
		 AND d.participants @> to_jsonb(ARRAY[c.login]::text[])
		WHERE c.repository_id=$1
		GROUP BY c.login, c.display_name
		ORDER BY COUNT(DISTINCT d.id) DESC, c.login
		LIMIT $2
	`, repositoryID, limit)
	if err != nil {
		return nil, fmt.Errorf("list maintainer influence: %w", err)
	}
	defer rows.Close()
	var out []MaintainerInfluence
	for rows.Next() {
		var m MaintainerInfluence
		if err := rows.Scan(&m.Login, &m.DisplayName, &m.DecisionsShaped); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (q *QueryService) GetModuleIntelligence(ctx context.Context, repositoryID int64, modulePath string) (ModuleIntelligence, error) {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		modulePath = "."
	}
	decisions, err := q.pool.Query(ctx, `
		SELECT id::text, repository_id, title, summary, status, confidence, tradeoffs, affected_modules, affected_files, participants, created_at, updated_at
		FROM architecture_decisions
		WHERE repository_id=$1
		  AND (
		    affected_modules @> to_jsonb(ARRAY[$2]::text[])
		    OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(affected_files) AS f WHERE f LIKE $2 || '%')
		  )
		ORDER BY updated_at DESC
		LIMIT 30
	`, repositoryID, modulePath)
	if err != nil {
		return ModuleIntelligence{}, err
	}
	defer decisions.Close()
	var items []DecisionRecord
	for decisions.Next() {
		var rec DecisionRecord
		var tradeoffsRaw, modulesRaw, filesRaw, participantsRaw []byte
		var status string
		if err := decisions.Scan(
			&rec.ID, &rec.RepositoryID, &rec.Title, &rec.Summary, &status, &rec.Confidence,
			&tradeoffsRaw, &modulesRaw, &filesRaw, &participantsRaw, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return ModuleIntelligence{}, err
		}
		rec.Status = DecisionStatus(status)
		_ = json.Unmarshal(tradeoffsRaw, &rec.Tradeoffs)
		_ = json.Unmarshal(modulesRaw, &rec.AffectedModules)
		_ = json.Unmarshal(filesRaw, &rec.AffectedFiles)
		_ = json.Unmarshal(participantsRaw, &rec.Participants)
		items = append(items, rec)
	}
	timeline, _ := q.ListTimeline(ctx, repositoryID, 40)
	filteredTimeline := make([]TimelineEntry, 0, len(timeline))
	for _, t := range timeline {
		for _, p := range t.RelatedFiles {
			if strings.HasPrefix(p, modulePath) {
				filteredTimeline = append(filteredTimeline, t)
				break
			}
		}
	}
	prs, _ := q.ListPRInsights(ctx, repositoryID, 20)
	influence, _ := q.ListMaintainerInfluence(ctx, repositoryID, 10)
	return ModuleIntelligence{
		ModulePath:     modulePath,
		DecisionCount:  len(items),
		Decisions:      items,
		RecentTimeline: filteredTimeline,
		TopMaintainers: influence,
		RelatedPRs:     prs,
	}, nil
}
