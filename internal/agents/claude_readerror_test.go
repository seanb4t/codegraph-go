package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// malformedSettingsJSON is deliberately truncated — unambiguously not a
// decodable JSON object, so encoding/json fails outright rather than
// succeeding into some unexpected shape (Plan 02 Task 2 <action>).
const malformedSettingsJSON = `{
  "hooks": {
    "SessionStart": [
`

// TestClaude_SettingsReadFailureNeverDestroysContent proves roadmap
// success criterion 4 across the full {install, uninstall} x
// {read-error, malformed} matrix for claudeSettingsPath: every cell must
// surface the read failure through WriteResult.Errors AND leave the
// file's raw bytes exactly as found. A surfaced error with a destroyed
// file is the v1.0 Phase 5 data-loss shape; an intact file with a
// swallowed error is the CR-01 swallowed-error shape — both are
// failures, so every cell asserts both halves.
func TestClaude_SettingsReadFailureNeverDestroysContent(t *testing.T) {
	install := func(c claudeTarget, loc Location) WriteResult {
		return c.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	}
	uninstall := func(c claudeTarget, loc Location) WriteResult {
		return c.Uninstall(loc)
	}

	cases := []struct {
		name          string
		op            func(c claudeTarget, loc Location) WriteResult
		readError     bool // true: chmod 0o000 instead of writing malformed JSON
		assertNotGone bool // uninstall cells additionally assert the file was not deleted
	}{
		{name: "install x malformed", op: install},
		{name: "install x read-error", op: install, readError: true},
		{name: "uninstall x malformed", op: uninstall, assertNotGone: true},
		{name: "uninstall x read-error", op: uninstall, readError: true, assertNotGone: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.readError && os.Geteuid() == 0 {
				t.Skip("skipping read-error cell: running as root bypasses the permission bit (0o000), which would make this cell pass vacuously rather than exercising the read-error path")
			}

			home := fakeHome(t)
			settingsPath := filepath.Join(home, ".claude", "settings.json")

			var fixture []byte
			if tc.readError {
				fixture = []byte("{\"someUnrelatedKey\":true}\n")
				writeFile(t, settingsPath, string(fixture))
				if err := os.Chmod(settingsPath, 0o000); err != nil {
					t.Fatalf("chmod settings.json to 0o000: %v", err)
				}
				t.Cleanup(func() {
					_ = os.Chmod(settingsPath, 0o644)
				})
			} else {
				fixture = []byte(malformedSettingsJSON)
				writeFile(t, settingsPath, string(fixture))
			}

			c := claudeTarget{}
			result := tc.op(c, LocationGlobal)

			// Half 1: the error surfaced, naming the settings path (so an
			// unrelated failure elsewhere in Install cannot satisfy this).
			if len(result.Errors) == 0 {
				t.Fatalf("expected result.Errors to be non-empty, got none: %+v", result)
			}
			foundSettingsError := false
			for _, e := range result.Errors {
				if strings.Contains(e.Error(), settingsPath) {
					foundSettingsError = true
				}
			}
			if !foundSettingsError {
				t.Fatalf("no error in result.Errors mentions the settings path %s: %v", settingsPath, result.Errors)
			}

			// Half 2: the file's bytes are exactly the fixture, unchanged.
			// Restore the read-error mode first so the file can be read for
			// comparison — the assertion is on content, not on the mode
			// itself, and t.Cleanup above provides a second, redundant
			// restore in case this one is skipped by a fatal failure.
			if tc.readError {
				if err := os.Chmod(settingsPath, 0o644); err != nil {
					t.Fatalf("restore chmod on settings.json: %v", err)
				}
			}
			got, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("read settings.json after operation: %v", err)
			}
			if string(got) != string(fixture) {
				t.Fatalf("settings.json bytes changed despite the surfaced error:\nbefore=%q\nafter=%q", fixture, got)
			}

			if tc.assertNotGone {
				if _, err := os.Stat(settingsPath); err != nil {
					t.Fatalf("settings.json should not have been deleted by uninstall: %v", err)
				}
			}
		})
	}
}

// TestClaude_SettingsReadFailureMatrixIsNotVacuous is the guard-the-guard
// companion to TestClaude_SettingsReadFailureNeverDestroysContent: the
// same install/uninstall operations, run against a well-formed
// settings.json, must produce zero Errors and must actually write —
// proving the matrix above discriminates a real read failure rather than
// passing because Install/Uninstall never touch settingsPath at all
// (mirrors TestHookRegistrationMatchesFragmentAndScript_ComparisonDiscriminates's
// discipline elsewhere in this package).
func TestClaude_SettingsReadFailureMatrixIsNotVacuous(t *testing.T) {
	t.Run("install against well-formed settings.json", func(t *testing.T) {
		home := fakeHome(t)
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		before := "{}\n"
		writeFile(t, settingsPath, before)

		c := claudeTarget{}
		result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
		if len(result.Errors) != 0 {
			t.Fatalf("expected zero errors against well-formed settings.json, got: %v", result.Errors)
		}

		after := readFile(t, settingsPath)
		if after == before {
			t.Fatalf("settings.json bytes did not change on install — the matrix would be vacuous")
		}
	})

	t.Run("uninstall against well-formed settings.json holding codegraph's own content", func(t *testing.T) {
		home := fakeHome(t)
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		c := claudeTarget{}
		c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
		before := readFile(t, settingsPath)

		result := c.Uninstall(LocationGlobal)
		if len(result.Errors) != 0 {
			t.Fatalf("expected zero errors against well-formed settings.json, got: %v", result.Errors)
		}

		after, err := os.ReadFile(settingsPath)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("read settings.json after uninstall: %v", err)
		}
		if string(after) == before {
			t.Fatalf("settings.json bytes did not change on uninstall — the matrix would be vacuous")
		}
	})
}
