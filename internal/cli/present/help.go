package present

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// RenderHelp writes a lipgloss-styled rendering of c's help text to w,
// replicating cobra's own stock help template's structure and wording
// (D-14): Long/Short, Usage:, then — for a command with subcommands —
// each of c.Groups() in registration order as a titled section listing
// its visible members, any visible ungrouped commands last under
// "Additional Commands:" (cobra's own wording, cited not tested per
// D-00), Flags:, Global Flags: when any exist, and the trailer. A
// subcommand with no groups of its own (every group belongs to the
// command that registered it via AddGroup, which is always root in this
// tree) renders no group section at all — only its own Usage:/flags.
//
// Every piece of text RenderHelp renders is this project's own static
// command/flag metadata (names, Short/Long strings, flag usage text) —
// never a user- or filesystem-derived value — so no sanitizeControl call
// is needed here (contrast present/status.go's project-path/worktree
// fields, which ARE adversarial and are sanitized before styling, CR-01).
func RenderHelp(c *cobra.Command, pal Palette, w io.Writer) error {
	var b strings.Builder

	switch {
	case c.Long != "":
		fmt.Fprintln(&b, pal.Value.Render(c.Long))
	case c.Short != "":
		fmt.Fprintln(&b, pal.Value.Render(c.Short))
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, pal.Header.Render("Usage:"))
	fmt.Fprintf(&b, "  %s\n", pal.Value.Render(c.UseLine()))
	if c.HasAvailableSubCommands() {
		fmt.Fprintf(&b, "  %s [command]\n", pal.Value.Render(c.CommandPath()))
	}

	if c.HasAvailableSubCommands() {
		renderCommandSections(&b, c, pal)
	}

	if c.HasAvailableLocalFlags() {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, pal.Header.Render("Flags:"))
		writeFlagUsages(&b, pal, c.LocalFlags().FlagUsages())
	}
	if c.HasAvailableInheritedFlags() {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, pal.Header.Render("Global Flags:"))
		writeFlagUsages(&b, pal, c.InheritedFlags().FlagUsages())
	}

	if c.HasAvailableSubCommands() {
		fmt.Fprintln(&b)
		trailer := fmt.Sprintf("Use %q for more information about a command.", c.CommandPath()+" [command] --help")
		fmt.Fprintln(&b, pal.Label.Render(trailer))
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// renderCommandSections writes one titled section per c.Groups() (in
// registration order) listing its visible members, then a trailing
// "Additional Commands:" section for any visible command that carries no
// GroupID at all — cobra's own fallback wording (D-14), never applied to
// a hidden command.
func renderCommandSections(b *strings.Builder, c *cobra.Command, pal Palette) {
	width := c.NamePadding()

	for _, g := range c.Groups() {
		var members []*cobra.Command
		for _, sub := range c.Commands() {
			if sub.GroupID == g.ID && sub.IsAvailableCommand() {
				members = append(members, sub)
			}
		}
		if len(members) == 0 {
			continue
		}
		fmt.Fprintln(b)
		fmt.Fprintln(b, pal.Header.Render(g.Title))
		writeCommandRows(b, pal, members, width)
	}

	var ungrouped []*cobra.Command
	for _, sub := range c.Commands() {
		if sub.GroupID == "" && sub.IsAvailableCommand() {
			ungrouped = append(ungrouped, sub)
		}
	}
	if len(ungrouped) == 0 {
		return
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, pal.Header.Render("Additional Commands:"))
	writeCommandRows(b, pal, ungrouped, width)
}

// writeCommandRows writes one "  <name padded> <short>\n" row per command
// in cmds, name styled via pal.Value and its one-line description via
// pal.Label — the same label/value split every other one-line renderer
// in this package uses (present.Line/KV, D-08).
func writeCommandRows(b *strings.Builder, pal Palette, cmds []*cobra.Command, width int) {
	for _, sub := range cmds {
		fmt.Fprintf(b, "  %s %s\n",
			pal.Value.Render(rpad(sub.Name(), width)),
			pal.Label.Render(sub.Short))
	}
}

// writeFlagUsages styles each non-empty line of usages (pflag's own
// FlagUsages() output, cited not re-tested per D-00) via pal.Value — the
// whole line, since splitting a flag-usage line into a name/description
// pair robustly (multi-line defaults, shorthand forms) is fragile and not
// worth the risk for what is structural chrome only.
func writeFlagUsages(b *strings.Builder, pal Palette, usages string) {
	for _, line := range strings.Split(strings.TrimRight(usages, "\n"), "\n") {
		if line == "" {
			continue
		}
		b.WriteString(pal.Value.Render(line))
		b.WriteString("\n")
	}
}

// rpad right-pads s with spaces to width columns, leaving s unchanged if
// it is already at or beyond width — mirrors cobra's own NamePadding
// convention for aligning a command-list column.
func rpad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
