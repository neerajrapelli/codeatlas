package socio

import (
	"time"

	"github.com/google/uuid"
)

// Risk levels for file_metrics and API responses.
const (
	RiskLow      = "low"
	RiskMedium   = "medium"
	RiskHigh     = "high"
	RiskCritical = "critical"
)

// Ingestion run status values.
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusSkipped   = "skipped"
	StatusPartial   = "partial"
)

// Socio ingestion phases (progressive enrichment).
const (
	PhaseGitHubHistory = "github_history"
	PhaseEngineering   = "engineering_memory"
	PhaseOperational   = "operational_intel"
)

// Ingestion steps within Phase 1.
const (
	StepResolveRepo     = "resolve_repository"
	StepSyncContributors = "sync_contributors"
	StepSyncCommits     = "sync_commits"
	StepSyncPullRequests = "sync_pull_requests"
	StepLinkFiles       = "link_files_to_graph"
	StepComputeMetrics  = "compute_file_metrics"
)

// Phase 2 steps (engineering memory).
const (
	StepSyncIssues        = "sync_issues"
	StepSyncPRDiscussions = "sync_pr_discussions"
	StepExtractSignals    = "extract_architecture_signals"
)

type ArchitectureSignal struct {
	ID          string  `json:"id"`
	FileID      *int64  `json:"fileId,omitempty"`
	Path        string  `json:"path,omitempty"`
	SignalType  string  `json:"signalType"`
	Summary     string  `json:"summary"`
	Confidence  float64 `json:"confidence"`
	SourceKind  string  `json:"sourceKind"`
	SourceLabel string  `json:"sourceLabel,omitempty"`
	ExtractedAt string  `json:"extractedAt"`
}

type PullRequestRef struct {
	ID     uuid.UUID
	Number int
	Title  string
}

type Contributor struct {
	ID           uuid.UUID `json:"id"`
	RepositoryID int64     `json:"repositoryId"`
	ExternalID   string    `json:"externalId"`
	Login        string    `json:"login"`
	DisplayName  string    `json:"displayName,omitempty"`
	AvatarURL    string    `json:"avatarUrl,omitempty"`
}

type FileMetrics struct {
	RepositoryID        int64      `json:"repositoryId"`
	FileID              int64      `json:"fileId"`
	Path                string     `json:"path,omitempty"`
	ChurnScore          float64    `json:"churnScore"`
	CommitCount90d      int        `json:"commitCount90d"`
	UniqueAuthors90d    int        `json:"uniqueAuthors90d"`
	BusFactor           int        `json:"busFactor"`
	HotspotScore        float64    `json:"hotspotScore"`
	RiskLevel           string     `json:"riskLevel"`
	IsHotspot           bool       `json:"isHotspot"`
	HasBusFactorRisk    bool       `json:"hasBusFactorRisk"`
	DominantOwnerID     *uuid.UUID `json:"dominantOwnerId,omitempty"`
	DominantOwnerShare  float64    `json:"dominantOwnerShare"`
	DominantOwnerLogin  string     `json:"dominantOwnerLogin,omitempty"`
	LastActivityAt      *time.Time `json:"lastActivityAt,omitempty"`
}

type OwnershipSummary struct {
	FileID             int64        `json:"fileId"`
	Path               string       `json:"path"`
	DominantOwner      *Contributor `json:"dominantOwner,omitempty"`
	ContributorCount   int          `json:"contributorCount"`
	BusFactor          int          `json:"busFactor"`
	RiskLevel          string       `json:"riskLevel"`
	DominantOwnerShare float64      `json:"dominantOwnerShare"`
	Contributors       []OwnerShare `json:"contributors,omitempty"`
}

type OwnerShare struct {
	Contributor Contributor `json:"contributor"`
	Share       float64     `json:"share"`
	CommitCount int         `json:"commitCount"`
}

type HotspotEntry struct {
	FileID       int64   `json:"fileId"`
	Path         string  `json:"path"`
	HotspotScore float64 `json:"hotspotScore"`
	ChurnScore   float64 `json:"churnScore"`
	RiskLevel    string  `json:"riskLevel"`
	BusFactor    int     `json:"busFactor"`
	CommitCount  int     `json:"commitCount90d"`
}

type IngestionStepStatus struct {
	Step            string  `json:"step"`
	Status          string  `json:"status"`
	DurationMs      *int64  `json:"durationMs,omitempty"`
	ItemsProcessed  int     `json:"itemsProcessed"`
	FailureMetadata map[string]any `json:"failureMetadata,omitempty"`
}

type IngestionStatusResponse struct {
	RepositoryID       int64               `json:"repositoryId"`
	CodeIndex          CodeIndexStatus     `json:"codeIndex"`
	SocioTechnical     SocioTechnicalStatus `json:"socioTechnical"`
	GraphCompleteness  GraphCompleteness   `json:"graphCompleteness"`
}

type CodeIndexStatus struct {
	Status          string  `json:"status"`
	Stage           string  `json:"stage"`
	ProgressPercent float64 `json:"progressPercent"`
	FilesIndexed    int     `json:"filesIndexed"`
}

type SocioTechnicalStatus struct {
	Phase             string                `json:"phase"`
	Status            string                `json:"status"`
	CompletionPercent float64               `json:"completionPercent"`
	Staleness         string                `json:"staleness"`
	LastSyncAt        *time.Time            `json:"lastSyncAt,omitempty"`
	ErrorDetails      string                `json:"errorDetails,omitempty"`
	Steps             []IngestionStepStatus `json:"steps"`
	AvailablePhases   []string              `json:"availablePhases"`
}

type GraphCompleteness struct {
	CodeGraphReady      bool `json:"codeGraphReady"`
	SocioHistoryReady   bool `json:"socioHistoryReady"`
	EngineeringReady    bool `json:"engineeringReady"`
	OperationalReady    bool `json:"operationalReady"`
	PartialDataWarning  bool `json:"partialDataWarning"`
}

// GraphOverlay enriches hierarchy layer nodes (graph-first).
type GraphOverlay struct {
	FileOverlays map[string]FileOverlay `json:"fileOverlays"`
}

type FileOverlay struct {
	FileID              string `json:"fileId"`
	IsHotspot           bool   `json:"isHotspot"`
	HasBusFactorRisk    bool   `json:"hasBusFactorRisk"`
	RiskLevel           string `json:"riskLevel,omitempty"`
	ArchitectureSignals int    `json:"architectureSignalCount"`
	DominantOwnerLogin  string `json:"dominantOwnerLogin,omitempty"`
}

type RepositoryRef struct {
	ID         int64
	SourceType string
	SourceURL  string
	Branch     string
	Status     string
}

type GitHubRepo struct {
	Owner string
	Name  string
}
