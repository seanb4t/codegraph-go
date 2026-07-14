---
phase: 06-agent-integrations-cli-lifecycle
reviewed: 2026-07-12T21:13:39Z
depth: deep
files_reviewed: 24
files_reviewed_list:
  - internal/agents/types.go
  - internal/agents/shared.go
  - internal/agents/registry.go
  - internal/agents/instructions.go
  - internal/agents/claude.go
  - internal/agents/cursor.go
  - internal/agents/gemini.go
  - internal/agents/kiro.go
  - internal/agents/antigravity.go
  - internal/agents/toml.go
  - internal/agents/codex.go
  - internal/agents/opencode.go
  - internal/agents/hermes.go
  - internal/cli/install.go
  - internal/cli/uninstall.go
  - internal/cli/version.go
  - internal/cli/telemetry.go
  - internal/cli/upgrade.go
  - internal/cli/root.go
  - internal/version/version.go
  - internal/upgrade/release.go
  - internal/upgrade/verify.go
  - internal/upgrade/swap.go
  - internal/upgrade/upgrade.go
  - go.mod
findings:
  critical: 3
  warning: 8
  info: 2
  total: 13
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-07-12T21:13:39Z
**Depth:** deep
**Files Reviewed:** 24
**Status:** issues_found

## Summary

Reviewed the agent-integration config editors (`internal/agents/*`, `internal/cli/install.go`/`uninstall.go`) and the self-update pipeline (`internal/upgrade/*`, `internal/cli/upgrade.go`) at deep depth, tracing call chains across the `AgentTarget` interface and the `Run()` resolve→check→download→verify→swap sequence.

The verify-before-swap ordering itself is sound: `upgrade.Run` never calls `swap` unless `verify` returns nil, and verification runs against the exact in-memory bytes that are later written to disk (no re-read from disk between the two, so no classic TOCTOU there). The most serious problems found are elsewhere:

- **`WriteResult` has no error field**, and every per-agent `Install`/`Uninstall` implementation follows an `if err == nil { append(...) }` pattern that silently discards I/O failures — a permission-denied or disk-full error while editing `~/.claude.json` produces no error, no non-zero exit, and no visible line in the CLI's per-file report. This is systemic across all 8 agent target files.
- `antigravity.go`'s legacy→unified config migration removes the entry from the legacy file and unconditionally marks migration "complete" even when the write of that entry into the new unified file is known (via an ignored error) to have failed — a genuine data-loss path, not just a silent no-op.
- `hermes.go`'s hand-rolled YAML block-boundary matcher does exact line equality without trimming `\r`, so a CRLF-line-ended `config.yaml` (a plausible real-world case for a project that ships Windows binaries) breaks both install idempotency (duplicate blocks appended on every run) and uninstall (never finds anything to remove).

The upgrade pipeline itself has several hardening gaps worth fixing before this ships as a real self-update mechanism: unbounded downloads, no HTTP timeouts anywhere in the network path, no downgrade/rollback protection, and a Windows swap failure mode that can leave the target binary missing if both the primary and the best-effort restore rename fail.

## Critical Issues

### CR-01: Install/Uninstall silently swallow every file I/O error across all 8 agent targets

**File:** `internal/agents/types.go:89-100`, `internal/agents/claude.go:222-234`, `internal/agents/cursor.go:87-97`, `internal/agents/gemini.go:66-78`, `internal/agents/kiro.go:78-84`, `internal/agents/antigravity.go:127-174`, `internal/agents/codex.go:90-109`, `internal/agents/opencode.go:268-278`, `internal/agents/hermes.go:321-343`, `internal/cli/install.go:206-226`

**Issue:** `WriteResult` (types.go:89-100) carries only `Files []FileResult` and `Notes []string` — there is no error field anywhere on the return path from `AgentTarget.Install`/`Uninstall`. Every call site follows the same pattern:

```go
if configPath, err := claudeConfigPath(loc); err == nil {
    if fr, err := writeMcpEntry(configPath, func() any { ... }); err == nil {
        result.Files = append(result.Files, fr)
    }
}
```

If `writeMcpEntry` (or the equivalent write helper in every other target file) returns a non-nil error — e.g. `EACCES` writing `~/.claude.json`, a full disk, a `MkdirAll` failure, `os.WriteFile` failing on the antigravity `.migrated` marker — the error is discarded. `printAgentResults` (`internal/cli/install.go:206-226`) then prints only whatever ended up in `result.Files`, so a total write failure for the primary MCP entry looks identical to "nothing needed to change" in the CLI's own status line (`"unchanged"`), and `codegraph install`/`uninstall` always exits 0 regardless. A user can believe an agent is fully configured (or fully unconfigured) when the actual filesystem write never happened.

