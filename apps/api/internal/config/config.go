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
