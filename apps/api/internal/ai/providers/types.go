package providers

import "context"

type ProviderName string

const (
	ProviderOpenAI     ProviderName = "openai"
	ProviderAnthropic  ProviderName = "anthropic"
	ProviderGemini     ProviderName = "gemini"
	ProviderHuggingFace ProviderName = "huggingface"
	ProviderOpenRouter ProviderName = "openrouter"
	ProviderLocal      ProviderName = "local"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
}

type ChatResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
}

type EmbedRequest struct {
	Model string
	Input string
}

type SummaryRequest struct {
	Model   string
	Content string
}

type StreamChunk struct {
	Delta string
	Done  bool
}

type Capabilities struct {
	Chat       bool
	Embedding  bool
	Summary    bool
	Streaming  bool
	DefaultModel string
}

type Provider interface {
	Name() ProviderName
	Capabilities() Capabilities
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	Embed(ctx context.Context, req EmbedRequest) ([]float32, error)
	Summarize(ctx context.Context, req SummaryRequest) (string, error)
	StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamChunk, <-chan error)
}
