---
phase: 08-surface-reconciliation-signed-v1-0-0-release
reviewed: 2026-07-19T00:00:00Z
depth: deep
files_reviewed: 24
files_reviewed_list:
  - internal/query/validate.go
  - internal/query/files.go
  - internal/query/traverse.go
  - internal/query/engine_test.go
  - internal/query/files_status_test.go
  - internal/query/traverse_test.go
  - internal/cli/impact.go
  - internal/cli/files.go
  - internal/cli/affected.go
  - internal/cli/affected_test.go
  - internal/cli/callers.go
  - internal/cli/callees.go
  - internal/cli/query.go
  - internal/cli/status.go
  - internal/cli/install.go
  - internal/cli/uninstall.go
  - internal/cli/upgrade.go
  - internal/cli/flag_parity_test.go
  - internal/cli/present/archtest/charm_cgo_test.go
  - internal/upgrade/upgrade.go
  - internal/upgrade/upgrade_test.go
  - docs/FLAG-PARITY.md
  - docs/RELEASE-PROCEDURES.md
  - docs/BENCHMARKS.md
findings:
  critical: 1
  warning: 7
  info: 1
  total: 9
status: issues_found
---

# Phase 08: Code Review Report

**Reviewed:** 2026-07-19
**Depth:** deep
**Files Reviewed:** 24
**Status:** issues_found

## Summary

Reviewed the Phase 8 surface-reconciliation changes at deep depth: the two
distinct depth-clamp defaults (`clampDepth`/`defaultDepth=2` for `impact`
vs `clampAffectedDepth`/`defaultAffectedDepth=5` for `affected`), the new
depth-bounded `Affected` BFS with test-leaf pruning, `files --dir`'s
prefix-match filter, the `affected --stdin` untrusted-input path, and the
`upgrade --force` same-version guard's interaction with signature
verification.

**Things verified as correct, not flagged:**
- `clampDepth`/`clampAffectedDepth` are NOT cross-wired — `Impact` calls
  `clampDepth` (default 2), `Affected` calls `clampAffectedDepth` (default
  5); both share `MaxDepth=50` as documented, and both reject negative
  depths via `validateDepth` before clamping (`internal/query/traverse.go:395-399,491-495`).
- `upgrade --force`'s same-version bypass is genuinely narrow: `Run` only
  skips the `target == currentVersion` early-return; `verify()` still runs
  unconditionally before `swap()` in every code path, and
  `upgrade_test.go`'s `TestUpgradeRun_ForceStillVerifiesBeforeSwap`
  directly proves a failing verify aborts before swap even with
  `Force: true` (`internal/upgrade/upgrade.go:82-153`). No weakening found.
- `affected --stdin` does not hang on empty/closed stdin —
  `bufio.Scanner.Scan()` returns `false` cleanly on EOF, and
  `TestAffectedStdinNeverHangs`/`TestAffectedEmptyStdinNoArgs` exercise this
  with a real timeout guard.

**Genuine defects found**, detailed below: an unbounded-input DoS gap in
`affected --stdin` that the codebase's own `MaxLimit`/`MaxFiles` validation
convention does not cover; a classic prefix-match boundary bug in
`dirPrefixMatches`; a JSON-null inconsistency on `AffectedResult.Files`
that the codebase's own WR-01 convention was supposed to close everywhere;
an unescaped-path injection risk in `affected --quiet`'s machine-readable
output; and a cross-file behavioral inconsistency where `Affected`'s
reverse BFS does not apply the same interface-dispatch composition that
`Callers`/`Impact` both apply.

## Critical Issues

### CR-01: `affected --stdin` has no cap on the number of ingested lines — unbounded-memory DoS

**File:** `internal/cli/affected.go:150-179` (compounded by `internal/query/traverse.go:491-558`)

**Issue:** Every other numeric input surface in this package is explicitly
bounded against exactly this class of attack — `internal/query/validate.go`
documents `MaxDepth`, `MaxLimit`, and `MaxFiles` as existing specifically
"so a caller — human or an untrusted/compromised MCP client — cannot force
an unbounded traversal or allocation just by passing a large number" (V5
Input Validation, T-03-02-DoS). `affected --stdin` is explicitly documented
as ingesting an **untrusted** path list (SURF-04/T-08-05-01), yet
`collectAffectedFiles` (internal/cli/affected.go:150) reads every line from
`cmd.InOrStdin()` via `bufio.NewScanner` with **no upper bound on line
count**, deduplicating into an ever-growing `seen` map and `files` slice.
`Engine.Affected` (internal/query/traverse.go:491) then builds
`fileSet := make(map[string]bool, len(files))` from that same
unbounded-size slice with no `validateMaxFiles`-style rejection.

