package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupIndexedFixtureWithTest is setupIndexedFixture (query_cli_test.go)
// plus one addition: gofixture ships no _test.go files (query_cli_test.go's
// TestAffectedCmd comment), so `affected` never has a real affected test to
// report. This variant writes a pkga_test.go that calls the same-package
// Alpha() BEFORE running init, so the BFS in query.Engine.Affected has a
// real isTestSymbol leaf (pkga/pkga_test.go's TestAlpha) to surface —
// letting the --quiet/--json/--filter assertions below exercise actual
// output instead of only the always-empty shape.
func setupIndexedFixtureWithTest(t *testing.T) string {
	t.Helper()

	dir := copyFixture(t)
	testFile := filepath.Join(dir, "pkga", "pkga_test.go")
	src := "package pkga\n\nimport \"testing\"\n\nfunc TestAlpha(t *testing.T) {\n\tif Alpha() != 1 {\n\t\tt.Fatal(\"unexpected Alpha() result\")\n\t}\n}\n"
	if err := os.WriteFile(testFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write pkga_test.go: %v", err)
	}
	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init fixture: unexpected error: %v", err)
	}
	return dir
}

// execCmdTimeout runs execCmdWithInput on a goroutine and fails the test
// (rather than hanging the whole suite until the go test binary's own
// timeout) if the command doesn't return within d — the explicit
// never-hang assertion for T-08-05-03 (bufio.Scanner over cmd.InOrStdin()
// must return on EOF for a piped/empty/closed stream, not block).
func execCmdTimeout(t *testing.T, d time.Duration, input string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	type result struct {
		stdout, stderr string
		err            error
	}
	done := make(chan result, 1)
	go func() {
		out, errOut, e := execCmdWithInput(input, args...)
		done <- result{out, errOut, e}
	}()

	select {
	case r := <-done:
		return r.stdout, r.stderr, r.err
	case <-time.After(d):
		t.Fatalf("affected %v: command did not return within %s — stdin read hung", args, d)
		return "", "", nil
	}
}

func TestAffectedStdinNeverHangs(t *testing.T) {
	dir := setupIndexedFixtureWithTest(t)

	out, _, err := execCmdTimeout(t, 5*time.Second, "pkga/pkga.go\n", "affected", "--stdin", "--quiet", "-p", dir)
	if err != nil {
		t.Fatalf("affected --stdin --quiet: unexpected error: %v", err)
	}
	if !strings.Contains(out, filepath.Join("pkga", "pkga_test.go")) {
		t.Fatalf("affected --stdin --quiet: expected %q in output, got %q", "pkga/pkga_test.go", out)
	}
}

func TestAffectedEmptyStdinNoArgs(t *testing.T) {
	// This branch resolves before query.OpenAt is ever called (08-05
	// action: "Handle zero input ... before opening the engine") — an
	// uninitialized directory proves no index is required for it.
	dir := t.TempDir()

	t.Run("non-quiet emits an advisory and exits 0", func(t *testing.T) {
		out, _, err := execCmdTimeout(t, 5*time.Second, "", "affected", "--stdin", "-p", dir)
		if err != nil {
			t.Fatalf("affected --stdin (empty): unexpected error: %v", err)
		}
		if !strings.Contains(out, "no files provided") {
			t.Fatalf("affected --stdin (empty): expected a %q advisory, got %q", "no files provided", out)
		}
	})

	t.Run("quiet emits nothing and exits 0", func(t *testing.T) {
		out, _, err := execCmdTimeout(t, 5*time.Second, "", "affected", "--stdin", "--quiet", "-p", dir)
		if err != nil {
			t.Fatalf("affected --stdin --quiet (empty): unexpected error: %v", err)
		}
		if strings.TrimSpace(out) != "" {
			t.Fatalf("affected --stdin --quiet (empty): expected empty stdout, got %q", out)
		}
	})
}

