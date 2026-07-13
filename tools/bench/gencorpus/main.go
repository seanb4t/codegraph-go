// Command gencorpus is the CLI wrapper around Generate: it materializes a
// deterministic, network-free synthetic source corpus for the PERF-02 CI
// regression gate and the INDX-06 100k+ file / bounded-memory assertion.
//
// Usage:
//
//	gencorpus -seed 42 -out /tmp/corpus -count 120000
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	seed := flag.Int64("seed", 42, "RNG seed; the same seed always produces the same output tree")
	out := flag.String("out", "", "output directory to materialize the corpus into (required)")
	count := flag.Int("count", ProductionFileCount, "total number of source files to generate")
	flag.Parse()

	if *out == "" {
		fmt.Fprintln(os.Stderr, "gencorpus: -out is required")
		os.Exit(2)
	}

	stats, err := Generate(Options{Seed: *seed, FileCount: *count, OutDir: *out})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gencorpus: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %d files (%d bytes) to %s (seed=%d)\n", stats.FilesWritten, stats.BytesWritten, *out, *seed)
}
