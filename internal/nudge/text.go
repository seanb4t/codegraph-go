package nudge

// Text is the PreToolUse nudge sentence (D-14): a factual one-liner, not an
// imperative, naming only the codegraph_explore MCP tool and its
// `codegraph explore` CLI fallback. Its bytes are pinned — by hand-typed
// oracles in the adapter tests here, and by the text drift guards 06-02
// adds — so any edit is a deliberate, test-visible change.
const Text = "This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions."
