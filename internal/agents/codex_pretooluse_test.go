package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// TestCodexHooksFragmentShape pins the embedded Codex PreToolUse fragment's
// exact shape (D-20): one event key, one group, matcher "^Bash$", one
// handler whose keys are exactly {type, command, timeout}, type "command",
// timeout 5, command == codexPreToolFragmentCommand.
func TestCodexHooksFragmentShape(t *testing.T) {
	data, err := claudeassets.CodexHooksFragment()
	if err != nil {
		t.Fatalf("CodexHooksFragment: %v", err)
	}
	var decoded struct {
		Hooks map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode fragment: %v", err)
	}
	if len(decoded.Hooks) != 1 {
		t.Fatalf("fragment has %d event keys, want 1: %#v", len(decoded.Hooks), decoded.Hooks)
	}
	var groups []map[string]any
	if err := json.Unmarshal(decoded.Hooks["PreToolUse"], &groups); err != nil {
		t.Fatalf("decode hooks.PreToolUse: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("hooks.PreToolUse has %d groups, want 1: %#v", len(groups), groups)
	}
	group := groups[0]
	if group["matcher"] != "^Bash$" {
		t.Fatalf("group matcher = %#v, want \"^Bash$\"", group["matcher"])
	}
	handlers, _ := group["hooks"].([]any)
	if len(handlers) != 1 {
		t.Fatalf("group has %d handlers, want 1: %#v", len(handlers), handlers)
	}
	handler, ok := handlers[0].(map[string]any)
	if !ok {
		t.Fatalf("handler is not an object: %#v", handlers[0])
	}
	if len(handler) != 3 {
		t.Fatalf("handler has %d keys, want exactly 3 (type, command, timeout): %#v", len(handler), handler)
	}
	if handler["type"] != "command" {
		t.Errorf("handler type = %#v, want \"command\"", handler["type"])
	}
	if handler["command"] != codexPreToolFragmentCommand {
		t.Errorf("handler command = %#v, want %q", handler["command"], codexPreToolFragmentCommand)
	}
	if handler["timeout"] != float64(5) {
		t.Errorf("handler timeout = %#v, want 5", handler["timeout"])
	}
}

// readJSONHooksFile decodes path as a generic JSON object, failing the
// test on error — used by the Codex hooks.json assertions below.
func readJSONHooksFile(t *testing.T, path string) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(readFile(t, path)), &decoded); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return decoded
}

