// Package githooks manages CodeGraph's marker-fenced git sync hooks
// (post-commit/post-merge/post-checkout) — a verbatim port of TS
// sync/git-hooks.js (D-01/D-02). Every write funnels through
// internal/fsatomic.WriteFile (D-05); hooks-dir resolution and the
// git-repo probe come from internal/gitmeta (IsGitRepo/HooksDir) — this
// package never shells out to git directly.
package githooks

import (
	"strings"
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
