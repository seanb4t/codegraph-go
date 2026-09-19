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

// preToolFragmentCommandLiteral is the local-scope registered command,
// hand-typed as an independent oracle for claudePreToolFragmentCommand.
const preToolFragmentCommandLiteral = "${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"

// readSettingsHooks decodes settingsPath and returns its "hooks" object.
func readSettingsHooks(t *testing.T, settingsPath string) map[string]any {
	t.Helper()
	var settings map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &settings); err != nil {
		t.Fatalf("decode %s: %v", settingsPath, err)
	}
	hooks, _ := settings["hooks"].(map[string]any)
	return hooks
}

// assertPreToolUseBlocks checks hooks.PreToolUse is exactly the four D-02
// blocks, every handler registering wantCommand with timeout 5 and no
// statusMessage (D-12).
func assertPreToolUseBlocks(t *testing.T, hooks map[string]any, wantCommand string) {
	t.Helper()
	blocks, ok := hooks["PreToolUse"].([]any)
	if !ok {
		t.Fatalf("hooks.PreToolUse missing or not an array: %#v", hooks["PreToolUse"])
	}
	wantMatchers := []string{"Bash", "Grep", "Glob", "Read"}
	if len(blocks) != len(wantMatchers) {
		t.Fatalf("hooks.PreToolUse has %d blocks, want %d: %#v", len(blocks), len(wantMatchers), blocks)
	}
	for i, b := range blocks {
		block, ok := b.(map[string]any)
		if !ok {
			t.Fatalf("block %d is not an object: %#v", i, b)
		}
		if block["matcher"] != wantMatchers[i] {
			t.Fatalf("block %d matcher = %#v, want %q", i, block["matcher"], wantMatchers[i])
		}
		handlers, _ := block["hooks"].([]any)
		wantIfs := []string{""}
		if wantMatchers[i] == "Bash" {
			wantIfs = []string{"Bash(grep *)", "Bash(rg *)", "Bash(find *)"}
		}
		if len(handlers) != len(wantIfs) {
			t.Fatalf("block %s has %d handlers, want %d: %#v", wantMatchers[i], len(handlers), len(wantIfs), handlers)
		}
		for j, h := range handlers {
			handler, ok := h.(map[string]any)
			if !ok {
				t.Fatalf("block %s handler %d is not an object: %#v", wantMatchers[i], j, h)
			}
			if handler["type"] != "command" {
				t.Errorf("block %s handler %d type = %#v, want command", wantMatchers[i], j, handler["type"])
			}
			if handler["command"] != wantCommand {
				t.Errorf("block %s handler %d command = %#v, want %q", wantMatchers[i], j, handler["command"], wantCommand)
			}
			if handler["timeout"] != float64(5) {
				t.Errorf("block %s handler %d timeout = %#v, want 5", wantMatchers[i], j, handler["timeout"])
			}
			if _, has := handler["statusMessage"]; has {
				t.Errorf("block %s handler %d carries statusMessage (D-12)", wantMatchers[i], j)
			}
			gotIf, hasIf := handler["if"]
			if wantIfs[j] == "" {
				if hasIf {
					t.Errorf("block %s handler %d has if = %#v, want none", wantMatchers[i], j, gotIf)
				}
			} else if gotIf != wantIfs[j] {
				t.Errorf("block %s handler %d if = %#v, want %q", wantMatchers[i], j, gotIf, wantIfs[j])
			}
		}
	}
}

func TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks(t *testing.T) {
	t.Run("local", func(t *testing.T) {
		fakeHome(t)
		t.Chdir(t.TempDir())

		res := claudeTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		if len(res.Errors) != 0 {
			t.Fatalf("Install errors: %v", res.Errors)
		}

		guard := filepath.Join(".claude", "hooks", "pretooluse-nudge.sh")
		info, err := os.Stat(guard)
		if err != nil {
			t.Fatalf("guard not written: %v", err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("guard mode %v is not executable", info.Mode())
		}
		content := readFile(t, guard)
		if !strings.HasPrefix(content, "#!/bin/sh\n") {
			t.Fatalf("guard does not start with #!/bin/sh:\n%s", content)
		}
		if !strings.Contains(content, "\ncodegraph_bin='/usr/local/bin/codegraph'\n") {
			t.Fatalf("guard lacks the rendered ExecPath line:\n%s", content)
		}
		if strings.Contains(content, "@codegraph-exec-path@") {
			t.Fatalf("guard still carries the unrendered token:\n%s", content)
		}

		hooks := readSettingsHooks(t, filepath.Join(".claude", "settings.json"))
		assertPreToolUseBlocks(t, hooks, preToolFragmentCommandLiteral)
		if _, ok := hooks["SessionStart"]; !ok {
			t.Fatalf("hooks.SessionStart missing after an opt-in install")
		}
	})

	t.Run("global", func(t *testing.T) {
		home := fakeHome(t)
		t.Chdir(t.TempDir())

		res := claudeTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		if len(res.Errors) != 0 {
			t.Fatalf("Install errors: %v", res.Errors)
		}
		guard := filepath.Join(home, ".claude", "hooks", "pretooluse-nudge.sh")
		if _, err := os.Stat(guard); err != nil {
			t.Fatalf("global guard not written: %v", err)
		}
		hooks := readSettingsHooks(t, filepath.Join(home, ".claude", "settings.json"))
		assertPreToolUseBlocks(t, hooks, guard)
		if _, ok := hooks["SessionStart"]; !ok {
			t.Fatalf("hooks.SessionStart missing after an opt-in install")
		}
	})
}

func TestClaude_Install_DefaultWritesNoPreToolUse(t *testing.T) {
	for _, loc := range []Location{LocationLocal, LocationGlobal} {
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())

			guard, err := claudePreToolGuardPath(loc)
			if err != nil {
				t.Fatalf("claudePreToolGuardPath(%s): %v", loc, err)
			}
			settingsPath, err := claudeSettingsPath(loc)
			if err != nil {
				t.Fatalf("claudeSettingsPath(%s): %v", loc, err)
			}

			res := claudeTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
			if len(res.Errors) != 0 {
				t.Fatalf("Install errors: %v", res.Errors)
			}
			if _, err := os.Lstat(guard); !os.IsNotExist(err) {
				t.Fatalf("default install wrote %s (Lstat err %v); the nudge is opt-in (D-09)", guard, err)
			}
			hooks := readSettingsHooks(t, settingsPath)
			if _, ok := hooks["SessionStart"]; !ok {
				t.Fatalf("hooks.SessionStart missing — the absence check below would be vacuous")
			}
			if _, ok := hooks["PreToolUse"]; ok {
				t.Fatalf("default install wrote hooks.PreToolUse; the nudge is opt-in (D-09)")
			}

			// Positive control: an opt-in install into the same scope makes
			// both the guard and the key appear, so the absence checks
			// above can see presence.
			res = claudeTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			if len(res.Errors) != 0 {
				t.Fatalf("opt-in Install errors: %v", res.Errors)
			}
			if _, err := os.Lstat(guard); err != nil {
				t.Fatalf("positive control: opt-in install did not write %s: %v", guard, err)
			}
			if _, ok := readSettingsHooks(t, settingsPath)["PreToolUse"]; !ok {
				t.Fatalf("positive control: opt-in install did not write hooks.PreToolUse")
			}
		})
	}
}

// preToolGuardEvent is a representative PreToolUse Grep event fed to the
// guard; the guard must hand it to the binary byte-identical.
const preToolGuardEvent = `{"session_id":"guard-1","hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"Alpha"}}`

// Stub binaries standing in for codegraph. Each records that it started
// under $STUB_DIR, so a test can assert positively that the guard did — or
// did not — start a process.
const (
	stubOK = "#!/bin/sh\ncat > \"$STUB_DIR/stdin\"\n: > \"$STUB_DIR/started\"\nprintf 'STUB-OUT\\n'\nexit 0\n"
	// stubFail exits non-zero; the guard must still exit 0.
	stubFail = "#!/bin/sh\n: > \"$STUB_DIR/started\"\nexit 3\n"
	// stubCrash dies on SIGSEGV; the guard must still exit 0.
	stubCrash = "#!/bin/sh\n: > \"$STUB_DIR/started\"\nkill -SEGV $$\n"
)

// runPreToolGuard runs guardPath with stdin, in runSessionNudge's shape:
// any inherited CLAUDE_PROJECT_DIR and CLAUDE_CODE_SESSION_ID are stripped
// (and any key extraEnv sets), then dir is passed via CLAUDE_PROJECT_DIR
// when useEnv, else as the working directory. A non-zero exit is returned
// as exit, not err; err is for harness failures only. It never calls
// t.Fatalf itself.
func runPreToolGuard(t *testing.T, guardPath, dir string, useEnv bool, extraEnv []string, stdin string) (stdout, stderr string, exit int, err error) {
	t.Helper()

	drop := map[string]bool{"CLAUDE_PROJECT_DIR": true, "CLAUDE_CODE_SESSION_ID": true}
	for _, kv := range extraEnv {
		if k, _, ok := strings.Cut(kv, "="); ok {
			drop[k] = true
		}
	}
	var env []string
	for _, kv := range os.Environ() {
		if k, _, ok := strings.Cut(kv, "="); ok && drop[k] {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, extraEnv...)

	cmd := exec.Command(guardPath)
	if useEnv {
		env = append(env, "CLAUDE_PROJECT_DIR="+dir)
	} else {
		cmd.Dir = dir
	}
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
		return stdout, stderr, 0, fmt.Errorf("runPreToolGuard(%s): %w", guardPath, runErr)
	}
	return stdout, stderr, exitErr.ExitCode(), nil
}

