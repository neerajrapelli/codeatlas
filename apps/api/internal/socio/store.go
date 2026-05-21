package socio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Pool exposes the underlying connection pool for analytics queries.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *Store) GetRepository(ctx context.Context, id int64) (RepositoryRef, error) {
	var ref RepositoryRef
	err := s.pool.QueryRow(ctx, `
		SELECT id, COALESCE(source_type,''), COALESCE(source_url,''), COALESCE(branch,''), status
		FROM repositories WHERE id=$1
	`, id).Scan(&ref.ID, &ref.SourceType, &ref.SourceURL, &ref.Branch, &ref.Status)
	if err != nil {
		return RepositoryRef{}, fmt.Errorf("get repository: %w", err)
	}
	return ref, nil
}

func (s *Store) FilePathIndex(ctx context.Context, repositoryID int64) (map[string]int64, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, relative_path FROM files WHERE repository_id=$1
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		out[path] = id
		out[normalizePath(path)] = id
	}
	return out, rows.Err()
}

// NormalizePath matches indexed file paths to GitHub change paths.
func NormalizePath(p string) string {
	return normalizePath(p)
}

func normalizePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "/")
	return p
}

func (s *Store) UpsertContributor(ctx context.Context, repositoryID int64, externalID, login, displayName, avatar string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO contributors(repository_id, external_id, login, display_name, avatar_url)
		VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''))
		ON CONFLICT (repository_id, external_id) DO UPDATE SET
		  login=EXCLUDED.login,
		  display_name=COALESCE(NULLIF(EXCLUDED.display_name,''), contributors.display_name),
		  avatar_url=COALESCE(NULLIF(EXCLUDED.avatar_url,''), contributors.avatar_url),
		  updated_at=NOW()
		RETURNING id
	`, repositoryID, externalID, login, displayName, avatar).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert contributor: %w", err)
	}
	return id, nil
}

func (s *Store) UpsertCommit(ctx context.Context, repositoryID int64, sha string, authorID *uuid.UUID, committedAt time.Time, message string, additions, deletions int) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO commits(repository_id, sha, author_contributor_id, committed_at, message_preview, additions, deletions)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (repository_id, sha) DO UPDATE SET
		  author_contributor_id=COALESCE(EXCLUDED.author_contributor_id, commits.author_contributor_id),
		  message_preview=EXCLUDED.message_preview,
		  additions=EXCLUDED.additions,
		  deletions=EXCLUDED.deletions
		RETURNING id
	`, repositoryID, sha, authorID, committedAt, message, additions, deletions).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *Store) UpsertCommitFile(ctx context.Context, commitID uuid.UUID, fileID int64, kind string, add, del int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO commit_files(commit_id, file_id, change_kind, additions, deletions)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (commit_id, file_id) DO UPDATE SET
		  change_kind=EXCLUDED.change_kind,
		  additions=EXCLUDED.additions,
		  deletions=EXCLUDED.deletions
	`, commitID, fileID, kind, add, del)
	return err
}

func (s *Store) UpsertPullRequest(ctx context.Context, repositoryID int64, number int, title, state string, authorID *uuid.UUID, created time.Time, merged, closed *time.Time, add, del, changed int) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO pull_requests(repository_id, external_number, title, state, author_contributor_id, created_at, merged_at, closed_at, additions, deletions, changed_files)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (repository_id, external_number) DO UPDATE SET
		  title=EXCLUDED.title, state=EXCLUDED.state,
		  merged_at=EXCLUDED.merged_at, closed_at=EXCLUDED.closed_at,
		  additions=EXCLUDED.additions, deletions=EXCLUDED.deletions, changed_files=EXCLUDED.changed_files
		RETURNING id
	`, repositoryID, number, title, state, authorID, created, merged, closed, add, del, changed).Scan(&id)
	return id, err
}

