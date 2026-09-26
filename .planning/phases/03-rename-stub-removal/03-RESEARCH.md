# Phase 3: Rename-Stub Removal - Research

**Researched:** 2026-09-26
**Domain:** Go/Cobra CLI command removal; census-instrument reuse; GSD phase-close capability observation (changie fragments); changie version derivation
**Confidence:** HIGH

## Summary

This phase deletes two hidden Cobra rename stubs (`query`, `unlock`) that v0.14.0 left behind for exactly one minor release, and — as a side effect of being the milestone's first user-visible change — becomes the first real firing of the Phase-2 `changie-fragments` capability (CAP-05). Almost every question this phase raises was answerable by **running the real thing** rather than reading source, and this session did so in a disposable `git worktree` (`git worktree add --detach`, removed afterward; `git status --porcelain` confirmed empty throughout and at the end). All four orchestrator-assigned verification items (D-01, D-03, D-02, D-08, D-12, TDD pattern, release-please scope) were confirmed with live command output, not inference.

Two load-bearing environmental findings surfaced that are **not** in CONTEXT.md and matter for the plan: (1) this machine's ambient Go toolchain is **1.27.1**, one minor ahead of `go.mod`'s bare `go 1.26.6` directive with no `toolchain` line — because `GOTOOLCHAIN=auto` only *upgrades* to match a newer `go` line, never downgrades, every `go build`/`go test`/`task build`/`task test:unit` invocation on this machine **fails to compile** (`cockroachdb/swiss`: `undefined: hashFn`, `undefined: fastrand64` — a Go-1.27 runtime-linkname break) unless explicitly run as `GOTOOLCHAIN=go1.26.6 go ...`; only 4 of the Taskfile's targets already pin this (`docs:cli`, `docs:cli:drift`, the proto tasks, the tmux test), so any new verify command this phase's plan writes must pin it explicitly too. (2) The v0.14.0 Phase 3 SUMMARY records, as **established, repeatedly-documented repo precedent**, that `gsd_run check tdd-red-evidence` has **no Go test support at all** (it is TAP/Node-only) and returns `INVALID_RED`/`zero_tests_discovered` against every real Go RED — this is not a defect to route around, it is the expected outcome, and RED evidence for this phase must be the pasted verbatim `--- FAIL:` transcript plus the `test(03-NN):`→`feat(cli)!:` commit sequence, exactly as engram rule `x1cjy9vyhq` and five-plus prior phases already establish.

**Primary recommendation:** Delete the stubs and their registration exactly as D-01–D-06 specify, rewrite `test/integration/renamed_stubs_test.go` in place as the RED-first guard, run the v0.14.0 census instrument byte-for-byte with its positive control, and treat the `feat(cli)!:` commit itself as the only thing that makes CAP-05 observable — do not write a fragment by hand unless the capability demonstrably skips or errors.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `query`/`unlock` command removal | CLI / Backend (`internal/cli`) | — | Commands are registered in one place, `root.go`'s `AddCommand` list; no web/API/UI tier is involved (`internal/cli/root.go:126-132` `[VERIFIED: internal/cli/root.go:126-132]`) |
| CLI reference & allowlist consistency | Build/CI tooling (`tools/clidoc`, `Taskfile.yml`) | CLI / Backend | The generator walks the live Cobra tree built by `internal/cli`; `docs:cli`/`docs:cli:drift` are Taskfile-owned (`Taskfile.yml:482-536` `[VERIFIED: Taskfile.yml:482-536]`) |
| Census / mutation-log evidence | Static-analysis tooling (`rg` invocation, no runtime component) | — | Pure text search across the tracked working tree; not part of the shipped binary |
| Phase-close changelog fragment (CAP-05) | GSD workflow / capability tier (`verify:post` hook + bundled script) | Git/VCS (commit into `.changes/unreleased/`) | Dispatched by the GSD orchestrator's `execute-phase` workflow (`aggregate_results` step) outside this repo's own runtime; the script itself is vendored under `.gsd/capabilities/changie/...` (`[VERIFIED: .gsd/capabilities/changie/skills/changie-fragments/scripts/write-fragments.sh:1-30]`) |
| Version derivation (`changie next auto`) | Release tooling (`changie` via `task changie --`) | Git/VCS | Reads `.changes/unreleased/*.yaml` and `.changie.yaml`'s `auto` mapping; no application code involved |

## User Constraints (from CONTEXT.md)

<user_constraints>

### Locked Decisions

