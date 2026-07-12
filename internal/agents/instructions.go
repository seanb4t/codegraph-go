package agents

// codegraphSectionStart and codegraphSectionEnd are the exact marker
// fences every install/uninstall must reproduce byte-for-byte (D-01a).
// This is a hard cross-implementation contract, not a fresh choice: a Go
// uninstall must recognize a marker block a TS install wrote, and
// vice-versa. Do not alter this text.
const (
	codegraphSectionStart = "<!-- CODEGRAPH_START -->"
	codegraphSectionEnd   = "<!-- CODEGRAPH_END -->"
)

// codegraphInstructionsBlock is the short marker-fenced pointer block
// install injects into the 4 of 8 agent targets that get an instructions
// file (Claude, Codex, opencode, Gemini — see the 06-RESEARCH.md
// Corrected Per-Agent Parity Table). It points agents at codegraph_explore
// / `codegraph explore` and explicitly defers full tool guidance to the
// MCP initialize response (Phase 3) — this is deliberately SHORT, not
// the old full playbook TS removed in #529/#704.
const codegraphInstructionsBlock = codegraphSectionStart + `
## CodeGraph

In repositories indexed by CodeGraph (a ` + "`.codegraph/`" + ` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): ` + "`codegraph_explore`" + ` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them.
- **Shell** (always works): ` + "`codegraph explore \"<symbol names or question>\"`" + ` prints the same output.

If there is no ` + "`.codegraph/`" + ` directory, skip CodeGraph entirely — indexing is the user's decision.
` + codegraphSectionEnd
