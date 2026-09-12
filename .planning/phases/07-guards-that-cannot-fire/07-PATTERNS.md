# Phase 7: Guards That Cannot Fire - Pattern Map

**Mapped:** 2026-09-08
**Files analyzed:** 6 (new/modified)
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|-----------------|---------------|
| `internal/query/archtest/<name>_test.go` (new, GRD-02) | test (architecture/boundary) | request-response (static analysis over `go/packages` graph) | `internal/graphstore/archtest/import_graph_test.go` | exact |
| `internal/bench/regression.go` (modified, GRD-01) | utility (validation function) | transform (metrics in → error/nil out) | itself, lines 107-111 (the two baseline positivity checks immediately above the insertion point) | exact (self-mirroring) |
| `internal/bench/regression_test.go` (modified, GRD-01) | test (table-driven) | transform | itself, existing table rows (`errHint` substring pattern) | exact (self-extending) |
| `scripts/<inject-cosign-key>.sh` (new, GRD-03) | utility (shell script, CI-invoked) | file-I/O (reads `.goreleaser.yaml`, writes generated config, diffs) | `scripts/pr_template_policy.py` (script-under-Task-target shape) + the two inline Taskfile blocks being extracted (`Taskfile.yml` `release:dry-run-signed` ~1800-1863, `release:rehearse-notarize` ~2377-2394) | role-match (no existing `scripts/*.sh` analog; python script gives the CLI/env-var/exit-code convention, the Taskfile blocks give the actual logic to lift verbatim) |
| `Taskfile.yml` (modified, GRD-03) | config (Task target body) | request-response (invokes external script, checks exit code) | itself — the `Install Task` / `task verify:*` invocation style already used throughout the file (e.g. `verify:release-assets`, `verify:gatekeeper` callers in `post-release-verify.yml`) | exact (self-mirroring pattern of "Task target shells out to one script") |
| `internal/upgrade/release_workflow_shape_test.go` (modified, GRD-04 add + GRD-05 delete) | test (parsed-YAML shape assertion) | request-response (parse workflow doc, assert per-job invariant) | `TestPostReleaseJobsDeclareCheckoutPolicy` (same file, ~1582-1631) and its `postReleaseCheckoutShapes` helper (~1306-1322); `TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError` (~1277-1288) for the empty-doc companion | exact (same file, same doc, same job map — just a different field) |
| `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` (new, GRD-06) | test (evidence artifact, not code) | batch (one entry per guard family, sequential) | `.planning/milestones/v0.11.0-phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md` | exact |

## Pattern Assignments

### `internal/query/archtest/<name>_test.go` (test, GRD-02)

**Analog:** `internal/graphstore/archtest/import_graph_test.go` (entire file, 90 lines — read in full, one pass)

**Package doc + constants pattern** (lines 1-19):
```go
// Package archtest enforces D-04a: no package outside internal/graphstore
// (and its own subpackages) may import the embedded key-value engine
// directly. ...
package archtest

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const (
	pebbleImportPath      = "github.com/cockroachdb/pebble/v2"
	allowedImporterPrefix = "github.com/seanb4t/codegraph-go/internal/graphstore"
)
```
For GRD-02: rename the package doc to name threat T-01-18 (D-04), define the forbidden set as the **union** the CONTEXT.md D-02 requires (`internal/uiserver`, `internal/mcp`, `internal/uiproto`, `connectrpc.com/connect`) plus the allow-listed leaves (`internal/indexer/goextract`, `internal/indexer/nodeid` per D-01), and the target package prefix `internal/query`.

