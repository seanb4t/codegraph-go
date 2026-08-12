---
phase: 02-apple-signing-notarization
plan: 04
subsystem: infra
tags: [goreleaser, cosign, quill, notarization, taskfile, go-yaml, apple-notary]

# Dependency graph
requires:
  - phase: 02-apple-signing-notarization
    provides: "plan 02-02's signs:/notarize: pipeline shape (D-18) — the configuration this rehearsal target exercises"
provides:
  - "Taskfile.yml `release:rehearse-notarize` target: 11 named preconditions (host/tool/worktree guards + 5 Apple credential variables per D-09), a full both-arches rehearsal mode, and a SIGN04_MUTATE=1 single-arch mutation mode reproducing the pre-D-18-ruling signs: shape"
  - "internal/upgrade/taskfile_shape_test.go: TestRehearseNotarizeDeclaresCredentialPreconditions (+ non-vacuity companion), verified RED->GREEN by temporarily removing one precondition"
  - "Live demonstration, on a real credential-free darwin host, that the target halts BY NAME on the first missing Apple credential precondition in 0.036s — the exact property D-09 exists to guarantee"
  - "Real Apple notarize submissions, on the maintainer's own Mac with a real Developer ID Application certificate: both darwin binaries notarized, cosign subject determined cryptographically, and the D-07 mis-order mutation's relationship measured to invert"
  - ".planning/phases/02-apple-signing-notarization/02-EVIDENCE.md SIGN-01/SIGN-04/A5 sections: the maintainer's real rehearsal transcript recorded as evidence, verbatim, for plan 02-07's pre-flight comparison and for the phase's go/no-go before the irreversible tag push"
affects: [02-07]

# Actuals (#2632)
actuals:
  tokens: 16050
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Generated-copy-only mutation with a two-edit allowlist diff guard, proven non-vacuous by feeding it a deliberately incomplete (single-edit) mutated config standalone and confirming rejection before confirming the correct two-edit form passes"
    - "dist: config field (not a --dist CLI flag, which GoReleaser v2.17.1 does not have on `release`) used to give each generated config copy a distinct, isolated output directory"
    - "Baseline determinism measured by an explicit double-build into two distinct dist paths, asserted byte-identical BEFORE any cosign verify-blob comparison, so a nondeterministic build and a genuine ordering finding can never be conflated"
    - "Cosign subject determined cryptographically (verify-blob against two candidate files, exactly one must verify) rather than by re-hashing a path GoReleaser's own artifact metadata names — the metadata names the signature file, which a mutation can rewrite in place"

key-files:
  created:
    - .planning/phases/02-apple-signing-notarization/02-04-SUMMARY.md
  modified:
    - Taskfile.yml
    - internal/upgrade/taskfile_shape_test.go
    - .planning/phases/02-apple-signing-notarization/02-EVIDENCE.md

key-decisions:
  - "GoReleaser v2.17.1's `release` command has NO --dist CLI flag (confirmed against the pinned module's cmd/release.go and cmd/root.go source) — the plan's action text assumed one. Used the config's own top-level `dist:` field instead (pkg/config/config.go + internal/pipe/dist/dist.go), set via a small `inject_dist` helper that prepends `dist: <path>` to each generated config copy. This is a plan-vs-reality correction, not a scope change: the property required (each run gets a distinct, isolated output directory, no copy-aside step) is preserved exactly."
  - "Mutation mode narrows notarize.macos[0].ids to a single darwin build id (arm64) via a SEPARATE edit, applied and diffed independently BEFORE the two-edit signs:-block mutation, so the mutation-mode diff guard (which permits exactly the two D-07 edits) never has to reason about the Apple-submission-budget narrowing."
  - "Task 2 (the real Apple rehearsal) was performed on the maintainer's real Mac with a real Developer ID Application certificate and App Store Connect API key, outside this worktree, per D-08's maintainer-only local-rehearsal rule. Its verbatim transcript was supplied as the sole source of truth for this session's Task 3 and is reproduced in 02-EVIDENCE.md without inference or reconstruction of any value not present in it."
  - "No comment in Taskfile.yml or .goreleaser.yaml was found to be refuted by the real rehearsal, so none was changed. The one candidate — the sequential-submission/timeout-pending claim in .goreleaser.yaml (lines 236-249) — was corroborated for its sequential half and left explicitly OPEN for its pending/timeout half, since no submission in the real run ever returned pending or timed out to exercise that code path."
  - "A defect the real rehearsal discovered (a `[ cond ] && VAR=x` and-list as the final statement of a loop body, fatal under go-task's set -e shell interpreter but silently safe under a plain sh) was already found and fixed by the maintainer before this session, committed as 775d4df — the base commit this worktree started from. Recorded as a phase learning in 02-EVIDENCE.md rather than re-fixed."

