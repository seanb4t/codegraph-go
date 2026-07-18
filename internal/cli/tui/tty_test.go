package tui

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestInteractiveAllowed drives every branch of InteractiveAllowed via the
// three injectable package-level seams (stdinIsInteractive/stdoutIsTTY/
// noColor) — no real pty required (D-10, RESEARCH.md Pattern 1).
func TestInteractiveAllowed(t *testing.T) {
	tests := []struct {
		name      string
		stdinTTY  bool
		stdoutTTY bool
		noColor   string
		want      bool
	}{
		{name: "piped stdin, TTY stdout, NO_COLOR empty -> false", stdinTTY: false, stdoutTTY: true, noColor: "", want: false},
		{name: "TTY stdin, piped stdout -> false", stdinTTY: true, stdoutTTY: false, noColor: "", want: false},
		{name: "TTY stdin, TTY stdout, NO_COLOR=1 -> false", stdinTTY: true, stdoutTTY: true, noColor: "1", want: false},
		{name: "TTY stdin, TTY stdout, NO_COLOR empty -> true", stdinTTY: true, stdoutTTY: true, noColor: "", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origStdin, origStdout, origNoColor := stdinIsInteractive, stdoutIsTTY, noColor
			t.Cleanup(func() {
				stdinIsInteractive = origStdin
				stdoutIsTTY = origStdout
				noColor = origNoColor
			})
			stdinIsInteractive = func(*cobra.Command) bool { return tc.stdinTTY }
			stdoutIsTTY = func() bool { return tc.stdoutTTY }
			noColor = func() string { return tc.noColor }

			cmd := &cobra.Command{}
			if got := InteractiveAllowed(cmd); got != tc.want {
				t.Errorf("InteractiveAllowed() = %v, want %v", got, tc.want)
			}
		})
	}
}
