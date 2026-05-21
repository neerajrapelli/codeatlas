package indexer

import (
	"context"
	"runtime"
	"sync"
)

const defaultParseWorkers = 8

// ParseWorkers returns the bounded worker count for Tree-sitter parsing.
func ParseWorkers(requested int) int {
	if requested <= 0 {
		n := runtime.NumCPU()
		if n < 2 {
			n = 2
		}
		if n > defaultParseWorkers {
			n = defaultParseWorkers
		}
		return n
	}
	if requested > 32 {
		return 32
	}
	return requested
}

type parserPool struct {
	ch chan Parser
}

func newParserPool(workers int, factory func() Parser) *parserPool {
	if workers < 1 {
		workers = 1
	}
	p := &parserPool{ch: make(chan Parser, workers)}
	for i := 0; i < workers; i++ {
		p.ch <- factory()
	}
	return p
}

func (p *parserPool) with(fn func(Parser) (ParsedFile, error)) (ParsedFile, error) {
	parser := <-p.ch
	defer func() { p.ch <- parser }()
	return fn(parser)
}

func parseFilesParallel(ctx context.Context, files []ScannedFile, factory func() Parser, workers int, onProgress func(done, total int)) ([]IndexedFile, error) {
	if len(files) == 0 {
		return nil, nil
	}
	workers = ParseWorkers(workers)
	if workers > len(files) {
		workers = len(files)
	}
	pool := newParserPool(workers, factory)

	type job struct {
		idx  int
		file ScannedFile
	}
	type result struct {
		idx int
		out IndexedFile
		err error
	}

	jobs := make(chan job)
	results := make(chan result, workers*2)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if ctx.Err() != nil {
					return
				}
				parsed, err := pool.with(func(p Parser) (ParsedFile, error) {
					return p.Parse(j.file)
				})
				results <- result{idx: j.idx, out: IndexedFile{ParsedFile: parsed}, err: err}
			}
		}()
	}

	go func() {
		for i, file := range files {
			jobs <- job{idx: i, file: file}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	out := make([]IndexedFile, len(files))
	done := 0
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		out[res.idx] = res.out
		done++
		if onProgress != nil && (done == len(files) || done%10 == 0) {
			onProgress(done, len(files))
		}
	}
	return out, nil
}
