package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"codeatlas/apps/api/internal/ai"
)

type PlanRequest struct {
	Role             string `json:"role"`
	ExperienceLevel  string `json:"experience_level"`
	FocusArea        string `json:"focus_area,omitempty"`
}

type PlanStep struct {
	Order       int      `json:"order"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	FilePaths   []string `json:"file_paths"`
	EstimatedMinutes int `json:"estimated_minutes"`
}

type PlanResponse struct {
	Role        string     `json:"role"`
	Experience  string     `json:"experience_level"`
	Summary     string     `json:"summary"`
	Steps       []PlanStep `json:"steps"`
}

type Service struct {
	ai *ai.Service
}

func NewService(aiSvc *ai.Service) *Service {
	return &Service{ai: aiSvc}
}

func (s *Service) Generate(ctx context.Context, repositoryID int64, req PlanRequest) (*PlanResponse, error) {
	if s.ai == nil {
		return fallbackPlan(req), nil
	}
	query := fmt.Sprintf(
		`Generate a JSON onboarding plan for a %s engineer (%s experience) exploring repository %d. Focus: %s. Return ONLY JSON: {"summary":"...","steps":[{"order":1,"title":"...","description":"...","file_paths":["path"],"estimated_minutes":30}]}`,
		req.Role, req.ExperienceLevel, repositoryID, req.FocusArea,
	)
	resp, err := s.ai.Answer(ctx, ai.ChatRequest{
		RepositoryID: repositoryID,
		Query:        query,
	})
	if err != nil {
		return fallbackPlan(req), nil
	}
	text := strings.TrimSpace(resp.Answer)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		text = text[start : end+1]
	}
	var partial struct {
		Summary string     `json:"summary"`
		Steps   []PlanStep `json:"steps"`
	}
	if err := json.Unmarshal([]byte(text), &partial); err != nil {
		return fallbackPlan(req), nil
	}
	return &PlanResponse{
		Role:       req.Role,
		Experience: req.ExperienceLevel,
		Summary:    partial.Summary,
		Steps:      partial.Steps,
	}, nil
}

func fallbackPlan(req PlanRequest) *PlanResponse {
	return &PlanResponse{
		Role:       req.Role,
		Experience: req.ExperienceLevel,
		Summary:    "Start with the repository entrypoints and core modules.",
		Steps: []PlanStep{
			{Order: 1, Title: "Map the layout", Description: "Review top-level packages and shared libraries.", FilePaths: []string{"README.md"}, EstimatedMinutes: 30},
			{Order: 2, Title: "Trace dependencies", Description: "Follow import edges from API layer to domain logic.", FilePaths: []string{"apps/api/cmd/server/main.go"}, EstimatedMinutes: 45},
		},
	}
}
