// Package claudeassets embeds this repository's own canonical .claude/
// package — the Claude Code skill, hooks fragment, and SessionStart nudge
// script authored and dogfooded in Phase 6 — so the running codegraph
// binary can install a byte-identical copy into any user's Claude Code
// read locations (Phase 7, AGENT-01).
//
// This file MUST live at the repository root, not under internal/. Go's
// //go:embed patterns may not contain ".." path elements (golang/go#46056)
// and are resolved only relative to, or below, the directory containing
// the source file that carries the directive. .claude/ and internal/ are
// sibling directories at the repository root, so no file under
// internal/agents/ (or any other package directory) can embed anything
// under .claude/ — the repository root is the only directory that is an
// ancestor of both. A later refactor that tries to relocate this file
// into internal/ will fail to compile, which is the intended guardrail
// (07-RESEARCH.md Critical Finding 2, empirically verified this repo's
// own session).
//
// Because the declared package name ("claudeassets") does not match the
// module path's last element ("codegraph-go"), every importer must use a
// named import: claudeassets "github.com/seanb4t/codegraph-go".
package claudeassets

import "embed"

// FS embeds exactly three files, named explicitly rather than as a
// directory pattern over .claude/skills/codegraph/ — embed's default
// dot/underscore filtering does not exclude
// .claude/skills/codegraph/verification/, so a directory pattern would
// also ship Phase 6's rehearsal-transcript evidence to every install
// target (07-RESEARCH.md Pitfall 2).
//
//go:embed .claude/skills/codegraph/SKILL.md
//go:embed .claude/hooks/hooks.json
//go:embed .claude/hooks/session-nudge.sh
var FS embed.FS

const (
	// SkillMarkdownPath is FS's path to the Claude Code skill's SKILL.md.
	SkillMarkdownPath = ".claude/skills/codegraph/SKILL.md"
	// HooksFragmentPath is FS's path to the versioned go:embed source for
	// the SessionStart hooks fragment (Phase 7 D-04) — not itself read by
	// Claude Code in this repository (internal/agents/hookpackage_test.go);
	// internal/agents derives the install-time hooks.SessionStart blocks
	// from it.
	HooksFragmentPath = ".claude/hooks/hooks.json"
	// SessionNudgeScriptPath is FS's path to the SessionStart nudge script.
	SessionNudgeScriptPath = ".claude/hooks/session-nudge.sh"
)

// SkillMarkdown returns the embedded SKILL.md content.
func SkillMarkdown() ([]byte, error) { return FS.ReadFile(SkillMarkdownPath) }

// HooksFragment returns the embedded hooks.json fragment content.
func HooksFragment() ([]byte, error) { return FS.ReadFile(HooksFragmentPath) }

// SessionNudgeScript returns the embedded session-nudge.sh content.
func SessionNudgeScript() ([]byte, error) { return FS.ReadFile(SessionNudgeScriptPath) }
