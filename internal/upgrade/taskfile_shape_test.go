package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// --- fixture paths ---------------------------------------------------------

// workflowsDir is the on-disk path (relative to this package) to the
// directory this guard scans for job `name:` fields.
const workflowsDir = "../../.github/workflows"

// rootGoModPath and the two tool-modfile paths this guard reads off disk to
// prove D-03's isolation held: the root module must declare no tool
// directive and require none of the three build-tool packages, and both
// tool modfiles must exist as distinct files with a non-empty rationale
// header.
const (
	rootGoModPath   = "../../go.mod"
	toolModfilePath = "../../go.tool.mod"
	lintModfilePath = "../../go.tool-lint.mod"
)

// requiredCheckNames is the literal fixture of GitHub ruleset 20157557's
// six required-status-check contexts plus pr-title (a seventh required
// context enforced by the same ruleset but living in its own workflow
// file). Source: `gh api repos/seanb4t/codegraph-go/rulesets/20157557`,
// re-verified live 2026-08-01 (10-01-PLAN.md Task 1). Re-verify the same
// way before editing this fixture — a stale fixture here would make this
// guard assert the wrong thing rather than fail loudly.
var requiredCheckNames = []string{
	"test",
	"govulncheck (DIST-03, blocking)",
	"reproducibility (double-build hash-diff, DIST-04)",
	"perf regression gate (PERF-02, INDX-06)",
	"actionlint (workflow static analysis)",
	"goreleaser check (config validation, DIST-01)",
	"pr-title",
}

// forbiddenToolPackages are the three build-tool import paths that must
// live ONLY in the isolated tool modfiles (go.tool.mod / go.tool-lint.mod),
// never as a tool directive or a require line in the root go.mod (D-03).
var forbiddenToolPackages = []string{
	"github.com/go-task/task",
	"github.com/goreleaser/goreleaser",
	"github.com/rhysd/actionlint",
}

// --- parseX/mustX helper pairs ---------------------------------------------
//
// Following the convention established in release_workflow_shape_test.go
// and pr_title_lint_test.go: every parser is a pure `parseX(src string)
// (T, error)` core returning a non-nil error whenever its target is
// absent — never a usable zero value on a parse miss (the CR-01 defect
// class this idiom exists to stop) — plus a thin `mustX(t *testing.T, src
// string) T` wrapper that fails the test on error.

// parseWorkflowJobNames returns every column-4 (job-level, not step-level)
// `name:` value in workflow YAML source src, in file order.
func parseWorkflowJobNames(src string) ([]string, error) {
	nameRe := regexp.MustCompile(`^    name:\s*(.+)$`)
	var names []string
	for _, line := range strings.Split(src, "\n") {
		m := nameRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		v := strings.TrimSpace(m[1])
		v = strings.Trim(v, `"'`)
		if v != "" {
			names = append(names, v)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("parseWorkflowJobNames: no column-4 job name: key found")
	}
	return names, nil
}

func mustWorkflowJobNames(t *testing.T, src string) []string {
	t.Helper()
	v, err := parseWorkflowJobNames(src)
	if err != nil {
		t.Fatalf("mustWorkflowJobNames: %v", err)
	}
	return v
}

// parseGoModToolPackages returns the import paths listed in a go.mod's
// `tool (...)` block form or single-line `tool <pkg>` form. Returns a
// non-nil error if the source declares no tool directive at all — this is
// the DESIRED state for the root go.mod (D-03 isolation), so callers
// guarding the root module treat a non-nil error here as "isolation
// intact", not as a test-infrastructure failure.
func parseGoModToolPackages(src string) ([]string, error) {
	lines := strings.Split(src, "\n")
	blockRe := regexp.MustCompile(`^tool\s*\(\s*$`)
	singleRe := regexp.MustCompile(`^tool\s+(\S+)\s*$`)

	var pkgs []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if m := singleRe.FindStringSubmatch(line); m != nil {
			pkgs = append(pkgs, m[1])
			continue
		}
		if blockRe.MatchString(line) {
			for j := i + 1; j < len(lines); j++ {
				inner := strings.TrimSpace(lines[j])
				if inner == ")" {
					break
				}
				if inner != "" {
					pkgs = append(pkgs, inner)
				}
			}
		}
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("parseGoModToolPackages: no tool directive found")
	}
	return pkgs, nil
}

