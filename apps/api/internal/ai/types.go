package ai

type ChatRequest struct {
	RepositoryID int64  `json:"repositoryId"`
	Query        string `json:"query"`
	Provider     string `json:"provider,omitempty"`
	Model        string `json:"model,omitempty"`
	Stream       bool   `json:"stream,omitempty"`
}

type RelatedFile struct {
	FileID int64  `json:"fileId"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type ChatResponse struct {
	Answer       string        `json:"answer"`
	RelatedFiles []RelatedFile `json:"relatedFiles"`
	Provider     string        `json:"provider,omitempty"`
	Model        string        `json:"model,omitempty"`
}

type ContextItem struct {
	FileID             int64
	Path               string
	Importance         float64
	Imports            []string
	Exports            []string
	Symbols            []string
	DependencyOut      int
	DependencyIn       int
	SelectionLabel     string
	DominantOwnerLogin string
	BusFactor          int
	ChurnScore         float64
	RiskLevel          string
	IsHotspot          bool
	HasBusFactorRisk   bool
	CommitCount90d     int
}