// writeStub writes a stub binary script at path with mode.
func writeStub(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for stub %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write stub %s: %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod stub %s: %v", path, err)
	}
}

// writeRenderedGuard renders the REAL embedded guard template for binPath
// and writes it executable into a fresh temp dir.
func writeRenderedGuard(t *testing.T, binPath string) string {
	t.Helper()
	rendered, err := renderPreToolGuard(binPath)
	if err != nil {
		t.Fatalf("renderPreToolGuard(%q): %v", binPath, err)
	}
	guard := filepath.Join(t.TempDir(), "pretooluse-nudge.sh")
	writeStub(t, guard, rendered, 0o755)
	return guard
}

// newProject returns a fresh project dir: indexed (a .codegraph/ dir), not
// indexed, or with .codegraph as a regular file.
func newProject(t *testing.T, codegraph string) string {
	t.Helper()
	dir := t.TempDir()
	switch codegraph {
	case "dir":
		if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
			t.Fatalf("mkdir .codegraph: %v", err)
		}
	case "file":
		writeFile(t, filepath.Join(dir, ".codegraph"), "not a directory\n")
	}
	return dir
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// TestPreToolUseGuard is the guard-level D-16 suite over the real embedded
// template, rendered with stub binaries: the guard exits 0 on every path,
// starts no process in an un-indexed repo (D-04), and passes stdin and
// stdout through untouched (D-01 guard jobs a-c).
func TestPreToolUseGuard(t *testing.T) {
	cases := []struct {
		name        string
		codegraph   string // "dir", "file" or "" (absent)
		stub        string // stub content; "" = binary missing
		stubMode    os.FileMode
		useEnv      bool
		wantStarted bool
		check       func(t *testing.T, stubDir, stdout, stderr string)
	}{
		{name: "indexed_binary_ok", codegraph: "dir", stub: stubOK, stubMode: 0o755, useEnv: true, wantStarted: true,
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
		{name: "not_indexed", codegraph: "", stub: stubOK, stubMode: 0o755, useEnv: true, wantStarted: false, check: wantNoOutput},
		{name: "codegraph_is_file", codegraph: "file", stub: stubOK, stubMode: 0o755, useEnv: true, wantStarted: false, check: wantNoOutput},
		{name: "binary_missing", codegraph: "dir", stub: "", useEnv: true, wantStarted: false, check: wantNoOutput},
		{name: "binary_not_executable", codegraph: "dir", stub: stubOK, stubMode: 0o644, useEnv: true, wantStarted: false,
			check: func(t *testing.T, _, stdout, _ string) {
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
			}},
		{name: "binary_exits_nonzero", codegraph: "dir", stub: stubFail, stubMode: 0o755, useEnv: true, wantStarted: true},
		{name: "binary_crashes", codegraph: "dir", stub: stubCrash, stubMode: 0o755, useEnv: true, wantStarted: true},
		{name: "project_dir_unset_indexed", codegraph: "dir", stub: stubOK, stubMode: 0o755, useEnv: false, wantStarted: true},
		{name: "project_dir_unset_not_indexed", codegraph: "", stub: stubOK, stubMode: 0o755, useEnv: false, wantStarted: false, check: wantNoOutput},
	}

	ran := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran++
			stubDir := t.TempDir()
			bin := filepath.Join(t.TempDir(), "codegraph")
			if tc.stub != "" {
				writeStub(t, bin, tc.stub, tc.stubMode)
			}
			guard := writeRenderedGuard(t, bin)
			project := newProject(t, tc.codegraph)

			stdout, stderr, exit, err := runPreToolGuard(t, guard, project, tc.useEnv, []string{"STUB_DIR=" + stubDir}, preToolGuardEvent)
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
	if ran < len(cases) || ran < 9 {
		t.Fatalf("ran %d guard subtests, want at least 9", ran)
	}
}

func wantNoOutput(t *testing.T, _, stdout, stderr string) {
	t.Helper()
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
	}
}

// shSyntaxOK runs `sh -n` over path.
func shSyntaxOK(t *testing.T, path string) {
	t.Helper()
	if out, err := exec.Command("/bin/sh", "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("sh -n %s: %v\n%s", path, err, out)
	}
}

