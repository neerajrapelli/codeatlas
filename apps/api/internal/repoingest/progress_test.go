package repoingest

import (
	"testing"

	"codeatlas/apps/api/internal/indexer"
)

func TestMapIndexerProgress_noEmbeddings_reaches100OnGraph(t *testing.T) {
	p := mapIndexerProgress(StatusParsing, indexer.ProgressEvent{Progress: 100}, false)
	if p < 49 || p > 51 {
		t.Fatalf("parse 100%%: got %v want 50", p)
	}
	p = mapIndexerProgress(StatusBuildingGraph, indexer.ProgressEvent{Progress: 100}, false)
	if p < 99 || p > 100 {
		t.Fatalf("graph 100%%: got %v want 100", p)
	}
}

func TestMapIndexerProgress_withEmbeddings_thirds(t *testing.T) {
	p := mapIndexerProgress(StatusParsing, indexer.ProgressEvent{Progress: 100}, true)
	if p < 32 || p > 34 {
		t.Fatalf("parse 100%%: got %v want ~33.3", p)
	}
	p = mapIndexerProgress(StatusBuildingGraph, indexer.ProgressEvent{Progress: 100}, true)
	if p < 65 || p > 67 {
		t.Fatalf("graph 100%%: got %v want ~66.7", p)
	}
	p = mapIndexerProgress(StatusGeneratingEmbeddings, indexer.ProgressEvent{Progress: 100}, true)
	if p < 99 || p > 100 {
		t.Fatalf("embed 100%%: got %v want 100", p)
	}
}
