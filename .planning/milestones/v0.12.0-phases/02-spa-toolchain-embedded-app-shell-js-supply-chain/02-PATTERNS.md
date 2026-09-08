# Phase 2: SPA Toolchain, Embedded App Shell & JS Supply Chain - Pattern Map

**Mapped:** 2026-08-23
**Files analyzed:** 17 (Go-side: 6; JS/TS-side: 8 no-analog; config/CI: 3)
**Analogs found:** 6 / 9 classifiable files (the 8 pure-JS/TS files have no in-repo analog and are listed separately)

This phase is split cleanly in two. The Go-side files (embed, SPA handler, Taskfile guards,
structural test, CI) have strong, well-documented analogs already in this repo — copy from
them directly. The JS/TS-side files (`web/package.json`, `web/svelte.config.js`,
`web/vite.config.ts`, `web/src/**`, generated `ui_pb.ts`) introduce a technology this repo has
never had; there is nothing to copy from internally, and RESEARCH.md's Code Examples /
Architecture Patterns sections are the only reference for those.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `web/embed.go` | config (embed declaration) | file-I/O | `claudeassets.go` (repo root, `.claude/` embed) + `internal/mcp/resources.go:14-20` (in-package embed) | exact (structural constraint identical; in-package placement matches `resources.go`, root-ancestor reasoning matches `claudeassets.go`) |
| `internal/uiserver/spa.go` | route/handler (controller) | request-response, file-I/O | `internal/uiserver/originguard.go`, `internal/uiserver/degrade.go`, `internal/uiserver/truncate.go` | role-match (package conventions transfer; no prior file-serving handler exists in-repo) |
| `Taskfile.yml` `web:drift` (new task, BLD-03) | utility/guard (CI task) | batch | `Taskfile.yml` `proto:drift` (~line 179) | exact (this is explicitly "the pattern to imitate" per CONTEXT/RESEARCH) |
| `Taskfile.yml` extended `proto:drift` (D-07 floor 3→4 + TS enumeration) | utility/guard (CI task) | batch | `Taskfile.yml` `proto:drift` itself (self-modification) | exact |
| `Taskfile.yml` JS supply-chain gates (BLD-05 `strictDepBuilds` assertion, BLD-06 `pnpm audit` + lockfile-count sibling assertion) | utility/guard (CI task) | batch | `Taskfile.yml` `vuln:selftest` (~line 342) | exact (non-vacuity-proof shape) |
| `internal/upgrade/taskfile_shape_test.go` (BLD-07 structural check: no node/npm/npx/pnpm in `.goreleaser.yaml`/`release.yml`) | test (structural fixture assertion) | batch | `requiredCheckNames` fixture (:36-52) + `TestRequiredCheckNamesPreserved` / `TestRequiredCheckNamesPreserved_ZeroJobsIsError` (:698-760) | exact (fixture-backed structural assertion with a zero-guard is this test file's canonical shape) |
| `internal/uiserver/spa_test.go` (embed-vs-disk file-list diff, Criterion 1) | test | file-I/O | `internal/mcp/resources_schema_drift_test.go:39-66` (`resourceStemSetDiff`) — cited directly in RESEARCH.md Code Examples; also `internal/agents/claude_skillpackage_test.go:459-481` (`fs.WalkDir` over `embed.FS`, count assertion) | exact |
| `.github/workflows/ci.yml` `test` job (folds in JS gates per D-13) | CI config | batch | `.github/workflows/ci.yml` `test:` job (:46-85, existing steps: `actions/setup-go`, `task build`, `task vet`, `task test:unit`) | exact (extend in place, same job, same step-list shape) |
| `buf.gen.ts.yaml` (new, D-07) | config | transform (codegen) | `buf.gen.yaml` (repo root, existing) | exact — same tool, new scoped sibling file |
| `web/package.json`, `web/pnpm-workspace.yaml`, `web/svelte.config.js`, `web/vite.config.ts`, `web/src/**`, `web/src/lib/gen/ui_pb.ts` (generated) | component/config/model (JS/TS) | request-response (browser), transform (codegen) | **none** | no analog — see below |

## Pattern Assignments

### `web/embed.go` (config, file-I/O)

**Analogs:** `claudeassets.go:1-19` (repo root) and `internal/mcp/resources.go:14-20`

**Why two analogs, and which parts to take from each:**
- `claudeassets.go:7-17` documents *why* the file must sit at an ancestor of everything it
  embeds — Go's `//go:embed` patterns forbid `..` path elements. `internal/uiserver/` and
  `web/build/` are **siblings**, not ancestor/descendant, so the directive cannot live in
  `internal/uiserver/spa.go` (this was CONTEXT.md's original discretion clause; RESEARCH.md
  Pitfall 5 corrects it). The only valid home is a file **inside `web/`**.
- `internal/mcp/resources.go:14-20` is the *placement* pattern to copy: the embed declaration
  lives beside the directory it embeds (in-package), not at the repo root — because `web/`
  itself (unlike `.claude/` vs `internal/`) is already the ancestor of `build/`.

**Doc-comment convention to copy** (`claudeassets.go:1-19`):
```go
// Package claudeassets embeds this repository's own canonical .claude/
// package ...
//
// This file MUST live at the repository root, not under internal/. Go's
// //go:embed patterns may not contain ".." path elements (golang/go#46056)
// and are resolved only relative to, or below, the directory containing
// the source file that carries the directive. ...
```
Adapt for `web/embed.go`: state that it must live inside `web/` (not `internal/uiserver/`)
for the identical reason, and cross-reference this exact prior instance.

**Embed declaration shape to copy** (`internal/mcp/resources.go:14-20`):
```go
package mcp

import (
	"embed"
	...
)

//go:embed resources/*.md
var resourcesFS embed.FS
```
Adapt: `package web`, `//go:embed all:build` (the `all:` prefix is REQUIRED — SvelteKit's
`_app/` directory is underscore-prefixed and silently dropped by Go's default embed walk
without it — D-03), `var BuildFS embed.FS`.

---

### `internal/uiserver/spa.go` (controller, request-response + file-I/O)

**Analogs:** `internal/uiserver/originguard.go:1-6`, `internal/uiserver/degrade.go:1-24`, `internal/uiserver/truncate.go:1-11`

**Package-level conventions to copy (all three files share this shape):**
1. One decision-driving concern per file (`originguard.go` = SRV-02 rebinding defense,
   `degrade.go` = degraded-store classification, `truncate.go` = D-11 output bounds).
2. A doc comment at the top of the file that **names the decision IDs it implements**, e.g.
   `originguard.go:1-5`:
   ```go
   // Package uiserver implements codegraph ui's local, loopback-only Connect
   // RPC server. originguard.go is the first piece to exist in this package
   // (SRV-02): the exact-match Origin/Host control ...
   ```
   `spa.go` should open the same way, naming D-09/D-10/D-11/D-12.
3. Package-level doc comment only appears once (in `originguard.go`, the first file
   chronologically) — do not repeat the `package uiserver` doc comment block in `spa.go`.
4. Exported symbols get long, decision-referencing doc comments explaining *why*, not just
   *what* — see `degrade.go:12-21`'s `degradeKind` type comment for the density expected.

**Wiring point** (`internal/uiserver/server.go:111-130`):
```go
mux := http.NewServeMux()
mux.Handle(uiv1connect.NewUIServiceHandler(
	&uiService{repoPath: o.RepoPath},
	connect.WithSendMaxBytes(transportSendMaxBytes),
	connect.WithReadMaxBytes(transportReadMaxBytes),
))

guarded := originHostGuard(port, mux)
```
`spa.go`'s handler registers on the same `mux` at `"/"` (D-09: Go 1.26 `ServeMux`
most-specific-pattern-wins handles RPC-vs-SPA precedence automatically) **before**
`originHostGuard` wraps the whole thing — the SPA handler must never be mounted outside
`originHostGuard`, per RESEARCH.md's Integration Points.

