# Phase 3: Homebrew Tap & Cask - Pattern Map

**Mapped:** 2026-08-09
**Files analyzed:** 9 (2 created Go, 5 modified config/CI/docs, 2 new/extended test files)
**Analogs found:** 9 / 9

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/cli/man.go` | controller (Cobra command) | request-response (one-shot generate-to-dir) | `internal/cli/githooks.go` | exact — same shape: Go-only divergence command, hidden vs visible differs but registration/doc-comment convention is identical |
| `internal/cli/man_test.go` | test | request-response | `internal/cli/githooks_test.go` (if present) else general `internal/cli/*_test.go` convention | role-match |
| `internal/cli/root.go` (modified) | config/wiring | request-response | itself (existing `AddCommand` list) | exact — in-place edit, not a new file |
| `.goreleaser.yaml` `homebrew_casks:` block (modified) | config | batch/CRUD (declarative release-time generation) | existing blocks in same file: `archives:`, `checksum:`, `notarize:`, `signs:`, `sboms:`, `release:` | exact — same file, same comment convention, same shape-test-backed pattern |
| `internal/upgrade/goreleaser_shape_test.go` (new tests appended) | test (shape test) | transform (parse YAML → assert property) | `TestRawArchiveEntryStaysBinaryFormat`, `TestChecksumCoversRawAndZipIdsOnly`, `TestReleaseBlockIsRerunIdempotent`, `TestReleaseBlockDoesNotRewriteReleaseBody` (same file) | exact |
| `.github/workflows/release.yml` (modified — new job to mint tap App token) | config/CI | event-driven (workflow job) | `.github/workflows/release-please.yml`'s existing App-token-mint step | exact — same action, same pattern, different App/secrets |
| `internal/upgrade/release_workflow_shape_test.go` (new tests appended) | test (shape test) | transform | `TestOIDCWriteScopedToSingleGoreleaserJob`, `TestAppleSecretsScopedToSingleReleaseJob` (same file) | exact |
| `Taskfile.yml` (modified — new verification target(s)) | config (task runner) | batch | existing `if [ ... ]; then ... fi` guarded targets (e.g. lines 469-493, 631-712) | exact |
| `internal/upgrade/taskfile_shape_test.go` (possibly extended) | test (shape test) | transform | `TestWorkflowRunBodiesInvokeTask` (same file) | exact |
| `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `README.md`, `docs/FLAG-PARITY.md` (modified) | docs | — | existing docs' own conventions; `docs/FLAG-PARITY.md`'s Go-only-divergence table entry for `githooks` | exact for FLAG-PARITY; role-match for the rest |
| `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` (amended) | planning artifact | — | N/A — value edits only, no new structure (see `planning-artifacts` rule) | n/a |

## Pattern Assignments

### `internal/cli/man.go` (controller, request-response)

**Analog:** `internal/cli/githooks.go` (full file read, 127 lines)

**Imports pattern** (githooks.go lines 1-10):
```go
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/githooks"
)
```
For `man.go`, the equivalent import is `github.com/spf13/cobra/doc` in place of an internal package — `man` has no internal package to delegate to; `doc.GenManTree` is called directly against `newRootCmd()`.

**Doc-comment divergence-recording convention** (githooks.go lines 12-17):
```go
// newGithooksCmd builds the `codegraph githooks` command tree: install /
// remove / status, each taking an optional [path] resolved via the
// package-level targetRoot (D-11), matching init/uninit/sync. This is a
// Go-only surface extension over internal/githooks — TS 1.3.1 has no
// standalone githooks command, only init/uninit call the equivalent
// functions internally (D-01).
```
`man.go`'s constructor comment must name D-01/D-02 the same way, and additionally explain why it is `Hidden: true` (githooks is NOT hidden — this is the one place the analog diverges and must not be copied literally). Cross-reference `docs/FLAG-PARITY.md`'s existing `githooks` entry for the divergence-table format to extend for `man`.

**Command construction shape** (githooks.go lines 18-25, adapted per RESEARCH.md's own sketch at lines 464-478):
```go
func newGithooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "githooks",
		Short: "Manage git sync hooks (post-commit/post-merge/post-checkout)",
	}
	cmd.AddCommand(newGithooksInstallCmd(), newGithooksRemoveCmd(), newGithooksStatusCmd())
	return cmd
}
```
`man.go`'s top-level constructor has no subcommands (it is a single leaf command, per RESEARCH.md's `newManCmd()` sketch), so it follows the *inner* leaf-command shape below rather than the tree-builder shape.

**Leaf command shape with RunE + Args** (githooks.go lines 27-54, `newGithooksInstallCmd`):
```go
func newGithooksInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [path]",
		Short: "Install marker-fenced git sync hooks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}
			result := githooks.Install(cmd.Context(), root)
			...
			return nil
		},
	}
}
```
`man.go` mirrors this exactly (`Args: cobra.ExactArgs(1)` per RESEARCH.md, RunE calling `doc.GenManTree`), plus `Hidden: true`. Note `githooks` subcommands take an *optional* `[path]` via `targetRoot(args)`; `man` takes a *required* output directory — do not reuse `targetRoot` for it, it resolves the codegraph project root, not an arbitrary output dir.

**Error handling / output convention** (githooks.go lines 43-51):
```go
printHookErrors(cmd, result.Errors)
if len(result.Installed) == 0 {
	fmt.Fprintln(cmd.OutOrStdout(), "Could not install git hooks. Run `codegraph sync` after changes instead.")
	return nil
}
fmt.Fprintf(cmd.OutOrStdout(), "Installed git %s hooks — the index refreshes in the background after each.\n",
	strings.Join(result.Installed, ", "))
```
Errors are written to `cmd.ErrOrStderr()`, not `os.Stderr` directly (see `printHookErrors`, lines 89-93) — always route through the `cmd` writers so tests can capture output. `man.go`'s `RunE` should return the `doc.GenManTree` error directly (wrapped with context) rather than printing-and-returning-nil, since a hidden command has no interactive user to address with a friendly message — this is a deliberate divergence from githooks' shape, not an oversight.

---

### `internal/cli/root.go` (config/wiring, modified in place)

**Analog:** itself — `newRootCmd()`, lines 34-53.

**Registration pattern** (root.go lines 45-51):
```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
	newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
	newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
	newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd(),
	newDaemonCmd(), newUnlockCmd(), newVersionCmd(), newTelemetryCmd(),
	newUpgradeCmd(), newInstallCmd(), newUninstallCmd(), newMigrateCmd(),
	newGithooksCmd())
```
Add `newManCmd()` to this list (last entry, matching githooks' own append-at-tail precedent when it was added). Also update the package doc comment (lines 1-9) and the `newRootCmd` doc comment (lines 26-33) to name `man` alongside `migrate`/`githooks` as a Go-only surface extension, following the existing pattern of naming each addition's decision id inline (`(D-01)`, `(D-01/D-05)`, `(D-08)`).

---

### `.goreleaser.yaml` `homebrew_casks:` block (config, new top-level block)

**Analog:** the file's own `archives:`, `checksum:`, `notarize:`, `signs:`, `sboms:`, `release:` blocks — read directly (lines 130-400+).

**The file's rationale-density convention** — every block names its decision id and its holding test inline, e.g.:
```yaml
# D-12: scoped to exactly the 8 downloadable payloads (4 raw binaries +
# 4 zips). ... Held by TestChecksumCoversRawAndZipIdsOnly.
checksum:
  ...
  ids: [raw, zip]
```
and:
```yaml
# `documents:` is NAME-derived (`{{ .ArtifactName }}`), NOT PATH-derived
# (`${artifact}`) — this is cycle-3 review HIGH-B and is a correctness
# requirement, not a style choice. ... Held by TestSbomsArePerBinaryWithSpdxNames,
# which RESOLVES this template ... and asserts the results are four
# DISTINCT strings, not a literal string match.
sboms:
  ...
```
`homebrew_casks:` must match this density: name D-14/D-15/D-16 for the tap/credential shape, name the `ids: [zip]` requirement and cite `ErrMultipleArchivesSameOS` (RESEARCH.md Pattern 1) the same way `checksum:`'s comment cites its own held-by test, and name D-05/D-16's "nothing added to the zip" constraint the same way the `zip` archive entry's own comment (lines 158-168) already states it from the other side. Use RESEARCH.md's draft block (Code Examples section) as the field-shape source — it is already traced from the pinned module's own config struct and golden fixtures, not guessed.

**Path-vs-Name discipline (four-times-recorded defect class in this repo — critical to carry forward):** leave `url.template` unset entirely (RESEARCH.md Pattern 2); if ever set, use `{{ .ArtifactName }}`, never `${artifact}`. This is the same discipline the `sboms:.documents` comment above documents, and the same the `signs:` block's `${artifact}` usage explains is safe only because `signs:` intentionally IS path-derived (a different, deliberate case) — do not copy `signs:`'s `${artifact}` idiom into `homebrew_casks:`.

---

### `internal/upgrade/goreleaser_shape_test.go` (new shape tests, appended to existing file)

**Analog (same file):** `TestRawArchiveEntryStaysBinaryFormat`, `TestChecksumCoversRawAndZipIdsOnly`, `TestReleaseBlockIsRerunIdempotent`, `TestReleaseBlockDoesNotRewriteReleaseBody`.

**Doc-comment convention — cite the requirement/decision id, state exactly what fails and why** (`TestRawArchiveEntryStaysBinaryFormat`, lines 276-281):
```go
// TestRawArchiveEntryStaysBinaryFormat holds REL-09's success criterion 4 as
// a machine check: the archives: entry with id: raw declares formats:
// containing exactly the single value "binary". Fails if the entry is
// absent, if formats: names anything other than binary, or if it names more
// than one format.
func TestRawArchiveEntryStaysBinaryFormat(t *testing.T) {
```

**Core parse-then-assert-a-property pattern** (`TestChecksumCoversRawAndZipIdsOnly`, lines 336-358):
```go
func TestChecksumCoversRawAndZipIdsOnly(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	block := mustGoreleaserTopLevelBlock(t, src, "checksum")
	rawIDs, ok := block["ids"]
	if !ok {
		t.Fatalf("checksum: block has no ids: key")
	}
	ids, err := toStringSlice(rawIDs)
	if err != nil {
		t.Fatalf("checksum.ids: %v", err)
	}
	want := sortedJoin([]string{"raw", "zip"})
	if got := sortedJoin(ids); got != want {
		t.Errorf("checksum.ids = %v, want exactly [raw zip]", ids)
	}
}
```
Helpers to reuse verbatim, already present in the file: `mustGoreleaserTopLevelBlock(t, src, key string) map[string]any` (line 247), `toStringSlice(v any) ([]string, error)` (line 260), `sortedJoin` — this is the **property-asserting** idiom (exact-set comparison via sorted-join), not a literal-string comparison. RESEARCH.md's Wave 0 Gaps section names the three assertions the new test(s) need: `ids: [zip]` as an exact set (not `raw`, not both), `url.template` absent (not merely empty), and `generate_completions_from_executable.shells` as the exact 3-element set `[bash, zsh, fish]`. Write each as its own `Test...` function mirroring the shape above, one property per test, following this file's existing one-property-per-function granularity.

**PITFALL — the `TestSbomsArePerBinaryWithSpdxNames` failure mode named in this repo's own history:** a shape test that pins a broken template *resists correction* because it looks green. The `sboms:` comment block (`.goreleaser.yaml`) explicitly documents that its held-by test "RESOLVES this template for all four platforms and asserts the results are four DISTINCT strings, not a literal string match" — i.e., it asserts a *property* (distinctness across resolved values), not a literal string. Any new `homebrew_casks:` shape test that needs to check a templated value (if one is added beyond the static ones already listed) must follow the same resolve-and-assert-property discipline, never pin a literal rendered string.

---

### `internal/upgrade/release_workflow_shape_test.go` (new shape test, appended)

**Analog (same file):** `TestOIDCWriteScopedToSingleGoreleaserJob` (lines 608-656), `TestAppleSecretsScopedToSingleReleaseJob` (lines 1171-1298).

**Direct analog for a new-App-secrets-scoping test:** `TestAppleSecretsScopedToSingleReleaseJob`'s three invariants (lines 1176-1195) are the template for a `TestHomebrewTapSecretsScopedTo...Job` test:
- every reference to the two new secret names (e.g. `HOMEBREW_TAP_APP_ID`/`HOMEBREW_TAP_APP_PRIVATE_KEY`) appears ONLY in the job that mints the tap token — a **separate, `id-token`-free job** per RESEARCH.md's Open Question 2 recommendation, distinct from the `TestOIDCWriteScopedToSingleGoreleaserJob`-guarded goreleaser job;
- every such reference is under a STEP-level `env:`, never job- or workflow-level;
- no workflow with `pull_request`/`pull_request_target` triggers references the names or declares `id-token: write`.

This test also independently re-verifies `id-token: write` is still held by exactly one job (line 1191's comment: "so this change cannot have widened T-01-11/D-11's existing invariant... independently of `TestOIDCWriteScopedToSingleGoreleaserJob`") — the new test for the tap credential should do the same independent re-check, not rely on the existing test alone.

**Runtime workflow-directory enumeration (never a fixture list)** (lines 1206-1222):
```go
func TestAppleSecretsScopedToSingleReleaseJob(t *testing.T) {
	entries, err := os.ReadDir(workflowsDir)
	...
	if len(files) == 0 {
		t.Fatalf("workflow directory scan found zero files in %s — this would make the whole test vacuous", workflowsDir)
	}
	...
}
```
Reuse `workflowsDir`, `decodeFullWorkflowDoc`, `mustGoreleaserInvokingJob`, `releaseWorkflowPath` — all already defined in this file/package; do not reimplement.

**Non-vacuity companion pattern** (lines 1286-1298, `TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError`): every new parser/helper this phase adds needs a companion test proving it errors on empty/degenerate input rather than silently returning a usable zero value — same discipline as `TestParseGoreleaserArchives_NoArchivesBlockIsError` in `goreleaser_shape_test.go`.

---

### `.github/workflows/release.yml` (modified — new job minting the tap App token)

**Analog:** `.github/workflows/release-please.yml` lines 72-92 (verbatim, already working App-token-mint pattern):
```yaml
      - name: Mint GitHub App installation token
        id: app-token
        # actions/create-github-app-token's current required input is
        # client-id (App's Client ID); app-id (the numeric App ID) is kept
        # only as a deprecated back-compat alias. Both work; app-id is used
        # here to match D-02's documented secret names (APP_ID,
        # APP_PRIVATE_KEY) — migrating to client-id later requires
        # re-seeding the secret with the App's Client ID, not its App ID.
        uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}

      - name: Run release-please
        id: release
        uses: googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0
        with:
          token: ${{ steps.app-token.outputs.token }}
```
Mirror this exactly for the tap: same pinned action/SHA (`bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0`), but with **new**, distinctly-named secrets (`HOMEBREW_TAP_APP_ID`/`HOMEBREW_TAP_APP_PRIVATE_KEY` or similar — never `secrets.APP_ID`/`secrets.APP_PRIVATE_KEY`, which belong to the release-please App installed on `codegraph-go` itself — Pitfall 2 in RESEARCH.md). Per RESEARCH.md's Open Question 2 recommendation and the `TestOIDCWriteScopedToSingleGoreleaserJob` guard's actual constraint (only forbids a *second holder* of `id-token: write`, not a second job), place this mint step in a **new job with no `id-token: write`**, and pass the token to the goreleaser job via `needs.<job>.outputs.<name>`. Then feed it into `.goreleaser.yaml`'s `homebrew_casks.repository.token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"` as a step-level `env:` on the goreleaser step — following the same step-level-env scoping `TestAppleSecretsScopedToSingleReleaseJob` already enforces for the Apple secrets.

---

### `Taskfile.yml` (modified — any new verification target(s))

**Analog:** existing `if [ ... ]; then ... fi` blocks, e.g. lines 469-493 and 631-712 (both read this session via grep, full context not re-read here — pattern is consistent across all matches).

**PITFALL — do not use a trailing `&&`-chained list as the last statement inside a loop.** This repo's own recorded failure mode: under go-task's `set -e` execution, a trailing `&&` chain as a loop's last statement aborts silently on the success branch, whereas plain `sh` would exit 0 in all three positions — making the bug invisible outside `task`. Always use an explicit `if`/`fi` conditional, matching the existing idiom at (for example) Taskfile.yml:469-475:
```sh
if [ ! -f dist/artifacts.json ]; then
  echo "..." >&2
  exit 1
fi
```
not a `cmd1 && cmd2 && cmd3` chain relied on for control flow.

**Enforcement lives in Taskfile, not the workflow inline** — confirmed by `TestWorkflowRunBodiesInvokeTask` (`internal/upgrade/taskfile_shape_test.go` line 1342+), which asserts every in-scope workflow step's run body actually invokes a `task` target rather than embedding logic inline. Any new BREW-0x verification (e.g. `task check:goreleaser` extended, or a new `task verify:homebrew-cask-shape`) must be defined as a Taskfile target and invoked from the workflow via `task <target>`, never written inline in `release.yml`'s `run:` block — a grep of the workflow alone would return a false negative for "no enforcement," per memory `yctys69cke` cited in CONTEXT.md/RESEARCH.md.

---

## Shared Patterns

### Cobra Go-only-divergence documentation
**Source:** `internal/cli/githooks.go` (constructor doc comments), `docs/FLAG-PARITY.md` (the divergence table)
**Apply to:** `internal/cli/man.go`, `internal/cli/root.go`
Every Go-only command names its decision id inline in its doc comment and gets an entry in `docs/FLAG-PARITY.md`. `man` differs from `githooks` in being `Hidden: true` — record that as a deliberate, named divergence in both places, not a silent omission from `--help`.

### `.goreleaser.yaml` inline-rationale density
**Source:** `.goreleaser.yaml` (`archives:`, `checksum:`, `notarize:`, `signs:`, `sboms:`, `release:` blocks)
**Apply to:** the new `homebrew_casks:` block
Every block names its decision id(s) and the shape test that holds each non-obvious field. This is a load-bearing repo convention, not decoration — a reviewer or future editor is expected to find the "why" in the file itself.

### Shape-test property-assertion discipline
**Source:** `internal/upgrade/goreleaser_shape_test.go` (`toStringSlice`/`sortedJoin` exact-set idiom), `.goreleaser.yaml`'s `sboms:` comment (resolve-and-assert-distinctness, not literal-string)
**Apply to:** all new shape tests in `goreleaser_shape_test.go`, `release_workflow_shape_test.go`, `taskfile_shape_test.go`
Assert a **property** (exact set membership, distinctness after template resolution, presence/absence of a key) — never a literal rendered string. This repo has a recorded instance (`TestSbomsArePerBinaryWithSpdxNames`'s predecessor pinning a broken template) where the literal-string failure mode made a test worse than no test, because it resisted correction once green.

### GitHub App token minting
**Source:** `.github/workflows/release-please.yml` lines 72-92
**Apply to:** the new tap-token-minting job in `.github/workflows/release.yml`
Reuse `actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0` verbatim (same pin); only the App identity and secret names differ. Never reuse `secrets.APP_ID`/`secrets.APP_PRIVATE_KEY` — those are the release-please App's, installed on `codegraph-go`, and reusing them for the tap would fail criterion 5 (D-16/D-17, Pitfall 2).

### Secret scoping: repository-level, step-level env
**Source:** `internal/upgrade/release_workflow_shape_test.go`'s `TestAppleSecretsScopedToSingleReleaseJob` (asserts step-level `env:`, single-job scoping)
**Apply to:** the new tap App secrets and the `homebrew_casks.repository.token` consumption in `.goreleaser.yaml`
Must be **repository** secrets, never environment secrets (D-16, echoing memory `q5yhyebw5k`'s Apple-secret precedent where environment-scoped secrets shipped an un-notarized release with a green log). Consumed only via step-level `env:` in the single job that needs them.

### Taskfile as the single definition of CI job bodies
**Source:** `internal/upgrade/taskfile_shape_test.go`'s `TestWorkflowRunBodiesInvokeTask`
**Apply to:** any new verification logic this phase adds (goreleaser config validation, shape-test invocation, etc.)
New logic belongs in a Taskfile target; the workflow step's `run:` body must invoke `task <target>`, not embed the logic inline.

## No Analog Found

None — every file in the expected set has a direct, same-repo, same-file-family analog. This phase is unusually well-covered because it extends three files (`internal/cli/root.go`+`githooks.go` family, `.goreleaser.yaml`, `internal/upgrade/*_shape_test.go`) that already carry the exact conventions the new work must follow.

## Superseded Mechanism — Do Not Plan For

RESEARCH.md's BREW-06/D-18 "scratch-repo mechanism" for demonstrating a failed tap push is **superseded by CONTEXT.md D-18R** (maintainer decision, 2026-08-09): `--snapshot` sets `skips.Publish` and `cask.Pipe{}` runs inside the Publish pipeline, so no local run can reach the failure path. D-18R accepts a **structural argument only** (citing `internal/pipe/publish/publish.go`'s literal pipe ordering — `release.Pipe{}` before `cask.Pipe{}`), with **no executed evidence**. Do not map or plan analogs for a rehearsal harness, scratch repo, or throwaway-token demonstration for this leg — RESEARCH.md's Pitfall 3 and Open Question 1 predate this correction and must not drive planning.

## Metadata

**Analog search scope:** `internal/cli/`, `internal/upgrade/`, `.goreleaser.yaml`, `.github/workflows/`, `Taskfile.yml`, `docs/`
**Files scanned:** `internal/cli/githooks.go`, `internal/cli/root.go`, `internal/upgrade/goreleaser_shape_test.go`, `internal/upgrade/release_workflow_shape_test.go`, `internal/upgrade/taskfile_shape_test.go` (targeted sections), `.goreleaser.yaml` (archives/checksum/notarize/signs/sboms/release blocks), `.github/workflows/release-please.yml`, `Taskfile.yml` (grep pass for `if [` idiom and the Phase-3 anticipation comment at line 1755)
**Pattern extraction date:** 2026-08-09
