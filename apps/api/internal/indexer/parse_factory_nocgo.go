//go:build !cgo

package indexer

func newParserFactory(base Parser) func() Parser {
	return func() Parser { return base }
}