// parseGoModRequireVersion returns the pinned version of the first
// `require`-block or single-line `require` entry whose module path has pkg
// as a prefix (covers major-version suffixes like /v2, /v3). Returns a
// non-nil error if no matching require line exists — the DESIRED state for
// the root go.mod's relationship to forbiddenToolPackages.
func parseGoModRequireVersion(src, pkg string) (string, error) {
	lineRe := regexp.MustCompile(`^\s*(\S+)\s+(v\S+)`)
	for _, line := range strings.Split(src, "\n") {
		m := lineRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		modPath, version := m[1], m[2]
		if modPath == pkg || strings.HasPrefix(modPath, pkg+"/") {
			return version, nil
		}
	}
	return "", fmt.Errorf("parseGoModRequireVersion: no require line found for package %q", pkg)
}

// parseToolModfileHeaderComment returns the leading `//`-comment block that
// precedes a tool modfile's `module` directive line, with the `// ` prefix
// stripped from each line and joined by spaces. Returns a non-nil error if
// the source has no leading comment block, or the module line appears with
// no comment above it at all.
func parseToolModfileHeaderComment(src string) (string, error) {
	lines := strings.Split(src, "\n")
	moduleRe := regexp.MustCompile(`^module\s+\S+`)

	var commentLines []string
	for _, line := range lines {
		if moduleRe.MatchString(line) {
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "//") {
			// Non-comment, non-blank line before module: — treat as no
			// header (a stray line, not a documented rationale).
			continue
		}
		commentLines = append(commentLines, strings.TrimSpace(strings.TrimPrefix(trimmed, "//")))
	}

	joined := strings.TrimSpace(strings.Join(commentLines, " "))
	if joined == "" {
		return "", fmt.Errorf("parseToolModfileHeaderComment: no leading comment block found before the module directive")
	}
	return joined, nil
}

func mustToolModfileHeaderComment(t *testing.T, src string) string {
	t.Helper()
	v, err := parseToolModfileHeaderComment(src)
	if err != nil {
		t.Fatalf("mustToolModfileHeaderComment: %v", err)
	}
	return v
}

// --- tests -------------------------------------------------------------

// TestRequiredCheckNamesPreserved is the T-10-01-05 information-disclosure
// guard: it reads every real, on-disk workflow file and asserts each of
// GitHub ruleset 20157557's required-context strings is present as a job
// `name:` field somewhere in the set. A renamed required check silently
// un-gates `main` — this test fails the build on any such rename, naming
// the specific missing context.
func TestRequiredCheckNamesPreserved(t *testing.T) {
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("read %s: %v", workflowsDir, err)
	}

	jobNames := make(map[string]bool)
	scanned := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		path := filepath.Join(workflowsDir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		names := mustWorkflowJobNames(t, string(data))
		for _, n := range names {
			jobNames[n] = true
		}
		scanned++
	}
	if scanned == 0 {
		t.Fatalf("TestRequiredCheckNamesPreserved: found zero workflow files under %s", workflowsDir)
	}

	var missing []string
	for _, want := range requiredCheckNames {
		if !jobNames[want] {
			missing = append(missing, want)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("required-status-check job name(s) not found in any workflow file (ruleset 20157557 required-context set is stale or a job was renamed): %v", missing)
	}
}

// TestRequiredCheckNamesPreserved_ZeroJobsIsError is the edge case: a
// workflow file that parses as text but declares zero job-level `name:`
// keys must surface as a non-nil error from parseWorkflowJobNames, never
// as a silently-passing empty set — the same CR-01 defect class every
// parser in this file is built to avoid.
func TestRequiredCheckNamesPreserved_ZeroJobsIsError(t *testing.T) {
	src := "name: empty\non:\n  push:\njobs: {}\n"
	if _, err := parseWorkflowJobNames(src); err == nil {
		t.Fatalf("parseWorkflowJobNames: expected a non-nil error for a workflow with zero job name: keys, got nil")
	}
}

