package anthropic

import (
    "context"
    "fmt"

    "codeatlas/apps/api/internal/ai/providers"
)

type Provider struct{}

func New() *Provider { return &Provider{} }
func (p *Provider) Name() providers.ProviderName { return providers.ProviderAnthropic }
func (p *Provider) Capabilities() providers.Capabilities { return providers.Capabilities{Chat: false, Embedding: false, Summary: false, Streaming: false, DefaultModel: "claude-3-5-sonnet"} }
func (p *Provider) Chat(_ context.Context, _ providers.ChatRequest) (providers.ChatResponse, error) { return providers.ChatResponse{}, fmt.Errorf("anthropic adapter not configured yet") }
func (p *Provider) Embed(_ context.Context, _ providers.EmbedRequest) ([]float32, error) { return nil, fmt.Errorf("anthropic embeddings not configured yet") }
func (p *Provider) Summarize(_ context.Context, _ providers.SummaryRequest) (string, error) { return "", fmt.Errorf("anthropic summarize not configured yet") }
func (p *Provider) StreamChat(_ context.Context, _ providers.ChatRequest) (<-chan providers.StreamChunk, <-chan error) {
    out := make(chan providers.StreamChunk)
    errCh := make(chan error, 1)
    close(out)
    errCh <- fmt.Errorf("anthropic streaming not configured yet")
    close(errCh)
    return out, errCh
}
