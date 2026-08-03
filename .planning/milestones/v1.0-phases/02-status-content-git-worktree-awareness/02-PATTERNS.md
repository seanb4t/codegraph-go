# Phase 2: status Content & Git/Worktree Awareness - Pattern Map

**Mapped:** 2026-07-15
**Files analyzed:** 12 (2 new package files + 1 fixture helper + 9 modified files)
**Analogs found:** 10 / 12 (2 explicit gaps — `internal/gitmeta` core logic, real-git fixture builder — nearest partial precedents named below)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/gitmeta/worktree.go` (new) | utility (bounded subprocess) | request-response (best-effort) | **no analog** — nearest partial: `testdata/golden/golden_parity_test.go`'s `gitHead` (git subprocess, no timeout) | no-analog |
| `internal/gitmeta/detect.go` (new) | service (pure algorithm) | transform | **no analog** — nearest partial: `internal/query/status.go`'s `computeStale` (best-effort filesystem probe returning bool, tolerant of missing state) | partial |
| `internal/gitmeta/cache.go` (new, `CachingDetector`) | utility (mutex-guarded cache) | request-response | `internal/graphstore/pebble_store.go`'s `sync.RWMutex`-guarded state (per RESEARCH's Standard Stack row) | role-match |
| `internal/gitmeta/*_test.go` (new, six-layout fixture builder) | test (fixture builder) | file-I/O + event-driven (git subprocess) | **no analog** — nearest partial: `testdata/golden/golden_parity_test.go`'s `resolveColbymchenryCorpus` (shells out to `git clone`, `t.Skip` on failure, but does not *build* repo topology — only clones one) | partial |
| `internal/query/status.go` (modified) | service (aggregation) | CRUD (read scan) | itself (extend existing `Status()`/`StatusResult`) | exact |
| `internal/query/engine.go` (modified, `OpenAt`/`startPath`) | service (constructor/seam) | request-response | itself (extend existing `OpenAt`) | exact |
| `internal/query/render_markdown.go` or new `render_status.go`/`render_results.go` (modified/new, `Render*` + `worktreeNotice`) | component (renderer) | transform | `RenderExplore`/`RenderNode` + `staleBanner`/`staleBannerText` (same file) | exact |
| `internal/mcp/tools.go` (modified, 6 call sites) | controller (MCP handler) | request-response | itself (`companionHandler` switch, `exploreHandler`'s "no re-rendering" precedent) | exact |
| `internal/mcp/server.go` (modified, server-scoped cache) | provider (server construction) | request-response | itself — `BuildServer`-scope construction is the only server-lifetime scope in this codebase | exact |
| `internal/cli/status.go` (modified, sectioned layout) | controller (CLI command) | request-response | `internal/cli/files.go`'s `newFilesCmd` (same `--json`/human-branch split, same `writeJSONLine` convention) | exact |
| `internal/query/files_status_test.go` (modified) | test | CRUD assertions | itself (existing placeholder-assertion subtests) | exact |
| `testdata/golden/golden_parity_test.go` (modified) | test | batch/parity | itself (existing status-key assertions ~line 635) | exact |

## Pattern Assignments

### `internal/gitmeta/worktree.go` (new) — `gitWorktreeRoot`/`gitCommonDir`/`realpath`

**No true analog exists.** `internal/gitmeta` is the first package in this codebase to shell out to `git` as *production* logic (not test-fixture setup). The closest partial precedent is a **test helper**, `gitHead` in `testdata/golden/golden_parity_test.go:128-137` — useful for the subprocess-invocation shape, but it has no timeout/context and is test-only:

```go
// testdata/golden/golden_parity_test.go:128-137 — subprocess shape to imitate,
// NOT the bounded/best-effort contract (that must be added per D-03)
func gitHead(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
```

RESEARCH's Pattern 1 (§ Architecture Patterns) is the actual specification to implement — copy it verbatim, it is already Go code ported from `sync/worktree.js`:

```go
// RESEARCH.md Pattern 1 — the actual target implementation (D-03)
func gitWorktreeRoot(ctx context.Context, dir string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output() // stderr discarded
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return ""
	}
	return realpath(trimmed)
}

func realpath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}
```

**Best-effort degradation precedent** (the *shape* of "any error ⇒ safe zero value, never propagate"), copy from `internal/query/status.go`'s `computeStale` (lines 135-157): note how it treats `os.Stat` `ErrNotExist` as a normal branch, not an error, and how `e.repoRoot == ""` degrades to `false, nil` rather than erroring — the same degrade-on-missing-context shape D-14 requires for `Engine`s built via `New`/`NewWithRoot`.

### `internal/gitmeta/detect.go` (new) — `DetectIndexMismatch` 4-gate cascade

**No analog** — this is a novel pure-algorithm function. Nearest precedent for "cascading early-return gates over best-effort probes" is `computeStale`'s two-branch check, but the shape is thin enough that RESEARCH's verbatim JS-to-Go transcription (§ Code Examples) is the actual source to copy:

```go
// Ported verbatim from sync/worktree.js's detectWorktreeIndexMismatch (D-02)
func DetectIndexMismatch(ctx context.Context, startPath, indexRoot string) *Mismatch {
	worktreeRoot := gitWorktreeRoot(ctx, startPath)
	if worktreeRoot == "" {
		return nil // gate 1
	}
	resolvedIndexRoot := realpath(indexRoot)
	if worktreeRoot == resolvedIndexRoot {
		return nil // gate 2
	}
	if gitWorktreeRoot(ctx, resolvedIndexRoot) != resolvedIndexRoot {
		return nil // gate 3
	}
	worktreeCommon := gitCommonDir(ctx, worktreeRoot)
	indexCommon := gitCommonDir(ctx, resolvedIndexRoot)
	if worktreeCommon != "" && indexCommon != "" && worktreeCommon != indexCommon {
		return nil // gate 4 — differing common dirs SUPPRESS (submodule/embedded clone)
	}
	return &Mismatch{WorktreeRoot: worktreeRoot, IndexRoot: resolvedIndexRoot}
}
```

### `internal/gitmeta/cache.go` (new) — `CachingDetector`

**Analog:** `internal/graphstore/pebble_store.go`'s mutex-guarded state convention (RESEARCH's Standard Stack row explicitly names this as the precedent for "no new dependency; matches the existing `pebbleStore`'s `sync.RWMutex` convention"). Read that file's lock pattern directly when implementing; the shape needed here is simpler (a plain `map[string]*Mismatch` + `sync.Mutex`, keyed on `startPath+"\x00"+indexRoot` per D-13/TS's own cache key), with **negative results cached too** — store a sentinel or use a second `map[string]bool` "checked" set, since a bare `nil` map value cannot distinguish "not yet checked" from "checked, no mismatch."

### `internal/gitmeta` fixture builder (new `_test.go`, TEST-02 six layouts)

**No repo-building test helper exists anywhere in this codebase.** Nearest partial precedent, `resolveColbymchenryCorpus` (`testdata/golden/golden_parity_test.go:221-231`):

```go
// testdata/golden/golden_parity_test.go:221-231 — the shell-out-in-test +
// t.Skip-on-failure shape to imitate; it CLONES, it does not BUILD topology
// (no `git worktree add`/`git submodule add`/nested-clone construction exists
// anywhere in this codebase today)
func resolveColbymchenryCorpus(t *testing.T) string {
	t.Helper()
	tmpRoot := t.TempDir()
	cmd := exec.Command("git", "clone", "--depth", "1", "--quiet",
		"https://github.com/colbymchenry/codegraph.git", tmpRoot)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("... unavailable (git clone failed, likely no network access): %v: %s", err, string(out))
	}
	return tmpRoot
}
```

Reuse: `t.TempDir()` scaffolding, `t.Skip` (not `t.Fatal`) on missing/failing git, `t.Helper()` marking. **New work required (no precedent):** `git init`, `git worktree add`, `git submodule add`, deterministic `GIT_*` env vars for hermetic commits (`-c init.defaultBranch=main`, `GIT_AUTHOR_NAME`/`GIT_AUTHOR_EMAIL`/`GIT_COMMITTER_*` or equivalent `-c user.name`/`-c user.email` flags) — none of this exists in the codebase; D-15 is asking for genuinely new test infrastructure. Skip-on-absent-git precedent: grep the repo for `exec.LookPath("git")`-style guards — none exist yet either; `resolveColbymchenryCorpus` skips only on clone failure (implicitly covers missing git via the exec error), which is an acceptable pattern to replicate (`t.Skip` on any `exec.Command("git", ...).Run()` error).

---

### `internal/query/status.go` (modified) — `StatusResult` + `Status()` + doc-comment decision table

**Analog:** itself. The file's own doc comment (lines 18-41) is **the project convention for recording every TS→Go key remap** — new keys (`dbSizeBytes`, `filesByLanguage`, live `worktreeMismatch`) MUST be added as new rows to that same table, in the same `TS key | Go/Pebble rendering | Rationale` format. Copy the table-row style verbatim, e.g.:

```
//	dbSizeBytes                       | filepath.WalkDir sum over .codegraph/store/ | D-07 — Pebble has no single-file page-count analog to SQLite; directory byte sum is the honest Go-truthful reading
```

**Core scan pattern to extend** (add `FilesByLanguage` inside the *existing* `IterateFiles` loop, per RESEARCH Corrections #4 — do NOT add a second `IterateFiles()` call):

```go
// internal/query/status.go:174-177 — the existing loop to extend in place
var fileCount int64
for fileIt.Next() {
	fileCount++
	// ADD: fileLang := fileIt.File().Language; filesByLang[fileLang]++
}
```

**Best-effort directory walk to add** (`dbSizeBytes`, D-07) — mirror `newestSourceMtime`'s exact best-effort `WalkDir` shape (same file, lines 95-123: skip-on-error rather than abort, `fs.SkipDir` for excluded dirs is N/A here since D-07 wants everything under `store/`):

```go
// Pattern 2 (RESEARCH), mirroring internal/query/status.go's existing
// newestSourceMtime WalkDir shape (best-effort, skip unreadable entries)
func dbSizeBytes(storeDir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(storeDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort: skip, don't abort
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}
```

**Error handling:** identical to the rest of `Status()` — every scan step returns `(StatusResult{}, err)` on hard failure but treats `graphstore.ErrNotFound` (missing `Meta`) as a soft "not yet indexed" state, not an error (lines 209-212). Follow the same pattern for any new failure mode.

### `internal/query/engine.go` (modified) — `Engine.startPath` (D-14)

**Analog:** itself — extend the existing `Engine` struct (line 21-24) and `OpenAt` (line 54-72) in place. Add `startPath` alongside `reader`/`repoRoot`, resolved via `filepath.Abs` before storage (Pitfall 3 — mirror `ResolveCodegraphDir`'s own `filepath.Abs` call, not shown here but referenced in RESEARCH). `New`/`NewWithRoot` (lines 32-41) must leave `startPath` as its zero value (`""`), and the worktree-detection call site must treat `startPath == ""` as "no context, no mismatch" — the exact same degrade-safely shape `computeStale` already uses for `e.repoRoot == ""`.

### `internal/query/render_markdown.go` (modified/new) — `worktreeNotice` + `Render*` siblings

**Analog:** `staleBanner`/`staleBannerText` (lines 18-32) is **the exact precedent** for `worktreeNotice`:

```go
// internal/query/render_markdown.go:18-32 — the prepend-a-warning pattern
// to copy verbatim, substituting the D-11 notice/warning text and the D-16
// blockquote transform for MCP status
const staleBannerText = "**⚠ Index may be stale — a sync is pending.**\n\n"

func staleBanner(stale bool) string {
	if !stale {
		return ""
	}
	return staleBannerText
}
```

Build `worktreeNotice(mismatch *gitmeta.Mismatch) string` the same way: nil → `""`, non-nil → the verbatim D-11 compact notice text (glyph U+26A0 `⚠`, no U+FE0F — this file already gets that glyph right at line 22, confirm byte-for-byte match, do not copy Phase 1's `⚠️`). Apply via the same `b.WriteString(banner)` prepend seen at `RenderExplore` line 362 (`b.WriteString(staleBanner(stale))`).

**`Render*` sibling pattern** (SURF-06, D-16) — model on `RenderExplore`/`RenderNode`'s signature shape (plain string return, no ANSI, deterministic ordering preserved from the input slice — RESEARCH's Ordering Determinism table confirms every result is pre-sorted, renderers must NOT re-sort):

```go
// internal/query/render_markdown.go:127-136 — RenderNode is the smallest
// complete Render* example to imitate for header + body-lines shape
func RenderNode(n *schema.Node, calls, calledBy []*schema.Node) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** (%s)\n\n", n.Name, n.Kind)
	fmt.Fprintf(&b, "**Location:** %s:%d\n", n.FilePath, n.StartLine)
	// ...
	return b.String()
}
```

RESEARCH's own recommended markdown table shape (§ SURF-06) is the concrete target for `RenderCallers`/`RenderCallees`/`RenderImpact`/`RenderSearch` (shared `renderLocationTable([]Location) string` helper) and `RenderFiles` (own `FileEntry` table + `printFileTree`-derived tree branch — see below). `RenderStatus` (CLI form, D-09) and `RenderStatusMarkdown` (MCP bolded-bullet form, D-17) are two distinct new functions per RESEARCH Corrections #5 — do not conflate them.

**Naming caution (RESEARCH, D-16):** avoid bare `RenderStatus`/`RenderFiles` if Phase 6 is expected to introduce colorized variants under those names — consider `RenderStatusText`/`RenderFilesMarkdown` or similar suffixed names now.

**Tree-format renderer source:** `internal/cli/files.go:82-90`'s `printFileTree` is the exact indented-plain-text algorithm to port into `internal/query` (RESEARCH explicitly flags this as a "consider moving" candidate):

```go
// internal/cli/files.go:80-90 — port this into internal/query as the
// SURF-06 tree-format renderer (Claude's Discretion: move vs. reimplement)
func printFileTree(out io.Writer, nodes []*query.FileTreeNode, indent string) {
	for _, n := range nodes {
		if n.IsDir {
			fmt.Fprintf(out, "%s%s/\n", indent, n.Name)
			printFileTree(out, n.Children, indent+"  ")
		} else {
			fmt.Fprintf(out, "%s%s (%s)\n", indent, n.Name, n.Language)
		}
	}
}
```

**Plain-text-only constraint enforcement:** no archtest currently gates `charm.land`/`lipgloss` imports (confirmed — `internal/graphstore/archtest` and `internal/migrate/archtest` both exist but neither checks these; a repo-wide search for `charm.land|lipgloss` found zero hits anywhere). Follow `internal/graphstore/archtest/import_graph_test.go`'s pattern (`go/packages` import-graph load + prefix-check, NOT regex over source) if this phase or Phase 6 wants to add such a gate — it is the established mechanism for "no package outside X may import Y" in this codebase, but building it is **out of this phase's explicit scope** per D-09/TUI-02 boundary; note only.

### `internal/mcp/tools.go` (modified) — 6 call-site swaps + notice injection

**Analog:** itself — `companionHandler`'s `switch` (lines 172-348). Each of the 5 SURF-06 branches (`search` 200-229, `callers` 230-253, `callees` 254-277, `impact` 278-301, `files` 302-326) follows an identical shape: parse args → `openEngine` → call `Engine` method → marshal/render → `mcp.NewToolResultText`. **Change only the marshal step** (line 224 `json.Marshal(locs)` inline, 248/272/296/321 `query.Marshal*JSON`) to call the new `Render*` function instead — do not touch anything else in each branch:

```go
// internal/mcp/tools.go:244-252 — the exact call-site shape to swap;
// everything above "data, err :=" is untouched
result, err := eng.Callers(symbol, limit)
if err != nil {
	return mcp.NewToolResultError(err.Error()), nil
}
data, err := query.MarshalCallersJSON(result)   // ← becomes query.RenderCallers(result)
if err != nil {
	return mcp.NewToolResultError(err.Error()), nil
}
return mcp.NewToolResultText(string(data)), nil // ← becomes mcp.NewToolResultText(data) (no string() cast needed, Render* returns string)
```

`status` (lines 327-344) needs its **own** treatment per D-17: swap `query.MarshalStatusJSON` for a new `query.RenderStatusMarkdown`, then wrap the verbose warning as a blockquote before/inside that call (`> ⚠ ` prefix, `\n` → `\n> ` mid-string transform — no existing blockquote-transform precedent in this codebase; implement inline, it's a 2-3 line string replace).

**Notice injection for the other 6 tools:** apply `worktreeNotice(mismatch)` the same way `staleBanner` is applied in `RenderExplore` — as a `+` string prefix immediately before returning `mcp.NewToolResultText`, guarded by "no-op on `isError` results" (D-12) — i.e. only prefix on the success return paths, never on the `mcp.NewToolResultError` early returns already present in every branch.

### `internal/mcp/server.go` (modified) — server-scoped `CachingDetector`

**No existing analog for server-construction-scoped state** — `BuildServer` (wherever it constructs tool handlers) is the only place in this codebase with server-lifetime scope; there is no existing example of closing a cache over multiple handlers here. Construct one `gitmeta.CachingDetector` in `BuildServer`, close it over every registered handler exactly as `defaultPath` is presumably already closed over per `exploreHandler(defaultPath)`/`companionHandler(name, defaultPath)`'s existing closure-parameter convention (`internal/mcp/tools.go:81`, `:172`) — add the detector as a second closed-over parameter following the same style: `exploreHandler(defaultPath, detector)`.

### `internal/cli/status.go` (modified) — sectioned layout (D-09)

**Analog:** `internal/cli/files.go`'s `newFilesCmd` (`--json` branch at lines 44-50, human branch below) for the `if jsonOut { ...; return }` / human-output split — `internal/cli/status.go` already has this exact shape (lines 40-52), just replace the terse `fmt.Fprintf` human-branch body with the D-09 sectioned layout, calling the new `query.RenderStatus(result)` (or building the sections inline — Claude's Discretion per CONTEXT). Keep the `--json` branch (`query.MarshalStatusJSON`, lines 41-46) **completely untouched**.

```go
// internal/cli/status.go:40-52 — the exact structure to preserve;
// only the human-output body (lines 48-52) changes
if jsonOut {
	data, err := query.MarshalStatusJSON(result)
	if err != nil { return err }
	return writeJSONLine(cmd, data)
}
out := cmd.OutOrStdout()
// REPLACE the single fmt.Fprintf here with the D-09 sectioned layout
```

Gate the new compact worktree notice line to the human-output branch only (RESEARCH's "TS CLI's own `--json` mode never calls `warn()`" finding, § Code Examples) — write it to `cmd.OutOrStdout()` (stdout), matching TS's `console.log`-based `warn()`, per D-12's explicit stdout requirement.

### `internal/query/files_status_test.go` (modified) — placeholder-assertion flip

**Analog:** itself — the "Phase-4-only keys render present-but-inert" subtest (lines 364-372) currently asserts `got.WorktreeMismatch != nil` fails the test. Flip this assertion's polarity for the worktree-mismatch fixture case (keep the same subtest *shape* — `t.Run(name, func(t *testing.T) {...})` with a `t.Fatalf` on violation) while `PendingChanges` stays asserted all-zero (D-06, unchanged).

### `testdata/golden/golden_parity_test.go` (modified) — `dbSizeBytes` exemption (D-08/Pitfall 2)

**Analog:** itself, line ~651's `findVolatileKeys(decoded, "our status.json")` call. Per RESEARCH Pitfall 2, add a narrowly-scoped exemption **at this call site only** (delete `"dbSizeBytes"` from the decoded map before the check, or add a `findVolatileKeysExcept` variant) — **do NOT touch** the shared `volatileKeys` map in `golden_test.go`, which correctly continues to strip `dbSizeBytes` from the frozen TS corpus fixtures.

---

## Shared Patterns

### Best-effort degradation on missing context
**Source:** `internal/query/status.go`'s `computeStale` (lines 135-157) — `e.repoRoot == ""` returns `false, nil` rather than erroring.
**Apply to:** `Engine.startPath == ""` in the worktree-detection call site (D-14's "Engines built via `New`/`NewWithRoot`... must degrade to 'no mismatch', never panic"); every `internal/gitmeta` subprocess call (any git error/timeout/absence ⇒ empty string / nil, never propagated).

### Prepend-a-banner-to-rendered-output
**Source:** `internal/query/render_markdown.go`'s `staleBanner`/`staleBannerText` (lines 18-32), applied at `RenderExplore` line 362.
**Apply to:** `worktreeNotice()` for all 7 non-status read renderers (D-12); the MCP status blockquote wrap is a variant of this same prepend pattern.

### Marshal*JSON is CLI-only, immutable, shared with golden parity
**Source:** every `Marshal*JSON` function in `internal/query/{traverse,files,status}.go`, confirmed by RESEARCH's exhaustive call-site table (`MarshalCallersJSON` ← `internal/cli/callers.go:41` AND `internal/mcp/tools.go:248`).
**Apply to:** all SURF-06 work — never edit a `Marshal*JSON` body; add sibling `Render*` functions; change only `internal/mcp/tools.go` call sites.

### `--json`/human-branch split gates warn-style lines to human output only
**Source:** `internal/cli/status.go` lines 40-52, `internal/cli/files.go` lines 44-71 (identical shape in both).
**Apply to:** the new CLI compact worktree-notice print — must live inside the human-output branch, never before/inside the `jsonOut` early return.

### Doc-comment decision table as the TS→Go remap convention
**Source:** `internal/query/status.go` lines 18-41 (`StatusResult`'s own doc comment).
**Apply to:** every new `StatusResult` field (`FilesByLanguage`, `DbSizeBytes`, live `WorktreeMismatch`) must gain a row in that same table, same format.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/gitmeta/worktree.go` (`gitWorktreeRoot`/`gitCommonDir` core subprocess logic) | utility | request-response (best-effort) | No production `os/exec` git-subprocess code exists anywhere in this codebase today — only test-only `gitHead`/`resolveColbymchenryCorpus` in `testdata/golden/golden_parity_test.go`, which lack the bounded-timeout/best-effort-nil-on-error contract D-03 requires. Use RESEARCH.md's Pattern 1 (already Go code, verbatim-ported from TS) as the implementation source instead of a codebase analog. |
| `internal/gitmeta`'s six-layout fixture builder (`git worktree add`/`git submodule add`/nested-clone/`.claude/worktrees/`/symlink construction) | test (fixture builder) | file-I/O + event-driven | No repo-*building* test helper exists — only repo-*cloning* (`resolveColbymchenryCorpus`). Building linked worktrees, submodules, and deterministic hermetic git commits (`GIT_AUTHOR_*`/`GIT_COMMITTER_*` env) is genuinely new test infrastructure; reuse only `t.TempDir()` + `t.Skip`-on-git-failure from the existing precedent. |

## Metadata

**Analog search scope:** `internal/query/`, `internal/mcp/`, `internal/cli/`, `internal/graphstore/`, `internal/migrate/`, `testdata/golden/` (via targeted Read + rg, not full directory Glob — file set was fully enumerated by CONTEXT.md/RESEARCH.md's canonical refs).
**Files scanned:** 12 (all files listed in File Classification) + 2 archtest packages + 1 golden fixture-builder file, read directly.
**Pattern extraction date:** 2026-07-15