**What NOT to copy:** none of the three existing files serve files or set `Content-Type`/
`Cache-Control` — there is no in-repo precedent for that half of `spa.go`. RESEARCH.md
Pattern 3 (hand-written handler using `fs.ReadFile`/`fs.Stat` against an `fs.Sub`, explicitly
avoiding `http.FileServer`/`http.FileServerFS`'s auto-redirect-on-`index.html` behavior) is
the only guidance for that part; there is no codebase analog to fall back on.

---

### `Taskfile.yml` — `web:drift` guard (BLD-03) and extended `proto:drift` (D-07)

**Analog:** `Taskfile.yml` `proto:drift` (:179-262)

**THE pattern to imitate — structure, quoted in full for the parts that transfer:**

1. **Reasoning documented in `desc:` before any `cmds:`** (:180-208) — states what is
   regenerated, into what (a scratch tree, never in place), what floor triggers failure and
   why, and explicitly invokes rule `84d1gfpywd` on itself:
   > "so an enumeration that silently finds nothing can never read as a clean pass (rule
   > 84d1gfpywd) — this is BLD-04's own 'report how many it compared' guard, so it is doubly
   > bound not to be the vacuous shape it exists to catch."

2. **Preconditions block** (:209-215) fails loudly on missing tooling before `cmds:` runs —
   copy this shape for `web:drift`'s Node/pnpm precondition (D-05's "must fail loudly, never
   skip" requirement):
   ```yaml
   preconditions:
     - sh: command -v go
       msg: "go not found — proto:drift builds protoc-gen-go/protoc-gen-connect-go and runs buf via go tool, same as proto:gen."
   ```

