---
phase: 01-corpus-selection-by-measurement
plan: 04
subsystem: testing
tags: [corpora, taskfile, ci-cache, git, supply-chain, fixt-01, fixt-02]

# Dependency graph
requires: []
provides:
  - "internal/corpora package: Entry/Manifest/Load/Validate, strict SHA/repo allowlists, collision-free Entry.Dir, CorpusRoot (XDG formula), LockedEntries"
  - "corpora/manifest.json: sole pin authority for Phase 1's corpora, 9 live-resolved candidates including rejected apache/arrow (D-09)"
  - "tools/corpora: -mode root / -mode entries (-locked), the single resolution path bash and future CI read"
  - "Taskfile.yml targets: corpora:fetch, corpora:fetch-one, corpora:assert-one, corpora:assert — shallow-fetch at pinned SHA, four-part content-integrity check, atomic claim + safe staging"
affects: [02-golden-reauthoring, 03-non-vacuity-mutation-proof, 06-corpus-lock, 07-ci-corpora-wiring]

# Actuals (#2632)
actuals:
  tokens: 10712
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Manifest-anchored shell interpolation: every value that reaches a git subprocess is validated twice — once in Go (internal/corpora.Validate), once independently in the Taskfile bash at the actual interpolation point"
    - "Four-part content-integrity check (real .git dir + HEAD==SHA + empty porcelain status + resolvable HEAD^{tree}) as the trust boundary for a fetched corpus, replacing the weaker HEAD-only pattern in tools/bench/runner/main.go:482"
    - "Claim (atomic mkdir) -> stage (mktemp -d under destination's parent) -> check (staged tree) -> promote (mv onto a guaranteed-absent path) for race-safe concurrent fetches"

key-files:
  created:
    - internal/corpora/manifest.go
    - internal/corpora/manifest_test.go
    - corpora/manifest.json
    - tools/corpora/main.go
  modified:
    - Taskfile.yml

key-decisions:
  - "internal/corpora is a separate package from tools/bench/realcorpus, not an extension of it — realcorpus performs no network I/O and deliberately carries a BSD-3-Clause entry that this phase's MIT/Apache-2.0 bar must reject (recorded in the plan objective and in both packages' doc comments)."
  - "The four-part integrity check is defined exactly once, in corpora:assert-one, and invoked by corpora:fetch-one and corpora:assert via `task <target>` subprocess calls rather than duplicated across Taskfile cmds blocks — Taskfile has no cross-task bash function sharing, so this is the DRY mechanism available."
  - "corpora:assert-one supports CORPUS_DIR_OVERRIDE so corpora:fetch-one can reuse the identical check body against a staging directory (pre-promotion) as well as the manifest-resolved final destination."
  - "apache/arrow is retained in the manifest, unlocked, with a rejection note — D-09 requires rejected candidates be recorded, not deleted. JamesNK/Newtonsoft.Json and serilog/serilog were seeded as the C# candidates arrow's rejection required (D-10's whole-repo-pin rule makes indexing all of arrow impractical for this spike)."
  - "The claim-then-check-existence race (a full fetch by another process landing between the pre-claim fast-path check and this process's mkdir claim) is closed by re-checking destination existence a second time immediately after claim acquisition, before staging — otherwise a losing process could still reach `mv` and nest a corpus inside itself."

patterns-established:
  - "Pattern: shell-side allowlist validation via `case \"$VAR\" in *[!ALLOWED-CHARS]*) ... ;; esac` — negated character class, not substring search, every expansion quoted — the last line of defence at a git-subprocess interpolation point."
  - "Pattern: Taskfile DRY via `task <target>` subprocess calls from within another target's bash block, when native `- task:` cmd entries can't express the needed if/else branching."

requirements-completed: [FIXT-02]

