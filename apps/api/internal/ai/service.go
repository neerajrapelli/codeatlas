package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"codeatlas/apps/api/internal/ai/providers"
)

type Service struct {
	retriever   *Retriever
	defaultModel string
	defaultProvider providers.ProviderName
	manager *providers.Manager
	logger      *slog.Logger
	tokenBudget int
}

func NewService(retriever *Retriever, defaultProvider providers.ProviderName, defaultModel string, tokenBudget int, manager *providers.Manager, logger *slog.Logger) *Service {
	return &Service{
		retriever:   retriever,
		defaultProvider: defaultProvider,
		defaultModel: defaultModel,
		tokenBudget: tokenBudget,
		manager: manager,
		logger:      logger,
	}
}

// PreparedChat is the retrieval + prompt packaging stage shared by JSON and streaming handlers.
type PreparedChat struct {
	RelatedFiles     []RelatedFile
	Provider         providers.ProviderName
	Model            string
	ChatReq          providers.ChatRequest
	EmbedProvider    providers.ProviderName
	ContextFileCount int
}

func (s *Service) PrepareChat(ctx context.Context, req ChatRequest) (*PreparedChat, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query is required")
	}

	providerName := providers.ProviderName(req.Provider)
	if providerName == "" {
		providerName = s.defaultProvider
	}
	model := req.Model
	if model == "" {
		model = s.defaultModel
	}

	embedding, embedProvider, err := s.manager.Embed(ctx, providerName, providers.EmbedRequest{
		Model: model,
		Input: req.Query,
	})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	contextItems, err := s.retriever.RetrieveContext(ctx, req.RepositoryID, embedding, 14)
	if err != nil {
		return nil, fmt.Errorf("retrieve context: %w", err)
	}

	related := make([]RelatedFile, 0, minInt(8, len(contextItems)))
	for i := range contextItems {
		if i >= 8 {
			break
		}
		related = append(related, RelatedFile{
			FileID: contextItems[i].FileID,
			Path:   contextItems[i].Path,
			Reason: contextItems[i].SelectionLabel,
		})
	}

	var userPrompt string
	if len(contextItems) == 0 {
		userPrompt = req.Query
	} else {
		userPrompt = buildUserPrompt(req.Query, contextItems, s.tokenBudget*4)
	}

	chatReq := providers.ChatRequest{
		Model: model,
		Messages: []providers.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   minInt(1200, s.tokenBudget/2),
		Temperature: 0.2,
	}

	return &PreparedChat{
		RelatedFiles:     related,
		Provider:         providerName,
		Model:            model,
		ChatReq:          chatReq,
		EmbedProvider:    embedProvider,
		ContextFileCount: len(contextItems),
	}, nil
}

func (s *Service) Answer(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	prepared, err := s.PrepareChat(ctx, req)
	if err != nil {
		return ChatResponse{}, err
	}
	if prepared.ContextFileCount == 0 {
		return ChatResponse{Answer: "I could not find relevant indexed context for this repository yet.", RelatedFiles: prepared.RelatedFiles}, nil
	}

	chatResp, chatProvider, err := s.manager.Chat(ctx, prepared.Provider, prepared.ChatReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("llm chat: %w", err)
	}

	s.logger.Info("ai_chat_answered", "repository_id", req.RepositoryID, "context_files", prepared.ContextFileCount, "related", len(prepared.RelatedFiles), "embed_provider", prepared.EmbedProvider, "chat_provider", chatProvider)
	return ChatResponse{Answer: chatResp.Content, RelatedFiles: prepared.RelatedFiles, Provider: string(chatProvider), Model: prepared.Model}, nil
}

// StreamCompletion streams chat token deltas from the configured provider manager.
func (s *Service) StreamCompletion(ctx context.Context, prepared *PreparedChat) (<-chan providers.StreamChunk, <-chan error, error) {
	return s.manager.StreamChat(ctx, prepared.Provider, prepared.ChatReq)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