patterns-established:
  - "Standalone proof of embedded shell-script guard logic: extract a Taskfile target's diff-guard bash logic into a throwaway script and feed it synthetic fixture pairs (rejecting and accepting) to prove non-vacuity without needing the full pipeline (GoReleaser, zig, Apple) to run."
  - "Recording a maintainer-performed, real-credential rehearsal transcript verbatim into a phase evidence file, with explicit scope-limit language (rehearsal vs. GREEN criterion) so a reader cannot later mistake a strong local result for the criterion it does not yet satisfy."

requirements-completed: [SIGN-01, SIGN-04]

duration: ~65min (this session: Task 3 recording and reconciliation; Task 1's ~50min recorded separately below; Task 2 performed by the maintainer on real hardware outside this worktree, duration not tracked by this agent)
completed: 2026-08-09
status: complete
---

# Phase 2 Plan 04: Notarize Rehearsal Target — Summary

**All three tasks are complete.** Task 1 (the guarded, maintainer-only `release:rehearse-notarize` Taskfile target and its shape test) was authored and committed in an earlier session, credential-free. Task 2 (the real Apple rehearsal, including the D-07 mis-order mutation) was performed by the maintainer on real hardware with a real Developer ID Application certificate and App Store Connect API key — its checkpoint transcript is the sole source of truth for everything recorded below. Task 3 (recording the SIGN-01/SIGN-04/A5 evidence) transcribes that transcript verbatim into `02-EVIDENCE.md` and reconciles the code against what the run actually showed.

**The ordering half of ROADMAP Phase 2 criterion 3 is now MEASURED**, not inferred from reading GoReleaser's source: the shipped `signs:` configuration verifies against the final, post-notarization binary; the deliberately mis-ordered `binary_signs:` shape verifies against the pre-sign baseline instead. Both darwin binaries were notarized by Apple for real and accepted by `spctl -a -vv -t install` (D-19's oracle). D-09's precondition discipline was demonstrated live: the target halts by name in 0.036 seconds with no credentials present.

## Performance

- **Duration:** Task 1 ~50 min (an earlier session); this session (Task 3) recording and reconciliation against the supplied Task 2 transcript
- **Tasks:** 3 of 3 completed
- **Files modified this session:** 1 (`02-EVIDENCE.md`); Task 1's earlier session modified 2 (`Taskfile.yml`, `internal/upgrade/taskfile_shape_test.go`)

## Task 1: The guarded, maintainer-only notarize rehearsal target

### What was built

`release:rehearse-notarize` in `Taskfile.yml` (~620 lines), following `release:dry-run-signed`'s established generated-config-copy mechanism and this phase's evidence-line conventions:

