package agents

// codegraphSectionStart and codegraphSectionEnd are the exact marker
// fences every install/uninstall must reproduce byte-for-byte (D-01a).
// This is a hard, stable contract, not a fresh choice: an uninstall must
// recognize the marker block an earlier install wrote, regardless of
// which agent wrote it. Do not alter this text.
const (
	codegraphSectionStart = "<!-- CODEGRAPH_START -->"
	codegraphSectionEnd   = "<!-- CODEGRAPH_END -->"
)

// codegraphInstructionsBlock is the short marker-fenced pointer block
// install injects into the 4 of 8 agent targets that declare an
// instructions file (Claude, Codex, opencode, Gemini — see the
// 06-RESEARCH.md per-agent install-coverage table). Codex and opencode
// share this same repo-root AGENTS.md at local scope, gated by
// instructionsRequestedElsewhere (D-11) so uninstall never strips it out
// from under the other. It points agents at codegraph_explore /
// `codegraph explore` and, generically, at resources/list for the
// per-tool reference — this is deliberately SHORT, not the old full
// playbook TS removed in #529/#704.
//
// It names no skill and no skill file, on purpose: the block text is
// frozen (D-01a) so no user's already-installed block is ever rewritten
// just because the skill's own reach changed. The skill package now
// reaches 7 of the 8 registered targets — every target but Hermes, which
// has no skill mechanism at either scope (D-29, docs/AGENT-CAPABILITIES.md)
// — and is announced elsewhere, by the wire-level instructions const in
// internal/mcp/server.go, not by this block.
// TestInstructionsBlockNamesOnlyShippedCapabilities holds this property.
const codegraphInstructionsBlock = codegraphSectionStart + `
## CodeGraph

In repositories indexed by CodeGraph (a ` + "`.codegraph/`" + ` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): ` + "`codegraph_explore`" + ` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them.
- **Shell** (always works): ` + "`codegraph explore \"<symbol names or question>\"`" + ` prints the same output.
- **Reference docs** (MCP only): call ` + "`resources/list`" + ` then ` + "`resources/read`" + ` for the per-tool reference beyond this summary.

If there is no ` + "`.codegraph/`" + ` directory, skip CodeGraph entirely — indexing is the user's decision.
` + codegraphSectionEnd
