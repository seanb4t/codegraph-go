package gitmeta

// warnGlyph is the notice glyph: U+26A0 WARNING SIGN, UTF-8 bytes
// e2 9a a0, deliberately WITHOUT a trailing U+FE0F (ef b8 8f) variation
// selector. This is the exact glyph internal/query/render_markdown.go's
// staleBanner already uses — it is NOT the emoji-presentation "⚠️" variant
// Phase 1's "no covering tests" warning uses. Byte-parity here is not
// caught by the compiler; getting it wrong is a silent divergence from TS
// (D-11) that no test outside notice_test.go's explicit byte assertions
// would catch.
const warnGlyph = "⚠"

// Warning renders the verbose, multi-line form of a detected mismatch,
// used by `status` only (D-12). Returns "" on a nil receiver so callers
// never need a nil guard — the same shape
// internal/query/render_markdown.go's staleBanner uses. Ported verbatim
// from TS sync/worktree.js's worktreeMismatchWarning (D-01/D-11); do not
// paraphrase, including the quoted "codegraph init -i" advice.
func (m *Mismatch) Warning() string {
	if m == nil {
		return ""
	}
	return "This CodeGraph index belongs to a different git working tree.\n" +
		"  Running in: " + m.WorktreeRoot + "\n" +
		"  Index from: " + m.IndexRoot + "\n" +
		"Results reflect that tree's code (often a different branch), not this worktree — " +
		"symbols changed only here are missing. Run \"codegraph init -i\" in this worktree " +
		"for a worktree-local index."
}

// Notice renders the compact, single-line form of a detected mismatch,
// prefixed onto the other seven read tools' output (D-12). Returns "" on a
// nil receiver. Ported verbatim from TS sync/worktree.js's
// worktreeMismatchNotice (D-01/D-11); do not paraphrase.
func (m *Mismatch) Notice() string {
	if m == nil {
		return ""
	}
	return warnGlyph + " CodeGraph results below come from a different git worktree (" + m.IndexRoot + "), " +
		"not where you're working (" + m.WorktreeRoot + ") — they may reflect another branch, " +
		"and symbols changed only here are missing. Run \"codegraph init -i\" here for a " +
		"worktree-local index."
}
