---
phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
plan: 06
subsystem: ci
tags: [taskfile, github-actions, pnpm, corepack, sveltekit, drift-guard, supply-chain]

requires:
  - phase: 02-01
    provides: "web/ pnpm toolchain, web/embed.go's //go:embed all:build, the committed web/build/ tree this plan's marker binds"
  - phase: 02-04
    provides: "TestJSInstallPathHasNoMutableCache — the structural no-mutable-JS-cache invariant on ci.yml's test job, authored one wave before the Node setup step this plan adds"
  - phase: 02-05
    provides: "the final Phase 2 web/build/ source tree (Tailwind v4 + shadcn tokens, four nav routes, live status page) this plan's marker is computed against"
provides:
  - "task web:deps — Corepack-or-pnpm-self-managed frozen-lockfile install, reporting which resolution path it took"
  - "task web:build — pnpm build plus the four-line web/build/.build-manifest marker (source-tree digest + independent output-tree digest)"
  - "task web:build:verify — the clean-checkout rebuild proof that actually runs on every PR, scratch-directory only, never touching web/build/"
  - "task web:drift — BLD-03's two-part staleness-and-tamper guard, watched RED six ways before being trusted green"
  - "Four new ci.yml test-job steps (Set up Node, Install JS deps, SPA clean-checkout rebuild check, SPA build drift guard) — no new CI job"
affects: [02-07]

actuals:
  tokens: 25303
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Shared hashing pipeline as a Taskfile var (WEB_HASH_LIB) templated into two targets' cmds, rather than duplicating the source/output enumeration-and-hash logic — the single-definition discipline Task 2 required"
    - "Two independent digests (source-tree + output-tree, both hashed in place, never regenerated) instead of proto:drift's regenerate-and-byte-compare shape — the correction RESEARCH.md Pitfall 1 required for a build tool with non-reproducible content-hash filenames"
    - "-t/--taskfile plus -d/--dir against a scratch copy of Taskfile.yml, used to demonstrate the narrowed-enumeration RED cases without ever touching the committed Taskfile.yml mid-demonstration"

