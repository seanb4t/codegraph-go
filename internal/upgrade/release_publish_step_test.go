package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// publishStepName is the exact `- name:` value of the release.yml step this
// test exercises end-to-end via a stubbed `gh` on PATH.
const publishStepName = "Publish GitHub release"

// ghArgvLogEnvVar is the environment variable name the stubbed `gh` script
// reads to locate its argv-recording log file. Using an env var (rather
// than baking the log path into the generated script) keeps stubGHDir's
// stub script content independent of where the log file actually lives.
const ghArgvLogEnvVar = "CODEGRAPH_TEST_GH_ARGV_LOG"

// ghStubScriptTemplate is a POSIX-shell stub standing in for the real `gh`
// CLI. It records its full argv as one line per invocation to the file
// named by ghArgvLogEnvVar, then exits viewExit for a release-existence
// lookup (`gh release view ...`), createExit for a release-creation call
// (`gh release create ...`), and 0 for anything else — notably `gh release
// upload ...`, which this suite never needs to fail.
const ghStubScriptTemplate = `#!/bin/sh
set -u
if [ -n "${%s:-}" ]; then
  echo "$@" >> "${%s}"
fi
if [ "${1:-}" = "release" ] && [ "${2:-}" = "view" ]; then
  exit %d
fi
if [ "${1:-}" = "release" ] && [ "${2:-}" = "create" ]; then
  exit %d
fi
exit 0
`

// stubGHDir creates a temp dir containing an executable `gh` stub script
// configured with the given release-view/release-create exit codes, plus
// the (not-yet-created) path of an argv log file. Returns the stub's
// directory (to be prepended to PATH) and the log file path (to be
// exported via ghArgvLogEnvVar when running the extracted step).
func stubGHDir(t *testing.T, viewExit, createExit int) (dir, logPath string) {
	t.Helper()
	dir = t.TempDir()
	logPath = filepath.Join(t.TempDir(), "gh-argv.log")

	script := fmt.Sprintf(ghStubScriptTemplate, ghArgvLogEnvVar, ghArgvLogEnvVar, viewExit, createExit)
	ghPath := filepath.Join(dir, "gh")
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write gh stub: %v", err)
	}
	return dir, logPath
}

// runExtractedStep reads release.yml off disk, extracts the "Publish
// GitHub release" step's run block via mustWorkflowStepRunBlock (09-01),
// writes it to a temp script, and runs it with bash from workDir against a
// PATH whose first element is stubDir plus the same env-var names the
// step's env: block supplies (GH_TOKEN/TAG/REPO) and the argv-log path
// variable. Returns the process exit code and the argv lines the gh stub
// recorded, in invocation order.
func runExtractedStep(t *testing.T, workDir, stubDir, logPath, tag string) (exitCode int, argvLines []string) {
	t.Helper()

	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	src := string(data)

	block := mustWorkflowStepRunBlock(t, src, publishStepName)
	if strings.TrimSpace(block) == "" {
		t.Fatalf("runExtractedStep: extracted run block for %q is empty — refusing to execute a vacuous script (WR-02 defect class)", publishStepName)
	}

	scriptPath := filepath.Join(t.TempDir(), "publish-step.sh")
	if err := os.WriteFile(scriptPath, []byte(block), 0o644); err != nil {
		t.Fatalf("write extracted step script: %v", err)
	}

	pathEnv := stubDir + string(os.PathListSeparator) + os.Getenv("PATH")

	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = workDir
	// Appended entries win over any duplicate key already present in
	// os.Environ() (documented exec.Cmd.Env behavior), so this reliably
	// overrides PATH to put the stub first.
	cmd.Env = append(os.Environ(),
		"PATH="+pathEnv,
		"GH_TOKEN=test-token-placeholder",
		"TAG="+tag,
		"REPO=seanb4t/codegraph-go",
		ghArgvLogEnvVar+"="+logPath,
	)
	out, runErr := cmd.CombinedOutput()
	t.Logf("extracted step output:\n%s", out)

	code := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitErr = ee
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("run extracted step: %v", runErr)
		}
	}

	if raw, readErr := os.ReadFile(logPath); readErr == nil {
		for _, l := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
			if l != "" {
				argvLines = append(argvLines, l)
			}
		}
	}

	return code, argvLines
}

