---
phase: 2
slug: status-content-git-worktree-awareness
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-07-17
---

# Phase 2 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> First-time (State-B) review, verified by gsd-security-auditor against the plan-time register.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| MCP tool input → shared Engine | tool args reach `Engine` via `confineToRepoRoot` (`tools.go:31-45`) which runs BEFORE `OpenAt` (`tools.go:69`) — rejects `..`/`../` traversal | tool arguments, resolved paths |
| CLI `--path` → status/query | operator-supplied `--path` is deliberately unconfined on the CLI (operator already has a shell) — NOT a network boundary | path string |
| process → git subprocess | `gitmeta` worktree probes use fixed argv `git rev-parse` (no `sh -c`), 5s timeout, `cmd.Stdin=nil`; untrusted value only via `cmd.Dir` | git output (worktree/common dir) |
| MCP worktree notice → agent | `WorktreeMismatch` detection compares caller `start` (cwd) vs resolved index root; notice rides inside the tool-result payload, not stdout | advisory notice text |
| process → filesystem | `status` DB-size walk is confined to `.codegraph/store/` (`status.go:288`); bounded Pebble file set | file sizes |
| JSON vs markdown output surfaces | notice/warning emitted strictly AFTER the `if jsonOut { return }` early return — JSON shape stays frame-pure | rendered payloads |
| golden parity harness → frozen goldens | determinism/provenance asserted against frozen v1.3.1 goldens after canonicalization | captured status/explore payloads |

---

## Threat Register

36 entries (T-02-01…T-02-35 + the cross-cutting T-02-SC). Register authored at plan time across all 7 PLAN `<threat_model>` blocks; verified at ASVS L1 by gsd-security-auditor — every `mitigate` located at a concrete boundary, every `accept` documented in its PLAN.

### Re-verified historical regressions (deep-review scar tissue)

| Regression | Status | Evidence |
|-----------|--------|----------|
| **CR-01** — MCP worktree notice was once dead code (index root resolved before `BuildServer`, gate short-circuited to nil on every real call) | ✅ FIXED at the real `serve --mcp` path | `serve.go:181` captures `start` (caller cwd) *before* `serveServerPaths` resolves the index root; `serve.go:252` passes `mcp.BuildServer(hasIndex, allowlist, repoPath, start)` — `repoPath` (confinement) and `start` (detection startPath) as **distinct** args; `openEngine` (`tools.go:69`) opens on `confined` from `start`, so `WorktreeMismatch` is non-nil on a borrowed-worktree call; reachability-tested via real `CallTool` (`markdown_test.go` Test 3/4) |
| **BL-01** — cancelable ctx + long-lived cache combined into permanent cache poisoning (a `notifications/cancelled` git spawn collapsed to a clean verdict, cached forever) | ✅ FIXED | `cache.go:104-106` — `if ctx.Err() != nil { return v }` returns the verdict for that call but **never writes it to the cache** |

### Mitigate (verified present in source)

| Threat ID | Category | Severity | Status | Evidence |
|-----------|----------|----------|--------|----------|
| T-02-01 | Tampering | medium | closed | `worktree.go:39-41` fixed argv `git rev-parse`, untrusted value only via `cmd.Dir`; no `sh -c` in package |
| T-02-02 | Denial of Service | high | closed | `worktree.go:28` `gitTimeout=5s`, `:36` `WithTimeout`, `:41/:67` `cmd.Stdin=nil` |
| T-02-03 | Denial of Service | medium | closed | `cache.go:82` two-value map memoizes pos+neg; BL-01 guard `:104` |
| T-02-07 | Denial of Service | medium | closed | `status.go:187` per-entry skip→nil, `:185` can't-start surfaced, `:288` Status swallows→DbSize 0 |
| T-02-09 | Tampering | medium | closed | `golden_parity_test.go:83` `findVolatileKeysExcept` single call site, shared `volatileKeys` untouched |
| T-02-12 | Tampering | high | closed | `traverse.go`/`search.go` unmodified (`git log main..HEAD` empty); Marshal* bodies intact, golden-enforced |
| T-02-15 | Elevation of Privilege | low | closed | `tools.go:65` `confineToRepoRoot` precedes `OpenAt:69` |
| T-02-16 | Denial of Service | medium | closed | `engine.go:131` `mismatchOnce.Do` + injected detector + 5s subprocess bound |
| T-02-17 | Tampering | high | closed | `status.go:60` `*gitmeta.Mismatch json:"worktreeMismatch"` — nil→JSON null, shape-stable |
| T-02-22 | Tampering | medium | closed | `render_status.go:170/:225` two renderers, doc comments name owning surface; Test 10 `Index Statistics:` exclusion |
| T-02-23 | Tampering | low | closed | `render_status.go:162-216` Journal/indexState/pendingRefs deliberately not ported (dead branches absent) |
| T-02-24 | Elevation of Privilege | high | closed | `tools.go:31-45` `confineToRepoRoot` rejects `..`/`../` via `filepath.Rel`; runs `:65` BEFORE `OpenAt:69` |
| T-02-26 | Denial of Service | medium | closed | `cache.go` server-scoped detector + `worktree.go` 5s + nil stdin |
| T-02-29 | Information Disclosure | medium | closed | no `fmt.Print`/`os.Stdout` in internal/mcp production; notice rides inside `NewToolResultText` payload |
| T-02-30 | Tampering | high | closed | `markdown_test.go:239/:266` asserts `json.Unmarshal` FAILS + markdown markers via real CallTool; handlers return `Render*Markdown` |
| T-02-31 | Tampering | high | closed | `status.go:49-54` & `query.go:66-80` notice/warning strictly AFTER `if jsonOut { return }` early return |
| T-02-35 | Tampering | medium | closed | `engine.go:185-186` `filepath.Abs(start)` sets startPath in OpenAt |
| T-02-SC | Tampering (supply chain) | high | closed | `gitmeta` stdlib-only (no third-party imports); go.mod/go.sum untouched this phase |