// TestCodex_Install_PreToolNudgeOn_WritesGuardAndGroup (D-18, D-19, D-20):
// an opt-in install writes an executable rendered guard and appends
// codegraph's own PreToolUse group LAST in hooks.json, leaving a
// pre-existing foreign "^Bash$" group untouched and first, and never
// carrying the ExecPath string in hooks.json itself.
func TestCodex_Install_PreToolNudgeOn_WritesGuardAndGroup(t *testing.T) {
	t.Run("local", func(t *testing.T) {
		fakeHome(t)
		t.Chdir(t.TempDir())

		hooksPath := filepath.Join(".codex", "hooks.json")
		writeFile(t, hooksPath, `{"hooks":{"PreToolUse":[{"matcher":"^Bash$","hooks":[{"type":"command","command":"echo foreign","timeout":10}]}]}}`)

		res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		if len(res.Errors) != 0 {
			t.Fatalf("Install errors: %v", res.Errors)
		}

		guard := filepath.Join(".codex", "hooks", "codegraph-pretooluse.sh")
		info, err := os.Stat(guard)
		if err != nil {
			t.Fatalf("guard not written: %v", err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("guard mode %v is not executable", info.Mode())
		}
		content := readFile(t, guard)
		if !strings.Contains(content, "\ncodegraph_bin='/usr/local/bin/codegraph'\n") {
			t.Fatalf("guard lacks the rendered ExecPath line:\n%s", content)
		}
		if strings.Contains(content, "@codegraph-exec-path@") {
			t.Fatalf("guard still carries the unrendered token:\n%s", content)
		}

		decoded := readJSONHooksFile(t, hooksPath)
		hooks, _ := decoded["hooks"].(map[string]any)
		groups, _ := hooks["PreToolUse"].([]any)
		if len(groups) != 2 {
			t.Fatalf("hooks.PreToolUse has %d groups, want 2 (foreign + codegraph's own): %#v", len(groups), groups)
		}
		firstGroup, _ := groups[0].(map[string]any)
		firstHandlers, _ := firstGroup["hooks"].([]any)
		firstHandler, _ := firstHandlers[0].(map[string]any)
		if firstHandler["command"] != "echo foreign" {
			t.Fatalf("the foreign group is no longer first: %#v", groups[0])
		}
		lastGroup, _ := groups[len(groups)-1].(map[string]any)
		if lastGroup["matcher"] != "^Bash$" {
			t.Fatalf("last group matcher = %#v, want \"^Bash$\"", lastGroup["matcher"])
		}
		lastHandlers, _ := lastGroup["hooks"].([]any)
		lastHandler, _ := lastHandlers[0].(map[string]any)
		if lastHandler["command"] != codexPreToolFragmentCommand {
			t.Fatalf("last group command = %#v, want %q", lastHandler["command"], codexPreToolFragmentCommand)
		}
		if lastHandler["timeout"] != float64(5) {
			t.Fatalf("last group timeout = %#v, want 5", lastHandler["timeout"])
		}
		if strings.Contains(readFile(t, hooksPath), "/usr/local/bin/codegraph") {
			t.Fatalf("hooks.json carries the ExecPath string (D-19 violation):\n%s", readFile(t, hooksPath))
		}
	})

	t.Run("global", func(t *testing.T) {
		home := fakeHome(t)
		t.Chdir(t.TempDir())

		res := codexTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		if len(res.Errors) != 0 {
			t.Fatalf("Install errors: %v", res.Errors)
		}

		guard := filepath.Join(home, ".codex", "hooks", "codegraph-pretooluse.sh")
		if _, err := os.Stat(guard); err != nil {
			t.Fatalf("global guard not written: %v", err)
		}
		hooksPath := filepath.Join(home, ".codex", "hooks.json")
		decoded := readJSONHooksFile(t, hooksPath)
		hooks, _ := decoded["hooks"].(map[string]any)
		groups, _ := hooks["PreToolUse"].([]any)
		if len(groups) != 1 {
			t.Fatalf("hooks.PreToolUse has %d groups, want 1: %#v", len(groups), groups)
		}
		group, _ := groups[0].(map[string]any)
		handlers, _ := group["hooks"].([]any)
		handler, _ := handlers[0].(map[string]any)
		wantCmd := shellSingleQuote(guard)
		if handler["command"] != wantCmd {
			t.Fatalf("group command = %#v, want %q", handler["command"], wantCmd)
		}
		if strings.Contains(readFile(t, hooksPath), "/usr/local/bin/codegraph") {
			t.Fatalf("hooks.json carries the ExecPath string (D-19 violation)")
		}
	})
}

// TestCodex_Install_DefaultWritesNoHooks: an install with the zero
// PreToolNudge value writes neither hooks.json nor the guard — the nudge
// is opt-in (D-18).
func TestCodex_Install_DefaultWritesNoHooks(t *testing.T) {
	for _, loc := range []Location{LocationLocal, LocationGlobal} {
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())

			guard, err := codexPreToolGuardPath(loc)
			if err != nil {
				t.Fatalf("codexPreToolGuardPath(%s): %v", loc, err)
			}
			hooksPath, err := codexHooksJSONPath(loc)
			if err != nil {
				t.Fatalf("codexHooksJSONPath(%s): %v", loc, err)
			}

			res := codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
			if len(res.Errors) != 0 {
				t.Fatalf("Install errors: %v", res.Errors)
			}
			if _, err := os.Lstat(guard); !os.IsNotExist(err) {
				t.Fatalf("default install wrote %s (Lstat err %v); the nudge is opt-in (D-18)", guard, err)
			}
			if _, err := os.Lstat(hooksPath); !os.IsNotExist(err) {
				t.Fatalf("default install wrote %s (Lstat err %v); the nudge is opt-in (D-18)", hooksPath, err)
			}

			// Positive control: an opt-in install into the same scope makes
			// both the guard and the registration appear, so the absence
			// checks above can see presence.
			res = codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			if len(res.Errors) != 0 {
				t.Fatalf("opt-in Install errors: %v", res.Errors)
			}
			if _, err := os.Lstat(guard); err != nil {
				t.Fatalf("positive control: opt-in install did not write %s: %v", guard, err)
			}
			if _, err := os.Lstat(hooksPath); err != nil {
				t.Fatalf("positive control: opt-in install did not write %s: %v", hooksPath, err)
			}
		})
	}
}

// TestCodex_Uninstall_RemovesHooks: after an On install, Uninstall removes
// the own group (deleting a hooks.json it empties) and the guard; on a
// never-opted location it reports not-found and no error.
func TestCodex_Uninstall_RemovesHooks(t *testing.T) {
	for _, loc := range []Location{LocationLocal, LocationGlobal} {
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())

			// Never-opted-in location: uninstall reports not-found, no error.
			res := codexTarget{}.Uninstall(loc)
			if len(res.Errors) != 0 {
				t.Fatalf("Uninstall on a never-opted location returned errors: %v", res.Errors)
			}

			installRes := codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			if len(installRes.Errors) != 0 {
				t.Fatalf("Install errors: %v", installRes.Errors)
			}
			guard, err := codexPreToolGuardPath(loc)
			if err != nil {
				t.Fatalf("codexPreToolGuardPath(%s): %v", loc, err)
			}
			hooksPath, err := codexHooksJSONPath(loc)
			if err != nil {
				t.Fatalf("codexHooksJSONPath(%s): %v", loc, err)
			}
			if _, err := os.Stat(guard); err != nil {
				t.Fatalf("precondition: guard not written: %v", err)
			}
			if _, err := os.Stat(hooksPath); err != nil {
				t.Fatalf("precondition: hooks.json not written: %v", err)
			}

			uninstallRes := codexTarget{}.Uninstall(loc)
			if len(uninstallRes.Errors) != 0 {
				t.Fatalf("Uninstall errors: %v", uninstallRes.Errors)
			}
			if _, err := os.Lstat(guard); !os.IsNotExist(err) {
				t.Fatalf("guard still present after uninstall (Lstat err %v)", err)
			}
			// Only codegraph's own group ever existed, so removing it should
			// empty and delete hooks.json entirely (writeHookEntry/
			// removeHookEntry's keep-clean cascade).
			if _, err := os.Lstat(hooksPath); !os.IsNotExist(err) {
				t.Fatalf("hooks.json still present after uninstall (Lstat err %v); it should be deleted (nothing else was ever in it)", err)
			}
		})
	}
}

