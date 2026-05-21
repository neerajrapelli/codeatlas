//go:build cgo

package indexer

func newParserFactory(_ Parser) func() Parser {
	return func() Parser { return NewTreeSitterParser() }
}
