# Phase 9: release-please + GoReleaser - Pattern Map

**Mapped:** 2026-07-28
**Files analyzed:** 8 (create: 5, modify: 3)
**Analogs found:** 8 / 8 (all role-match or exact; one explicit "no existing idiom" finding below)

This phase is CI/config/docs-heavy. There is no application "controller/service"
code in scope — every file is either a GitHub Actions workflow, a JSON config,
a Go test, or a Markdown runbook. Analog matching is therefore against sibling
workflow files, the existing Go test file it extends, and the existing runbook
it amends.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.github/workflows/release-please.yml` (NEW) | CI workflow (event-driven, push-trigger) | event-driven | `.github/workflows/release.yml` (structure/conventions) + `.github/workflows/ci.yml` (`pull_request`/`push` trigger shape) | role-match (no prior release-please-shaped workflow exists; conventions transfer exactly) |
| `release-please-config.json` (NEW) | config | — | none in-repo (first JSON config file of this kind) | no analog — new idiom, low risk (flat JSON, schema-driven by upstream tool) |
| `.release-please-manifest.json` (NEW) | config | — | none in-repo | no analog — new idiom, trivial shape |
| `.github/workflows/release.yml` (MODIFY — one step, line ~266–284) | CI workflow, `assemble` job | request-response (shell → `gh` CLI) | itself — same file, same job's existing `Checksums`/`Sign each binary` steps for the `env:`-indirection + `set -euo pipefail` idiom | exact (edit-in-place, follow the surrounding steps' own convention) |
| `.github/workflows/ci.yml` (MODIFY — new job, PR-title lint) | CI workflow, new job | request-response (event-driven) | `.github/workflows/ci.yml`'s own `test` job (shell step + `set -euo pipefail` conventions); trigger-type gap analog is `pull_request:` block itself | exact (same file — new sibling job following existing job shape) |
| New Go test in `internal/upgrade/` — `TestReleaseWorkflowFileMatchesPattern` (NEW) | test (static drift guard) | transform (parse YAML text → regex match) | `internal/upgrade/verify_test.go` lines 114–137 (`TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo` / `_AcceptsReleaseWorkflowTagRef`) | exact — same package, same accept/reject test shape, same constants |
| Stubbed-`gh` shell test for D-04 create-vs-upload branch (NEW) | test (shell/CLI stub) | request-response | **none — no shell/bats test harness or PATH-stubbed-binary test exists anywhere in this repo today** | no analog — new testing idiom for this repo (see below) |
| `docs/RELEASE-PROCEDURES.md` (MODIFY §3/§4/§7 + new App section) | docs | — | itself — existing §1/§2/§5/§6/§8 structure and heading/code-block style | exact (edit-in-place, match surrounding sections) |

## Pattern Assignments

### `.github/workflows/release-please.yml` (NEW)

**Analogs:** `.github/workflows/release.yml` (header-comment/SHA-pin/`env:`-indirection conventions) and `.github/workflows/ci.yml` (trigger block shape, `permissions:` minimalism).

**Header-comment convention** (`release.yml` lines 1–36): every LOCKED-adjacent workflow file opens with a comment block stating *why* its shape matters and what it must never do. The new file should carry an equivalent note explaining that it is what makes the App-token tag push (which `release.yml` depends on) exist at all — see D-01/D-02 in CONTEXT.md.

**Action SHA-pin convention** (`release.yml` lines 103, 110, 117, 122, 224, 244):
```yaml
uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
```
Full commit SHA + trailing `# vX.Y.Z` comment, no exceptions except the SLSA generic generator (which must stay tag-pinned, `provenance` job line 300 — `slsa-verifier` requires it). Apply this same pattern to `actions/create-github-app-token` and `googleapis/release-please-action` (SHAs already resolved in RESEARCH.md's Code Examples section: `bcd2ba49218906704ab6c1aa796996da409d3eb1` # v3.2.0, `8b8fd2cc23b2e18957157a9d923d75aa0c6f6ad5` # v5.0.0).

