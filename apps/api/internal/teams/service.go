package teams

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Team struct {
	ID           uuid.UUID `json:"id"`
	RepositoryID int64     `json:"repositoryId"`
	Slug         string    `json:"slug"`
	DisplayName  string    `json:"displayName"`
	Color        string    `json:"color"`
	Source       string    `json:"source"`
	FileCount    int       `json:"fileCount"`
}

type BoundaryViolation struct {
	SourceFile   string `json:"sourceFile"`
	TargetFile   string `json:"targetFile"`
	SourceTeam   string `json:"sourceTeam"`
	TargetTeam   string `json:"targetTeam"`
	Message      string `json:"message"`
}

type OwnershipGap struct {
	FilePath string `json:"filePath"`
	Message  string `json:"message"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) ListTeams(ctx context.Context, repositoryID int64) ([]Team, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.repository_id, t.slug, t.display_name, t.color, t.source,
		       COALESCE((SELECT count(*)::int FROM team_file_ownership tfo WHERE tfo.team_id = t.id), 0)
		FROM teams t WHERE t.repository_id = $1 ORDER BY t.display_name
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.RepositoryID, &t.Slug, &t.DisplayName, &t.Color, &t.Source, &t.FileCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Service) TeamFiles(ctx context.Context, repositoryID int64, teamID uuid.UUID) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT f.relative_path FROM team_file_ownership tfo
		JOIN files f ON f.id = tfo.file_id
		WHERE tfo.repository_id = $1 AND tfo.team_id = $2
		ORDER BY f.relative_path
	`, repositoryID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

func (s *Service) BoundaryViolations(ctx context.Context, repositoryID int64) ([]BoundaryViolation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT f1.relative_path, f2.relative_path, t1.slug, t2.slug
		FROM file_dependencies d
		JOIN files f1 ON f1.id = d.from_file_id
		JOIN files f2 ON f2.id = d.to_file_id
		JOIN team_file_ownership o1 ON o1.file_id = f1.id AND o1.repository_id = d.repository_id
		JOIN team_file_ownership o2 ON o2.file_id = f2.id AND o2.repository_id = d.repository_id
		JOIN teams t1 ON t1.id = o1.team_id
		JOIN teams t2 ON t2.id = o2.team_id
		WHERE d.repository_id = $1 AND t1.id <> t2.id
		LIMIT 200
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BoundaryViolation
	for rows.Next() {
		var v BoundaryViolation
		if err := rows.Scan(&v.SourceFile, &v.TargetFile, &v.SourceTeam, &v.TargetTeam); err != nil {
			return nil, err
		}
		v.Message = fmt.Sprintf("%s (%s) imports %s (%s)", v.SourceFile, v.SourceTeam, v.TargetFile, v.TargetTeam)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Service) OwnershipGaps(ctx context.Context, repositoryID int64) ([]OwnershipGap, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT f.relative_path FROM files f
		WHERE f.repository_id = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM team_file_ownership tfo WHERE tfo.file_id = f.id AND tfo.repository_id = f.repository_id
		  )
		ORDER BY f.relative_path LIMIT 100
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OwnershipGap
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, OwnershipGap{FilePath: p, Message: "no team ownership assigned"})
	}
	return out, rows.Err()
}

// UpsertFromCodeowners parses CODEOWNERS lines and assigns teams to files (simplified).
func (s *Service) UpsertFromCodeowners(ctx context.Context, repositoryID int64, content string) error {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pattern := strings.TrimPrefix(parts[0], "/")
		for _, owner := range parts[1:] {
			slug := strings.TrimPrefix(owner, "@")
			if slug == "" {
				continue
			}
			teamID, err := s.ensureTeam(ctx, repositoryID, slug)
			if err != nil {
				return err
			}
			rows, err := s.pool.Query(ctx, `SELECT id, relative_path FROM files WHERE repository_id = $1`, repositoryID)
			if err != nil {
				return err
			}
			for rows.Next() {
				var fid int64
				var path string
				if err := rows.Scan(&fid, &path); err != nil {
					rows.Close()
					return err
				}
				if matchTeamPattern(pattern, path) {
					_, _ = s.pool.Exec(ctx, `
						INSERT INTO team_file_ownership (repository_id, team_id, file_id)
						VALUES ($1, $2, $3)
						ON CONFLICT DO NOTHING
					`, repositoryID, teamID, fid)
				}
			}
			rows.Close()
		}
	}
	return nil
}

func (s *Service) ensureTeam(ctx context.Context, repositoryID int64, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO teams (repository_id, slug, display_name, source)
		VALUES ($1, $2, $3, 'codeowners')
		ON CONFLICT (repository_id, slug) DO UPDATE SET display_name = EXCLUDED.display_name
		RETURNING id
	`, repositoryID, slug, slug).Scan(&id)
	return id, err
}

func matchTeamPattern(pattern, path string) bool {
	path = strings.TrimPrefix(path, "./")
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}
