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
// install injects into the 4 of 8 agent targets that get an instructions
// file (Claude, Codex, opencode, Gemini — see the 06-RESEARCH.md
// per-agent install-coverage table). It points agents at codegraph_explore
// / `codegraph explore` and, generically, at resources/list for the
// per-tool reference — this is deliberately SHORT, not the old full
// playbook TS removed in #529/#704.
//
// It names no skill and no skill file, on purpose: only Claude Code
// receives the embedded skill package (Phase 7, AGENT-01, v1-scoped),
// while Codex, opencode and Gemini get this identical shared block and
// never receive one. The wire-level instructions const in
// internal/mcp/server.go scopes its own skill sentence to Claude Code for
// the same reason — see 08-RESEARCH.md Pitfall 1 for why the two surfaces
// deliberately diverge on that one point.
// TestInstructionsBlockNamesOnlyShippedCapabilities holds this property.
const codegraphInstructionsBlock = codegraphSectionStart + `
## CodeGraph

In repositories indexed by CodeGraph (a ` + "`.codegraph/`" + ` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): ` + "`codegraph_explore`" + ` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them.
- **Shell** (always works): ` + "`codegraph explore \"<symbol names or question>\"`" + ` prints the same output.
- **Reference docs** (MCP only): call ` + "`resources/list`" + ` then ` + "`resources/read`" + ` for the per-tool reference beyond this summary.

If there is no ` + "`.codegraph/`" + ` directory, skip CodeGraph entirely — indexing is the user's decision.
` + codegraphSectionEnd