coverage:
  - id: D1
    description: "internal/corpora.Validate strictly rejects malformed SHA, malformed/hostile repo (9+ shell-metacharacter payloads), and unknown license (including BSD-3-Clause) before any value reaches a shell"
    requirement: "FIXT-02"
    verification:
      - kind: unit
        ref: "internal/corpora/manifest_test.go#TestManifestRejectsMalformedEntry"
        status: pass
      - kind: unit
        ref: "internal/corpora/manifest_test.go#TestManifestRejectsShellMetacharacters"
        status: pass
      - kind: unit
        ref: "internal/corpora/manifest_test.go#TestManifestRejectsUnknownLicense"
        status: pass
    human_judgment: false
  - id: D2
    description: "Entry.Dir is collision-free by construction (SHA-256 digest over canonical repo string) and embeds the pinned SHA"
    requirement: "FIXT-02"
    verification:
      - kind: unit
        ref: "internal/corpora/manifest_test.go#TestEntryDirIsCollisionFree"
        status: pass
      - kind: unit
        ref: "internal/corpora/manifest_test.go#TestEntryDirEmbedsPinnedSHA"
        status: pass
    human_judgment: false
  - id: D3
    description: "corpora/manifest.json seeds exactly the 9 candidate repositories with live-resolved MIT/Apache-2.0 licenses and 40-hex pinned SHAs; apache/arrow present unlocked with a rejection note per D-09"
    requirement: "FIXT-02"
    verification:
      - kind: other
        ref: "go run ./tools/corpora -mode entries | python3 set-equality + SHA-format + dir-uniqueness check (01-04-PLAN.md Task 2 <verify>)"
        status: pass
    human_judgment: false
  - id: D4
    description: "task corpora:fetch-one fetches at a shallow depth into a collision-free out-of-tree destination and is idempotent (second run confirms via corpora:assert-one, no re-fetch, identical git TREE OBJECT and clean status both runs)"
    requirement: "FIXT-02"
    verification:
      - kind: manual_procedural
        ref: "Live run against gohugoio/hugo: fetched, tree 8969ef3e..., re-ran fetch-one — identical tree object, clean status both times (session transcript, this plan's execution)"
        status: pass
    human_judgment: false
  - id: D5
    description: "corpora:assert-one's four-part check rejects a tampered tracked file at the correct HEAD (content check, not commit-identity check)"
    requirement: "FIXT-02"
    verification:
      - kind: manual_procedural
        ref: "Live run: appended a line to .circleci/config.yml in the fetched hugo corpus, corpora:assert-one failed part 3/4, restored byte-clean via git checkout --"
        status: pass
    human_judgment: false
  - id: D6
    description: "Two concurrent corpora:fetch-one invocations for the same entry leave exactly one destination with no nested copy and no leftover .lock/tmp.* staging directory"
    requirement: "FIXT-02"
    verification:
      - kind: manual_procedural
        ref: "Live run: two backgrounded fetch-one calls for JamesNK/Newtonsoft.Json — one succeeded, one exited non-zero naming the held claim; final directory listing showed exactly one destination, no nesting, no lock/tmp leftovers, corpora:assert-one passed"
        status: pass
    human_judgment: false
  - id: D7
    description: "A hostile CORPUS_REPO (containing a semicolon) is rejected before any git interpolation and creates nothing under the corpus root"
    requirement: "FIXT-02"
    verification:
      - kind: manual_procedural
        ref: "Live run: CORPUS_REPO='evil/repo;id' rejected at the shell allowlist check; corpus root directory listing unchanged"
        status: pass
    human_judgment: false
  - id: D8
    description: "corpora:fetch and corpora:assert exit non-zero and name the empty locked set rather than silently no-op, since nothing is locked at this plan's completion"
    requirement: "FIXT-02"
    verification:
      - kind: manual_procedural
        ref: "task corpora:fetch and task corpora:assert both exited 1 with an ::error:: line naming the empty locked set"
        status: pass
    human_judgment: false

# Metrics
duration: 40min
completed: 2026-08-14
status: complete
---

# Phase 1 Plan 4: Corpora Manifest & Reproducible Pinned Fetch Summary

**Strictly-validated corpora manifest, live-resolved 9-candidate seed set (including a recorded-but-rejected apache/arrow), and a claim-stage-check-promote Taskfile fetch pipeline whose four-part integrity check catches a tampered tracked file that a HEAD-only check would miss.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-08-14T10:50:00-04:00 (approx.)
- **Completed:** 2026-08-14T11:30:20-04:00
- **Tasks:** 3
- **Files modified:** 5 (4 created, 1 modified)

## Accomplishments

- `internal/corpora`: typed `Entry`/`Manifest`, `Load`/`Validate` with strict SHA (40-lowercase-hex) and repo (single-slash `[A-Za-z0-9._-]`) allowlists, sentinel errors `ErrInvalidSHA`/`ErrInvalidRepo`, collision-free `Entry.Dir` (SHA-256 digest over the canonical repo string), and a literal-XDG `CorpusRoot` — deliberately a separate package from `tools/bench/realcorpus` (different resolution model, narrower licence bar, different lifecycle — all four reasons recorded in code comments).
- `corpora/manifest.json`: 9 entries, every license and 40-hex commit SHA resolved LIVE via the GitHub API in this session (never recalled from memory) — `gohugoio/hugo`, `nestjs/nest`, `google/guava`, `JamesNK/Newtonsoft.Json`, `serilog/serilog`, `tiangolo/fastapi`, `pydantic/pydantic`, `psf/requests`, and `apache/arrow` (retained unlocked with a rejection note per D-09). Nothing is locked yet — Plan 01-06 flips flags on the set the measurement justifies.
- `tools/corpora`: `-mode root` (prints `corpora.CorpusRoot`) and `-mode entries` (loads+validates the manifest, prints repo/sha/slug/dir JSON, `-locked` to restrict to the locked subset) — the single resolution path both the Taskfile bash and a future CI job read.
- `Taskfile.yml`: `corpora:assert-one` (the four-part integrity check, defined once), `corpora:fetch-one` (claim→stage→check→promote, idempotent, race-safe), `corpora:assert` and `corpora:fetch` (whole-locked-set positive assertions that refuse loudly on an empty locked set).

## Task Commits

Each task was committed atomically:

1. **Task 1: The typed manifest — schema, strict validation, collision-free paths** - `50b68dd` (feat)
2. **Task 2: Seed the manifest with live-verified candidates, including the rejected one** - `1a61500` (feat)
3. **Task 3: The fetch targets — content-integrity checks, safe staging, loud failure** - `7c118c2` (feat)

_No TDD RED/GREEN split commits were needed beyond Task 1's tdd="true" flow, which landed as a single commit since tests were authored alongside the implementation and verified failing-then-passing via the mutation-proof step documented below._

## Files Created/Modified

- `internal/corpora/manifest.go` - typed manifest schema, `Load`/`Validate`, `Entry.Slug`/`Dir`, `CorpusRoot`, `LockedEntries`
- `internal/corpora/manifest_test.go` - round-trip, malformed-entry, shell-metacharacter table (10 payloads), unknown-license (incl. BSD-3-Clause), duplicate-repo, collision-free `Dir`, SHA-embedding, XDG-resolution, and locked-filter tests
- `corpora/manifest.json` - the sole pin authority for this phase's corpora (D-09)
- `tools/corpora/main.go` - `-mode root` / `-mode entries -locked`
- `Taskfile.yml` - `corpora:assert-one`, `corpora:fetch-one`, `corpora:assert`, `corpora:fetch`

## Decisions Made

- **`internal/corpora` stays separate from `tools/bench/realcorpus`.** Four reasons recorded in the plan objective and mirrored in `internal/corpora/manifest.go`'s package doc: opposite resolution model (no-network-I/O discovery vs. this package's fetch-driving role), conflicting licence policy (realcorpus deliberately carries BSD-3-Clause), realcorpus is Phase-6-owned and about to be dismantled, and different lifecycle (compile-time Go literal vs. bash/CI-readable JSON).
- **The four-part integrity check lives in exactly one place** (`corpora:assert-one`), invoked by `corpora:fetch-one` and `corpora:assert` via `task <target>` subprocess calls rather than duplicated across Taskfile `cmds:` blocks. Taskfile has no native cross-task bash-function sharing, so this is the mechanism that keeps the check single-sourced while still allowing `corpora:fetch-one`'s if/else branching (which native `- task:` cmd entries can't express).
- **`CORPUS_DIR_OVERRIDE`** lets `corpora:assert-one` check an arbitrary staging directory (not just the manifest-resolved final destination), so `corpora:fetch-one` can run the identical check body against the staged tree before promotion without a second implementation.
- **Claim-then-check-existence race closed explicitly.** Between the pre-claim fast-path check and this process's `mkdir` claim, another process could complete an entire fetch. `corpora:fetch-one` re-checks destination existence immediately after acquiring the claim, before staging, so a losing process can never reach `mv` and nest a corpus inside itself.
- **`apache/arrow` retained, unlocked, with a rejection note** (D-09) — a whole-repo pin at the arrow monorepo's scale is impractical for this spike (D-10), and `JamesNK/Newtonsoft.Json` was sought and confirmed live as the dedicated C# replacement candidate.

## Deviations from Plan

None — plan executed exactly as written. No Rule 1-4 auto-fixes were needed; every acceptance criterion in the plan's three tasks was verified directly (unit tests, `go vet`, live `task` runs against a real fetched corpus including tamper and concurrency scenarios) before committing.

## Issues Encountered

None. The live GitHub API calls, real shallow git fetches (gohugoio/hugo, JamesNK/Newtonsoft.Json), and the concurrent-fetch rehearsal all succeeded on the first attempt; the mutation-proof step (permissive SHA regex → red → byte-clean revert) and the tamper-detection rehearsal (append line → red → `git checkout --` restore) both behaved exactly as designed.

## User Setup Required

None - no external service configuration required. `gh auth status` was already authenticated in this environment, satisfying Task 2's `<precondition>`.

## Next Phase Readiness

- `corpora/manifest.json` is live and validated; Plan 01-06 can lock a subset once the measurement instrument (a separate plan in this phase's wave) records real per-kind coverage.
- `task corpora:fetch` / `task corpora:assert` are ready for Plan 01-06 to call against a populated locked set — both currently refuse loudly (by design) since nothing is locked yet.
- No CI wiring yet (deliberately out of scope — Plan 01-07 owns `actions/cache` and the workflow job); this plan's Taskfile targets are the sole definition those future CI steps will invoke via `task <target>`, per this repo's `TestWorkflowRunBodiesInvokeTask` discipline.
- `tools/bench/realcorpus` reconciliation is explicitly deferred to Phase 6 (BENCH-01/02), as recorded in this plan's objective — not a Phase 1 gap.

## Self-Check: PASSED

- FOUND: internal/corpora/manifest.go
- FOUND: internal/corpora/manifest_test.go
- FOUND: corpora/manifest.json
- FOUND: tools/corpora/main.go
- FOUND: commit 50b68dd (Task 1)
- FOUND: commit 1a61500 (Task 2)
- FOUND: commit 7c118c2 (Task 3)

---
*Phase: 01-corpus-selection-by-measurement*
*Completed: 2026-08-14*