A hostile or merely oversized input source (a compromised git hook, a CI
step piping an attacker-influenced diff, or simply `yes | codegraph
affected --stdin`) can grow `files`/`seen`/`fileSet` without limit before
`Affected` ever runs its bounded graph scan, exhausting memory in a
long-running process (this matters most for the future MCP server, which
this exact package is designed to be reused by, per
`BuildReverseAdjacency`'s doc comment on `internal/query/traverse.go:29-31`).

**Fix:** Cap the number of lines/files accepted, mirroring
`validateMaxFiles`'s posture (reject outright with a clear error rather
than silently truncate):

```go
// in internal/query/validate.go
const MaxAffectedFiles = 10000 // or reuse MaxFiles

func validateAffectedFiles(n int) error {
	if n > MaxAffectedFiles {
		return fmt.Errorf("query: %d files exceeds maximum %d", n, MaxAffectedFiles)
	}
	return nil
}
```

```go
// in internal/cli/affected.go's collectAffectedFiles, or Engine.Affected
if stdinFlag {
	scanner := bufio.NewScanner(cmd.InOrStdin())
	for scanner.Scan() {
		if len(files) >= query.MaxAffectedFiles {
			break // or return an explicit error
		}
		add(strings.TrimSpace(scanner.Text()))
	}
}
```

## Warnings

### WR-01: `dirPrefixMatches` has no path-separator boundary check — `--dir foo` also matches sibling directory `foobar/`

**File:** `internal/query/files.go:103-108`

**Issue:** `dirPrefixMatches` is a plain `strings.HasPrefix(path, dir)`
check with no requirement that the match land on a path-separator
boundary. Given two sibling directories `pkga/` and `pkgab/` (or any pair
where one directory name is a literal string-prefix of another),
`--dir pkga` matches BOTH `pkga/foo.go` and `pkgab/bar.go` — the classic
prefix-match-without-boundary bug. The existing test table
(`internal/query/files_status_test.go:248-268`, `"dirPrefixMatches: plain
prefix semantics, not a glob"`) only exercises dir values that already end
in `/` (`"internal/"`, `"internal/query"` against a path that has a `/`
right after), so it never exercises the sibling-collision case. This is
distinct from — and not excused by — the intentionally-locked "prefix
match, not a glob" design (D-03): the gap is the missing separator
boundary, not the choice of prefix-vs-glob semantics.

**Fix:** Require the match to land on a path boundary (either an exact
segment match or followed by `/`), while still keeping the "not a glob"
contract:

```go
func dirPrefixMatches(path, dir string) bool {
	if dir == "" {
		return true
	}
	for _, p := range []string{path, "./" + path} {
		if p == dir || strings.HasPrefix(p, strings.TrimSuffix(dir, "/")+"/") {
			return true
		}
	}
	return false
}
```

(If this diverges from TS 1.3.1's own `bin/codegraph.js:1348-1354` behavior,
confirm TS has the identical bug before deciding whether to fix or
document-and-accept — but the current Go behavior should not be assumed
correct-by-parity without checking.)

### WR-02: `AffectedResult.Files` has no `omitempty`/empty-slice normalization — a nil `files` argument marshals as JSON `null`

**File:** `internal/query/traverse.go:258-261, 491-558`

**Issue:** `AffectedResult.Files []string \`json:"files"\`` carries no
`omitempty`, and `Engine.Affected` returns it as a direct pass-through of
the caller-supplied `files` parameter (`AffectedResult{Files: files, ...}`
at line 557) with no defensive normalization. When a caller passes `nil`
(exactly what `internal/query/traverse_test.go`'s
`TestAffectedEmptyFilesReturnsEmptyResultNoError` does at line 624, though
that test does not check JSON marshaling), `MarshalAffectedJSON` produces
`{"files":null,"affectedTests":[]}`.

