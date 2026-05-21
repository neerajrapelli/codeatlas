package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"codeatlas/apps/api/internal/ingestprogress"
	"codeatlas/apps/api/internal/repoingest"
	"codeatlas/apps/api/internal/socio"
	"github.com/jackc/pgx/v5/pgxpool"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSSE(w http.ResponseWriter, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", b)
	return err
}

func parseRepositoryID(r *http.Request) (int64, error) {
	raw := r.URL.Query().Get("repositoryId")
	if raw == "" {
		return 0, errors.New("repositoryId is required")
	}
	repoID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || repoID <= 0 {
		return 0, errors.New("repositoryId must be a positive integer")
	}
	return repoID, nil
}

func parseRepositoryIDGuarded(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) (int64, bool) {
	repoID, err := parseRepositoryID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return 0, false
	}
	if !guardRepository(w, r, pool, repoID) {
		return 0, false
	}
	return repoID, true
}

func parsePathRepositoryIDGuarded(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) (int64, bool) {
	repoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || repoID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid repository id"})
		return 0, false
	}
	if !guardRepository(w, r, pool, repoID) {
		return 0, false
	}
	return repoID, true
}

func streamEventToCodeIndex(ev ingestprogress.StreamEvent) socio.CodeIndexStatus {
	stage := ingestprogress.StepToRepoStage(ev.CurrentStep)
	if stage == "" {
		stage = ev.CurrentStep
	}
	status := ev.Status
	switch status {
	case ingestprogress.StatusQueued:
		status = string(repoingest.StatusQueued)
	case ingestprogress.StatusRunning:
		status = string(repoingest.StatusParsing)
	case ingestprogress.StatusComplete:
		status = string(repoingest.StatusReady)
	case ingestprogress.StatusFailed:
		status = string(repoingest.StatusFailed)
	}
	pct := float64(ev.Progress.Percent)
	if status == string(repoingest.StatusReady) || ev.Status == ingestprogress.StatusComplete {
		pct = 100
	}
	return socio.CodeIndexStatus{
		Status:          status,
		Stage:           stage,
		ProgressPercent: pct,
		FilesIndexed:    ev.Progress.ProcessedFiles,
	}
}
