package providers

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Manager struct {
	providers map[ProviderName]Provider
	defaultProvider ProviderName
	fallbackOrder []ProviderName
	logger *slog.Logger
	retries int
	timeout time.Duration
}

func NewManager(defaultProvider ProviderName, fallback []ProviderName, logger *slog.Logger) *Manager {
	return &Manager{
		providers: make(map[ProviderName]Provider),
		defaultProvider: defaultProvider,
		fallbackOrder: fallback,
		logger: logger,
		retries: 1,
		timeout: 30 * time.Second,
	}
}

func (m *Manager) Register(provider Provider) {
	m.providers[provider.Name()] = provider
}

func (m *Manager) Resolve(name ProviderName) (Provider, []Provider, error) {
	selected := name
	if selected == "" {
		selected = m.defaultProvider
	}
	primary, ok := m.providers[selected]
	if !ok {
		return nil, nil, fmt.Errorf("provider %q unavailable", selected)
	}
	fallbacks := make([]Provider, 0, len(m.fallbackOrder))
	for _, item := range m.fallbackOrder {
		if item == selected {
			continue
		}
		if p, ok := m.providers[item]; ok {
			fallbacks = append(fallbacks, p)
		}
	}
	return primary, fallbacks, nil
}

func (m *Manager) Chat(ctx context.Context, providerName ProviderName, req ChatRequest) (ChatResponse, ProviderName, error) {
	primary, fallbacks, err := m.Resolve(providerName)
	if err != nil {
		return ChatResponse{}, "", err
	}
	chain := append([]Provider{primary}, fallbacks...)
	var lastErr error
	for _, candidate := range chain {
		for attempt := 0; attempt <= m.retries; attempt++ {
			callCtx, cancel := context.WithTimeout(ctx, m.timeout)
			start := time.Now()
			resp, err := candidate.Chat(callCtx, req)
			cancel()
			if err == nil {
				m.logger.Info("ai_provider_chat_success", "provider", candidate.Name(), "latency_ms", time.Since(start).Milliseconds())
				return resp, candidate.Name(), nil
			}
			lastErr = err
			m.logger.Warn("ai_provider_chat_failed", "provider", candidate.Name(), "attempt", attempt+1, "error", err)
		}
	}
	return ChatResponse{}, "", fmt.Errorf("all providers failed: %w", lastErr)
}

func (m *Manager) Embed(ctx context.Context, providerName ProviderName, req EmbedRequest) ([]float32, ProviderName, error) {
	primary, fallbacks, err := m.Resolve(providerName)
	if err != nil {
		return nil, "", err
	}
	chain := append([]Provider{primary}, fallbacks...)
	var lastErr error
	for _, candidate := range chain {
		callCtx, cancel := context.WithTimeout(ctx, m.timeout)
		start := time.Now()
		vector, err := candidate.Embed(callCtx, req)
		cancel()
		if err == nil {
			m.logger.Info("ai_provider_embed_success", "provider", candidate.Name(), "latency_ms", time.Since(start).Milliseconds())
			return vector, candidate.Name(), nil
		}
		lastErr = err
		m.logger.Warn("ai_provider_embed_failed", "provider", candidate.Name(), "error", err)
	}
	return nil, "", fmt.Errorf("all providers failed: %w", lastErr)
}

// StreamChat streams completion deltas. On failure it falls back to non-streaming Chat for the same provider chain.
func (m *Manager) StreamChat(ctx context.Context, providerName ProviderName, req ChatRequest) (<-chan StreamChunk, <-chan error, error) {
	primary, fallbacks, err := m.Resolve(providerName)
	if err != nil {
		return nil, nil, err
	}
	chain := append([]Provider{primary}, fallbacks...)
	out := make(chan StreamChunk, 128)
	errOut := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errOut)
		var lastErr error
		for _, candidate := range chain {
			callCtx, cancel := context.WithTimeout(ctx, m.timeout)
			chunks, streamErrs := candidate.StreamChat(callCtx, req)
			streamErr := drainProviderStream(callCtx, out, chunks, streamErrs)
			if streamErr == nil {
				cancel()
				m.logger.Info("ai_provider_stream_success", "provider", candidate.Name())
				return
			}
			cancel()
			lastErr = streamErr
			m.logger.Warn("ai_provider_stream_failed", "provider", candidate.Name(), "error", streamErr)

			callCtx2, cancel2 := context.WithTimeout(ctx, m.timeout)
			resp, chatErr := candidate.Chat(callCtx2, req)
			cancel2()
			if chatErr != nil {
				lastErr = chatErr
				m.logger.Warn("ai_provider_chat_fallback_failed", "provider", candidate.Name(), "error", chatErr)
				continue
			}
			out <- StreamChunk{Delta: resp.Content}
			out <- StreamChunk{Done: true}
			m.logger.Info("ai_provider_chat_fallback_success", "provider", candidate.Name())
			return
		}
		errOut <- fmt.Errorf("all providers failed: %w", lastErr)
	}()
	return out, errOut, nil
}

func drainProviderStream(ctx context.Context, out chan<- StreamChunk, chunks <-chan StreamChunk, errs <-chan error) error {
	var sawDone bool
	for chunks != nil || errs != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return err
			}
		case ch, ok := <-chunks:
			if !ok {
				chunks = nil
				continue
			}
			out <- ch
			if ch.Done {
				sawDone = true
				return nil
			}
		}
	}
	if !sawDone {
		out <- StreamChunk{Done: true}
	}
	return nil
}
