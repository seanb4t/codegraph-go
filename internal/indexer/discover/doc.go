// Package discover lists the source files of a repository in path order.
// It gives each file the language that its extension selects in the
// extract registry, skips the directories that ShouldSkipDir names, and
// records why each excluded file was left out.
//
// It imports extract for the language registry. It does not import
// internal/graphstore or internal/indexer, and it starts no goroutine.
package discover