func TestAffectedQuietPathsOnly(t *testing.T) {
	dir := setupIndexedFixtureWithTest(t)

	out, _, err := execCmd("affected", "pkga/pkga.go", "-p", dir, "--quiet")
	if err != nil {
		t.Fatalf("affected --quiet: unexpected error: %v", err)
	}

	if strings.Contains(out, "affected test(s)") {
		t.Fatalf("affected --quiet: expected no summary line, got %q", out)
	}
	if strings.Contains(out, "⚠") {
		t.Fatalf("affected --quiet: expected no worktree notice glyph, got %q", out)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 || lines[0] != filepath.ToSlash(filepath.Join("pkga", "pkga_test.go")) {
		t.Fatalf("affected --quiet: expected exactly [%q], got %+v", "pkga/pkga_test.go", lines)
	}
}

func TestAffectedJSONQuiet(t *testing.T) {
	dir := setupIndexedFixtureWithTest(t)

	out, _, err := execCmd("affected", "pkga/pkga.go", "-p", dir, "--json", "--quiet")
	if err != nil {
		t.Fatalf("affected --json --quiet: unexpected error: %v", err)
	}

	var result struct {
		Files         []string `json:"files"`
		AffectedTests []struct {
			Name     string `json:"name"`
			FilePath string `json:"filePath"`
		} `json:"affectedTests"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("affected --json --quiet: invalid JSON: %v\noutput: %s", err, out)
	}
	found := false
	for _, l := range result.AffectedTests {
		if l.Name == "TestAlpha" {
			found = true
		}
	}
	if !found {
		t.Fatalf("affected --json --quiet: expected %q among affectedTests, got %+v", "TestAlpha", result.AffectedTests)
	}
}

func TestAffectedStdinCRLFAndBlankLines(t *testing.T) {
	dir := setupIndexedFixtureWithTest(t)

	out, _, err := execCmdTimeout(t, 5*time.Second, "pkga/pkga.go\r\n\r\n  \r\n", "affected", "--stdin", "--quiet", "-p", dir)
	if err != nil {
		t.Fatalf("affected --stdin (CRLF+blank): unexpected error: %v", err)
	}
	if !strings.Contains(out, "pkga_test.go") {
		t.Fatalf("affected --stdin (CRLF+blank): expected pkga_test.go in output, got %q", out)
	}
}

// TestAffectedStdinLineTooLong pins CR-01 (iteration-2 re-review): the
// scanner.Buffer initial-capacity argument must not exceed
// affectedStdinMaxLineBytes, or bufio.Scanner's documented
// max(maxArg, cap(buf)) ceiling silently raises the real limit past the
// intended one. A line one byte over the cap must produce the documented
// "--stdin line exceeds maximum %d bytes" error rather than being silently
// accepted.
func TestAffectedStdinLineTooLong(t *testing.T) {
	dir := setupIndexedFixtureWithTest(t)

	t.Run("line exceeding the cap by one byte errors", func(t *testing.T) {
		tooLong := strings.Repeat("a", affectedStdinMaxLineBytes+1) + "\n"
		_, _, err := execCmdTimeout(t, 5*time.Second, tooLong, "affected", "--stdin", "--quiet", "-p", dir)
		if err == nil {
			t.Fatalf("affected --stdin: expected error for %d-byte line (max %d), got nil", affectedStdinMaxLineBytes+1, affectedStdinMaxLineBytes)
		}
		if !strings.Contains(err.Error(), "exceeds maximum") {
			t.Fatalf("affected --stdin: expected 'exceeds maximum' error, got %v", err)
		}
	})

	t.Run("line just under the cap succeeds", func(t *testing.T) {
		// bufio.Scanner's documented contract is "token size must be less
		// than max" (strictly), so the largest line guaranteed to succeed
		// is affectedStdinMaxLineBytes-1, not affectedStdinMaxLineBytes
		// itself. Pad a real path out to exactly that length with
		// trailing filler that still resolves to no matched file
		// (harmless — we're only asserting no ErrTooLong here).
		underCap := "pkga/pkga.go" + strings.Repeat(" ", affectedStdinMaxLineBytes-1-len("pkga/pkga.go")-1) + "x\n"
		if len(underCap)-1 != affectedStdinMaxLineBytes-1 {
			t.Fatalf("test setup bug: line is %d bytes, want %d", len(underCap)-1, affectedStdinMaxLineBytes-1)
		}
		_, _, err := execCmdTimeout(t, 5*time.Second, underCap, "affected", "--stdin", "--quiet", "-p", dir)
		if err != nil {
			t.Fatalf("affected --stdin: unexpected error for %d-byte line (cap %d): %v", affectedStdinMaxLineBytes-1, affectedStdinMaxLineBytes, err)
		}
	})
}

func TestAffectedFilter(t *testing.T) {
	dir := setupIndexedFixtureWithTest(t)

	t.Run("matching glob narrows to test paths", func(t *testing.T) {
		// filepath.Match's "*" does not cross a Separator, so the glob
		// must account for the "pkga/" directory component — matching
		// TS-parity glob semantics (filepath.Match, not a ** dependency).
		out, _, err := execCmd("affected", "pkga/pkga.go", "-p", dir, "--quiet", "--filter", "*/*_test.go")
		if err != nil {
			t.Fatalf("affected --filter */*_test.go: unexpected error: %v", err)
		}
		if !strings.Contains(out, "pkga_test.go") {
			t.Fatalf("affected --filter */*_test.go: expected pkga_test.go, got %q", out)
		}
	})

	t.Run("non-matching glob yields an empty set", func(t *testing.T) {
		out, _, err := execCmd("affected", "pkga/pkga.go", "-p", dir, "--quiet", "--filter", "*.md")
		if err != nil {
			t.Fatalf("affected --filter *.md: unexpected error: %v", err)
		}
		if strings.TrimSpace(out) != "" {
			t.Fatalf("affected --filter *.md: expected empty output, got %q", out)
		}
	})
}
