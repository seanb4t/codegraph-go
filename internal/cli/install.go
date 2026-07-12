package cli

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// installStdinIsInteractive reports whether install should present the
// interactive multi-select (D-03): cmd's configured stdin must be the
// process's own os.Stdin AND that fd must be a character device (a real
// TTY), detected via stdlib os.ModeCharDevice only — no new terminal
// dependency (RESEARCH.md's stdlib-only TTY-detection guidance). A
// package-level var so install_test.go can force either branch without a
// real pty, mirroring upgrade.go's upgradeRunFunc injectable-seam pattern.
var installStdinIsInteractive = func(cmd *cobra.Command) bool {
	if cmd.InOrStdin() != os.Stdin {
		return false
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// parseLocationFlag validates --location against the two values
// agents.Location supports; any other value is a clear, immediate error
// rather than a silently-wrong config write (T-06-04-01's "unknown id"
// discipline applied to --location too).
func parseLocationFlag(raw string) (agents.Location, error) {
	switch agents.Location(raw) {
	case agents.LocationGlobal, agents.LocationLocal:
		return agents.Location(raw), nil
	default:
		return "", fmt.Errorf("--location must be \"global\" or \"local\" (got %q)", raw)
	}
}

// newInstallCmd builds `codegraph install` (AGNT-01, D-02): resolves
// --target/--location, resolves the running binary's absolute path once
// via os.Executable() (D-04 — the MCP config written points at THIS
// binary, not a PATH guess), selects targets from the internal/agents
// registry (an explicit --target, an interactive TTY multi-select, or the
// non-interactive auto fallback per D-03), and prints a per-agent status
// line. Contains no agent-specific logic — every quirk lives in the
// target's own file (06-02/06-03); this command only iterates the
// registry and delegates.
func newInstallCmd() *cobra.Command {
	var target string
	var location string
	var autoAllow bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Configure coding agents to use this codegraph binary as their MCP server",
		Long: "Detect and configure the agent roster (Claude Code, Cursor, Codex CLI,\n" +
			"opencode, Gemini CLI, Antigravity, Hermes, Kiro): write each agent's MCP\n" +
			"server entry plus, for the agents that support it, a short marker-fenced\n" +
			"instruction block. Idempotent — re-running install is a no-op when\n" +
			"nothing changed.",
		Example: "  codegraph install\n" +
			"  codegraph install --target all --location global\n" +
			"  codegraph install --target claude,cursor\n" +
			"  codegraph install --target none",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := parseLocationFlag(location)
			if err != nil {
				return fmt.Errorf("codegraph install: %w", err)
			}

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("codegraph install: resolve running binary path: %w", err)
			}

			var targets []agents.AgentTarget
			switch {
			case cmd.Flags().Changed("target"):
				targets, err = agents.ResolveTargetFlag(target, loc)
			case installStdinIsInteractive(cmd):
				targets, err = promptAgentMultiSelect(cmd, loc)
			default:
				// D-03: no TTY (or CI) never blocks on a prompt — resolve
				// straight to auto, same as an explicit --target auto.
				targets, err = agents.ResolveTargetFlag("auto", loc)
			}
			if err != nil {
				return fmt.Errorf("codegraph install: %w", err)
			}

			opts := agents.InstallOptions{AutoAllow: autoAllow, ExecPath: execPath}
			printAgentResults(cmd, targets, loc, func(t agents.AgentTarget) agents.WriteResult {
				return t.Install(loc, opts)
			}, installStatus)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "auto", "which agents to configure: auto|all|none|<comma-separated ids>")
	cmd.Flags().StringVar(&location, "location", string(agents.LocationGlobal), "config scope: global|local")
	cmd.Flags().BoolVar(&autoAllow, "auto-allow", false, "also add mcp__codegraph__* to Claude Code's permissions.allow list")

	return cmd
}