// argvLineContainsAll reports whether any recorded argv line contains
// every one of tokens.
func argvLineContainsAll(argvLines []string, tokens ...string) bool {
	for _, line := range argvLines {
		all := true
		for _, tok := range tokens {
			if !strings.Contains(line, tok) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// writePublishStepFixtureAssets populates dir with a small set of files
// matching the release-asset glob (codegraph_*) the publish step consumes:
// two binaries, a checksums file, and a signature sidecar — exercising the
// glob's real breadth, not just a single trivial match.
func writePublishStepFixtureAssets(t *testing.T, dir, tag string) {
	t.Helper()
	files := []string{
		"codegraph_" + tag + "_linux_amd64",
		"codegraph_" + tag + "_darwin_arm64",
		"codegraph_" + tag + "_checksums.txt",
		"codegraph_" + tag + "_linux_amd64.sigstore.json",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("fixture"), 0o644); err != nil {
			t.Fatalf("write fixture asset %s: %v", f, err)
		}
	}
}

// TestPublishReleaseStepBranches proves the D-04 create-if-absent-else-
// upload-clobber behavior of release.yml's "Publish GitHub release" step
// by executing the step's OWN shipped shell (extracted verbatim from the
// on-disk workflow) against a recording gh stub, never by reading the
// YAML text. Five subtests match the plan's <behavior> block exactly.
func TestPublishReleaseStepBranches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test extracts and executes bash from release.yml; skipping on windows")
	}

	t.Run("release_exists_uploads_with_clobber", func(t *testing.T) {
		tag := "v1.2.3"
		workDir := t.TempDir()
		writePublishStepFixtureAssets(t, workDir, tag)
		// view succeeds (exit 0) -> release already exists -> upload branch.
		stubDir, logPath := stubGHDir(t, 0, 0)

		code, argv := runExtractedStep(t, workDir, stubDir, logPath, tag)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0\nrecorded argv:\n%s", code, strings.Join(argv, "\n"))
		}
		if !argvLineContainsAll(argv, "release", "upload", "--clobber") {
			t.Fatalf("expected an upload invocation carrying --clobber, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
		if argvLineContainsAll(argv, "release", "create") {
			t.Fatalf("release-exists case must never invoke create, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
		if argvLineContainsAll(argv, "upload", "--generate-notes") {
			t.Fatalf("upload invocation must not carry a notes-generation flag, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
		if argvLineContainsAll(argv, "upload", "--prerelease") {
			t.Fatalf("upload invocation must not carry a prerelease flag, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
	})

	t.Run("release_absent_prerelease_tag_creates_with_prerelease_and_notes", func(t *testing.T) {
		tag := "v2.0.0-rc.1"
		workDir := t.TempDir()
		writePublishStepFixtureAssets(t, workDir, tag)
		// view fails (exit 1) -> release absent; create succeeds.
		stubDir, logPath := stubGHDir(t, 1, 0)

		code, argv := runExtractedStep(t, workDir, stubDir, logPath, tag)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0\nrecorded argv:\n%s", code, strings.Join(argv, "\n"))
		}
		if !argvLineContainsAll(argv, "release", "create", "--prerelease", "--generate-notes") {
			t.Fatalf("expected a create invocation carrying --prerelease and --generate-notes, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
		if argvLineContainsAll(argv, "release", "upload") {
			t.Fatalf("release-absent case must never invoke upload, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
	})

	t.Run("release_absent_stable_tag_creates_without_prerelease", func(t *testing.T) {
		tag := "v2.0.0"
		workDir := t.TempDir()
		writePublishStepFixtureAssets(t, workDir, tag)
		stubDir, logPath := stubGHDir(t, 1, 0)

		code, argv := runExtractedStep(t, workDir, stubDir, logPath, tag)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0\nrecorded argv:\n%s", code, strings.Join(argv, "\n"))
		}
		if !argvLineContainsAll(argv, "release", "create") {
			t.Fatalf("expected a create invocation, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
		if argvLineContainsAll(argv, "--prerelease") {
			t.Fatalf("stable-tag create invocation must not carry --prerelease, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
	})

	t.Run("zero_assets_fails_loud_invokes_neither_branch", func(t *testing.T) {
		tag := "v2.0.0"
		workDir := t.TempDir() // deliberately bare — no fixture assets written
		stubDir, logPath := stubGHDir(t, 1, 0)

		code, argv := runExtractedStep(t, workDir, stubDir, logPath, tag)
		if code == 0 {
			t.Fatalf("exit code = 0, want non-zero for a zero-asset working directory\nrecorded argv:\n%s", strings.Join(argv, "\n"))
		}
		if len(argv) != 0 {
			t.Fatalf("expected zero gh invocations for a zero-asset working directory, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
	})

	t.Run("release_absent_create_fails_no_upload_fallback", func(t *testing.T) {
		tag := "v2.0.0"
		workDir := t.TempDir()
		writePublishStepFixtureAssets(t, workDir, tag)
		// view fails (release absent) AND create fails.
		stubDir, logPath := stubGHDir(t, 1, 1)

		code, argv := runExtractedStep(t, workDir, stubDir, logPath, tag)
		if code == 0 {
			t.Fatalf("exit code = 0, want non-zero when the create call fails\nrecorded argv:\n%s", strings.Join(argv, "\n"))
		}
		if argvLineContainsAll(argv, "release", "upload") {
			t.Fatalf("a failed create must never fall back to upload, got recorded argv:\n%s", strings.Join(argv, "\n"))
		}
	})
}