This is precisely the class of bug the codebase already fixed for
`Callees`/`Callers`/`AffectedTests` — see `WR-01` in
`internal/query/traverse.go`'s doc comments and
`TestZeroMatchJSONShapesAreEmptyArraysNotNull`
(`internal/query/traverse_test.go:773-826`) — but that regression test
suite never exercises `Files` itself. The current CLI path
(`internal/cli/affected.go`'s `collectAffectedFiles`) happens to always
build a non-nil `make([]string, 0, len(args))`, masking the bug for
today's only caller, but any other caller of the exported `Engine.Affected`
(e.g. a future MCP tool) that passes `nil` would produce a `null` array a
downstream JSON consumer must not observe, per this codebase's own stated
convention.

**Fix:** Normalize defensively inside `Affected`, matching `tests :=
[]Location{}`'s existing pattern:

```go
func (e *Engine) Affected(files []string, depth int) (AffectedResult, error) {
	...
	if files == nil {
		files = []string{}
	}
	...
	return AffectedResult{Files: files, AffectedTests: tests}, nil
}
```

### WR-03: `affected --quiet` writes raw, unescaped file paths — a path containing embedded control characters can inject extra lines into machine-readable output

**File:** `internal/cli/affected.go:98-110`

**Issue:** The `--quiet` branch is explicitly documented as a
git-hook/CI-pipeline-safe, one-path-per-line machine-readable contract
("Safe to pipe straight into another command"). It writes
`fmt.Fprintf(out, "%s\n", l.FilePath)` for each affected test's `FilePath`
— sourced from the indexed graph's file paths, not directly from the
`--stdin` input, but ultimately derived from real filenames in whatever
repository was indexed. POSIX filesystems permit any byte except `NUL`
and `/` in a filename, including embedded newlines. A file whose path
contains an embedded `\n` (achievable on Linux, and something an
adversarial repository under CI could plant) would inject an extra,
attacker-controlled "line" into `--quiet`'s otherwise-trusted
one-path-per-line output, corrupting any downstream consumer that
naively does `for line in $(codegraph affected --quiet); do rm "$line"; done`
or similar — a real risk given the explicit CI-pipeline use case this flag
was built for (`git diff --name-only | codegraph affected --stdin --quiet`
piped onward).

**Fix:** Either reject/skip paths containing control characters before
emitting `--quiet` output, or use a NUL-delimited output mode (mirroring
`git diff -z`/`xargs -0`) for safe consumption:

```go
if quiet {
	for _, l := range result.AffectedTests {
		if strings.ContainsAny(l.FilePath, "\n\r") {
			continue // or return an error — a graph-indexed path should never contain these
		}
		...
	}
}
```

### WR-04: `Affected`'s reverse BFS omits the RES-02 interface-dispatch composition that `Callers`/`Impact` both apply

**File:** `internal/query/traverse.go:491-558` vs `323-378` (`Callers`) and `395-465` (`Impact`)

**Issue:** `Callers` and `Impact` both explicitly compose
`dispatchSiblingIDs`/`BuildImplementsIndex`/`buildContainsIndex` so that a
call dispatched dynamically through a shared interface is included in
their blast-radius/caller results (RES-02, documented at length in both
functions' doc comments). `Affected` performs the same
shape of reverse-adjacency BFS (`BuildReverseAdjacency` + frontier
expansion) but never builds or consults the implements/contains indices at
all — a test that depends on a method only reachable through dynamic
dispatch from a sibling implementation of an interface will be silently
missing from `affected`'s results, understating the actual test blast
radius for exactly the polymorphic code shapes RES-02 was built to handle
elsewhere in this same file. Nothing in `Affected`'s doc comment
(`traverse.go:478-490`) documents this as an intentional simplification —
it reads as an oversight given how deliberately RES-02 is called out on
its two siblings.

**Fix:** Compose the same `dispatchSiblingIDs` expansion into `Affected`'s
frontier loop that `Impact` already uses (lines 427-431), so a changed
file's dispatch-reachable dependents are not silently excluded from
`affectedTests`.

### WR-05: Traversal results (`Impact.Affected`, `Callers.Callers`, `Affected.AffectedTests`) are never independently sorted before being returned

**File:** `internal/query/traverse.go` (all three), contrast with `internal/query/files.go:226-242` (`sortFileTree`)

**Issue:** `files.go`'s `sortFileTree` explicitly sorts its output "so the
tree projection is deterministic across calls (matching this package's
general '--json output must be reproducible' convention, D-06)". None of
`Impact`, `Callers`, or `Affected` apply an equivalent final sort to their
result slices — each result's order is entirely a function of the
underlying `graphstore.Reader`'s `IterateNodes`/`IterateEdges` enumeration
order (via `BuildReverseAdjacency`'s scan-order-preserving append) plus
Go's frontier-processing order. This is very likely stable in practice if
the Pebble-backed reader iterates in sorted-key order, but that guarantee
lives entirely outside this package (not verified in any file in this
review's scope) and is not asserted by any test here. Per the reviewer
brief's explicit ask to check for "output-ordering non-determinism": this
is a real, if currently probably-benign, gap relative to the project's own
stated determinism convention.

**Fix:** Add an explicit final sort (e.g. by `FilePath`, then `Name`) to
`ImpactResult.Affected`, `CallersResult.Callers`, and
`AffectedResult.AffectedTests` before returning, so determinism does not
depend on an unverified upstream iteration-order contract.

### WR-06: `bufio.Scanner`'s default 64KB per-line limit silently truncates the *entire remaining* stdin stream with no error surfaced

**File:** `internal/cli/affected.go:166-176`

**Issue:** `collectAffectedFiles` uses `bufio.NewScanner(cmd.InOrStdin())`
with its default token-size ceiling (`bufio.MaxScanTokenSize`, 64KB). If a
single line exceeds that (a pathological or maliciously long single
"path" on `--stdin`), `Scan()` returns `false` with `Err() ==
bufio.ErrTooLong`, and the loop simply stops — but the comment at lines
171-176 documents this as deliberately swallowed ("a git-hook pipeline
should degrade to 'no files from stdin' rather than fail the whole
command"). The actual effect is stronger than that comment implies: it is
not "no files from stdin" but "only the files read *before* the oversized
line," with everything after silently dropped and no indication to the
caller that truncation occurred.

**Fix:** At minimum, call `scanner.Buffer(...)` with a bounded-but-larger
max token size appropriate for a file path (e.g. 4096 bytes, the common
`PATH_MAX`), so a legitimately long-but-valid path is not treated as
"too long," and consider surfacing (rather than silently swallowing) the
`bufio.ErrTooLong` case specifically, since it indicates malformed input
rather than ordinary EOF.

### WR-07: `isTestSymbol`'s naming heuristic can misclassify production code as a test symbol

**File:** `internal/query/traverse.go:471-476`

**Issue:** `isTestSymbol` treats any node whose `Name` starts with `Test`
or `Benchmark` as a test symbol, regardless of file location or function
signature. A production helper function named e.g. `TestConnectionPool`,
`TestModeEnabled`, or `BenchmarkSuiteResults` (none of which are Go test
functions) would be misclassified as an "affected test" by both `Affected`
and — via the same helper — nowhere else, but specifically inflates
`affected`'s output with false positives. This is a known
heuristic-not-ground-truth tradeoff (D-07 documents it as query-time
derivation, not a persisted test-coverage edge), so this is recorded as a
lower-confidence quality note rather than a hard defect, but worth
tightening given `affected`'s primary use case is automated CI/git-hook
consumption where false positives directly waste CI time.

**Fix (optional):** Tighten the heuristic to also require the node's
`Kind` be `function` and either its file end in `_test.go` (already
checked) or, for the name-based fallback, only apply it when
`_test.go`-ness is unknown/unavailable — or drop the name-based fallback
entirely, since the `_test.go`-suffix check alone already covers Go's own
test-function-naming convention (a `TestXxx` function not in a `_test.go`
file is not runnable by `go test` in the first place).

## Info

### IN-01: `downloadReleaseAsset` has no `context.Context` for early cancellation

**File:** `internal/upgrade/upgrade.go:221-224`

**Issue:** `downloadHTTPClient.Get(url)` is called with a `//nolint:gosec,noctx` suppression; the `noctx` half of that suppression waives go-vet's request that HTTP calls carry a cancelable context. The package-level `downloadHTTPClient`'s 5-minute `Timeout` does bound the call, but a user hitting Ctrl+C mid-download (or a future caller wanting to cancel on `SIGINT`) has no way to short-circuit before that ceiling.

**Fix:** Thread a `context.Context` through `Run`/`Options`/`downloadReleaseAsset` and use `http.NewRequestWithContext`, so `codegraph upgrade` can be interrupted immediately rather than only after up to 5 minutes.

---

_Reviewed: 2026-07-19_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
