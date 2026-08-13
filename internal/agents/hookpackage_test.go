package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// sessionNudgeScriptPath is the shipped SessionStart nudge script, reached
// from internal/agents the same way instructions_contract_test.go's
// readmePath reaches the repo-root README: a relative path read directly,
// with a missing file treated as a loud regression, never a t.Skip. The
// script must exist and be executable for NUDGE-01/02 to hold at all.
const sessionNudgeScriptPath = "../../.claude/hooks/session-nudge.sh"

// claudeSettingsFilePath is the project-scoped settings file Claude Code
// actually reads for a committed, non-plugin SessionStart hook
// registration (06-RESEARCH.md's Critical Correction). Same
// missing-file-is-a-regression convention as sessionNudgeScriptPath.
const claudeSettingsFilePath = "../../.claude/settings.json"

// claudeHooksFragmentPath is Phase 7's go:embed source fragment (D-04) —
// NOT itself read by Claude Code in this repository. It exists so a later
// phase has a stable, versioned path to embed from without re-deriving the
// SessionStart block from settings.json at that phase's authoring time.
const claudeHooksFragmentPath = "../../.claude/hooks/hooks.json"

// nudgeLine is D-06's message verbatim, hand-typed here rather than
// derived: it is a fixed one-line pointer with no runtime source to derive
// from, the same situation instructions_contract_test.go documents for its
// own allowlistEnvName literal. This is the FIRST of two layers pinning
// this text — the derived-honesty guards over the same string (real tool
// names only, no env var, no host path, no counts) land in 06-03's
// internal/mcp/skill_claims_drift_test.go, which extends
// resources_schema_drift_test.go's checkers to cover it. This constant
// only pins the bytes; it proves nothing about their honesty.
const nudgeLine = "This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions."

// runSessionNudge runs the shipped nudge script against dir. When useEnv is
// true, dir is passed via CLAUDE_PROJECT_DIR (with any inherited
// CLAUDE_PROJECT_DIR entry filtered out first, mirroring the useEnv==false
// branch below, so the override is deterministic regardless of what the
// invoking process happens to carry) and the process's own working
// directory is left at the test's cwd (so the script must resolve the path
// from the env var, not cwd). When useEnv is false, CLAUDE_PROJECT_DIR is
// stripped from the environment entirely and cmd.Dir is set to dir instead,
// exercising the "unset env, resolve against own working directory"
// boundary. Returns stdout, stderr, the process exit code, and an error for
// any failure that is not an *exec.ExitError (a non-zero exit is a normal,
// inspectable outcome here — anything else, e.g. the binary not existing, is
// a harness-level failure the caller did not ask about). This function does
// NOT call t.Fatalf itself: per the testing package's documented contract,
// FailNow/Fatal/Fatalf must be called from the goroutine running the test
// function, not from other goroutines spawned during the test (this helper
// is also called from spawned goroutines in the concurrency subtest below).
// Callers running on the test's own goroutine may t.Fatalf on a non-nil
// error immediately; callers in a spawned goroutine must capture the error
// and report it back to the goroutine that owns t.
func runSessionNudge(t *testing.T, dir string, useEnv bool) (stdout, stderr string, exitCode int, err error) {
	t.Helper()

	scriptPath, absErr := filepath.Abs(sessionNudgeScriptPath)
	if absErr != nil {
		return "", "", 0, fmt.Errorf("resolve script path: %w", absErr)
	}

	cmd := exec.Command(scriptPath)
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "CLAUDE_PROJECT_DIR=") {
			continue
		}
		env = append(env, kv)
	}
	if useEnv {
		cmd.Env = append(env, "CLAUDE_PROJECT_DIR="+dir)
	} else {
		cmd.Env = env
		cmd.Dir = dir
	}

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr == nil {
		return stdout, stderr, 0, nil
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		return stdout, stderr, 0, fmt.Errorf("runSessionNudge(%s): unexpected error (not an exit error): %w", dir, runErr)
	}
	return stdout, stderr, exitErr.ExitCode(), nil
}

