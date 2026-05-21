package socio

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// QueryService exposes read APIs for socio-technical intelligence.
type QueryService struct {
	store *Store
}

func NewQueryService(store *Store) *QueryService {
	return &QueryService{store: store}
}

func (q *QueryService) GetHotspots(ctx context.Context, repositoryID int64, limit int) ([]HotspotEntry, error) {
	return q.store.ListHotspots(ctx, repositoryID, limit)
}

func (q *QueryService) GetOwnership(ctx context.Context, repositoryID int64, fileID int64) ([]OwnershipSummary, error) {
	return q.store.ListOwnership(ctx, repositoryID, fileID, 50)
}

func (q *QueryService) GetFileOverlays(ctx context.Context, repositoryID int64) (GraphOverlay, error) {
	m, err := q.store.FileOverlays(ctx, repositoryID)
	if err != nil {
		return GraphOverlay{}, err
	}
	return GraphOverlay{FileOverlays: m}, nil
}

func (q *QueryService) BuildIngestionStatus(ctx context.Context, repositoryID int64, code CodeIndexStatus) (IngestionStatusResponse, error) {
	runID, phase, status, pct, completed, errDetails, err := q.store.LatestIngestionRun(ctx, repositoryID)
	if err != nil {
		return IngestionStatusResponse{}, err
	}
	var steps []IngestionStepStatus
	if runID != uuid.Nil {
		steps, _ = q.store.ListRunSteps(ctx, runID)
	}
	stale := "fresh"
	var lastSync *time.Time
	if completed != nil {
		lastSync = completed
		if time.Since(*completed) > 7*24*time.Hour {
			stale = "stale"
		}
	} else if status == StatusPending || status == "" {
		stale = "unknown"
	}

	socio := SocioTechnicalStatus{
		Phase:             phaseOrDefault(phase, PhaseGitHubHistory),
		Status:            statusOrDefault(status, StatusPending),
		CompletionPercent: pct,
		Staleness:         stale,
		LastSyncAt:        lastSync,
		ErrorDetails:      errDetails,
		Steps:             steps,
		AvailablePhases:   []string{PhaseGitHubHistory, PhaseEngineering, PhaseOperational},
	}

	graph := GraphCompleteness{
		CodeGraphReady:     code.Status == "ready" || code.FilesIndexed > 0,
		SocioHistoryReady:  status == StatusCompleted,
		EngineeringReady:   false,
		OperationalReady:   false,
		PartialDataWarning: code.Status != "ready" || (status != StatusCompleted && status != StatusSkipped),
	}

	return IngestionStatusResponse{
		RepositoryID:      repositoryID,
		CodeIndex:         code,
		SocioTechnical:    socio,
		GraphCompleteness: graph,
	}, nil
}

func phaseOrDefault(p, def string) string {
	if p == "" {
		return def
	}
	return p
}

func statusOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
