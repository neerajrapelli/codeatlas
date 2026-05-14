package indexer

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type PostgresStore struct {
	pool     *pgxpool.Pool
	embedder Embedder
}

func NewPostgresStore(pool *pgxpool.Pool, embedder Embedder) *PostgresStore {
	return &PostgresStore{pool: pool, embedder: embedder}
}

func (s *PostgresStore) UpsertRepositoryGraph(ctx context.Context, req PersistRequest) (PersistStats, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PersistStats{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	repoName := req.RepositoryName
	if repoName == "" {
		repoName = req.RepositoryPath
	}

	var repoID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO repositories(name, root_path)
		VALUES ($1, $2)
		ON CONFLICT(root_path)
		DO UPDATE SET name=EXCLUDED.name, updated_at=NOW()
		RETURNING id
	`, repoName, req.RepositoryPath).Scan(&repoID); err != nil {
		return PersistStats{}, fmt.Errorf("upsert repository: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM file_dependencies WHERE repository_id=$1`, repoID); err != nil {
		return PersistStats{}, fmt.Errorf("clear file dependencies: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM entity_embeddings WHERE repository_id=$1`, repoID); err != nil {
		return PersistStats{}, fmt.Errorf("clear embeddings: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM file_imports WHERE repository_id=$1`, repoID); err != nil {
		return PersistStats{}, fmt.Errorf("clear imports: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM file_exports WHERE repository_id=$1`, repoID); err != nil {
		return PersistStats{}, fmt.Errorf("clear exports: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM symbols WHERE repository_id=$1`, repoID); err != nil {
		return PersistStats{}, fmt.Errorf("clear symbols: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM files WHERE repository_id=$1`, repoID); err != nil {
		return PersistStats{}, fmt.Errorf("clear files: %w", err)
	}

	fileIDs := make(map[string]int64, len(req.IndexedFiles))
	stats := PersistStats{RepositoryID: repoID}

	for _, item := range req.IndexedFiles {
		var fileID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO files(repository_id, relative_path)
			VALUES ($1, $2)
			RETURNING id
		`, repoID, item.File.RelativePath).Scan(&fileID); err != nil {
			return PersistStats{}, fmt.Errorf("insert file %s: %w", item.File.RelativePath, err)
		}
		fileIDs[item.File.RelativePath] = fileID
		stats.Files++
		if req.OnProgress != nil {
			req.OnProgress(ProgressEvent{
				Stage:    StageBuildingGraph,
				Progress: (float64(stats.Files) / float64(maxInt(1, len(req.IndexedFiles)))) * 100,
				Files:    stats.Files,
				Symbols:  stats.Symbols,
				Edges:    stats.FileDependencies,
				Metadata: map[string]any{"totalFiles": len(req.IndexedFiles), "currentFile": item.File.RelativePath},
			})
		}
		if s.embedder != nil {
			fileSummary := fmt.Sprintf("file=%s imports=%v exports=%v symbols=%d", item.File.RelativePath, collectImportPaths(item.Imports), collectExportNames(item.Exports), len(item.Symbols))
			if err := insertEmbedding(ctx, tx, repoID, "file", fileID, 0, 0, fileSummary, s.embedder); err != nil {
				return PersistStats{}, fmt.Errorf("insert file embedding %s: %w", item.File.RelativePath, err)
			}
			stats.Embeddings++
		}

		for _, imp := range item.Imports {
			var importID int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO file_imports(repository_id, file_id, module_path, is_type_only, is_external)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, repoID, fileID, imp.ModulePath, imp.TypeOnly, !isLocalImport(imp.ModulePath)).Scan(&importID); err != nil {
				return PersistStats{}, fmt.Errorf("insert import %s: %w", item.File.RelativePath, err)
			}
			if s.embedder != nil {
				importText := fmt.Sprintf("file=%s imports=%s type_only=%t", item.File.RelativePath, imp.ModulePath, imp.TypeOnly)
				if err := insertEmbedding(ctx, tx, repoID, "import", fileID, 0, importID, importText, s.embedder); err != nil {
					return PersistStats{}, fmt.Errorf("insert import embedding %s: %w", item.File.RelativePath, err)
				}
				stats.Embeddings++
			}
			stats.Imports++
		}

		for _, ex := range item.Exports {
			if _, err := tx.Exec(ctx, `
				INSERT INTO file_exports(repository_id, file_id, export_name, source_path)
				VALUES ($1, $2, $3, NULLIF($4, ''))
			`, repoID, fileID, ex.Name, ex.SourcePath); err != nil {
				return PersistStats{}, fmt.Errorf("insert export %s: %w", item.File.RelativePath, err)
			}
			stats.Exports++
		}

		for _, sym := range item.Symbols {
			var symbolID int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO symbols(
					repository_id, file_id, name, kind, exported,
					start_line, start_col, end_line, end_col
				)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
				RETURNING id
			`, repoID, fileID, sym.Name, sym.Kind, sym.Exported, sym.StartLine, sym.StartCol, sym.EndLine, sym.EndCol).Scan(&symbolID); err != nil {
				return PersistStats{}, fmt.Errorf("insert symbol %s: %w", item.File.RelativePath, err)
			}
			if s.embedder != nil {
				symbolText := fmt.Sprintf("file=%s symbol=%s kind=%s exported=%t", item.File.RelativePath, sym.Name, sym.Kind, sym.Exported)
				if err := insertEmbedding(ctx, tx, repoID, "symbol", fileID, symbolID, 0, symbolText, s.embedder); err != nil {
					return PersistStats{}, fmt.Errorf("insert symbol embedding %s: %w", item.File.RelativePath, err)
				}
				stats.Embeddings++
				if req.OnProgress != nil {
					req.OnProgress(ProgressEvent{
						Stage:      StageGeneratingEmbeddings,
						Progress:   float64(stats.Embeddings),
						Files:      stats.Files,
						Symbols:    stats.Symbols,
						Edges:      stats.FileDependencies,
						Embeddings: stats.Embeddings,
						Metadata:   map[string]any{"currentFile": item.File.RelativePath},
					})
				}
			}
			stats.Symbols++
		}
	}

	for _, item := range req.IndexedFiles {
		fromID := fileIDs[item.File.RelativePath]
		for _, dep := range item.ResolvedDependencies {
			toID, ok := fileIDs[dep]
			if !ok {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO file_dependencies(repository_id, from_file_id, to_file_id)
				VALUES($1,$2,$3)
				ON CONFLICT(repository_id, from_file_id, to_file_id) DO NOTHING
			`, repoID, fromID, toID); err != nil {
				return PersistStats{}, fmt.Errorf("insert file dependency %s->%s: %w", item.File.RelativePath, dep, err)
			}
			stats.FileDependencies++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return PersistStats{}, fmt.Errorf("commit tx: %w", err)
	}
	return stats, nil
}


func isLocalImport(path string) bool {
	if len(path) == 0 {
		return false
	}
	return path[0] == '.'
}

func insertEmbedding(ctx context.Context, tx pgx.Tx, repositoryID int64, entityType string, fileID, symbolID, importID int64, content string, embedder Embedder) error {
	vector, err := embedder.Embed(ctx, content)
	if err != nil {
		return err
	}
	if len(vector) == 0 {
		return fmt.Errorf("empty embedding vector")
	}
	if len(vector) > 1536 {
		vector = vector[:1536]
	}
	if len(vector) < 1536 {
		pad := make([]float32, 1536-len(vector))
		vector = append(vector, pad...)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO entity_embeddings(repository_id, entity_type, file_id, symbol_id, import_id, content, embedding)
		VALUES ($1, $2, NULLIF($3,0), NULLIF($4,0), NULLIF($5,0), $6, $7)
	`, repositoryID, entityType, fileID, symbolID, importID, sanitizeContent(content), pgvector.NewVector(vector))
	return err
}

func collectImportPaths(imports []Import) []string {
	out := make([]string, 0, len(imports))
	for _, imp := range imports {
		out = append(out, imp.ModulePath)
	}
	return out
}

func collectExportNames(exports []Export) []string {
	out := make([]string, 0, len(exports))
	for _, ex := range exports {
		if ex.Name != "" {
			out = append(out, ex.Name)
		}
	}
	return out
}

func sanitizeContent(in string) string {
	return strings.TrimSpace(strings.ReplaceAll(in, "\n", " "))
}
