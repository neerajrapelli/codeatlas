package repoingest

import (
	"codeatlas/apps/api/internal/indexer"
)

// mapIndexerProgress maps per-stage progress (0–100) to overall ingest 0–100.
// Without embeddings: parse 0–50%, graph 50–100%. With embeddings: three equal thirds.
func mapIndexerProgress(stage Status, evt indexer.ProgressEvent, embeddingsEnabled bool) float64 {
	p := evt.Progress
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	if !embeddingsEnabled {
		switch stage {
		case StatusParsing:
			return p * 0.5
		case StatusBuildingGraph, StatusGeneratingEmbeddings:
			return 50 + p*0.5
		default:
			return p
		}
	}
	const third = 100.0 / 3.0
	switch stage {
	case StatusParsing:
		return p * third / 100.0
	case StatusBuildingGraph:
		return third + p*third/100.0
	case StatusGeneratingEmbeddings:
		return 2*third + p*third/100.0
	default:
		return p
	}
}