- **11 named preconditions**, in order: darwin host, `zig`/`syft`/`cosign`/`jq` on PATH, a clean `.goreleaser.yaml` worktree, then one precondition per Apple credential variable (`MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY`) — D-09's literal content, each naming the missing variable and pointing at `docs/RELEASE.md`.
- **Default mode**: generates a notarize-enabled config copy (throwaway local cosign key injected via the same awk anchor and additions-only diff guard `release:dry-run-signed` uses) and a notarize-DISABLED baseline template differing from it by exactly the `enabled:` predicate line (both emitted from one template, diff printed and hard-failed if it ever carries more than that line). The baseline is double-built into two distinct `dist:`-scoped output directories and asserted byte-identical (`BASELINE-NONDETERMINISTIC` on mismatch) BEFORE any cosign comparison. The real rehearsal then runs the identical `release --snapshot --skip=publish --clean` command against the notarize-enabled copy, and asserts from `dist/artifacts.json`: exactly 4 distinct Signature records, zip-content sha256 == final binary sha256, checksums-file sha256 == final binary sha256, and final sha256 != pre-sign baseline sha256, per darwin arch.
- **`SIGN04_MUTATE=1` mode**: narrows `notarize.macos[0].ids` to one darwin arch, then applies the two mandatory D-07 edits (top-level key rename `signs:` -> `binary_signs:`, plus removal of the `ids: [raw]` line) via a dedicated awk pass. A dedicated diff guard permits EXACTLY those two edits. The mutated config is structurally validated BEFORE any Apple round-trip (`MUTATION-CONFIG-INVALID`), and a zero-Signature result fails under its own distinct label (`MUTATION-PRODUCED-NO-SIGNATURE`), never conflated with `SUBJECT-INDETERMINATE`.
- **Cosign subject determined cryptographically**: for each assessed arch, `cosign verify-blob` runs against BOTH the preserved pre-sign baseline and the final binary; exactly one must verify, or the run hard-fails under `SUBJECT-INDETERMINATE`.
- `codesign -dvv` + a local-only `spctl -a -vv -t install` (D-19's oracle) against a synthetic-quarantine copy of each assessed binary.
- `NOTARIZE-EVIDENCE`/`SIGN04-EVIDENCE` lines (schema=1, fixed field order, emitted on both the pass and fail path), including the committed `.goreleaser.yaml`'s own sha256.
- Closes by asserting the committed `.goreleaser.yaml` is still byte-unchanged.

`TestRehearseNotarizeDeclaresCredentialPreconditions` (+ `_MissingTargetIsError` non-vacuity companion) added to `internal/upgrade/taskfile_shape_test.go`.

### Verification performed (all credential-free, no Apple round-trip)

1. `task --list-all` parses the entire Taskfile cleanly.
2. `go test ./internal/upgrade/ -run 'TestRehearseNotarizeDeclaresCredentialPreconditions|TestVerifyGatekeeperDeclaresNamedPreconditions' -v` — PASS (4/4 subtests).
3. RED->GREEN proof of non-vacuity: temporarily removed the `MACOS_NOTARY_KEY_ID` precondition, re-ran the test — FAILED for the expected reason, reverted, re-ran — PASS again.
4. `go vet ./internal/upgrade/` and `gofmt -l internal/upgrade/taskfile_shape_test.go` — both clean.
5. `task test:unit` — full suite, PASS.
6. Standalone mutation-mode diff guard proof: extracted the diff-guard bash logic into a throwaway script and ran it against a single-edit (pre-fix shape) fixture — REJECTED — and the correct two-edit form — PASSED.
7. D-09 halt-by-name demonstration on a credential-free host: `time task release:rehearse-notarize` with all five `MACOS_*` variables unset halted at exit 201 in 0.036 seconds, naming `MACOS_SIGN_P12` first.

## Task 2: Rehearse against Apple on the maintainer's Mac, and run the mis-order mutation

**Performed by the maintainer on real hardware, outside this worktree — not attempted by this agent.** D-08 makes this step maintainer-only and local-only by design; no CI or agent-driven rehearsal is a substitute. The full verbatim transcript (host details, both darwin arches' evidence lines, the D-07 mutation diff, the secret-leak spot check, and the maintainer's acknowledgement of what this rehearsal structurally cannot cover) was supplied to this session as the sole source of truth for Task 3, and is transcribed in full in `02-EVIDENCE.md`. See that file's `## SIGN-01`, `## SIGN-04`, and `## Assumption A5` sections for the complete record — nothing is summarized or paraphrased away from the original values here.

Headline results, for orientation only (the evidence file is authoritative):

- Both darwin arches (`arm64`, `amd64`) were notarized by Apple for real, with a genuine `Developer ID Application: Sean Brandt (8D762W58T4)` certificate, and accepted by `spctl -a -vv -t install` (exit 0, `source=Notarized Developer ID`).
- The pre-sign and final sha256 values differ per arch — the measurement that converts "quill rewrites the Mach-O" from inferred to observed.
- Under the shipped configuration, `cosign verify-blob` verifies the FINAL binary and fails against the pre-sign baseline. Under the D-07 mutation, that relationship INVERTS.
- The secret-leak spot check (a deliberately wrong `MACOS_SIGN_PASSWORD`) found zero certificate/key material in the failure output.
- `git diff --stat -- .goreleaser.yaml` was empty after every run.

## Task 3: Record the SIGN-01 rehearsal and the SIGN-04 ordering measurement

### What was built

`.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` gained three new sections, extending plan 02-01's evidence-line schema and heading conventions:

- **`## SIGN-01 — local notarize rehearsal (D-08)`**: host/certificate details, the D-09 precondition timing (0.036s), the full `BASELINE-DETERMINISM-OK`/`REHEARSAL-CODESIGN-DVV`/`REHEARSAL-SPCTL`/`NOTARIZE-EVIDENCE`/`SIGN04-EVIDENCE` transcript for the shipped configuration on both darwin arches, the committed `.goreleaser.yaml` sha256, the snapshot-mode naming shape stated as expected, the sequential Apple-submission timing (1m16s/49s/48s observed), the pending/timeout question left explicitly OPEN (no submission exercised that path), the D-03 entitlements verdict (HOLDS, with its scope limited to "Apple accepted it" rather than "it runs correctly"), and an explicit statement that this is a rehearsal observation, not the phase's GREEN criterion.
- **`## SIGN-04 — ordering, measured (D-07)`**: the converged (`mutate=0`) and inverted (`mutate=1`) hash/verification sets side by side per platform, the verbatim two-edit mutation diff, the `SIGN04-EVIDENCE` line for the mutation, assumption A3's verdict (SUPPORTED for every pipe the rehearsal exercised, explicitly not closed past `--skip=publish` and carried to plan 02-07 in that narrower form), the non-reproducible-signature finding (Apple's embedded trusted timestamp defeats final-binary reproducibility, with its consequence for any future darwin reproducibility leg), and an explicit statement that no permanent ordering regression test was added.
- **`## Assumption A5 — secret material in error paths`**: the redacted bad-password failure output, a RESOLVED verdict (no leak), and the resulting decision not to carry a masking requirement into plan 02-06.

