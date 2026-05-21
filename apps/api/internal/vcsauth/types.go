package vcsauth

import (
	"time"

	"github.com/google/uuid"
)

type Provider string

const (
	ProviderGitHub    Provider = "github"
	ProviderGitLab    Provider = "gitlab"
	ProviderBitbucket Provider = "bitbucket"
)

func (p Provider) Valid() bool {
	switch p {
	case ProviderGitHub, ProviderGitLab, ProviderBitbucket:
		return true
	default:
		return false
	}
}

type TokenType string

const (
	TokenOAuth       TokenType = "oauth"
	TokenPAT         TokenType = "pat"
	TokenAppPassword TokenType = "app_password"
)

type ProviderToken struct {
	ID             uuid.UUID
	TenantID       string
	UserSubject    string
	Provider       Provider
	TokenType      TokenType
	Scopes         []string
	ExternalUserID string
	ExpiresAt      *time.Time
	Metadata       map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RemoteRepository is a normalized listing entry from any VCS API.
type RemoteRepository struct {
	ID          string `json:"id"`
	FullName    string `json:"fullName"`
	CloneURL    string `json:"cloneUrl"`
	HTMLURL     string `json:"htmlUrl,omitempty"`
	DefaultBranch string `json:"defaultBranch"`
	Private     bool   `json:"private"`
}