// writeRenderedCodexGuard renders one of the two REAL embedded Codex guard
// templates (chosen by loc) for binPath and writes it, executable, at the
// exact installed layout the local guard's own root derivation depends on:
// <projectDir>/.codex/hooks/codegraph-pretooluse.sh. The global guard does
// not care about its own path (it checks $PWD, D-22), so it is written at
// the same location purely for helper reuse.
func writeRenderedCodexGuard(t *testing.T, loc Location, binPath, projectDir string) string {
	t.Helper()
	rendered, err := renderCodexPreToolGuard(loc, binPath)
	if err != nil {
		t.Fatalf("renderCodexPreToolGuard(%s, %q): %v", loc, binPath, err)
	}
	guard := filepath.Join(projectDir, ".codex", "hooks", "codegraph-pretooluse.sh")
	writeStub(t, guard, rendered, 0o755)
	return guard
}

// runCodexPreToolGuard runs guardPath with stdin, with the CHILD PROCESS's
// actual OS-level working directory (cmd.Dir, what getcwd() returns) set
// to procCwd — never left as whatever the test process itself inherited.
// $PWD is explicitly set to match procCwd: bash and other shells re-derive
// $PWD from getcwd() at startup whenever an inherited PWD does not match
// the real cwd (verified empirically — an attempt to fake a MISMATCHED
// $PWD here was silently self-healed by the shell, not a usable negative
// control), so the only way to genuinely exercise "what if this guard
// read $PWD/cwd instead of deriving its root from $0" is to make the
// PROCESS's real working directory itself wrong. Every LOCAL guard test
// below therefore passes a procCwd that is NOT the project directory the
// guard script actually lives under (Family (f2), 07-MUTATION-LOG.md) —
// the guard's own root derivation must still find the right .codegraph
// purely from $0, with no correct cwd to fall back on. forcedEnv, when
// non-nil, REPLACES the environment entirely (used by the empty-PATH
// subtest) — PWD is still appended afterward either way.
func runCodexPreToolGuard(t *testing.T, guardPath, procCwd string, extraEnv []string, forcedEnv []string, stdin string) (stdout, stderr string, exit int, err error) {
	t.Helper()

	var env []string
	if forcedEnv != nil {
		env = append([]string{}, forcedEnv...)
	} else {
		for _, kv := range os.Environ() {
			if k, _, ok := strings.Cut(kv, "="); ok && k == "PWD" {
				continue
			}
			env = append(env, kv)
		}
		env = append(env, extraEnv...)
	}
	env = append(env, "PWD="+procCwd)

	cmd := exec.Command(guardPath)
	cmd.Dir = procCwd
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout, stderr = outBuf.String(), errBuf.String()
	if runErr == nil {
		return stdout, stderr, 0, nil
	}
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		return stdout, stderr, 0, fmt.Errorf("runCodexPreToolGuard(%s): %w", guardPath, runErr)
	}
	return stdout, stderr, exitErr.ExitCode(), nil
}

