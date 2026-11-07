// Package extract holds the language registry and extracts one file. Each
// registered LanguageSpec names its extensions, its parser and its
// extractor, and File runs that extractor on the source of one file with
// a parser that the caller owns.
//
// It does not import internal/graphstore or internal/indexer, and it
// starts no goroutine. The worker pool that calls File lives in
// internal/indexer.
package extract
