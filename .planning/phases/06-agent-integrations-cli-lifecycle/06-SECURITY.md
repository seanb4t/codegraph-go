---
phase: 06-agent-integrations-cli-lifecycle
asvs_level: 1
block_on: high
threats_total: 22
threats_closed: 22
threats_open: 0
register_authored_at_plan_time: true
audited_at: 2026-07-12
result: SECURED
---

# SECURITY.md — Phase 6: Agent Integrations & CLI Lifecycle

Retroactive threat-mitigation verification for the Phase 6 threat registers
(PLAN blocks 06-01 through 06-06). Every declared mitigation was verified
present in the implemented code by grep/read at ASVS L1. Implementation files
were not modified. `threats_open` counts only OPEN threats at or above the
`block_on: high` threshold — it is 0.

## Result

All 22 registered threats are CLOSED. Zero open threats. No unregistered
attack-surface flags. Phase clears the `block_on: high` gate.

## Threat Verification

### 06-06 — Signature-verified self-update (`internal/upgrade`)

| Threat ID | Category | Severity | Disposition | Status | Evidence |
|-----------|----------|----------|-------------|--------|----------|
| T-06-06-01 | Tampering/Spoofing | high | mitigate | CLOSED | `verify.go:88-114` in-process sigstore-go `verifier.Verify` with `WithArtifactDigest` on the downloaded bytes + `WithCertificateIdentity` pinned to `releaseOIDCIssuer` (`verify.go:42`) and full-match anchored `releaseWorkflowRefPattern` (`verify.go:44`). No `os/exec` — enforced by `verify_test.go:147-148`. `TestVerifyRelease_RejectsTamperedArtifact` PASS. |
| T-06-06-02 | Tampering | high | mitigate | CLOSED | `upgrade.go:114-136` structured order: download → verify → swap; a non-nil verify error returns immediately (`upgrade.go:123-128`) and never reaches `swap` (`upgrade.go:134`). `TestUpgradeRun_TamperedDownloadNeverSwaps` PASS. |
| T-06-06-03 | Denial of Service | medium | mitigate | CLOSED | `swap.go:40-88` writes+chmods the temp file fully before touching the target; `os.Rename` (POSIX) is the sole final-path step; `cleanupTemp` defer removes the temp on any early return. Windows rename-aside with restore-on-failure (`swap.go:103-120`). Original survives a failed swap. |
| T-06-06-04 | Tampering | medium | mitigate | CLOSED | `upgrade.go:90-97` `--check` reports current-vs-latest; `cli/upgrade.go` passes `version.Info().Version` as `currentVersion`; a bare `upgrade` targets latest, an explicit `upgrade <version>` is user-initiated. Downgrade is never silent. |
| T-06-06-05 | Elevation/Access Control | low | mitigate | CLOSED | `swap.go:19-29` `checkWritable` probes the target dir; called in `upgrade.Run` BEFORE download (`upgrade.go:106`) and again inside `atomicSwap` (`swap.go:46`). Refuses non-writable target, no partial state. |
| T-06-06-SC | Tampering (supply chain) | low | accept | CLOSED (accepted) | See Accepted Risks. `sigstore-go v1.2.2` pinned exactly in `go.mod`; RESEARCH Package Legitimacy Audit verdict OK/Approved; `minio/selfupdate` explicitly NOT adopted (swap hand-rolled). |

### 06-04 — Install/uninstall CLI (`internal/cli/install.go`, `uninstall.go`)