### Accept (documented risk; behavior confirmed where code-verifiable)

| Threat ID | Category | Severity | Status | Rationale |
|-----------|----------|----------|--------|-----------|
| T-02-04 | Information Disclosure | low | closed | Mismatch absolute paths deliberate/load-bearing (operator's own machine) |
| T-02-05 | Tampering | low | closed | git from PATH = TS parity; PATH control implies existing code-exec |
| T-02-06 | Denial of Service | low | closed | walk root confined to `.codegraph/store/`; bounded Pebble file set |
| T-02-08 | Information Disclosure | low | closed | `status.go:57` `FilesByLanguage json:"-"` — excluded from JSON, render-only |
| T-02-10 | Tampering | low | closed | markdown pipe/backtick cosmetic; parity with existing RenderExplore |
| T-02-11 | Information Disclosure | low | closed | repo-relative FilePath/Path only; no host paths introduced |
| T-02-13 | Denial of Service | low | closed | output bounded by Engine limit/depth validation; markdown < JSON |
| T-02-14 | Information Disclosure | medium | closed | worktreeMismatch paths non-nil only in mismatch case (`engine.go:132` nil default); documented exception |
| T-02-18 | Spoofing | low | closed | best-effort false-negative bias; degrades to prior always-nil status quo |
| T-02-19 | Information Disclosure | medium | closed | same exception as T-02-14; paths only in mismatch render |
| T-02-20 | Denial of Service | low | closed | breakdown rows bounded by fixed kind/language vocabularies |
| T-02-21 | Tampering | low | closed | language/kind keys from indexer vocab, not arbitrary source |
| T-02-25 | Information Disclosure | medium | closed | paths within/parent of confined repo root the agent already reads; documented |
| T-02-27 | Tampering | low | closed | `cache.go:79` key `startPath\x00indexRoot` recomputes when index root changes |
| T-02-28 | Spoofing | low | closed | advisory notice, no action surface; parity with verbatim-source contract |
| T-02-32 | Information Disclosure | low | closed | CLI prints operator's own path to own terminal; no agent boundary |
| T-02-33 | Elevation of Privilege | low | closed | `--path` unconfined on CLI is intentional (operator has shell already) |
| T-02-34 | Denial of Service | low | closed | one-shot CLI, `sync.Once`-guarded, 5s bound |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

**High-severity threats (blocking tier): T-02-02, T-02-12, T-02-17, T-02-24, T-02-30, T-02-31, T-02-SC — all mitigate, all verified present at the correct boundary (0 open).**

---

## Accepted Risks Log

18 accepted risks (AR-02-01…AR-02-18), all low/medium, all plan-time dispositions dated 2026-07-17 by secure-phase. See the Accept table above for the per-threat rationale (T-02-04, -05, -06, -08, -10, -11, -13, -14, -18, -19, -20, -21, -25, -27, -28, -32, -33, -34). Common theme: repo-relative or operator-own-machine paths (no agent-boundary disclosure), bounded output vocabularies, and CLI-side operations where the operator already has shell access.

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-17 | 36 | 36 | 0 | gsd-security-auditor (opus, ASVS L1) |

State-B first-time review. Register authored at plan time (all 7 PLAN files carried `<threat_model>` blocks). 18 mitigate verified present at concrete boundaries (incl. T-02-SC), 18 accept documented. 7 high-severity threats (the blocking tier) all mitigate + verified. Both historical deep-review regressions (CR-01 dead-code notice, BL-01 cancelled-ctx cache poisoning) explicitly re-verified fixed at their real call sites. Zero ESCALATE. No unregistered attack surface (no `## Threat Flags` in any of the 7 SUMMARY files). No implementation files modified.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-17