- **D-01:** After removal, `codegraph query …` and `codegraph unlock …` fail with cobra's own unknown-command error and the process exits 1: `unknown command "query" for "codegraph"` / `unknown command "unlock" for "codegraph"`, printed once by `cmd/codegraph/main.go`'s single error path. No `SuggestFor`, no alias, no custom hint text — a hint naming the replacement verb is a rename message in another form. Reversible (a `SuggestFor` entry is a one-line follow-up).
- **D-02:** `internal/cli/renamed_test.go` is deleted. `test/integration/renamed_stubs_test.go` is **rewritten in place** as the removal's guard (renamed at Claude's discretion, e.g. `removed_verbs_test.go`), keeping its four invocations (bare `query`, `query --json`, bare `unlock`, `unlock` with a path), each asserting exit status 1, empty stdout, and stderr equal to D-01's line exactly. Rule `84d1gfpywd`: the guard must positively assert an executed-case count of 4, failing on 0 — "did not print the rename message" is not an acceptable assertion. This is the RED-first test for the removal (rule `x1cjy9vyhq`): the `test(03-NN):` commit lands while the stubs still exist, its `--- FAIL:` transcript is pasted into the SUMMARY, and the `feat` commit that deletes the stubs follows. Rejected: deleting both tests with no replacement (SC1 would rest on a one-off manual run); a unit-only guard through `execCmd` (doesn't exercise `main.go`'s exit path).
- **D-03:** No other test or golden should change. Hidden commands are already excluded from the generated reference, completions, the man tree, `documentedByReference`, and `commandGroups`. The researcher confirms this by grepping for `query`/`unlock` as command registrations, not the unrelated MCP `"query"` argument keys in `testdata/wireoracle/**`/`test/wireoracle/**`. If any golden changes, treat it as a finding and explain it — do not regenerate blindly. `task docs:cli` must produce no diff to `docs/CLI-REFERENCE.md` (SC2). `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` must be green after the two allowlist lines are removed (the allowlist's own grammar: "an entry matching nothing FAILS the guard" makes leaving the lines in place go RED — the removal's built-in second guard).
- **D-04:** The stub removal is one commit, `feat(cli)!:`, with a `BREAKING CHANGE:` footer naming both removed commands and their replacements (`search --full`, `daemon unlock`). The `!` is load-bearing — it is what the capability's kind rule (Phase 2 D-06) classifies as `Breaking`. Never `[ci skip]` (rule `f18zrdsgx5`). Pipeline commits may skip signing only as rule `xmz3xknbj0` allows. Reversible.
- **D-05:** Stale comments describing the stubs are updated in the same `feat` commit, not left to rot: `root.go:14-15`, `:46`, and `:99-100` (the "hidden rename stubs (renamed.go) … for one release" prose), plus any comment elsewhere found by the census or a grep for `renamed.go`. `CHANGELOG.md` and `.changes/v*.md` are tool-owned and never edited, even where they say "removed in v0.15.0".
- **D-06:** The v0.14.0 census instrument is re-run verbatim, byte-for-byte:
  ```
  rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
  ```
  The positive control runs first: the untracked `.github/__census_control__.md` with the same three lines, inside `trap 'rm -f …' EXIT`, never `git add`ed — found on `:1:` and `:3:` before the clean-tree run is trusted. Pre-/post-mutation `git status --porcelain .github` gates are kept. The clean-tree run must report zero hits (today it reports exactly four: `renamed.go:10-11` and the two allowlist lines). Evidence goes into a new `03-MUTATION-LOG.md` family entry, mirroring the v0.14.0 layout. **Consequence for new text:** the rewritten guard, its comments, updated `root.go` prose, and the SUMMARY-facing commit message must not contain the two-token phrase `codegraph query`/`codegraph unlock` in any tracked file outside `.planning/` — refer to the verbs as `query`/`unlock`, or write the invocation as argv (`"query", "main"`).
  - **The exact family (a)/(c) positive-control content, reused verbatim from `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md`, is three lines:**
    ```
    run codegraph unlock now
    and also run codegraph
    query now
    ```
    (line 1 is the single-line form; lines 2–3 are the line-wrapped form — the census's `-U`/`\s+` half.)
- **D-07:** The census is VERB-09's evidence **at execution time, before the phase close**. The capability then writes a fragment into `.changes/unreleased/`, which the census does not exclude. A hit confined to the capability-written fragment file is the changelog describing the removal, not a surviving reference — the verifier records it as expected, does not fail on it. The instrument is **not** edited to exclude `.changes/`.
- **D-08:** The fragment is not pre-written in any plan. The capability writes it at the phase close, or CAP-05 is not met. The phase is executed through `/gsd-execute-phase`; its `aggregate_results` step dispatches `verify:post` before `verify_phase_goal` runs `gsd-verifier`. The verifier records the evidence in VERIFICATION.md: fragment filename under `.changes/unreleased/`, its kind (`Breaking`), its body, the commit SHA, the `Changie-Phase`/`Changie-Summaries` trailers, and proof the commit is on `gsd/v0.15.0-milestone`. No invented section outside what the verifier already writes. Reversible.
- **D-09:** The SUMMARY states the change in user terms for the capability's model-judgment half to reach `Breaking` accurately: which commands were removed, that invoking them is now an unknown-command error, and what to run instead — `search --full <term>` and `daemon unlock [path]`. Phase 2 D-06's illustrative body ("run `codegraph explore` instead") is **wrong** for this change and must not be copied.
- **D-10:** If the capability-written fragment is inaccurate (wrong replacement, wrong kind), correct it in a **separate** commit (`docs(03): correct changelog fragment`), record both SHAs in VERIFICATION.md, and log the correction as a capability-judgment finding for a later capability release. CAP-05 is still satisfied by the firing itself. Never amend/rebase the capability's commit.
- **D-11:** If the step does not fire (skip, error, not dispatched): commit a hand-written `Breaking` fragment via `task changie -- new -k Breaking -b "…"`, record it as hand-written, leave CAP-05 open for re-attempt at Phase 4's close (then Phase 5's), and quote the capability's printed skip reason verbatim in VERIFICATION.md.
- **D-12:** With the fragment present, `task changie -- next auto` is run on the milestone branch; output must be `v0.15.0` (minor bump — `Breaking` is `auto: minor` pre-1.0). `changie` is not on `PATH` in this repo — always call it through `task changie --`.

### Claude's Discretion

- How many plans to use (one plan with RED test + removal + census is expected; census/evidence work may be split).
- The new guard's filename and test function name.
- Exact wording of the updated `root.go` comments, within D-05/D-06's constraint.
- Whether `.planning/phases/999.5-remove-the-query-unlock-rename-stubs/` (only a `.gitkeep`) is left alone. Default: leave it.

### Deferred Ideas (OUT OF SCOPE)