3. **Scratch-tree regeneration, count-before-compare — the positive assertion is line 229:**
   ```bash
   scratch=$(mktemp -d)
   trap 'rm -rf "${scratch}"' EXIT

   files=$(git ls-files -- 'internal/schema/*.pb.go' 'internal/uiproto/uiv1/*.pb.go' 'internal/uiproto/uiv1/uiv1connect/*.connect.go')
   nfiles=0
   if [ -n "${files}" ]; then
     nfiles=$(printf '%s\n' "${files}" | wc -l | tr -d ' ')
   fi
   echo "proto:drift: compared ${nfiles} generated files"     # <-- POSITIVE ASSERTION (rule 84d1gfpywd): count is printed and named BEFORE the comparison runs, so a broken enumeration cannot silently read as a clean pass.
   if [ "${nfiles}" -lt 3 ]; then
     echo "::error::proto:drift: enumerated only ${nfiles} committed generated files ..."
     exit 1
   fi
   ```
   **D-07's exact edit here:** the floor `-lt 3` at line 230 must move to `-lt 4` (RESEARCH.md
   settles the arithmetic: 3 Go files + exactly 1 TS file, `ui_pb.ts`, **provided**
   `buf.gen.ts.yaml` sets `opt: target=ts` — RESEARCH.md Pitfall 3 warns the default emits 2
   files, which would make 5 the wrong-but-plausible floor). The `files=$(git ls-files ...)`
   enumeration on line 224 must also gain a fourth glob: `'web/src/lib/gen/*.ts'`.

4. **Byte-diff against a fresh scratch regeneration** (:235-262) — copy this shape **only**
   for the Go-side extension of `proto:drift` (the new TS file it now also enumerates). **Do
   NOT copy this shape for `web:drift` / BLD-03 itself** — RESEARCH.md Pitfall 1 is explicit
   that Vite/SvelteKit builds are not byte-reproducible across runs (two cited, maintainer-
   declined-to-fix Vite issues), so a literal `proto:drift`-style byte-diff on `web/build/`
   produces false-positive RED runs. BLD-03's `web:drift` must instead be a **source-hash
   staleness check**: hash `git ls-files` under `web/src`, `web/static`, and the toolchain
   config files, store the digest in a committed marker (e.g. `web/build/.source-sha256`), and
   compare. Still copy the *reporting* discipline from this section — a count line before any
   comparison, a named `::error::` on mismatch — just not the `cmp -s` byte-diff mechanism.

5. **Success line at the end names what was proven** (:262):
   ```bash
   echo "proto:drift: all ${nfiles} generated files byte-identical to the pinned toolchain's regeneration (temporary tree only — source tree untouched)"
   ```

---

### `Taskfile.yml` — BLD-05/BLD-06 non-vacuity assertions

**Analog:** `Taskfile.yml` `vuln:selftest` (:342-382)

