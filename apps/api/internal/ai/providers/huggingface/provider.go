package huggingface

import (
    "context"
    "fmt"

    "codeatlas/apps/api/internal/ai/providers"
)

type Provider struct{}

func New() *Provider { return &Provider{} }
func (p *Provider) Name() providers.ProviderName { return providers.ProviderHuggingFace }
func (p *Provider) Capabilities() providers.Capabilities { return providers.Capabilities{Chat: false, Embedding: false, Summary: false, Streaming: false, DefaultModel: "mistral-7b-instruct"} }
func (p *Provider) Chat(_ context.Context, _ providers.ChatRequest) (providers.ChatResponse, error) { return providers.ChatResponse{}, fmt.Errorf("huggingface adapter not configured yet") }
func (p *Provider) Embed(_ context.Context, _ providers.EmbedRequest) ([]float32, error) { return nil, fmt.Errorf("huggingface embeddings not configured yet") }
func (p *Provider) Summarize(_ context.Context, _ providers.SummaryRequest) (string, error) { return "", fmt.Errorf("huggingface summarize not configured yet") }
func (p *Provider) StreamChat(_ context.Context, _ providers.ChatRequest) (<-chan providers.StreamChunk, <-chan error) {
    out := make(chan providers.StreamChunk)
    errCh := make(chan error, 1)
    close(out)
    errCh <- fmt.Errorf("huggingface streaming not configured yet")
    close(errCh)
    return out, errCh
}