// TestRenderPreToolGuard pins renderPreToolGuard's quoting and input
// validation (T-06-01): the rendered ExecPath is one POSIX single-quoted
// word that survives a space and a single quote.
func TestRenderPreToolGuard(t *testing.T) {
	t.Run("plain_path", func(t *testing.T) {
		rendered, err := renderPreToolGuard("/usr/local/bin/codegraph")
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if !strings.Contains(rendered, "\ncodegraph_bin='/usr/local/bin/codegraph'\n") {
			t.Fatalf("rendered guard lacks the quoted ExecPath line:\n%s", rendered)
		}
		if strings.Contains(rendered, preToolGuardExecPathToken) {
			t.Fatalf("rendered guard still carries the token")
		}
		guard := filepath.Join(t.TempDir(), "guard.sh")
		writeStub(t, guard, rendered, 0o755)
		shSyntaxOK(t, guard)
	})

	t.Run("quote_and_space_in_path", func(t *testing.T) {
		stubDir := t.TempDir()
		bin := filepath.Join(t.TempDir(), "it's a dir", "codegraph")
		writeStub(t, bin, stubOK, 0o755)
		guard := writeRenderedGuard(t, bin)
		shSyntaxOK(t, guard)

		stdout, stderr, exit, err := runPreToolGuard(t, guard, newProject(t, "dir"), true, []string{"STUB_DIR=" + stubDir}, preToolGuardEvent)
		if err != nil {
			t.Fatalf("run guard: %v", err)
		}
		if exit != 0 {
			t.Fatalf("guard exit = %d, want 0 (stderr %q)", exit, stderr)
		}
		if !exists(filepath.Join(stubDir, "started")) {
			t.Fatalf("binary at %q never started — the rendered path was mis-quoted (stdout %q, stderr %q)", bin, stdout, stderr)
		}
	})

	t.Run("relative_path_rejected", func(t *testing.T) {
		if out, err := renderPreToolGuard("codegraph"); err == nil {
			t.Fatalf("render(relative) = nil error, want an error; got:\n%s", out)
		}
	})

	t.Run("empty_path_rejected", func(t *testing.T) {
		if out, err := renderPreToolGuard(""); err == nil {
			t.Fatalf("render(empty) = nil error, want an error; got:\n%s", out)
		}
	})

	t.Run("template_token_exactly_once", func(t *testing.T) {
		tmpl, err := claudeassets.PreToolUseGuardTemplate()
		if err != nil {
			t.Fatalf("PreToolUseGuardTemplate: %v", err)
		}
		if n := strings.Count(string(tmpl), preToolGuardExecPathToken); n != 1 {
			t.Fatalf("embedded guard template carries the token %d times, want exactly 1", n)
		}
	})
}

// TestPreToolUseGuardSourceFallsBackToPATH runs the checked-in, UNRENDERED
// guard (this repository's dogfood copy): only it falls back to the
// codegraph on PATH (D-01b), and without one it still exits 0 silently.
func TestPreToolUseGuardSourceFallsBackToPATH(t *testing.T) {
	source, err := filepath.Abs(filepath.Join("..", "..", ".claude", "hooks", "pretooluse-nudge.sh"))
	if err != nil {
		t.Fatalf("resolve guard source: %v", err)
	}

	t.Run("stub_on_path", func(t *testing.T) {
		stubDir := t.TempDir()
		binDir := t.TempDir()
		writeStub(t, filepath.Join(binDir, "codegraph"), stubOK, 0o755)

		_, stderr, exit, err := runPreToolGuard(t, source, newProject(t, "dir"), true,
			[]string{"STUB_DIR=" + stubDir, "PATH=" + binDir + ":/usr/bin:/bin"}, preToolGuardEvent)
		if err != nil {
			t.Fatalf("run guard: %v", err)
		}
		if exit != 0 {
			t.Fatalf("guard exit = %d, want 0 (stderr %q)", exit, stderr)
		}
		if !exists(filepath.Join(stubDir, "started")) {
			t.Fatalf("the unrendered guard did not run the codegraph found on PATH")
		}
	})

	t.Run("no_codegraph_on_path", func(t *testing.T) {
		const path = "/usr/bin:/bin"
		t.Setenv("PATH", path)
		if found, err := exec.LookPath("codegraph"); err == nil {
			t.Fatalf("precondition: codegraph found on PATH=%s at %s; this case would not test the missing-binary fallback", path, found)
		}

		stdout, stderr, exit, err := runPreToolGuard(t, source, newProject(t, "dir"), true, []string{"PATH=" + path}, preToolGuardEvent)
		if err != nil {
			t.Fatalf("run guard: %v", err)
		}
		if exit != 0 {
			t.Fatalf("guard exit = %d, want 0 (stderr %q)", exit, stderr)
		}
		if stdout != "" {
			t.Fatalf("stdout = %q, want empty", stdout)
		}
	})
}
