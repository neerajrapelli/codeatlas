package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"codeatlas/apps/api/internal/config"
	"codeatlas/apps/api/internal/db"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, false)
	if err != nil {
		slog.Error("db_connect_failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.MigrateDir(ctx, pool, cfg.MigrationsDir); err != nil {
		slog.Error("db_migrate_failed", "error", err, "dir", cfg.MigrationsDir)
		os.Exit(1)
	}
	slog.Info("db_migrate_ok", "dir", cfg.MigrationsDir)
}