// TestSessionNudgeBehavesPerIndexPresence is table-driven per NUDGE-01/02:
// stdout, stderr, and exit code are asserted separately for every boundary
// the plan names — directory vs. regular-file vs. absent .codegraph, and
// CLAUDE_PROJECT_DIR present vs. entirely unset.
func TestSessionNudgeBehavesPerIndexPresence(t *testing.T) {
	cases := []struct {
		name       string
		useEnv     bool
		setup      func(t *testing.T, dir string)
		wantStdout string
		wantStderr string
		wantExit   int
	}{
		{
			name:   "codegraph dir present, env set",
			useEnv: true,
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
					t.Fatalf("mkdir .codegraph: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".codegraph", "marker"), []byte("x"), 0o644); err != nil {
					t.Fatalf("seed .codegraph/marker: %v", err)
				}
			},
			wantStdout: nudgeLine + "\n",
			wantStderr: "",
			wantExit:   0,
		},
		{
			name:       "no codegraph entry at all, env set",
			useEnv:     true,
			setup:      func(t *testing.T, dir string) {},
			wantStdout: "",
			wantStderr: "",
			wantExit:   0,
		},
		{
			name:   "codegraph present as a regular file, env set",
			useEnv: true,
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, ".codegraph"), []byte("not a directory"), 0o644); err != nil {
					t.Fatalf("seed .codegraph file: %v", err)
				}
			},
			wantStdout: "",
			wantStderr: "",
			wantExit:   0,
		},
		{
			name:   "codegraph present as an empty directory, env set",
			useEnv: true,
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
					t.Fatalf("mkdir empty .codegraph: %v", err)
				}
			},
			wantStdout: nudgeLine + "\n",
			wantStderr: "",
			wantExit:   0,
		},
		{
			name:   "env unset, cmd.Dir indexed",
			useEnv: false,
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
					t.Fatalf("mkdir .codegraph: %v", err)
				}
			},
			wantStdout: nudgeLine + "\n",
			wantStderr: "",
			wantExit:   0,
		},
		{
			name:       "env unset, cmd.Dir unindexed",
			useEnv:     false,
			setup:      func(t *testing.T, dir string) {},
			wantStdout: "",
			wantStderr: "",
			wantExit:   0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)

			stdout, stderr, exitCode, err := runSessionNudge(t, dir, tc.useEnv)
			if err != nil {
				t.Fatalf("runSessionNudge: %v", err)
			}
			if stdout != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tc.wantStdout)
			}
			if stderr != tc.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, tc.wantStderr)
			}
			if exitCode != tc.wantExit {
				t.Errorf("exit code = %d, want %d", exitCode, tc.wantExit)
			}
		})
	}
}

// TestSessionNudgeOutputIsPinnedAndStateless covers precision (byte-exact
// output), concurrency (N simultaneous invocations agree), and
// statelessness (the script touches no filesystem state besides the one
// read it performs) — plus a guard-the-guard sub-test proving nudgeLine
// itself is non-empty and that a deliberately wrong expectation would not
// match, following TestResourceCountCheckerIsNotVacuous's shape.
func TestSessionNudgeOutputIsPinnedAndStateless(t *testing.T) {
	t.Run("precision", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
			t.Fatalf("mkdir .codegraph: %v", err)
		}

		stdout, _, exitCode, err := runSessionNudge(t, dir, true)
		if err != nil {
			t.Fatalf("runSessionNudge: %v", err)
		}
		want := nudgeLine + "\n"
		if stdout != want {
			t.Fatalf("stdout not byte-equal:\ngot=%q\nwant=%q", stdout, want)
		}
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if got := strings.Count(stdout, "\n"); got != 1 {
			t.Fatalf("strings.Count(stdout, \"\\n\") = %d, want 1", got)
		}
	})

	t.Run("concurrency and statelessness", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
			t.Fatalf("mkdir .codegraph: %v", err)
		}

		before, err := walkEntries(dir)
		if err != nil {
			t.Fatalf("walk before: %v", err)
		}

		const n = 8
		results := make([]struct {
			stdout string
			exit   int
			err    error
		}, n)
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				// Capture the error rather than calling t.Fatalf here: per
				// the testing package's documented contract, Fatal/FailNow
				// must be called from the goroutine running the test
				// function, not from a goroutine spawned during the test.
				// The main test goroutine reports failures below, after
				// wg.Wait().
				stdout, _, exit, err := runSessionNudge(t, dir, true)
				results[i].stdout = stdout
				results[i].exit = exit
				results[i].err = err
			}(i)
		}
		wg.Wait()

		want := nudgeLine + "\n"
		for i, r := range results {
			if r.err != nil {
				t.Errorf("goroutine %d: runSessionNudge: %v", i, r.err)
				continue
			}
			if r.stdout != want {
				t.Errorf("goroutine %d: stdout = %q, want %q", i, r.stdout, want)
			}
			if r.exit != 0 {
				t.Errorf("goroutine %d: exit = %d, want 0", i, r.exit)
			}
		}

		after, err := walkEntries(dir)
		if err != nil {
			t.Fatalf("walk after: %v", err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("tree entries changed after concurrent burst:\nbefore=%v\nafter=%v", before, after)
		}
	})

	t.Run("guard the guard", func(t *testing.T) {
		if nudgeLine == "" {
			t.Fatalf("nudgeLine must be non-empty")
		}
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
			t.Fatalf("mkdir .codegraph: %v", err)
		}
		stdout, _, _, err := runSessionNudge(t, dir, true)
		if err != nil {
			t.Fatalf("runSessionNudge: %v", err)
		}
		wrong := "this is not the nudge line\n"
		if stdout == wrong {
			t.Fatalf("deliberately wrong expectation matched real output — assertion is vacuous")
		}
	})
}