// TestToolModfilesRemainIsolated is the D-03 isolation guard: go.tool.mod
// and go.tool-lint.mod must exist as two distinct files, the root go.mod
// must declare no tool directive and require none of the three build-tool
// packages, and each tool modfile's header comment must be non-empty and
// state the isolation rationale — without that header the two files read
// as an accident and someone merges them.
func TestToolModfilesRemainIsolated(t *testing.T) {
	toolInfo, err := os.Stat(toolModfilePath)
	if err != nil {
		t.Fatalf("stat %s: %v", toolModfilePath, err)
	}
	lintInfo, err := os.Stat(lintModfilePath)
	if err != nil {
		t.Fatalf("stat %s: %v", lintModfilePath, err)
	}
	if os.SameFile(toolInfo, lintInfo) {
		t.Fatalf("go.tool.mod and go.tool-lint.mod resolve to the same file — they must be two distinct modfiles (D-03)")
	}

	rootData, err := os.ReadFile(rootGoModPath)
	if err != nil {
		t.Fatalf("read %s: %v", rootGoModPath, err)
	}
	rootSrc := string(rootData)

	if pkgs, toolErr := parseGoModToolPackages(rootSrc); toolErr == nil {
		t.Fatalf("root go.mod declares a tool directive %v — build tools must live only in go.tool.mod/go.tool-lint.mod (D-03)", pkgs)
	}

	for _, pkg := range forbiddenToolPackages {
		if version, reqErr := parseGoModRequireVersion(rootSrc, pkg); reqErr == nil {
			t.Fatalf("root go.mod requires %s@%s directly — build tools must live only in go.tool.mod/go.tool-lint.mod (D-03)", pkg, version)
		}
	}

	for _, path := range []string{toolModfilePath, lintModfilePath} {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		comment := mustToolModfileHeaderComment(t, string(data))
		if !strings.Contains(strings.ToLower(comment), "isolat") {
			t.Fatalf("%s: header comment does not mention isolation rationale, got: %q", path, comment)
		}
	}
}

// TestToolModfilesRemainIsolated_AbsentModfileIsError is the edge case: a
// modfile that does not exist on disk at all must produce a non-nil error
// naming the missing path, never a skipped assertion.
func TestToolModfilesRemainIsolated_AbsentModfileIsError(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "go.tool-does-not-exist.mod")
	_, err := os.ReadFile(missingPath)
	if err == nil {
		t.Fatalf("os.ReadFile(%s): expected a non-nil error for an absent modfile, got nil", missingPath)
	}
	if !strings.Contains(err.Error(), missingPath) {
		t.Fatalf("os.ReadFile(%s) error does not name the missing path: %v", missingPath, err)
	}
}

// TestTaskfileShapeHelpersFailLoudly is the T-09-01-07-style repudiation
// guard extended to this file's own parsers: every pure parse core must
// return a non-nil error — never a usable zero value — when its target is
// absent from the source.
func TestTaskfileShapeHelpersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "parseWorkflowJobNames: empty input",
			fn: func() error {
				_, err := parseWorkflowJobNames("")
				return err
			},
		},
		{
			name: "parseGoModToolPackages: empty input",
			fn: func() error {
				_, err := parseGoModToolPackages("")
				return err
			},
		},
		{
			name: "parseGoModRequireVersion: empty input",
			fn: func() error {
				_, err := parseGoModRequireVersion("", "github.com/example/pkg")
				return err
			},
		},
		{
			name: "parseToolModfileHeaderComment: empty input",
			fn: func() error {
				_, err := parseToolModfileHeaderComment("")
				return err
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.fn(); err == nil {
				t.Fatalf("%s: expected a non-nil error, got nil", c.name)
			}
		})
	}
}
