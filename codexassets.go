// Package claudeassets (this file): Codex CLI's own embedded assets
// (Phase 7, CODEX-05) — the hand-authored .codex/hooks/ package this
// repository dogfoods and every install target's copy is rendered from.
// This mirrors claudeassets.go's Claude Code assets file above; see that
// file's doc comment for why this MUST live at the repository root, not
// under internal/ — the same golang/go#46056 sibling-of-root embed rule
// applies here unchanged, and the two files share one package
// declaration deliberately, so both harnesses' assets are reached through
// the same named import (claudeassets "github.com/seanb4t/codegraph-go").
package claudeassets

import "embed"

// CodexFS embeds exactly three files: the Codex PreToolUse hooks.json
// fragment and its two guard templates. Local and global get separate
// templates — deliberately named differently from the file install
// actually writes (codegraph-pretooluse.sh) so an install run inside this
// very repository never overwrites either template — because D-22 gives
// them different indexed-repo checks: the local guard derives its root
// from its own invocation path, the global guard checks $PWD.
//
//go:embed .codex/hooks/hooks.json
//go:embed .codex/hooks/codegraph-pretooluse-local.sh
//go:embed .codex/hooks/codegraph-pretooluse-global.sh
var CodexFS embed.FS

const (
	// CodexHooksFragmentPath is CodexFS's path to the embedded Codex
	// PreToolUse hooks.json fragment.
	CodexHooksFragmentPath = ".codex/hooks/hooks.json"
	// CodexPreToolUseGuardLocalPath is CodexFS's path to the unrendered
	// local-scope guard template.
	CodexPreToolUseGuardLocalPath = ".codex/hooks/codegraph-pretooluse-local.sh"
	// CodexPreToolUseGuardGlobalPath is CodexFS's path to the unrendered
	// global-scope guard template.
	CodexPreToolUseGuardGlobalPath = ".codex/hooks/codegraph-pretooluse-global.sh"
)

// CodexHooksFragment returns the embedded Codex hooks.json fragment
// content.
func CodexHooksFragment() ([]byte, error) { return CodexFS.ReadFile(CodexHooksFragmentPath) }

// CodexPreToolUseGuardLocalTemplate returns the embedded, unrendered local
// guard template.
func CodexPreToolUseGuardLocalTemplate() ([]byte, error) {
	return CodexFS.ReadFile(CodexPreToolUseGuardLocalPath)
}

// CodexPreToolUseGuardGlobalTemplate returns the embedded, unrendered
// global guard template.
func CodexPreToolUseGuardGlobalTemplate() ([]byte, error) {
	return CodexFS.ReadFile(CodexPreToolUseGuardGlobalPath)
}
