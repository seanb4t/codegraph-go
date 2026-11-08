// Package symbols holds the symbol tables that resolution reads. A Builder
// collects the nodes of every file, the members of each type and the
// overload sets and edges already in the store. Freeze returns a View
// that reads them and cannot change them, and a write to the Builder
// after Freeze panics. NewHierarchy builds the type hierarchy and the
// overload sets from a View and the type-level edges.
//
// It does not import internal/graphstore, internal/indexer or the resolve
// package, and it starts no goroutine.
package symbols
