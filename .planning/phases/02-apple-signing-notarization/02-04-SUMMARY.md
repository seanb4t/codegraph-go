---
phase: 02-apple-signing-notarization
plan: 04
subsystem: infra
tags: [goreleaser, cosign, quill, notarization, taskfile, go-yaml]

# Dependency graph
requires:
  - phase: 02-apple-signing-notarization
    provides: "plan 02-02's signs:/notarize: pipeline shape (D-18) — the configuration this rehearsal target exercises"
provides:
  - "Taskfile.yml `release:rehearse-notarize` target: 11 named preconditions (host/tool/worktree guards + 5 Apple credential variables per D-09), a full both-arches rehearsal mode, and a SIGN04_MUTATE=1 single-arch mutation mode reproducing the pre-D-18-ruling signs: shape"
  - "internal/upgrade/taskfile_shape_test.go: TestRehearseNotarizeDeclaresCredentialPreconditions (+ non-vacuity companion), verified RED->GREEN by temporarily removing one precondition"
  - "Live demonstration, on a real credential-free darwin host, that the target halts BY NAME on the first missing Apple credential precondition in 0.036s — the exact property D-09 exists to guarantee"
affects: [02-04-Task-2, 02-04-Task-3, 02-07]

# Actuals (#2632)
actuals:
  tokens: 10950
  tasks: 1
  commits: 1

tech-stack:
  added: []
  patterns:
    - "Generated-copy-only mutation with a two-edit allowlist diff guard, proven non-vacuous by feeding it a deliberately incomplete (single-edit) mutated config standalone and confirming rejection before confirming the correct two-edit form passes"
    - "dist: config field (not a --dist CLI flag, which GoReleaser v2.17.1 does not have on `release`) used to give each generated config copy a distinct, isolated output directory"
    - "Baseline determinism measured by an explicit double-build into two distinct dist paths, asserted byte-identical BEFORE any cosign verify-blob comparison, so a nondeterministic build and a genuine ordering finding can never be conflated"

key-files:
  created:
    - .planning/phases/02-apple-signing-notarization/02-04-SUMMARY.md
  modified:
    - Taskfile.yml
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "GoReleaser v2.17.1's `release` command has NO --dist CLI flag (confirmed against the pinned module's cmd/release.go and cmd/root.go source) — the plan's action text assumed one. Used the config's own top-level `dist:` field instead (pkg/config/config.go + internal/pipe/dist/dist.go), set via a small `inject_dist` helper that prepends `dist: <path>` to each generated config copy. This is a plan-vs-reality correction, not a scope change: the property required (each run gets a distinct, isolated output directory, no copy-aside step) is preserved exactly."
  - "Mutation mode narrows notarize.macos[0].ids to a single darwin build id (arm64) via a SEPARATE edit, applied and diffed independently BEFORE the two-edit signs:-block mutation, so the mutation-mode diff guard (which permits exactly the two D-07 edits) never has to reason about the Apple-submission-budget narrowing."
  - "Did not attempt to execute the rehearsal end-to-end (baseline double-build, main run, Apple submission) in this session. Task 1's own `<done>` criterion is explicit: 'the rehearsal target exists, refuses to start without credentials, and asserts its results from artifact metadata rather than exit codes. No Apple call has been made yet.' Static/structural verification (rg-based acceptance checks, `task --list-all` parse check, the two required Go tests, a standalone extraction-and-proof of the mutation diff guard's logic, and a live credential-free precondition-halt timing demonstration) fully satisfies that bar without needing zig/syft cross-builds or Apple credentials."

patterns-established:
  - "Standalone proof of embedded shell-script guard logic: extract a Taskfile target's diff-guard bash logic into a throwaway script and feed it synthetic fixture pairs (rejecting and accepting) to prove non-vacuity without needing the full pipeline (GoReleaser, zig, Apple) to run."

requirements-completed: []

duration: ~50min
completed: 2026-08-09
status: halted
---

# Phase 2 Plan 04: Notarize Rehearsal Target — Task 1 Summary (HALTED)

