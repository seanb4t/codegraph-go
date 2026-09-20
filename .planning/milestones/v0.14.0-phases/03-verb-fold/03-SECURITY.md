---
phase: "3"
slug: "verb-fold"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-16"
---

# Phase 3 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| user shell → `codegraph query` / `codegraph unlock` stubs | arbitrary args/flags reach `RunE` unparsed (`DisableFlagParsing: true`); the stub reads none of them, opens nothing, removes nothing, and returns one error whose text is the whole two-line rename message — `cmd/codegraph/main.go`'s single `os.Exit(1)` is the only exit path | argv only (ignored) |
| user shell → `codegraph daemon unlock [path]` | a positional path selects a `.codegraph/daemon.lock` to remove; the live-vs-stale decision stays in `internal/daemon.Unlock` (`isStale` guard, `ErrLockLive`), whose body and tests are byte-unchanged by the move | a local lock file |
| `search --full` → `internal/query.Engine` → stdout | already-validated term/kind/limit; output now includes `QualifiedName`/`Signature` (the data the removed `query` verb and the MCP `codegraph_node` tool already emitted); local index opened read-only via `query.OpenAt`, no network path | local symbol metadata |
| planted census control → working tree | an untracked `.github/__census_control__.md` exists for one shell step (EXIT trap), is proven found by the instrument, then proven absent from disk, index and history | none by construction |
| working-tree RED mutation → committed history | `renamed.go` is flipped (`Hidden: false`) only inside a scratch run for the Family (b) RED and reverted byte-clean before any `git add`; no commit in the phase carries the mutation | none by construction |
| feat! commit → release-please / CHANGELOG | one regex-conformant `feat(cli)!:` subject with a `BREAKING CHANGE:` footer is what categorises the release; the phase audit asserts exactly one such commit and no CI-skip marker in any subject or body | commit metadata |
| generated reference / allowlist → `TestEveryRegisteredFlagIsAccountedFor` + `task docs:cli:drift` | the gates decide whether a hidden-vs-visible change is caught; trusted only after Family (b) showed both RED on a one-token mutation | `docs/CLI-REFERENCE.md`, `testdata/cli-reference-allowlist.txt` |
| phase diff → MCP / wire surface | zero diff under `internal/mcp`, `testdata/wireoracle`, `test/wireoracle` across `21bb329e..HEAD`; 8 tool names unchanged; uncached wire-oracle run green | none by construction |
| `gsd-tools phase add` → `.planning/ROADMAP.md` | the Backlog row `999.5` is written only by the tool verb (additions only, one `###` heading, no version token); the parser's own `getMilestonePhaseFilter` proves the active milestone window is unchanged | planning values |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-01-01 | Repudiation | census "zero/14 hits" verdict trusted without a control | high | mitigate | planted `.github/__census_control__.md` (3 lines, EXIT-trapped) reported on `:1:`/`:2:`/`:3:` by the exact instrument before the real run; "before" 14 lines / 6 files reproduced from scratch at `21bb329e` in a detached worktree with the identical six-file set | closed |
| T-03-01-02 | Tampering | `rg` default hidden-dir skip hides `.github/`, `.claude/hooks/`, `SKILL.md` consumers | high | mitigate | `--hidden` proven load-bearing: the same invocation WITHOUT `--hidden` returned 0 hits on the planted control under `.github/`; with it, 3 | closed |
| T-03-01-03 | Tampering | planted control leaking into a commit | medium | mitigate | control absent after the trap; `git status --porcelain .github` empty; `git ls-files --error-unmatch` fails; `git log --all -- .github/__census_control__.md` = 0 commits | closed |
| T-03-01-04 | Tampering | the "before" baseline captured after an `internal/cli/` edit | medium | mitigate | `75cee732`'s parent is `21bb329e` and it touches only `03-MUTATION-LOG.md`; `git diff --quiet 21bb329e 5bc10ed8 -- internal cmd docs` holds; 0 commits in the 03-01 range touch the Go tree | closed |
| T-03-01-05 | Denial of Service | `protect-main` blocked by a `[ci skip]` marker | high | mitigate | `rg -i 'ci skip\|skip ci'` over every subject AND body in `21bb329e..HEAD` (23 commits): none; the scan proved on a planted `[ci skip]` line | closed |
| T-03-01-SC | Tampering | npm/pip/cargo installs | low | accept | R-03-02 | closed |
| T-03-02-01 | Tampering | stub `RunE` executing partial logic (engine open, lock removal) before erroring | high | mitigate | `renamed.go` (post-WR-01 `fa81672c`): imports only `fmt` + cobra; each `RunE` is a single `return fmt.Errorf(...)`; `Hidden: true` ×2, `DisableFlagParsing: true` ×2; zero hits for `query.OpenAt`, `daemon.Unlock`, `os.(Remove\|WriteFile\|ReadFile\|Open\|Create\|Stat)`, `Fprint`, `Deprecated`, `os.Exit`, `Args:`; no `PreRun`/`PersistentPreRun` hooks in `internal/cli`. Binary: dead-pid `daemon.lock` survived `unlock <dir>`; `query main -p <indexed>` → stdout 0 bytes, exit 1. `TestQueryStub`/`TestUnlockStub` (7 subtests) and `TestRenamedStubsPrintExactlyOnce` (4 subtests, real binary) `--- PASS`. The plan's literal "two `Fprintln` + one `Errorf`" is superseded by WR-01 — the current shape is strictly narrower and the intent holds | closed |
| T-03-02-02 | Tampering | unparsed args forwarded or interpreted by the stub | medium | mitigate | no `args[`/`range args`/`len(args)` in `renamed.go`; `query --json main -p <dir>` and bare `query` produce byte-identical stderr and empty stdout (binary `cmp`); integration test covers `unlock -p <dir>` and `unlock -p /nonexistent` | closed |
| T-03-02-03 | Elevation of Privilege | a second process-exit path bypassing `SilenceUsage`/`SilenceErrors` | medium | mitigate | no `os.Exit` in `renamed.go`; the only exit on the stub path is `cmd/codegraph/main.go:16`; binary `exit=1` for both stubs with/without flags (informational: `internal/graphstore/logger.go:50` `Fatalf` has a pre-existing `os.Exit(1)` unreachable from the stubs and untouched by the phase) | closed |
| T-03-02-04 | Denial of Service | `daemon unlock` removing a live daemon's lock | high | mitigate | whole-phase diff of `internal/daemon/lock.go` = one message string + doc comment; `isStale` and `Unlock`'s `!isStale → ErrLockLive` guard before `os.Remove` untouched; `lock_test.go` diff empty (`TestUnlockStaleOnly` `--- PASS`); old `unlock.go` `RunE` @`21bb329e` vs `daemon.go` byte-identical after indent normalisation; `daemon.Unlock(codegraphDir)` ×1; 0 `PersistentFlags` in `daemon.go`. Binary: live-pid lock → `daemon: lock is held by a live process … daemon still running`, exit 1, file left in place; dead-pid lock → `removed stale lock`; `-p` → `unknown shorthand flag` | closed |
| T-03-02-05 | Information Disclosure | `search --full` prints `Signature`/`QualifiedName` | low | accept | R-03-01 | closed |
| T-03-02-06 | Repudiation | message-site tests flipped without a RED, or new tests passing vacuously | medium | mitigate | RED commits `f6bd1ffb`/`4215e42f`/`4da74784` are ancestors of `5d69ee2e`; file sets test-only (+3-line `renderFullLine` placeholder in `f6bd1ffb`); the feat commit lists test files as `M` only. RED reproduced from scratch in a worktree at `4da74784`: `go build ./...` OK, all 7 target tests `--- FAIL` on assertions (`unknown flag: --full`, `unknown command "unlock"`, `expected a non-nil error, got nil`), 0 build-error lines. GREEN at HEAD asserted by count: 12/12 top-level `--- PASS`, 0 `FAIL` | closed |
| T-03-02-07 | Tampering | malformed `feat!` subject/footer mis-categorising the release or breaking `pr-title.yml` | medium | mitigate | `5d69ee2e` subject matches `pr-title.yml:47`'s regex; body line 20 `BREAKING CHANGE:` names `search --full`, `daemon unlock`, `removed in v0.15.0`; the footer is the last paragraph before the trailer; exactly 1 `feat(` and 1 `!:` subject in `21bb329e..HEAD`; only that body contains `BREAKING CHANGE` | closed |
| T-03-02-08 | Tampering | blanket regenerate of `docs/CLI-REFERENCE.md` hiding an unintended surface change | high | mitigate | every changed line of `git show 5d69ee2e -- docs/CLI-REFERENCE.md` (8 hunks, +28/−49) classified into the three groups (query/unlock bullets + sections gone; search Short/`--full`/`-j -k -l`; daemon Long + SEE ALSO + new `## codegraph daemon unlock` section) — nothing else; at HEAD 0 stub sections/bullets, 1 `## codegraph daemon unlock`, search options `--full -j -k -l -p`; `task docs:cli:drift` exit 0 byte-identical; 0 commits touch `docs/` after the feat commit | closed |
| T-03-02-09 | Denial of Service | `protect-main` blocked by a `[ci skip]` marker | high | mitigate | same CI-skip scan: none | closed |
| T-03-02-SC | Tampering | npm/pip/cargo installs | low | accept | R-03-03 | closed |
| T-03-03-01 | Repudiation | reference gates never shown RED (vacuous re-freeze) | high | mitigate | RED reproduced in a detached worktree at HEAD: one-token `Hidden: false` on the `query` stub → `task docs:cli:drift` exit 201 (`differs from a fresh regeneration`, `+## codegraph query`); `TestEveryRegisteredFlagIsAccountedFor` `--- FAIL` (`stale allowlist entry: codegraph query`); `__complete ""` offered `query`. Worktree discarded; main checkout `renamed.go` clean with `Hidden: true` ×2; both gates green on HEAD (`walked 37 commands … 3 accepted via testdata/cli-reference-allowlist.txt`) | closed |
| T-03-03-02 | Tampering | RED mutation leaking into a commit | medium | mitigate | `git log -S'Hidden: false' 21bb329e..HEAD` matches only `ef3e4c6b` (the mutation log quoting the diff); no Go file at HEAD contains `Hidden:\s+false`; `renamed.go` history in the phase = `5d69ee2e` + `fa81672c` only; 03-03's commits touch only `.planning/` | closed |
| T-03-03-03 | Tampering | blanket regenerate / re-baseline hiding a surface change | high | mitigate | 03-03 changed nothing under `internal`/`docs`/`cmd`/`testdata`; feat numstat `28 49 docs/CLI-REFERENCE.md`; two fresh `tools/clidoc` runs `cmp`-identical to each other and to the committed file | closed |
| T-03-03-04 | Tampering | MCP tool set or wire transcripts changed by the fold | high | mitigate | `git diff --quiet 5d69ee2e^ HEAD -- internal/mcp testdata/wireoracle test/wireoracle` holds, and over the whole `21bb329e..HEAD` (plus `web/` and `.goreleaser.yaml`); tool names = exactly the 8; `internal/mcp/tools.go` `eng.Search(` ×1, `eng.Query(` ×0; uncached `go test ./test/wireoracle/... -count=1` → `ok`, transcripts clean before and after | closed |
| T-03-03-05 | Repudiation | completion "absence" claimed from a script that never names commands | medium | mitigate | HEAD binary `__complete ""` = 24 entries: `search` 1, `daemon` 1, `query` 0, `unlock` 0, hidden `man` 0 (positive control); `__complete daemon ""` = `start stop unlock`; `completion bash \| rg -c -w 'query\|unlock'` = 0 AND `'search\|daemon'` = 0 (recorded as non-evidence); `man <dir>` = 29 pages, `codegraph-daemon-unlock.1` present, stub pages absent; 0 committed completion/man artefacts | closed |
| T-03-03-06 | Denial of Service | `protect-main` blocked by a `[ci skip]` marker | high | mitigate | same CI-skip scan: none | closed |
| T-03-03-SC | Tampering | npm/pip/cargo installs | low | accept | R-03-04 | closed |
| T-03-04-01 | Tampering | hand-authored ROADMAP structure truncating the milestone window | high | mitigate | `ef3cf0c4` touches exactly `ROADMAP.md` + `999.5-…/.gitkeep`; numstat `11 0`; exactly one `+### ` line, 0 removed lines; the heading carries no version token; sits after `## Backlog` and `999.4` with no `## ` heading after it; Goal value names `v0.15.0`, `renamed.go`, the allowlist, `VERB-03/VERB-04`, the feat SHA; `getMilestonePhaseFilter` → `01- 02- 03-` only; `roadmap validate` → `{"warnings":[]}` | closed |
| T-03-04-02 | Repudiation | after-census "zero" trusted without a control | high | mitigate | same control found `:1:`/`:2:`/`:3:` before the after-run; after-census at HEAD: `4 lines across 2 files (renamed.go=2, allowlist=2)`, 0 paths outside those two files | closed |
| T-03-04-03 | Tampering | planted control leaking into a commit | medium | mitigate | as T-03-01-03 (untracked, absent, never in any commit) | closed |
| T-03-04-04 | Tampering | a second `feat` commit or a footer-less breaking change mis-categorising the release | medium | mitigate | `F^..HEAD`: 0 subjects fail the regex; 1 `feat(cli)!:`, 1 `!:`, footer present; 0 other commits carry `BREAKING CHANGE` (informational: the post-plan review fix `fa81672c` is `fix(03):` — a regular non-breaking type outside the plan's literal `test/docs/chore` list, no `!:`/footer, so the release cannot be mis-categorised; under squash-merge the PR title governs) | closed |
| T-03-04-05 | Denial of Service | `protect-main` blocked by a `[ci skip]` marker | high | mitigate | same CI-skip scan over every subject and body: none | closed |
| T-03-04-06 | Denial of Service | a known load flake read as a phase failure (or a real failure hidden as a flake) | medium | mitigate | WINDOWS #37 (`32b75558`) records both results for the flake that actually occurred (`TestRunWatchdogCancelsRunOnSimulatedReparent`: 1/3 combined-suite failures post-fold, reproduced 1/2 on PRE-fold `5bc10ed8` in a scratch worktree, daemon package passes alone every time) — classified pre-existing only after the pre-fold reproduction; auditor: `TestDaemonSharedWriter` alone `--- PASS (0.22s)`; daemon lock tests 6/6 `--- PASS` | closed |
| T-03-04-SC | Tampering | npm/pip/cargo installs | low | accept | R-03-05 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

Cross-check: `go build ./...` ok; `gofmt -l` empty; `go vet` ok on `internal/cli`, `internal/daemon`, `test/integration`; `go test ./internal/cli/... -count=1` 5/5 packages `ok`; `task docs:cli:drift` byte-identical; wire oracle `ok` uncached — all under `GOTOOLCHAIN=go1.26.6`, re-run by the auditor at HEAD `1dcfc021`.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-03-01 | T-03-02-05 | `search --full` exposes `QualifiedName`/`Signature` that the removed `query --json` envelope (`renderQueryNodeJSON`) and the MCP `codegraph_node` markdown renderer already emitted; the engine opens only a local index path, no network path exists in `search.go`/`internal/query` | plan 03-02 register | 2026-09-16 |
| R-03-02 | T-03-01-SC | 03-01 commits touch only `.planning/`; no manifest change | plan 03-01 register | 2026-09-16 |
| R-03-03 | T-03-02-SC | `go.mod`/`go.sum` unchanged across the phase; 0 install commands in any commit message | plan 03-02 register | 2026-09-16 |
| R-03-04 | T-03-03-SC | 03-03 commits touch only `.planning/` | plan 03-03 register | 2026-09-16 |
| R-03-05 | T-03-04-SC | 03-04 commits touch only `.planning/` (via tool verbs) | plan 03-04 register | 2026-09-16 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-16 | 30 | 30 | 0 | gsd-security-auditor (opus) via /gsd-secure-phase 3, autonomous run — HEAD `1dcfc021`; both RED reproductions ran in detached scratch worktrees, removed afterwards |

Informational observations from the audit (not threats): (1) `internal/graphstore/logger.go:50` carries a pre-existing `os.Exit(1)` in `Fatalf` — the plans' phrase "the tree's only error exit" is slightly overstated, but that path is unreachable from the stubs and untouched by the phase; (2) the WINDOWS #37 flake is `TestRunWatchdogCancelsRunOnSimulatedReparent`, not the 03-01 Baseline's named `TestDaemonSharedWriter`; (3) `fa81672c` is a `fix(` type outside 03-04's literal `test/docs/chore` audit list; (4) IN-01 (`search` lacks a `Long` description) remains open from `03-REVIEW.md` as an Info finding, out of security scope. No unregistered surface: the one post-plan code change (`fa81672c`, WR-01) shrinks the stub's behaviour and adds a test-only integration file.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-16
