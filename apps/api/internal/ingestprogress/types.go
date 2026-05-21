package ingestprogress

import (
	"context"
	"sync"
	"time"
)

// Step names — frontend depends on these exact strings.
const (
	StepCloneRepository       = "clone_repository"
	StepExtractWorkspace      = "extract_workspace"
	StepIndexWorkspace        = "index_workspace"
	StepParseSources          = "parse_sources"
	StepBuildDependencyGraph  = "build_dependency_graph"
	StepSemanticEmbeddings    = "semantic_embeddings"
)

// Step status values.
const (
	StepPending  = "pending"
	StepRunning  = "running"
	StepComplete = "complete"
	StepFailed   = "failed"
)

// Stream status values.
const (
	StatusQueued   = "queued"
	StatusRunning  = "running"
	StatusComplete = "complete"
	StatusFailed   = "failed"
)

// StreamEvent is the SSE payload shape consumed by the web UI.
type StreamEvent struct {
	Phase       int          `json:"phase"`
	Status      string       `json:"status"`
	CurrentStep string       `json:"current_step"`
	Steps       []Step       `json:"steps"`
	Progress    FileProgress `json:"progress"`
	ETASeconds   *int         `json:"eta_seconds"`
}

type Step struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMs *int64 `json:"duration_ms"`
}

type FileProgress struct {
	TotalFiles     int `json:"total_files"`
	ProcessedFiles int `json:"processed_files"`
	Percent        int `json:"percent"`
}

// Broadcaster holds the latest SSE event per repository (fast path).
type Broadcaster struct {
	mu sync.RWMutex
	m  map[int64]StreamEvent
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{m: make(map[int64]StreamEvent)}
}

func (b *Broadcaster) Publish(repositoryID int64, ev StreamEvent) {
	b.mu.Lock()
	b.m[repositoryID] = ev
	b.mu.Unlock()
}

func (b *Broadcaster) Get(repositoryID int64) (StreamEvent, bool) {
	b.mu.RLock()
	ev, ok := b.m[repositoryID]
	b.mu.RUnlock()
	return ev, ok
}

func (b *Broadcaster) Delete(repositoryID int64) {
	b.mu.Lock()
	delete(b.m, repositoryID)
	b.mu.Unlock()
}

func (b *Broadcaster) Latest(repositoryID int64) (StreamEvent, bool) {
	return b.Get(repositoryID)
}

func (b *Broadcaster) Subscribe(ctx context.Context, repositoryID int64) (<-chan StreamEvent, func(), error) {
	_ = ctx
	_ = repositoryID
	return nil, func() {}, nil
}

// DefaultSteps returns the canonical step list (all pending).
func DefaultSteps() []Step {
	names := []string{
		StepCloneRepository,
		StepExtractWorkspace,
		StepIndexWorkspace,
		StepParseSources,
		StepBuildDependencyGraph,
		StepSemanticEmbeddings,
	}
	out := make([]Step, len(names))
	for i, n := range names {
		out[i] = Step{Name: n, Status: StepPending}
	}
	return out
}

// NewQueuedEvent builds the initial SSE event for a repository.
func NewQueuedEvent(repositoryID int64, phase int) StreamEvent {
	_ = repositoryID
	return StreamEvent{
		Phase:       phase,
		Status:      StatusQueued,
		CurrentStep: StepCloneRepository,
		Steps:       DefaultSteps(),
		Progress:    FileProgress{},
	}
}

// StepDurations tracks completed step timings for SSE.
type StepDurations struct {
	mu   sync.Mutex
	ms   map[string]int64
	start map[string]time.Time
}

func NewStepDurations() *StepDurations {
	return &StepDurations{
		ms:    make(map[string]int64),
		start: make(map[string]time.Time),
	}
}

func (d *StepDurations) Start(step string) {
	d.mu.Lock()
	d.start[step] = time.Now()
	d.mu.Unlock()
}

func (d *StepDurations) Complete(step string) {
	d.mu.Lock()
	if t, ok := d.start[step]; ok {
		ms := time.Since(t).Milliseconds()
		d.ms[step] = ms
		delete(d.start, step)
	}
	d.mu.Unlock()
}

func (d *StepDurations) Snapshot() map[string]int64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make(map[string]int64, len(d.ms))
	for k, v := range d.ms {
		out[k] = v
	}
	return out
}

