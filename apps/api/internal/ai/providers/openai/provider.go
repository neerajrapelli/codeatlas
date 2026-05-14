package openai

import (
	"context"

	"codeatlas/apps/api/internal/ai/providers"
	"codeatlas/apps/api/internal/llm"
)

type Provider struct {
	client *llm.OpenAIClient
}

func New(apiKey, chatModel, embeddingModel string) *Provider {
	return &Provider{client: llm.NewOpenAIClient(apiKey, chatModel, embeddingModel)}
}

func (p *Provider) Name() providers.ProviderName { return providers.ProviderOpenAI }
func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{Chat: true, Embedding: true, Summary: true, Streaming: true, DefaultModel: "gpt-4o-mini"}
}
func (p *Provider) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	resp, err := p.client.Chat(ctx, llm.ChatRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Messages:    toLLMMessages(req.Messages),
	})
	if err != nil {
		return providers.ChatResponse{}, err
	}
	return providers.ChatResponse{Content: resp.Content}, nil
}
func (p *Provider) Embed(ctx context.Context, req providers.EmbedRequest) ([]float32, error) {
	return p.client.Embed(ctx, req.Input)
}
func (p *Provider) Summarize(ctx context.Context, req providers.SummaryRequest) (string, error) {
	resp, err := p.Chat(ctx, providers.ChatRequest{
		Model: req.Model,
		Messages: []providers.Message{
			{Role: "system", Content: "Summarize this content for architecture decisions."},
			{Role: "user", Content: req.Content},
		},
		MaxTokens:   500,
		Temperature: 0.1,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
func (p *Provider) StreamChat(ctx context.Context, req providers.ChatRequest) (<-chan providers.StreamChunk, <-chan error) {
	out := make(chan providers.StreamChunk, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errCh)
		tokenCh, llmErrCh := p.client.ChatStream(ctx, llm.ChatRequest{
			Model:       req.Model,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			Messages:    toLLMMessages(req.Messages),
		})
		for tok := range tokenCh {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}
			out <- providers.StreamChunk{Delta: tok}
		}
		err, ok := <-llmErrCh
		if ok && err != nil {
			errCh <- err
			return
		}
		out <- providers.StreamChunk{Done: true}
	}()
	return out, errCh
}

func toLLMMessages(in []providers.Message) []llm.ChatMessage {
	out := make([]llm.ChatMessage, 0, len(in))
	for _, m := range in {
		out = append(out, llm.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return out
}