**Loader config pattern** (lines 30-38):
```go
cfg := &packages.Config{
	Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps,
	// Tests: true is required so an import statement that appears only
	// inside a _test.go file ... is not invisible to this check.
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
GRD-02 must check `pkg.Imports` (transitively resolved via `NeedDeps`/`pkg.Types.Imports()` or by walking `pkg.Imports` recursively) — D-02 requires the **resolved transitive dependency set**, not direct imports, so this loop needs one more level than the pebble analog (which only checks direct `pkg.Imports[pebbleImportPath]`). Consider using `packages.Visit` or a recursive walk over `pkg.Imports` to build the transitive set per `internal/query`-rooted package, since the analog's single-hop check is insufficient for D-02's "smuggled through an intermediary" requirement.

**Positive-control pattern** (lines 60-68, D-03's analog):
```go
foundGraphstoreImporter := false
for _, pkg := range pkgs {
	if _, imports := pkg.Imports[pebbleImportPath]; !imports {
		continue
	}
	...
	foundGraphstoreImporter = true
}
if !foundGraphstoreImporter {
	t.Fatal("no package under internal/graphstore was found importing pebble/v2 — this test cannot verify enforcement; ...")
}
```
D-03 wants **two** positive assertions (graphstore present AND goextract present in the resolved set) plus a "packages loaded > 0" count report — extend this shape to assert both, and print `len(pkgs)`.

**`isAllowedImporter` + `stripTestVariant` helpers** (lines 71-90) — copy verbatim, only the constant set changes:
```go
func isAllowedImporter(pkgPath string) bool {
	base := stripTestVariant(pkgPath)
	if base == allowedImporterPrefix {
		return true
	}
	return len(base) > len(allowedImporterPrefix) && base[:len(allowedImporterPrefix)+1] == allowedImporterPrefix+"/"
}