// TestCodexPreToolUseGuard is the guard-level D-16/D-22 suite over BOTH
// real embedded templates, rendered with stub binaries: every case exits
// 0, starts no process in an un-indexed repo, and — when it does start the
// stub — passes stdin and stdout through untouched.
func TestCodexPreToolUseGuard(t *testing.T) {
	cases := []struct {
		name        string
		loc         Location
		codegraph   string // "dir", "file" or "" (absent)
		stub        string // stub content; "" = binary missing
		stubMode    os.FileMode
		emptyPath   bool
		wantStarted bool
		check       func(t *testing.T, stubDir, stdout, stderr string)
	}{
		{name: "local/indexed_binary_ok", loc: LocationLocal, codegraph: "dir", stub: stubOK, stubMode: 0o755, wantStarted: true,
			check: func(t *testing.T, stubDir, stdout, stderr string) {
				if got := readFile(t, filepath.Join(stubDir, "stdin")); got != preToolGuardEvent {
					t.Errorf("binary stdin = %q, want the event byte-identical %q", got, preToolGuardEvent)
				}
				if stdout != "STUB-OUT\n" {
					t.Errorf("stdout = %q, want the binary's stdout unmodified %q", stdout, "STUB-OUT\n")
				}
				if stderr != "" {
					t.Errorf("stderr = %q, want empty", stderr)
				}
			}},
		{name: "local/not_indexed", loc: LocationLocal, codegraph: "", stub: stubOK, stubMode: 0o755, wantStarted: false, check: wantNoOutput},
		{name: "local/codegraph_is_file", loc: LocationLocal, codegraph: "file", stub: stubOK, stubMode: 0o755, wantStarted: false, check: wantNoOutput},
		{name: "local/binary_missing", loc: LocationLocal, codegraph: "dir", stub: "", wantStarted: false, check: wantNoOutput},
		{name: "local/binary_not_executable", loc: LocationLocal, codegraph: "dir", stub: stubOK, stubMode: 0o644, wantStarted: false,
			check: func(t *testing.T, _, stdout, _ string) {
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
			}},
		{name: "local/binary_exits_nonzero", loc: LocationLocal, codegraph: "dir", stub: stubFail, stubMode: 0o755, wantStarted: true},
		{name: "local/binary_crashes", loc: LocationLocal, codegraph: "dir", stub: stubCrash, stubMode: 0o755, wantStarted: true},
		{name: "local/root_from_own_path_with_empty_path", loc: LocationLocal, codegraph: "dir", stub: stubOK, stubMode: 0o755, emptyPath: true, wantStarted: true},
		{name: "global/indexed_pwd", loc: LocationGlobal, codegraph: "dir", stub: stubOK, stubMode: 0o755, wantStarted: true},
		{name: "global/not_indexed_pwd", loc: LocationGlobal, codegraph: "", stub: stubOK, stubMode: 0o755, wantStarted: false, check: wantNoOutput},
		{name: "global/binary_exits_nonzero", loc: LocationGlobal, codegraph: "dir", stub: stubFail, stubMode: 0o755, wantStarted: true},
	}

	// bogusCwd is a fresh, un-indexed directory, unrelated to any case's
	// own project dir — used as the CHILD PROCESS's actual OS-level cwd
	// for every LOCAL guard invocation below, so a mutation that made the
	// local guard consult $PWD/cwd instead of deriving its root from $0 is
	// caught (Family (f2)): with the real cwd wrong and lacking
	// .codegraph, only a correct $0-based derivation can still find the
	// project's own .codegraph directory.
	bogusCwd := t.TempDir()

	ran := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran++
			stubDir := t.TempDir()
			bin := filepath.Join(t.TempDir(), "codegraph")
			if tc.stub != "" {
				writeStub(t, bin, tc.stub, tc.stubMode)
			}
			project := newProject(t, tc.codegraph)
			guard := writeRenderedCodexGuard(t, tc.loc, bin, project)

			procCwd := bogusCwd
			if tc.loc == LocationGlobal {
				procCwd = project
			}
			var forcedEnv []string
			if tc.emptyPath {
				forcedEnv = []string{"STUB_DIR=" + stubDir, "PATH="}
			}
			stdout, stderr, exit, err := runCodexPreToolGuard(t, guard, procCwd, []string{"STUB_DIR=" + stubDir}, forcedEnv, preToolGuardEvent)
			if err != nil {
				t.Fatalf("run guard: %v", err)
			}
			if exit != 0 {
				t.Fatalf("guard exit = %d, want 0 (stdout %q, stderr %q)", exit, stdout, stderr)
			}
			started := exists(filepath.Join(stubDir, "started"))
			if started != tc.wantStarted {
				t.Fatalf("binary started = %v, want %v", started, tc.wantStarted)
			}
			if tc.check != nil {
				tc.check(t, stubDir, stdout, stderr)
			}
		})
	}
	if ran != len(cases) || ran < 11 {
		t.Fatalf("ran %d guard subtests, want 11", ran)
	}
}

