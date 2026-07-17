// Package githooks manages CodeGraph's marker-fenced git sync hooks
// (post-commit/post-merge/post-checkout) — a verbatim port of TS
// sync/git-hooks.js (D-01/D-02). Every write funnels through
// internal/fsatomic.WriteFile (D-05); hooks-dir resolution and the
// git-repo probe come from internal/gitmeta (IsGitRepo/HooksDir) — this
// package never shells out to git directly.
package githooks

import (
	"context"
	"fmt"
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
//
// The second return value is false when a begin marker is found with no
// matching end marker anywhere after it (a malformed/hand-edited hook
// file). TS's own stripMarkerBlock (sync/git-hooks.js:116-134) treats an
// unterminated begin marker as "block extends to EOF", silently dropping
// every line after it — a genuine data-loss bug (CR-01). D-02/D-03 lock
// verbatim TS semantics for well-formed marker blocks, but that guarantee
// does not extend to a malformed-input data-loss path; this is a
// deliberate, documented Go-only divergence (same convention as Phase 3's
// D-13 wording divergence), scoped narrowly to "don't destroy the user's
// file" — callers must treat ok==false as "do not trust this strip" and
// leave the file untouched rather than writing the truncated result back.
func stripMarkerBlock(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	var kept []string
	inBlock := false
	sawUnterminatedBegin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == markerBegin {
			inBlock = true
			sawUnterminatedBegin = true
			continue
		}
		if trimmed == markerEnd {
			inBlock = false
			sawUnterminatedBegin = false
			continue
		}
		if !inBlock {
			kept = append(kept, line)
		}
	}
	if sawUnterminatedBegin {
		return content, false
	}
	return strings.Join(kept, "\n"), true
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
// clean-skip contract. Errors accumulates one entry per hook that failed
// its individual write (e.g. unwritable file, read-only mount, disk full)
// so callers can surface *why* Installed came back short of all three
// hooks instead of failing silently (WR-01).
type InstallResult struct {
	Installed []string
	HooksDir  string
	Skipped   string
	Errors    []error
}

// RemoveResult reports the outcome of Remove. Removed lists the hooks that
// had a codegraph block stripped (file deleted or rewritten). Uses the
// Go-idiomatic field name Removed rather than TS's `{installed: removed}`
// naming quirk (RESEARCH.md note on removeGitSyncHook's result shape).
// Errors accumulates one entry per hook that failed its individual
// delete/write (WR-01) — the loop still continues past a failure, but the
// failure is no longer silently discarded.
type RemoveResult struct {
	Removed  []string
	HooksDir string
	Skipped  string
	Errors   []error
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
//
// Concurrency (WR-02): each individual hook write is atomic and
// crash-safe via fsatomic.WriteFile, but the surrounding read-modify-write
// sequence (read existing content, compute the new body, write it back) is
// not. Install is not safe to call concurrently against the same
// projectRoot — two overlapping Install/Remove invocations (or Install
// racing init's advisory path) can race on the same hook file and produce
// a lost update, with neither caller aware anything raced. Callers that
// need concurrent-safety must serialize their own calls (e.g. a lockfile
// around the hooks-dir mutation).
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
	var errs []error
	for _, hook := range defaultSyncHooks {
		file := filepath.Join(hooksDir, hook)
		var content string
		if existing, err := os.ReadFile(file); err == nil {
			base := string(existing)
			// An unterminated begin marker means the strip can't be
			// trusted (CR-01) — fall back to the raw existing content as
			// the base rather than risk truncating it, and let the fresh
			// block get appended after it (a stray dangling marker left
			// in place beats silently destroying the user's file).
			if stripped, ok := stripMarkerBlock(base); ok {
				base = stripped
			}
			base = strings.TrimRight(base, " \t\n")
			if base != "" {
				content = base + "\n\n" + block + "\n"
			} else {
				content = "#!/bin/sh\n" + block + "\n"
			}
		} else {
			content = "#!/bin/sh\n" + block + "\n"
		}
		if err := fsatomic.WriteFile(file, content); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", hook, err))
			continue
		}
		_ = os.Chmod(file, 0o755) // best-effort, TS swallows chmod errors too (Pitfall 4)
		installed = append(installed, hook)
	}
	return InstallResult{Installed: installed, HooksDir: hooksDir, Errors: errs}
}

// Remove strips codegraph's marker block from each of
// post-commit/post-merge/post-checkout (verbatim port of TS
// removeGitSyncHook, sync/git-hooks.js:192-216, D-02/D-05). Only files
// that actually contain the begin marker are touched — a hook never
// installed by codegraph, or an absent file, is skipped with no error. If
// the remainder after stripping is effectively empty (isEffectivelyEmpty),
// the file is deleted entirely via os.Remove; otherwise the trimmed
// remainder + a trailing newline is written via fsatomic.WriteFile and
// re-chmod'd 0755 (best-effort). Running Remove twice is a no-op on the
// second run (files already gone or already clean). In a non-repo,
// returns Skipped "not a git repository".
//
// Concurrency (WR-02): same caveat as Install — the read-modify-write
// sequence around each hook file is not atomic as a whole, only the
// individual fsatomic.WriteFile call is. Remove is not safe to call
// concurrently against the same projectRoot, including racing an Install
// against the same hooks directory.
func Remove(ctx context.Context, projectRoot string) RemoveResult {
	hooksDir := gitmeta.HooksDir(ctx, projectRoot)
	if hooksDir == "" {
		return RemoveResult{Skipped: "not a git repository"}
	}

	var removed []string
	var errs []error
	for _, hook := range defaultSyncHooks {
		file := filepath.Join(hooksDir, hook)
		original, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if !strings.Contains(string(original), markerBegin) {
			continue
		}
		stripped, ok := stripMarkerBlock(string(original))
		if !ok {
			// Unterminated begin marker (CR-01): don't trust the strip.
			// Leave the file untouched and don't report it as removed —
			// treating "no end marker" as "block extends to EOF" would
			// silently destroy any user content after the dangling
			// marker.
			continue
		}
		if isEffectivelyEmpty(stripped) {
			if err := os.Remove(file); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", hook, err))
				continue
			}
		} else {
			content := strings.TrimRight(stripped, " \t\n") + "\n"
			if err := fsatomic.WriteFile(file, content); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", hook, err))
				continue
			}
			_ = os.Chmod(file, 0o755) // best-effort, TS swallows chmod errors too (Pitfall 4)
		}
		removed = append(removed, hook)
	}
	return RemoveResult{Removed: removed, HooksDir: hooksDir, Errors: errs}
}

// Status reports per-hook install state for all three sync hooks
// (extends TS isSyncHookInstalled's aggregate some() with per-hook detail,
// D-11). A hook is Installed when its file exists and contains the begin
// marker — this includes hooks installed by TS CodeGraph itself, since the
// markers are byte-identical (D-03). In a non-repo, returns Skipped
// "not a git repository".
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
	return StatusResult{Hooks: hooks, HooksDir: hooksDir}
}
