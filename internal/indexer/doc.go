// Package indexer implements the two-pass Go indexing pipeline: Pass 0
// (deterministic file discovery, this file's neighbor discover.go), Pass 1
// (parallel per-file extraction), and Pass 2 (sequential cross-file
// resolution), writing through internal/graphstore.
package indexer