| Threat ID | Category | Severity | Disposition | Status | Evidence |
|-----------|----------|----------|-------------|--------|----------|
| T-06-04-01 | Tampering | medium | mitigate | CLOSED | `registry.go:94-106` unknown csv id → `fmt.Errorf("unknown agent target %q")`; `install.go:88-99` returns on that error before any `Install` call — no partial write. `parseLocationFlag` (`install.go:39-46`) applies the same discipline to `--location`. |
| T-06-04-02 | Denial of Service | medium | mitigate | CLOSED | `install.go:24-33` `installStdinIsInteractive` requires `os.Stdin` to be a char device; the non-TTY/CI branch (`install.go:92-96`) resolves straight to `auto`, never reading stdin. `uninstall.go` has no interactive path at all (defaults `--target all`). |
| T-06-04-03 | Spoofing | low | mitigate | CLOSED | `install.go:81` `os.Executable()` resolves the absolute running-binary path, threaded via `InstallOptions.ExecPath` (`types.go:118-121`) into every MCP entry — not a bare PATH-guessed command. |
| T-06-04-04 | Elevation | low | mitigate | CLOSED | `registry.go:73-107` `ResolveTargetFlag` resolves ids only against the fixed `registry` map via `GetTarget`; no user-controlled path interpolation. Each target owns fixed path resolvers (e.g. `claudeConfigPath`, `kiroConfigPath`). |

### 06-01 — Shared surgical-write foundation (`internal/agents/shared.go`)

| Threat ID | Category | Severity | Disposition | Status | Evidence |
|-----------|----------|----------|-------------|--------|----------|
| T-06-01 | Tampering/DoS | medium | mitigate | CLOSED | `shared.go:44-66` `readJSONFile`: missing/empty/unparseable → empty-map fallback (never panics); a genuine non-ENOENT I/O error is surfaced to the caller. |
| T-06-02 | Tampering | high | mitigate | CLOSED | `shared.go:203-241` `replaceOrAppendMarkedSection` splices only the marked span (`s[:startIdx] + body + s[endIdx+len(endMarker):]`), preserving all content outside the markers; `removeMarkedSection` (`shared.go:251-295`) is the byte-invariant inverse. |
| T-06-03 | Tampering/DoS | medium | mitigate | CLOSED | `shared.go:327-360` `atomicWriteFile` (temp-in-same-dir + `os.Rename`) is the sole write path; `writeJSONFile` and `replaceOrAppendMarkedSection` both funnel through it. `TestAtomicWriteFile_PreservesExistingFilePermissions` PASS (WR-05: perm preserved via `os.Stat`+`Chmod`, `shared.go:347-354`). |
| T-06-SC | Tampering (supply chain) | low | accept | CLOSED (accepted) | See Accepted Risks. `tailscale/hujson` pinned to exact pseudo-version in `go.mod`. |

### 06-02 — Per-agent JSON/Markdown editors

| Threat ID | Category | Severity | Disposition | Status | Evidence |
|-----------|----------|----------|-------------|--------|----------|
| T-06-02-01 | Tampering | high | mitigate | CLOSED | `shared.go:134-164` `writeMcpEntry` preserves every top-level and `mcpServers` sibling; `claude.go:121-148/153-192` `addClaudeAllowPermission`/`removeClaudeAllowPermission` append/remove only `mcp__codegraph__*`, leaving other `allow` entries and unrelated keys intact. |
| T-06-02-02 | Tampering | medium | mitigate | CLOSED | `claude.go:78-87` `claudeConfigPath(LocationLocal)` returns `.mcp.json` (never `.claude.json`); legacy `.claude.json` local entry is migrated then stripped (`claude.go:215-224`, `264-268`). |
| T-06-02-03 | Tampering | medium | mitigate | CLOSED | `antigravity.go:80-85` `antigravityEntry` builds `{command, args}` with NO `type` field, deliberately not routed through `stdioMcpEntry`. CR-02 migration-data-loss guard (`migrationOK`) present (`antigravity.go:151-207`). |
| T-06-02-04 | Tampering | low | mitigate | CLOSED | `cursor.go:72-80` and `kiro.go:73-81` `os.Remove` the legacy `.mdc`/`steering/codegraph.md` on install; neither is re-written or listed in `DescribePaths`. |

### 06-03 — TOML / JSONC / YAML editors

