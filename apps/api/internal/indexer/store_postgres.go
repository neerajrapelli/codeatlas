package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"codeatlas/apps/api/internal/tenant"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type PostgresStore struct {
	pool                *pgxpool.Pool
	embedder            Embedder
	embeddingMaxPerRepo int
}

func NewPostgresStore(pool *pgxpool.Pool, embedder Embedder, embeddingMaxPerRepo int) *PostgresStore {
	if embeddingMaxPerRepo < 0 {
		embeddingMaxPerRepo = 0
	}
	return &PostgresStore{
		pool:                pool,
		embedder:            embedder,
		embeddingMaxPerRepo: embeddingMaxPerRepo,
	}
}

func (s *PostgresStore) canEmbed(stats *PersistStats) bool {
	if s.embedder == nil {
		return false
	}
	if s.embeddingMaxPerRepo <= 0 {
		return true
	}
	return stats.Embeddings < s.embeddingMaxPerRepo
}

func (s *PostgresStore) skipEmbedding(stats *PersistStats) {
	stats.EmbeddingsSkipped++
}

func countPlannedEmbeddings(files []IndexedFile) int {
	n := 0
	for _, item := range files {
		n++
		n += len(item.Imports)
		n += len(item.Symbols)
	}
	return n
}

type embedTarget struct {
	entityType          string
	fileID, symbolID, importID int64
	content             string
	filePath            string
}

const graphPersistBatchSize = 50

