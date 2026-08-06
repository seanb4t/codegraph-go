# Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle - Pattern Map

**Mapped:** 2026-08-04
**Files analyzed:** 10 (new/modified)
**Analogs found:** 9 / 10

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `test/wireoracle/{capture,scenarios,normalize}.go` + `main.go` | utility (test-tooling, standalone spawn/scan) | streaming (stdio, request-response over JSON-RPC) | `test/integration/mcp_stdout_purity_test.go` | exact (same spawn/scan loop, generalized) |
| `test/wireoracle/*_test.go` (invokes capture, diffs frozen transcripts) | test | file-I/O + request-response | `test/integration/main_test.go` (`runBinary`/`copyFixture`/`TestMain`) | role-match |
| `testdata/wireoracle/fixture/` | config/fixture (inert data, not a package) | file-I/O | `internal/indexer/testdata/gofixture` (shape to copy, NOT reuse — D-08) | role-match, explicitly not to be shared |
| `testdata/wireoracle/*.golden` | fixture (frozen transcripts) | batch | none — no existing golden-transcript convention in this repo; `testdata/golden/golden_parity_test.go` is same directory pattern but a different comparison shape (typed SDK results, not raw bytes) | partial |
| `internal/mcp/archtest/` (new package) or extend `internal/graphstore/archtest` — VRFY-02 guard | test (archtest) | transform (AST/type predicate) | `internal/graphstore/archtest/import_graph_test.go` (predicate/import-graph shape) + `stdout_confinement_test.go`/`stdout_closure_selftest_test.go` (self-defeat guard shape) | exact |
| `internal/cli/archtest/` (new) or extend `internal/cli/present/archtest/import_graph_test.go` — SDK-02 guard | test (archtest) | transform (import-graph) | `internal/cli/present/archtest/import_graph_test.go` (`TestNoCharmInServeReachablePackages`) | exact |
| `internal/mcp/server.go` — `BuildServer` return type, `Server` seam, `AddAfterInitialize` hook, `version` const | production (service/seam) | request-response | itself (existing file, modified in place) — no external analog needed, it is its own precedent | n/a (self) |
| `internal/cli/serve.go` — drop `mark3labs/mcp-go/server` import, call `s.ServeStdio()` | controller (CLI command wiring) | request-response | itself (existing file, modified in place) | n/a (self) |
| stderr session line (VRFY-03/D-13/D-14) | utility (diagnostic emitter) | event-driven (fires once per connection, after `initialize`) | `internal/mcp/server.go`'s `WarnUnknownToolsTo` (existing stderr-writer-seam convention) | exact |
| `docs/MCP-2026-07-28-SCOPING.md`, `docs/MCP-8-AGENT-AUDIT.md` | docs | batch (dated evidence table) | `docs/FLAG-PARITY.md` | exact |

## Pattern Assignments

### `test/wireoracle/` capture core (VRFY-01/D-01)

**Analog:** `test/integration/mcp_stdout_purity_test.go` (read in full this session)

**Imports pattern** (lines 1-19):
```go
package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)
```
For the oracle's own package, drop the `mark3labs/mcp-go/mcp` import entirely (VRFY-01 explicitly forbids the SDK under test as its own oracle/type source) — replace `mcp.LATEST_PROTOCOL_VERSION` with the oracle's own hand-authored version string constants per scenario (D-06).

**Concurrency-safe stderr capture** (lines 21-44, copy verbatim — this is exactly WR-01's fix and applies unchanged):
```go
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}
```

**Core spawn/write/scan loop** (lines 71-162, the loop to generalize):
```go
cmd := exec.Command(binPath, "serve", "--mcp")
cmd.Dir = dir
cmd.Env = append(os.Environ(), "CODEGRAPH_MCP_TOOLS=status", "CODEGRAPH_NO_WATCH=1")

stdin, err := cmd.StdinPipe()
stdout, err := cmd.StdoutPipe()
stderrBuf := &syncBuffer{}
cmd.Stderr = stderrBuf // diagnostics land here — never asserted for purity (but VRFY-03/D-15 DOES target-assert stderr per scenario, unlike this ancestor test)

// ... cmd.Start(), t.Cleanup(Kill+Wait) ...

writeLine := func(v any) {
	b, _ := json.Marshal(v)
	b = append(b, '\n')
	stdin.Write(b)
}

scanner := bufio.NewScanner(stdout)
scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB cap — reuse verbatim
```

**Bounded-deadline pattern** (lines 164-209): the `time.After(30 * time.Second)` deadline + `select` over a channel-fed scanner goroutine is the reusable timeout shape — copy for the oracle's own per-scenario read loop (Security Domain's "bounded timeouts" note references this exact precedent).

