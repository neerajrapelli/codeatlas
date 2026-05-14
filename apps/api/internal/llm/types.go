package llm

import "context"

type ChatMessage struct {
	Role    string
	Content string
}

type ChatRequest struct {
	Model       string
	Messages    []ChatMessage
	MaxTokens   int
	Temperature float64
}

type ChatResponse struct {
	Content string
}

type ChatProvider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

type Embedder interface {
	Embed(ctx context.Context, input string) ([]float32, error)
}
