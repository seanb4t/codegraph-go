---
phase: 05-release-chain-release-please-retirement
reviewed: 2026-10-14T00:00:00Z
depth: deep
files_reviewed: 22
files_reviewed_list:
  - .changie.yaml
  - .github/workflows/ci.yml
  - .github/workflows/post-release-verify.yml
  - .github/workflows/pr-title.yml
  - .github/workflows/release-pr.yml
  - .github/workflows/release-tag.yml
  - .github/workflows/release.yml
  - .github/workflows/require-issue-link.yml
  - .goreleaser.yaml
  - CONTRIBUTING.md
  - Taskfile.yml
  - docs/RELEASE-PROCEDURES.md
  - docs/RELEASE.md
  - internal/upgrade/changie_release_shape_test.go
  - internal/upgrade/changie_shape_test.go
  - internal/upgrade/goreleaser_shape_test.go
  - internal/upgrade/release_workflow_shape_test.go
  - internal/upgrade/tag_authority_test.go
  - internal/upgrade/taskfile_shape_test.go
  - scripts/fragment_policy.py
  - scripts/pr_template_policy.py
  - scripts/test_fragment_policy.py
findings:
  critical: 0
  warning: 3
  info: 5
  total: 8
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-10-14
**Depth:** deep
**Files Reviewed:** 22
**Status:** issues_found

## Summary

Reviewed the phase diff against `500cdc65` for every listed file, and traced the release chain end to end: `release-pr.yml` -> `task release:batch` -> `peter-evans/create-pull-request` (App token) -> merge -> `release-tag.yml` -> `task release:tag` (App token) -> `release.yml` -> `task release:goreleaser`.

Verified directly:

- `task check:changie` passes 14 of 14 legs.
- `task release:batch` run in a scratch copy of HEAD renders `.changes/v0.15.0.md`, merges `CHANGELOG.md` correctly, writes `version=v0.15.0` to `GITHUB_OUTPUT` and deletes the fragment.
- `scripts/test_fragment_policy.py` runs 46 of 46 tests, matching the floor in `check:fragment-policy`.
- `go test ./internal/upgrade` for the release-chain shape tests passes.
- `release.yml` differs from `7b12cb92` by exactly one comment line (L188). No functional drift.
- No `release-please` reference remains in any reviewed file.
- The PR and the tag are authored with the App token, never `GITHUB_TOKEN`. Both mint steps precede the consuming step, and the shape tests pin that order.

No blocker was found. The release chain is sound in its main path. The findings below are an edge-case race in the release-PR job, two hygiene-gate weaknesses (one of them touched by this phase), and documentation or test-fragility defects.

## Warnings

### WR-01: `release-pr` leaves a stale open release PR when the fragment count drops to zero

**File:** `Taskfile.yml:1017-1022` (and `.github/workflows/release-pr.yml:89-98`)
**Issue:** When `.changes/unreleased` holds no `*.yaml`, `release:batch` exits 0 with no `version` output, and the `Open or update the release PR` step is skipped. Nothing ever closes or refreshes an already-open `changie/release` PR in that state.

Concrete sequence:
1. Push P1 to `main` starts release-pr run R1, which pushes and updates the `changie/release` PR.
2. A maintainer merges the previous version of that PR (push P2) before R1's PR step completes, or reverts or removes the fragments.
3. R2 runs on P2, sees zero fragments, and does nothing.

The PR R1 left open now targets a `main` that already carries the same `.changes/vX.Y.Z.md` (or has lost the fragments it was built from). It conflicts or is empty, and no later run repairs it until a new fragment lands. Because `concurrency` uses `cancel-in-progress: false`, R1 runs to completion after P2 is pushed, so the window is real, though narrow. The stale PR also has a mergeable-looking title (`chore(main): release vX.Y.Z`), so a merge would re-touch `CHANGELOG.md` and fire `release-tag.yml`.