**Error handling pattern**: every failure path is `t.Fatalf` with the raw offending bytes quoted verbatim (line 189: `t.Fatalf("non-JSON-RPC byte on stdout: %q", ln.raw)`) — never silently continuing. The oracle's byte-exact comparator should follow the same "quote the exact offending bytes" discipline, not a summarized diff.

**Binary/fixture staging pattern**: `test/integration/main_test.go`'s `TestMain` (lines 39-57, builds the real binary once via `go build -o binPath github.com/seanb4t/codegraph-go/cmd/codegraph`, hard `os.Exit(1)` on build failure) and `runBinary`/`copyFixture` (lines 126-183) are the binary-build-and-fixture-staging harness to reuse or closely mirror in `test/wireoracle`'s own `TestMain` — **except** `copyFixture` must NOT be reused as-is: D-08 requires a **dedicated** `testdata/wireoracle/fixture/` tree, not `internal/indexer/testdata/gofixture`, specifically so wire-oracle transcripts can never be invalidated by an unrelated fixture edit. Copy the `filepath.WalkDir`-based copy-to-tempdir mechanics from `copyFixture` (lines 132-163), but point `src` at the new dedicated fixture path.

---

### `internal/mcp/archtest/` — VRFY-02 guard (repo-owned protocol-version literal, no SDK-owned constant reference tree-wide)

**Analog:** `internal/graphstore/archtest/import_graph_test.go` (flat `Tests: true` load shape) + `internal/graphstore/archtest/stdout_confinement_test.go` + `stdout_closure_selftest_test.go` (self-defeat/mutation-proof guard shape)

