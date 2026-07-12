package cli

import (
	"strings"
	"testing"
)

// TestTelemetryStatement asserts the telemetry output states zero
// passive/background telemetry AND explicitly names `codegraph upgrade`
// as the sole intentional, user-initiated network path (D-15's honesty
// requirement — both claims must be present).
func TestTelemetryStatement(t *testing.T) {
	stdout, _, err := execCmd("telemetry")
	if err != nil {
		t.Fatalf("telemetry: %v", err)
	}

	lower := strings.ToLower(stdout)
	if !strings.Contains(lower, "zero") || !strings.Contains(lower, "telemetry") {
		t.Errorf("output %q does not assert zero telemetry", stdout)
	}
	if !strings.Contains(lower, "background") && !strings.Contains(lower, "passive") {
		t.Errorf("output %q does not disclaim passive/background collection", stdout)
	}
	if !strings.Contains(lower, "codegraph upgrade") {
		t.Errorf("output %q does not name `codegraph upgrade` as the sole network path", stdout)
	}
}

// TestTelemetryStatementIsConst asserts the printed output equals the
// package-level telemetryStatement const verbatim (modulo the trailing
// newline Fprintln adds) — RunE performs no computation beyond printing
// it, so this pins the "no network I/O, no state read" behavior by
// construction.
func TestTelemetryStatementIsConst(t *testing.T) {
	stdout, _, err := execCmd("telemetry")
	if err != nil {
		t.Fatalf("telemetry: %v", err)
	}
	if strings.TrimRight(stdout, "\n") != telemetryStatement {
		t.Errorf("output = %q, want telemetryStatement verbatim %q", stdout, telemetryStatement)
	}
}

// TestHelpEveryCommand asserts every registered command exposes a
// non-empty Short description, so `codegraph help <command>` and
// `<command> --help` are useful (D-10).
func TestHelpEveryCommand(t *testing.T) {
	root := newRootCmd()
	for _, cmd := range root.Commands() {
		if strings.TrimSpace(cmd.Short) == "" {
			t.Errorf("command %q has empty Short", cmd.Name())
		}
	}
}

// TestHelpVersionCommand asserts `codegraph help version` prints
// version-specific text (a help-smoke check that Cobra's built-in help
// surfaces per-command Short/Long, D-10).
func TestHelpVersionCommand(t *testing.T) {
	stdout, _, err := execCmd("help", "version")
	if err != nil {
		t.Fatalf("help version: %v", err)
	}
	if !strings.Contains(stdout, "version") {
		t.Errorf("output %q does not mention version", stdout)
	}
}