func (s *Store) UpsertPRFile(ctx context.Context, prID uuid.UUID, fileID int64, kind string, add, del int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO pr_files(pull_request_id, file_id, change_kind, additions, deletions)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (pull_request_id, file_id) DO UPDATE SET
		  change_kind=EXCLUDED.change_kind, additions=EXCLUDED.additions, deletions=EXCLUDED.deletions
	`, prID, fileID, kind, add, del)
	return err
}

func (s *Store) ClearSocioData(ctx context.Context, repositoryID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM socio_ingestion_steps WHERE run_id IN (SELECT id FROM socio_ingestion_runs WHERE repository_id=$1)`, repositoryID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM socio_ingestion_runs WHERE repository_id=$1`, repositoryID)
	return err
}

func (s *Store) StartIngestionRun(ctx context.Context, repositoryID int64, phase string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO socio_ingestion_runs(repository_id, phase, status, completion_percent, started_at, last_heartbeat)
		VALUES ($1,$2,'running',0,NOW(),NOW())
		RETURNING id
	`, repositoryID, phase).Scan(&id)
	return id, err
}

func (s *Store) UpdateRunProgress(ctx context.Context, runID uuid.UUID, percent float64, status string, errDetails string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE socio_ingestion_runs
		SET completion_percent=LEAST(GREATEST($2,0),100),
		    status=COALESCE(NULLIF($3,''), status),
		    error_details=NULLIF($4,''),
		    last_heartbeat=NOW(),
		    completed_at=CASE WHEN $3 IN ('completed','failed','skipped','partial') THEN NOW() ELSE completed_at END
		WHERE id=$1
	`, runID, percent, status, errDetails)
	return err
}

func (s *Store) RecordStep(ctx context.Context, runID uuid.UUID, step, status string, durationMs int64, items int, failure map[string]any) error {
	raw, _ := json.Marshal(failure)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO socio_ingestion_steps(run_id, step, status, duration_ms, items_processed, failure_metadata, started_at, completed_at)
		VALUES ($1,$2,$3,NULLIF($4,0),$5,$6::jsonb,NOW(),CASE WHEN $3 IN ('completed','failed','skipped') THEN NOW() ELSE NULL END)
	`, runID, step, status, durationMs, items, string(raw))
	return err
}

func (s *Store) LatestIngestionRun(ctx context.Context, repositoryID int64) (uuid.UUID, string, string, float64, *time.Time, string, error) {
	var id uuid.UUID
	var phase, status string
	var pct float64
	var completed *time.Time
	var errDetails string
	err := s.pool.QueryRow(ctx, `
		SELECT id, phase, status, completion_percent, completed_at, COALESCE(error_details,'')
		FROM socio_ingestion_runs
		WHERE repository_id=$1
		ORDER BY created_at DESC
		LIMIT 1
	`, repositoryID).Scan(&id, &phase, &status, &pct, &completed, &errDetails)
	if err == pgx.ErrNoRows {
		return uuid.Nil, "", StatusPending, 0, nil, "", nil
	}
	return id, phase, status, pct, completed, errDetails, err
}

