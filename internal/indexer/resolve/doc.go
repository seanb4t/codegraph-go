// Package resolve turns extracted files into graph edges. Types resolves
// the type-level references of one file, and Calls resolves its calls.
// Link joins the outputs of every file into one Graph and adds the edges
// that need every file: implements, overrides, the conformance retry and
// the overload set edges. CollapseEdges merges duplicate edges.
//
// It reads only a symbols.View and a symbols.Hierarchy. It does not import
// internal/graphstore or internal/indexer, and it starts no goroutine.
package resolve
