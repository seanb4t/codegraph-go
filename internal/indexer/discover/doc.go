// Package discover lists the source files of a repository in path order.
// It gives each file the language that its extension selects in the
// extract registry, skips what the DirFilter of the root skips, and
// records why it left out each file. A part of the index that no name rule
// skips gets no record.
//
// It imports extract for the language registry. It does not import
// internal/graphstore or internal/indexer, and it starts no goroutine.
package discover
