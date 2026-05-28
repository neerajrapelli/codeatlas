package archintel

import "time"

// DecisionStatus represents lifecycle status for an architecture decision.
type DecisionStatus string

const (
	DecisionProposed  DecisionStatus = "proposed"
	DecisionAccepted  DecisionStatus = "accepted"
	DecisionRejected  DecisionStatus = "rejected"
	DecisionDeprecated DecisionStatus = "deprecated"
)

// EvidenceKind captures the source category backing a decision/tradeoff.
type EvidenceKind string

const (
	EvidencePRComment   EvidenceKind = "pr_comment"
	EvidencePRReview    EvidenceKind = "pr_review"
	EvidenceIssue       EvidenceKind = "issue"
	EvidenceDiscussion  EvidenceKind = "discussion"
	EvidenceRFC         EvidenceKind = "rfc"
	EvidenceADR         EvidenceKind = "adr"
	EvidenceDesignDoc   EvidenceKind = "design_doc"
)

type DecisionRecord struct {
	ID              string         `json:"id"`
	RepositoryID    int64          `json:"repositoryId"`
	Title           string         `json:"title"`
	Summary         string         `json:"summary"`
	Status          DecisionStatus `json:"status"`
	Tradeoffs       []string       `json:"tradeoffs"`
	AffectedModules []string       `json:"affectedModules"`
	AffectedFiles   []string       `json:"affectedFiles"`
	Participants    []string       `json:"participants"`
	Confidence      float64        `json:"confidence"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type DecisionEvent struct {
	ID         string    `json:"id"`
	DecisionID string    `json:"decisionId"`
	EventType  string    `json:"eventType"`
	Summary    string    `json:"summary"`
	Actor      string    `json:"actor,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type TimelineEntry struct {
	ID              string       `json:"id"`
	OccurredAt      time.Time    `json:"occurredAt"`
	Kind            string       `json:"kind"`
	Title           string       `json:"title"`
	Summary         string       `json:"summary"`
	DecisionID      string       `json:"decisionId,omitempty"`
	RelatedModules  []string     `json:"relatedModules"`
	RelatedFiles    []string     `json:"relatedFiles"`
	Participants    []string     `json:"participants"`
	EvidenceKind    EvidenceKind `json:"evidenceKind"`
	EvidenceRef     string       `json:"evidenceRef,omitempty"`
}

type PRInsight struct {
	PullRequestID    int64    `json:"pullRequestId"`
	Number           int      `json:"number"`
	Title            string   `json:"title"`
	Author           string   `json:"author"`
	Summary          string   `json:"summary"`
	DecisionIDs      []string `json:"decisionIds"`
	KeyTradeoffs     []string `json:"keyTradeoffs"`
	AffectedModules  []string `json:"affectedModules"`
	Participants     []string `json:"participants"`
	ReviewDisagreeCt int      `json:"reviewDisagreementCount"`
	UpdatedAt        string   `json:"updatedAt"`
}

type MaintainerInfluence struct {
	Login             string   `json:"login"`
	DisplayName       string   `json:"displayName,omitempty"`
	DecisionsShaped   int      `json:"decisionsShaped"`
	AcceptedProposals int      `json:"acceptedProposals"`
	RejectedProposals int      `json:"rejectedProposals"`
	ModulesTouched    []string `json:"modulesTouched"`
	LastActiveAt      string   `json:"lastActiveAt,omitempty"`
}

type ModuleIntelligence struct {
	ModulePath      string          `json:"modulePath"`
	DecisionCount   int             `json:"decisionCount"`
	Decisions       []DecisionRecord `json:"decisions"`
	RecentTimeline  []TimelineEntry `json:"recentTimeline"`
	TopMaintainers  []MaintainerInfluence `json:"topMaintainers"`
	RelatedPRs      []PRInsight     `json:"relatedPRs"`
}

type SearchHit struct {
	ID             string       `json:"id"`
	Kind           string       `json:"kind"`
	Title          string       `json:"title"`
	Summary        string       `json:"summary"`
	Score          float64      `json:"score"`
	KeywordScore   float64      `json:"keywordScore"`
	VectorScore    float64      `json:"vectorScore"`
	RecencyBoost   float64      `json:"recencyBoost"`
	ModuleBoost    float64      `json:"moduleBoost"`
	MaintainerBoost float64     `json:"maintainerBoost"`
	MatchedModules []string     `json:"matchedModules"`
	Participants   []string     `json:"participants"`
	EvidenceKind   EvidenceKind `json:"evidenceKind"`
	OccurredAt     *time.Time   `json:"occurredAt,omitempty"`
}

type TimelineResponse struct {
	RepositoryID int64         `json:"repositoryId"`
	Items        []TimelineEntry `json:"items"`
}

type DecisionsResponse struct {
	RepositoryID int64          `json:"repositoryId"`
	Items        []DecisionRecord `json:"items"`
}

type PRInsightsResponse struct {
	RepositoryID int64       `json:"repositoryId"`
	Items        []PRInsight `json:"items"`
}

type MaintainerInfluenceResponse struct {
	RepositoryID int64               `json:"repositoryId"`
	Items        []MaintainerInfluence `json:"items"`
}

type SearchResponse struct {
	RepositoryID int64       `json:"repositoryId"`
	Query        string      `json:"query"`
	Items        []SearchHit `json:"items"`
}