**Fix:** Add an `Errors []error` (or a single aggregated `error`) field to `WriteResult`, populate it at every one of the ~40 swallow sites in the 8 target files, and have `printAgentResults` print any errors and cause `install`/`uninstall`'s `RunE` to return a non-nil error (non-zero exit) when any target reports one:

```go
type WriteResult struct {
    Files  []FileResult
    Notes  []string
    Errors []error
}
// ...
if fr, err := writeMcpEntry(configPath, ...); err != nil {
    result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, err))
} else {
    result.Files = append(result.Files, fr)
}
```

### CR-02: antigravity legacy→unified migration can lose the codegraph entry entirely on a partial write failure

**File:** `internal/agents/antigravity.go:140-166`

**Issue:**

```go
if !fileExists(marker) && !fileExists(unified) && fileExists(legacy) {
    if legacyEntry, ok := readMcpEntry(legacy); ok {
        ...
        if err := writeJSONFile(unified, unifiedExisting); err == nil {
            result.Files = append(result.Files, FileResult{Path: unified, Action: ActionUpdated})
        }
        // err != nil here is silently ignored — execution falls through anyway
    }
    if fr, err := removeMcpEntry(legacy); err == nil && fr.Action == ActionRemoved {
        result.Files = append(result.Files, fr)
    }
}
...
if !fileExists(marker) {
    if err := os.MkdirAll(filepath.Dir(marker), 0o755); err == nil {
        if err := os.WriteFile(marker, []byte("migrated\n"), 0o644); err == nil {
            result.Files = append(result.Files, FileResult{Path: marker, Action: ActionCreated})
        }
    }
}
```