**Minimal top-level `permissions:` with per-job elevation** (`release.yml` lines 45–46, `ci.yml` lines 42–43):
```yaml
permissions:
  contents: read
```
`release.yml`'s `assemble` job re-elevates locally (lines 196–201):
```yaml
    permissions:
      contents: write # gh release create + asset uploads
      id-token: write # cosign keyless OIDC signing (Finding 1) — this
      # job's OIDC token is what produces the cert SAN
      # releaseWorkflowRefPattern anchors on...
```
The new `release-please.yml` needs `contents: write`, `issues: write`, `pull-requests: write` at the job (or workflow) level per RESEARCH.md Pattern 1 — follow the same "state *why* this permission is needed" comment style, not a bare list.

**Trigger block shape** (`ci.yml` lines 36–40):
```yaml
on:
  pull_request:
  push:
    branches:
      - main
```
The new file's trigger is simpler (`push: branches: [main]` only, per D-02/CONTEXT.md) — same key structure, narrower scope.

**`env:`-indirection for `run:` blocks** (`release.yml` lines 136–146, the load-bearing security comment):
```yaml
env:
  TAG: ${{ github.ref_name }}
  GOOS: ${{ matrix.goos }}
  GOARCH: ${{ matrix.goarch }}
run: |
  set -euo pipefail
  ...
```
The comment directly above (lines 137–140) is the canonical statement of *why* — reuse verbatim reasoning for any new step in `release-please.yml`/`release.yml` that touches `$TAG`/`$REPO` (e.g. the pretag-gate job's `GOOS`/`GOARCH` loop, RESEARCH.md Pattern 5).

---

### `.github/workflows/release.yml` — `assemble` job, `Publish GitHub release` step (MODIFY, lines 266–284)

**Analog:** the file's own preceding steps in the same job (`Checksums` lines 215–221, `Sign each binary individually` lines 226–241) — all follow `working-directory: release` + `env:` block + `set -euo pipefail` + inline comment explaining the *why*.

**Current step to replace** (lines 266–284):
```yaml
      - name: Publish GitHub release
        working-directory: release
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAG: ${{ github.ref_name }}
          REPO: ${{ github.repository }}
        run: |
          set -euo pipefail
          PRERELEASE_FLAG=""
          case "$TAG" in
          *-*) PRERELEASE_FLAG="--prerelease" ;;
          esac
          # shellcheck disable=SC2086
          gh release create "$TAG" \
            --repo "$REPO" \
            --title "$TAG" \
            --generate-notes \
            $PRERELEASE_FLAG \
            codegraph_*
```
**Target shape** (D-04, RESEARCH.md Pattern 4 — keeps the exact same `env:`/`set -euo pipefail`/`shellcheck disable=SC2086` conventions, only the body branches):
```yaml
      - name: Publish GitHub release
        working-directory: release
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAG: ${{ github.ref_name }}
          REPO: ${{ github.repository }}
        run: |
          set -euo pipefail
          PRERELEASE_FLAG=""
          case "$TAG" in
          *-*) PRERELEASE_FLAG="--prerelease" ;;
          esac
          if gh release view "$TAG" --repo "$REPO" >/dev/null 2>&1; then
            # release-please already created this release (body/prerelease
            # flag are release-please's — do NOT pass --generate-notes or
            # re-set --prerelease here, see D-04).
            gh release upload "$TAG" --repo "$REPO" --clobber codegraph_*
          else
            # manual rc tag path (D-10) — unchanged from today.
            # shellcheck disable=SC2086
            gh release create "$TAG" \
              --repo "$REPO" \
              --title "$TAG" \
              --generate-notes \
              $PRERELEASE_FLAG \
              codegraph_*
          fi
```
This is the single highest-risk edit in the phase (D-04) — keep it a minimal diff against the existing step, do not restructure the surrounding job.

---

### `.github/workflows/ci.yml` — new PR-title conventional-commit lint job (MODIFY)

**Analog:** the file's own `test` job shell-step conventions (lines 46–147) — every `run:` step uses `set -euo pipefail` where multi-line, plain `run: <cmd>` where single-line.

**Pitfall to encode from RESEARCH.md Pitfall 1:** the shared `pull_request:` trigger (line 37, no `types:` override) defaults to `opened, synchronize, reopened` — a title-only edit does NOT re-trigger. RESEARCH.md's own recommendation: either widen `ci.yml`'s single `pull_request:` block to `types: [opened, edited, synchronize, reopened]` (re-runs the whole heavy suite on every title edit) or split the lint into its own lightweight sibling workflow file with a narrowly-scoped trigger. This is an open implementation decision for the planner — CONTEXT.md names `ci.yml` but doesn't resolve the tradeoff; both options are valid per D-08's discretion.

**Shape (hand-written `grep -E`, matching D-08's discretion + this repo's "auditable CI shell over opaque actions" convention, RESEARCH.md line 27/100):**
```yaml
  pr-title-lint:
    name: PR title (conventional commits)
    runs-on: ubuntu-latest
    steps:
      - name: Check PR title matches Conventional Commits
        env:
          TITLE: ${{ github.event.pull_request.title }}
        run: |
          set -euo pipefail
          echo "$TITLE" | grep -E '^(feat|fix|perf|refactor|docs|chore|ci|test|build|revert)(\([a-z0-9_-]+\))?(!)?: .+' \
            || { echo "::error::PR title is not Conventional-Commits-shaped: $TITLE"; exit 1; }
```
Note `env:`-indirection is used here too (never `run: echo "${{ github.event.pull_request.title }}"` directly) — same injection-safety convention as `release.yml`.

---

### New Go test in `internal/upgrade/` — workflow-shape drift guard (NEW)

**Analog:** `internal/upgrade/verify_test.go` lines 114–137 (read in full below — both existing tests are the direct model).

```go
// TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo is the
// WR-08 regression test: the pre-fix pattern was an unanchored PREFIX
// match, so a SAN merely starting with this repo's GitHub URL would pass
// — including a signature produced by an unrelated, weaker-trust-boundary
// workflow (e.g. a pull_request-triggered CI workflow) in the same repo,
// not just the intended tag-triggered release workflow.
func TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo(t *testing.T) {
	re := regexp.MustCompile(releaseWorkflowRefPattern)
	unrelated := "https://github.com/" + releaseRepoSlug + "/.github/workflows/ci.yml@refs/heads/main"
	if re.MatchString(unrelated) {
		t.Fatalf("releaseWorkflowRefPattern must reject a non-release workflow in the same repo, matched: %q", unrelated)
	}
}

// TestReleaseWorkflowRefPattern_AcceptsReleaseWorkflowTagRef proves the
// anchored pattern still accepts the identity it's meant to authorize: the
// release workflow itself, triggered by a version-tag push.
func TestReleaseWorkflowRefPattern_AcceptsReleaseWorkflowTagRef(t *testing.T) {
	re := regexp.MustCompile(releaseWorkflowRefPattern)
	valid := "https://github.com/" + releaseRepoSlug + "/.github/workflows/release.yml@refs/tags/v1.2.3"
	if !re.MatchString(valid) {
		t.Fatalf("releaseWorkflowRefPattern must accept the release workflow's own tag-triggered ref, got no match for: %q", valid)
	}
}
```

**What the new test must do differently:** the two tests above assert the *pattern in isolation* (hand-built strings). The new `TestReleaseWorkflowFileMatchesPattern` (RESEARCH.md's Validation Architecture, Wave 0 gap #1) must instead **read the literal `.github/workflows/release.yml` file off disk**, extract its `name:` value and `on.push.tags` value (simple text/YAML parse — no need for a full YAML library if a targeted regex/line-scan suffices, matching this file's existing "belt-and-suspenders... no exec usage" self-contained style at `verify_test.go` lines 139+), reconstruct the SAN string that a real tag push would produce (`https://github.com/<releaseRepoSlug>/.github/workflows/<name>.yml@refs/tags/v1.2.3`), and assert it **still matches** `releaseWorkflowRefPattern`. Follow the **accept-and-reject shape**: one case proves the current `release.yml` passes; a second constructs a deliberately-wrong value (e.g. a hypothetical renamed file) and proves that value is rejected — non-vacuous, per the Phase-8 CR-01/WR-02 lesson cited throughout CONTEXT.md and RESEARCH.md.

**Constants block being tested against** (`internal/upgrade/verify.go` lines 20–45, read in full):
```go
// Named release-identity constants (D-12/D-14). releaseOIDCIssuer is
// GitHub Actions' Sigstore-public-good OIDC issuer — stable, not a
// placeholder. releaseRepoSlug and releaseWorkflowRefPattern ARE
// Phase-8-finalized...
//
// releaseWorkflowRefPattern is a FULL-MATCH pattern (WR-08): anchored at
// both ^ and $, and scoped to the specific release-publishing workflow
// file and a tag-triggered ref...
const (
	releaseOIDCIssuer         = "https://token.actions.githubusercontent.com"
	releaseRepoSlug           = "seanb4t/codegraph-go"
	releaseWorkflowRefPattern = `^https://github\.com/` + releaseRepoSlug + `/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`
)
```
The new test reads these three constants directly (same package, no export needed) — exactly as `verify_test.go`'s existing tests do.

---

### Stubbed-`gh` shell test for D-04 create-vs-upload branch (NEW)

**No existing analog — this is a new testing idiom for this repo.**

Searched: no `.bats` files anywhere in the repo; no shell test harness directory; no Go test that stubs a binary on `PATH` (the closest thing, `verify_test.go`'s `TestVerifyGo_NoExecUsage`, statically greps `verify.go`'s source for `os/exec` usage — it does not stub or invoke any external binary). `test/integration/` and `testdata/golden/` exist but both drive the `codegraph` binary itself via Go's `os/exec`/subprocess harness, not a stubbed third-party CLI like `gh`.

The planner should treat this as **introducing a new pattern**, not extending one. Two reasonable shapes, per RESEARCH.md's Standard Stack / Wave 0 gaps section (neither has an in-repo precedent to copy):
1. A small standalone shell script (`.github/workflows/testdata/` or a repo-root `scripts/` location — pick per repo convention once one exists) that puts a fake `gh` function/script earlier on `PATH`, sources/execs the extracted shell logic from the `Publish GitHub release` step, and asserts which `gh` subcommand fired for both the release-exists and release-absent cases.
2. A Go test using `os/exec.Command` with a temp-dir `PATH` override and a stub `gh` script — consistent with the repo's general preference for Go over ad hoc shell tooling (RESEARCH.md's "Don't Hand-Roll" table character), but still a new idiom since no existing Go test stubs an external binary this way.

Either is defensible; flag explicitly in the plan which is chosen, since it establishes precedent for future CI-logic tests in this repo.

---

### `docs/RELEASE-PROCEDURES.md` — §3/§4/§7 rewrite + new App section (MODIFY)

**Analog:** the document's own surviving sections (§1, §2, §5, §6, §8).

**Section heading + intro style** (lines 1–10, 11–29, 31–44):
```markdown
## 1. Pre-tag gate (mandatory — D-09)

