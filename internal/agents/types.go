// Package agents implements the codegraph install/uninstall subsystem: the
// AgentTarget interface, one implementation per roster agent (Claude Code,
// Cursor, Codex CLI, opencode, Hermes, Gemini CLI, Antigravity, Kiro), and
// the shared surgical-write helpers every target uses to edit external
// agent config files in place. Install/uninstall logic lives here — never
// in internal/cli, which only resolves flags/paths and delegates (D-02).
//
// This package writes external tool config files; it never reads or
// mutates the project's own graph store, indexer, or query engine
// packages (phase 6 boundary discipline) — install/uninstall never touch
// the graph.
package agents

// Location is the install/uninstall scope: a per-user (global) config or a
// per-project (local) config. Not every agent supports both — Codex,
// Antigravity, and Hermes are global-only (D-03, D-05).
type Location string

const (
	// LocationGlobal is the per-user config scope (e.g. ~/.claude.json).
	LocationGlobal Location = "global"
	// LocationLocal is the per-project config scope (e.g. ./.mcp.json).
	LocationLocal Location = "local"
)

// TargetID is the stable registry key for one roster agent (D-02). Used by
// --target csv parsing, the registry map, and the interactive multi-select
// — never derived from DisplayName, which may change wording over time.
type TargetID string

const (
	Claude      TargetID = "claude"
	Cursor      TargetID = "cursor"
	Codex       TargetID = "codex"
	Opencode    TargetID = "opencode"
	Hermes      TargetID = "hermes"
	Gemini      TargetID = "gemini"
	Antigravity TargetID = "antigravity"
	Kiro        TargetID = "kiro"
)

// DetectionResult reports whether an agent is installed on this machine
// and whether codegraph is already configured for it, used by --target
// auto (D-03) to decide which targets to configure without prompting.
type DetectionResult struct {
	// Installed is true when the agent's own config dir/primary file
	// exists, independent of whether codegraph has been configured for it.
	Installed bool
	// AlreadyConfigured is true when the codegraph MCP entry (and/or
	// marker block) is already present for this target/location.
	AlreadyConfigured bool
	// ConfigPath is the primary config file path this detection checked,
	// surfaced for install/uninstall status reporting.
	ConfigPath string
}

// FileAction records what happened to one file during Install/Uninstall,
// mirroring the TS parity oracle's per-file action enum (D-07, D-08).
type FileAction string

const (
	// ActionCreated: the file did not exist before this write.
	ActionCreated FileAction = "created"
	// ActionUpdated: the file existed and its content changed.
	ActionUpdated FileAction = "updated"
	// ActionUnchanged: the file existed and already matched the desired
	// content — no write performed (D-07 idempotency: re-running install
	// twice is a byte-level no-op).
	ActionUnchanged FileAction = "unchanged"
	// ActionRemoved: uninstall deleted the codegraph entry/section, and
	// the file was rewritten (or removed entirely) to reflect that.
	ActionRemoved FileAction = "removed"
	// ActionNotFound: uninstall found nothing to remove — the file didn't
	// exist or had no codegraph entry (D-08: never errors in this case).
	ActionNotFound FileAction = "not-found"
	// ActionKept: a file was left untouched on purpose (e.g. a marker
	// block target that had no markers present).
	ActionKept FileAction = "kept"
)

// FileResult is one file's outcome from an Install/Uninstall call.
type FileResult struct {
	// Path is the file that was read/written (or would have been).
	Path string
	// Action is what happened to Path (D-07/D-08).
	Action FileAction
}

// WriteResult is the full outcome of one AgentTarget.Install/Uninstall
// call — every file touched, plus any human-readable notes (e.g. D-08's
// "MCP support is disabled by default in Kiro IDE" install-time note).
type WriteResult struct {
	// Files lists every file this call touched, in the order they were
	// written.
	Files []FileResult
	// Notes carries agent-specific advisory text surfaced to the user
	// (e.g. Kiro's "enable MCP in Settings" reminder); empty for most
	// targets.
	Notes []string
}

// InstallOptions carries per-run install configuration threaded down from
// the CLI to every AgentTarget.Install call.
type InstallOptions struct {
	// AutoAllow, when true, additionally writes
	// permissions.allow += ["mcp__codegraph__*"] to Claude Code's
	// settings.json (Claude-only; a no-op for every other target, D-05).
	AutoAllow bool
	// ExecPath is the absolute path to the running codegraph binary,
	// resolved once by the CLI via os.Executable() and passed down so
	// every agent's MCP command entry launches the exact binary the user
	// ran `install` from, not a PATH guess (D-04).
	ExecPath string
}

// AgentTarget is the interface every roster agent implements; the
// registry (registry.go) iterates it so install/uninstall never branch on
// agent identity outside a target's own file (D-02).
type AgentTarget interface {
	// ID returns this target's stable registry key.
	ID() TargetID

	// DisplayName returns the agent's human-readable name for CLI output
	// and the interactive multi-select (D-03).
	DisplayName() string

	// SupportsLocation reports whether this target accepts loc. Codex,
	// Antigravity, and Hermes are global-only and return false for
	// LocationLocal (D-03, D-05).
	SupportsLocation(loc Location) bool

	// Detect reports whether the agent is installed and whether
	// codegraph is already configured for it at loc, driving --target
	// auto's detected-only default (D-03).
	Detect(loc Location) DetectionResult

	// Install writes (or updates, idempotently) this agent's MCP config
	// entry and, for the 4 agents that get one, its marker-fenced
	// instructions file (D-01a, D-07).
	Install(loc Location, opts InstallOptions) WriteResult

	// Uninstall reverses everything Install wrote for this target/loc,
	// preserving every unrelated key/section in every file it touches
	// (D-02, D-07, D-08). Never errors on a target that was never
	// installed.
	Uninstall(loc Location) WriteResult

	// DescribePaths returns every config/instructions file path this
	// target reads or writes at loc, for --print-config-style reporting
	// and test assertions.
	DescribePaths(loc Location) []string
}