// TestRenderCodexPreToolGuard pins renderCodexPreToolGuard's quoting and
// input validation over BOTH templates: the rendered ExecPath is one
// POSIX single-quoted word that survives a space and a single quote, and
// invalid ExecPaths are rejected at both locations.
func TestRenderCodexPreToolGuard(t *testing.T) {
	bothLocs := []Location{LocationLocal, LocationGlobal}

	t.Run("plain_path", func(t *testing.T) {
		for _, loc := range bothLocs {
			rendered, err := renderCodexPreToolGuard(loc, "/usr/local/bin/codegraph")
			if err != nil {
				t.Fatalf("(%s) render: %v", loc, err)
			}
			if !strings.Contains(rendered, "\ncodegraph_bin='/usr/local/bin/codegraph'\n") {
				t.Fatalf("(%s) rendered guard lacks the quoted ExecPath line:\n%s", loc, rendered)
			}
			if strings.Contains(rendered, preToolGuardExecPathToken) {
				t.Fatalf("(%s) rendered guard still carries the token", loc)
			}
			guard := filepath.Join(t.TempDir(), "guard.sh")
			writeStub(t, guard, rendered, 0o755)
			shSyntaxOK(t, guard)
		}
	})

	t.Run("quote_and_space_in_path", func(t *testing.T) {
		for _, loc := range bothLocs {
			stubDir := t.TempDir()
			bin := filepath.Join(t.TempDir(), "it's a dir", "codegraph")
			writeStub(t, bin, stubOK, 0o755)
			project := newProject(t, "dir")
			guard := writeRenderedCodexGuard(t, loc, bin, project)
			shSyntaxOK(t, guard)

			stdout, stderr, exit, err := runCodexPreToolGuard(t, guard, project, []string{"STUB_DIR=" + stubDir}, nil, preToolGuardEvent)
			if err != nil {
				t.Fatalf("(%s) run guard: %v", loc, err)
			}
			if exit != 0 {
				t.Fatalf("(%s) guard exit = %d, want 0 (stderr %q)", loc, exit, stderr)
			}
			if !exists(filepath.Join(stubDir, "started")) {
				t.Fatalf("(%s) binary at %q never started — the rendered path was mis-quoted (stdout %q, stderr %q)", loc, bin, stdout, stderr)
			}
		}
	})

	t.Run("relative_path_rejected", func(t *testing.T) {
		for _, loc := range bothLocs {
			if out, err := renderCodexPreToolGuard(loc, "codegraph"); err == nil {
				t.Fatalf("(%s) render(relative) = nil error, want an error; got:\n%s", loc, out)
			}
		}
	})

	t.Run("empty_path_rejected", func(t *testing.T) {
		for _, loc := range bothLocs {
			if out, err := renderCodexPreToolGuard(loc, ""); err == nil {
				t.Fatalf("(%s) render(empty) = nil error, want an error; got:\n%s", loc, out)
			}
		}
	})

	t.Run("template_token_exactly_once", func(t *testing.T) {
		local, err := claudeassets.CodexPreToolUseGuardLocalTemplate()
		if err != nil {
			t.Fatalf("CodexPreToolUseGuardLocalTemplate: %v", err)
		}
		if n := strings.Count(string(local), preToolGuardExecPathToken); n != 1 {
			t.Fatalf("local template carries the token %d times, want exactly 1", n)
		}
		global, err := claudeassets.CodexPreToolUseGuardGlobalTemplate()
		if err != nil {
			t.Fatalf("CodexPreToolUseGuardGlobalTemplate: %v", err)
		}
		if n := strings.Count(string(global), preToolGuardExecPathToken); n != 1 {
			t.Fatalf("global template carries the token %d times, want exactly 1", n)
		}
	})
}

// The sticky Keep/On/Off lifecycle of the Codex PreToolUse nudge (07-08,
// D-18/D-19/D-23). Unlike Claude's manifest-backed stickiness
// (preToolNudgeEvidenced), Codex's stickiness evidence is its own
// exact-identity hooks.json group directly (D-23) — Codex has no skill
// manifest concept for this opt-in.

// countHooksJSONFileAction returns the FileResult.Action recorded for path
// in res.Files, or "" when res.Files has no entry for it.
func countHooksJSONFileAction(res WriteResult, path string) FileAction {
	for _, f := range res.Files {
		if f.Path == path {
			return f.Action
		}
	}
	return ""
}

// TestCodexPreToolNudge_OnWritesAndNotesTrust (D-19): both local and global
// On installs write the guard and register codegraph's own PreToolUse
// group, and each carries exactly one Note mentioning /hooks.
func TestCodexPreToolNudge_OnWritesAndNotesTrust(t *testing.T) {
	for _, loc := range []Location{LocationLocal, LocationGlobal} {
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())

			res := codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			if len(res.Errors) != 0 {
				t.Fatalf("Install errors: %v", res.Errors)
			}
			guard, err := codexPreToolGuardPath(loc)
			if err != nil {
				t.Fatalf("codexPreToolGuardPath(%s): %v", loc, err)
			}
			if _, statErr := os.Stat(guard); statErr != nil {
				t.Fatalf("guard not written: %v", statErr)
			}
			hooksPath, err := codexHooksJSONPath(loc)
			if err != nil {
				t.Fatalf("codexHooksJSONPath(%s): %v", loc, err)
			}
			_, ownCommands, err := codexPreToolUseBlocks(loc)
			if err != nil {
				t.Fatalf("codexPreToolUseBlocks(%s): %v", loc, err)
			}
			has, herr := hasOwnHookBlock(hooksPath, "PreToolUse", ownCommands)
			if herr != nil || !has {
				t.Fatalf("hooks.json has no own PreToolUse group after On install (has=%v err=%v)", has, herr)
			}
			n := 0
			for _, note := range res.Notes {
				if strings.Contains(note, "/hooks") {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("Notes mentioning /hooks = %d, want 1: %#v", n, res.Notes)
			}
		})
	}
}

