package query

import (
	"strings"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// warnGlyph is the notice glyph: U+26A0 WARNING SIGN, UTF-8 bytes
// e2 9a a0, deliberately WITHOUT a trailing U+FE0F (ef b8 8f) variation
// selector — the same byte sequence internal/gitmeta's own (unexported)
// warnGlyph and this file's sibling staleBanner glyph use (D-11). Declared
// locally rather than imported since gitmeta.warnGlyph is unexported.
const warnGlyph = "⚠"

// WorktreeNotice returns m.Notice() followed by a blank line (D-12) — the
// ONE uniform mechanism prefixed onto all 7 non-status read tools
// (explore/node/search/callers/callees/impact/files) whenever a
// borrowed-index mismatch is detected. status is excluded: it embeds its
// own verbose Warning() form instead, wrapped as a blockquote by
// WorktreeWarningBlockquote below. Mirrors staleBanner's exact shape
// (render_markdown.go) — nil-safe via Mismatch.Notice()'s own nil-receiver
// guard, "" when there is nothing to warn about, so every call site needs
// no nil guard.
//
// A previously-proposed per-tool "_worktreeNotice" JSON-field hybrid was
// WITHDRAWN (CONTEXT.md D-12, corrected 2026-07-15): its premise was that a
// JSON contract needed protecting from a text-prefix, but MCP text content
// is consumed by a language model, not a parser — nothing in this repo,
// nor Claude Code, unmarshals it — and SURF-06 (D-16) moves every one of
// these tools to markdown regardless. There is one shape and one
// mechanism; do not reintroduce a per-tool notice mechanism.
func WorktreeNotice(m *gitmeta.Mismatch) string {
	notice := m.Notice()
	if notice == "" {
		return ""
	}
	return notice + "\n\n"
}

// WorktreeWarningBlockquote wraps m.Warning() as a markdown blockquote —
// "> ⚠ " prefixed onto the warning's first line, and every subsequent
// "\n" rewritten to "\n> " — for MCP status only (D-12), matching TS's
// mcp/tools.js construction. The CLI does NOT use this form: it prints
// the warning through its own warn-style line (plan 02-07), because TS
// ships two structurally different status renderings (D-17) and a
// blockquote is meaningless outside markdown.
func WorktreeWarningBlockquote(m *gitmeta.Mismatch) string {
	warning := m.Warning()
	if warning == "" {
		return ""
	}
	return "> " + warnGlyph + " " + strings.ReplaceAll(warning, "\n", "\n> ")
}
