package indexer

import "testing"

func TestCountPlannedEmbeddings(t *testing.T) {
	files := []IndexedFile{
		{ParsedFile: ParsedFile{Imports: []Import{{}, {}}, Symbols: []Symbol{{}}}},
		{ParsedFile: ParsedFile{Symbols: []Symbol{{}, {}}}},
	}
	got := countPlannedEmbeddings(files)
	if got != 6 {
		t.Fatalf("planned embeddings: got %d want 6", got)
	}
}

func TestPostgresStore_canEmbed_respectsCap(t *testing.T) {
	s := &PostgresStore{embedder: struct{ Embedder }{}, embeddingMaxPerRepo: 2}
	stats := PersistStats{}
	if !s.canEmbed(&stats) {
		t.Fatal("expected first slot")
	}
	stats.Embeddings++
	if !s.canEmbed(&stats) {
		t.Fatal("expected second slot")
	}
	stats.Embeddings++
	if s.canEmbed(&stats) {
		t.Fatal("expected cap at 2")
	}
}

func TestPostgresStore_canEmbed_unlimitedWhenZero(t *testing.T) {
	s := &PostgresStore{embedder: struct{ Embedder }{}, embeddingMaxPerRepo: 0}
	stats := PersistStats{Embeddings: 10_000}
	if !s.canEmbed(&stats) {
		t.Fatal("zero cap means unlimited")
	}
}