// TestCodexPreToolNudge_KeepRefreshesWhenOwnGroupPresent (D-23): On with
// ExecPath A, then Keep with ExecPath B refreshes the guard for the moved
// binary, reports hooks.json unchanged (the definition itself did not
// change), and prints no trust Note (nothing new to trust).
func TestCodexPreToolNudge_KeepRefreshesWhenOwnGroupPresent(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())

	const execA = "/opt/a/codegraph"
	const execB = "/opt/b/codegraph"

	if res := (codexTarget{}).Install(LocationLocal, InstallOptions{ExecPath: execA, PreToolNudge: PreToolNudgeOn}); len(res.Errors) != 0 {
		t.Fatalf("On Install errors: %v", res.Errors)
	}
	guard, err := codexPreToolGuardPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexPreToolGuardPath: %v", err)
	}
	hooksPath, err := codexHooksJSONPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexHooksJSONPath: %v", err)
	}
	beforeHooks := readFile(t, hooksPath)

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: execB, PreToolNudge: PreToolNudgeKeep})
	if len(res.Errors) != 0 {
		t.Fatalf("Keep Install errors: %v", res.Errors)
	}
	content := readFile(t, guard)
	if !strings.Contains(content, "\ncodegraph_bin='"+execB+"'\n") {
		t.Fatalf("Keep did not re-render the guard for the moved binary:\n%s", content)
	}
	if got := countHooksJSONFileAction(res, hooksPath); got != ActionUnchanged {
		t.Fatalf("hooks.json FileResult after Keep = %q, want %q", got, ActionUnchanged)
	}
	if got := readFile(t, hooksPath); got != beforeHooks {
		t.Fatalf("hooks.json bytes changed after Keep, want byte-identical:\nbefore=%q\nafter=%q", beforeHooks, got)
	}
	for _, note := range res.Notes {
		if strings.Contains(note, "/hooks") {
			t.Fatalf("Keep with an unchanged definition printed a trust note: %#v", res.Notes)
		}
	}
}

// TestCodexPreToolNudge_KeepNoopWhenNotOptedIn: Keep on a fresh (never
// opted-in) location writes nothing.
func TestCodexPreToolNudge_KeepNoopWhenNotOptedIn(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())

	guard, err := codexPreToolGuardPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexPreToolGuardPath: %v", err)
	}
	hooksPath, err := codexHooksJSONPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexHooksJSONPath: %v", err)
	}

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeKeep})
	if len(res.Errors) != 0 {
		t.Fatalf("Keep Install errors: %v", res.Errors)
	}
	if _, statErr := os.Lstat(guard); !os.IsNotExist(statErr) {
		t.Fatalf("a fresh Keep install wrote %s (Lstat err %v)", guard, statErr)
	}
	if _, statErr := os.Lstat(hooksPath); !os.IsNotExist(statErr) {
		t.Fatalf("a fresh Keep install wrote %s (Lstat err %v)", hooksPath, statErr)
	}
}

// TestCodexPreToolNudge_KeepWithMalformedHooksJSONTouchesNothing: a
// malformed hooks.json with Keep writes nothing, reports no error, and
// leaves the file byte-identical (hasOwnHookBlock's read failure is
// treated as "cannot tell", never surfaced as an Install error).
func TestCodexPreToolNudge_KeepWithMalformedHooksJSONTouchesNothing(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())

	hooksPath, err := codexHooksJSONPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexHooksJSONPath: %v", err)
	}
	writeFile(t, hooksPath, "{not json")
	before := readFile(t, hooksPath)
	guard, err := codexPreToolGuardPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexPreToolGuardPath: %v", err)
	}

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeKeep})
	if len(res.Errors) != 0 {
		t.Fatalf("Keep with a malformed hooks.json returned errors, want none: %v", res.Errors)
	}
	if got := readFile(t, hooksPath); got != before {
		t.Fatalf("Keep with a malformed hooks.json changed its bytes:\nbefore=%q\nafter=%q", before, got)
	}
	if _, statErr := os.Lstat(guard); !os.IsNotExist(statErr) {
		t.Fatalf("Keep with a malformed hooks.json wrote the guard (Lstat err %v)", statErr)
	}
}