Before tagging **anything** (rc or stable), run the following for **all 6**
release targets. This is the exact check that would have caught the v0.1
`rc.1` failure: ...

```sh
for pair in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
  ...
done
```

All 6 lines must print `OK`. A single `FAIL` means ...
```
Pattern: `## N. <imperative-ish title> (<parenthetical D-ref or qualifier if load-bearing>)`, a short prose paragraph stating *why* (often citing a real past incident — the `rc.1` linux-only failure), then a fenced code block, then a follow-up prose paragraph on how to interpret the output. The §3/§4/§7 rewrite and the new App-prerequisite section should match this exact rhythm rather than reading as a bolt-on.

**§5's "verbatim, source-wins" convention** (lines 85–115) is the model for how the rewritten §3/§4 should treat any code/config it quotes (the new `release-please-config.json`, the App-token workflow snippet): state explicitly that the doc is not authoritative and the source file wins if they disagree — this repo has been bitten twice by doc-vs-code drift (RESEARCH.md D-07 rationale) and §5 already encodes the house style for avoiding a third instance.

**§3/§4 replacement content is fully specified in CONTEXT.md's Folded Todos section** — the manual-`git tag` steps (current §4 lines 58–69) are replaced by the release-please PR-merge flow; §3's "squash-merge to main" (line 51) must be corrected to fast-forward per D-09's evidence-based override, not silently left as-is.