**Task 1 (the guarded, maintainer-only `release:rehearse-notarize` Taskfile target and its shape test) is done and committed. Task 2 (the real Apple rehearsal) is blocked on absent Developer ID Application credentials on this host and was correctly NOT attempted. Task 3 (recording the SIGN-01/SIGN-04 evidence) is blocked transitively on Task 2's checkpoint output, which does not exist.**

This plan is **NOT complete**. This SUMMARY documents a deliberately scoped, partial execution decided by the orchestrator before dispatch — see `.planning/phases/02-apple-signing-notarization/02-04-PLAN.md` Tasks 2 and 3 for what remains.

## Performance

- **Duration:** ~50 min of active agent work (Task 1 authoring, verification, and the standalone mutation-guard proof)
- **Tasks:** 1 of 3 completed (`type="auto"`); Task 2 (`checkpoint:human-verify`) and Task 3 (`type="auto"`, depends on Task 2's output) not attempted
- **Files modified:** 2

## Why this is halted, not complete

The blocker, established by the orchestrator before dispatch and reconfirmed here:

- `env | grep MACOS_` returns nothing on this host — none of `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY` are set.
- `security find-identity -v -p codesigning` returns exactly one identity: an Apple Configurator device-provisioning certificate, not a Developer ID Application certificate. Gatekeeper's `-t install` assessment (D-19's oracle) accepts only `source=Notarized Developer ID` — this identity cannot produce an accepted binary regardless of pipeline correctness.
- `xcrun notarytool` is available, but there is no App Store Connect API key to feed it.

Per the plan's own Task 2 checkpoint, this is a **maintainer-only, real-hardware, real-Apple-account** step. No workaround (self-signed cert, adhoc signing, mocked notary response) was attempted or is acceptable — that would manufacture exactly the false-verification artifact D-19/this entire phase exists to eliminate.

## Task 1: The guarded, maintainer-only notarize rehearsal target

### What was built

`release:rehearse-notarize` in `Taskfile.yml` (~620 lines), following `release:dry-run-signed`'s established generated-config-copy mechanism and this phase's evidence-line conventions:

- **11 named preconditions**, in order: darwin host, `zig`/`syft`/`cosign`/`jq` on PATH, a clean `.goreleaser.yaml` worktree, then one precondition per Apple credential variable (`MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY`) — D-09's literal content, each naming the missing variable and pointing at `docs/RELEASE.md`.
- **Default mode**: generates a notarize-enabled config copy (throwaway local cosign key injected via the same awk anchor and additions-only diff guard `release:dry-run-signed` uses) and a notarize-DISABLED baseline template differing from it by exactly the `enabled:` predicate line (both emitted from one template, diff printed and hard-failed if it ever carries more than that line). The baseline is double-built into two distinct `dist:`-scoped output directories and asserted byte-identical (`BASELINE-NONDETERMINISTIC` on mismatch) BEFORE any cosign comparison — because this repository's own reproducibility gate has no darwin leg and a non-blocking arm64 leg, so this double-build is the only evidence for darwin goreleaser determinism. The real rehearsal then runs the identical `release --snapshot --skip=publish --clean` command against the notarize-enabled copy, and asserts from `dist/artifacts.json` (never a filename glob, never a bare exit code): exactly 4 distinct Signature records, zip-content sha256 == final binary sha256, checksums-file sha256 == final binary sha256, and final sha256 != pre-sign baseline sha256, per darwin arch.
- **`SIGN04_MUTATE=1` mode**: narrows `notarize.macos[0].ids` to one darwin arch (Apple-submission budget — a second real Apple submission during a mutation run would buy no extra evidence for a per-pipe mechanism), then applies the two mandatory D-07 edits (top-level key rename `signs:` -> `binary_signs:`, plus removal of the `ids: [raw]` line) via a dedicated awk pass. A dedicated diff guard permits EXACTLY those two edits (proven non-vacuous — see below). The mutated config is structurally validated BEFORE any Apple round-trip (`MUTATION-CONFIG-INVALID`), and a zero-Signature result after the mutated run fails under its own distinct label (`MUTATION-PRODUCED-NO-SIGNATURE`), never conflated with `SUBJECT-INDETERMINATE`.
- **Cosign subject determined cryptographically** (never by re-hashing a path): for each assessed arch, `cosign verify-blob` runs against BOTH the preserved pre-sign baseline and the final binary; exactly one must verify, or the run hard-fails under `SUBJECT-INDETERMINATE` (a distinct failure from `BASELINE-NONDETERMINISTIC`, which runs first and independently, so a nondeterministic build is always excluded as the cause before this check can fire).
- `codesign -dvv` + a local-only `spctl -a -vv -t install` (D-19's oracle, never `-t exec`) against a synthetic-quarantine copy of each assessed binary — deliberately not `verify:gatekeeper`, whose contract is a published, tag-resolved asset.
- `NOTARIZE-EVIDENCE`/`SIGN04-EVIDENCE` lines (schema=1, fixed field order, emitted on both the pass and fail path), including the committed `.goreleaser.yaml`'s own sha256 so plan 02-07's pre-flight config-equality check has a value to compare against.
- Closes by asserting the committed `.goreleaser.yaml` is still byte-unchanged.

`TestRehearseNotarizeDeclaresCredentialPreconditions` (+ `_MissingTargetIsError` non-vacuity companion) added to `internal/upgrade/taskfile_shape_test.go`, reusing plan 02-01's real-YAML-decoder structs (`gatekeeperTaskfileRoot`/`gatekeeperTaskYAML`/`gatekeeperPrecondition`, which are generic to any task's `preconditions:` list, not gatekeeper-specific).

