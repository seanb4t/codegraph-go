// Package indexer builds the code graph of a repository and writes it to
// internal/graphstore. Run indexes from scratch and Sync re-indexes the
// files that changed.
//
// Both run the same layers in order. The discover package lists the
// source files. A bounded pool of workers extracts each file with the
// extract package. A symbols.Builder collects the extracted symbols, and
// Freeze turns it into a read-only symbols.View, from which
// symbols.NewHierarchy builds the type hierarchy. The resolve package
// resolves the references of each file against the View and the
// Hierarchy, and resolve.Link joins the files into one graph. stage
// writes that graph through a Sink, in key order.
//
// This package is the only layer that imports internal/graphstore and the
// only one that starts goroutines.
package indexer
