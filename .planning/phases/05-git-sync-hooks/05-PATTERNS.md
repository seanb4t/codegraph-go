# Phase 5: Git Sync Hooks - Pattern Map

**Mapped:** 2026-07-16
**Files analyzed:** 8
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `internal/fsatomic/fsatomic.go` | utility | file-I/O | `internal/agents/shared.go` (`atomicWriteFile`, lines 312-360) | exact (verbatim extraction, D-09) |
| `internal/gitmeta/githooks.go` (new file, `IsGitRepo`/`HooksDir`) | utility | request-response (subprocess exec) | `internal/gitmeta/worktree.go` (`WorktreeRoot`/`CommonDir`, lines 35-84) | exact (same package, same exec contract) |
| `internal/gitmeta/githooks_test.go` | test | request-response | `internal/gitmeta/fixtures_test.go` (`runGit`/`initRepo` helpers) | exact |
| `internal/githooks/githooks.go` (markers, Install/Remove/Status) | service | file-I/O (CRUD on hook scripts) | `internal/agents/shared.go` (`upsertInstructionsEntry`/`replaceOrAppendMarkedSection`, lines 194-310) — **pattern shape only, semantics diverge per D-02/D-09** | role-match (structurally similar splice-and-write service, but marker/strip/delete rules are a verbatim TS port, not a reuse) |
| `internal/githooks/splice.go` (optional split) | utility | transform | same as above | role-match |
| `internal/cli/githooks.go` (new: `newGithooksCmd()` + 3 subcommands) | controller (CLI command) | request-response | `internal/cli/uninit.go` (single-subcommand shape) + `internal/cli/root.go` (AddCommand wiring) | exact |
| `internal/cli/init.go` (D-07 advisory insertion on success path) | controller | event-driven (post-success hook) | `internal/watch/policy.go` (`WatchDisabledReason` + injectable `Probe`) for the gate; `internal/cli/init.go` itself for the insertion point | exact |
| `internal/cli/uninit.go` (D-06 best-effort cleanup insertion) | controller | event-driven (post-removal hook) | `internal/cli/uninit.go` itself (existing `RunE` body, lines 26-56) | exact |
| `internal/agents/shared.go` (rewire `atomicWriteFile` → `fsatomic.WriteFile`) | utility | file-I/O | itself (pre-rewire) | exact |

## Pattern Assignments

### `internal/fsatomic/fsatomic.go` (utility, file-I/O)

**Analog:** `internal/agents/shared.go` lines 312-360 (`atomicWriteFile`)

**Extraction is verbatim** — copy the function body unchanged into a new package, rename to an exported `WriteFile(path, content string) error`, keep the doc comment's crash-safety and mode-preservation rationale, and add a package doc noting the deliberate narrowing (D-09: only the atomic write is extracted, NOT the marker-splice helpers).

```go
package fsatomic

import (
	"os"
	"path/filepath"
)

// WriteFile writes content to path via a temp file created in the same
// directory followed by os.Rename, so a crash or interrupt mid-write
// never leaves a truncated or corrupted file on disk. Callers across
// internal/agents and internal/githooks funnel every hook/config write
// through this one function.
//
// os.CreateTemp creates the temp file with mode 0600 on POSIX; if path
// already exists, its mode is preserved by chmod'ing the temp file to
// match before the rename — otherwise the first write to a file that
// previously had, say, 0644 permissions would silently tighten it to
// 0600. A new file gets the conventional 0644 default.
func WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".codegraph-write-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
```

**Rewiring `internal/agents/shared.go`:** replace the `atomicWriteFile` body (lines 312-360) with a thin call-through so every existing call site (`upsertInstructionsEntry` and others) is unaffected:

```go
func atomicWriteFile(path, content string) error {
	return fsatomic.WriteFile(path, content)
}
```

Add `"github.com/seanb4t/codegraph-go/internal/fsatomic"` to the existing import block (lines 1-9). Do not touch `replaceOrAppendMarkedSection`/`removeMarkedSection` (lines 194-295) — those stay in `internal/agents`, unreused by githooks (D-09/D-02).

