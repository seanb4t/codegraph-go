// Package realcorpus is the pinned real-repo manifest for the PERF-01
// head-to-head benchmark: the freshly-built Go `codegraph` binary vs the
// installed TS `codegraph@1.3.1`, run over real, publicly available
// source trees rather than the synthetic PERF-02/INDX-06 corpus
// (tools/bench/gencorpus).
//
// Every entry MUST carry a commit SHA — never just a branch or tag name
// — so a published head-to-head number stays reproducible run-to-run and
// machine-to-machine (CONTEXT.md D-04's reproducibility discipline,
// extended from the synthetic corpus to real repos). This reuses the
// exact provenance shape established by
// tools/spike/testdata/ATTRIBUTION.md (recoverable via
// `git show e5da8e7:tools/spike/testdata/ATTRIBUTION.md`) and by this
// project's own testdata/golden/README.md "Corpus (D-06a)" table, which
// already pins two of these three entries (weft-go, colbymchenry-
// codegraph) for the Phase-3 golden-parity oracle.
//
// This package performs no network I/O: Resolve only reports a local
// checkout path when one already exists (via an env var override or the
// conventional sibling-checkout directory) or returns ErrNeedsClone
// otherwise. Any actual shallow clone is the caller's responsibility
// (tools/bench/runner, or the bench.yml CI job), and MUST be pinned to
// exactly the entry's CommitSHA — never at HEAD (V5: no
// attacker-controlled or unpinned ref is ever resolved into a subprocess
// argument).
package realcorpus

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Entry is one pinned real-repo benchmark input. Every field below (aside
// from Tag, which may legitimately be empty for a pin taken at a
// non-tagged commit) is required so a head-to-head number is always
// traceable back to an exact source, license, and file set — the same
// discipline tools/spike/testdata/ATTRIBUTION.md established.
type Entry struct {
	// Name is a short, filesystem-safe identifier used for cache
	// directory names and reported bench.Metrics.Repo values.
	Name string
	// SourceURL is the canonical repo URL (https, no ".git" suffix)
	// a caller should shallow-clone if no local checkout is found.
	SourceURL string
	// Tag is the human-readable release ref the pin corresponds to
	// (e.g. "v2.1.6"). May be empty when the pin was taken at a
	// repo's default-branch HEAD at capture time rather than a
	// tagged release — CommitSHA is authoritative either way.
	Tag string
	// CommitSHA is the exact pinned commit every head-to-head run
	// MUST resolve to. This is the one field that makes a published
	// number reproducible (see package doc).
	CommitSHA string
	// License is the repo's SPDX-style license identifier at
	// CommitSHA.
	License string
	// SelectionRule documents exactly what subset of the repo's tree
	// is indexed, mirroring tools/spike/testdata/ATTRIBUTION.md's
	// "Selection" field discipline.
	SelectionRule string
	// QueryTerms is the fixed query set (D-05) run against this
	// entry's index for the query-latency metric: real symbol names
	// confirmed present in the source tree at CommitSHA, so every
	// measured query is a genuine (non-empty) lookup rather than an
	// arbitrary string that may or may not match.
	QueryTerms []string
	// EnvVar, if set, is checked first by Resolve: an operator can
	// point it at an existing local checkout to skip cloning
	// entirely. Mirrors testdata/golden's own CODEGRAPH_WEFT_CORPUS
	// convention (see testdata/golden/golden_parity_test.go).
	EnvVar string
	// SiblingDir, if set, is a conventional local checkout path
	// (relative to this repo's parent directory) Resolve checks
	// second, before reporting ErrNeedsClone.
	SiblingDir string
}

// Corpora returns the fixed, pinned real-repo manifest for the PERF-01
// head-to-head benchmark. Order is deliberate: compact-and-already-
// vendored-adjacent first, multi-language second, larger-scale third
// (per CONTEXT.md D-04's "plus a few larger real repos" + this plan's
// "at least one larger real repo pinned by commit SHA to exercise
// scale" acceptance criterion).
func Corpora() []Entry {
	return []Entry{
		{
			// Same pin testdata/golden/README.md's D-06a corpus table
			// and testdata/golden/golden_parity_test.go's
			// resolveWeftCorpus already use for the Phase-3 golden
			// oracle — reusing it here keeps the project's set of
			// pinned real repos small and consistent rather than
			// introducing a fourth unrelated pin.
			Name:          "weft-go",
			SourceURL:     "https://github.com/seanb4t/weft",
			Tag:           "",
			CommitSHA:     "f89ae3ea4e4c37509f7302fd4e37986212a72079",
			License:       "Apache-2.0",
			SelectionRule: "entire repo at CommitSHA (compact, ~84 files, mostly Go)",
			QueryTerms:    []string{"Load", "Install"},
			EnvVar:        "CODEGRAPH_WEFT_CORPUS",
			SiblingDir:    "weft",
		},
		{
			// The original TS CodeGraph project this Go port
			// replaces — multi-language (TS/JS/Python/Astro/YAML),
			// exercises the tool surface broadly. Same pin
			// testdata/golden/README.md's D-06a corpus table uses.
			Name:          "colbymchenry-codegraph",
			SourceURL:     "https://github.com/colbymchenry/codegraph",
			Tag:           "",
			CommitSHA:     "edb9f2f14cd7394a4d31f94ebc871531ef498ab0",
			License:       "MIT",
			SelectionRule: "entire repo at CommitSHA (multi-language: TS/JS/Python/Astro/YAML)",
			QueryTerms:    []string{"ExtractionOrchestrator", "TreeSitterExtractor"},
			EnvVar:        "CODEGRAPH_TSCG_CORPUS",
			SiblingDir:    "codegraph-ts",
		},
		{
			// A real, substantially larger Go codebase — this
			// project's own Pebble storage dependency (see
			// .claude/CLAUDE.md's Storage recommendation) — chosen to
			// exercise PERF-01 at scale beyond weft-go's compact 84
			// files, per this plan's "at least one larger real repo"
			// acceptance criterion.
			Name:          "cockroachdb-pebble",
			SourceURL:     "https://github.com/cockroachdb/pebble",
			Tag:           "v2.1.6",
			CommitSHA:     "dbdc1acb859689dc4237b40ef8fcdbb877526a84",
			License:       "BSD-3-Clause",
			SelectionRule: "entire repo at CommitSHA",
			QueryTerms:    []string{"Open", "DB"},
			EnvVar:        "CODEGRAPH_PEBBLE_CORPUS",
			SiblingDir:    "pebble",
		},
	}
}

// ErrNeedsClone is returned by Resolve when no local checkout of an
// Entry is present via either its EnvVar override or its conventional
// sibling-checkout directory. The caller (tools/bench/runner or a CI
// job) is responsible for shallow-cloning Entry.SourceURL and checking
// out exactly Entry.CommitSHA — never HEAD — into a scratch directory
// when this is returned. realcorpus itself never touches the network.
var ErrNeedsClone = errors.New("realcorpus: no local checkout found; shallow-clone SourceURL at CommitSHA")

// Resolve returns a local filesystem path containing e's source tree,
// checked in this order:
//  1. e.EnvVar, if set and pointing at an existing directory (operator
//     override, mirrors CODEGRAPH_WEFT_CORPUS).
//  2. e.SiblingDir, if set, resolved relative to this repo's parent
//     directory (the "../weft next to this repo" convention already
//     established by testdata/golden/golden_parity_test.go).
//
// Resolve does not verify the checkout is actually pinned at
// e.CommitSHA — callers that need that guarantee (as
// golden_parity_test.go does for weft, and as
// tools/bench/runner.resolveOrClone's pinnedAt check does for the
// PERF-01 head-to-head benchmark, WR-02 Phase 8 re-review) should check
// it themselves; this function's job is only path discovery, kept
// deliberately simple so it has no network or git-plumbing surface of
// its own beyond a single local `git rev-parse --show-toplevel` to
// locate the sibling directory.
func (e Entry) Resolve() (string, error) {
	if e.EnvVar != "" {
		if p := os.Getenv(e.EnvVar); p != "" {
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				return p, nil
			}
		}
	}
	if e.SiblingDir != "" {
		if root, err := repoRootDir(); err == nil {
			candidate := filepath.Join(filepath.Dir(root), e.SiblingDir)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("%w: %s (%s)", ErrNeedsClone, e.Name, e.SourceURL)
}

// repoRootDir returns this checkout's top-level directory via `git
// rev-parse --show-toplevel`, run from the current working directory.
// This is local git metadata only — no network access — used solely to
// anchor SiblingDir's "next to this repo" convention regardless of the
// caller's own cwd.
func repoRootDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
