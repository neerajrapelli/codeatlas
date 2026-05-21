package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	AllowedOrigins      []string
	MigrationsDir       string
	WorkspaceRoot       string
	OpenAIAPIKey        string
	OpenAIChatModel     string
	OpenAIEmbeddingModel string
	AnthropicAPIKey     string
	GeminiAPIKey        string
	HuggingFaceAPIKey   string
	OpenRouterAPIKey    string
	AIDefaultProvider   string
	AIDefaultModel      string
	AIContextTokenBudget int
	ZipMaxBytes          int64
	ZipMaxFiles          int
	GitHubToken          string
	JWTSecret            string
	AuthBootstrapSecret  string
	AuthDisabled         bool
	IngestRatePerMinute  int
	ChatRatePerMinute    int
	IngestWorkerConcurrency int
	RedisURL                string
	IndexerParseWorkers     int
	GraphMaxFileLimit       int
	GraphMaxDepth           int
	OTELServiceName         string
	OTELExporterEndpoint    string
	OTELDisabled            bool
	TokenEncryptionKey      string
	PublicAPIBaseURL        string
	FrontendURL             string
	GitHubOAuthClientID     string
	GitHubOAuthClientSecret string
	GitLabOAuthClientID     string
	GitLabOAuthClientSecret string
	BitbucketOAuthClientID  string
	BitbucketOAuthClientSecret string
	MaxIndexFileBytes       int64
	MaxIndexFiles           int
	MaxRepoBytes            int64
	EmbeddingMaxPerRepo     int
}

func Load() Config {
	return Config{
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		MigrationsDir:       getEnv("MIGRATIONS_DIR", "./migrations"),
		WorkspaceRoot:       getEnv("WORKSPACE_ROOT", "./workspace"),
		OpenAIAPIKey:        os.Getenv("OPENAI_API_KEY"),
		OpenAIChatModel:     getEnv("OPENAI_CHAT_MODEL", "gpt-4o-mini"),
		OpenAIEmbeddingModel: getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		AnthropicAPIKey:     os.Getenv("ANTHROPIC_API_KEY"),
		GeminiAPIKey:        os.Getenv("GEMINI_API_KEY"),
		HuggingFaceAPIKey:   os.Getenv("HUGGINGFACE_API_KEY"),
		OpenRouterAPIKey:    os.Getenv("OPENROUTER_API_KEY"),
		AIDefaultProvider:   getEnv("AI_DEFAULT_PROVIDER", "local"),
		AIDefaultModel:      getEnv("AI_DEFAULT_MODEL", "local-default"),
		AIContextTokenBudget: parseInt(getEnv("AI_CONTEXT_TOKEN_BUDGET", "7000"), 7000),
		ZipMaxBytes:          parseInt64(getEnv("ZIP_MAX_BYTES", "104857600"), 104857600),
		ZipMaxFiles:          parseInt(getEnv("ZIP_MAX_FILES", "5000"), 5000),
		GitHubToken:          os.Getenv("GITHUB_TOKEN"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		AuthBootstrapSecret:  os.Getenv("AUTH_BOOTSTRAP_SECRET"),
		AuthDisabled:         getEnv("AUTH_DISABLED", "false") == "true",
		IngestRatePerMinute:  parseInt(getEnv("INGEST_RATE_PER_MINUTE", "6"), 6),
		ChatRatePerMinute:    parseInt(getEnv("CHAT_RATE_PER_MINUTE", "30"), 30),
		IngestWorkerConcurrency: parseInt(getEnv("INGEST_WORKER_CONCURRENCY", "2"), 2),
		RedisURL:                os.Getenv("REDIS_URL"),
		IndexerParseWorkers:     parseInt(getEnv("INDEXER_PARSE_WORKERS", "8"), 8),
		GraphMaxFileLimit:       parseInt(getEnv("GRAPH_MAX_FILE_LIMIT", "500"), 500),
		GraphMaxDepth:           parseInt(getEnv("GRAPH_MAX_DEPTH", "6"), 6),
		OTELServiceName:         getEnv("OTEL_SERVICE_NAME", "codeatlas-api"),
		OTELExporterEndpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTELDisabled:            getEnv("OTEL_SDK_DISABLED", "false") == "true",
		TokenEncryptionKey:      os.Getenv("TOKEN_ENCRYPTION_KEY"),
		PublicAPIBaseURL:        strings.TrimRight(getEnv("PUBLIC_API_BASE_URL", "http://localhost:8080"), "/"),
		FrontendURL:             strings.TrimRight(getEnv("FRONTEND_URL", "http://localhost:5173"), "/"),
		GitHubOAuthClientID:     os.Getenv("GITHUB_OAUTH_CLIENT_ID"),
		GitHubOAuthClientSecret: os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"),
		GitLabOAuthClientID:     os.Getenv("GITLAB_OAUTH_CLIENT_ID"),
		GitLabOAuthClientSecret: os.Getenv("GITLAB_OAUTH_CLIENT_SECRET"),
		BitbucketOAuthClientID:  os.Getenv("BITBUCKET_OAUTH_CLIENT_ID"),
		BitbucketOAuthClientSecret: os.Getenv("BITBUCKET_OAUTH_CLIENT_SECRET"),
		MaxIndexFileBytes:       parseInt64(getEnv("MAX_INDEX_FILE_BYTES", "5242880"), 5<<20),
		MaxIndexFiles:           parseInt(getEnv("MAX_INDEX_FILES", "5000"), 5000),
		MaxRepoBytes:            parseInt64(getEnv("MAX_REPO_BYTES", "524288000"), 500<<20),
		EmbeddingMaxPerRepo:     parseInt(getEnv("EMBEDDING_MAX_PER_REPO", "0"), 0),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseInt(raw string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func parseInt64(raw string, fallback int64) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
