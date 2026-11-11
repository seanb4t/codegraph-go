// Package indexer builds the code graph of a repository and writes it to
// internal/graphstore. Run indexes from scratch and Sync re-indexes the
// files that changed.
//
// Both use the same layers. The discover package lists the source files.
// The extract package extracts each file. A symbols.Builder collects the
// extracted declarations, and Freeze turns it into a read-only
// symbols.View, from which symbols.NewHierarchy builds the type hierarchy.
// The resolve package resolves the references of each file against the
// View and the Hierarchy, and stage writes records through a Sink, in key
// order.
//
// Run is a full index. It holds the working state of at most one file for
// each worker, whatever the size of the repository, and it replaces the
// store only when it succeeds. It runs three passes with two barriers
// between them, each pass on the same workers and the same claim loop
// (forEachFile):
//
//   - Pass 1 extracts each file and writes the result to a spill file
//     (the spill package) in a scratch directory. Route detection then
//     runs over the files of the languages whose detector fired.
//   - Barrier 1 builds the View from the declarations of every file.
//   - Pass 2 reads each result back, resolves its types, and keeps only
//     its structural edges.
//   - Barrier 2 builds the Hierarchy, and resolve.NewGlobal runs the
//     passes that need every file: the implements and overrides
//     synthesis, the overload set edges and the conformance index.
//   - Pass 3 reads each result back again, resolves it in full, and
//     resolve.Global.Assemble finishes its unit: the collapsed edges of
//     that file.
//
// One writer goroutine stages each unit into a bounded graphstore Writer,
// which commits a batch whenever the batch passes a cap. A file holds a
// token from the moment a worker claims it until the writer staged its
// unit, so the workers wait for the writer and not the reverse.
//
// The writer writes into a staging store in the scratch directory.
// graphstore.ReplaceStore then exchanges that store with the live store in
// one atomic step, so a run that fails or stops halfway leaves the
// previous graph in place. A full index holds the writer lock of the store
// (graphstore.LockWriter) for its whole run, and so does a sync, so one
// writer works on a store at a time. A query takes no writer lock and
// reads the previous graph until the exchange.
//
// Sync resolves the files that it reparses in one step, through
// resolve.Link, which loops over the same per-file assembly.
//
// This package is the only layer that imports internal/graphstore and the
// only one that starts goroutines.
package indexer
