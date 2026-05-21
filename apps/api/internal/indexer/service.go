package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"
)

type Service struct {
	scanner Scanner
	parser  Parser
	store   Store
	logger  *slog.Logger
}

func New(scanner Scanner, parser Parser, store Store, logger *slog.Logger) *Service {
	return &Service{scanner: scanner, parser: parser, store: store, logger: logger}
}

func (s *Service) Run(ctx context.Context, req Request) (Result, error) {
	started := time.Now()
	repoAbs, err := filepath.Abs(req.RepositoryPath)
	if err != nil {
		return Result{}, fmt.Errorf("resolve repository path: %w", err)
	}

	s.logger.Info("index_scan_start", "repo_path", repoAbs)
	files, err := s.scanner.Scan(repoAbs)
	if err != nil {
		return Result{}, fmt.Errorf("scan repository: %w", err)
	}
	known := make(map[string]struct{}, len(files))
	for _, file := range files {
		known[file.RelativePath] = struct{}{}
	}

	parsedFiles, err := parseFilesParallel(ctx, files, newParserFactory(s.parser), req.ParseWorkers, func(done, total int) {
		if req.OnProgress == nil {
			return
		}
		req.OnProgress(ProgressEvent{
			Stage:    StageParsing,
			Progress: (float64(done) / float64(maxInt(1, total))) * 100,
			Files:    done,
			Metadata: map[string]any{"totalFiles": total},
		})
	})
	if err != nil {
		return Result{}, err
	}
	if req.OnProgress != nil {
		req.OnProgress(ProgressEvent{
			Stage:    StageBuildingGraph,
			Progress: 0,
			Files:    len(parsedFiles),
			Metadata: map[string]any{"totalFiles": len(files), "phase": "graph"},
		})
	}
	indexed := make([]IndexedFile, 0, len(parsedFiles))
	for i, file := range files {
		parsed := parsedFiles[i].ParsedFile
		indexed = append(indexed, IndexedFile{
			ParsedFile:           parsed,
			ResolvedDependencies: resolveDependencies(repoAbs, file, parsed.Imports, known),
		})
	}

	stats, err := s.store.UpsertRepositoryGraph(ctx, PersistRequest{
		RepositoryID:   req.RepositoryID,
		RepositoryPath: repoAbs,
		RepositoryName: req.RepositoryName,
		TenantID:       req.TenantID,
		IndexedFiles:   indexed,
		OnProgress:     req.OnProgress,
	})
	if err != nil {
		return Result{}, fmt.Errorf("persist graph: %w", err)
	}

	result := Result{
		RepositoryID:      stats.RepositoryID,
		Files:             stats.Files,
		Symbols:           stats.Symbols,
		Imports:           stats.Imports,
		Exports:           stats.Exports,
		FileDependencies:  stats.FileDependencies,
		Embeddings:        stats.Embeddings,
		EmbeddingsSkipped: stats.EmbeddingsSkipped,
		Duration:          time.Since(started),
	}
	s.logger.Info("index_complete",
		"repo_id", result.RepositoryID,
		"files", result.Files,
		"symbols", result.Symbols,
		"imports", result.Imports,
		"embeddings", result.Embeddings,
		"embeddings_skipped", result.EmbeddingsSkipped,
		"duration_ms", result.Duration.Milliseconds(),
	)
	return result, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
