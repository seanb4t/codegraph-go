// Command cpudiag is a one-shot CPU-topology diagnostic (10-04-PLAN, Rule 2
// deviation): it prints what a Go process on this host believes about its
// own parallelism, for comparison against the host's REAL CPU allotment.
//
// Why this exists: tools/bench/runner invokes `codegraph index --force
// --quiet` without ever passing --workers, so the extraction worker pool
// defaults to runtime.NumCPU() (internal/indexer/extract.go,
// internal/indexer/pipeline.go, internal/indexer/sync.go all default this
// way). runtime.NumCPU() reports HOST-VISIBLE CPUs, not a container's
// cgroup CPU quota — on a runner where those two numbers diverge, Go
// spawns far more OS threads/goroutines than the cgroup can actually run
// concurrently, and the resulting scheduler contention shows up as
// wall-clock throughput variance with stable memory usage, not as a code
// regression. This tool exists to check that hypothesis before acting on
// it, per the "run the control before concluding" lesson in
// tools/bench/BASELINE.md.
//
// This is a diagnostic, not a gate: it never fails the process (no
// os.Exit(1) path) and it is never invoked by any CI gate — only by the
// `task diag:cpu` target, itself only ever dispatched manually.
package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf("runtime.NumCPU(): %d\n", runtime.NumCPU())
	fmt.Printf("runtime.GOMAXPROCS(0): %d\n", runtime.GOMAXPROCS(0))
}
