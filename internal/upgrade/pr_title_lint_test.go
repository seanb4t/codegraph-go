package upgrade

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// prTitleWorkflowPath is the on-disk path (relative to this package) to the
// PR-title conventional-commit lint workflow this test executes against.
// Written BEFORE the workflow exists (Task 1, RED) so the first run fails
// loudly on a missing file; consumed unchanged once Task 2 creates the
// workflow (GREEN).
const prTitleWorkflowPath = "../../.github/workflows/pr-title.yml"

// prTitleLintStepName is the exact `- name:` value of the pr-title.yml step
// this test extracts and executes via mustWorkflowStepRunBlock (09-01).
// Task 2's workflow MUST use this exact step name.
const prTitleLintStepName = "Check PR title matches Conventional Commits"

// prTitleEnvVar is the environment variable name the lint step's env: block
// must bind the PR title to. The title reaches the shell ONLY through this
// variable — never interpolated into the command text (T-09-03-01); a PR
// title is attacker-controlled input crossing into a shell interpreter.
const prTitleEnvVar = "TITLE"

// runPRTitleLintStep reads pr-title.yml off disk, extracts the lint step's
// run block via mustWorkflowStepRunBlock (09-01), writes it to a temp
// script, and executes it with bash with the title supplied ONLY through
// prTitleEnvVar (never as a command-line argument or interpolated text).
// Returns the process exit code and combined stdout/stderr.
func runPRTitleLintStep(t *testing.T, title string) (exitCode int, output string) {
	t.Helper()

	data, err := os.ReadFile(prTitleWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", prTitleWorkflowPath, err)
	}
	src := string(data)

	block := mustWorkflowStepRunBlock(t, src, prTitleLintStepName)
	if strings.TrimSpace(block) == "" {
		t.Fatalf("runPRTitleLintStep: extracted run block for %q is empty — refusing to execute a vacuous script (WR-02 defect class)", prTitleLintStepName)
	}

	scriptPath := filepath.Join(t.TempDir(), "pr-title-lint.sh")
	if err := os.WriteFile(scriptPath, []byte(block), 0o644); err != nil {
		t.Fatalf("write extracted step script: %v", err)
	}

	cmd := exec.Command("bash", scriptPath)
	// Appended entries win over any duplicate key already present in
	// os.Environ() (documented exec.Cmd.Env behavior).
	cmd.Env = append(os.Environ(), prTitleEnvVar+"="+title)
	out, runErr := cmd.CombinedOutput()

	code := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitErr = ee
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("run pr-title lint step: %v", runErr)
		}
	}

	return code, string(out)
}

// TestPRTitleLintAcceptsAndRejects proves the D-08 PR-title
// conventional-commit gate by executing the lint step's OWN shipped shell
// (extracted verbatim from the on-disk workflow) against a table of titles,
// never by reading the YAML or the regex source as text. Reject rows assert
// a non-zero exit plus a workflow error annotation naming the offending
// title; accept rows assert exit 0, one per accepted type word plus scope,
// breaking-marker, and adversarial-injection cases.
func TestPRTitleLintAcceptsAndRejects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test extracts and executes bash from pr-title.yml; skipping on windows")
	}

	// Extraction non-vacuity guard (plan acceptance criterion): confirm the
	// step's run block is non-empty BEFORE the table loop runs, so a
	// silently empty extraction can't make every row vacuously pass.
	data, err := os.ReadFile(prTitleWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", prTitleWorkflowPath, err)
	}
	block := mustWorkflowStepRunBlock(t, string(data), prTitleLintStepName)
	if strings.TrimSpace(block) == "" {
		t.Fatalf("extracted run block for %q is empty — refusing to run table rows against a vacuous script", prTitleLintStepName)
	}

	type row struct {
		name   string
		title  string
		accept bool
	}

	rows := []row{
		// --- reject rows ---
		{"reject_bare_descriptive_title_no_type_prefix", "update some stuff", false},
		{"reject_unaccepted_type_word", "wip: something not conventional", false},
		{"reject_missing_colon_space_separator", "feat:no space after the colon", false},
		{"reject_empty_subject_after_colon", "feat: ", false},

		// --- accept rows: one per accepted type word ---
		{"accept_feat", "feat: add support for foo", true},
		{"accept_fix", "fix: correct a bug", true},
		{"accept_perf", "perf: speed up the indexer", true},
		{"accept_refactor", "refactor: simplify the extractor", true},
		{"accept_docs", "docs: update the readme", true},
		{"accept_chore", "chore: bump dependencies", true},
		{"accept_ci", "ci: add a workflow step", true},
		{"accept_test", "test: add coverage for foo", true},
		{"accept_build", "build: update the build script", true},
		{"accept_revert", "revert: revert the previous commit", true},

		// --- accept rows: scope + breaking-change marker ---
		{"accept_with_lowercase_alphanumeric_scope", "feat(parser): add scope support", true},
		{"accept_with_breaking_change_marker", "feat!: introduce a breaking change", true},
	}

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			code, out := runPRTitleLintStep(t, r.title)
			if r.accept {
				if code != 0 {
					t.Fatalf("title %q: exit code = %d, want 0\noutput:\n%s", r.title, code, out)
				}
				return
			}
			if code == 0 {
				t.Fatalf("title %q: exit code = 0, want non-zero (non-conformant title must be rejected)", r.title)
			}
			if !strings.Contains(out, "::error::") {
				t.Fatalf("title %q: expected a workflow error annotation (::error::) in output, got:\n%s", r.title, out)
			}
			if !strings.Contains(out, r.title) {
				t.Fatalf("title %q: expected the error annotation to name the offending title, got:\n%s", r.title, out)
			}
		})
	}

	// --- injection-safety adversarial accept row ---
	//
	// The executable form of the env-indirection guarantee: a conformant
	// title whose subject embeds shell metacharacters and a command
	// substitution must pass the lint AND produce no side effect — proving
	// the title crosses into the step as data, never as script text.
	t.Run("accept_adversarial_shell_metacharacters_no_side_effect", func(t *testing.T) {
		markerDir := t.TempDir()
		markerPath := filepath.Join(markerDir, "INJECTED")
		title := "feat: totally normal subject $(touch " + markerPath + ") `touch " + markerPath + "` ; rm -rf / #"

		code, out := runPRTitleLintStep(t, title)
		if code != 0 {
			t.Fatalf("adversarial title: exit code = %d, want 0\noutput:\n%s", code, out)
		}
		if _, statErr := os.Stat(markerPath); statErr == nil {
			t.Fatalf("adversarial title executed as shell — marker file %s was created, title reached the shell as script text instead of data", markerPath)
		}
	})
}
