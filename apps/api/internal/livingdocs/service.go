package livingdocs

import (
	"context"
	"fmt"
	"strings"

	"codeatlas/apps/api/internal/graphhierarchy"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) C4Diagram(ctx context.Context, repositoryID int64, level string) (string, error) {
	if level == "" {
		level = "container"
	}
	layer, err := graphhierarchy.BuildLayer(ctx, s.pool, repositoryID, "", "", 0)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("```mermaid\nflowchart TB\n")
	if level == "context" {
		b.WriteString("  repo[Code Repository]\n")
		b.WriteString("  user[Developer]\n")
		b.WriteString("  user --> repo\n")
	} else {
		for _, c := range layer.Clusters {
			id := sanitizeID(c.ID)
			b.WriteString(fmt.Sprintf("  %s[%s]\n", id, c.Label))
		}
		for _, e := range layer.Edges {
			b.WriteString(fmt.Sprintf("  %s --> %s\n", sanitizeID(e.From), sanitizeID(e.To)))
		}
	}
	b.WriteString("```\n")
	return b.String(), nil
}

func (s *Service) ADRs(ctx context.Context, repositoryID int64) ([]map[string]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT summary, signal_type, extracted_at::text
		FROM architecture_signals
		WHERE repository_id = $1 AND signal_type = 'architectural_decision'
		ORDER BY extracted_at DESC LIMIT 20
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]string
	for rows.Next() {
		var summary, stype, at string
		if err := rows.Scan(&summary, &stype, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]string{
			"title": summary, "status": "accepted", "date": at, "type": stype,
		})
	}
	return out, rows.Err()
}

func (s *Service) Diff(ctx context.Context, repositoryID int64, since string) (map[string]any, error) {
	var files, deps int
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM files WHERE repository_id = $1`, repositoryID).Scan(&files)
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM file_dependencies WHERE repository_id = $1`, repositoryID).Scan(&deps)
	return map[string]any{
		"since": since, "files": files, "dependencies": deps,
		"summary": fmt.Sprintf("%d files, %d dependency edges indexed", files, deps),
	}, nil
}

func (s *Service) ExportMarkdown(ctx context.Context, repositoryID int64) (string, error) {
	c4, err := s.C4Diagram(ctx, repositoryID, "container")
	if err != nil {
		return "", err
	}
	adrs, _ := s.ADRs(ctx, repositoryID)
	var b strings.Builder
	b.WriteString("# Architecture Documentation\n\n")
	b.WriteString("## C4 Container Diagram\n\n")
	b.WriteString(c4)
	b.WriteString("\n## Architecture Decision Records\n\n")
	if len(adrs) == 0 {
		b.WriteString("_No ADR signals ingested yet._\n")
	} else {
		for _, a := range adrs {
			b.WriteString(fmt.Sprintf("- **%s** (%s)\n", a["title"], a["date"]))
		}
	}
	return b.String(), nil
}

func sanitizeID(id string) string {
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, ".", "_")
	if id == "" {
		return "node"
	}
	return id
}