**Fix:** In the zero-fragment path, close any open PR on `changie/release` (App token). For example, after the batch step:
```yaml
- name: Close a stale release PR
  if: steps.batch.outputs.version == ''
  env:
    GH_TOKEN: ${{ steps.app-token.outputs.token }}
  run: gh pr close changie/release --delete-branch --comment "No unreleased fragments remain." || true
```
Keep the `gh` call in a Taskfile target if the repo's "run bodies invoke task" rule applies. Alternatively, document the stale-PR case in `docs/RELEASE-PROCEDURES.md` §4 beside the "Expected silent no-op" paragraph.

### WR-02: Path-based hygiene exemption fails open on PRs with more than 100 files

**File:** `.github/workflows/require-issue-link.yml:45-62, 79-95` and `scripts/pr_template_policy.py:128-137`
**Issue:** `CHANGED_FILES` comes from `gh pr view --json files`. `scripts/fragment_policy.py` documents in its own module docstring that this call "stops at 100 files", and was deliberately written to avoid it. `require-issue-link.yml` and `pr_template_policy.is_housekeeping` do not avoid it. A PR with 100 housekeeping paths (now including trivially creatable `.changes/**` files, which this phase added to both allowlists) plus shipped code beyond the first 100 entries is classified `all_exempt=true` and skips both the issue-link and the template gates. This phase widened the exempt set (`.changes/`) and added the mirror test, but left the truncation unhandled, so the two hygiene gates are weaker than the fragment gate in the same directory.

**Fix:** Use the paginated REST listing as `fragment_policy.fetch_changed_files` does, and compare the count against `changed_files`. Fail closed (not exempt) on mismatch:
```sh
gh api --paginate --slurp "repos/${GITHUB_REPOSITORY}/pulls/${PR_NUMBER}/files?per_page=100" --jq '.[][].filename'
```
Then verify the count equals `gh api repos/.../pulls/N --jq .changed_files`.

### WR-03: `pr_template_policy.emit` writes the attacker-controlled exemption reason to stderr unescaped

**File:** `scripts/pr_template_policy.py:122-125, 131-133`
**Issue:** `emit()` runs `print(f"{action}: {reason}", file=sys.stderr)`, and `main()` passes `f"explicitly exempt: {reason}"`. `reason` is the PR body text between the marker and `-->`. The regex `[^>]*?` allows embedded newlines. A body such as `<!-- pr-template-exempt: x` followed by a newline, then `::stop-commands::tok` or `::error::...`, then `-->`, puts a line starting with `::` into the step log, where the runner parses it as a workflow command (annotation spoofing, command suppression). `fragment_policy.report` already guards the identical hazard with `escape_data`, and its docstring says "Every untrusted value passes through escape_data first". This phase rewrote `find_exemption` and the call site in `pr_template_policy.py`, so the unescaped sink is inside the changed code.

**Fix:** Escape the log line, or refuse multi-line reasons in `find_exemption` so both policies share one grammar:
```python
def escape_data(text: str) -> str:
    return text.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")

print(f"{action}: {escape_data(reason)}", file=sys.stderr)
```
Better: move `escape_data` to a shared place, or reject a reason containing `\n` or `\r` in both `find_exemption` functions.

## Info

### IN-01: `CONTRIBUTING.md` and `.changie.yaml` give a fragment command that blocks without a TTY

**File:** `CONTRIBUTING.md:135-141`, `.changie.yaml:7-9`
**Issue:** Both document `task changie -- new -k <Kind> -b "<sentence>"` with `-m PR=<n>` "optional". Run without `CI=true`, changie prompts interactively for the optional `PR` custom field. I reproduced a hang (killed by timeout, "model conversion failed") with stdin closed and `CI` unset. `scripts/fragment_policy.py` already prints the correct form (`CI=true task changie -- new ...`) in its failure message, and `check:changie` leg 5 uses `CI=true`. So the contributor docs and the gate's own guidance disagree, and any agent or scripted contributor following the docs hangs.
**Fix:** Document `CI=true task changie -- new -k <Kind> -b "<sentence>"` in `CONTRIBUTING.md`. Optionally note the interactive prompt for a human on a terminal.

