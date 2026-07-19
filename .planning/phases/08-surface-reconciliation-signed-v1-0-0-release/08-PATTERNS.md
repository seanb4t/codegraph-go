# Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Pattern Map

**Mapped:** 2026-07-19
**Files analyzed:** 14 (edits) + 3 new docs + 1 planning edit + 0 new source files
**Analogs found:** 14 / 14 (all edits are in-place; every "analog" is the file's own current state / a sibling command)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/query/validate.go` (defaultDepth 5→2; NEW defaultAffectedDepth=5) | service/config constant | transform (BFS bound) | itself (`clampDepth`/`defaultDepth`, lines 31-55) | exact — edit in place |
| `internal/query/traverse.go` (`Affected` → depth-bounded BFS) | service | event-driven/transform (graph BFS) | `Impact` in the same file (lines 395-465) — same package, same BFS shape needed | exact — same-file sibling method |
| `internal/cli/impact.go` (`-d`/`-j` shorts) | controller (cobra command) | request-response | itself (current `Flags()` block, lines 64-66) | exact — edit in place |
| `internal/cli/files.go` (`--dir` new flag, `-j` short) | controller (cobra command) | request-response | itself; `FilesOptions` struct call site (lines 43-48, 83-88) | exact — edit in place |
| `internal/cli/affected.go` (`--stdin`/`--depth`/`--filter`/`--quiet`, relax `Args`) | controller (cobra command) | request-response + stdin/file-I/O | `impact.go`'s depth-flag idiom (IntVar) + `index.go`'s quiet/verbose BoolVarP idiom | role-match — composite of two analogs |
| every other `internal/cli/*.go` (short-flag adds, SURF-03) | controller (cobra command) | request-response | `internal/cli/index.go` lines 82-84 (`-f`/`-q`/`-v` `BoolVarP` idiom); `internal/cli/node.go` lines 69-71 (`-p`/`-f`/`-l` mixed StringVarP/IntVarP idiom) | exact — established repo-wide idiom |
| `docs/FLAG-PARITY.md` (NEW) | docs/config | batch (static matrix) | `docs/RELEASE.md` (structure: numbered sections, verified-command style, status notes) | role-match |
| `docs/RELEASE-PROCEDURES.md` (NEW) | docs/config | batch | `docs/RELEASE.md` (maintainer-facing sibling; same verbatim-command style) | exact structural sibling |
| `.planning/PROJECT.md` (caveat retirement, 4 sites) | docs/config | transform (text edit) | itself — 4 located line ranges | exact — edit in place |
| `.goreleaser.yaml`, `.github/workflows/release.yml`/`ci.yml`/`bench.yml` | config (CI/CD) | batch | REUSE ONLY — no edits unless a lockstep change to `verify.go` constants is required | n/a — do not rewrite |
| `internal/upgrade/verify.go` | service (verifier) | request-response (verify) | itself — REUSE, only touch in lockstep with a `release.yml` rename (not expected this phase) | n/a — reference only |
| Optional: `internal/cli/archtest/flag_parity_test.go` (NEW, discretionary) | test | batch (tree-walk) | `internal/cli/present/archtest/import_graph_test.go` (package-graph walk via `golang.org/x/tools/go/packages`) — same "walk + assert" shape but over `cmd.Commands()`/`Flags().VisitAll` instead of import graph | role-match |

## Pattern Assignments

### `internal/query/validate.go` (SURF-01 depth default)

**Analog:** itself — `defaultDepth`/`clampDepth` (lines 17-55)

**Current shared-engine constant + clamp** (lines 17-55, verbatim):
```go
const (
	// MaxDepth bounds impact/affected BFS depth. 50 comfortably exceeds
	// any realistic call-chain depth in a real codebase...
	MaxDepth = 50
	...
	// defaultDepth is applied when a caller passes a non-positive depth
	// (clampDepth's "0 means default" convention, matching the CLI flags'
	// zero-value default).
	defaultDepth = 5
	...
)

// clampDepth returns min(n, MaxDepth), treating n<=0 as defaultDepth
// rather than "unbounded" or "zero traversal" — a caller that omits
// --depth gets a small, useful default instead of an error.
func clampDepth(n int) int {
	if n <= 0 {
		n = defaultDepth
	}
	if n > MaxDepth {
		return MaxDepth
	}
	return n
}
```

**SURF-01 change:** `defaultDepth = 5` → `defaultDepth = 2` (D-02). Verified sole non-test consumer via `rg -n "clampDepth\(" internal/query/*.go` → only `traverse.go:399` inside `Impact()` (re-verify this grep at execution time per RESEARCH Pitfall 1).

**SURF-04 companion constant (new, do NOT reuse `defaultDepth`):** add a sibling constant next to `defaultDepth`, following the exact same naming/doc-comment shape:
```go
// defaultAffectedDepth is Affected's own "0 means default" value —
// deliberately NOT defaultDepth (impact=2, affected=5 per TS 1.3.1;
// RESEARCH "Anti-Patterns"). Shares MaxDepth as its ceiling.
const defaultAffectedDepth = 5
```
Add a parallel `clampAffectedDepth(n int) int` (or a `clampDepthWithDefault(n, def int) int` helper reused by both `clampDepth` and the new call site) — same body shape as `clampDepth`, swapping in `defaultAffectedDepth`.

---

### `internal/query/traverse.go` (SURF-04 `Affected` BFS extension)

**Analog:** `Impact` in the same file (lines 395-465) — the BFS frontier/next-frontier shape to crib from.

**Core BFS pattern to replicate** (lines 424-456, verbatim shape):
```go
for d := 0; d < depth && len(frontier) > 0; d++ {
	var next []*schema.Node
	for _, n := range frontier {
		...
		for _, tid := range targetIDs {
			for _, edge := range rev[tid] {
				edgeCount++
				if visited[edge.Source] {
					continue
				}
				visited[edge.Source] = true
				srcNode, err := e.reader.GetNode(edge.Source)
				if err != nil {
					if errors.Is(err, graphstore.ErrNotFound) {
						continue // WR-04: dangling edge, skip not abort
					}
					return ImpactResult{}, err
				}
				affected = append(affected, nodeLocation(srcNode))
				next = append(next, srcNode)
			}
		}
	}
	frontier = next
}
```

**Current `Affected` (single-hop, lines 478-509+) — the code being replaced:**
```go
// Affected derives impacted test files/symbols for a set of changed
// files at query time (D-07): no persisted test-coverage edge kind —
// walk the D-04 reverse-adjacency map from every symbol defined in the
// changed files, keeping reverse-caller targets that pass the
// isTestSymbol heuristic.
func (e *Engine) Affected(files []string) (AffectedResult, error) {
	rev, err := BuildReverseAdjacency(e.reader)
	...
	// (currently ONE hop: `for _, edge := range rev[id]`, no depth loop)
}
```

**Required shape:** port `Impact`'s frontier loop, bounded by `clampDepth`-equivalent using `defaultAffectedDepth`, PLUS TS's leaf-pruning rule (RESEARCH Pitfall 2): a node passing `isTestSymbol` (lines 467-476, unchanged — keep Go's Go-specific heuristic, do not port TS's JS path-suffix heuristic per the documented divergence) is added to the result set and NOT expanded further; only non-test dependents get queued into `next`.

**`isTestSymbol` heuristic to keep as-is** (lines 467-476, verbatim, do not modify):
```go
func isTestSymbol(n *schema.Node) bool {
	if strings.HasSuffix(n.FilePath, "_test.go") {
		return true
	}
	return strings.HasPrefix(n.Name, "Test") || strings.HasPrefix(n.Name, "Benchmark")
}
```

**Error handling pattern:** identical to `Impact`'s dangling-edge skip (`errors.Is(err, graphstore.ErrNotFound)` → `continue`, not abort) — reuse verbatim.

---

### `internal/cli/impact.go` (SURF-01/03: short flags, no behavior change)

**Analog:** itself, current flags block (lines 64-66):
```go
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().IntVar(&depth, "depth", 0, "BFS depth (default 5, max 50)")
cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON output")
```
**SURF-03 change:** `IntVar` → `IntVarP(&depth, "depth", "d", 0, "BFS depth (default 2, max 50)")` (update the help text's stale "default 5" too, per D-02); `BoolVar` → `BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")`. Both `-d`/`-j` are free in this command per RESEARCH's per-command letter map.

---

### `internal/cli/files.go` (SURF-02 `--dir`, SURF-03 `-j`)

**Analog:** itself — current options wiring (lines 43-48) + flags block (lines 83-88):
```go
result, err := eng.Files(query.FilesOptions{
	Pattern: pattern,
	Filter:  filter,
	Depth:   depth,
	Format:  format,
})
...
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().StringVar(&pattern, "pattern", "", "shell glob narrowing the result set")
cmd.Flags().StringVar(&filter, "filter", "", "restrict to one language")
cmd.Flags().IntVar(&depth, "depth", 0, "directory-nesting cap (0 = unlimited)")
cmd.Flags().StringVar(&format, "format", "", `"flat" (default) or "tree"`)
cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON output")
```

**Add sibling `--dir` flag + `FilesOptions.Dir` field**, following the exact same `StringVar` shape as `--filter`/`--pattern`:
```go
cmd.Flags().StringVar(&dir, "dir", "", "directory-path prefix filter (matches TS --filter <dir>)")
```
and wire `Dir: dir` into the `query.FilesOptions{}` literal alongside `Pattern`/`Filter`/`Depth`/`Format`.

**Exact TS matching semantics to replicate in `internal/query`'s `Files` implementation** (from RESEARCH, `bin/codegraph.js:1348-1354`, verified this session):
```javascript
if (options.filter) {
    const filter = options.filter;
    files = files.filter(f => f.path.startsWith(filter) || f.path.startsWith('./' + filter));
}
```
Go equivalent: `strings.HasPrefix(path, dir) || strings.HasPrefix(path, "./"+dir)` — a plain prefix check, NOT a glob (do not reach for `doublestar` or any pattern library; this is a deliberate "Don't Hand-Roll" item per RESEARCH).

Add `-j` short to the existing `--json` `BoolVar` (free per RESEARCH table).

---

### `internal/cli/affected.go` (SURF-04: `--stdin`/`--depth`/`--filter`/`--quiet`)

**Analog A (depth flag idiom):** `impact.go` lines 64-66 (`IntVarP` with `-d` short, once SURF-01/03 land on that file).
**Analog B (quiet/verbose BoolVarP idiom):** `internal/cli/index.go` lines 82-84:
```go
cmd.Flags().BoolVarP(&force, "force", "f", false, "rebuild without prompting for confirmation")
cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress and summary output")
cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "emit per-file/per-pass detail")
```

**Current command shape being extended (whole file, lines 1-70)** — the `Args: cobra.MinimumNArgs(1)` line (25) is the blocking gotcha (RESEARCH Pitfall 3):
```go
cmd := &cobra.Command{
	Use:   "affected <files...>",
	Short: "List test symbols impacted by changes to the given files",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		...
		result, err := eng.Affected(args)
		...
	},
}
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON output")
```

**Required edits:**
1. `Args: cobra.MinimumNArgs(1)` → `cobra.ArbitraryArgs` (or a custom validator requiring `len(args) > 0 || stdinFlag`); handle the zero-input case inside `RunE` (print advisory, exit 0 unless `--quiet`).
2. Add flags following the analogs above:
   ```go
   cmd.Flags().BoolVar(&stdin, "stdin", false, "read changed file paths from stdin (one per line)")
   cmd.Flags().IntVarP(&depth, "depth", "d", 0, "BFS depth (default 5, max 50)")
   cmd.Flags().StringVarP(&filter, "filter", "f", "", "glob narrowing affected paths")
   cmd.Flags().BoolVarP(&quiet, "quiet", "q", true /*wired to established -q*/, "suppress the human summary; machine-readable path list only")
   cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")
   ```
   (verify default value/semantics against RESEARCH's exact TS default table before finalizing per-flag defaults — `-d`/`-f`/`-j`/`-q` are all unverified-free-in-this-command but were confirmed free in RESEARCH's SURF-03 table.)
3. Call `eng.Affected(files, depth)` — updated signature per the `traverse.go` BFS extension above.
4. Reading stdin: no existing analog in this codebase (`affected` is the first command needing stdin input) — implement directly with `bufio.NewScanner(cmd.InOrStdin())`, one path per line, trimming whitespace/blank lines — this is genuinely new I/O, not a copy from elsewhere in `internal/cli`.

**Rendering-seam constraint (Phase 6, MUST honor):** `--stdin --quiet` output must stay plain (no ANSI) — do NOT route this path through `present.RenderFiles`/`present.ChoosePresentation` the way `files.go` does; keep it as a raw `fmt.Fprintf` loop over paths, matching the archtest guarantee in `internal/cli/present/archtest/import_graph_test.go` (below) that scripting output stays outside the styled-rendering path.

---

### SURF-03 short-flag idiom (all `internal/cli/*.go` files needing new shorts)

**Analog 1 — BoolVarP with `-f`/`-q`/`-v`:** `internal/cli/index.go` lines 82-84 (reproduced above).

**Analog 2 — mixed StringVarP/IntVarP in one command:** `internal/cli/node.go` lines 69-71:
```go
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().StringVarP(&file, "file", "f", "", "file path — disambiguates symbol, or selects file-mode when symbol is omitted")
cmd.Flags().IntVarP(&line, "line", "l", 0, "line number — narrows an overloaded symbol to the definition containing (or nearest) this line (NODE-03)")
```

**Per-command target letters (from RESEARCH, verified against live TS 1.3.1 `--help`):**
- `status.go`: `--json` → add `-j` (only `-p` used today)
- `query.go`: `--limit`→`-l`, `--kind`→`-k`, `--json`→`-j` (all free)
- `callers.go`/`callees.go`: `--limit`→`-l`, `--json`→`-j` (both free)
- `impact.go`: `--depth`→`-d`, `--json`→`-j` (both free)
- `install.go`/`uninstall.go`: `--target`→`-t`, `--location`→`-l` (both free; `-y` already bound)
- `upgrade.go`: ADD a whole new `--force`/`-f` flag (currently absent, not just missing a short — RESEARCH Pitfall: this is a name-and-behavior addition, not a short-alias add)

**Pattern to apply uniformly:** change `*Var(` to `*VarP(` and insert the short-letter string arg, following Analog 1/2's argument order (`&var, name, short, default, help`) exactly — no other change to each command's body.

---

### `docs/FLAG-PARITY.md` (SURF-05, NEW)

**Analog:** `docs/RELEASE.md` structure (lines 1-60 read) — numbered `##` sections, a "Status note" callout block, verbatim shell/verification commands inline, cross-referenced to specific source files by path. Reuse this exact prose style: a per-command `##` section (or one big table, per D-06) with columns `TS flag+default | Go flag+default | status`. Record, verbatim, the divergences RESEARCH's SURF-03 table enumerates (dual `--filter`/`--dir`, `search`/`migrate` as Go-only/accepted, `install --auto-allow` behavioral divergence, `node` missing file-mode, `files --format` default mismatch, `upgrade --force` addition).

---

### `docs/RELEASE-PROCEDURES.md` (REL-02 folded runbook, NEW)

**Analog:** `docs/RELEASE.md` (full file — this is the *user*-facing verification sibling; the new doc is the *maintainer*-facing runbook). Reuse:
- The "Status note" blockquote style (lines 10-15) for any not-yet-true state.
- The verbatim numbered-section + fenced-shell-command style (lines 17-45) for: pre-tag 6-target `go list -mod=readonly` sweep (D-09), tag conventions (`v0.0.0-rc.N` prerelease / `vX.Y.Z` stable / `milestone-v*` never fires), the LOCKED `verify.go` constants (below), post-release `cosign verify-blob`/`slsa-verifier verify-artifact` commands (copy RESEARCH's Pattern 3 verbatim), rollback/cleanup of a failed rc tag, and the `-c commit.gpgsign=false` pipeline-only caveat.

**LOCKED constants to cite verbatim** (`internal/upgrade/verify.go` lines 42-44, confirmed via Read):
```go
releaseOIDCIssuer         = "https://token.actions.githubusercontent.com"
releaseRepoSlug           = "seanb4t/codegraph-go"
releaseWorkflowRefPattern = `^https://github\.com/` + releaseRepoSlug + `/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`
```

---

### `.planning/PROJECT.md` (REL-04 caveat retirement)

**Analog:** itself — 4 confirmed sites (verified via `rg -n "not yet drop-in|drop-in parity"` this session):
1. Line 13 (milestone goal line): `...then cut the first real signed \`v1.0.0\` release — retiring the "not yet drop-in" caveat v0.1 shipped with.`
2. Line 72 (repo-state paragraph): `**Not yet drop-in parity** — the CLI command surface diverges from TS CodeGraph; closing that gap is the remaining bar for a 1.0.`
3. Line 88 (Key Decisions table row): `⚠️ Partial — v0.1 shipped the core capabilities + a signed release, but the CLI command surface diverges from TS, so it is NOT yet a drop-in parity replacement...`
4. Line 95 (related decision-log row, no literal caveat phrase but same narrative): `— In progress: behavioral parity (\`explore\`/\`node\`), watcher-on-MCP default, git/worktree awareness, output hygiene, Charm TUI, then signed v1.0.0`

**Edit pattern:** each site needs individual, context-appropriate rewording to reflect "parity closed, v1.0.0 shipped" rather than a single find/replace — re-run `rg -n "not yet drop-in" .planning/PROJECT.md` after editing; it must return zero hits (RESEARCH Pitfall 6 warning sign).

---

### Optional SURF-05 drift test (Claude's discretion)

**Analog:** `internal/cli/present/archtest/import_graph_test.go` (full pattern read, lines 1-50) — package-level "walk + assert" shape using `golang.org/x/tools/go/packages`. For a flag-parity drift test, the equivalent walk is over `newRootCmd()`'s `cmd.Commands()` tree + each command's `Flags().VisitAll(...)`, asserting every registered flag name appears somewhere in `docs/FLAG-PARITY.md`'s text (a simple string-containment check against the doc's rendered bytes, not an AST-level import walk). Reuse the archtest package's "fails closed until the guard is real" build-order philosophy (comment lines 15-19) if this test lands before all SURF edits are in place — same self-defeat-guard discipline.

## Shared Patterns

### cobra short-flag registration
**Source:** `internal/cli/index.go:82-84`, `internal/cli/node.go:69-71`
**Apply to:** every `internal/cli/*.go` file listed in SURF-03's per-command table.
```go
cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress and summary output")
cmd.Flags().StringVarP(&file, "file", "f", "", "file path — disambiguates symbol...")
cmd.Flags().IntVarP(&line, "line", "l", 0, "line number — narrows...")
```

### Shared-engine default (CLI==MCP), single point of change
**Source:** `internal/query/validate.go:31-55`
**Apply to:** `internal/query` only — never duplicate a depth/limit default inside `internal/cli` or `internal/mcp`; both surfaces call the same `Engine` method and inherit the change automatically.

### Compact worktree notice + `--json` early-return ordering
**Source:** every `internal/cli/*.go` `RunE` (e.g. `impact.go:42-54`, `affected.go:43-57`) — verbatim comment: "Compact worktree notice (WORK-02, D-12): lives strictly inside the human-output branch, AFTER the --json early return above."
**Apply to:** `affected.go`'s new `--quiet`/`--stdin` paths — the notice must stay inside the non-JSON, non-quiet branch; do not print it when `--quiet` is set.

### Error handling — dangling edge skip, not abort
**Source:** `internal/query/traverse.go:440-449` (`Impact`'s `errors.Is(err, graphstore.ErrNotFound) { continue }`)
**Apply to:** the new `Affected` BFS extension — must reuse this exact skip-not-abort behavior at every node dereference inside the new frontier loop.

### Docs prose/structure style
**Source:** `docs/RELEASE.md` (whole file — numbered sections, "Status note" blockquote, verbatim shell commands, path-referenced claims)
**Apply to:** `docs/FLAG-PARITY.md`, `docs/RELEASE-PROCEDURES.md`.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| stdin-reading in `affected.go` (`--stdin`) | controller | file-I/O (stdin stream) | No existing `internal/cli` command reads from stdin today — first stdin consumer in the CLI surface; implement directly with `bufio.NewScanner(cmd.InOrStdin())`, no in-repo analog to copy. |
| `go list -deps -json` CGo-closure audit script (REL-01) | tooling/CI script | batch | Not a source file per se — a runnable shell command (already verified working in RESEARCH §Package Legitimacy Audit); no code file to write, only a CI step or doc reference. |

## Metadata

**Analog search scope:** `internal/cli/`, `internal/query/`, `docs/`, `.planning/PROJECT.md`, `internal/upgrade/verify.go`, `internal/cli/present/archtest/`
**Files scanned:** 14 `internal/cli/*.go` command files (full read or grep), `internal/query/validate.go` (full), `internal/query/traverse.go` (lines 380-510), `internal/cli/root.go` (full), `docs/RELEASE.md` (lines 1-60), `internal/upgrade/verify.go` (grep), `internal/cli/present/archtest/import_graph_test.go` (lines 1-50), `.planning/PROJECT.md` (grep)
**Pattern extraction date:** 2026-07-19
