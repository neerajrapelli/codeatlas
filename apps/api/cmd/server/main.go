package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeatlas/apps/api/internal/ai"
	"codeatlas/apps/api/internal/ai/providers"
	anthropicprovider "codeatlas/apps/api/internal/ai/providers/anthropic"
	geminiprovider "codeatlas/apps/api/internal/ai/providers/gemini"
	hfprovider "codeatlas/apps/api/internal/ai/providers/huggingface"
	localprovider "codeatlas/apps/api/internal/ai/providers/local"
	openaiprovider "codeatlas/apps/api/internal/ai/providers/openai"
	openrouterprovider "codeatlas/apps/api/internal/ai/providers/openrouter"
	"codeatlas/apps/api/internal/config"
	"codeatlas/apps/api/internal/db"
	"codeatlas/apps/api/internal/httpserver"
	"codeatlas/apps/api/internal/indexer"
	"codeatlas/apps/api/internal/llm"
	"codeatlas/apps/api/internal/repoingest"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	bootCtx := context.Background()
	pool, err := db.NewPool(bootCtx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db_connect_failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.MigrateDir(bootCtx, pool, cfg.MigrationsDir); err != nil {
		slog.Error("db_migrate_failed", "error", err)
		os.Exit(1)
	}

	var aiService *ai.Service
	var embedClient llm.Embedder
	providerManager := providers.NewManager(providers.ProviderName(cfg.AIDefaultProvider), []providers.ProviderName{
		providers.ProviderOpenAI, providers.ProviderOpenRouter, providers.ProviderAnthropic, providers.ProviderGemini, providers.ProviderHuggingFace, providers.ProviderLocal,
	}, logger)
	providerManager.Register(localprovider.New())
	providerManager.Register(anthropicprovider.New())
	providerManager.Register(geminiprovider.New())
	providerManager.Register(hfprovider.New())
	providerManager.Register(openrouterprovider.New())
	if cfg.OpenAIAPIKey != "" {
		openaiAdapter := openaiprovider.New(cfg.OpenAIAPIKey, cfg.OpenAIChatModel, cfg.OpenAIEmbeddingModel)
		providerManager.Register(openaiAdapter)
		embedClient = llm.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIChatModel, cfg.OpenAIEmbeddingModel)
	} else {
		embedClient = llm.NewLocalClient()
	}
	retriever := ai.NewRetriever(pool)
	aiService = ai.NewService(retriever, providers.ProviderName(cfg.AIDefaultProvider), cfg.AIDefaultModel, cfg.AIContextTokenBudget, providerManager, logger)

	idxService := indexer.New(
		indexer.NewTypeScriptFileScanner(),
		indexer.NewTreeSitterTypeScriptParser(),
		indexer.NewPostgresStore(pool, embedClient),
		logger,
	)
	ingestService := repoingest.NewService(
		cfg.WorkspaceRoot,
		repoingest.NewStore(pool),
		idxService,
		logger,
		cfg.ZipMaxBytes,
		cfg.ZipMaxFiles,
	)

	srv := httpserver.New(cfg, pool, aiService, ingestService)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("http_listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server_crashed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful_shutdown_failed", "error", err)
		os.Exit(1)
	}
}