---

### `internal/gitmeta/githooks.go` (utility, request-response)

**Analog:** `internal/gitmeta/worktree.go` lines 1-99 (package doc + `WorktreeRoot`/`CommonDir`/`realpath`)

**Imports pattern** (worktree.go lines 14-20):
```go
import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)
```
Reuse the existing package-level `gitTimeout = 5 * time.Second` constant (worktree.go line 28) — do not redeclare it.

**Core exec-contract pattern** (worktree.go lines 35-51, the template for both new functions):
```go
func WorktreeRoot(ctx context.Context, dir string) string {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	cmd.Stdin = nil // git must never be able to block on an interactive prompt
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return ""
	}
	return realpath(trimmed)
}
```

**New functions to add, following this exact shape:**
- `IsGitRepo(ctx, dir) bool` — `git rev-parse --is-inside-work-tree`; success + trimmed stdout `== "true"` → `true`, any error/other output → `false`. No `realpath` needed (boolean result).
- `HooksDir(ctx, projectRoot) string` — `git rev-parse --git-path hooks`; error/empty → `""`; if `filepath.IsAbs(out)` return as-is, else `filepath.Join(projectRoot, out)` — mirrors `CommonDir`'s relative-path resolution at worktree.go lines 76-83 (do NOT call `realpath` here since D-04 says pass absolute through / resolve relative against project root, not symlink-resolve).

**Relative-path resolution reference** (worktree.go lines 76-83, `CommonDir`):
```go
resolved := trimmed
if !filepath.IsAbs(resolved) {
	resolved = filepath.Join(dir, resolved)
}
return realpath(resolved)
```

**Package doc note:** worktree.go's existing package comment (lines 1-12) already says "so Phase 5's git sync hooks can reuse it unchanged" — no changes needed there, just add the new file as a sibling.

---

### `internal/gitmeta/githooks_test.go` (test, request-response)

**Analog:** `internal/gitmeta/fixtures_test.go` (`runGit`, `initRepo`, and the linked-worktree fixture helper)

```go
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{
		"-c", "init.defaultBranch=main",
		"-c", "user.name=codegraph-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always",
	}
	full := append(append([]string{}, base...), args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("git %v failed (git missing or fixture unsupported here): %v: %s", args, err, string(out))
	}
	return string(out)
}
```
Reuse `runGit`/`initRepo` directly from `fixtures_test.go` (same package `gitmeta`, no import needed) — do not duplicate. For the linked-worktree D-12 case, add `runGit(t, main, "worktree", "add", "-b", "feature", wt)`. For `core.hooksPath`, add `runGit(t, dir, "config", "core.hooksPath", relOrAbsPath)` before invoking `HooksDir`.

---

### `internal/githooks/githooks.go` (service, file-I/O — CRUD on hook scripts)

**Analog (structural shape only — NOT semantics):** `internal/agents/shared.go` lines 194-310 for how a splice-and-write service in this codebase is organized (small pure functions + a thin orchestrator + `atomicWriteFile`/`fsatomic.WriteFile` at the write boundary). **Do NOT copy `replaceOrAppendMarkedSection`'s logic** — see the verbatim TS semantics CONTEXT.md/RESEARCH.md already transcribed (D-02/D-09), reproduced here for direct copy-in:

**Imports:**
```go
import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/fsatomic"
	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)
```

**Marker constants and block** (verbatim TS bytes, `sync/git-hooks.js:58-59,102-114` — copy byte-for-byte, do not hand-retype):
```go
const markerBegin = "# >>> codegraph sync hook >>>"
const markerEnd = "# <<< codegraph sync hook <<<"

var defaultSyncHooks = []string{"post-commit", "post-merge", "post-checkout"}

func markerBlock() string {
	return strings.Join([]string{
		markerBegin,
		"# Keeps the CodeGraph index fresh while the live file watcher is off",
		"# (e.g. WSL2 /mnt drives). Runs in the background so it never blocks git.",
		"# Managed by codegraph; remove with `codegraph uninit` or delete this block.",
		"if command -v codegraph >/dev/null 2>&1; then",
		"  ( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1",
		"fi",
		markerEnd,
	}, "\n")
}
```