- A `SuggestFor`/"did you mean `search --full`" hint for the removed verbs — rejected for this phase (D-01); a future CLI-ergonomics phase could revisit it.
- Capability judgment defects found at this close (D-10) go into a later `gsd-capability-changie` release, not this phase.
- Changing the capability itself (it is at v0.1.2; a defect found here goes into a follow-up capability release).
- Removing release-please (Phase 5), the fragment gate (Phase 4), any new CLI behavior beyond the removal (no aliases, no suggestions, no new hints).

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| VERB-09 | Hidden `query`/`unlock` rename stubs removed: `renamed.go`/`renamed_test.go` deleted, `newQueryCmd()`/`newUnlockCmd()` dropped from `root.go`, two allowlist lines removed, `task docs:cli` produces no reference change, census reports zero hits — recorded as this milestone's `Breaking` fragment | D-01 confirmed live in a scratch worktree: exact stderr/exit-code pairs for all four invocations (see Code Examples). D-03 confirmed live: `docs:cli:drift` byte-identical after removal (37 commands/117 flags vs. baseline 39/117+2), floors (26 cmds/50 flags/24 visible) cleared with room. Census confirmed live: 4 hits today, 0 after removal, positive control finds planted references at `:1:`/`:3:`. |
| CAP-05 | The `verify:post` step is observed firing at this phase's close: fragment in `.changes/unreleased/`, its commit on the milestone branch, dispatch recorded in VERIFICATION.md/SUMMARY | D-08 confirmed by reading `execute-phase.md`'s `aggregate_results` (dispatches `verify:post` step hooks generically, not just the secure-phase one) before `verify_phase_goal`; `--no-transition` only affects post-verification chaining (`offer-next.md`), never this dispatch. Capability script (`write-fragments.sh`) read in full: exact skip/note/error vocabulary, exact commit message and trailer shape (`Changie-Phase: <N>`, `Changie-Summaries: <ids>`), confirmed against Phase 2's own live-fired precedent (02-VERIFICATION.md, commits `9ff51d0f`/`0d2410f1`, trailers `Changie-Phase: 1`/`Changie-Summaries: 01-01,01-02`). |

</phase_requirements>

## Standard Stack

No new external packages are introduced by this phase — it is a deletion of existing internal code plus reuse of already-present tooling (`rg`, `go test`, `changie` via Taskfile, the already-installed `gsd-capability-changie`). See **Package Legitimacy Audit** below.

### Core (already present, reused unchanged)

