package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient struct {
	apiKey         string
	baseURL        string
	chatModel      string
	embeddingModel string
	httpClient     *http.Client
	streamClient   *http.Client
}

func NewOpenAIClient(apiKey, chatModel, embeddingModel string) *OpenAIClient {
	return &OpenAIClient{
		apiKey:         apiKey,
		baseURL:        "https://api.openai.com/v1",
		chatModel:      chatModel,
		embeddingModel: embeddingModel,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		streamClient:   &http.Client{Timeout: 0},
	}
}

// ChatStream requests a streamed completion and emits UTF-8 token deltas on out.
func (c *OpenAIClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan string, <-chan error) {
	out := make(chan string, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errCh)

		model := req.Model
		if model == "" {
			model = c.chatModel
		}
		if model == "" {
			model = "gpt-4o-mini"
		}

		payload := map[string]any{
			"model":       model,
			"messages":    req.Messages,
			"max_tokens":  req.MaxTokens,
			"temperature": req.Temperature,
			"stream":      true,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			errCh <- err
			return
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			errCh <- err
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.streamClient.Do(httpReq)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()

		var buf bytes.Buffer
		if resp.StatusCode >= 300 {
			_, _ = buf.ReadFrom(resp.Body)
			errCh <- fmt.Errorf("openai status %d: %s", resp.StatusCode, buf.String())
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if data == "[DONE]" {
				return
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			t := chunk.Choices[0].Delta.Content
			if t != "" {
				out <- t
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()
	return out, errCh
}

func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.chatModel
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	payload := map[string]any{
		"model":       model,
		"messages":    req.Messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}

	body, err := c.doJSON(ctx, "/chat/completions", payload)
	if err != nil {
		return ChatResponse{}, err
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return ChatResponse{}, fmt.Errorf("decode chat response: %w", err)
	}
	if len(out.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("empty chat response")
	}
	return ChatResponse{Content: out.Choices[0].Message.Content}, nil
}

func (c *OpenAIClient) Embed(ctx context.Context, input string) ([]float32, error) {
	model := c.embeddingModel
	if model == "" {
		model = "text-embedding-3-small"
	}
	payload := map[string]any{
		"model": model,
		"input": input,
	}

	body, err := c.doJSON(ctx, "/embeddings", payload)
	if err != nil {
		return nil, err
	}

	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return out.Data[0].Embedding, nil
}

func (c *OpenAIClient) doJSON(ctx context.Context, path string, payload map[string]any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	var raw bytes.Buffer
	if _, err := raw.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai status %d: %s", resp.StatusCode, raw.String())
	}
	return raw.Bytes(), nil
}
