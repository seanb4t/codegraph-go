package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAffectedQuietSkipsControlCharacterPaths pins SURF-04's defense against
// HIGH threat T-08-05-01: `affected --quiet` emits one path per line and is
// explicitly documented as safe to pipe into another command, so a
// graph-indexed FilePath containing an embedded \n or \r must be SKIPPED
// rather than emitted verbatim — otherwise an attacker who can land a file
// with a newline in its name (POSIX permits every byte but NUL and '/')
// injects an extra attacker-controlled "line" into that machine-readable
// stream.
//
// Layer pinned: the full CLI command (`affected --quiet` through
// newAffectedCmd's RunE), driven end-to-end over a real indexed fixture whose
// working tree genuinely contains a `_test.go` file with an embedded newline
// in its name. The guard being pinned (affected.go's
// strings.ContainsAny(l.FilePath, "\n\r") skip) lives inline in RunE and is
// not separately callable, so an end-to-end drive is the only way to reach
// it; going through the real indexer also proves such a path can actually
// arrive from the graph rather than only from a hand-crafted Location.
//
// The subtest is skipped (not failed) if the host filesystem refuses the
// filename — the guard is a property of the renderer, not of any one OS, and
// a filesystem that cannot represent the hostile input cannot exercise it.
func TestAffectedQuietSkipsControlCharacterPaths(t *testing.T) {
	dir := copyFixture(t)

	// An ordinary affected test, so the assertion below can prove the
	// clean path is still emitted while the hostile one is not — a guard
	// that suppressed everything would pass a "not emitted" check alone.
	cleanSrc := "package pkga\n\nimport \"testing\"\n\nfunc TestAlpha(t *testing.T) {\n\tif Alpha() != 1 {\n\t\tt.Fatal(\"unexpected Alpha() result\")\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "pkga", "pkga_test.go"), []byte(cleanSrc), 0o644); err != nil {
		t.Fatalf("write pkga_test.go: %v", err)
	}

	const evilBase = "ev\nil_test.go"
	evilSrc := "package pkga\n\nimport \"testing\"\n\nfunc TestEvilNewlinePath(t *testing.T) {\n\tif Alpha() != 1 {\n\t\tt.Fatal(\"unexpected Alpha() result\")\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "pkga", evilBase), []byte(evilSrc), 0o644); err != nil {
		t.Skipf("host filesystem rejects a filename with an embedded newline (%v) — cannot exercise T-08-05-01 here", err)
	}

	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init fixture: unexpected error: %v", err)
	}

	// Sanity gate: the hostile path must actually be in the graph and
	// reachable as an affected test, or the --quiet assertion below would
	// pass vacuously (nothing to suppress).
	jsonOut, _, err := execCmd("affected", "pkga/pkga.go", "-p", dir, "--json")
	if err != nil {
		t.Fatalf("affected --json: unexpected error: %v", err)
	}
	if !strings.Contains(jsonOut, "TestEvilNewlinePath") {
		t.Skipf("indexer did not surface the embedded-newline test file as affected — cannot exercise T-08-05-01 here (json: %s)", jsonOut)
	}

	out, _, err := execCmd("affected", "pkga/pkga.go", "-p", dir, "--quiet")
	if err != nil {
		t.Fatalf("affected --quiet: unexpected error: %v", err)
	}

	t.Run("no emitted line contains a control character", func(t *testing.T) {
		for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			if strings.Contains(line, "il_test.go") || strings.HasSuffix(line, "ev") {
				t.Fatalf("affected --quiet: embedded-newline path leaked into machine-readable output as %q (full output %q)", line, out)
			}
		}
	})

	t.Run("ordinary paths are still emitted", func(t *testing.T) {
		want := filepath.ToSlash(filepath.Join("pkga", "pkga_test.go"))
		if !strings.Contains(out, want) {
			t.Fatalf("affected --quiet: expected clean path %q still emitted, got %q", want, out)
		}
	})
}