func (s *PostgresStore) UpsertRepositoryGraph(ctx context.Context, req PersistRequest) (PersistStats, error) {
	tid := tenant.Normalize(req.TenantID)
	repoID, err := s.resolveRepositoryID(ctx, req, tid)
	if err != nil {
		return PersistStats{}, err
	}
	if err := s.clearGraphForRepository(ctx, repoID); err != nil {
		return PersistStats{}, err
	}

	totalFiles := len(req.IndexedFiles)
	if req.OnProgress != nil {
		req.OnProgress(ProgressEvent{
			Stage:    StageBuildingGraph,
			Progress: 0,
			Files:    0,
			Metadata: map[string]any{"totalFiles": totalFiles, "phase": "graph"},
		})
	}

	fileIDs := make(map[string]int64, totalFiles)
	stats := PersistStats{RepositoryID: repoID}
	var embedQueue []embedTarget

	for batchStart := 0; batchStart < totalFiles; batchStart += graphPersistBatchSize {
		batchEnd := batchStart + graphPersistBatchSize
		if batchEnd > totalFiles {
			batchEnd = totalFiles
		}
		if err := s.persistGraphBatch(ctx, req, tid, repoID, req.IndexedFiles[batchStart:batchEnd], fileIDs, &stats, &embedQueue, totalFiles); err != nil {
			return PersistStats{}, err
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PersistStats{}, fmt.Errorf("begin deps tx: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, item := range req.IndexedFiles {
		fromID := fileIDs[item.File.RelativePath]
		for _, dep := range item.ResolvedDependencies {
			toID, ok := fileIDs[dep]
			if !ok {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO file_dependencies(repository_id, from_file_id, to_file_id, tenant_id)
				VALUES($1,$2,$3,$4)
				ON CONFLICT(repository_id, from_file_id, to_file_id) DO NOTHING
			`, repoID, fromID, toID, tid); err != nil {
				return PersistStats{}, fmt.Errorf("insert file dependency %s->%s: %w", item.File.RelativePath, dep, err)
			}
			stats.FileDependencies++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return PersistStats{}, fmt.Errorf("commit deps tx: %w", err)
	}

	if req.OnProgress != nil {
		req.OnProgress(ProgressEvent{
			Stage:             StageBuildingGraph,
			Progress:          100,
			Files:             stats.Files,
			Symbols:           stats.Symbols,
			Edges:             stats.FileDependencies,
			Metadata:          map[string]any{"totalFiles": totalFiles, "phase": "graph"},
		})
	}

	if s.embedder != nil && len(embedQueue) > 0 {
		s.runEmbeddingPass(ctx, req, repoID, tid, &stats, embedQueue)
	}
	return stats, nil
}

func (s *PostgresStore) resolveRepositoryID(ctx context.Context, req PersistRequest, tid string) (int64, error) {
	if req.RepositoryID > 0 {
		return req.RepositoryID, nil
	}
	repoName := req.RepositoryName
	if repoName == "" {
		repoName = req.RepositoryPath
	}
	var repoID int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO repositories(name, root_path, tenant_id)
		VALUES ($1, $2, $3)
		ON CONFLICT(root_path)
		DO UPDATE SET name=EXCLUDED.name, tenant_id=EXCLUDED.tenant_id, updated_at=NOW()
		RETURNING id
	`, repoName, req.RepositoryPath, tid).Scan(&repoID)
	if err != nil {
		return 0, fmt.Errorf("upsert repository: %w", err)
	}
	return repoID, nil
}

func (s *PostgresStore) clearGraphForRepository(ctx context.Context, repoID int64) error {
	tables := []string{
		"file_dependencies",
		"entity_embeddings",
		"file_imports",
		"file_exports",
		"symbols",
		"files",
	}
	for _, table := range tables {
		if _, err := s.pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE repository_id=$1", table), repoID); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	return nil
}

func (s *PostgresStore) persistGraphBatch(
	ctx context.Context,
	req PersistRequest,
	tid string,
	repoID int64,
	batch []IndexedFile,
	fileIDs map[string]int64,
	stats *PersistStats,
	embedQueue *[]embedTarget,
	totalFiles int,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin graph batch tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range batch {
		var fileID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO files(repository_id, relative_path, tenant_id)
			VALUES ($1, $2, $3)
			RETURNING id
		`, repoID, item.File.RelativePath, tid).Scan(&fileID); err != nil {
			return fmt.Errorf("insert file %s: %w", item.File.RelativePath, err)
		}
		fileIDs[item.File.RelativePath] = fileID
		stats.Files++
		if req.OnProgress != nil {
			req.OnProgress(ProgressEvent{
				Stage:    StageBuildingGraph,
				Progress: (float64(stats.Files) / float64(maxInt(1, totalFiles))) * 100,
				Files:    stats.Files,
				Symbols:  stats.Symbols,
				Edges:    stats.FileDependencies,
				Metadata: map[string]any{"totalFiles": totalFiles, "currentFile": item.File.RelativePath, "phase": "graph"},
			})
		}

		if s.embedder != nil {
			fileSummary := fmt.Sprintf("file=%s imports=%v exports=%v symbols=%d", item.File.RelativePath, collectImportPaths(item.Imports), collectExportNames(item.Exports), len(item.Symbols))
			*embedQueue = append(*embedQueue, embedTarget{
				entityType: "file",
				fileID:     fileID,
				content:    fileSummary,
				filePath:   item.File.RelativePath,
			})
		}

		for _, imp := range item.Imports {
			var importID int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO file_imports(repository_id, file_id, module_path, is_type_only, is_external, tenant_id)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id
			`, repoID, fileID, imp.ModulePath, imp.TypeOnly, !isLocalImport(imp.ModulePath), tid).Scan(&importID); err != nil {
				return fmt.Errorf("insert import %s: %w", item.File.RelativePath, err)
			}
			stats.Imports++
			if s.embedder != nil {
				importText := fmt.Sprintf("file=%s imports=%s type_only=%t", item.File.RelativePath, imp.ModulePath, imp.TypeOnly)
				*embedQueue = append(*embedQueue, embedTarget{
					entityType: "import",
					fileID:     fileID,
					importID:   importID,
					content:    importText,
					filePath:   item.File.RelativePath,
				})
			}
		}

		for _, ex := range item.Exports {
			if _, err := tx.Exec(ctx, `
				INSERT INTO file_exports(repository_id, file_id, export_name, source_path, tenant_id)
				VALUES ($1, $2, $3, NULLIF($4, ''), $5)
			`, repoID, fileID, ex.Name, ex.SourcePath, tid); err != nil {
				return fmt.Errorf("insert export %s: %w", item.File.RelativePath, err)
			}
			stats.Exports++
		}

		for _, sym := range item.Symbols {
			var symbolID int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO symbols(
					repository_id, file_id, name, kind, exported,
					start_line, start_col, end_line, end_col, tenant_id
				)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
				RETURNING id
			`, repoID, fileID, sym.Name, sym.Kind, sym.Exported, sym.StartLine, sym.StartCol, sym.EndLine, sym.EndCol, tid).Scan(&symbolID); err != nil {
				return fmt.Errorf("insert symbol %s: %w", item.File.RelativePath, err)
			}
			stats.Symbols++
			if s.embedder != nil {
				symbolText := fmt.Sprintf("file=%s symbol=%s kind=%s exported=%t", item.File.RelativePath, sym.Name, sym.Kind, sym.Exported)
				*embedQueue = append(*embedQueue, embedTarget{
					entityType: "symbol",
					fileID:     fileID,
					symbolID:   symbolID,
					content:    symbolText,
					filePath:   item.File.RelativePath,
				})
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit graph batch tx: %w", err)
	}
	return nil
}

func (s *PostgresStore) runEmbeddingPass(ctx context.Context, req PersistRequest, repoID int64, tenantID string, stats *PersistStats, queue []embedTarget) {
	planned := len(queue)
	total := planned
	if s.embeddingMaxPerRepo > 0 && total > s.embeddingMaxPerRepo {
		total = s.embeddingMaxPerRepo
	}
	if total < 1 {
		total = 1
	}
	capActive := s.embeddingMaxPerRepo > 0 && planned > s.embeddingMaxPerRepo

	for _, target := range queue {
		if !s.canEmbed(stats) {
			s.skipEmbedding(stats)
			continue
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			slog.Warn("embedding_tx_begin_failed", "repository_id", repoID, "error", err)
			continue
		}
		err = insertEmbedding(ctx, tx, repoID, tenantID, target.entityType, target.fileID, target.symbolID, target.importID, target.content, s.embedder)
		if err != nil {
			_ = tx.Rollback(ctx)
			slog.Warn("embedding_skipped", "repository_id", repoID, "file", target.filePath, "entity", target.entityType, "error", err)
			continue
		}
		if err := tx.Commit(ctx); err != nil {
			slog.Warn("embedding_tx_commit_failed", "repository_id", repoID, "error", err)
			continue
		}
		stats.Embeddings++
		if req.OnProgress == nil {
			continue
		}
		embPct := (float64(stats.Embeddings) / float64(total)) * 100
		if embPct > 100 {
			embPct = 100
		}
		meta := map[string]any{
			"currentFile":    target.filePath,
			"embeddingTotal": total,
			"embeddingsDone": stats.Embeddings,
			"phase":          "embeddings",
		}
		if capActive {
			meta["embeddingPlanned"] = planned
			meta["embeddingCap"] = s.embeddingMaxPerRepo
			meta["embeddingsSkipped"] = stats.EmbeddingsSkipped
		}
		req.OnProgress(ProgressEvent{
			Stage:      StageGeneratingEmbeddings,
			Progress:   embPct,
			Files:      stats.Files,
			Symbols:    stats.Symbols,
			Edges:      stats.FileDependencies,
			Embeddings: stats.Embeddings,
			Metadata:   meta,
		})
	}
}

func isLocalImport(path string) bool {
	if len(path) == 0 {
		return false
	}
	return path[0] == '.'
}

const embeddingCallTimeout = 90 * time.Second

func insertEmbedding(ctx context.Context, tx pgx.Tx, repositoryID int64, tenantID, entityType string, fileID, symbolID, importID int64, content string, embedder Embedder) error {
	embedCtx, cancel := context.WithTimeout(ctx, embeddingCallTimeout)
	defer cancel()
	vector, err := embedder.Embed(embedCtx, content)
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
		INSERT INTO entity_embeddings(repository_id, entity_type, file_id, symbol_id, import_id, content, embedding, tenant_id)
		VALUES ($1, $2, NULLIF($3,0), NULLIF($4,0), NULLIF($5,0), $6, $7, $8)
	`, repositoryID, entityType, fileID, symbolID, importID, sanitizeContent(content), pgvector.NewVector(vector), tenantID)
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