If `writeJSONFile(unified, ...)` at line 152 fails, the code does **not** stop or skip the legacy removal — `removeMcpEntry(legacy)` at line 156 runs unconditionally, stripping the codegraph entry from the only file that still had it. The subsequent `writeMcpEntry(unified, ...)` at line 161 (which overwrites whatever the migration step wrote with the current entry anyway, making the migration write's own success/failure moot for correctness) can fail for the same underlying reason (same file, same permission/disk problem). If it does, the `.migrated` marker at lines 168-174 is still written unconditionally (a different path, may not hit the same I/O failure), permanently recording "migration done" even though the entry now exists in **neither** the legacy nor the unified file. There is no rollback, no retry, and no error surfaced (per CR-01) — the user's antigravity MCP config silently reverts to unconfigured.

**Fix:** Only remove the legacy entry after confirming the unified write succeeded, and don't write the `.migrated` marker unless the final `writeMcpEntry` to `unified` also succeeded:

```go
migratedOK := true
if legacyEntry, ok := readMcpEntry(legacy); ok {
    ...
    if err := writeJSONFile(unified, unifiedExisting); err != nil {
        migratedOK = false
    } else {
        result.Files = append(result.Files, FileResult{Path: unified, Action: ActionUpdated})
    }
}
if migratedOK {
    if fr, err := removeMcpEntry(legacy); err == nil && fr.Action == ActionRemoved {
        result.Files = append(result.Files, fr)
    }
}
```
and guard the marker write on the final `writeMcpEntry(unified, ...)` call's own error.

### CR-03: Hermes YAML block matching is not CRLF-safe — breaks install idempotency and disables uninstall entirely for CRLF config files

**File:** `internal/agents/hermes.go:52-81` (`yamlBlockRange`), `internal/agents/hermes.go:92-122` (`yamlListBlockRange`)

**Issue:** Both functions locate a header line via exact string equality against the raw split line, with no trimming of a trailing `\r`:

```go
func yamlBlockRange(content, header string, indent int) (start, end int, found bool) {
	headerLine := strings.Repeat(" ", indent) + header
	lines := strings.Split(content, "\n")
	...
	for i, line := range lines {
		if line == headerLine {
```

If `$HERMES_HOME/config.yaml` has CRLF line endings (common on Windows, and this project ships Windows release binaries — a Windows-authored or Windows-tool-generated Hermes config is a realistic scenario), every line in `lines` carries a trailing `\r`. `"mcp_servers:\r" != "mcp_servers:"`, so `yamlBlockRange` never finds the existing `mcp_servers:`/`platform_toolsets:` header:

- On `Install`, `hermesSpliceMcpServersBlock`/`hermesAppendCliToolset` treat the block as absent and **append a second, duplicate top-level `mcp_servers:`/`platform_toolsets:` key** on every single run — not just once, since the header is never recognized as already present, `hermesConfigured` (used by `Detect`) also always reports `AlreadyConfigured: false`.
- On `Uninstall`, `hermesStripMcpServersBlock`/`hermesRemoveCliToolset`'s block-scoped removal (via the same `yamlBlockRange`) never finds the block either, so `codegraph uninstall --target hermes` is a **permanent no-op** for any CRLF-line-ended config, even though the file plainly contains a codegraph block.

Compare with `internal/agents/toml.go`'s equivalent `findTOMLTableRange`, which correctly uses `strings.TrimSpace(line) == header` and is therefore CRLF-safe — the two hand-rolled splicers are inconsistent on this point.

**Fix:** Trim a trailing `\r` (or use `strings.TrimRight(line, "\r")`) before the equality check in both `yamlBlockRange` and `yamlListBlockRange`, matching `toml.go`'s approach:

```go
if strings.TrimRight(line, "\r") == headerLine {
```

## Warnings

### WR-01: Release/bundle downloads have no size bound

**File:** `internal/upgrade/upgrade.go:181-194` (`downloadReleaseAsset`)

**Issue:** `io.ReadAll(resp.Body)` reads the entire HTTP response body into memory with no cap, for both the binary asset and its `.sigstore.json` bundle. Signature verification only happens *after* the full download completes (`upgrade.go:94-108`), so a compromised or misbehaving release asset (or CDN/cache poisoning at the GitHub Releases layer) can exhaust memory before verification ever runs and rejects it.

**Fix:** Wrap the body in `io.LimitReader` with a generous but bounded cap (e.g. 500 MB for the binary, a few MB for the bundle) and treat hitting the limit as a download error:

```go
const maxAssetBytes = 500 << 20
body := io.LimitReader(resp.Body, maxAssetBytes+1)
data, err := io.ReadAll(body)
if err == nil && len(data) > maxAssetBytes {
    return nil, fmt.Errorf("download %s: exceeds %d byte limit", url, maxAssetBytes)
}
```

### WR-02: No HTTP timeout anywhere in the upgrade network path

**File:** `internal/upgrade/release.go:38-44` (`newLatestRedirectClient`), `internal/upgrade/upgrade.go:184` (`http.Get`), `internal/upgrade/verify.go:44-50` (`fetchTrustedRoot`)

**Issue:** `newLatestRedirectClient` returns an `*http.Client{CheckRedirect: ...}` with no `Timeout` set; `downloadReleaseAsset` uses the package-level `http.Get` (which uses `http.DefaultClient`, also untimed); `fetchTrustedRoot` delegates to `sigstore-go`'s `root.FetchTrustedRoot()` with no context/deadline passed in. A hung or slow-lorising GitHub/Sigstore endpoint causes `codegraph upgrade` (or even `--check`) to block indefinitely with no way for the user to know it's stuck versus still working.

**Fix:** Set an explicit `Timeout` on every `*http.Client` used in this package (e.g. 30s for the redirect/API calls, a longer but still bounded timeout for the binary download), and thread a `context.Context` with a deadline through to `fetchTrustedRoot` if the `sigstore-go` API supports it.

### WR-03: No downgrade/rollback protection in `upgrade.Run`

**File:** `internal/upgrade/upgrade.go:54-120`

**Issue:** `Run` resolves `target` (either the pinned `opts.Version` or the resolved `latest`) and installs whatever version that is, as long as it's validly signed by the expected identity. There is no check that `target` is >= `currentVersion` (semver-aware or otherwise) when the user did not explicitly pin a version. If release infrastructure is ever compromised such that an older, validly-signed release with a known vulnerability gets served in place of `latest` (or an attacker with any control over the resolve path forces resolution to an old tag), `upgrade.Run` will happily "upgrade" the user to it — a classic rollback attack that signature verification alone does not prevent.

**Fix:** When no explicit version is pinned, refuse to swap if the resolved `latest` semver-compares as less than `currentVersion` (with a clear opt-in override via the existing explicit `[version]` argument for intentional downgrades):

```go
if opts.Version == "" && semverLess(latest, currentVersion) {
    return fmt.Errorf("upgrade: resolved version %s is older than the running version %s; pin explicitly to downgrade", latest, currentVersion)
}
```

### WR-04: Windows swap can leave the target binary completely missing if the restore-after-failure also fails

**File:** `internal/upgrade/swap.go:96-110` (`swapWindows`)

**Issue:**

```go
if err := os.Rename(targetPath, asidePath); err != nil {
    return fmt.Errorf("upgrade: rename running binary aside: %w", err)
}
if err := os.Rename(tmpPath, targetPath); err != nil {
    _ = os.Rename(asidePath, targetPath) // best-effort restore, don't brick the install
    return fmt.Errorf("upgrade: rename new binary into place: %w", err)
}
```

The restore attempt's error is discarded (`_ = ...`). If the second rename fails (e.g. the new binary is somehow locked or the disk fills up between the two renames) *and* the best-effort restore also fails, `targetPath` ends up pointing at nothing — the user's `codegraph` binary is gone, contradicting the function's own doc comment ("don't leave the install with no binary at targetPath at all"). This is exactly the failure mode a self-update tool needs to be most defensive about, since a broken update here can't be fixed by re-running the broken tool.

**Fix:** Check the restore's error and, if it also fails, surface both errors clearly and point the user at `asidePath` as the recovery path (the old binary is still there under `.old`, just not at the expected name):

```go
if err := os.Rename(tmpPath, targetPath); err != nil {
    if restoreErr := os.Rename(asidePath, targetPath); restoreErr != nil {
        return fmt.Errorf("upgrade: install failed (%v) AND restoring the original binary failed (%v); "+
            "your original binary is preserved at %s — rename it back manually", err, restoreErr, asidePath)
    }
    return fmt.Errorf("upgrade: rename new binary into place: %w", err)
}
```

### WR-05: `atomicWriteFile` doesn't preserve the target file's original permissions

**File:** `internal/agents/shared.go:293-318`

**Issue:** `atomicWriteFile` creates the temp file via `os.CreateTemp` (which on POSIX creates the file with mode `0600`) and renames it directly over the existing target — there is no `os.Chmod` step to match the original file's mode before the rename, unlike `internal/upgrade/swap.go:75` which explicitly does `os.Chmod(tmpPath, info.Mode().Perm()|0o111)` to preserve the original binary's permission bits. Every codegraph write to an agent config that previously had, say, `0644` permissions (the common default for `~/.claude.json`, `~/.cursor/mcp.json`, etc.) silently tightens it to `0600` on the very first install/uninstall run — a behavior change with no corresponding note anywhere in the tool's output.

**Fix:** Stat the existing file (if any) before writing and `os.Chmod` the temp file to match its mode prior to the rename, mirroring `swap.go`'s pattern:

```go
mode := os.FileMode(0o644)
if info, err := os.Stat(path); err == nil {
    mode = info.Mode().Perm()
}
if err := os.Chmod(tmpPath, mode); err != nil {
    os.Remove(tmpPath)
    return err
}
```

### WR-06: JSON-shaped config writes reformat the entire file (alphabetical key reorder, potential integer precision loss), not just the codegraph section

**File:** `internal/agents/shared.go:42-51` (`writeJSONFile`), used by `writeMcpEntry`/`removeMcpEntry`/`addClaudeAllowPermission`/`removeClaudeAllowPermission`

**Issue:** Every write funnels the *entire* decoded config through `json.MarshalIndent(data, "", "  ")` on a `map[string]any`. Go's `encoding/json` package unconditionally sorts map keys alphabetically when marshaling, and decodes all JSON numbers into `float64` when unmarshaling into `any` (losing exact representation for integers beyond `2^53`, and reformatting numeric literal styles). The package doc (`agents/types.go:1-11`) and several inline comments describe these as "surgical" edits that preserve "every sibling key... untouched," but that's only true at the *value* level — key **order** is not preserved on any write that actually changes something (the whole file gets re-serialized, alphabetized). For a user who tracks dotfiles like `~/.claude.json` in git, every `install`/`uninstall` run that touches the file produces a full-file diff unrelated to the actual change, and any large numeric IDs in the existing config risk silent reformatting.

**Fix:** If byte-level fidelity for untouched keys matters (as the comments claim), consider a JSON‑preserving patch approach analogous to what `opencode.go` already does via `hujson` (Parse → RFC 6902 Patch → Format → Pack) for the other JSON-shaped targets (`claude.go`, `cursor.go`, `gemini.go`, `kiro.go`, `antigravity.go`), instead of full `map[string]any` round-tripping. At minimum, document that key order is not preserved so the "surgical"/"preserves every sibling key" language in the comments isn't overstated.

### WR-07: `hermesRemoveCliToolset` removes the first matching line anywhere in the file, not scoped to `platform_toolsets.cli`

**File:** `internal/agents/hermes.go:268-283`

**Issue:**

```go
func hermesRemoveCliToolset(content string) string {
	lines := strings.Split(content, "\n")
	removed := false
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !removed && strings.TrimSpace(line) == "- "+hermesCliToolsetName {
			removed = true
			continue
		}
		out = append(out, line)
	}
	...
}
```

Unlike its counterpart `hermesAppendCliToolset` (which correctly scopes its search to the `platform_toolsets.cli` block via `yamlListBlockRange`), the removal function scans the *whole file* for the first line whose trimmed text exactly equals `"- mcp-codegraph"`. If a user's config happens to have an unrelated list elsewhere containing that exact literal item (e.g. a custom toolset list, or a copy-pasted example), `Uninstall` will delete that unrelated entry instead of (or in addition to, since it stops at the first match) the one actually appended under `platform_toolsets.cli`.

**Fix:** Scope the search to the `platform_toolsets.cli` block the same way `hermesAppendCliToolset` does, via `yamlListBlockRange`, and only remove the matching line within that range.

### WR-08: `releaseWorkflowRefPattern` is an unanchored prefix regex that authorizes any workflow in the repo

**File:** `internal/upgrade/verify.go:21-25`

**Issue:**

```go
releaseWorkflowRefPattern = "^https://github.com/" + releaseRepoSlug + "/"
```

This regex is anchored at the start only. Passed as `sanRegex` to `verify.NewShortCertificateIdentity(releaseOIDCIssuer, "", "", sanRegex)`, any certificate SAN that merely *starts with* `https://github.com/seanb4t/codegraph-go/` will pass the identity check — including a signature produced by an arbitrary, unrelated workflow file in the same repository (e.g. a CI workflow with `pull_request` triggers, which is a much weaker trust boundary than a tag-triggered release workflow), not just the intended release-publishing workflow. The file's own comment acknowledges this is "Phase-8-finalized" and not yet exercised against real releases, but it is already wired into `defaultVerify` (`upgrade.go:133-144`) and will be the production policy the day real signed releases ship.

**Fix:** Anchor the pattern to the specific release workflow file and ref type, e.g. `^https://github.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9]`, once the actual workflow filename is finalized (tracked as D-14 per the existing comment) — flagging now so it isn't forgotten before DIST-02 ships.

## Info

### IN-01: `checkWritable` is invoked twice per upgrade run

**File:** `internal/upgrade/upgrade.go:86`, `internal/upgrade/swap.go:19-29,46`

**Issue:** `Run` calls `checkWritable(targetPath)` directly (to fail fast before downloading, per its own comment) and `atomicSwap` calls it again internally a few lines later. Harmless — just duplicated directory-write-probe work on every successful upgrade — but worth consolidating so the precondition lives in one place.

**Fix:** Have `atomicSwap` accept a `alreadyCheckedWritable bool` (or just drop the internal call and rely on `Run`'s check, keeping `atomicSwap`'s own check only for callers that don't already pre-check, if any exist elsewhere).

### IN-02: `hermesAppendCliToolset`'s `cli:` search assumes a fixed 2-space parent indent

**File:** `internal/agents/hermes.go:217-263`

**Issue:** `yamlListBlockRange(parentBlock, "cli:", 2)` only finds an existing `cli:` key if it sits at exactly 2-space indentation under `platform_toolsets:`. This matches PyYAML's default emission (the documented common case) but a hand-authored config with a different indent style for the `cli:` key itself (not just its list items, which the code does handle per the Pitfall-5 comment) would not be found, causing a duplicate `cli:` mapping key to be appended — a milder instance of the same class of issue as WR-07/CR-03's exact-indent assumptions.

**Fix:** If broader indent tolerance is a design goal (as it clearly is for list items), consider detecting the actual `cli:` indent similarly to how list-item indent is detected, rather than hardcoding `2`.

---

_Reviewed: 2026-07-12T21:13:39Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
