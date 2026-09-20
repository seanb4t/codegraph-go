---
phase: "4"
slug: "cli-glow-up"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-19"
---

# Phase 4 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
>
> Built retroactively (State B — the secure-phase hook was never run for this phase). The
> register is assembled from the eight plans' own `<threat_model>` blocks; every mitigation
> was verified against the tree **as it stands at HEAD `b8cfadd8`**, not against the phase
> boundary. Phases 5, 6 and 7 modified several files this phase's mitigations depend on —
> those are called out per row.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| process environment + `--color` → resolver | `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE`, `TERM`, `COLORTERM` are CI- and shell-controlled; `rewriteEnviron` (`colorflag.go:106-172`) builds a fresh slice, never mutating the caller's, and feeds exactly one `colorprofile.Detect` (`:206`). `--color` itself is a closed `auto\|always\|never` enum rejected at pflag parse time (`:40-48`) | env strings, one flag value |
| resolver → the user's terminal (OSC 11) | `lipgloss.HasDarkBackground` puts stdin in raw mode and blocks up to its own 2 s timeout; fired only when Styled AND both `out` and `in` are real `*os.File` TTYs (`:214-221`), at most once per RunE. `resolveColorStderr` (`:239-250`) has no query path at all | a terminal query |
| indexed repository → styled terminal | symbol names, kinds, paths, qualified names, signatures and verbatim source lines are adversarial by definition; `present.sanitizeControl` (`present/sanitize.go:26-37`) drops every `unicode.IsControl` rune except tab before `Style.Render` | repo-derived strings |
| filesystem/env paths → styled terminal (`internal/cli`) | `codegraphDir`, repo roots, hook dirs, agent config paths and error texts go through `sanitizePathForDisplay` (`uninit.go:149-159`) before `pal.X.Render` | paths, error strings |
| agent/MCP surface ↔ charm family | six serve-reachable packages (`mcp`, `graphstore`, `daemon`, `watch`, `indexer`, `query`) may never import anything under `charm.land/` or `github.com/charmbracelet/` — PREFIX-matched, with a self-defeat probe so the guard cannot pass vacuously (`present/archtest/import_graph_test.go:59-62,:151-186`) | import paths |
| `serve --mcp` stdout (JSON-RPC) ↔ stderr (human banners) | the one process where both coexist; only the writer handed to `serveWatchStart` is ever wrapped (`serve.go:267-269`); stdout is never touched | wire frames vs chrome |
| the pre-glow-up plain output → committed goldens | 29 goldens frozen before any renderer existed; regeneration is `-update-plain-goldens`-gated, with a ≥28 floor and two-way set-equality so neither a missing nor an orphan file can hide (`plain_golden_test.go:26,:460,:488,:526-545`) | frozen bytes |
| developer machine → committed goldens | `$TMPDIR`/home/project paths, the ui port and build identity are normalized to `<VERSION>`/`<COMMIT>`/`<DATE>`/`<GO>`/`<OS>`/`<ARCH>`/`<HOME>` placeholders | placeholders only |
| `os.Args` → process env | the fang-adopted branch would have mirrored a flag value into env vars. **fang was declined** (04-FANG-VERDICT.md), so this boundary does not exist in shipped code | nothing |
| cobra help metadata → styled writer | command names, `Short`/`Long`, group titles and `FlagUsages()` — all repo-owned static literals (every string flag's default is a compile-time constant), so `present/help.go` deliberately carries no sanitizer | static metadata |
| generated reference ↔ live Cobra tree | `docs/CLI-REFERENCE.md` is regenerated only by `task docs:cli`; `task docs:cli:drift` asserts byte-identity against a fresh generation | a generated doc |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | Info Disclosure | `testdata/plain/*.golden` | low | mitigate | normalizers replace tmpdir/home/project paths, the ui port and build identity with placeholders; an `rg` over all 31 goldens at HEAD for `/var/folders`, `/tmp/`, `T/tmp.` and `/Users/` matches nothing; the placeholders were observed live in the planted-mutation transcript (`<VERSION> (commit <COMMIT>, built <DATE>) <GO> <OS>/<ARCH>`) | closed |
| T-04-02 | Tampering | `plain_golden_test.go` regeneration | medium | mitigate | writes happen only under `-update-plain-goldens` (`:26,:488`); floor `len(cases) >= 28` (`:460`); two-way set-equality rejects both missing and orphan goldens (`:526-545`); a missing golden is fail-closed (`:497`). Family (a) independently reproduced RED in a throwaway worktree — both the byte-mismatch AND the ESC-present assertion fired in the same subtest | closed |
| T-04-03 | Repudiation | `04-MUTATION-LOG.md` | low | mitigate | 6 verbatim `--- FAIL` transcripts and 15 byte-clean/`git diff --quiet` proofs; Families (a), (b), (c), (e) present, (d) recorded as explicitly not-applicable rather than silently omitted | closed |
| T-04-04 | Tampering | present archtest denylist | high | mitigate | `forbiddenImportPathPrefixes` covers BOTH vanity roots by prefix (`import_graph_test.go:59-62`); `packages.Load` resolved-count assertion and the `assertCharmImporterExists` self-defeat probe (Tests: true) block a vacuous pass. **Independently reproduced RED at HEAD:** a planted `import _ "github.com/charmbracelet/colorprofile"` in `internal/query` produced `import_graph_test.go:202: package …/internal/query imports github.com/charmbracelet/colorprofile` | closed |
| T-04-05 | DoS / Repudiation | fang `DefaultErrorHandler` + `main.go` | medium | mitigate | the declined branch: `rg 'charm.land/fang'` matches nothing in any `.go`, `go.mod` or `go.sum`; `cmd/codegraph/main.go` is still the single `Fprintln(os.Stderr, err)` + `os.Exit(1)` path; `TestRenamedStubsPrintExactlyOnce` 4/4 PASS at HEAD | closed |
| T-04-06 | Info Disclosure | fang help/error chrome | low | accept | AR-04-02 — moot, fang declined | closed |
| T-04-SC | Tampering | package installs (plans 01, 03–08) | low | accept | AR-04-01 — no module was installed by these plans | closed |
| T-04-SC-02 | Tampering | `go get charm.land/fang/v2@v2.0.1` (adopted only) | high | mitigate | never installed. The only `go.mod` change across the whole phase (`99d36a62..777ae086`) is the promotion of `github.com/charmbracelet/colorprofile v0.4.3` from `// indirect` to direct; `go.sum` is byte-unchanged. `TestCharmCgoClosure` green (charm closure non-empty, 0 `CgoFiles`); `govulncheck ./internal/cli/...` at HEAD reports 0 reachable vulnerabilities | closed |
| T-04-07 | Tampering (V5 input validation) | `--color` (`colorChoice.Set`) | medium | mitigate | closed enum at `colorflag.go:40-48`; empty and bogus both rejected at pflag parse time with a message naming all three values; `TestColorFlagInvalidValue` PASS (and asserts `--color=never` is byte-identical to the bare run); error reaches exit 1 via `main.go`'s single path | closed |
| T-04-08 | DoS | `lipgloss.HasDarkBackground` (raw stdin, 2 s) | high | mitigate | gated on Styled AND `out`/`in` both being real `*os.File` TTYs (`colorflag.go:214-221`); `resolveColorStderr` has no query path (`:239-250`); `TestResolveColorMatrix` PASS. **Re-audited at HEAD across all 29 call sites** — every RunE resolves exactly once, including the CR-01 `init` fix (`init.go:84-86`), `sync.go:82`, `daemon stop` after Phase 7's WR-02 change (`daemon.go:246`), and `search`'s two mutually-exclusive branches (`search.go:104` returns before `:144`). Hostile `TERM`/`COLORTERM` fuzz (200 000 bytes, embedded OSC, newlines, empty, `dumb`) — no panic, no unbounded work (≤0.025 s), 0 ESC on a pipe | closed |
| T-04-09 | Tampering | ANSI on the agent/JSON/piped path | high | mitigate | `--json` early return and the plain fallback structurally precede every styled branch; `TestPlainGolden` + `TestNoColorNonTTYRegression` (31 cases × 3 env states) green at HEAD. **Real-binary probe at HEAD** under `CLICOLOR_FORCE=1 … --color=always`: `status/search/callers/impact/files/version/telemetry --json` and `affected --quiet` all emitted 0 ESC bytes, with a non-vacuity positive control showing 8/8 human verbs styled (4–118 ESC bytes each) | closed |
| T-04-10 | Tampering | adversarial path/name in status/files | medium | mitigate | `status.go:163` (project path) and the `WorktreeMismatch` field-wise sanitize at `:170-181`; `files.go:22,:25,:46`. **Live:** `codegraph status --color=always` run from a cwd literally named `repo<ESC>]0;PWNED<BEL>x` rendered `…/repo]0;PWNEDx` — inert; `files` showed `sub]0;PWNED_DIR/b.go`. Breakdown keys and `Backend` are parser/storage enums, not repo text (AR-04-08) | closed |
| T-04-11 | Info Disclosure | OSC-11 response parsing | low | accept | AR-04-03 — third-party (lipgloss v2.0.5); worst case is the wrong light/dark hex | closed |
| T-04-12 | Tampering (escape injection) | `present/results.go` renderers | high | mitigate | 9 `sanitizeControl` calls cover every repo-derived value (`:28,:30,:32,:64,:67,:80,:96,:112,:170`); the remaining `Render` arguments are `strconv.Itoa` counts and literals. Contract tests include control-byte fixtures; `internal/cli/present` 100% green | closed |
| T-04-13 | Tampering | `--json`, `affected --quiet`, non-TTY | high | mitigate | same structural ordering as T-04-09; verified by the same real-binary probe | closed |
| T-04-14 | Info Disclosure | styled vs plain content divergence | medium | mitigate | `TestRenderResultsStrippedEqualsPlain`, `TestRenderExploreStrippedEqualsMarkdownContract`, `TestRenderNodeStrippedEqualsMarkdownContract`, `TestRenderStatus_MatchesPipedSectionOrder` — all PASS; `task test:golden` green (25 s) | closed |
| T-04-15 | Tampering (escape injection via source lines / names) | `present/explore.go`, `node.go` | high | mitigate | 13 + 13 `sanitizeControl` calls; every `Render` without one takes a literal, a `strconv.Itoa`, a `strconv.Quote` (which escapes control bytes itself, `node.go:164`), or a helper that sanitizes internally (`joinSymbolKindList`, `explore.go:108-114`). **Live, non-vacuous:** a Go file carrying `<ESC>]0;PWNED_COMMENT<BEL>` in a comment and an OSC 52 clipboard payload in a string literal rendered through `explore --color=always` as `// ]0;PWNED_COMMENT` and `"]52;c;cHduZWQ="` — text present, escapes gone | closed |
| T-04-16 | Info Disclosure (path confinement) | source bytes | medium | mitigate | `rg 'os\.ReadFile\|os\.Open\|ioutil\.\|os\.Stat'` over every non-test file in `internal/cli/present/` matches nothing; bytes arrive only through the engine's repo-root-confined reader | closed |
| T-04-17 | DoS (unbounded multi-def render) | `present/node.go` budget loop | medium | mitigate | `nodeMultiDefHardCap = 16`, `nodeMultiDefBodyBudget = 12000`, `nodeMultiDefListCap = 20` (`node.go:20-22`), enforced at `:147` (fetch stop), `:156` (budget), `:179`/`:191` (list cap); `Definition(n)` is invoked lazily | closed |
| T-04-18 | Tampering | plain markdown path | high | mitigate | styled branch inserted before `eng.Explore`/`eng.Node`; goldens byte-identical; `task test:golden` green | closed |
| T-04-19 | Tampering (escape injection) | `present/line.go` helpers | medium | mitigate | `Line` (`:17`), `KV` (`:37`) and `lineWriter.Write` (`:72,:82,:89`) all sanitize every segment, including the partial tail; `TestLineSanitizesAndStyles` and `TestLineWriterStylesEachLine` PASS | closed |
| T-04-20 | Tampering | `version --json`, `--quiet` summaries | medium | mitigate | JSON return and quiet guard precede the resolver (`init.go:126-130` skips even `colorprofile.Detect` when quiet); real-binary `version --json --color=always` → 0 ESC | closed |
| T-04-21 | Info Disclosure | none new | low | accept | AR-04-04 — the same strings already reached stdout before the phase | closed |
| T-04-22 | Tampering | `serve --mcp` stdout | high | mitigate | only the stderr writer is wrapped (`serve.go:267-269`); `resolveColorStderr` never queries. **Live at HEAD:** `serve --mcp --color=always` with empty stdin wrote **0 bytes** to stdout and 0 ESC to stderr; a real `initialize` request under `CLICOLOR_FORCE=1 TERM=xterm-256color --color=always` returned a JSON-RPC frame with **0 ESC bytes**. `go test ./test/wireoracle/...` green (49 s); all 42 frozen transcripts are ESC-free | closed |
| T-04-23 | Repudiation (masked write failure) | `printAgentResults` | high | mitigate | `errors.Join(errs...)` at `install.go:269`; the `  error:` line is emitted in BOTH the styled (`:262`) and plain (`:264`) branches and every error is still appended to `errs` (`:266`); `TestInstall_WriteFailure_ReportsErrorAndNonZeroExit` and `TestInstall_StyledOutputStripsToPlain` PASS | closed |
| T-04-24 | Tampering (terminal escape injection) | githooks/daemon/install path strings | medium | mitigate | **Held as shipped in Phase 4, regressed at HEAD.** `githooks.go:173`, `daemon.go:132,:275,:308`, `install.go:248,:262` and `uninit.go:51,:100` all route through `sanitizePathForDisplay` — verified. But `install.go:255` renders `result.Notes` entries as `pal.Value.Render(note)` with **no sanitizer**. At the Phase 4 boundary (`f1bb4aad`) the only note was the constant `kiroDisabledByDefaultNote`, so the mitigation was complete as written; Phases 5 and 7 added path-bearing notes (`agents/codex.go:129-132` `codexTrustNote`, `:224`, `agents/shared.go:849`, `agents/skillshared.go:268`). **Reproduced at HEAD:** `codegraph install --target codex --location local --color=always` run from a directory named `repo<ESC>]0;PWNED<BEL>x` emitted the raw OSC-0 set-title sequence to stdout on the `note:` line (the `%q` copy in the same note IS escaped; the leading `%s` is not). See Remediation. **FIXED 2026-09-19 during the milestone audit** (`841aea91` RED, `27dd6311` fix): `install.go` now sanitizes each note in BOTH branches, and `upgrade.go`'s file and note lines got the same treatment; pinned by `TestInstall_NotesAreControlCharacterSanitized` with mutation Family (k1) in `.planning/phases/07-codex-parity/07-MUTATION-LOG.md`; re-reproduced with the real binary (0 OSC-0 sequences after) | closed |
| T-04-25 | DoS | background query on stderr resolution | medium | mitigate | `resolveColorStderr` has no query path by construction (`colorflag.go:239-250` — it returns `Dark: true` unconditionally and never calls `queryDarkBackground`) | closed |
| T-04-26 | Tampering (V5 input validation) | `applyColorFlagToProcessEnv` (adopted only) | medium | mitigate | not applicable — fang was declined, the function does not exist, and no code mirrors `os.Args` into the process env. The equivalent live surface, `rewriteEnviron`, acts on exactly the three documented keys and never mutates the caller's slice (`colorflag.go:106-172`); `TestRewriteEnviron` PASS | closed |
| T-04-27 | EoP | `GroupID` → cobra panic on an unregistered id | low | mitigate | ids come from four consts and one table (`root.go:49-88`), applied only where the table has an entry (`:140-144`); `TestEveryCommandHasGroupID` asserts group count, registration order, membership in the registered set, and that hidden commands stay groupless; cobra's `checkCommandGroups` is the backstop | closed |
| T-04-28 | Tampering | `docs/CLI-REFERENCE.md` | medium | mitigate | `task docs:cli:drift` at HEAD: "byte-identical to a fresh regeneration (temporary file only — source tree untouched)"; `TestEveryRegisteredFlagIsAccountedFor` green | closed |
| T-04-29 | DoS | `--help` on a TTY triggering the background query | low | accept | AR-04-05 — `installHelpFunc` uses the same `resolveColor(c)` gate as every other verb (`root.go:165`), so at most one query, and none on a pipe | closed |
| T-04-30 | Tampering (NO_COLOR bypass on a non-TTY) | `rewriteEnviron` auto branch | low | mitigate | **Not mitigated at HEAD.** `colorflag.go:70-74` documents the precedence as `NO_COLOR > CLICOLOR_FORCE > CLICOLOR > TTY`, and amendment A1 drops `CLICOLOR_FORCE` for `--color=never` — but the `auto` branch does not. colorprofile v0.4.3 only honours `NO_COLOR` when the sink is already a TTY, so on a pipe `CLICOLOR_FORCE=1` wins. **Measured at HEAD:** `NO_COLOR=1 CLICOLOR_FORCE=1 codegraph version \| cat` → 8 ESC bytes (bare, `NO_COLOR=1` and `NO_COLOR=banana` alone → 0; `--color=never` → 0 under every combination). `TestNoColorNonTTYRegression` never sets `CLICOLOR_FORCE`, so the guard does not cover the pair. Human-output verbs only — the JSON/agent/MCP paths are protected structurally, not by this env logic, and were re-probed clean under exactly this env | **open — below high threshold (non-blocking)** |
| T-04-31 | Spoofing (Trojan Source / bidi display deception) | `sanitizeControl` scope | low | accept | AR-04-09. Go's `unicode.IsControl` is latin-1-only, so Cf/Zl codepoints are not stripped. **Confirmed at HEAD:** U+202E, U+2066 and U+2069 planted in a Go comment survive into `explore --color=always` output. This is display deception, not escape injection — no CSI/OSC/DCS sequence can be formed from them — and the frozen plain renderer has the identical property by design (TUI-02) | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*
*Totals: 33 threats — 0 critical, 10 high, 13 medium, 10 low. 27 mitigate, 6 accept, 0 transfer. 31 closed, 2 open (both below the high gate). `threats_open: 0`.*

---

## What Phases 5–7 changed under this phase's mitigations

Files this phase's threat model depends on that later phases edited, and whether the mitigation survived:

- **`internal/cli/testdata/plain/*.golden` (T-04-01/02/09/18).** Survived, and the updates were scoped: Phase 5 ADDED two goldens (`print-config-style{,-local}`) and edited only those; Phase 6 added two lines to `uninstall-local.golden` for a real behaviour change; Phase 7 edited only the two Phase-5 goldens. **None of the 29 Phase-4-frozen goldens was re-frozen wholesale** — the tamper-evidence property is intact.
- **`internal/cli/install.go` `printAgentResults` (T-04-23/24).** `errors.Join` and the `error:` line survived; the `note:` fragment did not keep pace with the notes Phases 5/7 added — this is T-04-24's open row.
- **`internal/cli/daemon.go` (T-04-08).** Phase 7's WR-02 (`--all`/`--path` conflict) touched `daemon stop`'s RunE; the single-resolution-per-RunE property still holds (`daemon.go:246`).
- **`internal/cli/printconfigstyle.go` (new in Phase 5, T-04-08/10/19).** Correctly reuses the Phase 4 contract: one `resolveColor` per RunE, early return, and `present.KV` (which sanitizes) for every styled line.
- **`internal/agents/*` (T-04-24).** `codexTrustNote`, `instructionsKeptNote`, the `AGENTS.override.md` note and the shared-skill-dir note all interpolate filesystem paths into `Notes`; only `codexTrustNote`'s is cwd-absolute and therefore adversary-influenceable.
- **`internal/cli/present/*` (T-04-12/15/19).** Unchanged by Phases 5–7 apart from Phase 4's own CR-02 tab narrowing; all renderers re-probed live at HEAD.

---

## Remediation (T-04-24 FIXED 2026-09-19; T-04-30 tracked)

1. **T-04-24** — one line: `install.go:255` should read `pal.Value.Render(sanitizePathForDisplay(note))`, matching the sibling `error:` line at `:262`. Optionally also `%q` the first path in `agents/codex.go:131` so the plain path (reached on a TTY with `NO_COLOR` set) is covered too.
2. **T-04-30** — extend `rewriteEnviron`'s `auto` branch to drop `CLICOLOR_FORCE=` when a truthy `NO_COLOR` is present, mirroring amendment A1; and add a `NO_COLOR=1 CLICOLOR_FORCE=1` row to `TestNoColorNonTTYRegression` so the pair is guarded.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-04-01 | T-04-SC | Plans 01 and 03–08 installed no module; the phase's only dependency movement is `colorprofile` indirect→direct, with `go.sum` byte-unchanged | plan-time register, audit-confirmed | 2026-09-19 |
| AR-04-02 | T-04-06 | fang's help/error chrome would have rendered the same `err.Error()`/help strings already reaching stderr. Moot — fang was declined (04-FANG-VERDICT.md) | plan-time register | 2026-09-19 |
| AR-04-03 | T-04-11 | OSC-11 response parsing is lipgloss v2.0.5's; the worst case is picking the wrong light/dark hex out of two pre-defined values — cosmetic | plan-time register | 2026-09-19 |
| AR-04-04 | T-04-21 | The one-line renderers add hue to strings that already reached stdout; no new data path | plan-time register | 2026-09-19 |
| AR-04-05 | T-04-29 | `--help` on a real TTY pays the same at-most-once background query every styled verb pays; on a pipe it never fires | plan-time register | 2026-09-19 |
| AR-04-06 | T-04-10 (adjacent) | **IN-01:** `RenderStatus`'s `Project:` VALUE is sanitized but never styled (`present/status.go:163`). Info-level, out of `fix_scope` for the 04-REVIEW `--auto` loop; recorded in STATE.md as a Phase 4 residual. Cosmetic only — the security-relevant half (sanitization) is present | 04-REVIEW.md / STATE.md, maintainer choice | 2026-09-19 |
| AR-04-07 | T-04-08 (adjacent) | The D-11 ~2 s OSC-11 timeout on a non-answering terminal (tmux without allow-passthrough, SSH) was never observed locally — tmux is not installed on the maintainer's machine and SSH was not attempted; every measured run answered in ≤0.16 s. CI's `tmux-e2e` job covers the PTY leg (GH issue #75). An evidence gap, not a mitigation gap: the fd/TTY gate that bounds the query is asserted directly by `TestResolveColorMatrix` | 04-VERIFICATION.md / 04-UAT.md, maintainer decision | 2026-09-19 |
| AR-04-08 | T-04-10 (adjacent) | `present/status.go:119` writes `kc.Key` and `:187` writes `r.Backend` without a sanitizer. Both are closed enumerations produced by the parsers and the storage layer (node kinds, edge kinds, language names, backend name) — never repo text. A defence-in-depth gap, not a reachable vector | security audit observation | 2026-09-19 |
| AR-04-09 | T-04-31 | Go's `unicode.IsControl` is latin-1-scoped, so bidi overrides and isolates (U+202E, U+2066–U+2069), zero-width and line/paragraph separators pass through `sanitizeControl` into styled `explore`/`node` source listings (reproduced at HEAD). This is Trojan-Source-style display deception, not escape injection — those codepoints cannot begin a CSI/OSC/DCS sequence — and the frozen plain renderer has the identical property by deliberate design (TUI-02). Widening the strip predicate would break legitimate RTL identifiers and comments | security audit observation | 2026-09-19 |
| AR-04-10 | T-04-15 (adjacent) | The `styledTabWidth` / `TabWidth` / `NoTabConversion` advisory: CR-02 deliberately exempts the bare tab from `sanitizeControl` so tab-indented source keeps its indentation, and lipgloss then expands each tab to 4 spaces independently of the sanitizer (`present/node_test.go:35-57`). A bare tab cannot initiate an escape sequence, so the exemption adds no injection surface; the test oracle models the expansion rather than the raw byte | 04-REVIEW.md CR-02, audit-confirmed | 2026-09-19 |
| AR-04-11 | T-04-05 (adjacent) | `charm.land/fang/v2` was declined on D-03 grounds (its `DefaultErrorHandler` wraps stderr in a `*colorprofile.Writer`, which has no `Fd()`, so the plain-print branch is structurally unreachable and every error renders as a styled box). Help is hand-rolled in `present/help.go` instead. The decision is recorded in 04-FANG-VERDICT.md and PROJECT.md key decisions; the spike was reverted byte-clean | 04-FANG-VERDICT.md (maintainer-approved) | 2026-09-19 |

*Accepted risks do not resurface in future audit runs.*

---

## Unregistered Flags

None. No Phase 4 SUMMARY carries a `## Threat Flags` section (the executor emitted none across all eight plans), so there is no executor-reported attack surface left unmapped. The two open rows (T-04-24, T-04-30) and the accepted T-04-31 were found by this audit, not flagged by the executor.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-19 | 33 | 31 | 2 (both below the high gate; `threats_open: 0`) | gsd-security-auditor (opus) at `b8cfadd8`, building the register retroactively from the eight plans' `<threat_model>` blocks. Verified deeper than L1: traced each mitigation to its boundary at HEAD rather than at the phase commit. Re-ran the enforcing suites with `GOTOOLCHAIN=go1.26.6` — `./internal/cli/...` (5 packages green), `./test/wireoracle/...` (49 s, 42 transcripts), `task test:golden`, `task docs:cli:drift` (byte-identical), `TestRenamedStubsPrintExactlyOnce` 4/4, and `govulncheck ./internal/cli/...` (0 reachable). Independently reproduced two mutation-family positive controls in a throwaway worktree, removed afterwards with the main tree left clean: Family (c) — a planted `github.com/charmbracelet/colorprofile` import in `internal/query` turned `TestNoCharmInServeReachablePackages` RED naming both package and import; Family (a) — a planted ESC byte in `version.go` turned `TestPlainGolden/version` RED on BOTH assertions plus every `TestNoColorNonTTYRegression` variant. Ran real-binary probes against scratch HOMEs and scratch repos only (never the real HOME, never `~/.codex`): a 5-value hostile `TERM`/`COLORTERM` fuzz including a 200 KB value (no panic, ≤0.025 s, 0 ESC); an 11-row `NO_COLOR`/`CLICOLOR`/`CLICOLOR_FORCE`/`--color` precedence matrix (which surfaced T-04-30); 10 JSON/quiet invocations under forced colour (0 ESC) against an 8-verb styled positive control (4–118 ESC); `serve --mcp --color=always` with a real `initialize` request (0 stdout ESC, 0 stdout bytes when idle); and escape-injection probes through an ESC-named cwd, an ESC-named source directory, an ESC-laden comment and an OSC-52 payload in a string literal — all rendered inert except the `install` `note:` line, which leaked a raw OSC-0 sequence (T-04-24). Audited the `go.mod`/`go.sum` delta across the full phase range and the per-commit scope of every Phase 5–7 golden edit. No file outside `.planning/phases/04-cli-glow-up/04-SECURITY.md` was written. |

---

## Sign-Off

- [x] All 33 threats have a disposition (27 mitigate / 6 accept / 0 transfer)
- [x] Accepted risks documented in Accepted Risks Log (AR-04-01..11)
- [x] `threats_open: 0` confirmed — both open threats (T-04-24 medium, T-04-30 low) are below the `high` blocking gate and are tracked with remediation above
- [x] No high or critical threat is open
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-19