func applyDurations(steps []Step, durations map[string]int64) []Step {
	out := make([]Step, len(steps))
	for i, s := range steps {
		out[i] = s
		if ms, ok := durations[s.Name]; ok {
			v := ms
			out[i].DurationMs = &v
		}
	}
	return out
}

// MapRepoStage maps repoingest status strings to stream step + overall status.
func BuildEvent(phase int, repoStatus string, currentStep string, filesIndexed, totalEstimate int, percent float64, durations map[string]int64) StreamEvent {
	steps := DefaultSteps()
	status := StatusRunning
	if repoStatus == "queued" {
		status = StatusQueued
	}
	if repoStatus == "ready" {
		status = StatusComplete
		currentStep = StepSemanticEmbeddings
	}
	if repoStatus == "failed" {
		status = StatusFailed
	}

	for i := range steps {
		steps[i].Status = stepStatusFor(steps[i].Name, currentStep, repoStatus, durations)
	}
	steps = applyDurations(steps, durations)

	pct := int(percent)
	if pct > 100 {
		pct = 100
	}
	processed := filesIndexed
	total := totalEstimate
	if total < processed {
		total = processed
	}
	if status == StatusComplete {
		pct = 100
		if total == 0 && processed > 0 {
			total = processed
		}
	}

	var eta *int
	if status == StatusRunning && pct > 0 && pct < 100 {
		remaining := 100 - pct
		sec := (remaining * 120) / pct
		if sec < 10 {
			sec = 10
		}
		eta = &sec
	}

	return StreamEvent{
		Phase:       phase,
		Status:      status,
		CurrentStep: currentStep,
		Steps:       steps,
		Progress: FileProgress{
			TotalFiles:     total,
			ProcessedFiles: processed,
			Percent:        pct,
		},
		ETASeconds: eta,
	}
}

func stepStatusFor(name, current, repoStatus string, done map[string]int64) string {
	if repoStatus == "failed" && name == current {
		return StepFailed
	}
	if repoStatus == "ready" {
		return StepComplete
	}
	if _, ok := done[name]; ok {
		return StepComplete
	}
	if name == current {
		return StepRunning
	}
	order := []string{StepCloneRepository, StepExtractWorkspace, StepIndexWorkspace, StepParseSources, StepBuildDependencyGraph, StepSemanticEmbeddings}
	curIdx := indexOf(order, current)
	nameIdx := indexOf(order, name)
	if curIdx >= 0 && nameIdx >= 0 && nameIdx < curIdx {
		return StepComplete
	}
	return StepPending
}

func indexOf(slice []string, v string) int {
	for i, s := range slice {
		if s == v {
			return i
		}
	}
	return -1
}

// StepToRepoStage maps SSE step names back to repository status strings for API/UI.
func StepToRepoStage(step string) string {
	switch step {
	case StepCloneRepository:
		return "cloning"
	case StepExtractWorkspace:
		return "extracting"
	case StepIndexWorkspace:
		return "indexing"
	case StepParseSources:
		return "parsing"
	case StepBuildDependencyGraph:
		return "building_graph"
	case StepSemanticEmbeddings:
		return "generating_embeddings"
	default:
		return ""
	}
}

// CurrentStepForProgress maps DB stage + overall percent to the active UI step.
func CurrentStepForProgress(repoStage string, percent float64, embeddingsEnabled bool) string {
	if repoStage == "parsing" {
		threshold := 50.0
		if embeddingsEnabled {
			threshold = 100.0 / 3.0
		}
		if percent >= threshold {
			return StepBuildDependencyGraph
		}
	}
	return RepoStatusToCurrentStep(repoStage)
}

// RepoStatusToCurrentStep maps repository ingest status to SSE step name.
func RepoStatusToCurrentStep(repoStatus string) string {
	switch repoStatus {
	case "cloning":
		return StepCloneRepository
	case "extracting":
		return StepExtractWorkspace
	case "indexing":
		return StepIndexWorkspace
	case "parsing":
		return StepParseSources
	case "building_graph":
		return StepBuildDependencyGraph
	case "generating_embeddings":
		return StepSemanticEmbeddings
	case "ready":
		return StepSemanticEmbeddings
	default:
		return StepCloneRepository
	}
}