### Verification performed (all credential-free, no Apple round-trip)

1. `task --list-all` parses the entire Taskfile cleanly — no Go-template rendering error, confirming no stray `{{ }}` anywhere in the new target's `cmds:` string or comments (checked explicitly; one was caught and fixed during drafting).
2. `go test ./internal/upgrade/ -run 'TestRehearseNotarizeDeclaresCredentialPreconditions|TestVerifyGatekeeperDeclaresNamedPreconditions' -v` — the plan's specified `<verify>` command — **PASS** (4/4 subtests).
3. **RED->GREEN proof of non-vacuity**: temporarily removed the `MACOS_NOTARY_KEY_ID` precondition from `Taskfile.yml`, re-ran the test — **FAILED** for the expected reason (`declares no precondition naming "MACOS_NOTARY_KEY_ID"`), reverted, re-ran — **PASS** again. `git diff --stat` confirmed a clean revert.
4. `go vet ./internal/upgrade/` and `gofmt -l internal/upgrade/taskfile_shape_test.go` — both clean.
5. `task test:unit` — full suite, **PASS** (every package `ok`, none skipped or failing).
6. **Standalone mutation-mode diff guard proof** (acceptance criterion requiring the guard be shown to REJECT an incomplete mutation and PASS the correct one): extracted the exact diff-guard bash logic into a throwaway script (`/tmp/mutation-guard-proof/guard-test.sh`) and ran it against two synthetic fixture pairs — a single-edit (key-rename-only, the previous revision's known-wrong shape) and the correct two-edit form. Result:
   ```
   === single-edit (pre-fix shape, key rename ONLY) ===
   REJECT: removed=1 added=1 (want removed=2 added=1)
   REJECT: missing removal of '    ids: [raw]' line
   RESULT: single-edit (pre-fix shape, key rename ONLY): GUARD REJECTED (bad=1)

   === two-edit (correct D-07 mutation) ===
   RESULT: two-edit (correct D-07 mutation): GUARD PASSED (bad=0)
   ```
7. **D-09 halt-by-name demonstration on this credential-free host** (the scope-restriction's requested negative control): with all five `MACOS_*` variables absent and every other precondition (darwin host, zig, syft, cosign, jq, clean worktree) satisfied on this host, ran `task release:rehearse-notarize` under `time`:
   ```
   $ time task release:rehearse-notarize
   task: MACOS_SIGN_P12 is not set. release:rehearse-notarize is MAINTAINER-ONLY and
   requires a real Developer ID Application certificate (base64-encoded .p12, or a
   filesystem path — quill accepts either). See docs/RELEASE.md.
   task: Failed to run task "release:rehearse-notarize": task: precondition not met
   task release:rehearse-notarize  0.02s user 0.01s system 99% cpu 0.036 total
   ```
   Exit code 201. Halted BY NAME on the first missing credential precondition (`MACOS_SIGN_P12`, matching the declared order) in **0.036 seconds** — before any build work, any zig/syft/cosign invocation beyond the presence checks, and any network round-trip. This is the exact property D-09 exists to guarantee, and it had never been demonstrated on a genuinely credential-absent host before this run.

### Acceptance criteria not fully machine-checked in this session

The plan's acceptance criteria list includes several items whose full verification requires actually running the rehearsal against real Apple credentials (e.g., the live 4-signature/zip/checksums byte-identity chain, the live `SUBJECT-INDETERMINATE`/cosign-subject-inversion measurement, the live snapshot-naming-shape observation). These are Task 2's job by design (`<done>`: "No Apple call has been made yet"). What this session verified is the **declared shape**: every required label (`BASELINE-NONDETERMINISTIC`, `SUBJECT-INDETERMINATE`, `MUTATION-CONFIG-INVALID`, `MUTATION-PRODUCED-NO-SIGNATURE`, `NOTARIZE-EVIDENCE`, `SIGN04-EVIDENCE`) is present, correctly ordered relative to the Apple-round-trip and `cosign verify-blob` boundaries (confirmed via `rg -n` line-number comparison), and reachable on a failure path — per the plan's own division of labor between Task 1 (author + assert declared shape) and Task 2 (execute against real Apple credentials).

## Task Commits

1. **Task 1: The guarded, maintainer-only notarize rehearsal target** - `a5aea0b` (feat)

No plan-metadata commit follows, because this plan is not complete — STATE.md/ROADMAP.md updates are the orchestrator's responsibility per the worktree contract, and this SUMMARY itself will be committed by the orchestrator's own process, not by this agent (worktree mode, per `<parallel_execution>`).

## Files Created/Modified

- `Taskfile.yml` — new `release:rehearse-notarize` target (~620 lines)
- `internal/upgrade/taskfile_shape_test.go` — `TestRehearseNotarizeDeclaresCredentialPreconditions` + `_MissingTargetIsError` non-vacuity companion

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Plan action text assumed a `--dist` CLI flag that does not exist in GoReleaser v2.17.1**
- **Found during:** Drafting the target's baseline double-build and generated-config mechanism
- **Issue:** The plan's action text repeatedly instructs "an explicit per-run `--dist` pointing into the run-scoped temp tree." Checked against the pinned module's `cmd/release.go` and `cmd/root.go` source (`rg -n '"dist"|--dist|distDir|opts.dist'`) — no such flag exists on the `release` command in this GoReleaser version.
- **Fix:** Used the config's own top-level `dist:` field instead (`pkg/config/config.go`, `internal/pipe/dist/dist.go` — confirmed it defaults to `"dist"` and accepts any path), via a small `inject_dist` helper that prepends `dist: <path>` to each generated config copy. The required property — each run gets a distinct, isolated output directory, with no copy-aside step — is preserved exactly; only the mechanism changed.
- **Files modified:** `Taskfile.yml` (within the new target)
- **Verification:** `task --list-all` parses cleanly; `rg -c 'dist'` within the target shows the two distinct baseline `dist:` values plus the main run's.
- **Committed in:** `a5aea0b` (part of Task 1's commit — caught and corrected before that commit, not a separate follow-up)

**2. [Rule 1 - Bug] Own comment quoting a Go-template field literally contained `{{ }}`**
- **Found during:** First `task --list-all` validation pass of the drafted target
- **Issue:** A comment explaining that Apple credentials reach quill "through the process environment via the committed config's own `{{ .Env.* }}` templates" contained the literal double-brace sequence — exactly the class of bug 02-01 self-caught and this repo's own standing rule warns against (comments are rendered by Task's `text/template` pass too).
- **Fix:** Reworded to "the committed config's own Env-bound Go-template fields, unchanged" — no literal braces.
- **Files modified:** `Taskfile.yml`
- **Verification:** `rg -n '\{\{|\}\}'` over the drafted target returns zero matches; `task --list-all` parses the whole file cleanly.
- **Committed in:** `a5aea0b` (caught during drafting, before the commit)

**3. [Rule 1 - Bug] `${var@Q}` bash 4.4+ parameter transformation used in an error message**
- **Found during:** Self-review for shell-portability under go-task's `mvdan.cc/sh` interpreter
- **Issue:** One error message used `${argline@Q}` (bash's quote-transformation operator) to render a loop variable — an exotic bash feature with uncertain support in go-task's POSIX-plus-extensions shell interpreter.
- **Fix:** Replaced with plain bracket interpolation (`[${argline}]`) — equally readable, no exotic shell feature dependency.
- **Files modified:** `Taskfile.yml`
- **Verification:** `task --list-all` parses cleanly.
- **Committed in:** `a5aea0b`

**4. [Rule 1 - Bug] `find`-based checksums-file discovery violated the "assert from artifacts.json, never a filename glob" discipline**
- **Found during:** Self-review against this repo's own standing convention, stated repeatedly elsewhere in `Taskfile.yml`
- **Issue:** The first draft located the checksums file via `find "${MAIN_DIST}" -maxdepth 1 -name 'codegraph_*_checksums.txt'` — a filename glob, the exact pattern this repo's own comments elsewhere explicitly reject as insufficient evidence.
- **Fix:** Replaced with a `jq` lookup for the `Checksum`-typed record in `dist/artifacts.json`, consistent with every other artifact resolution in the target.
- **Files modified:** `Taskfile.yml`
- **Verification:** `rg -n 'artifacts.json'` within the target shows the Checksum-typed lookup.
- **Committed in:** `a5aea0b`

---

**Total deviations:** 4 auto-fixed (all Rule 1 — bugs found and fixed during drafting, before the commit)
**Impact on plan:** All four fixes were necessary for the target to be correct or to parse at all. No scope creep; deviation 1 is the only one with any plan-text implication, and it preserves the exact property the plan asked for via GoReleaser's actual v2.17.1 mechanism rather than an assumed CLI flag.

## Issues Encountered

None beyond the deviations above.

## Known Stubs

None. The `apple_status` evidence field is a best-effort proxy derived from the local `spctl` exit status (`notarized` on exit 0, `rejected-or-unnotarized` on exit 3) rather than a direct read of Apple's notarization API response, because no live Apple submission occurred in this session to observe. This is not a stub — it is documented target behavior — but Task 3's evidence-recording step should confirm against Task 2's real transcript whether this proxy value ever needs distinguishing from a genuine Apple `Accepted`/`Invalid` status string.

## Threat Flags

None — this task operates entirely within the `<threat_model>` T-02-11/T-02-12/T-02-13 dispositions already declared in `02-04-PLAN.md`; no new security-relevant surface was introduced outside that register. Credentials are never written into any generated config file (confirmed: `rg` over the target shows Apple credentials referenced only via the committed config's own `.Env.*` templates, never injected as literal values).

## Next Phase Readiness — BLOCKED

**This plan cannot proceed to Task 2 or Task 3 without maintainer action.**

- **Blocked on:** a real Apple Developer ID Application certificate (with private key) and an App Store Connect API key, exported as the five `MACOS_*` environment variables, on a real macOS host (D-08: this cannot be rehearsed in CI).
- **When unblocked:** re-run `task release:rehearse-notarize` (checkpoint step 1: precondition proof with variables unset — already demonstrated above and reusable), then with variables exported (checkpoint step 2: the real rehearsal, both arches), then `SIGN04_MUTATE=1 task release:rehearse-notarize` (checkpoint step 3: the D-07 mutation, single arch), then the secret-leak spot check (step 4) and cleanliness confirmation (step 5), per `02-04-PLAN.md`'s Task 2 `<how-to-verify>`.
- **No workaround exists or was attempted.** Self-signed/adhoc signing or a mocked notary response would manufacture exactly the false-verification artifact this entire phase exists to eliminate — explicitly forbidden by the scope restriction governing this run.
- Task 3 (recording SIGN-01/SIGN-04 evidence in `02-EVIDENCE.md`) is transitively blocked: its own `<action>` states Task 2's checkpoint output is "the sole source of truth for every value recorded here; nothing may be inferred or reconstructed." No placeholder/TBD sections were added to `02-EVIDENCE.md` — an absent section is correct; a placeholder that looks like evidence would be worse.

---
*Phase: 02-apple-signing-notarization*
*Halted: 2026-08-09 (Task 1 of 3 complete; Task 2 blocked on absent Apple Developer ID credentials)*
