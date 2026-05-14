package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"codeatlas/apps/api/internal/config"
	"codeatlas/apps/api/internal/db"
	"codeatlas/apps/api/internal/indexer"
	"codeatlas/apps/api/internal/llm"
)

func main() {
	repoPath := flag.String("repo", "", "absolute or relative path to a local TypeScript repository")
	repoName := flag.String("name", "", "optional repository display name")
	flag.Parse()

	if *repoPath == "" {
		fmt.Fprintln(os.Stderr, "missing required -repo argument")
		os.Exit(2)
	}

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(2)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db_connect_failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.MigrateDir(ctx, pool, cfg.MigrationsDir); err != nil {
		slog.Error("db_migrate_failed", "error", err, "dir", cfg.MigrationsDir)
		os.Exit(1)
	}

	var embedder indexer.Embedder
	if cfg.OpenAIAPIKey != "" {
		client := llm.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIChatModel, cfg.OpenAIEmbeddingModel)
		embedder = client
	} else {
		embedder = llm.NewLocalClient()
	}

	svc := indexer.New(
		indexer.NewTypeScriptFileScanner(),
		indexer.NewTreeSitterTypeScriptParser(),
		indexer.NewPostgresStore(pool, embedder),
		logger,
	)
	result, err := svc.Run(ctx, indexer.Request{RepositoryPath: *repoPath, RepositoryName: *repoName})
	if err != nil {
		slog.Error("indexing_failed", "error", err)
		os.Exit(1)
	}

	slog.Info("indexing_complete",
		"repository_id", result.RepositoryID,
		"files", result.Files,
		"symbols", result.Symbols,
		"imports", result.Imports,
		"exports", result.Exports,
		"file_dependencies", result.FileDependencies,
	)
}