## Shared Patterns

### SHA-pinning every third-party GitHub Action, tag-pinning only where the consumer requires it
**Source:** `.github/workflows/release.yml` lines 32–36 (header comment), applied throughout the file (e.g. lines 103, 110, 117, 122, 224, 244, and the SLSA generator's documented tag-pin exception at line 300).
**Apply to:** `.github/workflows/release-please.yml` (new `actions/create-github-app-token`, `googleapis/release-please-action` uses), any new action added to `ci.yml`'s PR-title job if an off-the-shelf action is chosen over hand-written `grep -E`.

### `env:`-indirection — never interpolate `${{ }}` directly into `run:`
**Source:** `.github/workflows/release.yml` lines 136–144 (the step's own explanatory comment is the canonical statement of the rule).
**Apply to:** every new/modified `run:` step touching `$TAG`, `$REPO`, or `${{ github.event.pull_request.title }}` across `release-please.yml`, the modified `release.yml` step, and the new `ci.yml` job.

### `set -euo pipefail` in every multi-line `run:` block
**Source:** `.github/workflows/release.yml` (every multi-line step, e.g. lines 145–146, 219–221, 228–229) and `.github/workflows/ci.yml` line 70.
**Apply to:** all new multi-line shell steps in this phase (pretag-gate loop, PR-title grep, the D-04 branch).

### Minimal top-level `permissions:`, elevated per-job with an inline "why" comment
**Source:** `.github/workflows/release.yml` lines 45–46 (top-level `contents: read`) + lines 196–201 (`assemble` job's elevation with rationale comment).
**Apply to:** `release-please.yml`'s job-level `contents: write` / `issues: write` / `pull-requests: write` grant — state *why* each is needed (mirrors RESEARCH.md Pitfall 2's App-vs-workflow-permission distinction), not just a bare list.

### Accept-and-reject (non-vacuous) test pairs for any pattern/guard
**Source:** `internal/upgrade/verify_test.go` lines 114–137 — `TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo` + `TestReleaseWorkflowRefPattern_AcceptsReleaseWorkflowTagRef`.
**Apply to:** the new `TestReleaseWorkflowFileMatchesPattern` drift guard, the PR-title lint's own sample-title test table, and (if D-03's `workflow_dispatch` fallback is ever implemented) its ref-shape guard test — CONTEXT.md and RESEARCH.md both explicitly invoke the Phase-8 CR-01/WR-02 lesson ("a guard that is present but never fires is not a guard") as the reason this shape is mandatory, not optional, for every new guard in this phase.

### Documented divergence over silent drift
**Source:** `docs/RELEASE-PROCEDURES.md` §5 (lines 85–115, "the source wins if they disagree") and CONTEXT.md's own D-05 treatment (roadmap criterion 3 divergence recorded explicitly, not dropped).
**Apply to:** the `docs/RELEASE-PROCEDURES.md` rewrite (§3's fast-forward-vs-squash correction) and D-05's GoReleaser-role divergence, both of which must be recorded in-doc, not silently resolved.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `release-please-config.json` | config | — | First JSON config file of this shape in the repo; schema is entirely upstream-tool-defined (release-please), so RESEARCH.md's Code Examples section is the authoritative source, not an in-repo analog. |
| `.release-please-manifest.json` | config | — | Same as above — trivial `{".": "0.1.0"}` shape, no in-repo precedent needed. |
| Stubbed-`gh` shell test (D-04 branch coverage) | test | request-response | **Confirmed: no shell/bats test harness or PATH-stubbed-binary test exists anywhere in this repo.** This introduces a new testing idiom — see the dedicated writeup above. The planner must pick and document one of the two proposed shapes rather than assume an existing convention. |

## Metadata

**Analog search scope:** `.github/workflows/` (all 2 existing workflow files, read in full or targeted ranges), `internal/upgrade/verify.go` + `verify_test.go` (targeted line ranges per phase-specific guidance), `docs/RELEASE-PROCEDURES.md` (first 120 lines + heading index for full-doc structure), repo-wide search for `.bats` files and shell-test harnesses (none found).
**Files scanned:** 6 read directly (`release.yml` full, `verify.go` lines 1–50, `verify_test.go` lines 100–145, `ci.yml` lines 34–148, `RELEASE-PROCEDURES.md` lines 1–120 + heading grep, `09-CONTEXT.md`/`09-RESEARCH.md` in full).
**Pattern extraction date:** 2026-07-28