### IN-02: `check:changie` leg 14 silently depends on fragments left behind by legs 4-13

**File:** `Taskfile.yml` (leg 14 block, about lines 906-985)
**Issue:** Leg 14's sub-check (iii) requires both `### Breaking` and `### Fixes` in the batched file. It gets them only because legs 4, 5, 9 and 11 left Fixes and Breaking fragments in the scratch copy. Sub-check (iv) also hard-codes `tail -n +3 CHANGELOG.md`, which assumes the header template is exactly one line plus one blank line. Reordering the legs, or adding a fragment-removing leg, breaks leg 14 for a reason unrelated to what it tests, and the failure message does not say why.
**Fix:** Have leg 14 write its own Breaking and Fixes fragments (or assert the precondition with a labelled message before running). Derive the skip count from `wc -l < .changes/header.tpl.md` plus the one separator line rather than the literal `3`.

### IN-03: `docs/RELEASE-PROCEDURES.md` §2 still lists a hand-pushed release candidate as a normal tag shape

**File:** `docs/RELEASE-PROCEDURES.md:48`
**Issue:** The tag-conventions table row "Release candidate | `v0.0.0-rc.3` | ... | Triggers `release.yml`? Yes" contradicts §4's D-15 paragraph (no hand-pushed prerelease tag) and `release:goreleaser`, which refuses any tag lacking `.changes/<tag>.md`. A reader following §2 pushes an rc tag, starts the macOS signing job, and fails after the tap-token mint step.
**Fix:** Change the row to say an `rc` tag does start `release.yml` but fails at `release:goreleaser` unless a matching `.changes/<tag>.md` exists, or remove the row and point to §4.

### IN-04: The "linear-time" claim for the exemption markers is not accurate

**File:** `scripts/fragment_policy.py:48-50`, `scripts/pr_template_policy.py:82-86`, `scripts/test_fragment_policy.py` (`MarkerGrammarTests`)
**Issue:** The comments and test name assert the new grammar `<!--\s*name:(?P<reason>[^>]*?)-->` is linear. It is linear per opener, and the comment says so, but a body that repeats the opener with no `>` (`<!--pr-template-exempt:` x 3000, about 64 KB) is quadratic. Measured: about 0.5 s for a 64 KB body, 0.26 s at 46 KB. Not exploitable at a 64 KB body cap, but the tests only cover two single-opener shapes (spaces after one opener), so the repeated-opener shape is untested and the 1.0 s bound has little headroom on a slow runner.
**Fix:** Either reword the comments to "no catastrophic backtracking" and drop the "linear-time" claim, or add a repeated-opener case to `test_exempt_markers_are_linear_on_adversarial_bodies`. A single-pass `str.find("-->")` parse would be truly linear.

### IN-05: The tag job keeps the App token in `.git/config` while `go tool` runs repository-pinned third-party code

**File:** `.github/workflows/release-tag.yml:75-90`
**Issue:** `actions/checkout` with `token: <App token>` persists the credential for the rest of the job. `task release:tag` then builds and runs changie via `go tool -modfile=go.tool-changie.mod` before it pushes. The module is pinned by `go.sum`, so this is an accepted-surface hardening point, not a defect. The App token has contents write on the repository for the whole job.
**Fix (optional):** Check out with `persist-credentials: false` and have `release:tag` push with an explicit `git -c http.extraheader=...` or an authenticated remote URL supplied through `env:` only at the push. This conflicts with the current `validateTagJobCredential` test and the "persist-credentials stays at its default" comment, so it is a design decision, not a required fix.

---

_Reviewed: 2026-10-14_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
