package wireoracle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// binPath is the absolute path to the codegraph binary every test in this
// package spawns — either freshly built from source by TestMain, or an
// externally supplied binary named by the testBinEnvVar environment
// variable (see resolveTestBinPath's doc comment for the resolution
// contract). Never silently the wrong one: TestMain aborts by name rather
// than falling back to a build when the override is set but invalid.
var binPath string

// CRITERION-4 SCOPE VERDICT (plan 02-03, Task 2): this package IS IN
// SCOPE for ROADMAP Phase 2 criterion 4 — a post-release CI job may run
// this suite against the re-downloaded, notarized published binary via
// the testBinEnvVar override, alongside test/integration.
//
// This was answered by running it, not by reasoning about it. A binary
// built with the same -X github.com/seanb4t/codegraph-go/internal/
// version.Version=<tag> ldflags .goreleaser.yaml's version_ldflags anchor
// injects at release build time was pointed at via the override, and
// TestFrozenTranscriptsMatch passed all 27 scenarios clean, including
// toolslist-repeat (the one serverInfo.version-bearing tools/list
// response). The reason this works despite each frozen transcript
// embedding a specific version string: normalize.go's serverVersion rule
// already replaces every serverInfo.version value with the <VERSION>
// placeholder before comparison (D-04's named-field-anchor normalization,
// ExpectFires: true) — a release tag's version string and a locally
// built "dev" version string are both normalized away identically, so
// the differing field never reaches the byte comparison. No transcript
// re-freeze was needed or performed.

// testBinEnvVar is the environment variable name that lets this harness
// run against an externally supplied binary instead of building one from
// source: CODEGRAPH_TEST_BIN. Defined once so the literal appears exactly
// twice in this package — this doc comment and the assignment below.
const testBinEnvVar = "CODEGRAPH_TEST_BIN"

// resolveTestBinPath is a deliberate SECOND COPY of
// test/integration/main_test.go's function of the same name — Go test
// helpers are not importable across packages, matching the precedent this
// repo already set for its four runGit* helpers (test/integration's own
// runGitI doc comment names the other three). A shared non-test
// internal/testbin package was considered and rejected: two copies of a
// ~20-line pure function with identical table tests in both packages is
// cheaper than a new exported internal package, and if a third harness
// ever needs this seam, that is the signal to extract it.
//
// It resolves the raw value of the testBinEnvVar environment variable
// into a usable binary path. It is a pure function — no os.Getenv, no
// os.Exit, no writes — specifically so it is a table test's ideal
// subject; TestMain is the only caller that touches the environment or
// the process exit code.
//
// Contract: for a non-empty raw value there are exactly two outcomes —
// (path, true, nil): the override is valid, use it; or ("", false, err):
// the override is invalid, abort by name. There is no third outcome: no
// input returns useEnv=true together with a non-nil error, and no
// non-empty input returns useEnv=false with a nil error. That absence is
// the property that forbids a silent fallback to a local `go build` on a
// bad override — a fallback here would let a job claiming to test the
// notarized release binary quietly test a locally rebuilt one instead.
//
// The checks below are STAT-LEVEL only: they confirm raw exists, is a
// regular file, and carries at least one UNIX execute-permission bit
// (owner, group, or other) — never that it is a valid,
// architecture-compatible executable. This is deliberate, not an
// oversight: this repository's harnesses target macOS and Linux only, so
// UNIX mode semantics are the correct and only check to make here. The
// real "does this binary actually run on this machine" question is
// answered by the test suite itself failing, not by a probe-execute here.
func resolveTestBinPath(raw string) (path string, useEnv bool, err error) {
	if raw == "" {
		return "", false, nil
	}

	info, statErr := os.Stat(raw)
	if statErr != nil {
		return "", false, fmt.Errorf("%s=%q: %w", testBinEnvVar, raw, statErr)
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("%s=%q: not a regular file (is a directory)", testBinEnvVar, raw)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("%s=%q: not a regular file", testBinEnvVar, raw)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", false, fmt.Errorf("%s=%q: not executable (no execute permission bit set)", testBinEnvVar, raw)
	}

	abs, absErr := filepath.Abs(raw)
	if absErr != nil {
		return "", false, fmt.Errorf("%s=%q: resolve absolute path: %w", testBinEnvVar, raw, absErr)
	}
	return abs, true, nil
}

// TestMain resolves the testBinEnvVar override first (see
// resolveTestBinPath), mirroring test/integration/main_test.go's TestMain
// exactly (VRFY-04: the oracle must run against the real, unmodified
// binary — whether that binary was just built or externally supplied).
// When unset, it builds the real binary hermetically, once, into a
// package-level temp dir before any test in this package runs; a build
// failure is a hard stop. When set and valid, it runs every test against
// that externally supplied binary instead — no temp dir is created, no
// build runs, and nothing this harness did not create is cleaned up. When
// set and invalid, TestMain aborts before creating a temp dir, before
// building, and before running a single test: it prints a message naming
// the environment variable and the offending path to stderr and exits
// non-zero.
func TestMain(m *testing.M) {
	resolved, useEnv, err := resolveTestBinPath(os.Getenv(testBinEnvVar))
	if err != nil {
		fmt.Fprintln(os.Stderr, "wireoracle: TestMain:", err)
		os.Exit(1)
	}
	if useEnv {
		binPath = resolved
		os.Exit(m.Run())
	}

	tmpDir, err := os.MkdirTemp("", "codegraph-wireoracle-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "wireoracle: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}

	binPath = filepath.Join(tmpDir, "codegraph")
	buildCmd := exec.Command("go", "build", "-o", binPath, "github.com/seanb4t/codegraph-go/cmd/codegraph")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "wireoracle: TestMain: go build github.com/seanb4t/codegraph-go/cmd/codegraph failed: %v\n%s\n", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}
