package driftdetector

import (
	"time"

	"github.com/google/uuid"
)

type Rule struct {
	ID             uuid.UUID `json:"id"`
	RepositoryID   int64     `json:"repositoryId"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	RuleType       string    `json:"ruleType"`
	SourcePattern  string    `json:"sourcePattern"`
	TargetPattern  string    `json:"targetPattern"`
	Severity       string    `json:"severity"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Violation struct {
	ID          uuid.UUID `json:"id,omitempty"`
	RuleID      uuid.UUID `json:"ruleId"`
	RuleName    string    `json:"ruleName"`
	SourceFile  string    `json:"sourceFile"`
	TargetFile  string    `json:"targetFile"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	DetectedAt  time.Time `json:"detectedAt,omitempty"`
}

type CreateRuleRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	RuleType      string `json:"ruleType"`
	SourcePattern string `json:"sourcePattern"`
	TargetPattern string `json:"targetPattern"`
	Severity      string `json:"severity,omitempty"`
	Enabled       *bool  `json:"enabled,omitempty"`
}

type CheckPRRequest struct {
	ChangedFiles []string `json:"changedFiles"`
}