// walkEntries returns the sorted set of every path under dir (relative to
// dir), used to prove a burst of script invocations creates, modifies, or
// removes nothing.
func walkEntries(dir string) ([]string, error) {
	var entries []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		entries = append(entries, rel)
		return nil
	})
	return entries, err
}

// sessionStartBlock decodes path's JSON and returns the value at
// hooks.SessionStart, t.Fatalf'ing if the file is missing or the path is
// absent — used to compare .claude/settings.json's live registration
// against .claude/hooks/hooks.json's embed fragment (T-06 Task 2).
func sessionStartBlock(t *testing.T, path string) any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("sessionStartBlock: read %s: %v", path, err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("sessionStartBlock: unmarshal %s: %v", path, err)
	}
	hooks, ok := decoded["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("sessionStartBlock: %s has no top-level \"hooks\" object", path)
	}
	sessionStart, ok := hooks["SessionStart"]
	if !ok {
		t.Fatalf("sessionStartBlock: %s has no hooks.SessionStart", path)
	}
	return sessionStart
}

// TestHookRegistrationMatchesFragmentAndScript pins Phase 7's embed
// fragment equal to the live registration this repository actually runs
// (D-04, A2), then proves every command path the registration names
// resolves to a real, executable file on disk — so a rename of the script
// fails this test instead of silently disabling the nudge.
func TestHookRegistrationMatchesFragmentAndScript(t *testing.T) {
	settingsBlock := sessionStartBlock(t, claudeSettingsFilePath)
	fragmentBlock := sessionStartBlock(t, claudeHooksFragmentPath)

	if !reflect.DeepEqual(settingsBlock, fragmentBlock) {
		t.Fatalf("hooks.SessionStart differs between %s and %s — Phase 7 would embed a fragment that differs from what actually runs here.\nsettings.json: %#v\nhooks.json:    %#v",
			claudeSettingsFilePath, claudeHooksFragmentPath, settingsBlock, fragmentBlock)
	}

	entries, ok := settingsBlock.([]any)
	if !ok {
		t.Fatalf("hooks.SessionStart is not an array: %#v", settingsBlock)
	}
	if len(entries) == 0 {
		t.Fatalf("hooks.SessionStart is empty")
	}

	for _, e := range entries {
		entry, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("SessionStart entry is not an object: %#v", e)
		}
		hooksArr, ok := entry["hooks"].([]any)
		if !ok || len(hooksArr) == 0 {
			t.Fatalf("SessionStart entry has no hooks array: %#v", entry)
		}
		for _, h := range hooksArr {
			hookObj, ok := h.(map[string]any)
			if !ok {
				t.Fatalf("hook entry is not an object: %#v", h)
			}
			command, _ := hookObj["command"].(string)
			if command == "" {
				t.Fatalf("hook entry has no command string: %#v", hookObj)
			}
			resolved := strings.Replace(command, "${CLAUDE_PROJECT_DIR}", "../..", 1)
			info, err := os.Stat(resolved)
			if err != nil {
				t.Fatalf("command path %q (resolved %q) does not exist: %v", command, resolved, err)
			}
			if info.Mode()&0o111 == 0 {
				t.Fatalf("command path %q (resolved %q) is not executable: mode %v", command, resolved, info.Mode())
			}
		}
	}
}