func stripTestVariant(pkgPath string) string {
	if i := strings.IndexByte(pkgPath, ' '); i >= 0 {
		pkgPath = pkgPath[:i]
	}
	pkgPath = strings.TrimSuffix(pkgPath, "_test")
	pkgPath = strings.TrimSuffix(pkgPath, ".test")
	return pkgPath
}
```
GRD-02's shape is inverted from the pebble test (pebble test = allow-list of importers of a fixed target; GRD-02 = forbid `internal/query` from importing a fixed set of forbidden packages, with two allow-listed exceptions) — `stripTestVariant` still applies to normalize package paths on both sides of the comparison.

---

### `internal/bench/regression.go` (utility, GRD-01)

**Analog:** itself — `internal/bench/regression.go` lines 107-111 (already in context above), the two baseline positivity checks D-10 says the two new lines mirror exactly, inserted immediately after them and before the delta math (line 114 onward).

**Existing baseline positivity pattern to mirror:**
```go
if baseline.FilesPerSec <= 0 {
	return fmt.Errorf("bench: invalid baseline: FilesPerSec must be positive, got %.4f", baseline.FilesPerSec)
}
if baseline.PeakRSSBytes <= 0 {
	return fmt.Errorf("bench: invalid baseline: PeakRSSBytes must be positive, got %d", baseline.PeakRSSBytes)
}
```
D-10's two new lines (current-metrics positivity), same shape, `current` in place of `baseline`, inserted at this exact point:
```go
if current.FilesPerSec <= 0 {
	return fmt.Errorf("bench: invalid current: FilesPerSec must be positive, got %.4f", current.FilesPerSec)
}
if current.PeakRSSBytes <= 0 {
	return fmt.Errorf("bench: invalid current: PeakRSSBytes must be positive, got %d", current.PeakRSSBytes)
}
```
Function-level doc comment (lines 17-33) already documents "CheckRegression NEVER... panics: a degenerate baseline... returns a plain error" — extend this sentence to cover degenerate *current* metrics too, since D-01/backlog 999.4 is specifically about `current.PeakRSSBytes = 0` passing through undetected today.

**Error-message shape convention:** `"bench: invalid <side>: <Field> must be positive, got <value>"` — `%.4f` for `FilesPerSec` (float), `%d` for `PeakRSSBytes` (int64). Keep identical verbs/nouns, only swap `baseline`→`current`.

---

### `internal/bench/regression_test.go` (test, GRD-01)

**Analog:** itself — the existing table shape (lines 8-22 struct, rows 23-.. through line ~520 assertion loop, already in context above).

**Table row shape to add two cases into (same `tests := []struct{...}` slice, same `errHint` convention):**
```go
{
	name: "zero baseline throughput yields a clear error, not a panic",
	baseline: Metrics{
		FilesPerSec:  0,
		PeakRSSBytes: 500_000_000,
	},
	current: Metrics{
		FilesPerSec:  50.0,
		PeakRSSBytes: 500_000_000,
	},
	ceiling: ceiling,
	wantErr: true,
	errHint: "baseline",
},
```
D-10's RED replay: **exact historical frame** — `CheckRegression(baseline, current, ceiling=1)` with `current.PeakRSSBytes = 0`, otherwise-matching frame, `errHint: "current"` (or a substring naming the field, matching whatever wording D-10's lines actually emit — align errHint to the literal string chosen in regression.go). Companion case: `current.FilesPerSec = 0`. Use `ceiling: 1` for the primary replay case specifically because D-10 says this is the historical Phase 10 audit frame, not a fresh synthetic one — do not reuse the file's usual `ceiling` constant (`1_000_000_000`) for that specific case.

**Assertion loop convention** (already in context, lines 519-521):
```go
if tt.wantErr && tt.errHint != "" {
	if !containsFold(err.Error(), tt.errHint) {
		t.Errorf("error %q does not mention expected hint %q", err.Error(), tt.errHint)
```
No changes needed here — new rows slot into the existing loop for free.

---

### `scripts/<inject-cosign-key>.sh` (new, GRD-03)

**Analogs:**
1. The two Taskfile blocks being extracted verbatim — `Taskfile.yml` `release:dry-run-signed` cmds block (~1793-1863) and `release:rehearse-notarize`'s block (~2365-2394). These ARE the logic to lift; the script is a straight extraction, not a rewrite.
2. `scripts/pr_template_policy.py` for this repo's script-under-Task-target conventions (env-var input, `$GITHUB_OUTPUT`-style contract, docstring naming its own non-obviousness) — note this is Python; GRD-03's script will be POSIX shell given the awk/diff/grep pipeline it's extracted from, but the "read from args/env, write one verdict, exit status is meaningful" shape carries over.

**Core awk-injection pattern to extract (from `release:dry-run-signed`, ~1836-1841; identical block repeats at `release:rehearse-notarize` ~2377-2381):**
```bash
GENERATED_CONFIG="${SIGNTEST_DIR}/.goreleaser.signtest.yaml"
awk -v keyline="      - \"--key=${SIGNTEST_DIR}/cosign.key\"" '
  { print }
  /^      - "sign-blob"$/ { print keyline }
' .goreleaser.yaml > "${GENERATED_CONFIG}"
```

**Additions-only diff guard to extract (~1843-1863):**
```bash
DIFF_OUT="$(diff .goreleaser.yaml "${GENERATED_CONFIG}" || true)"
if printf '%s\n' "${DIFF_OUT}" | grep -q '^<'; then
  echo "::error::${GENERATED_CONFIG} differs from the committed .goreleaser.yaml by more than additions:"
  printf '%s\n' "${DIFF_OUT}"
  exit 1
fi
BAD_ADDITIONS="$(printf '%s\n' "${DIFF_OUT}" | grep '^> ' | grep -v -- '--key=' | grep -v 'COSIGN_PASSWORD' || true)"
if [ -n "${BAD_ADDITIONS}" ]; then
  echo "::error::${GENERATED_CONFIG} has additions not matching --key= or COSIGN_PASSWORD:"
  printf '%s\n' "${BAD_ADDITIONS}"
  exit 1
fi
echo "generated config differs from the committed .goreleaser.yaml by additions only:"
printf '%s\n' "${DIFF_OUT}"
```

**D-06's new positive assertion (not yet in either Taskfile block — must be added during extraction, not just copied):** count `--key=` lines **added** in the diff and require exactly 1:
```bash
KEY_LINE_COUNT="$(printf '%s\n' "${DIFF_OUT}" | grep -c '^> .*--key=' || true)"
echo "injected --key= lines: ${KEY_LINE_COUNT}"
if [ "${KEY_LINE_COUNT}" != "1" ]; then
  echo "::error::expected exactly 1 injected --key= line, found ${KEY_LINE_COUNT} — the sign-blob anchor may no longer match (0) or a duplicated sign-blob block exists (2+)"
  exit 1
fi
```

**Interface (Claude's discretion per D-05):** script takes committed config path, output path, and key path as positional args (matching the three values `SIGNTEST_DIR`/`REHEARSE_DIR`-scoped variables currently supply: `.goreleaser.yaml`, `${DIR}/.goreleaser.<x>.yaml`, `${DIR}/cosign.key`). Both call sites already compute an equivalent run-scoped tmpdir (`SIGNTEST_DIR`, `REHEARSE_DIR`) before this block — the script receives paths, not directories, and does not itself manage the tmpdir/cleanup trap (that logic differs between the two callers and stays in Task).

**RED demonstration:** run the script against a copy of `.goreleaser.yaml` with the `sign-blob` block re-indented — the `/^      - "sign-blob"$/` anchor (exact 6-space indent, exact quoting) is what breaks; a re-indented copy makes `KEY_LINE_COUNT` come back 0.

---

### `Taskfile.yml` (modified, GRD-03)

**Analog:** itself — both target bodies being replaced (see above); after extraction each cmds: block becomes a single script invocation. The convention for "Task target shells out to one script/binary and checks output" is already established throughout this file (e.g. `task verify:release-assets`, `task verify:gatekeeper` as called from `post-release-verify.yml`, and `Install Task` / `./.github/actions/install-task` step patterns) — no new Task-authoring convention needed, just replace the inline `awk`/`diff`/`grep` block with:
```yaml
      - |
        set -euo pipefail
        ./scripts/<name>.sh "${SIGNTEST_DIR}/.goreleaser.yaml" "${SIGNTEST_DIR}/.goreleaser.signtest.yaml" "${SIGNTEST_DIR}/cosign.key"
```
(paths/varnames illustrative; keep each call site's own tmpdir variable names — `SIGNTEST_DIR` vs `REHEARSE_DIR` — unchanged, only the injected block is replaced by the script call). Per D-07, do **not** add a `taskfile_shape_test.go` assertion that both targets call the script.

---

### `internal/upgrade/release_workflow_shape_test.go` (modified: GRD-04 add, GRD-05 delete)

**Analog (parse-for-free, D-08):** `decodeFullWorkflowDoc` + `fullWorkflowDoc`/`fullWorkflowJob` structs (lines 1032-1069, already in context) — reused as-is. **Gap to close:** `fullWorkflowJob` currently has no `If` field:
```go
type fullWorkflowJob struct {
	Env         map[string]any     `yaml:"env"`
	Permissions map[string]any     `yaml:"permissions"`
	Steps       []fullWorkflowStep `yaml:"steps"`
}
```
GRD-04 needs `If string \`yaml:"if"\`` added to this struct (a job-level addition shared by all consumers of `fullWorkflowDoc` — safe since Go zero-values the field for jobs/files that omit `if:`).

**Job-map traversal + per-job invariant pattern to copy (analog `TestPostReleaseJobsDeclareCheckoutPolicy`, lines 1582-1631, already in context):**
```go
func TestPostReleaseJobsDeclareCheckoutPolicy(t *testing.T) {
	doc, err := decodeFullWorkflowDoc(postReleaseWorkflowPath)
	if err != nil {
		t.Fatalf("decodeFullWorkflowDoc(%s): %v", postReleaseWorkflowPath, err)
	}
	shapes := postReleaseCheckoutShapes(doc)
	if len(shapes) == 0 {
		t.Fatalf("%s declares zero jobs", postReleaseWorkflowPath)
	}
	...
}
```
GRD-04's version (per D-08): iterate `doc.Jobs` directly (no separate shape-slice type needed unless the planner prefers symmetry with `postReleaseCheckoutShapes`), require `len(doc.Jobs) > 0`, and for every job assert `job.If == conclusionGuardDisjunct` exactly:
```go
const conclusionGuardDisjunct = "github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'"

func TestPostReleaseJobsDeclareConclusionGuard(t *testing.T) {
	doc, err := decodeFullWorkflowDoc(postReleaseWorkflowPath)
	if err != nil {
		t.Fatalf("decodeFullWorkflowDoc(%s): %v", postReleaseWorkflowPath, err)
	}
	if len(doc.Jobs) == 0 {
		t.Fatalf("%s declares zero jobs", postReleaseWorkflowPath)
	}
	inspected := 0
	for jobID, job := range doc.Jobs {
		inspected++
		if job.If != conclusionGuardDisjunct {
			t.Errorf("job %q if: = %q, want exactly %q", jobID, job.If, conclusionGuardDisjunct)
		}
	}
	t.Logf("inspected %d job(s)", inspected)
}
```
Reuse `postReleaseWorkflowPath` const (line 25, already defined) and confirm all 5 jobs in `.github/workflows/post-release-verify.yml` (`resolve-tag`, `verify-supply-chain`, `self-upgrade`, `gatekeeper`, `notarized-suite`) carry the disjunct verbatim on their `if:` line — verified present on all 5 in the current file.

**Empty-doc companion pattern to copy (analog `TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError`, lines 1277-1288, already in context):**
```go
func TestPostReleaseJobsDeclareConclusionGuard_EmptyDocIsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yml")
	if err := os.WriteFile(path, []byte("name: empty\non:\n  push:\njobs: {}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	if _, err := decodeFullWorkflowDoc(path); err == nil {
		t.Fatalf("decodeFullWorkflowDoc(%s): expected a non-nil error for a workflow with zero jobs:, got nil", path)
	}
}
```
This companion is likely **shared/redundant** with the existing `TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError` and `TestPostReleaseJobsDeclareCheckoutPolicy`'s own zero-jobs guard (`decodeFullWorkflowDoc` itself already errors on zero jobs — this is a property of the shared helper, not per-test) — Claude's discretion whether GRD-04 needs its own copy or can rely on the shared helper's existing coverage; D-08 doesn't mandate a fresh copy, only "an empty-or-unparseable-document-is-error companion."

**Deletion target (D-09):** `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` (lines 1560-1573, already in context) — delete the whole function. Do not touch `homebrewTapCredentialNames` (line 1366, used elsewhere) or `releasePleaseAppSecretNames` (local to the deleted function, safe to remove with it).

---

## Shared Patterns

### `go/packages`-based archtest loader (D-02, D-03, D-04)
**Source:** `internal/graphstore/archtest/import_graph_test.go` lines 30-46, 60-68
**Apply to:** `internal/query/archtest/<name>_test.go` — the `packages.Config{Mode: NeedImports|NeedName|NeedDeps, Tests: true}` + zero-package fatal + positive-control-with-count pattern is the load-bearing shape for any future archtest in this repo; do not regex source.

### Workflow-doc parse-once, assert-many (D-08)
**Source:** `internal/upgrade/release_workflow_shape_test.go` — `decodeFullWorkflowDoc`/`fullWorkflowDoc` (lines 1032-1069)
**Apply to:** GRD-04's new test and any future `post-release-verify.yml` assertion — never a line-scanner/regex over the YAML; every negative assertion pairs with an "empty doc is error" twin (already satisfied by the shared helper's own zero-jobs check).

### Mutation-log structure (D-11, GRD-06)
**Source:** `.planning/milestones/v0.11.0-phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md`
**Apply to:** `07-MUTATION-LOG.md` — per-entry shape: pre-mutation cleanliness gate (`git diff --quiet -- <file>` → exit 0), mutation applied (exact diff/edit described), pasted failing test output (verbatim `go test`/script output, not paraphrased), revert command (`git checkout -- <file>`), byte-clean proof (`git diff --stat` empty), green re-run output. Four full entries (GRD-01..04); GRD-05 gets a one-line deletion record, not a demonstration, per D-11.

```markdown
## Family (a) — <guard name>

**Test name:** `<TestName>` — <what it proves>

**Pre-mutation gate:** `git diff --quiet -- <file>` → exit 0 (clean).

**Mutation applied:** <exact change>

**Observed failure (pasted, `<exact command>`):**
```
<verbatim failing output>
```

**Revert:** `git checkout -- <file>`.

**Byte-clean proof:** `git diff --stat <file>` empty after revert; `<command>` → `ok` (green).
```

## No Analog Found

None — all 6 files/changes have a strong same-repo, same-shape analog. `scripts/<name>.sh` has no prior POSIX-shell script under `scripts/` (only `.py`, `.go`, and one `.sh` — `live-push-concurrency-check.sh`, not read here since the Taskfile blocks being extracted are themselves the primary source), but the extraction source (the two Taskfile blocks) is stronger than any general script-style analog would be, since it's a verbatim lift, not a new design.

## Metadata

**Analog search scope:** `internal/graphstore/archtest/`, `internal/upgrade/`, `internal/bench/`, `scripts/`, `Taskfile.yml`, `.github/workflows/post-release-verify.yml`, `.planning/milestones/v0.11.0-phases/03-non-vacuity-proof-unconditional-ci-execution/`
**Files scanned:** 8 read directly (all git-tracked, confirmed via `git ls-files`)
**Pattern extraction date:** 2026-09-08
</content>
</invoke>
