package tui

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
)

// stdinIsInteractive reports whether cmd's configured stdin is the
// process's own os.Stdin AND that fd is a real character-device TTY —
// copied from install.go's installStdinIsInteractive (D-10, RESEARCH.md
// Pattern 1). A package-level var so tty_test.go can force either branch
// without a real pty.
var stdinIsInteractive = func(cmd *cobra.Command) bool {
	if cmd.InOrStdin() != os.Stdin {
		return false
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// stdoutIsTTY reports whether the process's own stdout is a real terminal
// — the same term.IsTerminal probe present's own RunE call sites use. A
// package-level var so tty_test.go can force either branch.
var stdoutIsTTY = func() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// noColor returns the NO_COLOR environment variable's raw value. A
// package-level var so tty_test.go can force either branch.
var noColor = func() string {
	return os.Getenv("NO_COLOR")
}

// InteractiveAllowed reports whether an interactive component may call
// tea.NewProgram() for cmd: BOTH stdin and stdout must be real terminals,
// composing present.ChoosePresentation (the SAME stdout+NO_COLOR seam
// every other renderer gates on, D-10) with the additional stdin-TTY
// requirement bubbletea's raw-mode input driver needs. Not a TTY on either
// fd (piped/CI) or NO_COLOR set ⇒ false — the caller falls back to a
// non-interactive path and never issues a blocking stdin read (TUI-04's
// never-hang invariant).
func InteractiveAllowed(cmd *cobra.Command) bool {
	return stdinIsInteractive(cmd) && present.ChoosePresentation(stdoutIsTTY(), noColor())
}
