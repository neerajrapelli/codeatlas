package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"codeatlas/apps/api/internal/ai"
)

func (a *API) registerAIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /ai/chat", a.handleAIChat)
	mux.HandleFunc("POST /repositories/{id}/ai/validate-mentions", a.handleValidateMentions)
}

func (a *API) handleValidateMentions(w http.ResponseWriter, r *http.Request) {
	if a.aiService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "AI service is not configured"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	var body ai.ValidateMentionsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	resp, err := a.aiService.ValidateMentions(ctx, repoID, tenantFromRequest(r.Context()), body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleAIChat(w http.ResponseWriter, r *http.Request) {
	if a.aiService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "AI service is not configured"})
		return
	}

	var req ai.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.RepositoryID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repositoryId is required"})
		return
	}
	if !guardRepository(w, r, a.pool, int64(req.RepositoryID)) {
		return
	}
	req.TenantID = tenantFromRequest(r.Context())

	if !req.Stream {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		resp, err := a.aiService.Answer(ctx, req)
		if err != nil {
			slog.Error("ai_chat_failed", "error", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	prepared, err := a.aiService.PrepareChat(ctx, req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	if prepared.ContextFileCount == 0 {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
			return
		}
		_ = writeSSE(w, map[string]any{
			"type":         "meta",
			"relatedFiles": prepared.RelatedFiles,
			"provider":     string(prepared.Provider),
			"model":        prepared.Model,
		})
		flusher.Flush()
		_ = writeSSE(w, map[string]any{"type": "token", "token": "I could not find relevant indexed context for this repository yet."})
		_ = writeSSE(w, map[string]any{"type": "done"})
		flusher.Flush()
		return
	}

	chunks, chunkErrs, err := a.aiService.StreamCompletion(ctx, prepared)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}

	if err := writeSSE(w, map[string]any{
		"type":         "meta",
		"relatedFiles": prepared.RelatedFiles,
		"provider":     string(prepared.Provider),
		"model":        prepared.Model,
	}); err != nil {
		return
	}
	flusher.Flush()

	var answerBuf strings.Builder
	streamDone := false
	for chunks != nil || chunkErrs != nil {
		select {
		case err, ok := <-chunkErrs:
			if !ok {
				chunkErrs = nil
				continue
			}
			if err != nil {
				_ = writeSSE(w, map[string]any{"type": "error", "error": err.Error()})
				flusher.Flush()
				return
			}
		case ch, ok := <-chunks:
			if !ok {
				chunks = nil
				continue
			}
			if ch.Delta != "" {
				answerBuf.WriteString(ch.Delta)
				if err := writeSSE(w, map[string]any{"type": "token", "token": ch.Delta}); err != nil {
					return
				}
				flusher.Flush()
			}
			if ch.Done {
				streamDone = true
				chunks = nil
			}
		}
	}
	if streamDone && answerBuf.Len() > 0 {
		sanitized, validation, err := a.aiService.GuardStreamAnswer(ctx, req.RepositoryID, req.TenantID, answerBuf.String())
		if err != nil {
			slog.Warn("ai_guard_failed", "error", err)
		} else if sanitized != answerBuf.String() {
			_ = writeSSE(w, map[string]any{"type": "validated", "content": sanitized, "validation": validation})
			flusher.Flush()
		} else {
			_ = writeSSE(w, map[string]any{"type": "validated", "content": sanitized, "validation": validation})
			flusher.Flush()
		}
	}
	_ = writeSSE(w, map[string]any{"type": "done"})
	flusher.Flush()
}