// TestHookRegistrationMatchesFragmentAndScript_ComparisonDiscriminates is
// the guard-the-guard sub-test for the reflect.DeepEqual comparison this
// test relies on, following TestResourceStemSetDiffIsNotVacuous's shape:
// proves the comparison actually discriminates equal blocks from a
// differing matcher and a differing command, rather than passing
// unconditionally.
func TestHookRegistrationMatchesFragmentAndScript_ComparisonDiscriminates(t *testing.T) {
	base := []any{
		map[string]any{
			"matcher": "startup",
			"hooks": []any{
				map[string]any{"type": "command", "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh"},
			},
		},
	}
	sameShapeDifferentBacking := []any{
		map[string]any{
			"matcher": "startup",
			"hooks": []any{
				map[string]any{"type": "command", "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh"},
			},
		},
	}
	differingMatcher := []any{
		map[string]any{
			"matcher": "clear",
			"hooks": []any{
				map[string]any{"type": "command", "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh"},
			},
		},
	}
	differingCommand := []any{
		map[string]any{
			"matcher": "startup",
			"hooks": []any{
				map[string]any{"type": "command", "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/other-script.sh"},
			},
		},
	}

	cases := []struct {
		name string
		a, b any
		want bool
	}{
		{"equal blocks", base, sameShapeDifferentBacking, true},
		{"differing matcher", base, differingMatcher, false},
		{"differing command", base, differingCommand, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := reflect.DeepEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("reflect.DeepEqual = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestClaudeInstallPreservesHooksBlock pins T-06-03: codegraph install
// --local followed by codegraph uninstall --local must not disturb the
// dogfooded hook registration. Drives the real addClaudeAllowPermission /
// removeClaudeAllowPermission merge functions (internal/agents/claude.go)
// against a copy of the committed settings.json, per claude.go's own
// doc comment that both round-trip the whole decoded map and only touch
// "permissions".
func TestClaudeInstallPreservesHooksBlock(t *testing.T) {
	original, err := os.ReadFile(claudeSettingsFilePath)
	if err != nil {
		t.Fatalf("read committed settings.json: %v", err)
	}

	dir := t.TempDir()
	copyPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(copyPath, original, 0o644); err != nil {
		t.Fatalf("seed copy: %v", err)
	}

	var originalDecoded map[string]any
	if err := json.Unmarshal(original, &originalDecoded); err != nil {
		t.Fatalf("unmarshal original: %v", err)
	}
	originalHooks := originalDecoded["hooks"]

	if _, err := addClaudeAllowPermission(copyPath); err != nil {
		t.Fatalf("addClaudeAllowPermission: %v", err)
	}

	afterAdd, err := readJSONFile(copyPath)
	if err != nil {
		t.Fatalf("read after add: %v", err)
	}
	if !reflect.DeepEqual(afterAdd["hooks"], originalHooks) {
		t.Fatalf("hooks block changed after addClaudeAllowPermission:\nbefore=%#v\nafter=%#v", originalHooks, afterAdd["hooks"])
	}
	permissions, ok := afterAdd["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permissions missing after addClaudeAllowPermission: %#v", afterAdd)
	}
	allow, ok := permissions["allow"].([]any)
	if !ok {
		t.Fatalf("permissions.allow missing after addClaudeAllowPermission: %#v", permissions)
	}
	found := false
	for _, v := range allow {
		if s, ok := v.(string); ok && s == claudeAllowToken {
			found = true
		}
	}
	if !found {
		t.Fatalf("permissions.allow does not contain %q: %v", claudeAllowToken, allow)
	}

	if _, err := removeClaudeAllowPermission(copyPath); err != nil {
		t.Fatalf("removeClaudeAllowPermission: %v", err)
	}

	afterRemove, err := readJSONFile(copyPath)
	if err != nil {
		t.Fatalf("read after remove: %v", err)
	}
	if !reflect.DeepEqual(afterRemove["hooks"], originalHooks) {
		t.Fatalf("hooks block changed after removeClaudeAllowPermission:\nbefore=%#v\nafter=%#v", originalHooks, afterRemove["hooks"])
	}
}