// promptAgentMultiSelect renders a numbered list of every registered
// target (agents.AllTargets(), deterministically sorted), pre-marks the
// ones agents.DetectAll(loc) reports as installed, and reads a selection
// line via cmd.InOrStdin()/cmd.OutOrStdout() — the same testable-I/O idiom
// confirm() (uninit.go) uses. Empty input (bare Enter) accepts the
// detected defaults; EOF/unreadable input degrades to the same "auto"
// resolution the non-interactive path uses, never blocking (D-03).
func promptAgentMultiSelect(cmd *cobra.Command, loc agents.Location) ([]agents.AgentTarget, error) {
	all := agents.AllTargets()
	detection := agents.DetectAll(loc)

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Select agents to configure (comma-separated numbers, \"all\", \"none\", or Enter to accept the detected defaults):")
	var preselected []int
	for i, t := range all {
		mark := " "
		if detection[t.ID()].Installed {
			mark = "x"
			preselected = append(preselected, i)
		}
		fmt.Fprintf(out, "  [%s] %d) %s\n", mark, i+1, t.DisplayName())
	}
	fmt.Fprint(out, "> ")

	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return agents.ResolveTargetFlag("auto", loc)
	}
	line = strings.TrimSpace(line)

	switch {
	case line == "":
		return selectByIndices(all, preselected), nil
	case strings.EqualFold(line, "all"):
		return all, nil
	case strings.EqualFold(line, "none"):
		return nil, nil
	}

	var indices []int
	for _, tok := range strings.Split(line, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		n, err := strconv.Atoi(tok)
		if err != nil || n < 1 || n > len(all) {
			return nil, fmt.Errorf("invalid selection %q", tok)
		}
		indices = append(indices, n-1)
	}
	return selectByIndices(all, indices), nil
}

// selectByIndices returns all[i] for each i in indices, de-duplicated and
// in ascending index order (agents.AllTargets()'s own deterministic order,
// not input order) — mirrors --target csv's resolution order.
func selectByIndices(all []agents.AgentTarget, indices []int) []agents.AgentTarget {
	sort.Ints(indices)
	out := make([]agents.AgentTarget, 0, len(indices))
	seen := make(map[int]bool, len(indices))
	for _, i := range indices {
		if seen[i] {
			continue
		}
		seen[i] = true
		out = append(out, all[i])
	}
	return out
}

// installStatus rolls WriteResult's per-file actions up into one word for
// install's per-agent summary line: "unchanged" only when every touched
// file was already correct (D-07 idempotency), "configured" otherwise.
func installStatus(result agents.WriteResult) string {
	for _, f := range result.Files {
		if f.Action != agents.ActionUnchanged {
			return "configured"
		}
	}
	return "unchanged"
}

// printAgentResults is install's and uninstall's shared per-agent
// reporting loop: skip (report "unsupported") any target that doesn't
// support loc without calling do() at all — Install/Uninstall return an
// empty WriteResult for an unsupported location, which would otherwise
// print as a confusing no-op rather than an explicit status (D-08). For
// every supported target, call do(), print statusOf(result) as the
// headline, then one indented line per touched file and note.
func printAgentResults(cmd *cobra.Command, targets []agents.AgentTarget, loc agents.Location, do func(agents.AgentTarget) agents.WriteResult, statusOf func(agents.WriteResult) string) {
	out := cmd.OutOrStdout()
	if len(targets) == 0 {
		fmt.Fprintln(out, "no agents selected")
		return
	}
	for _, t := range targets {
		if !t.SupportsLocation(loc) {
			fmt.Fprintf(out, "%s: unsupported (%s not supported)\n", t.DisplayName(), loc)
			continue
		}
		result := do(t)
		fmt.Fprintf(out, "%s: %s\n", t.DisplayName(), statusOf(result))
		for _, f := range result.Files {
			fmt.Fprintf(out, "  %s: %s\n", f.Action, f.Path)
		}
		for _, note := range result.Notes {
			fmt.Fprintf(out, "  note: %s\n", note)
		}
	}
}