key-files:
  created:
    - web/build/.build-manifest
  modified:
    - Taskfile.yml
    - web/vite.config.ts
    - .github/workflows/ci.yml
    - web/build/** (rebuilt once, as part of writing the marker)

key-decisions:
  - "web/svelte.config.js does not exist in this toolchain (02-01-SUMMARY.md Deviation 1) — the CODEGRAPH_WEB_BUILD_DIR output-directory override lives in web/vite.config.ts's sveltekit() adapter-static options instead, exactly where the rest of the adapter config already lives."
  - "WEB_HASH_LIB is a Taskfile var (not a shell script file, not a second internal task) so web:build and web:drift template in the identical function bodies — chosen over Task's own task-composition mechanism (cmds: - task: <name>) because sharing output between tasks that way needs a file or env-var handoff, and a shared var keeps everything in one readable place with zero extra indirection."
  - "SRC_FLOOR=8 and OUT_FLOOR=3 are derived, not copied from the observed 22/25: 6 structurally-mandatory toolchain config files this toolchain version actually has (package.json, pnpm-lock.yaml, pnpm-workspace.yaml, vite.config.ts, tsconfig.json, components.json — NOT svelte.config.js) plus src/app.html and at least one route; index.html plus one immutable .js plus one immutable .css."
  - "actions/setup-node pinned to v7.0.0 (820762786026740c76f36085b0efc47a31fe5020), resolved via gh api repos/actions/setup-node/git/ref/tags/v7.0.0 — the latest release at plan time, matching this workflow's own SHA-pinning convention for every other action."
requirements-completed: []  # BLD-01 is declared by 02-01/02-03/02-04/02-06 — all four now have SUMMARYs, so requirements.ready-ids is expected to mark it complete at this plan's state-update step. BLD-03 is declared by 02-06/02-07 — 02-07 has no SUMMARY yet, so BLD-03 stays blocked per the shared-ID gate (#2388) until 02-07 finishes. Left empty here per the same contract 02-01/02-05 followed.

coverage:
  - id: D1
    description: "task web:deps resolves the pinned pnpm version reproducibly — Corepack when present, a pnpm>=10 already on PATH when Corepack is absent (this exact machine: Node 26, no Corepack) — reporting which path it took, and fails loudly with neither on PATH"
    requirement: BLD-01
    verification:
      - kind: other
        ref: "task web:deps — pnpm self-managed path taken and reported on this Corepack-less Node 26 host; web/pnpm-lock.yaml asserted byte-unchanged (cmp against a pre-install copy)"
        status: pass
      - kind: other
        ref: "env PATH=/Users/sean/.local/bin:/usr/bin:/bin task web:deps — named error 'neither corepack nor pnpm is on PATH', exit 201 (task's wrapped exit 1)"
        status: pass
    human_judgment: false
  - id: D2
    description: "task web:build writes a four-line web/build/.build-manifest binding an independent source-tree digest AND an independent output-tree digest (find over the committed web/build/, excluding the marker itself) — closing the review's HIGH finding that a source-only marker would pass a hand-edited shipped byte"
    requirement: BLD-03
    verification:
      - kind: other
        ref: "task web:build — marker is exactly 4 lines, each matching its key/shape pattern (source-files/source-sha256/output-files/output-sha256), verified with rg against the pinned regex shapes"
        status: pass
      - kind: other
        ref: "two consecutive task web:build runs with no source change: SOURCE half stable, OUTPUT half churns (Vite content-hash filename non-determinism, vitejs/vite#15555/#13071) — the exact, expected asymmetry recorded per the plan's own acceptance criterion"
        status: pass
    human_judgment: false
  - id: D3
    description: "task web:build:verify proves the pinned toolchain still produces a structurally valid build from this source on a clean runner (ROADMAP criterion 3's rebuild half, previously asserted by no gate) — scratch-directory only, never writing into web/build/"
    requirement: BLD-01
    verification:
      - kind: other
        ref: "task web:build:verify against the clean committed tree — 25 files built, non-empty index.html, 14 non-empty immutable .js, 1 non-empty immutable .css, git diff --exit-code -- web/ clean, git status --porcelain web/build empty"
        status: pass
    human_judgment: false
  - id: D4
    description: "task web:drift is a two-part staleness-and-tamper guard, watched RED six ways (stale source; tampered/added/removed output byte; narrowed enumeration per half; missing marker), each reverted byte-clean, before being trusted green"
    requirement: BLD-03
    verification:
      - kind: other
        ref: "baseline green run: 'hashed 22 source files' / 'manifested 25 output files' printed before comparison, both halves MATCH"
        status: pass
      - kind: other
        ref: "six RED transcripts captured below (Deviations/Verification Evidence section), each followed by a byte-clean git diff --exit-code web/ revert"
        status: pass
      - kind: other
        ref: "web:drift contains no cmp, no scratch-tree build, no pnpm build invocation, and does not list web:build in deps: — verified by reading the target"
        status: pass
    human_judgment: false
  - id: D5
    description: "Four new steps (Set up Node, Install JS deps, SPA clean-checkout rebuild check, SPA build drift guard) fold into the existing required test CI job — no new job, no cache on the JS install path, every run: body exactly task <target>"
    requirement: BLD-01
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/... -run 'TestWorkflowRunBodiesInvokeTask|TestRequiredCheckNamesPreserved|TestJSInstallPathHasNoMutableCache' -v — 3/3 named PASS, 0 FAIL (TestWorkflowRunStepsInvokeTaskTargets does not exist in this codebase — see Deviations)"
        status: pass
      - kind: unit
        ref: "go test ./internal/upgrade/... — full package green"
        status: pass
      - kind: other
        ref: "task lint:actions — clean; job-key count in ci.yml unchanged (10 before/after)"
        status: pass
    human_judgment: false

duration: ~1h5min
completed: 2026-08-25
status: complete
---

# Phase 2 Plan 6: SPA Toolchain, Embedded App Shell & JS Supply Chain — CI Build & Drift Guard Summary

**`task web:deps`/`web:build`/`web:build:verify`/`web:drift` and four new `ci.yml` steps close ROADMAP criterion 3's both halves: CI actually rebuilds the SPA into scratch on every PR, and a two-independent-digest guard (source tree + the committed output tree's own bytes) proves the committed `web/build/` is neither stale nor tampered — watched fail six ways before being trusted green.**

## Performance

- **Duration:** ~1h5min (three task commits span 20:25:54 -> 20:29:54 local; total wall-clock includes the RED-demonstration cycles for `web:drift` and the plan-verification runs, not reflected in the commit-to-commit gap alone)
- **Completed:** 2026-08-25T00:31:50Z
- **Tasks:** 3
- **Files:** 1 created, 4 modified (including the rebuilt `web/build/` tree)

## Accomplishments

- `task web:deps`: resolves pnpm via Corepack when present, falls back to a `pnpm` at major 10+ already on PATH when Corepack is absent — demonstrated on this exact machine (Node 26.7.0, `corepack` not on PATH), which reproduces 02-01-SUMMARY.md's own recorded finding and gives it a durable, reported fix. `pnpm install --frozen-lockfile` runs, and the lockfile is asserted byte-unchanged. Demonstrated the named failure path too: with neither `corepack` nor `pnpm` on `PATH`, the target exits with a named error rather than a confusing pnpm-not-found trace.
- `web/vite.config.ts` gained a `CODEGRAPH_WEB_BUILD_DIR` environment-variable override on `adapter-static`'s `pages`/`assets`, defaulting to the literal `'build'` — the override that lets `web:build:verify` prove buildability without ever writing into the committed tree. Placed in `vite.config.ts`, not `web/svelte.config.js` (see Deviations).
- `task web:build`: depends on `web:deps`, runs `pnpm build`, then writes `web/build/.build-manifest` — exactly four lines, two independent digests. The **source** digest hashes `git ls-files` over `web/src`, `web/static`, and six toolchain config files at their working-tree content (not the git index blob). The **output** digest hashes every file `find` locates under `web/build/`, excluding the marker's own path — so an added, removed, or edited file inside the committed build output is caught, which is exactly the bytes `//go:embed all:build` ships to every user. Both pipelines live once, in a shared `WEB_HASH_LIB` Taskfile var, sourced by both `web:build` and `web:drift` — never duplicated.
- **Observed and recorded (not a defect):** the OUTPUT digest is **not** stable across two consecutive `task web:build` runs with zero source change — Vite/Rollup's content-hash filenames genuinely differ between builds of identical source (`vitejs/vite#15555`, closed not-planned; `#13071`), exactly as RESEARCH.md Pitfall 1 predicted. The SOURCE digest, by contrast, was stable across the same two runs. This is precisely why `web:drift` must never rebuild — and reinforces why `web:build` itself must not be run gratuitously (see Issues Encountered, which hit this directly during verification).
- `task web:build:verify`: builds into a `mktemp -d` scratch directory via the new env-var override, prints the built-file count before comparing it against a derived floor, asserts a non-empty `index.html`, at least one non-empty immutable `.js`, at least one non-empty immutable `.css`, then asserts `git diff --exit-code -- web/` and `git status --porcelain web/build` are both clean — proof the scratch build never touched the committed tree.
- `task web:drift`: recomputes both digests via the same shared pipeline, prints both counts (`hashed N source files`, `manifested N output files`) unconditionally before any comparison, checks each against its own derived floor (`SRC_FLOOR=8`, `OUT_FLOOR=3` — small, stable lower bounds, never the observed 22/25), then compares all four values against the committed marker with two textually distinct mismatch messages (SOURCE-half vs OUTPUT-half). Never depends on `web:build` and never rebuilds.
- **Watched RED six ways, each reverted byte-clean** (full transcripts in Verification Evidence below):
  1. Stale source (edited `web/src/lib/utils.ts`, no rebuild) — SOURCE-half mismatch; output half unaffected.
  2. **Tampered output byte** (appended a line to a committed `_app/immutable/nodes/*.js` file) — OUTPUT-half mismatch **while the SOURCE half reported MATCH in the same run** — the direct evidence a source-only marker would have passed exactly this tamper, which is the review's HIGH finding this plan exists to close.
  3. Added output file (planted an untracked stray under `_app/immutable/chunks/`) — OUTPUT-half mismatch, manifested count 25→26.
  4. Removed output file (`web/build/robots.txt`) — OUTPUT-half mismatch, manifested count 25→24.
  5. Narrowed enumeration, both halves independently, via a scratch copy of `Taskfile.yml` (`task -t /tmp/scratch-Taskfile.yml -d "$(pwd)"`) so the committed `Taskfile.yml` was never touched mid-demonstration — SOURCE narrowed to 1 file tripped `SRC_FLOOR=8`; OUTPUT narrowed via `-maxdepth 1` tripped `OUT_FLOOR=3`.
  6. Missing marker (moved `.build-manifest` aside) — named remedy message ("run `task web:build`"), never a pass.
- Four new steps folded into `ci.yml`'s existing, already-required `test` job (no new job — D-13): `Set up Node` (`actions/setup-node@820762786026740c76f36085b0efc47a31fe5020` / `v7.0.0`, resolved via `gh api repos/actions/setup-node/git/ref/tags/v7.0.0`; `node-version: "24"`; no `cache:` input), `Install JS deps (frozen lockfile, BLD-01)` (`task web:deps`), `SPA clean-checkout rebuild check (BLD-01)` (`task web:build:verify`), `SPA build drift guard (BLD-03)` (`task web:drift`) — placed last of the two JS gates so it would catch the rebuild check ever mutating the committed tree.
- `TestJSInstallPathHasNoMutableCache` (authored in 02-04, one wave ahead of this step existing) is now enforced against the real `Set up Node` step, not just a hypothetical future one — confirmed green.
- No caching action was added for `node_modules` or the pnpm store — a deliberate omission (RESEARCH.md, D-16, this plan's own threat register T-02-06-06), now structurally enforced by 02-04's test rather than merely observed.

## Task Commits

1. **Task 1: `web:deps`, `web:build` and `web:build:verify`** — `f225aaa` `feat(02-06): web:deps, web:build, web:build:verify — reproducible install and the marker-writing build`
2. **Task 2: `web:drift` — the two-part staleness-and-tamper guard, watched RED six ways** — `d7e2174` `feat(02-06): web:drift — two-part staleness-and-tamper guard, watched RED six ways`
3. **Task 3: fold Node setup, frozen install, rebuild check and drift guard into `ci.yml`'s `test` job** — `769b0f9` `feat(02-06): fold Node setup, frozen install, rebuild check, and drift guard into ci.yml's test job`

**Plan metadata:** committed separately after this SUMMARY (see below).

_Note: this plan's tasks were `type="auto"`, not `tdd="true"` — each is a single `feat` commit, not a RED/GREEN pair, matching the plan's own frontmatter (no `type: tdd`, no per-task `tdd="true"` attribute)._

## Files Created/Modified

- `Taskfile.yml` — `WEB_HASH_LIB` var (shared hashing pipeline), `web:deps`, `web:build`, `web:build:verify`, `web:drift` targets
- `web/vite.config.ts` — `adapter-static`'s `pages`/`assets` read `CODEGRAPH_WEB_BUILD_DIR`, defaulting to `'build'`
- `web/build/.build-manifest` — the new four-line committed marker
- `web/build/**` — rebuilt once as part of writing the marker (Vite content-hash filename churn only — no behavioral change)
- `.github/workflows/ci.yml` — four new steps inside the existing `test` job

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `web/svelte.config.js` does not exist in this toolchain — the output-directory override lives in `web/vite.config.ts` instead**
- **Found during:** Task 1, step (b)
- **Issue:** The plan's own text names `web/svelte.config.js` as the file that gains the `CODEGRAPH_WEB_BUILD_DIR` override. That file does not exist in this SvelteKit toolchain version — `02-01-SUMMARY.md`'s own Deviation 1 already recorded that adapter-static's configuration has always lived inside `web/vite.config.ts`'s `sveltekit()` plugin options, because this toolchain's `sv@0.17.0` scaffold for `@sveltejs/kit ^2.63.0` no longer reads `svelte.config.js` at all.
- **Fix:** Added the `CODEGRAPH_WEB_BUILD_DIR` override to the same `adapter: adapter({ pages: ..., assets: ... })` block already inside `web/vite.config.ts`, with a comment cross-referencing this deviation and 02-01-SUMMARY.md's original one. `web_source_files()` in the new `WEB_HASH_LIB` var also omits `web/svelte.config.js` from its enumeration (a nonexistent `git ls-files` pathspec is silently a no-op, but listing a file that will never exist would read as an oversight rather than a deliberate, recorded choice) — this shifted the source floor's toolchain-config-file count from the plan's stated "seven" to six actual files.
- **Files modified:** `web/vite.config.ts`, `Taskfile.yml` (`WEB_HASH_LIB`'s `web_source_files()`, `web:drift`'s `SRC_FLOOR=8` derivation comment)
- **Verification:** `task web:build` succeeds; `web:build:verify` builds into scratch via the override and confirms the output landed there, not in `web/build/`, when the variable is set — and in `web/build/` when it is unset (the default-unchanged path)
- **Committed in:** `f225aaa`

**2. [Rule 3 - Blocking] `TestWorkflowRunStepsInvokeTaskTargets`, named in the plan's own `<verify>`/acceptance-criteria text, does not exist anywhere in this codebase**
- **Found during:** Task 3, running the plan's exact `<automated>` verify command
- **Issue:** The plan's Task 3 `<verify>` block and acceptance criteria both name four workflow-shape tests to run and each expects exactly one `--- PASS` line: `TestWorkflowRunBodiesInvokeTask`, `TestWorkflowRunStepsInvokeTaskTargets`, `TestRequiredCheckNamesPreserved`, `TestJSInstallPathHasNoMutableCache`. `git log --all -S 'TestWorkflowRunStepsInvokeTaskTargets'` across the whole repository finds this string ONLY in this phase's own planning-document commits (`docs(02): ...`) — it was never real Go code, in this plan or any prior one. The acceptance criterion's own stated purpose ("`go test -run` exits 0 when the pattern matches nothing at all — a `-run` expression naming a nonexistent test can silently contribute zero to a green run") is the exact failure this deviation is about: including a nonexistent name in the `-run` alternation does not fail the command, it just silently matches nothing for that one name while the other three still run and pass.
- **Fix:** Ran and verified the three tests that DO exist and DO enforce the properties this plan's Task 3 needs (`TestWorkflowRunBodiesInvokeTask` — every new `run:` body is exactly `task <target>`; `TestRequiredCheckNamesPreserved` — no required-status-check context was renamed; `TestJSInstallPathHasNoMutableCache` — no mutable JS-scoped cache on the new `Set up Node` step), each printing exactly one `--- PASS` and zero `--- FAIL`. Did NOT author a new, always-passing stub test merely to satisfy the literal name in the plan text — inventing a vacuous test purely to make an acceptance-criteria string match would be exactly the kind of guard rule `84d1gfpywd` forbids (a test whose only job is to exist, proving nothing).
- **Files modified:** none (verification-only; no Go test file changes belong to this deviation)
- **Verification:** `go test ./internal/upgrade/... -run 'TestWorkflowRunBodiesInvokeTask|TestRequiredCheckNamesPreserved|TestJSInstallPathHasNoMutableCache' -v` — 3/3 named `--- PASS`, 0 `--- FAIL`; full `go test ./internal/upgrade/...` also green
- **Committed in:** n/a (no code change; recorded here as a plan-authoring correction)

---

**Total deviations:** 2 auto-fixed (both Rule 3 - blocking, both plan-text/reality mismatches inherited from an already-documented prior deviation or a stale/hallucinated test name — neither is a design or architecture change).
**Impact on plan:** Both are corrections to make the plan's own text consistent with what already exists in this repository. Neither changes BLD-01's or BLD-03's actual guarantees: the output-directory override still does exactly what the plan asked, in the one place this toolchain actually reads adapter config from; the CI wiring is verified by the tests that genuinely enforce the properties in question, not a fabricated one.

## Issues Encountered

**A gratuitous re-run of `task web:build` during my own final verification pass polluted the working tree with stray untracked files, which I initially mistook for a bug in `web:build:verify`.** Re-running `task web:build` a second time (with no source change) produced new Vite content-hash filenames — expected non-determinism, already documented above — and `git checkout -- web/build` afterward restored the tracked files but left the newly-created (never-committed) files sitting on disk as untracked strays. The next `web:build:verify`/`web:drift` run then correctly reported those strays as a dirty tree / an OUTPUT-half mismatch. This was **not** a defect in either target — it was my own test procedure violating the plan's explicit instruction not to run `web:build` gratuitously. Resolved by manually removing the six stray files (never `git clean`, per this repo's destructive-git-operations prohibition) and re-verifying against the untouched committed tree, which passed cleanly. Recorded here because it is a real, reproducible demonstration of exactly the caveat the plan's own acceptance criteria warned about, not a new finding.

## Verification Evidence (plan `<verification>` block)

- `task web:deps`, `task web:build`, `task web:build:verify`, `task web:drift` all green on the clean committed tree (re-confirmed at HEAD after all three commits, without an intervening `web:build` rerun).
- `task web:drift` output at HEAD:
  ```
  web:drift: hashed 22 source files
  web:drift: manifested 25 output files
  web:drift: source half MATCH (22 files, ef7626f00d3505a48742ae38a3cb81c6e246fb535ba88ba85f3fb5bb04c4eb3e)
  web:drift: output half MATCH (25 files, ca7d9d60f2a26b863bdf952d3ee1c225acb08fcaef4ae1b2b61561a702825543)
  web:drift: PASS — hashed 22 source files, manifested 25 output files, committed web/build/ matches both digests
  ```
- **RED 1 (stale source):** `::error::web:drift: SOURCE-half mismatch — the committed build output is stale relative to its source (marker: 22 files / ef7626f0...; recomputed: 22 files / 0ddc65d4...). Run \`task web:build\` to rebuild.` — output half in the same run: `web:drift: output half MATCH (25 files, ca7d9d60...)`. Reverted via `git checkout -- web/src/lib/utils.ts`; `git diff --exit-code web/` clean after.
- **RED 2 (tampered output byte — the review's core finding):** `web:drift: source half MATCH (22 files, ef7626f0...)` immediately followed by `::error::web:drift: OUTPUT-half mismatch — the committed web/build/ bytes are NOT the ones \`task web:build\` produced: someone edited, added, or removed a file inside the committed build output (marker: 25 files / ca7d9d60...; recomputed: 25 files / 89abbb77...).` Tampered file: `web/build/_app/immutable/nodes/3.CAq2GsoK.js`. Reverted via `git checkout --`; clean after.
- **RED 3 (added output file):** `web:drift: manifested 26 output files` / `OUTPUT-half mismatch (... recomputed: 26 files / d1539f11...)`. Reverted via `rm -f` of the planted stray; clean after.
- **RED 4 (removed output file):** `web:drift: manifested 24 output files` / `OUTPUT-half mismatch (... recomputed: 24 files / e4614b3b...)`. `web/build/robots.txt` restored from a `/tmp` backup copy; clean after.
- **RED 5a (narrowed SOURCE enumeration):** `web:drift: hashed 1 source files` / `::error::web:drift: hashed only 1 source files, want at least 8 (...)`. Run via a scratch `Taskfile.yml` copy (`task -t /tmp/scratch-Taskfile.yml -d "$(pwd)"`) — the committed `Taskfile.yml` was never touched.
- **RED 5b (narrowed OUTPUT enumeration):** `web:drift: manifested 2 output files` / `::error::web:drift: manifested only 2 output files, want at least 3 (...)`. Same scratch-Taskfile mechanism.
- **RED 6 (missing marker):** `::error::web:drift: web/build/.build-manifest is missing — run \`task web:build\` to rebuild and write it. This is never a pass.` Restored via `mv` from a `/tmp` backup; clean after.
- `task web:build:verify` at HEAD: `web:build:verify: built 25 files into a scratch directory` / `index.html present and non-empty` / `14 non-empty immutable .js file(s) present` / `1 non-empty immutable .css file(s) present` / `PASS — 25 files built into scratch, structural invariants held, committed tree untouched`; `git diff --exit-code -- web/` clean; `git status --porcelain web/build` empty.
- `task lint:actions` clean; `go test ./internal/upgrade/...` green (`TestWorkflowRunBodiesInvokeTask`, `TestRequiredCheckNamesPreserved` + its `_ZeroJobsIsError` sibling, `TestJSInstallPathHasNoMutableCache` all `--- PASS`, 0 `--- FAIL` — see Deviation 2 for `TestWorkflowRunStepsInvokeTaskTargets`'s absence).
- `git diff --exit-code web/` clean after every guard run at HEAD.
- `go build ./...` clean.
- CI job-key count in `.github/workflows/ci.yml` unchanged (10 before, 10 after) — confirms no new top-level job was introduced.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `web/build/.build-manifest` now exists and is watched by `task web:drift`, which the `test` CI job runs on every PR/push from this commit onward.
- **02-07** (BLD-05/BLD-06/BLD-03) inherits `web:deps` as its install step and the same `WEB_HASH_LIB`-backed drift guard already wired into CI — no further Taskfile plumbing for the install/build/drift path should be needed there.
- **BLD-01 requirement readiness:** declared by 02-01/02-03/02-04/02-06 — all four now have SUMMARYs, so `requirements.ready-ids` is expected to mark `BLD-01` complete at this plan's state-update step.
- **BLD-03 requirement readiness:** declared by 02-06/02-07 — 02-07 has no SUMMARY yet, so `BLD-03` stays blocked per the shared-ID gate (#2388) until 02-07 finishes, even though this plan's own two-part guard is fully implemented, watched RED six ways, and green in CI.
- No blockers for 02-07.

## Self-Check

- `[ -f Taskfile.yml ]`, `[ -f web/vite.config.ts ]`, `[ -f web/build/.build-manifest ]`, `[ -f .github/workflows/ci.yml ]`, `[ -f .planning/phases/02-spa-toolchain-embedded-app-shell-js-supply-chain/02-06-SUMMARY.md ]` → all FOUND
- `git log --oneline --all | grep -q f225aaa` → FOUND
- `git log --oneline --all | grep -q d7e2174` → FOUND
- `git log --oneline --all | grep -q 769b0f9` → FOUND
- Re-ran the plan's exact `<verification>` block at HEAD (post-commit): `task web:deps`, `task web:build:verify`, `task web:drift` all green; `task lint:actions` clean; `go test ./internal/upgrade/...` green; `git diff --exit-code web/` clean; `git status --porcelain web/build` empty.

## Self-Check: PASSED

---
*Phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain*
*Plan: 06*
*Completed: 2026-08-25*
