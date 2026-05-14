package llm

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
)

const localEmbeddingDims = 1536

type LocalClient struct{}

func NewLocalClient() *LocalClient { return &LocalClient{} }

func (c *LocalClient) Embed(_ context.Context, input string) ([]float32, error) {
	vector := make([]float32, localEmbeddingDims)
	for _, token := range tokenize(input) {
		h := fnv.New64a()
		_, _ = h.Write([]byte(token))
		idx := int(h.Sum64() % uint64(localEmbeddingDims))
		vector[idx] += 1
	}
	normalize(vector)
	return vector, nil
}

func (c *LocalClient) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	userPrompt := ""
	for _, m := range req.Messages {
		if m.Role == "user" {
			userPrompt = m.Content
		}
	}
	if userPrompt == "" {
		return ChatResponse{Content: "No context provided."}, nil
	}

	lines := strings.Split(userPrompt, "\n")
	question := "Unknown question"
	related := make([]string, 0, 6)
	for _, line := range lines {
		if strings.HasPrefix(line, "Question: ") {
			question = strings.TrimPrefix(line, "Question: ")
		}
		if strings.Contains(line, "file=") {
			start := strings.Index(line, "file=")
			end := strings.Index(line[start:], " ")
			if end == -1 {
				related = append(related, strings.TrimPrefix(line[start:], "file="))
			} else {
				related = append(related, strings.TrimPrefix(line[start:start+end], "file="))
			}
			if len(related) >= 6 {
				break
			}
		}
	}

	return ChatResponse{
		Content: fmt.Sprintf(
			"Based on indexed graph context, likely impact areas for \"%s\" are concentrated in files with direct dependencies and exported symbols.\n\nRelated files:\n- %s",
			question,
			strings.Join(related, "\n- "),
		),
	}, nil
}

func tokenize(input string) []string {
	clean := strings.ToLower(strings.TrimSpace(input))
	if clean == "" {
		return []string{"empty"}
	}
	return strings.Fields(clean)
}

func normalize(vector []float32) {
	var norm float32
	for _, v := range vector {
		norm += v * v
	}
	if norm == 0 {
		return
	}
	for i, v := range vector {
		vector[i] = v / norm
	}
}
