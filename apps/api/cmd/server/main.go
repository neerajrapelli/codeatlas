package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"codeatlas/apps/api/internal/ai"
	"codeatlas/apps/api/internal/archintel"
	"codeatlas/apps/api/internal/blastradius"
	"codeatlas/apps/api/internal/driftdetector"
	"codeatlas/apps/api/internal/livingdocs"
	"codeatlas/apps/api/internal/mcp"
	"codeatlas/apps/api/internal/onboarding"
	"codeatlas/apps/api/internal/teams"
	"codeatlas/apps/api/internal/ai/providers"
	anthropicprovider "codeatlas/apps/api/internal/ai/providers/anthropic"
	geminiprovider "codeatlas/apps/api/internal/ai/providers/gemini"
	hfprovider "codeatlas/apps/api/internal/ai/providers/huggingface"
	localprovider "codeatlas/apps/api/internal/ai/providers/local"
	openaiprovider "codeatlas/apps/api/internal/ai/providers/openai"
	openrouterprovider "codeatlas/apps/api/internal/ai/providers/openrouter"
	"codeatlas/apps/api/internal/config"
	"codeatlas/apps/api/internal/db"
	"codeatlas/apps/api/internal/telemetry"
	"codeatlas/apps/api/internal/github"
	"codeatlas/apps/api/internal/httpserver"
	"codeatlas/apps/api/internal/indexer"
	"codeatlas/apps/api/internal/ingestprogress"
	"codeatlas/apps/api/internal/ingestion"
	"codeatlas/apps/api/internal/jobqueue"
	"codeatlas/apps/api/internal/llm"
	"codeatlas/apps/api/internal/repoingest"
	"codeatlas/apps/api/internal/socio"
	"codeatlas/apps/api/internal/vcsauth"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	bootCtx := context.Background()
	otelShutdown, err := telemetry.Init(bootCtx, telemetry.Config{
		ServiceName:  cfg.OTELServiceName,
		OTLPEndpoint: cfg.OTELExporterEndpoint,
		Disabled:     cfg.OTELDisabled,
	})
	if err != nil {
		slog.Error("otel_init_failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := telemetry.ShutdownTimeout(context.Background(), otelShutdown, 5*time.Second); err != nil {
			slog.Warn("otel_shutdown_failed", "error", err)
		}
	}()

	pool, err := db.NewPool(bootCtx, cfg.DatabaseURL, true)
	if err != nil {
		slog.Error("db_connect_failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.MigrateDir(bootCtx, pool, cfg.MigrationsDir); err != nil {
		slog.Error("db_migrate_failed", "error", err, "dir", cfg.MigrationsDir)
		os.Exit(1)
	}

	var aiService *ai.Service
	var embedClient llm.Embedder
	embeddingsEnabled := strings.TrimSpace(cfg.OpenAIAPIKey) != ""
	providerManager := providers.NewManager(providers.ProviderName(cfg.AIDefaultProvider), []providers.ProviderName{
		providers.ProviderOpenAI, providers.ProviderOpenRouter, providers.ProviderAnthropic, providers.ProviderGemini, providers.ProviderHuggingFace, providers.ProviderLocal,
	}, logger)
	providerManager.Register(localprovider.New())
	providerManager.Register(anthropicprovider.New())
	providerManager.Register(geminiprovider.New())
	providerManager.Register(hfprovider.New())
	providerManager.Register(openrouterprovider.New())
	if embeddingsEnabled {
		openaiAdapter := openaiprovider.New(cfg.OpenAIAPIKey, cfg.OpenAIChatModel, cfg.OpenAIEmbeddingModel)
		providerManager.Register(openaiAdapter)
		embedClient = llm.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIChatModel, cfg.OpenAIEmbeddingModel)
	} else {
		logger.Info("indexing_embeddings_disabled", "hint", "set OPENAI_API_KEY to enable semantic embeddings; code graph works without it")
	}
	socioStore := socio.NewStore(pool)
	retriever := ai.NewRetriever(pool, socioStore)
	aiService = ai.NewService(retriever, providers.ProviderName(cfg.AIDefaultProvider), cfg.AIDefaultModel, cfg.AIContextTokenBudget, providerManager, logger)

	scanner := indexer.NewMultiLanguageScanner()
	scanner.MaxFileBytes = cfg.MaxIndexFileBytes
	scanner.MaxFiles = cfg.MaxIndexFiles
	scanner.MaxRepoBytes = cfg.MaxRepoBytes
	idxService := indexer.New(
		scanner,
		indexer.NewTreeSitterParser(),
		indexer.NewPostgresStore(pool, embedClient, cfg.EmbeddingMaxPerRepo),
		logger,
	)
	ghClient := github.NewClient(cfg.GitHubToken, logger)
	socioIngest := ingestion.NewService(socioStore, ghClient, logger)
	socioQuery := socio.NewQueryService(socioStore)
	archQuery := archintel.NewQueryService(pool)

	progressBus := ingestprogress.NewEventBus(cfg.RedisURL, logger)
	ingestQueue := jobqueue.NewPostgresQueue(pool)
	blastSvc := blastradius.NewService(pool)
	driftEngine := driftdetector.NewEngine(pool)
	driftStore := driftdetector.NewStore(pool)
	teamsSvc := teams.NewService(pool)
	var vcsSvc *vcsauth.Service
	var cloneResolver repoingest.CloneResolver
	var vcsStore *vcsauth.Store
	if cfg.TokenEncryptionKey != "" {
		cipher, err := vcsauth.NewCipher(cfg.TokenEncryptionKey)
		if err != nil {
			slog.Error("token_encryption_misconfigured", "error", err)
			os.Exit(1)
		}
		vcsStore = vcsauth.NewStore(pool, cipher)
		vcsSvc = vcsauth.NewService(vcsStore, cfg)
		cloneResolver = repoingest.NewVCSCloneResolver(vcsSvc)
	}
	ingestService := repoingest.NewService(
		cfg.WorkspaceRoot,
		repoingest.NewStore(pool),
		idxService,
		socioIngest,
		ingestQueue,
		progressBus,
		driftEngine,
		teamsSvc,
		logger,
		cfg.ZipMaxBytes,
		cfg.ZipMaxFiles,
		cfg.IndexerParseWorkers,
		cloneResolver,
		vcsStore,
		embeddingsEnabled,
	)
	ingestRunner := repoingest.NewRunner(ingestService)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jobqueue.StartWorker(ctx, ingestQueue, ingestRunner, cfg.IngestWorkerConcurrency, logger)

	mcpServer := mcp.NewServer(pool, blastSvc, driftEngine, socioQuery)
	onboardingSvc := onboarding.NewService(aiService)
	livingDocsSvc := livingdocs.NewService(pool)
	srv := httpserver.New(cfg, pool, aiService, ingestService, ingestQueue, socioQuery, blastSvc, driftEngine, driftStore, mcpServer, teamsSvc, onboardingSvc, livingDocsSvc, progressBus, vcsSvc, archQuery)

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
