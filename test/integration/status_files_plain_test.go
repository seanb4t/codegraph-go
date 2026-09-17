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

// TestStatusColorFlagRealBinary covers the 04-03 tracer's live, real-binary
// proof (D-09/D-11/CLI-02/CLI-03): --color drives ESC-byte presence
// independently of the pipe's own non-TTY status. --color=always must
// force ANSI even over a plain os/exec pipe (colorprofile floors a forced
// profile at ANSI); --color=never must produce zero ESC bytes and be
// byte-identical to the bare (flagless) invocation; --color=bogus must be
// a usage error naming all three accepted values.
func TestStatusColorFlagRealBinary(t *testing.T) {
	dir := copyFixture(t)
	if _, stderr, err := runBinary(t, dir, nil, "init", dir); err != nil {
		t.Fatalf("init fixture via subprocess binary: %v: %s", err, stderr)
	}

	t.Run("--color=always emits ANSI on a pipe", func(t *testing.T) {
		out, stderr, err := runBinary(t, dir, nil, "status", "--path", dir, "--color=always")
		if err != nil {
			t.Fatalf("status --color=always: %v: %s", err, stderr)
		}
		if !strings.Contains(out, "\x1b[") {
			t.Errorf("status --color=always: expected an ANSI escape sequence on a pipe, got none:\n%s", out)
		}
	})

	t.Run("--color=never emits no ANSI and equals the bare pipe output", func(t *testing.T) {
		bareOut, bareErr, err := runBinary(t, dir, nil, "status", "--path", dir)
		if err != nil {
			t.Fatalf("status (bare): %v: %s", err, bareErr)
		}
		neverOut, neverErr, err := runBinary(t, dir, nil, "status", "--path", dir, "--color=never")
		if err != nil {
			t.Fatalf("status --color=never: %v: %s", err, neverErr)
		}
		if strings.Contains(neverOut, "\x1b[") {
			t.Errorf("status --color=never: unexpected ANSI escape sequence:\n%s", neverOut)
		}
		if neverOut != bareOut {
			t.Errorf("status --color=never output differs from the bare (flagless) pipe output\n--- bare ---\n%s\n--- never ---\n%s", bareOut, neverOut)
		}
	})

	t.Run("--color=bogus is a usage error naming all three values", func(t *testing.T) {
		_, stderr, err := runBinary(t, dir, nil, "status", "--path", dir, "--color=bogus")
		if err == nil {
			t.Fatalf("status --color=bogus: expected a non-zero exit, got none (stderr: %s)", stderr)
		}
		for _, want := range []string{"auto", "always", "never"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("status --color=bogus: stderr missing %q:\n%s", want, stderr)
			}
		}
	})
}