**Structure to copy exactly** (this is the repo's canonical non-vacuity proof):
- `desc:` states plainly that, unlike its sibling advisory task, **this** task is allowed to
  fail the build (:343-356) — same posture BLD-06's sibling assertion needs relative to
  `pnpm audit`'s own advisory-style run.
- **The positive assertion is the two `if` blocks at :372-380** — not the exit code check
  alone, but the exit code check **AND** the specific-string check:
  ```bash
  if [ "${status}" -ne 3 ]; then
    echo "::error::vuln:selftest FAILED — govulncheck -mode=binary exited ${status} (want 3) ..."
    exit 1
  fi

  if ! printf '%s' "${out}" | grep -q "GO-2026-5932"; then
    echo "::error::vuln:selftest FAILED — scan exited 3 but its output does not name GO-2026-5932; a different advisory firing is not accepted as proof this specific detection path works"
    exit 1
  fi
  ```
  This is the shape to copy for **BLD-05** (assert `strictDepBuilds` is actually in effect —
  exact-match assertion, not a string grep against a warning that could reword): assert both
  an exit code/exit-status AND a specific named condition, never either alone.
- **For BLD-06 specifically, D-15 diverges from this shape on purpose** — do not pin an exact
  advisory ID the way `vuln:selftest` pins `GO-2026-5932`, because `pnpm audit --json` has no
  scanned-package-count field and a pinned JS advisory rots (packages get yanked/patched
  faster than the Go proxy-cached ecosystem). Instead the sibling assertion counts scanned
  packages from `pnpm-lock.yaml` — copy the **reporting discipline** (count printed, named
  failure) from `vuln:selftest`'s shape, not its literal "assert one exact advisory" mechanism.
- Final success line names exactly what was proven (:382):
  ```bash
  echo "vuln:selftest: PASS — govulncheck -mode=binary exited 3 and named GO-2026-5932 against testdata/vulnredpoc"
  ```

---

### `internal/upgrade/taskfile_shape_test.go` — BLD-07 structural test

**Analog:** `requiredCheckNames` fixture (:36-52) + `TestRequiredCheckNamesPreserved` /
`TestRequiredCheckNamesPreserved_ZeroJobsIsError` (:698-760) — both in the same file.

**Fixture-backed structural assertion with a zero-guard — the exact shape to copy:**
```go
// requiredCheckNames is the literal fixture of GitHub ruleset 20157557's
// six required-status-check contexts plus pr-title ... Source: `gh api
// repos/seanb4t/codegraph-go/rulesets/20157557`, re-verified live
// 2026-08-01 ... Re-verify the same way before editing this fixture — a
// stale fixture here would make this guard assert the wrong thing rather
// than fail loudly.
var requiredCheckNames = []string{ ... }
```
CONTEXT.md's own "Baseline verified clean" note for BLD-07 (zero matches for
`node`/`npm`/`npx`/`pnpm` in `.goreleaser.yaml`/`.github/workflows/release.yml`, positive-
controlled against 29/14 `go` hits in the same files) is exactly the `forbiddenToolPackages`
pattern already used a few lines below `requiredCheckNames` in this same test file — reuse
that positive-control discipline for the new assertion rather than inventing a fresh one.

**The zero-guard pairing to copy** (:698-760):
```go
func TestRequiredCheckNamesPreserved(t *testing.T) {
	entries, err := os.ReadDir(workflowsDir)
	...
	scanned := 0
	for _, entry := range entries {
		...
		scanned++
	}
	if scanned == 0 {
		t.Fatalf("TestRequiredCheckNamesPreserved: found zero workflow files under %s", workflowsDir)
	}
	...
}

// TestRequiredCheckNamesPreserved_ZeroJobsIsError is the edge case: ...
// must surface as a non-nil error ..., never as a silently-passing empty
// set — the same CR-01 defect class every parser in this file is built
// to avoid.
func TestRequiredCheckNamesPreserved_ZeroJobsIsError(t *testing.T) { ... }
```
For BLD-07: write the paired test (scans `.goreleaser.yaml` + `.github/workflows/release.yml`
for `node`/`npm`/`npx`/`pnpm` tokens, asserts zero matches) **plus** its own zero-guard sibling
test proving the scanner itself is not vacuously matching nothing because it read an empty or
missing file — the exact CR-01 defect class this pair exists to avoid.

---

### `internal/uiserver/spa_test.go` — Criterion 1 embed-vs-disk diff

**Analogs:** `internal/mcp/resources_schema_drift_test.go:39-66` (`resourceStemSetDiff`,
cited verbatim in RESEARCH.md) and `internal/agents/claude_skillpackage_test.go:459-481`
(`fs.WalkDir` count assertion + "guard-the-guard" non-vacuous on-disk cross-check).

**Set-diff function to copy verbatim in shape** (from RESEARCH.md's own Code Examples,
sourced this session from `internal/mcp/resources_schema_drift_test.go:39-66`):
```go
func resourceStemSetDiff(expected, actual []string) (missing, orphaned []string) {
	expectedSet := make(map[string]bool, len(expected))
	for _, s := range expected {
		expectedSet[s] = true
	}
	actualSet := make(map[string]bool, len(actual))
	for _, s := range actual {
		actualSet[s] = true
	}
	for s := range expectedSet {
		if !actualSet[s] {
			missing = append(missing, s)
		}
	}
	for s := range actualSet {
		if !expectedSet[s] {
			orphaned = append(orphaned, s)
		}
	}
	sort.Strings(missing)
	sort.Strings(orphaned)
	return missing, orphaned
}
```
Adapt: `expected` = `filepath.WalkDir` over on-disk `web/build/`; `actual` = `fs.WalkDir` over
the embedded `web.BuildFS`. Both empty ⇒ Criterion 1 satisfied. This test goes RED by
construction if `//go:embed build` is substituted for `//go:embed all:build` — every
`_app/`-prefixed path is dropped by Go's default (non-`all:`) embed walk.

**Walk-and-count pattern to copy** (`internal/agents/claude_skillpackage_test.go:459-481`,
read this session):
```go
func TestClaudeAssets_EmbedsNoVerificationTranscripts(t *testing.T) {
	var walked []string
	err := fs.WalkDir(claudeassets.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			walked = append(walked, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk claudeassets.FS: %v", err)
	}
	if len(walked) != 3 {
		t.Fatalf("expected exactly 3 embedded files, got %d: %v", len(walked), walked)
	}
	// Guard-the-guard: ... confirming the real on-disk verification/
	// directory it must NOT have picked up is actually non-empty.
	...
}
```
Copy the "guard-the-guard" idea directly: alongside the embed-vs-disk diff, assert the on-disk
`web/build/` side actually contains something non-trivial (e.g. `> 0` files, or specifically
contains an `_app/immutable/` entry) so the test cannot pass vacuously against an empty build
directory.

---

### `.github/workflows/ci.yml` `test` job (D-13 fold-in)

**Analog:** the job itself, `.github/workflows/ci.yml:46-85` (existing, extend in place)

**Existing step shape to copy for each new JS step** (:53-85):
```yaml
- name: Set up Go
  uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
  with:
    go-version-file: go.mod
    cache: false
...
- name: Test (excluding internal/daemon — isolated below)
  # IN-08 (03-REVIEW.md): `set -e` does not propagate failures out of
  # a process substitution — ...
  run: task test:unit
```
Two conventions to copy exactly for the new `actions/setup-node` step and every JS `task`
invocation:
1. **Pin actions by full commit SHA with a version comment** (`uses: actions/setup-go@924ae...  # v6.5.0`) — apply the same pinning discipline to `actions/setup-node`. RESEARCH.md
   Pitfall 4 requires pinning `node-version: '24'` explicitly (not `'latest'`) since Corepack
   was removed from Node 25+.
2. **A comment above any non-obvious step explaining why**, referencing the decision ID —
   every existing step in this job does this (`IN-08 (03-REVIEW.md)` above `task test:unit`).
   New steps (`corepack enable`, `pnpm install --frozen-lockfile`, `task web:audit`,
   `task web:drift`) should follow the same comment-with-citation convention, e.g. citing D-13,
   D-16, Pitfall 4.

No new job is created (D-13) — new steps are appended inside this existing `test:` job block.

---

### `buf.gen.ts.yaml` (new, D-07)

**Analog:** `buf.gen.yaml` (repo root, unchanged by this phase) — same mechanism, new sibling
scoped file. Quoted from RESEARCH.md Code Examples (read this session from the actual repo
file):
```yaml
# buf.gen.yaml — UNCHANGED by this phase.
version: v2
plugins:
  - local: protoc-gen-go
    out: .
    opt: paths=source_relative
  - local: protoc-gen-connect-go
    out: .
    opt: paths=source_relative
```
New sibling:
```yaml
# buf.gen.ts.yaml — NEW this phase (D-07's resolved mechanism)
version: v2
inputs:
  - directory: internal/uiproto/uiv1
plugins:
  - local: protoc-gen-es
    out: web/src/lib/gen
    opt:
      - target=ts   # REQUIRED — default is "js+dts" (2 files), not "ts" (1 file).
```

---

## No Analog Found

Files with no close match anywhere in this repository (planner must use RESEARCH.md's
Standard Stack / Architecture Patterns / Code Examples sections instead, not an invented
in-repo analog):

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `web/package.json` | config | n/a | No prior `package.json` in the tree at all |
| `web/pnpm-workspace.yaml` | config | n/a | First pnpm usage in this repo |
| `web/svelte.config.js` | config | n/a | First SvelteKit usage |
| `web/vite.config.ts` | config | n/a | First Vite usage |
| `web/src/routes/+layout.svelte`, `+page.svelte` | component | request-response (browser) | First Svelte component in the repo |
| `web/src/lib/utils.ts` (`cn()` helper) | utility | transform | No prior TS utility code |
| `web/src/lib/gen/ui_pb.ts` (generated) | model | transform (codegen output) | Generated by a tool this repo has never run before (`protoc-gen-es`); the Go-side generated-file *conventions* (committed + drift-guarded) transfer, but there is no TS generated-file precedent to read code from |
| `web/components.json` | config | n/a | shadcn-svelte-specific, first use |

**Positive control on the absence claim:** searched for any pre-existing JS/TS source outside
test fixtures —
```
$ rg -l --type-add 'jsts:*.{js,ts,svelte,json}' -tjsts . -g '!web/**' -g '!node_modules/**'
```
The only matches are `package.json`-shaped or `.js`-suffixed files under
`internal/indexer/testdata/**`, which are parser test **inputs** (fixtures the Go indexer
parses), not project source or convention to imitate — confirmed by inspecting their directory
depth (`internal/indexer/testdata/`) and content (single-purpose synthetic snippets, not an
application). This is a real, controlled zero, not a mis-aimed search: the same query without
the `-g '!web/**'` exclusion (irrelevant pre-Phase-2, since `web/` doesn't exist yet) and
without the testdata exclusion returns exactly those fixture files and nothing else, confirming
the glob itself is not silently failing to match.

## Shared Patterns

### Rule `84d1gfpywd` — every new guard MUST carry a positive assertion it did its work

Applies to all four Taskfile guard additions in this phase (`web:drift`'s count line, the
extended `proto:drift` floor, BLD-05's `strictDepBuilds`-in-effect assertion, BLD-06's
lockfile-scanned-count assertion). The two canonical shapes already in this repo:

- **Count-printed-before-comparison** (`proto:drift` :229, quoted above) — the "N compared"
  echo runs unconditionally before any pass/fail branch, so a broken enumeration prints `0`
  and is caught by the floor check rather than silently reading as "nothing differed."
- **Exact-match-plus-named-value** (`vuln:selftest` :372-380, quoted above) — never trust a
  bare exit code alone if a more specific signal (an advisory ID, a setting's literal value)
  is available; assert both.

### One-concern-per-file + decision-ID doc comments (`internal/uiserver` package convention)

`degrade.go`, `truncate.go`, `originguard.go` (and now `spa.go`) each open with a doc comment
naming the specific decision IDs (SRV-02, D-11, D-14/D-15/D-16, etc.) the file implements, and
each exported symbol's comment explains *why*, at comparable density to `degrade.go:12-21`'s
`degradeKind` comment. Apply this convention to `spa.go` and `web/embed.go` alike.

### `go:embed` ancestor-of-target constraint (`claudeassets.go` precedent)

Any new embed directive in this codebase must be checked against this constraint before
placement is finalized: the file carrying `//go:embed` must be an ancestor (same directory or
above) of everything the pattern matches — no `..` is permitted. This repo has now hit this
twice (`claudeassets.go` for `.claude/`, `web/embed.go` for `web/build/`); the planner should
treat this as a standing rule for any future embed addition, not a one-off surprise.

## Metadata

**Analog search scope:** repo root (`claudeassets.go`), `internal/uiserver/*.go`,
`internal/mcp/resources.go` + drift test, `internal/agents/claude_skillpackage_test.go`,
`internal/upgrade/taskfile_shape_test.go`, `Taskfile.yml`, `.github/workflows/ci.yml`,
`buf.gen.yaml`, `.gitignore`; controlled negative search for pre-existing JS/TS across the
whole tree excluding `internal/indexer/testdata/**`.
**Files scanned:** 13 read directly this session (line ranges cited above) + 1 negative-control
search.
**Pattern extraction date:** 2026-08-23
