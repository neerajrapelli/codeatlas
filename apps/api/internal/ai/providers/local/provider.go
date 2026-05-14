package local

import (
	"context"
	"fmt"

	"codeatlas/apps/api/internal/ai/providers"
	"codeatlas/apps/api/internal/llm"
)

type Provider struct {
	client *llm.LocalClient
}

func New() *Provider {
	return &Provider{client: llm.NewLocalClient()}
}

func (p *Provider) Name() providers.ProviderName { return providers.ProviderLocal }
func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{Chat: true, Embedding: true, Summary: true, Streaming: false, DefaultModel: "local-default"}
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
			{Role: "system", Content: "Summarize the following content into concise architecture notes."},
			{Role: "user", Content: req.Content},
		},
		MaxTokens:   400,
		Temperature: 0.1,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
func (p *Provider) StreamChat(_ context.Context, _ providers.ChatRequest) (<-chan providers.StreamChunk, <-chan error) {
	out := make(chan providers.StreamChunk)
	errCh := make(chan error, 1)
	close(out)
	errCh <- fmt.Errorf("streaming is not supported for local provider")
	close(errCh)
	return out, errCh
}

func toLLMMessages(in []providers.Message) []llm.ChatMessage {
	out := make([]llm.ChatMessage, 0, len(in))
	for _, m := range in {
		out = append(out, llm.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return out
}