**Install pattern** (port of `sync/git-hooks.js:155-186`, using `fsatomic.WriteFile` at the write boundary per D-05):
```go
func Install(ctx context.Context, projectRoot string) InstallResult {
	hooksDir := gitmeta.HooksDir(ctx, projectRoot)
	if hooksDir == "" {
		return InstallResult{Skipped: "not a git repository"}
	}
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return InstallResult{HooksDir: hooksDir, Skipped: "could not access the git hooks directory"}
	}
	block := markerBlock()
	var installed []string
	for _, hook := range defaultSyncHooks {
		file := filepath.Join(hooksDir, hook)
		var content string
		if existing, err := os.ReadFile(file); err == nil {
			base := strings.TrimRight(stripMarkerBlock(string(existing)), " \t\n")
			if base != "" {
				content = base + "\n\n" + block + "\n"
			} else {
				content = "#!/bin/sh\n" + block + "\n"
			}
		} else {
			content = "#!/bin/sh\n" + block + "\n"
		}
		if err := fsatomic.WriteFile(file, content); err != nil {
			continue
		}
		_ = os.Chmod(file, 0o755) // best-effort, TS swallows chmod errors too (Pitfall 4)
		installed = append(installed, hook)
	}
	return InstallResult{Installed: installed, HooksDir: hooksDir}
}
```

**Strip pattern (trimmed-line matching)** — verbatim TS logic, `sync/git-hooks.js:116-134`:
```go
func stripMarkerBlock(content string) string {
	lines := strings.Split(content, "\n")
	var kept []string
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == markerBegin {
			inBlock = true
			continue
		}
		if trimmed == markerEnd {
			inBlock = false
			continue
		}
		if !inBlock {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}
```

**Effectively-empty gate (deletion trigger)** — verbatim TS logic, `sync/git-hooks.js:136-141`:
```go
func isEffectivelyEmpty(content string) bool {
	for _, l := range strings.Split(content, "\n") {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#!") {
			return false
		}
	}
	return true
}
```

**Remove pattern** (port of `sync/git-hooks.js:192-216`; file deletion is plain `os.Remove`, per D-05):
```go
func Remove(ctx context.Context, projectRoot string) RemoveResult {
	hooksDir := gitmeta.HooksDir(ctx, projectRoot)
	if hooksDir == "" {
		return RemoveResult{Skipped: "not a git repository"}
	}
	var removed []string
	for _, hook := range defaultSyncHooks {
		file := filepath.Join(hooksDir, hook)
		original, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if !strings.Contains(string(original), markerBegin) {
			continue
		}
		stripped := stripMarkerBlock(string(original))
		if isEffectivelyEmpty(stripped) {
			if err := os.Remove(file); err != nil {
				continue
			}
		} else {
			content := strings.TrimRight(stripped, " \t\n") + "\n"
			if err := fsatomic.WriteFile(file, content); err != nil {
				continue
			}
			_ = os.Chmod(file, 0o755)
		}
		removed = append(removed, hook)
	}
	return RemoveResult{Removed: removed, HooksDir: hooksDir}
}
```
Note the Go-idiomatic field naming (`Removed` not `Installed`) per RESEARCH.md's explicit call-out that TS's `{installed: removed}` naming quirk should NOT be copied.

