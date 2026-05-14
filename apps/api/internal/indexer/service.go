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

	indexed := make([]IndexedFile, 0, len(files))
	for i, file := range files {
		parsed, err := s.parser.Parse(file)
		if err != nil {
			return Result{}, fmt.Errorf("parse file %s: %w", file.RelativePath, err)
		}
		indexed = append(indexed, IndexedFile{ParsedFile: parsed, ResolvedDependencies: resolveDependencies(repoAbs, file, parsed.Imports, known)})
		if req.OnProgress != nil && (i == len(files)-1 || i%10 == 0) {
			req.OnProgress(ProgressEvent{
				Stage:    StageParsing,
				Progress: (float64(i+1) / float64(maxInt(1, len(files)))) * 100,
				Files:    i + 1,
				Metadata: map[string]any{"totalFiles": len(files), "currentFile": file.RelativePath},
			})
		}
	}

	stats, err := s.store.UpsertRepositoryGraph(ctx, PersistRequest{
		RepositoryPath: repoAbs,
		RepositoryName: req.RepositoryName,
		IndexedFiles:   indexed,
		OnProgress:     req.OnProgress,
	})
	if err != nil {
		return Result{}, fmt.Errorf("persist graph: %w", err)
	}

	result := Result{
		RepositoryID:     stats.RepositoryID,
		Files:            stats.Files,
		Symbols:          stats.Symbols,
		Imports:          stats.Imports,
		Exports:          stats.Exports,
		FileDependencies: stats.FileDependencies,
		Embeddings:       stats.Embeddings,
		Duration:         time.Since(started),
	}
	s.logger.Info("index_complete", "repo_id", result.RepositoryID, "files", result.Files, "symbols", result.Symbols, "imports", result.Imports, "duration_ms", result.Duration.Milliseconds())
	return result, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