// TestCodexPreToolNudge_OffRemovesAndForgets: On then Off removes the
// guard and codegraph's own hooks.json group; a later Keep adds nothing
// back (D-18's opt-in stays explicit — Off is not merely a pause).
func TestCodexPreToolNudge_OffRemovesAndForgets(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())

	guard, err := codexPreToolGuardPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexPreToolGuardPath: %v", err)
	}
	hooksPath, err := codexHooksJSONPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexHooksJSONPath: %v", err)
	}

	if res := (codexTarget{}).Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn}); len(res.Errors) != 0 {
		t.Fatalf("On Install errors: %v", res.Errors)
	}
	if _, statErr := os.Stat(guard); statErr != nil {
		t.Fatalf("precondition: guard not written: %v", statErr)
	}

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOff})
	if len(res.Errors) != 0 {
		t.Fatalf("Off Install errors: %v", res.Errors)
	}
	if _, statErr := os.Lstat(guard); !os.IsNotExist(statErr) {
		t.Fatalf("Off left the guard (Lstat err %v)", statErr)
	}
	if _, statErr := os.Lstat(hooksPath); !os.IsNotExist(statErr) {
		t.Fatalf("Off left hooks.json (Lstat err %v)", statErr)
	}

	res = codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeKeep})
	if len(res.Errors) != 0 {
		t.Fatalf("Keep after Off errors: %v", res.Errors)
	}
	if _, statErr := os.Lstat(guard); !os.IsNotExist(statErr) {
		t.Fatalf("Keep after Off re-added the guard (Lstat err %v)", statErr)
	}
	if _, statErr := os.Lstat(hooksPath); !os.IsNotExist(statErr) {
		t.Fatalf("Keep after Off re-added hooks.json (Lstat err %v)", statErr)
	}
}

// TestCodexPreToolNudge_SkippedWhenHooksDisabled (D-18): an On install is
// skipped, writing neither the guard nor hooks.json and carrying exactly
// one Note, whenever the governing config.toml explicitly disables Codex
// hooks — recognized via any of tomlBoolSetting's three forms, at either
// local or (falling back) global scope, under either the current or the
// deprecated key name. A LOCAL true overrides a GLOBAL false (local wins),
// and an unqualified true install still writes.
func TestCodexPreToolNudge_SkippedWhenHooksDisabled(t *testing.T) {
	cases := []struct {
		name         string
		localConfig  string
		globalConfig string
		wantWrites   bool
	}{
		{name: "features_hooks_false_local", localConfig: "[features]\nhooks = false\n"},
		{name: "global_false_applies_to_local", globalConfig: "[features]\nhooks = false\n"},
		{name: "codex_hooks_false_deprecated", localConfig: "[features]\ncodex_hooks = false\n"},
		{name: "dotted_root_key", localConfig: "features.hooks = false\n"},
		{name: "inline_table", localConfig: "features = { hooks = false }\n"},
		{name: "local_true_overrides_global_false", localConfig: "[features]\nhooks = true\n", globalConfig: "[features]\nhooks = false\n", wantWrites: true},
		{name: "hooks_true_writes", localConfig: "[features]\nhooks = true\n", wantWrites: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := fakeHome(t)
			t.Chdir(t.TempDir())

			if tc.localConfig != "" {
				localPath, err := codexConfigPath(LocationLocal)
				if err != nil {
					t.Fatalf("codexConfigPath(local): %v", err)
				}
				writeFile(t, localPath, tc.localConfig)
			}
			if tc.globalConfig != "" {
				writeFile(t, filepath.Join(home, ".codex", "config.toml"), tc.globalConfig)
			}

			guard, err := codexPreToolGuardPath(LocationLocal)
			if err != nil {
				t.Fatalf("codexPreToolGuardPath: %v", err)
			}
			hooksPath, err := codexHooksJSONPath(LocationLocal)
			if err != nil {
				t.Fatalf("codexHooksJSONPath: %v", err)
			}

			res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			if len(res.Errors) != 0 {
				t.Fatalf("Install errors: %v", res.Errors)
			}

			_, guardErr := os.Lstat(guard)
			_, hooksErr := os.Lstat(hooksPath)
			guardExists := guardErr == nil
			hooksExists := hooksErr == nil

			if tc.wantWrites {
				if !guardExists || !hooksExists {
					t.Fatalf("expected writes when hooks are enabled, guardExists=%v hooksExists=%v (guardErr=%v hooksErr=%v)", guardExists, hooksExists, guardErr, hooksErr)
				}
				return
			}

			if guardExists || hooksExists {
				t.Fatalf("expected no writes when hooks are disabled, guardExists=%v hooksExists=%v", guardExists, hooksExists)
			}
			// A local install ALSO carries codexTrustNote's unrelated D-10
			// "loads this project's MCP server only once trusted" Note —
			// count only Notes naming the disabled setting, not len(Notes).
			n := 0
			for _, note := range res.Notes {
				if strings.Contains(note, "hooks are explicitly disabled") {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("Notes naming the disabled setting = %d, want exactly 1: %#v", n, res.Notes)
			}
		})
	}
}

// TestCodexPreToolNudge_ReinstallIsIdempotent: On twice reports every file
// unchanged on the second run.
func TestCodexPreToolNudge_ReinstallIsIdempotent(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())

	if res := (codexTarget{}).Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn}); len(res.Errors) != 0 {
		t.Fatalf("first On Install errors: %v", res.Errors)
	}

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
	if len(res.Errors) != 0 {
		t.Fatalf("second On Install errors: %v", res.Errors)
	}
	if len(res.Files) == 0 {
		t.Fatalf("second On install reported no files")
	}
	for _, f := range res.Files {
		if f.Action != ActionUnchanged {
			t.Errorf("second On install: %s = %q, want %q", f.Path, f.Action, ActionUnchanged)
		}
	}
}

