package present

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// buildHelpTestTree constructs a small cobra tree that mirrors root.go's
// D-13 shape (four groups, in the same titles/order, help+completion
// filed under Maintenance) without importing internal/cli — present must
// never import cli (that would be circular; cli already imports present).
// It also carries one visible ungrouped command ("extra", exercising the
// "Additional Commands:" fallback) and one hidden command ("hidden",
// which must never appear in rendered output) so
// TestRenderHelpGroupsInOrder can assert both branches RenderHelp must
// handle.
func buildHelpTestTree() (root, status *cobra.Command) {
	root = &cobra.Command{
		Use:   "codegraph",
		Short: "Pre-indexed code knowledge graph for coding agents",
		Long:  "codegraph builds and maintains a local knowledge graph of a repository's symbols, edges, and files.",
	}
	root.PersistentFlags().String("color", "auto", "colour output: auto (detect), always, or never")

	noop := func(*cobra.Command, []string) {}
	status = &cobra.Command{Use: "status", Short: "Report index health and counts", Run: noop}
	status.Flags().StringP("path", "p", "", "repo path (default: cwd)")
	explore := &cobra.Command{Use: "explore <query...>", Short: "Explore relevant symbols", Run: noop}
	initCmd := &cobra.Command{Use: "init [path]", Short: "Create .codegraph/ and build the graph", Run: noop}
	serve := &cobra.Command{Use: "serve", Short: "Run the codegraph MCP server", Run: noop}
	version := &cobra.Command{Use: "version", Short: "Print build version information", Run: noop}
	extra := &cobra.Command{Use: "extra", Short: "An ungrouped visible command", Run: noop}
	hidden := &cobra.Command{Use: "hidden", Short: "A hidden command", Hidden: true, Run: noop}

	root.AddCommand(status, explore, initCmd, serve, version, extra, hidden)
	root.AddGroup(
		&cobra.Group{ID: "query", Title: "Query the graph:"},
		&cobra.Group{ID: "build", Title: "Build the index:"},
		&cobra.Group{ID: "agents", Title: "Agents & serving:"},
		&cobra.Group{ID: "maintenance", Title: "Maintenance:"},
	)
	status.GroupID = "query"
	explore.GroupID = "query"
	initCmd.GroupID = "build"
	serve.GroupID = "agents"
	version.GroupID = "maintenance"
	// extra and hidden are deliberately left ungrouped.

	root.SetHelpCommandGroupID("maintenance")
	root.SetCompletionCommandGroupID("maintenance")
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	return root, status
}

// TestRenderHelpGroupsInOrder is CLI-06/D-14's guard for the hand-rolled
// help renderer: it asserts the four D-13 groups render in registration
// order with their members beneath, ungrouped visible commands fall back
// to "Additional Commands:" (cobra's own wording, cited not tested per
// D-00), a hidden command never appears, help/completion are filed under
// Maintenance, and a subcommand's own help carries Usage:/its own flags
// with no group titles at all.
func TestRenderHelpGroupsInOrder(t *testing.T) {
	root, status := buildHelpTestTree()
	pal := NewPalette(true)

	var b strings.Builder
	if err := RenderHelp(root, pal, &b); err != nil {
		t.Fatalf("RenderHelp(root): %v", err)
	}
	styled := b.String()

	// Positive ESC-presence control (rule 84d1gfpywd): the styled output
	// must actually carry ANSI, or the "stripped" assertions below would
	// pass vacuously against plain, unstyled text.
	if !strings.Contains(styled, "\x1b[") {
		t.Errorf("styled RenderHelp(root) output does not contain an ESC byte")
	}

	stripped := stripANSI(styled)

	wantOrder := []string{"Query the graph:", "Build the index:", "Agents & serving:", "Maintenance:"}
	titleIdx := make(map[string]int, len(wantOrder))
	lastIdx := -1
	for _, title := range wantOrder {
		idx := strings.Index(stripped, title)
		if idx < 0 {
			t.Fatalf("stripped RenderHelp(root) output missing group title %q:\n%s", title, stripped)
		}
		if idx <= lastIdx {
			t.Fatalf("group title %q does not appear after the previous title (D-13 order violated):\n%s", title, stripped)
		}
		titleIdx[title] = idx
		lastIdx = idx
	}

	groupMembers := map[string][]string{
		"Query the graph:":  {"status", "explore"},
		"Build the index:":  {"init"},
		"Agents & serving:": {"serve"},
		"Maintenance:":      {"version", "help", "completion"},
	}
	for i, title := range wantOrder {
		start := titleIdx[title]
		end := len(stripped)
		if i+1 < len(wantOrder) {
			end = titleIdx[wantOrder[i+1]]
		}
		section := stripped[start:end]
		for _, member := range groupMembers[title] {
			if !strings.Contains(section, member) {
				t.Errorf("group %q's own section does not contain member %q:\n%s", title, member, section)
			}
		}
	}

	if !strings.Contains(stripped, "Additional Commands:") {
		t.Errorf("stripped output missing \"Additional Commands:\" for the ungrouped visible command")
	}
	if !strings.Contains(stripped, "extra") {
		t.Errorf("stripped output missing ungrouped visible command %q", "extra")
	}
	if strings.Contains(stripped, "hidden") {
		t.Errorf("stripped output must never list the hidden command %q", "hidden")
	}

	if !strings.Contains(stripped, "Flags:") {
		t.Errorf("stripped output missing \"Flags:\" section")
	}
	if !strings.Contains(stripped, "--color") {
		t.Errorf("stripped output missing root's own --color flag")
	}

	// Subcommand help: Usage:/its own flags, no group titles at all.
	var sb strings.Builder
	if err := RenderHelp(status, pal, &sb); err != nil {
		t.Fatalf("RenderHelp(status): %v", err)
	}
	subStripped := stripANSI(sb.String())
	if !strings.Contains(subStripped, "Usage:") {
		t.Errorf("subcommand help missing \"Usage:\"")
	}
	if !strings.Contains(subStripped, "status") {
		t.Errorf("subcommand help missing its own usage line")
	}
	if !strings.Contains(subStripped, "--path") {
		t.Errorf("subcommand help missing its own --path flag")
	}
	for _, title := range wantOrder {
		if strings.Contains(subStripped, title) {
			t.Errorf("subcommand help unexpectedly contains group title %q", title)
		}
	}
}