### Reconciliation against code

No comment in `Taskfile.yml`'s `release:rehearse-notarize` target, and no comment in `.goreleaser.yaml`'s `notarize:`/`signs:` blocks, was refuted by the real run. `.goreleaser.yaml`'s existing sequential-submission/timeout-pending comment was corroborated for its sequential half (measured: 1m16s+49s, then 48s) and left explicitly OPEN for its pending/timeout half, since no submission in any of the four runs performed ever returned pending or timed out — the code path that comment describes was never exercised. **No code was changed by this task**, and this is recorded in `02-EVIDENCE.md` rather than silently doing nothing.

### Verification performed

1. `go test ./internal/upgrade/ -run 'TestRehearseNotarizeDeclaresCredentialPreconditions' -v` — PASS (both subtests).
2. `rg -c 'SIGN04-EVIDENCE' 02-EVIDENCE.md` = 5 (3 evidence lines + 2 prose mentions; the plan's `>= 3` acceptance criterion counts occurrences and is satisfied either way).
3. `rg -c 'NOTARIZE-EVIDENCE' 02-EVIDENCE.md` = 3 (2 evidence lines + 1 prose mention; `>= 2` satisfied).
4. `rg -c 'Team Identifier|TeamIdentifier' 02-EVIDENCE.md` = 3 (2 `codesign` observations + 1 prose mention; `>= 2` satisfied).
5. `git diff --stat -- .goreleaser.yaml` — empty.
6. `task test:unit` — one flaky failure on the first run (`TestFrozenTranscriptsMatch/toolslist-repeat` in `test/wireoracle`, a request-id ordering mismatch under full-suite parallel load — already documented elsewhere in `Taskfile.yml` as a pre-existing flake class, line 90). Re-ran the single test in isolation — PASS (all 17 subtests). Re-ran the full `task test:unit` — PASS, clean. This failure is unrelated to this task's doc-only change (no Go source was touched) and is out of scope per the deviation rules' scope boundary; not fixed, only confirmed as pre-existing and non-blocking.

## Task Commits

1. **Task 1: The guarded, maintainer-only notarize rehearsal target** - `a5aea0b` (feat) — earlier session
2. **Task 2: Rehearse against Apple, run the mis-order mutation** — performed by the maintainer directly, no code commit from this task (evidence-only; the one code fix the run surfaced, the and-list defect, was committed as `775d4df` before this worktree was created and is this worktree's base commit)
3. **Task 3: Record the SIGN-01/SIGN-04/A5 evidence** - `c202513` (docs) — this session

Plan-metadata commit (this SUMMARY, STATE.md, ROADMAP.md) is the orchestrator's responsibility per the worktree contract and is not made by this agent.

## Files Created/Modified

- `Taskfile.yml` — `release:rehearse-notarize` target (~620 lines) — Task 1, earlier session
- `internal/upgrade/taskfile_shape_test.go` — `TestRehearseNotarizeDeclaresCredentialPreconditions` + non-vacuity companion — Task 1, earlier session
- `.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` — three new sections (SIGN-01, SIGN-04, Assumption A5) — Task 3, this session

## Deviations from Plan

### Auto-fixed Issues (Task 1, earlier session)

**1. [Rule 1 - Bug] Plan action text assumed a `--dist` CLI flag that does not exist in GoReleaser v2.17.1**
- **Found during:** Drafting the target's baseline double-build and generated-config mechanism
- **Issue:** The plan's action text repeatedly instructs "an explicit per-run `--dist` pointing into the run-scoped temp tree." Checked against the pinned module's `cmd/release.go` and `cmd/root.go` source — no such flag exists on the `release` command in this GoReleaser version.
- **Fix:** Used the config's own top-level `dist:` field instead, via a small `inject_dist` helper. The required property (each run gets a distinct, isolated output directory, no copy-aside step) is preserved exactly; only the mechanism changed.
- **Files modified:** `Taskfile.yml`
- **Committed in:** `a5aea0b`

**2. [Rule 1 - Bug] Own comment quoting a Go-template field literally contained `{{ }}`**
- **Found during:** First `task --list-all` validation pass of the drafted target
- **Fix:** Reworded to avoid literal braces (Task's `text/template` pass renders comments too).
- **Files modified:** `Taskfile.yml`
- **Committed in:** `a5aea0b`

**3. [Rule 1 - Bug] `${var@Q}` bash 4.4+ parameter transformation used in an error message**
- **Fix:** Replaced with plain bracket interpolation — no exotic shell feature dependency under go-task's `mvdan.cc/sh` interpreter.
- **Files modified:** `Taskfile.yml`
- **Committed in:** `a5aea0b`

**4. [Rule 1 - Bug] `find`-based checksums-file discovery violated the "assert from artifacts.json, never a filename glob" discipline**
- **Fix:** Replaced with a `jq` lookup for the `Checksum`-typed record in `dist/artifacts.json`.
- **Files modified:** `Taskfile.yml`
- **Committed in:** `a5aea0b`

### Task 3 (this session)

**None.** No comment in `Taskfile.yml` or `.goreleaser.yaml` was found to be refuted by the real rehearsal run (see "Reconciliation against code" above), so no Rule 1/2/3 fix was applicable. The instructed pending/timeout claim was left explicitly OPEN rather than restated as settled — recording an honest gap is not a deviation, it is the task's own required behavior.

---

**Total deviations:** 4 auto-fixed in Task 1 (all Rule 1 — bugs found and fixed during drafting, before that commit). None in Task 3.
**Impact on plan:** All four Task 1 fixes were necessary for the target to be correct or to parse at all; no scope creep. Task 3 required no code changes.

## Issues Encountered

- A pre-existing, documented parallel-load test flake (`TestFrozenTranscriptsMatch/toolslist-repeat`) surfaced on the first `task test:unit` run this session and was confirmed non-blocking (isolated re-run: PASS; full-suite re-run: PASS). Not caused by this task's doc-only change; not fixed, per the scope boundary.

## Known Stubs

None. The `apple_status` evidence field (`notarized` on `spctl` exit 0, `rejected-or-unnotarized` on exit 3) was flagged in Task 1's session as a proxy pending confirmation against a real transcript. Task 2's real run recorded `apple_status=notarized` directly from the successful rehearsal — the proxy behaved as intended and needs no further distinguishing against a genuine Apple API status string for this phase's purposes.

## Threat Flags

None — this plan operates entirely within the `<threat_model>` T-02-11/T-02-12/T-02-13 dispositions already declared in `02-04-PLAN.md`. T-02-11 (secret material in error paths) is now CLOSED per Assumption A5's RESOLVED verdict, recorded in `02-EVIDENCE.md`. T-02-13 (tampering via the mutation experiment) held throughout: `git diff --stat -- .goreleaser.yaml` was empty after every run, including the mutation.

## Next Phase Readiness — READY

This plan is complete. SIGN-01 and SIGN-04 requirements are satisfied: the notarize rehearsal mechanism is built, guarded, and has been exercised against real Apple credentials with all measured relationships recorded in `02-EVIDENCE.md`. Assumption A3 is carried to plan 02-07 in its narrower, honestly-scoped form (supported through `--skip=publish`, not yet measured past it — no pipe currently exists in that gap, but the property itself is unmeasured there). The maintainer has explicitly acknowledged that the SLSA attestation leg of ROADMAP criterion 3 cannot be exercised by any local rehearsal and closes for the first time only on the irreversible release in plan 02-07.

---
*Phase: 02-apple-signing-notarization*
*Completed: 2026-08-09*

## Self-Check: PASSED

- FOUND: `.planning/phases/02-apple-signing-notarization/02-04-SUMMARY.md`
- FOUND: `.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md`
- FOUND commit `a5aea0b` (Task 1)
- FOUND commit `c202513` (Task 3)
- FOUND commit `775d4df` (and-list fix, this worktree's base commit, referenced as pre-existing in Task 2's transcript)