// TestCodexPreToolNudge_HandEditedOwnGroupDuplicates: ownership is the
// exact command string, never the matcher (242ec0a). Hand-editing the
// installed group's handler command makes it no longer codegraph's own —
// the next On install leaves it byte-identical and appends a fresh owned
// group beside it, rather than overwriting it.
func TestCodexPreToolNudge_HandEditedOwnGroupDuplicates(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())

	if res := (codexTarget{}).Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn}); len(res.Errors) != 0 {
		t.Fatalf("On Install errors: %v", res.Errors)
	}
	hooksPath, err := codexHooksJSONPath(LocationLocal)
	if err != nil {
		t.Fatalf("codexHooksJSONPath: %v", err)
	}
	ownCommand, err := codexPreToolHookCommand(LocationLocal)
	if err != nil {
		t.Fatalf("codexPreToolHookCommand: %v", err)
	}

	decoded := readJSONHooksFile(t, hooksPath)
	hooks, _ := decoded["hooks"].(map[string]any)
	groups, _ := hooks["PreToolUse"].([]any)
	if len(groups) != 1 {
		t.Fatalf("after the first On install PreToolUse has %d groups, want 1: %#v", len(groups), groups)
	}
	group, _ := groups[0].(map[string]any)
	handlers, _ := group["hooks"].([]any)
	handler, _ := handlers[0].(map[string]any)
	if handler["command"] != ownCommand {
		t.Fatalf("own group's handler command = %#v, want %q", handler["command"], ownCommand)
	}
	handler["command"] = ownCommand + " --edited"
	editedGroup, err := normalizeJSON(group)
	if err != nil {
		t.Fatalf("normalize edited group: %v", err)
	}
	out, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	writeFile(t, hooksPath, string(out)+"\n")

	if res := (codexTarget{}).Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn}); len(res.Errors) != 0 {
		t.Fatalf("second On Install errors: %v", res.Errors)
	}

	after := readJSONHooksFile(t, hooksPath)
	afterHooks, _ := after["hooks"].(map[string]any)
	afterGroups, _ := afterHooks["PreToolUse"].([]any)
	if len(afterGroups) != 2 {
		t.Fatalf("PreToolUse has %d groups, want 2 (the hand-edited one + 1 fresh owned): %#v", len(afterGroups), afterGroups)
	}
	sawEdited := 0
	for _, g := range afterGroups {
		if jsonDeepEqual(g, editedGroup) {
			sawEdited++
		}
	}
	if sawEdited != 1 {
		t.Fatalf("the hand-edited group appears %d times byte-identical, want 1: %#v", sawEdited, afterGroups)
	}
	_, ownCommands, err := codexPreToolUseBlocks(LocationLocal)
	if err != nil {
		t.Fatalf("codexPreToolUseBlocks: %v", err)
	}
	has, herr := hasOwnHookBlock(hooksPath, "PreToolUse", ownCommands)
	if herr != nil || !has {
		t.Fatalf("hasOwnHookBlock after reinstall = (%v, %v), want (true, nil)", has, herr)
	}
}

// TestCodexNotesNeverAdviseTrustBypass (D-19, T-07-24): every Note produced
// by a Codex install — across On/Keep/Off, both scopes, and the
// hooks-disabled skip — never advises bypassing Codex's hook trust review.
// The forbidden token is built by concatenation so this test's own source
// never matches it (grep-proofing the negative assertion itself).
func TestCodexNotesNeverAdviseTrustBypass(t *testing.T) {
	forbidden := "--dangerously-" + "bypass-hook-trust"

	check := func(t *testing.T, notes []string) {
		t.Helper()
		for _, note := range notes {
			if strings.Contains(note, forbidden) {
				t.Fatalf("a Codex Note advises trust bypass: %q", note)
			}
		}
	}

	for _, loc := range []Location{LocationLocal, LocationGlobal} {
		t.Run(string(loc)+"/on", func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())
			res := codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			check(t, res.Notes)
		})
		t.Run(string(loc)+"/keep_after_on", func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())
			codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			res := codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeKeep})
			check(t, res.Notes)
		})
		t.Run(string(loc)+"/off", func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())
			codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			res := codexTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOff})
			check(t, res.Notes)
		})
	}
	t.Run("hooks_disabled_skip", func(t *testing.T) {
		fakeHome(t)
		t.Chdir(t.TempDir())
		localPath, err := codexConfigPath(LocationLocal)
		if err != nil {
			t.Fatalf("codexConfigPath: %v", err)
		}
		writeFile(t, localPath, "[features]\nhooks = false\n")
		res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		check(t, res.Notes)
	})
}
