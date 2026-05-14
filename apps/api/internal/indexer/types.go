package indexer

import (
	"context"
	"time"
)

type Request struct {
	RepositoryPath string
	RepositoryName string
	OnProgress     func(ProgressEvent)
}

type Result struct {
	RepositoryID     int64
	Files            int
	Symbols          int
	Imports          int
	Exports          int
	FileDependencies int
	Embeddings       int
	Duration         time.Duration
}

type Stage string

const (
	StageParsing              Stage = "parsing"
	StageBuildingGraph        Stage = "building_graph"
	StageGeneratingEmbeddings Stage = "generating_embeddings"
)

type ProgressEvent struct {
	Stage     Stage
	Progress  float64
	Files     int
	Symbols   int
	Edges     int
	Embeddings int
	Metadata  map[string]any
}

type ScannedFile struct {
	AbsolutePath string
	RelativePath string
}

type SymbolKind string

const (
	SymbolFunction  SymbolKind = "function"
	SymbolClass     SymbolKind = "class"
	SymbolInterface SymbolKind = "interface"
)

type Symbol struct {
	Name      string
	Kind      SymbolKind
	Exported  bool
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

type Import struct {
	ModulePath string
	TypeOnly   bool
}

type Export struct {
	Name       string
	SourcePath string
}

type ParsedFile struct {
	File    ScannedFile
	Symbols []Symbol
	Imports []Import
	Exports []Export
}

type IndexedFile struct {
	ParsedFile
	ResolvedDependencies []string
}

type Scanner interface {
	Scan(repoPath string) ([]ScannedFile, error)
}

type Parser interface {
	Parse(file ScannedFile) (ParsedFile, error)
}

type Store interface {
	UpsertRepositoryGraph(ctx context.Context, req PersistRequest) (PersistStats, error)
}

type Embedder interface {
	Embed(ctx context.Context, input string) ([]float32, error)
}

type PersistRequest struct {
	RepositoryPath string
	RepositoryName string
	IndexedFiles   []IndexedFile
	OnProgress     func(ProgressEvent)
}

type PersistStats struct {
	RepositoryID     int64
	Files            int
	Symbols          int
	Imports          int
	Exports          int
	FileDependencies int
	Embeddings       int
}
