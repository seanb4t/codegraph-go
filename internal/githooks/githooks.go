// Package githooks manages CodeGraph's marker-fenced git sync hooks
// (post-commit/post-merge/post-checkout) — a verbatim port of TS
// sync/git-hooks.js (D-01/D-02). Every write funnels through
// internal/fsatomic.WriteFile (D-05); hooks-dir resolution and the
// git-repo probe come from internal/gitmeta (IsGitRepo/HooksDir) — this
// package never shells out to git directly.
package githooks

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/fsatomic"
	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// markerBegin/markerEnd are verbatim TS bytes (sync/git-hooks.js:58-59) —
// a detection key, not documentation. Byte-identical markers mean hooks
// installed by TS CodeGraph are recognized and managed by this binary.
const (
	markerBegin = "# >>> codegraph sync hook >>>"
	markerEnd   = "# <<< codegraph sync hook <<<"
)

// defaultSyncHooks is the fixed hook trio (TS DEFAULT_SYNC_HOOKS), always
// processed in this order.
var defaultSyncHooks = []string{"post-commit", "post-merge", "post-checkout"}

// markerBlock returns the 8-line marker-fenced shell snippet, verbatim TS
// bytes (sync/git-hooks.js:102-114) — do not reformat or "clean up" any
// line, including the exact subshell-backgrounding invocation
// `( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1` (Pitfall 5: a
// single `&` can let git block on the backgrounded job).
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

// stripMarkerBlock removes any codegraph marker block from content,
// matching markers on the TRIMMED line (an indented marker still counts,
// TS sync/git-hooks.js:116-134) so content outside the block — including
// blank lines — is preserved verbatim. Content with no markers is returned
// unchanged.
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

// isEffectivelyEmpty reports whether every trimmed line in content is
// blank or shebang-prefixed (TS sync/git-hooks.js:136-141) — the gate that
// decides whether Remove deletes the hook file entirely rather than
// leaving a bare shebang behind.
func isEffectivelyEmpty(content string) bool {
	for _, l := range strings.Split(content, "\n") {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#!") {
			return false
		}
	}
	return true
}

// InstallResult reports the outcome of Install. Installed lists the hooks
// actually written, in the fixed defaultSyncHooks order. Skipped is set
// (and Installed left empty) when the target isn't a git repository or the
// hooks directory couldn't be created — never an error, per D-04's
// clean-skip contract.
type InstallResult struct {
	Installed []string
	HooksDir  string
	Skipped   string
}

// RemoveResult reports the outcome of Remove. Removed lists the hooks that
// had a codegraph block stripped (file deleted or rewritten). Uses the
// Go-idiomatic field name Removed rather than TS's `{installed: removed}`
// naming quirk (RESEARCH.md note on removeGitSyncHook's result shape).
type RemoveResult struct {
	Removed  []string
	HooksDir string
	Skipped  string
}

// HookStatus is one hook's install state, as reported by Status.
type HookStatus struct {
	Name      string
	Installed bool
}

// StatusResult reports per-hook install state for all three sync hooks.
type StatusResult struct {
	Hooks    []HookStatus
	HooksDir string
	Skipped  string
}

// Install writes the marker-fenced sync-hook block into each of
// post-commit/post-merge/post-checkout, in that fixed order (verbatim port
// of TS installGitSyncHook, sync/git-hooks.js:155-186, D-02/D-05). For each
// hook file: any existing content has a prior codegraph block stripped and
// trailing whitespace trimmed; if what remains is non-empty, the current
// block is appended after a blank-line separator; otherwise (no existing
// file, or an effectively-empty base) the file is seeded with
// "#!/bin/sh\n" + block. This is strip-then-append-at-end, not in-place
// replacement (Pitfall 2). Every write goes through fsatomic.WriteFile;
// chmod 0755 is best-effort (Pitfall 4, TS swallows chmod errors too). In
// a non-repo, returns Skipped "not a git repository" and writes nothing.
//
// Note (verbatim TS quirk, confirmed against sync/git-hooks.js): the very
// first install of a fresh hook file seeds "#!/bin/sh\n"+block (no
// blank-line separator), but the moment that file is read back on a
// second install, the surviving "#!/bin/sh" line is treated as non-empty
// base content, so the round-tripped form becomes "#!/bin/sh\n\n"+block
// (one blank line inserted). From that second install onward the
// round-tripped form is a stable fixed point — re-installing again never
// changes it. Only the very first-vs-second install transition adds that
// one blank line; this is TS's real behavior, faithfully reproduced here,
// not a Go-side bug.
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