| Threat ID | Category | Severity | Disposition | Status | Evidence |
|-----------|----------|----------|-------------|--------|----------|
| T-06-03-01 | Tampering | high | mitigate | CLOSED | opencode: `opencode.go:105-179` hujson Parse→Patch→Format→Pack preserves comments/keys. Codex TOML: `toml.go:18-40` single-table splice (`content[:start] + newBlock + content[end:]`). Hermes YAML: `hermes.go:173-198` line-range splice preserving other top-level keys and sibling servers. |
| T-06-03-02 | Tampering/DoS | medium | mitigate | CLOSED | Defensive reads: `readJSONFile` empty-map fallback; `opencode.go:119-128` empty-object fallback on malformed JSONC; all writes via `atomicWriteFile`. |
| T-06-03-03 | Tampering | medium | mitigate | CLOSED | `opencode.go:41-50` `resolveOpencodeConfigDir` uses `XDG_CONFIG_HOME`/`~/.config` unconditionally — no `runtime.GOOS` branch (Pitfall 4). |
| T-06-03-04 | Tampering | medium | mitigate | CLOSED | `hermes.go:226-272` `hermesAppendCliToolset` detects existing list-item indent (`len(line)-len(trimmed)`) and matches it; falls back to PyYAML default 2 when no items exist. CR-03: `\r` stripped in `yamlBlockRange`/`yamlListBlockRange` (`hermes.go:66,80,107,121`). |

### 06-05 — Version / telemetry info commands

| Threat ID | Category | Severity | Disposition | Status | Evidence |
|-----------|----------|----------|-------------|--------|----------|
| T-06-05-01 | Repudiation/Info | low | mitigate | CLOSED | `telemetry.go` prints a static honest statement naming `upgrade` as the sole user-initiated network path; imports only `fmt`+`cobra` — no network package (verified by grep). |
| T-06-05-02 | Tampering | low | accept | CLOSED (accepted) | See Accepted Risks. `version.go` injects build identity via `-ldflags -X`, defaults `dev`/`unknown`; no untrusted input path; build provenance is Phase 8 (DIST-02) scope. |

## Accepted Risks Log

| Threat ID | Severity | Rationale |
|-----------|----------|-----------|
| T-06-06-SC | low | Adds `sigstore-go v1.2.2` (official Sigstore org, High reputation, 93.12 benchmark) plus its ~20-module official Sigstore/crypto/protobuf transitive subtree. Verified against proxy.golang.org (RESEARCH Package Legitimacy Audit, verdict OK/Approved). Pinned to an exact version in `go.mod` (no floating `@latest`). `minio/selfupdate` ([SUS], stale) explicitly not adopted — the swap is hand-rolled (D-13). |
| T-06-SC (06-01) | low | Adds `tailscale/hujson` (Tailscale org, High reputation) for comment-preserving JSONC edits. Verified existent + version-pinned against proxy.golang.org; Go modules sit outside the npm/pip/cargo legitimacy gate. Pinned to an exact pseudo-version in `go.mod`. |
| T-06-05-02 (06-05) | low | `version` build identity is only as trustworthy as the release build that sets the ldflags vars; there is no untrusted input path to version reporting. Cryptographic provenance of the build itself is deferred to Phase 8 signing/attestation (DIST-02). |

## Unregistered Flags

None. No `## Threat Flags` section appears in any 06-0x SUMMARY.md; no new
attack surface was declared during implementation that lacks a registered
threat mapping. All new surface (upgrade network path, registry-driven config
writes) maps to registered threats above.

## Prior-Review Fixes Confirmed Present

The 06-REVIEW.md deep-review fixes were verified in code (not re-flagged):
CR-01 swallowed I/O errors → `recordFile`/`WriteResult.Errors` funnel
(`shared.go:17-34`, surfaced+non-zero-exit via `printAgentResults`,
`install.go:213-239`); CR-02 antigravity migration data-loss → `migrationOK`
gating (`antigravity.go:151-207`); CR-03 hermes CRLF → `\r`-trim in YAML range
scanners; WR-08 unanchored release-workflow-ref → full-match `^...$`
`releaseWorkflowRefPattern`; WR-01/02 HTTP timeouts + bounded downloads
(`upgrade.go:16,29,205-227`, `verify.go:18,68-77`, `release.go:24,46-53`);
WR-04 Windows swap restore (`swap.go:103-120`); WR-05 permission preservation
(`shared.go:347-354`).