func (s *Store) ListRunSteps(ctx context.Context, runID uuid.UUID) ([]IngestionStepStatus, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT step, status, duration_ms, items_processed, failure_metadata
		FROM socio_ingestion_steps WHERE run_id=$1 ORDER BY started_at NULLS LAST, step
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IngestionStepStatus
	for rows.Next() {
		var st IngestionStepStatus
		var dur *int64
		var meta []byte
		if err := rows.Scan(&st.Step, &st.Status, &dur, &st.ItemsProcessed, &meta); err != nil {
			return nil, err
		}
		if dur != nil {
			st.DurationMs = dur
		}
		_ = json.Unmarshal(meta, &st.FailureMetadata)
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *Store) ReplaceFileMetrics(ctx context.Context, repositoryID int64, metrics []FileMetrics) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM contributor_file_ownership WHERE repository_id=$1`, repositoryID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM file_metrics WHERE repository_id=$1`, repositoryID); err != nil {
		return err
	}
	for _, m := range metrics {
		_, err := tx.Exec(ctx, `
			INSERT INTO file_metrics(
			  repository_id, file_id, churn_score, commit_count_90d, unique_authors_90d, bus_factor,
			  hotspot_score, risk_level, is_hotspot, has_bus_factor_risk,
			  dominant_owner_id, dominant_owner_share, last_activity_at, computed_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW())
		`, repositoryID, m.FileID, m.ChurnScore, m.CommitCount90d, m.UniqueAuthors90d, m.BusFactor,
			m.HotspotScore, m.RiskLevel, m.IsHotspot, m.HasBusFactorRisk,
			m.DominantOwnerID, m.DominantOwnerShare, m.LastActivityAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ReplaceContributorOwnership(ctx context.Context, repositoryID int64, rows []OwnerRow) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM contributor_file_ownership WHERE repository_id=$1`, repositoryID); err != nil {
		return err
	}
	for _, r := range rows {
		_, err := tx.Exec(ctx, `
			INSERT INTO contributor_file_ownership(repository_id, file_id, contributor_id, commit_count, ownership_share)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (repository_id, file_id, contributor_id) DO UPDATE SET
			  commit_count=EXCLUDED.commit_count, ownership_share=EXCLUDED.ownership_share
		`, repositoryID, r.FileID, r.ContributorID, r.CommitCount, r.Share)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

type OwnerRow struct {
	FileID        int64
	ContributorID uuid.UUID
	CommitCount   int
	Share         float64
}

func (s *Store) ListHotspots(ctx context.Context, repositoryID int64, limit int) ([]HotspotEntry, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.pool.Query(ctx, `
		SELECT f.id, f.relative_path, fm.hotspot_score, fm.churn_score, fm.risk_level, fm.bus_factor, fm.commit_count_90d
		FROM file_metrics fm
		JOIN files f ON f.id = fm.file_id
		WHERE fm.repository_id=$1
		ORDER BY fm.hotspot_score DESC, fm.churn_score DESC
		LIMIT $2
	`, repositoryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HotspotEntry
	for rows.Next() {
		var h HotspotEntry
		if err := rows.Scan(&h.FileID, &h.Path, &h.HotspotScore, &h.ChurnScore, &h.RiskLevel, &h.BusFactor, &h.CommitCount); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) ListOwnership(ctx context.Context, repositoryID int64, fileID int64, limit int) ([]OwnershipSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `
		SELECT f.id, f.relative_path, fm.bus_factor, fm.risk_level, fm.dominant_owner_share,
		       c.id, c.login, c.display_name, c.avatar_url
		FROM file_metrics fm
		JOIN files f ON f.id = fm.file_id
		LEFT JOIN contributors c ON c.id = fm.dominant_owner_id
		WHERE fm.repository_id=$1
	`
	args := []any{repositoryID}
	if fileID > 0 {
		q += ` AND f.id=$2`
		args = append(args, fileID)
	}
	q += ` ORDER BY fm.risk_level DESC, fm.churn_score DESC LIMIT ` + fmt.Sprintf("%d", limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var summaries []OwnershipSummary
	for rows.Next() {
		var sum OwnershipSummary
		var ownerID *uuid.UUID
		var login, display, avatar *string
		if err := rows.Scan(&sum.FileID, &sum.Path, &sum.BusFactor, &sum.RiskLevel, &sum.DominantOwnerShare,
			&ownerID, &login, &display, &avatar); err != nil {
			return nil, err
		}
		if ownerID != nil && login != nil {
			sum.DominantOwner = &Contributor{
				ID: *ownerID, RepositoryID: repositoryID, Login: *login,
			}
			if display != nil {
				sum.DominantOwner.DisplayName = *display
			}
			if avatar != nil {
				sum.DominantOwner.AvatarURL = *avatar
			}
		}
		summaries = append(summaries, sum)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range summaries {
		shares, err := s.ownershipSharesForFile(ctx, repositoryID, summaries[i].FileID)
		if err != nil {
			return nil, err
		}
		summaries[i].Contributors = shares
		summaries[i].ContributorCount = len(shares)
	}
	return summaries, nil
}

func (s *Store) ownershipSharesForFile(ctx context.Context, repositoryID, fileID int64) ([]OwnerShare, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.login, c.display_name, c.avatar_url, cfo.ownership_share, cfo.commit_count
		FROM contributor_file_ownership cfo
		JOIN contributors c ON c.id = cfo.contributor_id
		WHERE cfo.repository_id=$1 AND cfo.file_id=$2
		ORDER BY cfo.ownership_share DESC
		LIMIT 8
	`, repositoryID, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OwnerShare
	for rows.Next() {
		var os OwnerShare
		var display, avatar *string
		if err := rows.Scan(&os.Contributor.ID, &os.Contributor.Login, &display, &avatar, &os.Share, &os.CommitCount); err != nil {
			return nil, err
		}
		if display != nil {
			os.Contributor.DisplayName = *display
		}
		if avatar != nil {
			os.Contributor.AvatarURL = *avatar
		}
		out = append(out, os)
	}
	return out, rows.Err()
}

func (s *Store) FileOverlays(ctx context.Context, repositoryID int64) (map[string]FileOverlay, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT f.id::text, fm.is_hotspot, fm.has_bus_factor_risk, fm.risk_level, COALESCE(c.login,''),
		       (SELECT count(*)::int FROM architecture_signals s WHERE s.file_id=f.id AND s.confidence >= 0.7)
		FROM files f
		LEFT JOIN file_metrics fm ON fm.file_id=f.id AND fm.repository_id=f.repository_id
		LEFT JOIN contributors c ON c.id=fm.dominant_owner_id
		WHERE f.repository_id=$1
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]FileOverlay)
	for rows.Next() {
		var o FileOverlay
		var risk *string
		if err := rows.Scan(&o.FileID, &o.IsHotspot, &o.HasBusFactorRisk, &risk, &o.DominantOwnerLogin, &o.ArchitectureSignals); err != nil {
			return nil, err
		}
		if risk != nil {
			o.RiskLevel = *risk
		}
		out[o.FileID] = o
	}
	return out, rows.Err()
}

