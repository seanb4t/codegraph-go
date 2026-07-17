package integration

import (
	"strings"
	"testing"
)

// TestStatusFilesPlainByteIdentity covers TUI-02's byte-identity
// guarantee (D-02/D-04/D-06): status/files output captured through
// runBinary's bytes.Buffer stdout — a real, non-TTY stream — must
// contain zero ANSI escape bytes, and must be identical whether or not
// NO_COLOR is set. The pretty branch (present.ChoosePresentation)
// requires isTTY true, which a subprocess piped into a bytes.Buffer can
// never satisfy (Pitfall 3) — so this asserts the plain path stays
// exactly what it was before Phase 6's wiring landed, regardless of the
// NO_COLOR gate that only matters on a genuine TTY.
func TestStatusFilesPlainByteIdentity(t *testing.T) {
	dir := copyFixture(t)
	if _, stderr, err := runBinary(t, dir, nil, "init", dir); err != nil {
		t.Fatalf("init fixture via subprocess binary: %v: %s", err, stderr)
	}

	// status's plain/pretty output embeds the caller's own start path (D-09's
	// "Project: <path>" line, sourced from resolveStartPath's os.Getwd()
	// fallback when --path is empty). os.Getwd() consults $PWD as a
	// same-inode heuristic before falling back to a full syscall
	// reconstruction (see Go's os/getwd.go) — a heuristic that can resolve
	// symlinks (macOS's /var -> /private/var) differently across separate
	// subprocess invocations even with an identical cmd.Dir, which is
	// orthogonal to this test's actual subject (ANSI presence, NO_COLOR
	// gating). Passing --path explicitly pins the reported string to the
	// same literal value in both invocations, keeping the comparison
	// focused on the byte-identity claim TUI-02 actually makes.
	cases := []struct {
		name string
		args []string
	}{
		{"status", []string{"status", "--path", dir}},
		{"files flat", []string{"files", "--path", dir}},
		{"files tree", []string{"files", "--path", dir, "--format", "tree"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plainOut, plainErr, err := runBinary(t, dir, nil, tc.args...)
			if err != nil {
				t.Fatalf("%v: %v: %s", tc.args, err, plainErr)
			}
			if strings.Contains(plainOut, "\x1b[") {
				t.Errorf("%v: non-TTY output contains an ANSI escape sequence:\n%s", tc.args, plainOut)
			}

			noColorOut, noColorErr, err := runBinary(t, dir, []string{"NO_COLOR=1"}, tc.args...)
			if err != nil {
				t.Fatalf("%v (NO_COLOR=1): %v: %s", tc.args, err, noColorErr)
			}
			if strings.Contains(noColorOut, "\x1b[") {
				t.Errorf("%v (NO_COLOR=1): non-TTY output contains an ANSI escape sequence:\n%s", tc.args, noColorOut)
			}
			if noColorOut != plainOut {
				t.Errorf("%v: NO_COLOR=1 output differs from unset-NO_COLOR output (byte-identity broken)\n--- plain ---\n%s\n--- NO_COLOR=1 ---\n%s", tc.args, plainOut, noColorOut)
			}
		})
	}
}
