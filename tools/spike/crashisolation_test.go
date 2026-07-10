// Crash-isolation dimension of the D-05 spike (RESEARCH Pitfall 5: this
// property only manifests on malformed/adversarial input, never on valid
// code, so it must be measured with deliberately broken input — see
// cases.go).
//
// Design: each malformed case is run in a genuinely separate OS process
// (tools/spike's own `crash-case` subcommand, built once into a temp
// binary), not merely a recovered panic inside this test binary. This
// matters because the CGo arm's crash-isolation tail risk (T-01-01) is an
// uncatchable C-level segfault — Go's recover() cannot intercept it, and if
// it happened inside this test's own process it would take the whole test
// binary down with no way to record the outcome. Running it as a child
// process lets the PARENT observe the child's abnormal termination (a
// captured OS signal) from the outside, which is the only way to safely
// measure this property at all.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var (
	buildOnce sync.Once
	spikeBin  string
	buildErr  error
)

// spikeBinary builds tools/spike itself into a standalone binary exactly
// once per test run, so each of the ~20 crash cases below execs a
// pre-built binary rather than paying a `go build`/`go run` compile cost
// per case.
func spikeBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		root, err := findRepoRoot()
		if err != nil {
			buildErr = fmt.Errorf("find repo root: %w", err)
			return
		}
		bin := filepath.Join(os.TempDir(), "codegraph-go-spike-harness")
		cmd := exec.Command("go", "build", "-o", bin, "./tools/spike")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("build spike harness: %w: %s", err, out)
			return
		}
		spikeBin = bin
	})
	if buildErr != nil {
		t.Fatalf("spike harness build failed: %v", buildErr)
	}
	return spikeBin
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found in any parent directory")
		}
		dir = parent
	}
}

// crashOutcome is one case's observed result: did the child process exit
// cleanly (with a PARSE_OK/PARSE_ERROR result printed by main.go's
// crash-case subcommand), did it die from an uncaught OS signal (a genuine
// crash — CGo's uncatchable tail risk), or did it have to be killed after
// exceeding the per-case timeout (a hang, distinct from either).
type crashOutcome struct {
	Backend    string
	Lang       string
	Case       string
	Stdout     string
	ExitCode   int
	Signaled   bool
	SignalName string
	TimedOut   bool
}

func (o crashOutcome) String() string {
	switch {
	case o.TimedOut:
		return "TIMEOUT (killed after deadline — treat as a hang, not a crash)"
	case o.Signaled:
		return fmt.Sprintf("CRASHED — killed by signal %s (uncatchable from inside the process)", o.SignalName)
	default:
		return fmt.Sprintf("SURVIVED — exited %d, output %q", o.ExitCode, o.Stdout)
	}
}

const crashCaseTimeout = 15 * time.Second

func runCrashCase(t *testing.T, bin, backend, lang, caseName string) crashOutcome {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), crashCaseTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "crash-case", backend, lang, caseName)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	outcome := crashOutcome{Backend: backend, Lang: lang, Case: caseName, Stdout: strings.TrimSpace(stdout.String())}

	if ctx.Err() == context.DeadlineExceeded {
		outcome.TimedOut = true
		return outcome
	}
	if err == nil {
		return outcome
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		outcome.ExitCode = exitErr.ExitCode()
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			outcome.Signaled = true
			outcome.SignalName = status.Signal().String()
		}
		return outcome
	}

	t.Fatalf("running crash-case %s/%s/%s: %v (stderr: %q)", backend, lang, caseName, err, stderr.String())
	return outcome
}

// caseSpec ties a malformed-input case (cases.go) to the language(s) it is
// meaningful for. "garbage" and "oversized" are language-agnostic (raw
// bytes / size, not grammar-specific); the truncated/deep-nesting cases are
// built from and target one specific grammar.
var caseSpecs = []struct {
	name  string
	langs []string
}{
	{"truncated_go", []string{"go"}},
	{"truncated_python", []string{"python"}},
	{"garbage", []string{"go", "python"}},
	{"deep_nesting_go", []string{"go"}},
	{"deep_nesting_python", []string{"python"}},
	{"oversized", []string{"go", "python"}},
}

// TestCrashIsolation is the D-05 crash-isolation measurement referenced by
// PARSER-DECISION.md: for every (backend, language, malformed-case)
// combination, record whether the backend's process survived, crashed via
// an uncatchable OS signal, or hung past the deadline.
func TestCrashIsolation(t *testing.T) {
	bin := spikeBinary(t)

	backends := []string{"cgo", "wazero"}
	langs := []string{"go", "python"}

	var results []crashOutcome
	for _, backend := range backends {
		for _, lang := range langs {
			for _, cs := range caseSpecs {
				applies := false
				for _, l := range cs.langs {
					if l == lang {
						applies = true
						break
					}
				}
				if !applies {
					continue
				}

				outcome := runCrashCase(t, bin, backend, lang, cs.name)
				results = append(results, outcome)
				t.Logf("[%s/%s/%s] %s", backend, lang, cs.name, outcome)
			}
		}
	}

	t.Log("=== TestCrashIsolation summary (fold into PARSER-DECISION.md) ===")
	for _, r := range results {
		t.Logf("%-8s %-8s %-22s -> %s", r.Backend, r.Lang, r.Case, r)
	}
}