**`packages.Load` config for a flat, whole-tree "did anyone write this anywhere" scan** (import_graph_test.go lines 29-45 — this is the RIGHT shape per RESEARCH's Anti-Patterns section, not the six-package `NeedDeps` closure walk):
```go
cfg := &packages.Config{
	Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps,
	// Tests: true is required so a reference that appears only inside a
	// _test.go file is not invisible to this check.
	Tests: true,
}
pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/...")
if err != nil {
	t.Fatalf("packages.Load: %v", err)
}
if len(pkgs) == 0 {
	t.Fatal("packages.Load returned no packages — the module import graph did not resolve")
}
```
VRFY-02 additionally needs `NeedTypes | NeedSyntax | NeedTypesInfo` (not present in `import_graph_test.go`'s import-only check) since it must resolve `info.Uses` on `*ast.SelectorExpr` nodes, per RESEARCH Pattern 2's `isExternalProtocolVersionConstant` predicate — mirror `stdout_confinement_test.go`'s fuller `Mode` (lines 106-111) for the syntax/types bits, but keep `import_graph_test.go`'s flat `Tests: true, "./..."` target rather than that file's six named `guardedPackages` roots (RESEARCH's Anti-Patterns section explicitly warns against the reachability-closure shape here — this is a "did anyone write this anywhere" question, not a "is this reachable from serve" question).

**Self-defeat / mutation-proof sanity check pattern** (import_graph_test.go lines 59-66, and stdout_closure_selftest_test.go in full — the pattern every new guard in this phase must copy):
```go
// Sanity check that this test can actually detect a real importer: if
// [the thing being guarded] no longer exists, the check above is
// vacuously true for the wrong reason — silently disabling this test's
// ability to ever fail.
if !foundGraphstoreImporter {
	t.Fatal("no package under internal/graphstore was found importing pebble/v2 — this test cannot verify enforcement; check that pebble_store.go still imports pebble/v2 and that packages.Load resolved it")
}
```
For VRFY-02, the equivalent self-defeat proof is `stdout_closure_selftest_test.go`'s `packages.Config.Overlay` technique (full file quoted above in code_context — read this file in the plan, not just this excerpt): inject a synthetic file (via `Overlay: map[string][]byte{...}`) that imports the real `mcp` package and references `mcp.LATEST_PROTOCOL_VERSION`, confirm the guard goes RED, THEN confirm it is green against the real tree only after the 6 known sites are migrated off the constant. RESEARCH's own "Mutation-proof requirement" paragraph (Pattern 2 section) names this exact technique.

**Predicate shape to adapt** (`stdout_confinement_test.go` lines 237-254, `isOSStdoutRef` — the shape RESEARCH Pattern 2 explicitly generalizes):
```go
func isOSStdoutRef(sel *ast.SelectorExpr, info *types.Info) bool {
	if sel.Sel.Name != "Stdout" {
		return false
	}
	xIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkgName, ok := info.Uses[xIdent].(*types.PkgName)
	if !ok || pkgName.Imported().Path() != "os" {
		return false
	}
	obj, ok := info.Uses[sel.Sel]
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "os" {
		return false
	}
	return true
}
```
Generalize to: external package (`!= this module`) + name matches `(?i)protocol.?version` + not a struct field (`v, ok := obj.(*types.Var); ok && v.IsField()` excludes `req.Params.ProtocolVersion` field access) — this exact predicate is spelled out in full in `01-RESEARCH.md`'s Pattern 2 section (`isExternalProtocolVersionConstant`); copy from there directly, it is already written to this repo's conventions.

**Failure-loudly pattern for "resolved fewer packages than expected"** (stdout_closure_selftest_test.go / stdout_confinement_test.go lines 121-123):
```go
if len(pkgs) != len(guardedPackages) {
	t.Fatalf("packages.Load resolved %d packages, want %d (guardedPackages) — a guarded package may have been renamed or moved, silently disabling this test's coverage of it", len(pkgs), len(guardedPackages))
}
```

---

### `internal/cli/archtest/` or extended `internal/cli/present/archtest/import_graph_test.go` — SDK-02 guard (`serve.go` imports no `mark3labs/mcp-go`)

**Analog:** `internal/cli/present/archtest/import_graph_test.go`'s `TestNoCharmInServeReachablePackages` — same shape as `internal/graphstore/archtest/import_graph_test.go` (both files were read this session; the `internal/cli/present/archtest` one specifically targets `internal/cli`-adjacent import boundaries, making it the closer sibling for this guard). Reuse `isAllowedImporter`/`stripTestVariant`-style helpers (import_graph_test.go lines 69-90) inverted: instead of asserting "only X may import Y," assert "package `internal/cli` (specifically `serve.go`) must NOT import `github.com/mark3labs/mcp-go/...`" — a negative-import assertion. `pkg.Imports["github.com/mark3labs/mcp-go/server"]` should be absent from `internal/cli`'s import set once SDK-02 lands; the guard should fail loudly with the exact `pkg.PkgPath`.

---

### `internal/mcp/server.go` — `BuildServer` seam + stderr session line (SDK-02, VRFY-03)

**Analog:** the file itself — quoted in full above in code_context (94 lines, all read this session). This is a modification, not a new-file pattern; the plan should treat the following excerpts as the load-bearing current state to diff against:

**Current signature and doc comment to change** (server.go lines 73-98):
```go
func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath, startPath string) *server.MCPServer {
	s := server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true))
	if !hasIndex {
		return s
	}
	// ...
	s.AddTool(exploreTool(), exploreHandler(repoPath, startPath, detector))
	for _, name := range companionNames {
		if allowlist[name] {
			s.AddTool(companionTool(name), companionHandler(name, repoPath, startPath, detector))
		}
	}
	return s
}
```

**Existing stderr-writer-seam convention to extend for D-13/D-14's session line** (server.go lines 63-71 — `WarnUnknownToolsTo` is the house style: an explicit `io.Writer` parameter, never a hidden global, doc comment explicitly says "stdout is reserved... callers must pass os.Stderr, never os.Stdout, in production"):
```go
func WarnUnknownToolsTo(w io.Writer, unknown []string) {
	for _, name := range unknown {
		fmt.Fprintf(w, "codegraph mcp: unknown tool name %q in CODEGRAPH_MCP_TOOLS, ignoring\n", name)
	}
}
```
The VRFY-03 session line should be wired the same way — `BuildServer` gains an explicit `stderr io.Writer` parameter (RESEARCH Pattern 4's sketch confirms this), and the line itself uses `fmt.Fprintf(stderr, "codegraph: mcp-session requested=%s negotiated=%s client=%s/%s tools=%d\n", ...)`, matching `WarnUnknownToolsTo`'s `fmt.Fprintf(w, "codegraph ...")` prefix-and-Fprintf convention exactly (D-14's "fixed prefix + key=value pairs" format is a direct sibling of this existing `"codegraph mcp: ..."` line, not an invented new style).

**Hook mechanism** — `server.Hooks.AddAfterInitialize`, confirmed in RESEARCH's Pattern 3 (module source `github.com/mark3labs/mcp-go@v0.56.0/server/hooks.go:74-75`, verbatim):
```go
type OnAfterInitializeFunc func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult)
```

**Seam interface sketch** (RESEARCH Pattern 4, not verbatim — planner's call per CONTEXT.md's "Claude's Discretion"):
```go
type Server interface {
	ServeStdio() error
}

type mark3labsServer struct{ inner *server.MCPServer }

func (m *mark3labsServer) ServeStdio() error { return server.ServeStdio(m.inner) }
```

**Ripple this pattern-map must flag for the planner** (Pitfall 4, RESEARCH): `internal/mcp/server_test.go`'s `listToolNames(t, s *server.MCPServer)` (used at server_test.go:122,153,164 per the set-equality tests below) and `testdata/golden/golden_parity_test.go`'s `callExploreViaMCP` both construct `mcpclient.NewInProcessClient(s)` against the CONCRETE mark3labs type — changing `BuildServer`'s return type breaks these call sites unless a test-only concrete-type accessor is retained. This is a real, budgeted ripple, not a footnote.

---

### `internal/cli/serve.go` — SDK-02's two production lines

**Current state** (serve.go, read this session):
```go
// line 13
"github.com/mark3labs/mcp-go/server"

// lines 252-253
s := mcp.BuildServer(hasIndex, allowlist, repoPath, start)
return server.ServeStdio(s)
```
**Target shape** (RESEARCH Pattern 4's sketch, confirmed against the real surrounding code this session — `allowlist, unknown := mcp.ParseAllowlist(...)` at line 241 and `mcp.WarnUnknownToolsTo(cmd.ErrOrStderr(), unknown)` at line 242 already establish `cmd.ErrOrStderr()` as this file's existing stderr-writer idiom to thread into `BuildServer`'s new stderr parameter):
```go
s := mcp.BuildServer(hasIndex, allowlist, repoPath, start, cmd.ErrOrStderr())
return s.ServeStdio()
```
Import line 13 (`"github.com/mark3labs/mcp-go/server"`) is deleted entirely — this is SDK-02's literal acceptance criterion.

---

## Shared Patterns

### Archtest-over-grep (house style)
**Source:** `internal/graphstore/archtest/{import_graph_test.go,stdout_confinement_test.go,stdout_closure_selftest_test.go}`, `internal/cli/present/archtest/import_graph_test.go`
**Apply to:** VRFY-02's guard, SDK-02's guard
Every guard in this phase must be a `go/packages`-based archtest with a self-defeat/mutation-proof companion, never a `rg`/grep-shaped CI gate — this repo has a documented incident (referenced in both CONTEXT.md and RESEARCH.md) of an inverted `rg -qv` gate.

### Fail-loudly / self-defeat companion tests
**Source:** `internal/upgrade/taskfile_shape_test.go:662` (`TestTaskfileShapeHelpersFailLoudly`), `:896` (`TestCheckCrossParsersFailLoudly`)
**Apply to:** any new parsing/classification helper introduced by this phase (e.g. a normalization-rule parser, a scenario-name parser)
```go
func TestTaskfileShapeHelpersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "parseWorkflowJobNames: empty input",
			fn: func() error {
				_, err := parseWorkflowJobNames("")
				return err
			},
		},
		// ...
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.fn(); err == nil {
				t.Fatalf("%s: expected a non-nil error, got nil", c.name)
			}
		})
	}
}
```
**[CORRECTED by orchestrator 2026-08-04]** All **three** `FailLoudly` tests exist — the mapper's "not found" claim was a measurement error (`rg -ln FailLoudly` returns files-with-matches, i.e. 2 *files*, which was misread as 2 *tests*; the second file holds two of them). Verified via `rg -n 'func Test\w*FailLoudly' --glob '*_test.go'`:

- `internal/upgrade/taskfile_shape_test.go:662` — `TestTaskfileShapeHelpersFailLoudly`
- `internal/upgrade/taskfile_shape_test.go:896` — `TestCheckCrossParsersFailLoudly`
- `internal/upgrade/release_workflow_shape_test.go:583` — `TestWorkflowSourceHelpersFailLoudly`

`TestWorkflowSourceHelpersFailLoudly` is the cleanest template for this phase's new guards — a table of `{name string; fn func() error}` cases, each feeding deliberately-malformed input to one parser helper and asserting `err != nil`:

```go
func TestWorkflowSourceHelpersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "parseWorkflowTopLevelName: missing column-0 name: key",
			fn: func() error {
				_, err := parseWorkflowTopLevelName("on:\n  push:\n    branches: [main]\n")
				return err
			},
		},
		{
			name: "parseWorkflowPushTagPatterns: tags: list has zero items",
			fn: func() error {
				_, err := parseWorkflowPushTagPatterns("name: example\non:\n  push:\n    tags:\njobs:\n  build:\n    runs-on: ubuntu-latest\n")
				return err
			},
		},
		// ...
	}
}
```

Every parser helper the wire oracle introduces (transcript framing, placeholder substitution, scenario-manifest parsing) gets an entry in an equivalent table.

### Set-equality, never non-empty
**Source:** `internal/mcp/server_test.go:118-170` (`TestDefaultToolVisibility`, `TestAllowlist`, `TestNoIndexZeroTools`)
**Apply to:** the oracle's `tools/list` scenario assertions (D-05's three `tools/list` variants), VRFY-02/SDK-02 guard assertions
```go
func TestDefaultToolVisibility(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)
	s := BuildServer(true, map[string]bool{}, dir, dir)
	got := listToolNames(t, s)
	want := []string{"codegraph_explore"}
	if !equalStrings(got, want) {
		t.Fatalf("registered tools = %v, want %v", got, want)
	}
}

func TestNoIndexZeroTools(t *testing.T) {
	dir := t.TempDir()
	s := BuildServer(false, map[string]bool{"node": true, "status": true}, dir, dir)
	got := listToolNames(t, s)
	if len(got) != 0 {
		t.Fatalf("registered tools = %v, want none (MCP-03: no .codegraph/ means zero tools)", got)
	}
}
```
Note exact-set (`equalStrings(got, want)`) and exact-zero (`len(got) != 0`) — never `len(got) > 0`. The oracle's own scenario-count assertion (D-07's permanent non-vacuity mechanism) must follow this same exact-count discipline.

### `_IsError` / input-rejection companion test convention
**[CORRECTED by orchestrator 2026-08-04]** The convention **does** exist and is well established — the mapper searched for a literal `*_IsError` *filename* pattern instead of function names. `rg -n 'func Test\w*IsError\w*' --glob '*_test.go'` finds **13** of them, all under `internal/upgrade/`:

`TestSwap_MissingTargetIsError` (`swap_test.go:71`), `TestResolveLatestVersionViaAPI_EmptyTagIsError` (`release_test.go:86`), and eleven in `taskfile_shape_test.go` — `TestRequiredCheckNamesPreserved_ZeroJobsIsError` (:590), `TestToolModfilesRemainIsolated_AbsentModfileIsError` (:647), `TestTaskfileGatesFailLoud_EmptyFileIsError` (:748), `TestTaskfileWrapperIsSerial_MissingWrapperIsError` (:800), `TestCheckCrossMatchesGoreleaserTargets_EmptyBuildsIsError` (:842), `TestCheckCrossMatchesGoreleaserTargets_MissingCheckCrossIsError` (:854), `TestRunBodyExceptionsHaveReasons_EmptyReasonIsError` (:1089), `TestParseWorkflowJobSteps_MissingJobIsError` (:1099), `TestParseWorkflowJobSteps_NoJobsIsError` (:1113), `TestParseWorkflowJobSteps_ZeroStepsIsError` (:1123), `TestContributingReferencesRealTaskTargets_UnknownTargetIsError` (:1275).

**The convention:** `Test<MainTestName>_<WhatIsWrong>IsError` — a companion sitting directly beside the main test, feeding it synthetic input carrying a deliberate defect and asserting the check catches it. `TestContributingReferencesRealTaskTargets_UnknownTargetIsError` is the reference implementation: it builds a synthetic `CONTRIBUTING.md` naming one real target and one fake one, then verifies the real check would flag the fake.

**Why this matters for this phase:** these two conventions are *not* interchangeable. `FailLoudly` proves a **parser helper** rejects malformed input. `_IsError` proves the **main assertion itself** goes red on a planted defect — the non-vacuity proof. D-07's permanent self-defeat guards and every new archtest in this phase need **both**: a `FailLoudly` table for their parsing helpers, and an `_IsError` companion for each top-level guard. Per `95z1dw4vmv`, a helper-level non-vacuity proof is NOT a main-test non-vacuity proof.

### Diagnostic stderr-writer seam
**Source:** `internal/mcp/server.go:63-71` (`WarnUnknownToolsTo`)
**Apply to:** VRFY-03's session line, the VRFY-05 capture shim's own logging (if any)
```go
func WarnUnknownToolsTo(w io.Writer, unknown []string) {
	for _, name := range unknown {
		fmt.Fprintf(w, "codegraph mcp: unknown tool name %q in CODEGRAPH_MCP_TOOLS, ignoring\n", name)
	}
}
```
Explicit `io.Writer` parameter, `"codegraph ..."`-prefixed `fmt.Fprintf` line, never a package-global logger or bare stdout write — this is the entire house convention D-14's format needs to match.

### Bounded-timeout subprocess reads
**Source:** `test/integration/mcp_stdout_purity_test.go:165` (`deadline := time.After(30 * time.Second)`)
**Apply to:** every wire-oracle scenario read loop (Security Domain's DoS-of-test-run mitigation)

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `testdata/wireoracle/*.golden` (frozen transcript format itself) | fixture | batch | No existing byte-exact, placeholder-substituted golden-transcript convention exists in this repo. `testdata/golden/golden_parity_test.go` is the nearest sibling by directory convention but compares typed SDK objects (Trap A), the opposite of what D-04's named-field placeholder substitution over raw bytes requires — do not borrow its comparison mechanism, only its `testdata/` placement convention. |
| VRFY-05 capture/proxy shim (`stdio` tee-then-proxy) | service (one-off measurement tool) | event-driven / streaming | No existing proxying-stdio-shim precedent in this repo; RESEARCH's Architecture Patterns diagram (§ VRFY-05 shim) is the design source of record, not an in-repo analog. Build fresh from that diagram plus the `mcp_stdout_purity_test.go` spawn/pipe mechanics for the plumbing half only. |

**[CORRECTED by orchestrator 2026-08-04]** Two rows were removed from this table. The mapper listed `_IsError`-named companion tests and `TestWorkflowSourceHelpersFailLoudly` as "no analog found"; both claims were false. Both conventions exist and are documented with verified file:line citations under Shared Patterns above. Only the two rows retained here are genuine gaps.

## Metadata

**Analog search scope:** `test/integration/`, `test/wireoracle/` (does not yet exist), `internal/mcp/`, `internal/cli/`, `internal/graphstore/archtest/`, `internal/cli/present/archtest/`, `internal/upgrade/`, `docs/`, `internal/indexer/testdata/gofixture/`
**Files scanned:** 14 read in full or targeted excerpt this session (`mcp_stdout_purity_test.go`, `main_test.go`, `import_graph_test.go` ×2, `stdout_confinement_test.go`, `stdout_closure_selftest_test.go`, `server.go`, `serve.go` (2 excerpts), `server_test.go` (excerpt), `taskfile_shape_test.go` (excerpt), `FLAG-PARITY.md` headings, `gofixture/` tree listing), plus 2 `rg` searches (`FailLoudly`, `mark3labs` in `serve.go`)
**Pattern extraction date:** 2026-08-04