**Status pattern** (port of `sync/git-hooks.js:218-226`, extended for per-hook detail per D-11):
```go
func Status(ctx context.Context, projectRoot string) StatusResult {
	hooksDir := gitmeta.HooksDir(ctx, projectRoot)
	if hooksDir == "" {
		return StatusResult{Skipped: "not a git repository"}
	}
	var hooks []HookStatus
	for _, hook := range defaultSyncHooks {
		file := filepath.Join(hooksDir, hook)
		installed := false
		if content, err := os.ReadFile(file); err == nil {
			installed = strings.Contains(string(content), markerBegin)
		}
		hooks = append(hooks, HookStatus{Name: hook, Installed: installed})
	}
	return StatusResult{HooksDir: hooksDir, Hooks: hooks}
}
```

**Result types (Go-idiomatic, Claude's discretion per D-11):**
```go
type InstallResult struct {
	Installed []string
	HooksDir  string
	Skipped   string
}

type RemoveResult struct {
	Removed  []string
	HooksDir string
	Skipped  string
}

type HookStatus struct {
	Name      string
	Installed bool
}

type StatusResult struct {
	Hooks    []HookStatus
	HooksDir string
	Skipped  string
}
```

---

### `internal/cli/githooks.go` (controller, request-response)

**Analogs:** `internal/cli/uninit.go` (single-purpose subcommand shape, `targetRoot` reuse, `RunE` structure) + `internal/cli/root.go` (`AddCommand` wiring)

**Imports pattern** (uninit.go lines 1-11):
```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/githooks"
)
```

**Command tree shape** (mirrors `newUninitCmd`, uninit.go lines 19-62 — `targetRoot`, `Args: cobra.MaximumNArgs(1)`, `RunE` closure, flags declared after the `&cobra.Command{}` literal):
```go
func newGithooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "githooks",
		Short: "Manage git sync hooks (post-commit/post-merge/post-checkout)",
	}
	cmd.AddCommand(newGithooksInstallCmd(), newGithooksRemoveCmd(), newGithooksStatusCmd())
	return cmd
}

func newGithooksInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [path]",
		Short: "Install marker-fenced git sync hooks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}
			result := githooks.Install(cmd.Context(), root)
			// report result.Installed / result.Skipped to cmd.OutOrStdout()
			return nil
		},
	}
}
```
`install`/`remove`/`status` all follow this identical shape — reuse the package-level `targetRoot` (init.go lines 79-86), no new path-resolution helper needed.

**Root wiring** (root.go lines 45-50, `AddCommand` list — append, do not reorder existing entries):
```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
	newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
	newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
	newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd(),
	newDaemonCmd(), newUnlockCmd(), newVersionCmd(), newTelemetryCmd(),
	newUpgradeCmd(), newInstallCmd(), newUninstallCmd(), newMigrateCmd(),
	newGithooksCmd())
```

---

### `internal/cli/init.go` (D-07 advisory insertion)

**Analog:** `internal/watch/policy.go`'s `WatchDisabledReason`/`Probe` (the gate) + `init.go`'s own success path (insertion point: after `printSummary(cmd, stats, quiet, verbose)` at line 67, before `return nil`)

**Gate pattern** (policy.go lines 95-117, call shape only — `Probe{}` zero value defaults to `os.Getenv`/`DetectWSL`):
```go
reason := watch.WatchDisabledReason(root, watch.Probe{})
if reason != "" {
	fmt.Fprintf(cmd.OutOrStdout(), "Live file watching is disabled here — %s.\n", reason)
	fmt.Fprintln(cmd.OutOrStdout(), "Until you re-sync, the CodeGraph index stays frozen — it will not pick up edits on its own.")
	if !gitmeta.IsGitRepo(cmd.Context(), root) {
		fmt.Fprintln(cmd.OutOrStdout(), "Run `codegraph sync` after changing files to refresh the index.")
	} else {
		status := githooks.Status(cmd.Context(), root)
		installed := false
		for _, h := range status.Hooks {
			if h.Installed {
				installed = true
				break
			}
		}
		if installed {
			fmt.Fprintln(cmd.OutOrStdout(), "Git sync hooks are already installed — the index refreshes after commit / pull / checkout.")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Run `codegraph githooks install` to keep the index fresh automatically.")
		}
	}
}
```
D-13's test-seam requirement: this call must go through an injectable `watch.Probe` reachable from a test that forces "disabled" (e.g. `Probe{NoWatch: true}` or an env override) — do not hardcode `watch.Probe{}` if that breaks the test seam; thread the `Probe` the same way `internal/daemon` already does (see `internal/watch/policy.go` doc comments for the existing injection convention).

Add imports `"github.com/seanb4t/codegraph-go/internal/watch"`, `"github.com/seanb4t/codegraph-go/internal/gitmeta"`, `"github.com/seanb4t/codegraph-go/internal/githooks"` to init.go's existing import block (lines 1-11).

---

### `internal/cli/uninit.go` (D-06 cleanup insertion)

**Analog:** its own existing `RunE` body (lines 26-56) — insert the cleanup call right after the successful `os.RemoveAll(codegraphDir)` (line 51) and before the final success message (line 54), non-fatal (never returns the cleanup's own error):

```go
if err := os.RemoveAll(codegraphDir); err != nil {
	return err
}
if result := githooks.Remove(cmd.Context(), root); len(result.Removed) > 0 {
	fmt.Fprintf(cmd.OutOrStdout(), "Removed git %s sync hook%s\n",
		strings.Join(result.Removed, ", "), plural(len(result.Removed)))
}
fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", codegraphDir)
return nil
```
Add `"github.com/seanb4t/codegraph-go/internal/githooks"` to uninit.go's import block (lines 1-11); `strings` is already imported.

## Shared Patterns

### gitmeta exec contract (single-seam confinement)
**Source:** `internal/gitmeta/worktree.go` lines 35-51 (`WorktreeRoot`)
**Apply to:** `IsGitRepo`, `HooksDir` — the ONLY two new git-exec call sites this phase introduces. No other file (`internal/githooks`, `internal/cli`) may call `exec.Command("git", ...)` directly.
```go
ctx, cancel := context.WithTimeout(ctx, gitTimeout)
defer cancel()
cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
cmd.Dir = dir
cmd.Stdin = nil
out, err := cmd.Output()
if err != nil { return false }
```

### Atomic write (crash-safety)
**Source:** `internal/fsatomic/fsatomic.go` (this phase's own extraction of `internal/agents/shared.go:312-360`)
**Apply to:** every hook-file write in `internal/githooks.Install`/`Remove` (never `os.WriteFile` directly) — deletion still uses plain `os.Remove` (D-05).

### targetRoot path resolution
**Source:** `internal/cli/init.go` lines 79-86
**Apply to:** all three `githooks` subcommands' `[path]` argument — reuse the existing package-level function, do not reimplement.

### Best-effort / non-fatal degrade-to-message
**Source:** `internal/watch/policy.go`'s "every function degrades to a safe zero value on ANY failure" convention (doc comment, gitmeta package, lines 8-11) + `internal/cli/uninit.go`'s "already absent → clean message, not error" branch (lines 33-38)
**Apply to:** `githooks install` in a non-repo (skip message, exit 0), `init`'s D-07 advisory (never blocks init success), `uninit`'s D-06 cleanup (never fails uninit).

## No Analog Found

None — every file in scope has a strong (exact or role-match) analog in the existing codebase. The only genuinely new *semantics* (not new *code shape*) are the marker-strip/effectively-empty/shebang-seed rules, which are a verbatim TS port rather than a codebase analog by design (D-02 explicitly forbids reusing `internal/agents`' splice helpers).

## Metadata

**Analog search scope:** `internal/agents/`, `internal/gitmeta/`, `internal/cli/`, `internal/watch/`
**Files scanned:** `internal/agents/shared.go`, `internal/gitmeta/worktree.go`, `internal/gitmeta/detect.go`, `internal/gitmeta/fixtures_test.go`, `internal/cli/init.go`, `internal/cli/uninit.go`, `internal/cli/root.go`, `internal/cli/sync.go`, `internal/watch/policy.go`
**Pattern extraction date:** 2026-07-16
