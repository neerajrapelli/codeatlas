package repoingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrRepositoryNotFound is returned when a repository row does not exist.
var ErrRepositoryNotFound = errors.New("repository not found")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Create(ctx context.Context, repo Repository) (Repository, error) {
	var out Repository
	var metadataRaw []byte
	tenantID := repo.TenantID
	if tenantID == "" {
		tenantID = "default"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO repositories(name, root_path, source_type, source_url, branch, workspace_path, status, current_stage, error_details, tenant_id)
		VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7,$8,NULLIF($9,''),$10)
		RETURNING
		  id,name,source_type,COALESCE(source_url,''),COALESCE(branch,''),workspace_path,status,current_stage,
		  progress_percent,files_indexed,symbols_indexed,edges_indexed,embeddings_indexed,stage_metadata,
		  COALESCE(error_details,''),created_at,updated_at
	`, repo.Name, repo.WorkspacePath, repo.SourceType, repo.SourceURL, repo.Branch, repo.WorkspacePath, repo.Status, repo.Status, repo.ErrorDetails, tenantID).Scan(
		&out.ID, &out.Name, &out.SourceType, &out.SourceURL, &out.Branch, &out.WorkspacePath, &out.Status, &out.CurrentStage,
		&out.ProgressPercent, &out.FilesIndexed, &out.SymbolsIndexed, &out.EdgesIndexed, &out.EmbeddingsIndexed, &metadataRaw,
		&out.ErrorDetails, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return Repository{}, fmt.Errorf("insert repository: %w", err)
	}
	_ = json.Unmarshal(metadataRaw, &out.StageMetadata)
	out.TenantID = tenantID
	return out, nil
}

func (s *Store) UpdateStatus(ctx context.Context, id int64, status Status, errDetails string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE repositories
		SET status=$2,
		    current_stage=$2,
		    error_details=NULLIF($3,''),
		    indexing_started_at=CASE WHEN $2='parsing' THEN NOW() ELSE indexing_started_at END,
		    indexed_at=CASE WHEN $2='ready' THEN NOW() ELSE indexed_at END,
		    progress_percent=CASE WHEN $2='ready' THEN 100 ELSE progress_percent END,
		    last_heartbeat=NOW(),
		    updated_at=NOW()
		WHERE id=$1
	`, id, status, errDetails)
	if err != nil {
		return fmt.Errorf("update repository status: %w", err)
	}
	return nil
}

type ProgressUpdate struct {
	Stage Status
	ProgressPercent float64
	FilesIndexed int
	SymbolsIndexed int
	EdgesIndexed int
	EmbeddingsIndexed int
	StageMetadata map[string]any
}

func (s *Store) UpdateProgress(ctx context.Context, id int64, p ProgressUpdate) error {
	metadata := p.StageMetadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE repositories
		SET status=$2,
		    current_stage=$2,
		    progress_percent=LEAST(GREATEST($3, 0), 100),
		    files_indexed=$4,
		    symbols_indexed=$5,
		    edges_indexed=$6,
		    embeddings_indexed=$7,
		    stage_metadata=$8::jsonb,
		    last_heartbeat=NOW(),
		    updated_at=NOW()
		WHERE id=$1
	`, id, p.Stage, p.ProgressPercent, p.FilesIndexed, p.SymbolsIndexed, p.EdgesIndexed, p.EmbeddingsIndexed, string(raw))
	if err != nil {
		return fmt.Errorf("update progress: %w", err)
	}
	return nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (Repository, error) {
	var r Repository
	var metadataRaw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id,name,COALESCE(source_type,''),COALESCE(source_url,''),COALESCE(branch,''),COALESCE(workspace_path,''),status,current_stage,progress_percent,
		       files_indexed,symbols_indexed,edges_indexed,embeddings_indexed,stage_metadata,COALESCE(error_details,''),created_at,updated_at
		FROM repositories
		WHERE id=$1
	`, id).Scan(&r.ID, &r.Name, &r.SourceType, &r.SourceURL, &r.Branch, &r.WorkspacePath, &r.Status, &r.CurrentStage, &r.ProgressPercent,
		&r.FilesIndexed, &r.SymbolsIndexed, &r.EdgesIndexed, &r.EmbeddingsIndexed, &metadataRaw, &r.ErrorDetails, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Repository{}, ErrRepositoryNotFound
		}
		return Repository{}, fmt.Errorf("get repository: %w", err)
	}
	_ = json.Unmarshal(metadataRaw, &r.StageMetadata)
	return r, nil
}

func (s *Store) DeleteRepository(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM repositories WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete repository: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

// DeleteFilesForRepository removes indexed graph rows for a repo while keeping the repository row.
func (s *Store) DeleteFilesForRepository(ctx context.Context, repositoryID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM files WHERE repository_id=$1`, repositoryID)
	if err != nil {
		return fmt.Errorf("delete indexed files: %w", err)
	}
	return nil
}

func (s *Store) ListRecent(ctx context.Context, tenantID string, limit int) ([]Repository, error) {
	if limit <= 0 {
		limit = 20
	}
	baseSelect := `
		SELECT id,name,COALESCE(source_type,''),COALESCE(source_url,''),COALESCE(branch,''),COALESCE(workspace_path,''),status,current_stage,progress_percent,
		       files_indexed,symbols_indexed,edges_indexed,embeddings_indexed,stage_metadata,COALESCE(error_details,''),created_at,updated_at,
		       COALESCE(tenant_id,'')
		FROM repositories`
	var rows pgx.Rows
	var err error
	if tenantID == "" {
		rows, err = s.pool.Query(ctx, baseSelect+` ORDER BY created_at DESC LIMIT $1`, limit)
	} else {
		rows, err = s.pool.Query(ctx, baseSelect+` WHERE tenant_id = $2 ORDER BY created_at DESC LIMIT $1`, limit, tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	defer rows.Close()
	out := make([]Repository, 0, limit)
	for rows.Next() {
		var r Repository
		var metadataRaw []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.SourceType, &r.SourceURL, &r.Branch, &r.WorkspacePath, &r.Status, &r.CurrentStage, &r.ProgressPercent,
			&r.FilesIndexed, &r.SymbolsIndexed, &r.EdgesIndexed, &r.EmbeddingsIndexed, &metadataRaw, &r.ErrorDetails, &r.CreatedAt, &r.UpdatedAt, &r.TenantID); err != nil {
			return nil, fmt.Errorf("scan repository: %w", err)
		}
		_ = json.Unmarshal(metadataRaw, &r.StageMetadata)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetProgress(ctx context.Context, id int64) (ProgressResponse, error) {
	var p ProgressResponse
	p.RepositoryID = id
	var metadataRaw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT status,current_stage,progress_percent,files_indexed,symbols_indexed,edges_indexed,embeddings_indexed,stage_metadata,COALESCE(error_details,'')
		FROM repositories
		WHERE id=$1
	`, id).Scan(&p.Status, &p.Stage, &p.ProgressPercent, &p.Metrics.FilesIndexed, &p.Metrics.SymbolsIndexed, &p.Metrics.EdgesIndexed, &p.Metrics.EmbeddingsIndexed, &metadataRaw, &p.ErrorDetails)
	if err != nil {
		return ProgressResponse{}, fmt.Errorf("get progress: %w", err)
	}
	_ = json.Unmarshal(metadataRaw, &p.StageMetadata)
	return p, nil
}

func workspaceName(sourceType SourceType) string {
	return fmt.Sprintf("%s-%d", sourceType, time.Now().UnixNano())
}
