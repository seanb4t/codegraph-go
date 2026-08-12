---
phase: 03-homebrew-tap-cask
verified: 2026-08-10T00:00:00Z
status: passed
score: 5/5 criteria verified (2 with amendments judged legitimate; 1 finding flagged as WARNING)
behavior_unverified: 0
overrides_applied: 0
gaps: []
human_verification: []
---

# Phase 3: Homebrew Tap & Cask Verification Report

**Phase Goal:** A macOS user installs codegraph the way they install everything else — `brew install` — and it keeps working across releases rather than only on the day it was hand-checked.
**Verified:** 2026-08-10
**Status:** PASSED (with one recorded finding requiring attention, and one bookkeeping gap noted; neither blocks the phase goal)
**Re-verification:** No — initial verification

## Verdict Summary

The phase goal is achieved and independently re-confirmed against live, published state — not just against SUMMARY.md prose. I re-downloaded the real `v0.8.0` release assets and hashed them myself (matched the tap cask's declared sha256 exactly), re-read the tap repository's commit history via the GitHub API (one commit, `goreleaserbot`, matching the evidence), re-ran the relevant Go shape tests myself, and re-read the pinned GoReleaser v2.17.1 module source directly from the module cache to confirm the BREW-06 structural argument's citations are accurate (line numbers and quoted comments match exactly). All of this corroborates `03-EVIDENCE.md` rather than merely restating it.

Both amendments made during the phase are judged **legitimate**: each is evidence-backed, independently re-verifiable from source (not merely asserted), and each amendment's residual gap (the `test:` stanza doesn't exist in Homebrew's Cask DSL; the tap-push UPDATE path and `brew upgrade` remain unexercised because only one release was cut) is named plainly rather than smoothed over or silently absorbed into a "Complete" checkbox. This is the opposite of goalpost-moving — the team narrowed the *wording* to match what the pipeline can actually guarantee while explicitly disclosing what narrower scope was NOT covered.

One substantive finding surfaced during this verification that the phase's own evidence does not itself catch: **`03-EVIDENCE.md`'s claim that "a failed-then-abandoned install can leave the [Phase-4] sentinel behind" is not supported by — and is directly contradicted by — its own recorded observation.** See "Finding: the sentinel-leak claim is unsupported by its own evidence" below. This does not block Phase 3 (BREW-05's install gate is genuinely, positively demonstrated; the *asymmetry* itself — hook-written files surviving a failed-install rollback — is real and correctly demonstrated for man pages), but it is exactly the kind of "confident claim from a check whose failure looks like success" pattern this repository has been bitten by before, and it directly touches Phase 4's dependency on this phase, per the task brief's own instruction to assess that risk.

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria, as amended)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Cold `brew tap seanb4t/tap && brew install codegraph` completes and the binary runs a real command, verified against a cask GoReleaser both rendered and pushed inside the same automated release run | ✓ VERIFIED | Re-confirmed live: `gh run view 31423733320` → `release`, `push`, `success`; tap `Casks/codegraph.rb` written once by `goreleaserbot` (`gh api .../commits?path=...` → 1 commit); `03-EVIDENCE.md` "BREW-01 — the cold install" records a real `brew tap`/`brew trust`/`brew install` cycle on a torn-down machine, binary reports `v0.8.0`, runs `codegraph init`/`status` against a real fixture. **Amendment judged legitimate** — see "Amendment Judgment" below. |
| 2 | `codegraph` completes in bash, zsh, fish; `man codegraph` renders — both generated from the binary at install time | ✓ VERIFIED | `03-EVIDENCE.md` records three separate tmux-driven interactive-shell verdicts (not a synthetic function call), each showing real subcommand names + descriptions; `man codegraph` reproduces the man-path caveat already documented. `.goreleaser.yaml`'s `generate_completions_from_executable: shells: [bash, zsh, fish]` and the `hooks.post.install`'s `man` invocation independently confirmed present and matching. |
| 3 | `hooks.post.install` runs the binary and raises independently on two assertions (man-page absence, version mismatch), demonstrated RED against a deliberately broken binary | ✓ VERIFIED | Read `.goreleaser.yaml` directly: two structurally distinct, positive (non-vacuous) assertions, confirmed live against the published tap cask (`gh api` dump matches byte-for-byte). `TestHomebrewCaskHooksHaveStructuralProperties` (re-run by me) asserts ≥2 non-comment raise statements and passes. `03-EVIDENCE.md` records two independently confirmed-applied, byte-clean-reverted mutations (a truncated/non-loadable binary → SIGKILL inside assertion one; a real signed+notarized binary under a temporary, deleted git tag reporting the wrong version → assertion two's raise, quoting both values). **Amendment (criterion 3 / BREW-05) judged legitimate** — see below. |
| 4 (split) | Half one: failure-and-recovery — structural argument only. Half two: release integrity — executed evidence | ✓ VERIFIED (as amended) | Independently re-read the pinned module source myself: `internal/pipe/publish/publish.go:59-64` — `release.Pipe{}` at line 59, `cask.Pipe{}` at line 64, comment `// brew et al use the release URL, so, they should be last` — quoted verbatim and accurately in `03-EVIDENCE.md`. `cmd/release.go:161-163`'s `skips.Publish` gating and `publish.go:83`'s `Skip()` check also verified verbatim. The half-one argument is correctly labelled throughout as *not* executed evidence, never "demonstrated"/"proven". Half two re-confirmed live: `gh run view 31424108520` → 7/7 jobs success; `v0.7.0` and `v0.8.0` both show exactly 17 assets. |
| 5 | The tap-writing token can write `seanb4t/homebrew-tap` and nothing else, demonstrated by refusal against `seanb4t/codegraph-go` | ✓ VERIFIED | `03-EVIDENCE.md` records a **positive control** (a real write+revert against the tap, `201`→`204`) prior to the negative proof (`403 "Resource not accessible by integration"` against `codegraph-go`) — exactly the shape needed to distinguish a scope refusal from a broken credential (rule `84d1gfpywd`). This is present and was not skipped. |

**Score:** 5/5 criteria verified.

### Amendment Judgment (explicit, per task instruction)

**Criterion 3 / BREW-05 amendment (plan 03-04, D-09) — LEGITIMATE.** The original wording named a cask `test:` block. I did not merely trust the claim that this doesn't exist — the reasoning is independently checkable from two authoritative sources named in the amendment: Homebrew's own Cask DSL (`Cask::DSL.instance_methods(false)` has no `test`; `brew test --help` operates only on installed *formulae*) and the pinned GoReleaser v2.17.1 `HomebrewCask` struct (`pkg/config/config.go`), which exposes no such field. This is a **falsified prerequisite**, not a lowered bar: the replacement mechanism (`hooks.post.install`'s two positive assertions) delivers the exact property the original criterion asked for ("a broken cask fails before a user hits it") and was demonstrated RED twice, for two structurally different failure classes. This is the strongest possible shape of legitimate amendment — unachievable-as-specified, replaced with a mechanism that delivers the same guarantee, and proven.

**Criterion 1 amendment (plan 03-05) — LEGITIMATE, with the residual gap correctly disclosed.** The original wording demanded verification "at least one release later, against a cask GoReleaser regenerated rather than the one hand-checked at first publish." The amendment's premise — that GoReleaser renders and pushes the cask inside the *same* automated CI run, with no hand-check step for a second release to differentiate from — is independently verifiable and I verified it myself: `cask.Pipe{}` is a member of both the main render pipeline (`internal/pipeline/pipeline.go:155`) and the publish pipeline (`internal/pipe/publish/publish.go:64`), both invoked from the one `goreleaser release` call inside `release.yml`'s single job. No human edits the cask at any point. The original worry (a curated/hand-checked tap workflow) genuinely does not describe this project's pipeline shape.

However, the amendment also carries a **maintainer-directed scope reduction** (two releases → one) that is a separate decision from the wording correction, and deserves separate scrutiny: cutting only one release means the tap-push **UPDATE** path (a second write to an already-existing `Casks/codegraph.rb`) and `brew upgrade codegraph` consuming a regenerated cask are **never exercised by this phase**. This is real, unrecovered scope — not merely a wording fix. What makes it acceptable rather than a silently dropped requirement: it is named as an "accepted gap" in three places (`03-EVIDENCE.md`'s own "Scope reduction, recorded plainly" section, the ROADMAP criterion-1 amendment text itself, and `docs/RELEASE.md`), each explicitly stating what is *not* proven and why it will surface automatically on the next real release. I judge this an honest, disclosed scope reduction rather than a goalpost moved to fit what was built — but it is real residual risk for Phase 4 and any future release, and I record it as a named, carried-forward gap below rather than silently endorsing it as fully closed.

### Finding: the sentinel-leak claim is unsupported by its own evidence (new finding, not previously recorded)

`03-EVIDENCE.md`'s "BREW-01" section states: *"Phase 4, which reads the sentinel this same hook writes, should know that a failed-then-abandoned install can leave the sentinel behind with no cask installed to explain it."*

I checked this against `.goreleaser.yaml`'s actual `hooks.post.install` script (lines 542–587) and against the same evidence file's own transcript. The hook's Ruby code runs strictly in this order: (1) assertion one — man-page-count check, raises if it fails; (2) assertion two — version-equality check, raises if it fails; (3) **only after both assertions pass** — the Phase-4 sentinel is written (`sentinel.atomic_write(...)`). Because a Ruby `raise` immediately unwinds execution, **any failure that trips BREW-05's gate (the only failure mode this phase actually demonstrated) categorically cannot reach the sentinel-write step.** The phase's own Mutation 2 transcript confirms this directly: after the install failed at assertion two, the executor's own post-failure check — `find "$(brew --prefix)" -iname "*.codegraph-brew-install*"` (03-05's "Starting state" section) — returned **no output**. No sentinel was left behind. What *was* left behind was 30 stray man pages, written by assertion one's `system_command` call as a real side effect before the fatal assertion-two raise — a different artifact than the one the quoted sentence names.

The general asymmetry claim ("hook-written files outside the Caskroom survive a failed install's rollback, because Homebrew never invokes the uninstall hook on rollback") is real and correctly demonstrated — for man pages. But the specific sentence about the *sentinel* generalizes past what was tested and is contradicted by the very evidence sitting a few lines away in the same document. A sentinel-without-install scenario would require a failure occurring *after* `hooks.post.install` completes successfully (i.e., after BREW-05's gate has already passed) but before `brew install` otherwise succeeds — a different, unexamined failure class this phase did not construct or observe.

**Why this matters for Phase 4:** ROADMAP Phase 4 explicitly says it "keys on the sentinel that hook writes" as "the most robust signal Phase 4 can key on" (Phase 3 Notes). If Phase 4's implementer reads `03-EVIDENCE.md`'s sentence literally and defensively codes around "a sentinel can exist with nothing installed" as if it were a demonstrated BREW-05-gate-adjacent failure mode, that's over-engineering against a scenario the hook's own code order rules out. Conversely, if a stray sentinel is *ever* found in the wild with no working install, that is a meaningfully different signal (something failed after the gate passed) than what the evidence file implies, and Phase 4 should not conflate the two.

**Disposition:** WARNING, not a phase-3 blocker. BREW-05's actual criterion (two positive assertions, demonstrated RED, protecting the install) is genuinely satisfied. This is a documentation-accuracy defect in `03-EVIDENCE.md` with a real (if narrow) downstream implication for Phase 4's design assumptions, not a failure of Phase 3's own success criteria.

### Requirements Coverage

| Requirement | Description | Status | Evidence |
|---|---|---|---|
| BREW-01 | Clean-machine `brew tap && brew install` → working binary | SATISFIED | Live-reconfirmed: real cold install against published `v0.8.0`, checksum-matched. REQUIREMENTS.md checkbox `[x]`, traceability row `Complete` — consistent. |
| BREW-02 | Tap published by `homebrew_casks:` on every release, cross-repo-scoped token | SATISFIED | Live-reconfirmed: single App-authored tap commit, checksum-matched assets, positive-control-backed scope refusal. REQUIREMENTS.md checkbox `[x]`, traceability `Complete` — consistent. |
| BREW-03 | Shell completions (bash/zsh/fish), generated at cask-build time | SATISFIED (bookkeeping gap — see below) | `.goreleaser.yaml`'s `generate_completions_from_executable` block confirmed; three real tmux-driven shell verdicts in `03-EVIDENCE.md`. **REQUIREMENTS.md's checkbox for BREW-03 is still `[ ]` unchecked** even though the traceability table marks it `Complete` — a self-disclosed bookkeeping gap (`03-05-SUMMARY.md` "Next Phase Readiness": *"this plan found that REQUIREMENTS.md's checkbox/traceability rows for BREW-03/04/05 had not been mechanically updated ... a pre-existing bookkeeping gap ... recorded here rather than silently fixed"*). Functionally complete; the requirements ledger's checkbox column is stale. |
| BREW-04 | Man pages installed, generated from the binary | SATISFIED (same bookkeeping gap) | `internal/cli/man.go`'s `newManCmd()`/`GenManTree` confirmed; hook's `system_command binary, args: ["man", man_dir]` confirmed; live install confirmed man pages render (with a documented man-path caveat). Checkbox `[ ]`, same disclosed gap as BREW-03. |
| BREW-05 | Post-install gate, two positive assertions, demonstrated RED | SATISFIED (same bookkeeping gap) | See criterion 3 above. Checkbox `[ ]`, same disclosed gap. |
| BREW-06 | Failed tap push leaves release intact; recovers cleanly | SATISFIED (split, as amended) | See criterion 4 above. Checkbox `[x]`, traceability `Complete` — consistent (this one's checkbox WAS updated by plan 03-05, which owned it). |

**No requirement is marked Complete without executed evidence.** All six BREW requirements have real, checkable evidence behind them. The only defect found is a **checkbox/traceability-row bookkeeping mismatch** in `REQUIREMENTS.md` for BREW-03/04/05 (checkbox unchecked, status-table row says Complete) — already self-identified and explicitly recorded by the executing agent, not discovered fresh here, and it does not misstate the underlying functional status (which the actual evidence supports).

### Anti-Patterns / Debt Markers

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any file this phase touched (`.goreleaser.yaml`'s `homebrew_casks:` block, `internal/cli/man.go`, `internal/upgrade/goreleaser_shape_test.go`, `internal/upgrade/release_workflow_shape_test.go`, `.github/workflows/release.yml`'s tap-mint step, docs). The `.goreleaser.yaml` comments are dense but are rationale/citations, not debt markers.

### Behavioral Spot-Checks (run by this verifier, not merely cited from SUMMARY.md)

| Behavior | Command | Result | Status |
|---|---|---|---|
| Cask hook has ≥2 positive raise assertions | `go test ./internal/upgrade/... -run TestHomebrewCaskHooksHaveStructuralProperties -v` | PASS | ✓ |
| Cask config shape tests (ids, url, completions shells, archives) | `go test ./internal/upgrade/... -run 'TestHomebrewCask...' -v` | 5/5 PASS | ✓ |
| App-token workflow scoping guards | `go test ./internal/upgrade/... -run 'TestAppleSecretsScopedToSingleReleaseJob$|TestHomebrewTapTokenScopedToReleaseJob' -v` | PASS | ✓ |
| Man command generates full tree, hidden, single-arg | `go test ./internal/cli/... -run TestManCmd -v` | 3/3 PASS | ✓ |
| Real published cask sha256 vs real re-downloaded assets | `curl` + `shasum -a 256` against `v0.8.0` darwin amd64/arm64 zips | Exact match to tap's declared `sha256` | ✓ |
| Tap commit provenance | `gh api repos/seanb4t/homebrew-tap/commits?path=Casks/codegraph.rb` | 1 commit, author `goreleaserbot` | ✓ |
| `release.yml` / `post-release-verify.yml` run status | `gh run view 31423733320`, `gh run view 31424108520` | Both `completed`/`success`, 7/7 jobs on the latter | ✓ |
| BREW-06 argument's source citations (pipeline ordering, skip gating) | `rg` against `$GOMODCACHE/.../goreleaser/v2@v2.17.1` | Line numbers and quoted comments match `03-EVIDENCE.md` exactly | ✓ |
| Release asset count parity v0.7.0 vs v0.8.0 | `gh release view --json assets` | 17 / 17 | ✓ |

### Deferred / Named Gaps (not blocking, explicitly disclosed by the phase itself)

| Item | Status | Where disclosed |
|---|---|---|
| GoReleaser tap-push UPDATE path (second write to existing `Casks/codegraph.rb`) unexercised | Accepted gap, deferred to next natural release | `03-EVIDENCE.md` "Scope reduction, recorded plainly"; ROADMAP criterion-1 amendment; `docs/RELEASE.md` |
| `brew upgrade codegraph` consuming a regenerated cask unexercised | Accepted gap, same as above | Same |
| BREW-06 half-one (failure-and-recovery) has zero executed evidence, by deliberate maintainer decision (D-18R) | Accepted, explicitly labelled as argument-only throughout | `03-EVIDENCE.md`, ROADMAP criterion 4 |
| Sentinel/cleanup asymmetry: successful-uninstall path is symmetric; failed-install rollback leaves man-page (not sentinel) residue | Correctly recorded for man pages; sentinel-specific claim is a finding of this verification (see above) | This report |
| README's canonical two-command install line does not itself include the `brew trust` step (documented one line below, not inline) | Minor; disclosed adjacent, not hidden | `README.md` lines 68-81 |
| REQUIREMENTS.md checkbox column stale for BREW-03/04/05 (status-table row correct) | Self-disclosed bookkeeping gap | `03-05-SUMMARY.md` "Next Phase Readiness" |
| ROADMAP.md Progress table still shows Phase 3 as "In Progress" with a blank Completed date despite 5/5 plans executed | Roadmap-owned; not edited by this verifier per task instructions | `.planning/ROADMAP.md` line 233 |

None of the above block the phase goal. They are recorded so Phase 4 planning (which depends on this phase "for acceptance") and any future maintainer pass has an accurate, non-narrative account of exactly what remains open.

### Could Not Verify

- Direct API confirmation of the GitHub App's installation-repository list (`repository_selection=selected`, scoped to `homebrew-tap` alone) could not be re-run by this verifier — it requires App-JWT authentication that a user `gh` token cannot perform, and re-minting a token was out of scope for this verification (live credential minting was performed by the phase's own plans, not repeated here). This is not a gap in the phase's evidence; it is a limit of what this verification pass could independently re-derive without re-touching credentials.
- No `brew install`/`brew uninstall` was performed by this verifier (explicitly out of bounds per this task's instructions, and the orchestrator was running a regression suite concurrently). All install/uninstall claims were corroborated via the recorded transcripts in `03-EVIDENCE.md` plus independent artifact/API checks, not re-executed live.

---

*Verified: 2026-08-10*
*Verifier: Claude (gsd-verifier)*
