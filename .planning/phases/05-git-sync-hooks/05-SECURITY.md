---
phase: 5
slug: git-sync-hooks
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-07-17
---

# Phase 5 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| process → filesystem | `fsatomic.WriteFile` writes to a caller-supplied path; paths originate from trusted internal callers (agent config paths, githooks hook-dir paths), not network input | file contents, file mode |
| process → git subprocess | `git rev-parse` output derives a filesystem write target (the hooks dir); a spoofed `--git-path` result could redirect writes | hooks-dir path string |
| githooks package → git hooks dir | writes 3 fixed-named script files into the resolved hooks dir; shell content is a fixed constant, target dir from `gitmeta.HooksDir` | hook script bytes (mode 0755) |
| existing hook file → strip/splice | pre-existing hook content (TS-written or user-authored) is parsed for markers and re-spliced | arbitrary pre-existing hook bytes |
| CLI arg `[path]` → filesystem | the optional `[path]` positional is the only user-supplied input; resolves through `targetRoot` → `filepath.Abs` before reaching githooks | project-root path string |
| environment (`CODEGRAPH_NO_WATCH` / WSL) → advisory gate | the disabled-reason gate reads env/WSL via the injectable `watch.Probe`; strict `== "1"` comparison prevents unrelated env noise from flipping state | env var values |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-05-01 | Tampering | `markerBlock()` shell content (command injection) | high | mitigate | Block is a fixed constant `strings.Join` of literal lines with **zero interpolation** of any runtime value (no path/username/env); `internal/githooks/githooks.go:39-50`, asserted by `TestMarkerBlock` | closed |
| T-05-01b | Elevation of Privilege | writing to an attacker-controlled path via hook name | low | mitigate | Hook names are 3 fixed constants (`defaultSyncHooks`, `githooks.go:32`); write target always `filepath.Join(hooksDir, fixedName)` (`:254`); `HooksDir` degrades to `""` → Install returns Skipped (`:243-245`) | closed |
| T-05-02 | Spoofing | `HooksDir` (`--git-path` output redirects write target) | medium | mitigate | `HooksDir` degrades to `""` on any error/empty output (`internal/gitmeta/githooks.go:46-52`); Install only ever writes `filepath.Join(hooksDir, <one of 3 fixed names>)` — a spoofed dir cannot reach an arbitrary attacker-named file | closed |
| T-05-02b | Denial of Service | `IsGitRepo`/`HooksDir` on a hung git | low | mitigate | 5s `context.WithTimeout(ctx, gitTimeout)` + `cmd.Stdin = nil` on both probes (`gitmeta/githooks.go:17-22, 39-44`) — git can never block on an interactive prompt | closed |
| T-05-03 | Tampering | path traversal via `[path]` CLI arg | low | mitigate | Reuses the shared `targetRoot(args)` → `filepath.Abs` helper (`internal/cli/init.go:85-90`) used at all 3 githooks CLI sites (`cli/githooks.go:34,63,102`); no new path-handling code | closed |
| T-05-03b | Denial of Service | git subprocess hang during a CLI command | low | mitigate | Inherited gitmeta 5s timeout + `Stdin = nil`; command degrades to a skip message, never hangs | closed |
| T-05-04 | Tampering | partial/corrupt file on crash (fsatomic + hook writes) | low | mitigate | Temp-file-in-same-dir + `os.Rename` — only whole-file content is ever observable at the target; every error path removes the temp file (`internal/fsatomic/fsatomic.go:37-64`) | closed |
| T-05-04b | Tampering | mode-tightening regression on rewrite | low | mitigate | `os.Stat` reads and `os.Chmod` re-applies the existing file's mode before rename, preventing a silent 0644→0600 tightening (`fsatomic.go:52-59`); asserted by test | closed |
| T-05-05 | Tampering | in-place-edit divergence corrupting a TS-installed hook | low | mitigate | Strip-then-append-at-end parity (`githooks.go:260-278`); malformed marker block → skip hook byte-untouched + accumulate error (CR-01/WR-01 data-loss guard); verbatim-TS-block fixture test proves detect/replace/remove interop | closed |
| T-05-06 | Denial of Service | advisory/cleanup blocking init/uninit on a hung git or slow probe | low | mitigate | `WatchDisabledReason` is pure/fast; gitmeta probes carry the 5s timeout + `Stdin nil`; `githooks.Remove` degrades to Skipped; cleanup error never propagates (best-effort, called at tail of RunE) | closed |
| T-05-07 | Information Disclosure | advisory printing host paths | low | accept | Advisory prints the disabled reason + generic guidance, not host paths; consistent with the plain-text-to-stdout convention (color/host-path policy owned elsewhere) — see Accepted Risks Log | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-05-01 | T-05-07 | The watcher-fallback advisory prints only the disabled reason plus generic `codegraph githooks install` / `codegraph sync` guidance — no host paths, usernames, or repo contents. Low-severity info-disclosure with no sensitive data crossing the boundary; disposition authored at plan time (05-05). | secure-phase (plan-time disposition) | 2026-07-17 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-17 | 11 | 11 | 0 | secure-phase (L1 orchestrator, grep-depth) |

Register authored at plan time (all 5 PLAN files carried `<threat_model>` blocks). ASVS L1 + `threats_open: 0` + register-authored-at-plan-time satisfies the short-circuit rule — mitigations verified at grep-depth by reading source; no `gsd-security-auditor` L2/L3 spawn required. 10 threats mitigated (source-verified), 1 accepted (T-05-07). Highest severity (high, T-05-01 command injection) verified closed by elimination: `markerBlock()` interpolates no runtime value.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-17
