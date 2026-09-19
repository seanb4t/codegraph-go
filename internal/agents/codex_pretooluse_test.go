package agents

import (
	"encoding/json"
	"os"
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
