package repoingest

import "time"

type SourceType string

const (
	SourceGitHub    SourceType = "github"
	SourceGitLab    SourceType = "gitlab"
	SourceBitbucket SourceType = "bitbucket"
	SourceZIP       SourceType = "zip"
)

type Status string

const (
	StatusQueued     Status = "queued"
	StatusCloning    Status = "cloning"
	StatusExtracting Status = "extracting"
	StatusParsing    Status = "parsing"
	StatusBuildingGraph Status = "building_graph"
	StatusGeneratingEmbeddings Status = "generating_embeddings"
	StatusReady      Status = "ready"
	StatusFailed     Status = "failed"
)

type CreateRequest struct {
	SourceType SourceType `json:"sourceType"`
	SourceURL  string     `json:"sourceUrl"`
	Branch     string     `json:"branch"`
	DisplayName string    `json:"displayName"`
	ZIPPath    string
}

type Repository struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	SourceType SourceType `json:"sourceType"`
	SourceURL  string     `json:"sourceUrl"`
	Branch     string     `json:"branch"`
	WorkspacePath string  `json:"workspacePath"`
	Status     Status     `json:"status"`
	ProgressPercent float64 `json:"progressPercent"`
	FilesIndexed int `json:"filesIndexed"`
	SymbolsIndexed int `json:"symbolsIndexed"`
	EdgesIndexed int `json:"edgesIndexed"`
	EmbeddingsIndexed int `json:"embeddingsIndexed"`
	CurrentStage Status `json:"currentStage"`
	StageMetadata map[string]any `json:"stageMetadata,omitempty"`
	ErrorDetails string   `json:"errorDetails,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type ProgressResponse struct {
	RepositoryID int64 `json:"repositoryId"`
	Stage Status `json:"stage"`
	Status Status `json:"status"`
	ProgressPercent float64 `json:"progressPercent"`
	Metrics struct {
		FilesIndexed int `json:"filesIndexed"`
		SymbolsIndexed int `json:"symbolsIndexed"`
		EdgesIndexed int `json:"edgesIndexed"`
		EmbeddingsIndexed int `json:"embeddingsIndexed"`
	} `json:"metrics"`
	StageMetadata map[string]any `json:"stageMetadata,omitempty"`
	ErrorDetails string `json:"errorDetails,omitempty"`
}