func (s *Store) SocioContextForFiles(ctx context.Context, repositoryID int64, fileIDs []int64) (map[int64]FileSocioContext, error) {
	if len(fileIDs) == 0 {
		return map[int64]FileSocioContext{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT fm.file_id, fm.churn_score, fm.bus_factor, fm.risk_level, fm.is_hotspot, fm.has_bus_factor_risk,
		       COALESCE(c.login,''), fm.commit_count_90d, fm.unique_authors_90d
		FROM file_metrics fm
		LEFT JOIN contributors c ON c.id=fm.dominant_owner_id
		WHERE fm.repository_id=$1 AND fm.file_id = ANY($2)
	`, repositoryID, fileIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]FileSocioContext)
	for rows.Next() {
		var ctx FileSocioContext
		if err := rows.Scan(&ctx.FileID, &ctx.ChurnScore, &ctx.BusFactor, &ctx.RiskLevel, &ctx.IsHotspot, &ctx.HasBusFactorRisk,
			&ctx.DominantOwnerLogin, &ctx.CommitCount90d, &ctx.UniqueAuthors90d); err != nil {
			return nil, err
		}
		out[ctx.FileID] = ctx
	}
	return out, rows.Err()
}

type FileSocioContext struct {
	FileID             int64
	ChurnScore         float64
	BusFactor          int
	RiskLevel          string
	IsHotspot          bool
	HasBusFactorRisk   bool
	DominantOwnerLogin string
	CommitCount90d     int
	UniqueAuthors90d   int
}

func MapChangeKind(status string) string {
	switch status {
	case "added":
		return "add"
	case "removed":
		return "delete"
	case "renamed":
		return "rename"
	default:
		return "modify"
	}
}
