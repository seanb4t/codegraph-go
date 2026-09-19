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

// runCodexPreToolGuard runs guardPath with stdin and the process cwd set
// to cwd. pwd is the value the $PWD ENVIRONMENT VARIABLE is explicitly set
// to — deliberately a SEPARATE parameter from cwd, never left to whatever
// the test process itself inherited (the actual shell running `go test`
// leaves its own PWD in os.Environ(), which would otherwise leak in). The
// two are the same value only for the global guard's own tests, which
// legitimately depend on $PWD (D-22); every local guard test call passes
// a DELIBERATELY WRONG pwd (bogusPWD) distinct from cwd and lacking
// .codegraph, so a mutation that made the local guard consult $PWD
// instead of deriving its root from $0 would be caught immediately by
// indexed_binary_ok turning silent (Family (f2), 07-MUTATION-LOG.md) —
// this is the negative control that makes root_from_own_path_with_empty_path
// mean something beyond "still works," not just an accidentally-correct
// PWD along for the ride. forcedEnv, when non-nil, REPLACES the
// environment entirely (used by the empty-PATH subtest) — pwd is still
// appended afterward either way.
func runCodexPreToolGuard(t *testing.T, guardPath, cwd, pwd string, extraEnv []string, forcedEnv []string, stdin string) (stdout, stderr string, exit int, err error) {
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
	env = append(env, "PWD="+pwd)

	cmd := exec.Command(guardPath)
	cmd.Dir = cwd
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

	// bogusPWD is a fresh, un-indexed directory distinct from every case's
	// own project dir — passed as $PWD for every LOCAL guard invocation
	// below so a mutation that made the local guard consult $PWD instead
	// of deriving its root from $0 is caught (Family (f2)), rather than
	// accidentally passing because $PWD happened to equal the right
	// directory anyway.
	bogusPWD := t.TempDir()

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

			pwd := bogusPWD
			if tc.loc == LocationGlobal {
				pwd = project
			}
			var forcedEnv []string
			if tc.emptyPath {
				forcedEnv = []string{"STUB_DIR=" + stubDir, "PATH="}
			}
			stdout, stderr, exit, err := runCodexPreToolGuard(t, guard, project, pwd, []string{"STUB_DIR=" + stubDir}, forcedEnv, preToolGuardEvent)
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

			stdout, stderr, exit, err := runCodexPreToolGuard(t, guard, project, project, []string{"STUB_DIR=" + stubDir}, nil, preToolGuardEvent)
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
