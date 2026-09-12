//go:build tmux

// Package tmux is the real-PTY harness (D-07/D-08): every test here spawns
// the real codegraph binary inside a real tmux pane and asserts on
// `capture-pane` output, gated behind the `tmux` build tag — this repo's
// FIRST feature build tag (every prior constraint here is a GOOS gate or a
// bare `//go:build ignore`).
//
// Because of that tag, `go build ./...`, `go vet ./...` and a bare
// `go test ./...` all skip this package entirely unless `-tags tmux` is
// supplied. That means the named `tmux-e2e` CI job (.github/workflows/ci.yml)
// is not defense-in-depth on top of some other check — it is the ONLY way
// this package ever runs at all, which is exactly why `task test:tmux`'s
// executed-count gate exists: a green CI checkmark on its own cannot
// distinguish "the suite ran and passed" from "the tag was misspelled and
// nothing ran."
//
// The executed-count constant this gate compares against is
// TMUX_EXPECTED_TESTS, declared in the test:tmux target's own vars: block in
// Taskfile.yml. Adding or removing a top-level Test* function in this
// package means updating that constant in the SAME commit.
//
// A local `go build`/`go test` against this package requires a Go toolchain
// resolving to go.mod's pinned 1.26.6. Anything newer fails inside
// github.com/cockroachdb/swiss with `undefined: hashFn` / `undefined:
// fastrand64` — a pre-existing toolchain landmine (see
// .github/workflows/ci.yml:260-272), not a tmux, bubbletea, or test-logic
// problem. Prefix local invocations with GOTOOLCHAIN=go1.26.6, or set
// CODEGRAPH_TEST_BIN to bypass the local build entirely.
package tmux

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
// contract).
var binPath string

// testBinEnvVar is the environment variable name that lets this harness run
// against an externally supplied binary instead of building one from
// source: CODEGRAPH_TEST_BIN. Shared literally with test/integration's own
// constant of the same name and value (D-09 chose duplication of the
// resolver, not a shared package); reusing the same env var name across
// both packages is safe since it is a value, not an exported type.
const testBinEnvVar = "CODEGRAPH_TEST_BIN"

// resolveTestBinPath resolves the raw value of the testBinEnvVar
// environment variable into a usable binary path. It is a pure function —
// no os.Getenv, no os.Exit, no writes — so it is a table test's ideal
// subject; TestMain is the only caller that touches the environment or the
// process exit code.
//
// Contract (D-09's no-silent-fallback rule, restated here in this
// package's own doc comment rather than left for a reader to find in
// test/integration): for a non-empty raw value there are exactly two
// outcomes — (path, true, nil): the override is valid, use it; or
// ("", false, err): the override is invalid, abort by name. There is no
// third outcome: no input returns useEnv=true together with a non-nil
// error, and no non-empty input returns useEnv=false with a nil error.
// That absence is the property that forbids a silent fallback to a local
// `go build` on a bad override — a fallback here would let a job claiming
// to drive the notarized release binary through a real tmux pane quietly
// drive a locally rebuilt one instead.
//
// The checks below are STAT-LEVEL only: they confirm raw exists, is a
// regular file, and carries at least one UNIX execute-permission bit —
// never that it is a valid, architecture-compatible executable. This
// mirrors test/integration/main_test.go's resolveTestBinPath exactly
// (byte-faithful duplication per D-09), including this same deliberate
// scope limit.
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
// resolveTestBinPath). When unset, it builds the real release binary
// hermetically, once, into a package-level temp dir before any test in
// this package runs. When set and valid, it runs every test against that
// externally supplied binary instead. When set and invalid, TestMain aborts
// before creating a temp dir and before building: it prints a message
// naming the environment variable and the offending path to stderr and
// exits non-zero.
//
// TestMain always resolves the binary regardless of tmux's presence on
// PATH — a failed build must stay loud and immediate. tmux absence is
// handled per-test by requireTmux (session.go), never by skipping the
// build here.
func TestMain(m *testing.M) {
	resolved, useEnv, err := resolveTestBinPath(os.Getenv(testBinEnvVar))
	if err != nil {
		fmt.Fprintln(os.Stderr, "tmux: TestMain:", err)
		os.Exit(1)
	}
	if useEnv {
		binPath = resolved
		os.Exit(m.Run())
	}

	tmpDir, err := os.MkdirTemp("", "codegraph-tmux-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tmux: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}

	binPath = filepath.Join(tmpDir, "codegraph")
	buildCmd := exec.Command("go", "build", "-o", binPath, "github.com/seanb4t/codegraph-go/cmd/codegraph")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "tmux: TestMain: go build github.com/seanb4t/codegraph-go/cmd/codegraph failed: %v\n%s\n", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}