| Tool | Version (verified this session) | Purpose | Why Standard |
|------|---------|---------|--------------|
| `spf13/cobra` | already vendored (`go.mod`) | CLI command tree; supplies the unknown-command error path this phase relies on | Already the project's CLI framework; D-01's exact stderr text is cobra's own default behavior, observed live |
| `ripgrep` (`rg`) | whatever is on `PATH` — the v0.14.0 census used the identical invocation and the same binary class | Census instrument (`codegraph\s+(query\|unlock)` word-boundary search) | The instrument is contractually reused byte-for-byte (D-06); no alternative tool is in scope |
| `changie` v1.26.0+ | invoked only via `task changie --` (not on `PATH`) `[VERIFIED: .planning/phases/02-phase-close-fragment-capability CONTEXT D-02, confirmed live via 'task changie -- next auto' → v0.15.0 this session]` | Version derivation / fragment authoring | Locked by CHG-01 (Phase 1); this phase only consumes it, never reconfigures it |
| Go toolchain | **must be pinned to `go1.26.6`** via `GOTOOLCHAIN=go1.26.6` prefix — ambient `go` on this machine is 1.27.1 `[VERIFIED: live 'go version' → go1.27.1; live 'GOTOOLCHAIN=go1.26.6 go build ./internal/cli/...' → exit 0; unprefixed 'go build ./internal/cli/...' → 'undefined: hashFn' etc. in cockroachdb/swiss]` | Compiling/testing any package that imports `internal/graphstore` (pebble→swiss transitively) | `go.mod` declares `go 1.26.6` with **no** `toolchain` line, so `GOTOOLCHAIN=auto` never downgrades a newer ambient Go — this is a pre-existing, deliberately out-of-scope gap (STATE.md's reviewed-not-folded todo "Pin the Go toolchain so Go 1.27 can build codegraph-go", score 0.6) that nonetheless blocks every unprefixed `go`/`task build`/`task test:unit` invocation in THIS session's environment |

**Installation:** None — no new dependency is added. Any executor running this phase's verify commands on a machine with an ambient Go newer than 1.26.6 MUST prefix every `go build`/`go test`/`task build`/`task test:unit`/`task test:integration` with `GOTOOLCHAIN=go1.26.6` (already the pattern used by 4 existing Taskfile targets: `docs:cli`, `docs:cli:drift`, the two proto tasks, and the tmux test target).

## Package Legitimacy Audit

**Not applicable.** This phase installs no new external packages — it deletes `internal/cli/renamed.go`/`renamed_test.go`, edits `root.go`, and removes two lines from an existing allowlist. No `npm view`/`pip index`/`cargo search` gate is required.

## Architecture Patterns

### System Architecture Diagram

```
Developer runs:  codegraph query <term>   /   codegraph unlock <path>
                         │
                         ▼
              cmd/codegraph/main.go
              calls cli.Execute()
                         │
                         ▼
        internal/cli.newRootCmd().Execute()
      (root.AddCommand list — no "query"/"unlock"
       constructors registered after this phase)
                         │
                         ▼
     cobra's own command-tree lookup fails to find
     "query" or "unlock" as a registered child of root
                         │
                         ▼
     cobra returns a "legacyArgs" error:
     `unknown command "query" for "codegraph"`
     (root.SilenceUsage=true, SilenceErrors=true ⇒
      cobra itself prints nothing)
                         │
                         ▼
     cli.Execute() returns that error to main()
                         │
                         ▼
     main.go's single error path:
     fmt.Fprintln(os.Stderr, err); os.Exit(1)
                         │
                         ▼
        stderr: `unknown command "query" for "codegraph"\n`
        exit code: 1
        stdout: empty
```

Before this phase, the flow instead reached `newQueryCmd()`/`newUnlockCmd()`'s `RunE`, which returned the two-line D-06 rename message as the error text — same single print seam in `main.go`, different error content. After this phase, the tree never contains a node named `query`/`unlock` at all, so cobra's generic unknown-command path is the only thing that can fire — this is the entire mechanism behind D-01, verified live (see Code Examples).

### Recommended Project Structure

No new directories. Files touched:
```
internal/cli/
├── renamed.go            # DELETED (the two stub constructors)
├── renamed_test.go       # DELETED (in-process stub unit tests)
├── root.go               # AddCommand list loses newQueryCmd()/newUnlockCmd();
│                         #   comments at :14-15, :46, :99-100 updated (D-05)
├── cli_reference_test.go # unchanged — its walk/floor logic already handles
│                         #   the shrink from 39→37 commands, 119→117 flags
└── testdata/
    └── cli-reference-allowlist.txt  # two "codegraph query"/"codegraph unlock" lines removed

test/integration/
└── renamed_stubs_test.go → rewritten in place as the removal's guard
    (rename optional, e.g. removed_verbs_test.go — Claude's discretion)

.planning/phases/03-rename-stub-removal/
└── 03-MUTATION-LOG.md    # NEW — mirrors v0.14.0's Family (a)/(c) layout
```

### Pattern 1: Cobra unknown-command error is silent-by-default and un-suggested here

**What:** With `SilenceUsage: true` and `SilenceErrors: true` set on `root` (already true today, `root.go:122-123`), an unmatched subcommand name produces exactly one line, `unknown command "<name>" for "codegraph"`, with **no** "Did you mean" suggestion for either `query` or `unlock` — confirmed live against typo'd names too (`quary`, `unlck` — no suggestion for either).

**When to use:** This is cobra's default behavior; no code change is needed to get it — simply removing the two `AddCommand(...)` calls and the two constructor functions is sufficient. Do not add `SuggestFor`, `DisableSuggestions: false` tuning, or an alias — D-01 forbids all three.

**Example (observed this session, byte-exact):**
```
$ codegraph query main
(stdout: empty)
(stderr): unknown command "query" for "codegraph"
(exit code): 1

$ codegraph query --json main
(stdout: empty)
(stderr): unknown command "query" for "codegraph"
(exit code): 1

$ codegraph unlock -p /tmp
(stdout: empty)
(stderr): unknown command "unlock" for "codegraph"
(exit code): 1

$ codegraph unlock -p /nonexistent
(stdout: empty)
(stderr): unknown command "unlock" for "codegraph"
(exit code): 1
```
`[VERIFIED: live run this session — see Code Examples for the exact reproduction steps and byte dump]`

### Anti-Patterns to Avoid

- **Adding a `SuggestFor`/hint mentioning the new verb name:** D-01 explicitly forbids this — "a hint that names the replacement verb is a rename message in another form." The unknown-command error must stay cobra's bare default.
- **Regenerating goldens/completions/man pages "just in case":** D-03 confirms none of them change (hidden commands were already excluded from all of them); regenerating and re-committing unchanged output is noise, and CONTEXT.md explicitly calls out treating an unexpected golden diff as a finding to explain, not to blindly accept.
- **Editing the census instrument to exclude `.changes/`:** D-07 forbids this even though the capability's own fragment will contain `codegraph query`/`codegraph unlock` text; a hit confined to that one file is expected and is not a defect in the instrument.
- **Relying on `gsd_run check tdd-red-evidence` to certify the RED commit:** established repo precedent (five-plus prior phases, most directly `.planning/milestones/v0.14.0-phases/03-verb-fold/03-02-SUMMARY.md`) shows this checker is TAP/Node-only and returns `INVALID_RED`/`zero_tests_discovered` against every genuine Go test failure — this is expected, not a sign the RED is invalid. RED must be certified by pasting the verbatim `--- FAIL:` transcript into the SUMMARY.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Detecting whether an old-verb reference survives the removal | A custom AST/grep tool, or a hand-tuned `grep -r "query\|unlock"` | The v0.14.0 census instrument, byte-for-byte (`rg -nU -w --hidden 'codegraph\s+(query\|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .`) | D-06's whole point is that the "before" and "after" counts are only comparable because the instrument itself never changed; a hand-rolled variant would silently invalidate that comparison |
| Proving a Go test genuinely went RED | `gsd_run check tdd-red-evidence` | Pasting the verbatim `--- FAIL:` transcript into the SUMMARY | The checker has zero Go support and will report `INVALID_RED` on every legitimate Go RED — treating that output as a signal would be a false negative loop, established and documented across 5+ prior phases |
| Compiling/testing this phase's changed packages | Trusting the ambient `go`/`task build`/`task test:unit` on a machine with Go > 1.26.6 | `GOTOOLCHAIN=go1.26.6 go ...` explicitly, matching the 4 Taskfile targets that already do this | `go.mod` has no `toolchain` line, so `GOTOOLCHAIN=auto` will use whatever `go` is ambient once it's ≥ the declared minimum — verified this session to fail to compile `internal/cli` on Go 1.27.1 |

**Key insight:** every "don't hand-roll" item above is really the same rule: this phase's evidence is only trustworthy if the exact tool/version/invocation that produced the *baseline* also produces the *after* measurement. Swap any of the three (census regex, RED-evidence method, or Go toolchain) and the comparison silently stops meaning what the plan claims it means.

## Runtime State Inventory

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — `.codegraph/` index storage never persists a CLI verb name as a key, collection, or ID. `[VERIFIED: grep of internal/graphstore and internal/schema shows no verb-name-keyed records; the removal touches only internal/cli/root.go and internal/cli/renamed.go]` | None |
| Live service config | None — no external service (n8n, Datadog, Tailscale, Cloudflare, etc.) is part of this project's runtime; the only "external" consumer of these verb names is documentation and this repo's own tests, both covered by the census. | None |
| OS-registered state | None — no Task Scheduler/pm2/launchd/systemd registration names or references either verb. The Homebrew cask's post-install hook invokes `codegraph man` (unrelated, confirmed unaffected — see Family (b)/D-03 note above and `.goreleaser.yaml`'s `generate_completions_from_executable` stanza, which derives content from the live binary's `Hidden` field, already excluding `query`/`unlock` today). | None |
| Secrets/env vars | None — no secret name or env var references `query`/`unlock`. `CODEGRAPH_TEST_BIN` (the integration harness's binary-override env var) is unrelated in name and purpose. | None |
| Build artifacts / installed packages | A previously installed `codegraph` binary (e.g., via Homebrew, or a stale local `go build` output) will keep printing the OLD two-line rename message until the user upgrades — this is ordinary versioned-software behavior, not stale in-repo state requiring migration. No egg-info-style build artifact in this repo carries the old verb names forward. | None — no action; document in the `Breaking` fragment's body that users must upgrade to lose the stub, which the fragment already does by describing the removal. |

**Canonical question answered:** after every file in this repo is updated, nothing outside git history still has the old verb names cached, stored, or registered — the only two files that legitimately still need to name the verbs (`internal/cli/root.go`'s updated prose, if any, and `internal/cli/testdata/cli-reference-allowlist.txt`) are being removed or edited by this very phase, and the census (D-06/D-07) is the mechanism that proves it.

## Common Pitfalls

### Pitfall 1: Trusting `gsd_run check tdd-red-evidence` for this phase's RED commit
**What goes wrong:** The plan or executor runs the checker against the `test(03-NN):` commit's `go test` output expecting a pass/fail verdict, and gets `INVALID_RED`/`zero_tests_discovered`, which reads like the RED was done wrong.
**Why it happens:** The checker parses TAP-style Node `--test` output (`# tests N` / `ok N - name` summary lines); `go test -v` output has no such lines, so the parser finds nothing regardless of whether the underlying test genuinely failed.
**How to avoid:** Treat `INVALID_RED` from this checker on a Go test as a known non-signal (per `.planning/milestones/v0.14.0-phases/03-verb-fold/03-02-SUMMARY.md` and prior phases 01, 02, 06, 08, 09, 10). Certify RED by pasting the verbatim `--- FAIL:` transcript into the SUMMARY and confirming the failure is an assertion failure (not a build error) on the named target test.
**Warning signs:** A SUMMARY or plan that cites the checker's verdict as the sole RED evidence, with no pasted transcript.

### Pitfall 2: Plain `go build`/`go test`/`task build`/`task test:unit` silently fails to compile on this machine
**What goes wrong:** `go build ./internal/cli/...` (or any command touching a package that imports `internal/graphstore`) fails with `github.com/cockroachdb/swiss: undefined: hashFn` / `undefined: fastrand64` / `undefined: getRuntimeHasher`.
**Why it happens:** The ambient `go` binary is 1.27.1; `go.mod` declares `go 1.26.6` with no `toolchain` directive, and `GOTOOLCHAIN=auto` never *downgrades* to an older toolchain than what's already installed — it only upgrades when the ambient version is *older* than declared. `cockroachdb/swiss`'s `map.go`/`options.go` use runtime-internal linknames that changed shape in Go 1.27.
**How to avoid:** Prefix every `go`/`task build`/`task test:unit`/`task test:integration` invocation touching these packages with `GOTOOLCHAIN=go1.26.6` — the cached toolchain is already present locally (`~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.darwin-arm64`) and resolves with zero network access. This mirrors the pattern 4 existing Taskfile targets already use.
**Warning signs:** A build/test failure mentioning `cockroachdb/swiss`, `hashFn`, or `fastrand64` with no other explanation.

### Pitfall 3: Assuming the flag-accounting floors (`cliReferenceMinCommands`/`cliReferenceMinFlags`/`cliReferenceMinVisibleCommands`) need bumping down after removal
**What goes wrong:** A plan author sees the walked-command count drop (39→37) and assumes the floor constants in `cli_reference_test.go` need editing.
**Why it happens:** The floors (26 commands, 50 flags, 24 visible commands) were set as a "never silently drops to zero" tripwire, not as an exact-count pin — this session's live run confirms 37 commands / 117 flags / 24 visible commands after removal, comfortably above all three floors.
**How to avoid:** Do not touch `cliReferenceMinCommands`, `cliReferenceMinFlags`, or `cliReferenceMinVisibleCommands` — they need no edit for this phase's removal. `TestEveryRegisteredFlagIsAccountedFor` and `TestEveryCommandHasGroupID` both pass unmodified once the allowlist lines are removed (confirmed live this session).
**Warning signs:** A diff to `cli_reference_test.go`'s constant block in this phase's plan — that is itself the finding to question.

## Code Examples

### Live reproduction of D-01 (byte-exact stderr/exit code)

```bash
# Performed in a disposable git worktree, never the main checkout:
# git worktree add --detach <scratch-dir> HEAD
# (edit root.go's AddCommand list to drop newQueryCmd()/newUnlockCmd();
#  rm internal/cli/renamed.go internal/cli/renamed_test.go)
GOTOOLCHAIN=go1.26.6 go build -o /tmp/codegraph-scratch-bin ./cmd/codegraph   # exit 0

/tmp/codegraph-scratch-bin query main            # exit=1, stderr: unknown command "query" for "codegraph"
/tmp/codegraph-scratch-bin query --json main      # exit=1, stderr: unknown command "query" for "codegraph"
/tmp/codegraph-scratch-bin unlock -p /tmp         # exit=1, stderr: unknown command "unlock" for "codegraph"
/tmp/codegraph-scratch-bin unlock -p /nonexistent # exit=1, stderr: unknown command "unlock" for "codegraph"

# byte dump of stderr (xxd), confirming exactly one trailing \n, no "Error: " prefix:
# 756e6b6e6f776e20636f6d6d616e6420 2271756572792220666f7220 22636f646567726170 68220a
#  u  n  k  n  o  w  n     c  o  m  m  a  n  d     "  q  u  e  r  y  "     f  o  r     "  c  o  d  e  g  r  a  p  h  "  \n

# No "Did you mean" suggestion, even for near-typos of the removed names:
/tmp/codegraph-scratch-bin quary main   # unknown command "quary" for "codegraph", exit=1
/tmp/codegraph-scratch-bin unlck /tmp   # unknown command "unlck" for "codegraph", exit=1
```
Source: this session's live shell output, in a disposable worktree removed afterward (`git worktree remove --force`), main checkout confirmed `git status --porcelain` clean before and after `[VERIFIED: live commands, this session]`.

### Live reproduction of the census instrument (D-06/D-07)

```bash
# Clean tree, stubs present — matches the "today" claim in CONTEXT.md D-06:
rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
# ./internal/cli/testdata/cli-reference-allowlist.txt:8:codegraph query	...
# ./internal/cli/testdata/cli-reference-allowlist.txt:9:codegraph unlock	...
# ./internal/cli/renamed.go:10:// ... `codegraph query <term>`
# ./internal/cli/renamed.go:11:// ... `codegraph unlock [path]` ...
# => 4 hits across 2 files

# Positive control (main checkout, untracked, never git-added, removed via trap):
cat > .github/__census_control__.md <<'EOF'
positive control line 1: codegraph query should be found
irrelevant line with no match
positive control line 3: codegraph unlock should also be found
EOF
rg -nU -w --hidden 'codegraph\s+(query|unlock)' ... | grep __census_control__
# ./.github/__census_control__.md:1:...codegraph query...
# ./.github/__census_control__.md:3:...codegraph unlock...
rm -f .github/__census_control__.md   # git status --porcelain .github -> empty, confirmed

# Scratch worktree, stubs + allowlist lines removed:
rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
# (no output; rg exit code 1 = zero matches)
```
Source: this session's live shell output `[VERIFIED: live rg invocations, this session]`. The exact positive-control content to reuse, verbatim from `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md` Family (c):
```
run codegraph unlock now
and also run codegraph
query now
```

### Live confirmation of D-03 (docs/flag-accounting unaffected)

```bash
# In the scratch worktree, after removing the stubs AND the two allowlist lines:
task docs:cli:drift
# docs:cli:drift: compared 1 generated file
# docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration

GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run 'TestEveryRegisteredFlagIsAccountedFor|TestEveryCommandHasGroupID' -v
# walked 37 commands (hidden included), inspected 117 flags:
#   113 accepted via docs/CLI-REFERENCE.md, 4 accepted via the allowlist
# --- PASS: TestEveryRegisteredFlagIsAccountedFor
# inspected 24 visible commands across 4 groups
# --- PASS: TestEveryCommandHasGroupID

# Baseline (main checkout, stubs present, for comparison):
#   walked 39 commands, inspected 119 flags, 6 accepted via the allowlist
# Floors (cli_reference_test.go): cliReferenceMinCommands=26, cliReferenceMinFlags=50,
#   cliReferenceMinVisibleCommands=24 — all cleared with room both before and after.
```
Source: live `go test -v` output, this session `[VERIFIED: internal/cli/cli_reference_test.go:33-34,300 for floor constants; live test run for counts]`.

### Live confirmation of D-12 (`changie next auto`)

```bash
# Main checkout, empty .changes/unreleased/ (only .gitkeep):
task changie -- next auto
# Error: no unreleased changes found for automatic bumping   (exit 201 via task)
task changie -- latest
# v0.14.0

# Disposable worktree, one planted Breaking fragment:
CI=true task changie -- new -k Breaking -b "Removed the hidden query and unlock rename stubs; run search --full or daemon unlock instead."
task changie -- next auto
# v0.15.0
```
Source: live `task changie --` output, this session, in a worktree removed immediately afterward `[VERIFIED: live commands, this session]`.

### The capability's exact commit shape (CAP-05 evidence to check for)

```bash
# .gsd/capabilities/changie/skills/changie-fragments/scripts/write-fragments.sh:499-518
PLAN_IDS_SORTED=$(printf '%s' "${SUMMARIES_ARG}" | tr ',' '\n' | sort | tr '\n' ',' | sed 's/,$//')
# commit message:
#   docs(<padded-phase>): add changelog fragments
#
#   Changie-Phase: <phase-number-with-leading-zero-stripped>
#   Changie-Summaries: <sorted,comma,joined,plan,ids>
#
# final stdout line on success:
#   changie-fragments: wrote <N> fragment(s) in <short-sha>: <files>
```
For phase 3, `Changie-Phase: 3` is the expected trailer value `[VERIFIED: .gsd/capabilities/changie/skills/changie-fragments/scripts/write-fragments.sh:277-278,499-518]`. This exact shape was independently observed already firing once, for Phase 1's own close, per Phase 2's verification: commits `9ff51d0f`/`0d2410f1`, trailers `Changie-Phase: 1` / `Changie-Summaries: 01-01,01-02` `[CITED: .planning/phases/02-phase-close-fragment-capability/02-VERIFICATION.md line 29]`.

### TDD commit pattern to follow (from v0.14.0 Phase 3, same milestone shape)

```
test(03-02): add failing rename-stub tests      # RED — TestQueryStub/TestUnlockStub fail
                                                 # against the still-live commands
...
feat(cli)!: fold query into search --full and unlock into daemon unlock
                                                 # GREEN — single commit, BREAKING CHANGE footer
```
Source: `git log --oneline 5d69ee2e~5..5d69ee2e` this session — three `test(03-02):` commits (`4da74784`, `4215e42f`, `f6bd1ffb`) preceded the single `feat(cli)!:` commit (`5d69ee2e`) `[VERIFIED: git log, this session]`. This phase's shape is the mirror image: the RED test proves the OLD (stub) behavior is what breaks first, then the single `feat(cli)!:` commit deletes the stubs and makes the rewritten guard pass.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `codegraph query <term>` / `codegraph unlock [path]` printing a two-line rename notice and exiting 1 | Both names removed entirely from the Cobra tree; cobra's own unknown-command error fires instead | This phase (v0.15.0, following the one-release grace period v0.14.0 D-05/D-09 promised) | Any script or muscle-memory still invoking the old verbs now sees a generic `unknown command "..." for "codegraph"` instead of a message naming the replacement — this is intentional per D-01, not a regression |

**Deprecated/outdated:** `internal/cli/renamed.go` and its two constructors are removed outright, not deprecated further — the one-release grace period is over.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `PHASE_NUMBER` passed to the `verify:post` skill dispatch for this phase resolves to `"03"` (matching `init.phase-op`'s `padded_phase`), which the script strips to `Changie-Phase: 3` | Code Examples / CAP-05 evidence | Low — even if the unpadded `"3"` is passed instead, `PHASE_STRIPPED="${PHASE#0}"` produces `"3"` either way; the trailer value is `3` under both inputs, so this assumption does not change the plan's checkable must-have |
| A2 | The Homebrew cask / `.goreleaser.yaml` completions and man-page generation are unaffected by this removal, based on the v0.14.0 Family (b) verdict (a prior phase's finding, not re-verified live this session for the `query`/`unlock` case specifically) | Runtime State Inventory | Low — the mechanism (live-binary `Hidden` field, no committed completions/man artifacts) is unchanged since v0.14.0's fold, and D-03 already establishes hidden commands are excluded from every generated artifact by construction |

**If this table is empty:** N/A — two low-risk assumptions remain, both because the phase does not close until execution and neither materially changes what the plan should specify.

## Open Questions

1. **Exact filename/test-function name for the rewritten `test/integration` guard**
   - What we know: D-02 leaves this to Claude's discretion; the CONTEXT.md example name is `removed_verbs_test.go`.
   - What's unclear: Whether the planner prefers to keep `renamed_stubs_test.go`'s filename (git-history continuity via `git mv`) or rename it.
   - Recommendation: `git mv test/integration/renamed_stubs_test.go test/integration/removed_verbs_test.go` preserves blame history through the rename and matches the CONTEXT.md-suggested name; either choice satisfies D-02.

2. **Whether the capability fires as `Breaking` or requires the D-10 correction path**
   - What we know: The capability's model-judgment half picks its own kind/body from the SUMMARY; D-09 dictates what the SUMMARY must say (which commands were removed, that they're now unknown-command errors, and the two replacement invocations) precisely so the capability's own judgment lands on `Breaking` with accurate wording.
   - What's unclear: This can only be observed at actual phase close — it is not something research (or the plan) can pre-determine.
   - Recommendation: The plan should state D-09's SUMMARY requirements as a must-have, and VERIFICATION.md must record whichever of D-08 (fired correctly), D-10 (fired, needed correction), or D-11 (skipped, hand-written fallback) actually occurred — all three are pre-specified, so no re-planning is needed regardless of outcome.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `go` (ambient) | Building/testing `internal/cli`, `cmd/codegraph` | ✓ (wrong default version) | 1.27.1 | `GOTOOLCHAIN=go1.26.6` prefix (cached locally, no network) |
| Go toolchain 1.26.6 | Matches `go.mod`'s declared `go 1.26.6`; required to avoid the `cockroachdb/swiss` linkname break | ✓ | cached at `~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.darwin-arm64` | — |
| `rg` (ripgrep) | Census instrument (D-06) | ✓ | whatever is on `PATH` (used successfully this session) | — |
| `task` (go-task) | `docs:cli`, `docs:cli:drift`, `test:integration`, `changie` targets | ✓ | resolved via `command -v task` this session | — |
| `changie` | Version derivation (D-12) — never invoked directly, only via `task changie --` | ✓ (via Taskfile wrapper) | pinned per CHG-01 (v1.26.0+) | — |
| `gh` (GitHub CLI) | Optional PR-number lookup inside `write-fragments.sh`; absence is a `note`, never a `skip` | not probed this session (not required for this phase's own verify commands) | — | Script continues without a PR number if `gh` is missing (D-13, already-decided upstream behavior) |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** The ambient Go toolchain mismatch (1.27.1 vs. declared 1.26.6) — fallback is the explicit `GOTOOLCHAIN=go1.26.6` prefix, already the established pattern in 4 Taskfile targets.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) — no separate config file |
| Config file | none — `go.mod` |
| Quick run command | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... ./test/integration/... -run 'TestQueryStub\|TestUnlockStub\|TestEveryRegisteredFlagIsAccountedFor\|TestEveryCommandHasGroupID\|TestRenamedStubs\|TestRemovedVerbs'` (adjust the last two patterns to the guard's actual final name) |
| Full suite command | `GOTOOLCHAIN=go1.26.6 task test:unit && GOTOOLCHAIN=go1.26.6 task test:integration && task docs:cli:drift` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|--------------------|-------------|
| VERB-09 (SC1) | `query`/`unlock` exit 1 as unknown commands, no rename message | integration (real spawned binary) | `GOTOOLCHAIN=go1.26.6 go test ./test/integration/... -run <rewritten-guard-name> -v` | ❌ Wave 0 — file must be rewritten from `renamed_stubs_test.go`'s current (stub-asserting) content |
| VERB-09 (SC2) | `task docs:cli` no-op; `docs:cli:drift`/`TestEveryRegisteredFlagIsAccountedFor` green after allowlist edit; census zero hits | task + unit + scripted census | `task docs:cli:drift`; `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run TestEveryRegisteredFlagIsAccountedFor -v`; `rg -nU -w --hidden 'codegraph\s+(query\|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .` (expect zero) | ✅ (`docs:cli:drift`, `TestEveryRegisteredFlagIsAccountedFor` already exist and pass on the modified tree; the census is a one-off `rg` invocation, not a persisted test file) |
| VERB-09 (SC3/SC4, CAP-05) | Capability fires at phase close; fragment/commit/trailers present; `changie next auto` answers `v0.15.0` | process/evidence-based (not a `go test` command) | `git log --grep='Changie-Phase: 3' -1 --format='%H %(trailers:key=Changie-Phase,valueonly) %(trailers:key=Changie-Summaries,valueonly)'`; `task changie -- next auto` | N/A — evidence gathered post-hoc by the verifier from the actual `execute-phase` run, not a pre-written test |

### Sampling Rate

- **Per task commit:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... ./test/integration/... -v` (targeted to the touched packages)
- **Per wave merge:** `GOTOOLCHAIN=go1.26.6 task test:unit && GOTOOLCHAIN=go1.26.6 task test:integration && task docs:cli:drift`
- **Phase gate:** Full suite green, plus the census (D-06/D-07) and `changie next auto` (D-12) evidence captured in VERIFICATION.md, before `/gsd:verify-work`.

### Wave 0 Gaps

- [ ] `test/integration/renamed_stubs_test.go` (or its renamed successor) must be **rewritten** before the removal lands — this is the RED-first guard (D-02/rule `x1cjy9vyhq`); today it asserts the OLD stub-message behavior and will itself need to go RED against the *new* expectation before the `feat(cli)!:` commit makes it GREEN.
- [ ] `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` does not yet exist — it is a phase deliverable (D-06/D-07's evidence file), not a pre-existing gap, but must be created during execution, not left to the verifier to reconstruct.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | Not touched — no auth surface in this phase |
| V3 Session Management | No | Not touched |
| V4 Access Control | No | Not touched |
| V5 Input Validation | Marginal | Cobra's own argument/flag parsing is unaffected; removing two `DisableFlagParsing: true` hidden commands *reduces* the parsed-input surface (one fewer code path that previously accepted arbitrary unvalidated args/flags without inspection) |
| V6 Cryptography | No | Not touched |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Dead/hidden command left registered indefinitely, eventually reintroducing a forgotten behavior via config drift or a bad merge | Tampering (of the command surface, not user data) | This phase's own census instrument (D-06/D-07) plus the allowlist's fail-closed grammar ("an entry matching nothing FAILS the guard") are the standing mitigations — no new control is needed, this phase exercises the existing one |
| A future contributor accidentally re-adds a `query`/`unlock` alias without realizing the rename history | Tampering | The updated `root.go` comments (D-05) and the `BREAKING CHANGE:` commit footer (D-04) are the durable record; no runtime guard is warranted for a removed dead code path |

## Sources

### Primary (HIGH confidence — live command output, this session)

- Live `git worktree add --detach` / `git build` / binary invocations against a scratch worktree with the stubs and allowlist lines removed — D-01, D-03 evidence
- Live `rg -nU -w --hidden 'codegraph\s+(query|unlock)' ...` runs (baseline, positive control, post-removal) — D-06/D-07 evidence
- Live `task changie -- next auto` / `task changie -- latest` / `CI=true task changie -- new -k Breaking ...` in a disposable worktree — D-12 evidence
- `internal/cli/root.go` (read in full, lines 1-190), `internal/cli/renamed.go`, `internal/cli/renamed_test.go`, `test/integration/renamed_stubs_test.go`, `internal/cli/cli_reference_test.go` (read in full), `internal/cli/testdata/cli-reference-allowlist.txt`, `cmd/codegraph/main.go`, `Taskfile.yml` (`docs:cli`/`docs:cli:drift`/`test:integration` sections), `.changie.yaml`, `.gsd/capabilities/changie/skills/changie-fragments/scripts/write-fragments.sh` (read in full), `~/.claude/skills/gsd-changie-fragments/SKILL.md` (read in full)
- `~/.claude/gsd-core/workflows/execute-phase.md` (`aggregate_results` lines 1081-1180, `verify_phase_goal` lines 1186+) and `~/.claude/gsd-core/references/loop-hook-dispatch.md` (full read) — D-08 dispatch-ordering evidence
- `.planning/phases/02-phase-close-fragment-capability/02-VERIFICATION.md` — CAP-05's Phase 2 skip-path precedent and the one real observed firing's exact trailer/commit shape
- `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md` (Family (a), (b), (c) sections read in full) and `03-02-SUMMARY.md` — census layout to mirror, TDD-red-evidence checker's documented Go-incompatibility, and the `test(...)`→`feat(cli)!:` commit sequencing precedent
- `git log` on this repository — confirmed commit sequence `4da74784`/`4215e42f`/`f6bd1ffb` (RED) → `5d69ee2e` (`feat(cli)!:`, GREEN) for the v0.14.0 fold
- `.github/workflows/release-please.yml` (trigger: `on: push: branches: [main]`) and `.github/workflows/ci.yml` (`test:integration` step) — release-please scope and integration-test wiring confirmation

### Secondary (MEDIUM confidence)

None used — no library documentation lookup was needed for this phase (`brave_search`/`firecrawl`/`exa_search` are all disabled in this project's `.planning/config.json`, and every question was answerable by reading this repository's own source and running its own tooling).

### Tertiary (LOW confidence)

None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; all tool versions/behavior confirmed live this session
- Architecture: HIGH — the entire removal mechanism (cobra unknown-command path) was reproduced end-to-end in a disposable worktree
- Pitfalls: HIGH — both the toolchain pitfall and the TDD-red-evidence-checker pitfall were confirmed either live (toolchain) or via repeatedly-documented repo precedent (checker)

**Research date:** 2026-09-26
**Valid until:** Effectively permanent for VERB-09's mechanics (cobra's unknown-command behavior is stable); the CAP-05 capability-firing outcome can only be confirmed at actual phase close, not by research — flagged in Open Questions
