package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// releaseWorkflowPath is the on-disk path (relative to this package) to the
// LOCKED release workflow file this guard reads off disk. It is the single
// mechanical link (T-09-01-01) between release.yml's real, on-disk shape and
// internal/upgrade's compiled-in releaseWorkflowRefPattern: a drift here
// turns into a red test instead of a silent `codegraph upgrade` break for
// every shipped binary.
const releaseWorkflowPath = "../../.github/workflows/release.yml"

// --- workflow-source helper pairs -----------------------------------------
//
// Each helper below is a pure `parseX(src string) (T, error)` core plus a
// thin `mustX(t *testing.T, src string) T` wrapper that fails the test on
// error. Every core returns a non-nil error — never a usable zero value —
// when its target is absent (CR-01 class defect this phase exists to stop;
// see TestWorkflowSourceHelpersFailLoudly).

// parseWorkflowTopLevelName returns the value of the column-0 `name:` key
// in workflow YAML source src.
func parseWorkflowTopLevelName(src string) (string, error) {
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(line, "name:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			v = strings.Trim(v, `"'`)
			if v == "" {
				return "", fmt.Errorf("parseWorkflowTopLevelName: column-0 name: key present but empty")
			}
			return v, nil
		}
	}
	return "", fmt.Errorf("parseWorkflowTopLevelName: no column-0 name: key found")
}

func mustWorkflowTopLevelName(t *testing.T, src string) string {
	t.Helper()
	v, err := parseWorkflowTopLevelName(src)
	if err != nil {
		t.Fatalf("mustWorkflowTopLevelName: %v", err)
	}
	return v
}

// parseWorkflowOnKeys returns the two-space-indented keys directly under the
// column-0 `on:` key.
func parseWorkflowOnKeys(src string) ([]string, error) {
	lines := strings.Split(src, "\n")
	onRe := regexp.MustCompile(`^on:\s*$`)
	onIdx := -1
	for i, line := range lines {
		if onRe.MatchString(line) {
			onIdx = i
			break
		}
	}
	if onIdx == -1 {
		return nil, fmt.Errorf("parseWorkflowOnKeys: no column-0 on: key found")
	}

	keyRe := regexp.MustCompile(`^  ([A-Za-z0-9_]+):`)
	var keys []string
	for _, line := range lines[onIdx+1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if m := keyRe.FindStringSubmatch(line); m != nil {
			keys = append(keys, m[1])
			continue
		}
		if !strings.HasPrefix(line, " ") {
			// back to column 0 — the on: block has ended
			break
		}
		// a more deeply nested line (e.g. tags: - "v[0-9]*") — skip
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("parseWorkflowOnKeys: on: block has no 2-space-indented child keys")
	}
	return keys, nil
}

func mustWorkflowOnKeys(t *testing.T, src string) []string {
	t.Helper()
	v, err := parseWorkflowOnKeys(src)
	if err != nil {
		t.Fatalf("mustWorkflowOnKeys: %v", err)
	}
	return v
}

// parseWorkflowPushTagPatterns returns the list items under
// on: -> push: -> tags:, quote-stripped.
func parseWorkflowPushTagPatterns(src string) ([]string, error) {
	lines := strings.Split(src, "\n")
	onRe := regexp.MustCompile(`^on:\s*$`)
	onIdx := -1
	for i, line := range lines {
		if onRe.MatchString(line) {
			onIdx = i
			break
		}
	}
	if onIdx == -1 {
		return nil, fmt.Errorf("parseWorkflowPushTagPatterns: no column-0 on: key found")
	}

	pushRe := regexp.MustCompile(`^  push:\s*$`)
	pushIdx := -1
	for i := onIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if pushRe.MatchString(line) {
			pushIdx = i
			break
		}
		if !strings.HasPrefix(line, " ") {
			break // on: block ended without a push: child
		}
	}
	if pushIdx == -1 {
		return nil, fmt.Errorf("parseWorkflowPushTagPatterns: on: block has no push: child")
	}

	tagsRe := regexp.MustCompile(`^    tags:\s*$`)
	tagsIdx := -1
	for i := pushIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if tagsRe.MatchString(line) {
			tagsIdx = i
			break
		}
		if !strings.HasPrefix(line, "    ") {
			break // push: block ended without a tags: child
		}
	}
	if tagsIdx == -1 {
		return nil, fmt.Errorf("parseWorkflowPushTagPatterns: push: block has no tags: child")
	}

	itemRe := regexp.MustCompile(`^\s*-\s*(.+)$`)
	var patterns []string
	for i := tagsIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if m := itemRe.FindStringSubmatch(line); m != nil && strings.HasPrefix(line, "      ") {
			v := strings.TrimSpace(m[1])
			v = strings.Trim(v, `"'`)
			patterns = append(patterns, v)
			continue
		}
		break
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("parseWorkflowPushTagPatterns: tags: list has zero items")
	}
	return patterns, nil
}

func mustWorkflowPushTagPatterns(t *testing.T, src string) []string {
	t.Helper()
	v, err := parseWorkflowPushTagPatterns(src)
	if err != nil {
		t.Fatalf("mustWorkflowPushTagPatterns: %v", err)
	}
	return v
}

// parseWorkflowStepRunBlock locates the `- name: <stepName>` step in
// workflow YAML source src and returns that step's `run:` block body with
// its common leading indentation removed. Kept general on purpose: it is
// consumed unchanged by plans 09-02 and 09-03.
func parseWorkflowStepRunBlock(src, stepName string) (string, error) {
	lines := strings.Split(src, "\n")
	nameRe := regexp.MustCompile(`^(\s*)-\s*name:\s*(.+)$`)

	stepIdx := -1
	baseIndent := ""
	for i, line := range lines {
		m := nameRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := strings.TrimSpace(m[2])
		name = strings.Trim(name, `"'`)
		if name == stepName {
			stepIdx = i
			baseIndent = m[1]
			break
		}
	}
	if stepIdx == -1 {
		return "", fmt.Errorf("parseWorkflowStepRunBlock: no step named %q found", stepName)
	}

	childIndent := baseIndent + "  "

	// Find the end of this step's block: the next non-blank line at
	// indentation <= len(baseIndent), or EOF.
	stepEnd := len(lines)
	for i := stepIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent <= len(baseIndent) {
			stepEnd = i
			break
		}
	}

	runBlockRe := regexp.MustCompile(`^` + regexp.QuoteMeta(childIndent) + `run:\s*[|>][+-]?\s*$`)
	runInlineRe := regexp.MustCompile(`^` + regexp.QuoteMeta(childIndent) + `run:\s+(\S.*)$`)

	runIdx := -1
	inlineRunValue := ""
	for i := stepIdx; i < stepEnd; i++ {
		line := lines[i]
		if runBlockRe.MatchString(line) {
			runIdx = i
			break
		}
		if m := runInlineRe.FindStringSubmatch(line); m != nil {
			runIdx = i
			inlineRunValue = strings.TrimSpace(m[1])
			break
		}
	}
	if runIdx == -1 {
		return "", fmt.Errorf("parseWorkflowStepRunBlock: step %q has no run: key", stepName)
	}
	if inlineRunValue != "" {
		return inlineRunValue, nil
	}

	var bodyLines []string
	minIndent := -1
	for i := runIdx + 1; i < stepEnd; i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			bodyLines = append(bodyLines, "")
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent <= len(childIndent) {
			break
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
		bodyLines = append(bodyLines, line)
	}
	if minIndent == -1 {
		return "", fmt.Errorf("parseWorkflowStepRunBlock: step %q has a run: key with no body", stepName)
	}

	out := make([]string, len(bodyLines))
	for i, l := range bodyLines {
		if l == "" {
			continue
		}
		out[i] = l[minIndent:]
	}
	return strings.Join(out, "\n"), nil
}

func mustWorkflowStepRunBlock(t *testing.T, src, stepName string) string {
	t.Helper()
	v, err := parseWorkflowStepRunBlock(src, stepName)
	if err != nil {
		t.Fatalf("mustWorkflowStepRunBlock(%q): %v", stepName, err)
	}
	return v
}

// releaseMatrixEntry is one `- runner: ...` item under the `build` job's
// `strategy.matrix.include:` list.
type releaseMatrixEntry struct {
	Runner   string
	Goos     string
	Goarch   string
	NeedsZig bool
}

// parseReleaseBuildMatrix returns every `- runner: ...` entry (plus its
// goos/goarch/needs_zig siblings) in the build job's matrix include: list.
// Each entry starts at a `- runner:` line; goos:/goarch:/needs_zig: lines
// that follow, up to the next `- runner:` line or the job's `steps:` key,
// belong to that entry.
func parseReleaseBuildMatrix(src string) ([]releaseMatrixEntry, error) {
	lines := strings.Split(src, "\n")
	entryRe := regexp.MustCompile(`^\s*-\s*runner:\s*(\S+)\s*$`)
	fieldRe := regexp.MustCompile(`^\s*(goos|goarch|needs_zig):\s*(\S+)\s*$`)
	stepsRe := regexp.MustCompile(`^\s*steps:\s*$`)

	var entries []releaseMatrixEntry
	var cur *releaseMatrixEntry
	for _, line := range lines {
		if stepsRe.MatchString(line) && cur != nil {
			// The matrix include: list ends where the job's steps: key
			// begins — stop before misreading an unrelated goos:/goarch:
			// pair from later in the file.
			break
		}
		if m := entryRe.FindStringSubmatch(line); m != nil {
			if cur != nil {
				entries = append(entries, *cur)
			}
			cur = &releaseMatrixEntry{Runner: strings.Trim(m[1], `"'`)}
			continue
		}
		if cur == nil {
			continue
		}
		if m := fieldRe.FindStringSubmatch(line); m != nil {
			val := strings.Trim(m[2], `"'`)
			switch m[1] {
			case "goos":
				cur.Goos = val
			case "goarch":
				cur.Goarch = val
			case "needs_zig":
				cur.NeedsZig = val == "true"
			}
		}
	}
	if cur != nil {
		entries = append(entries, *cur)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("parseReleaseBuildMatrix: no matrix include: entries found (no '- runner: ...' lines)")
	}
	return entries, nil
}

func mustReleaseBuildMatrix(t *testing.T, src string) []releaseMatrixEntry {
	t.Helper()
	v, err := parseReleaseBuildMatrix(src)
	if err != nil {
		t.Fatalf("mustReleaseBuildMatrix: %v", err)
	}
	return v
}

// macOSClassRunnerPatterns is a small allow-set of runner-label FAMILIES
// (not one literal string) that count as "a real macOS host" for D-08's
// purposes: GitHub-hosted macos-* labels, and Namespace's
// namespace-profile-macos-* profiles. A later, deliberate move to a
// different native macOS profile within either family stays green; a move
// to any linux profile (namespace-profile-linux-* or ubuntu-*) still goes
// red — see TestDarwinLegsBuildNatively.
var macOSClassRunnerPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^macos-`),
	regexp.MustCompile(`^namespace-profile-macos-`),
}

func isMacOSClassRunner(label string) bool {
	for _, re := range macOSClassRunnerPatterns {
		if re.MatchString(label) {
			return true
		}
	}
	return false
}

// provenanceJobShape is the subset of the `provenance:` job's YAML this
// package cares about: which reusable workflow it calls, and whether it
// declares a runs-on: of its own (it must not — D-07).
type provenanceJobShape struct {
	Uses      string
	HasRunsOn bool
}

// parseReleaseProvenanceJob locates the top-level `provenance:` job and
// returns its `uses:` reusable-workflow reference plus whether it declares
// its own `runs-on:` key anywhere in its body.
func parseReleaseProvenanceJob(src string) (provenanceJobShape, error) {
	lines := strings.Split(src, "\n")
	jobRe := regexp.MustCompile(`^  provenance:\s*$`)
	jobIdx := -1
	for i, line := range lines {
		if jobRe.MatchString(line) {
			jobIdx = i
			break
		}
	}
	if jobIdx == -1 {
		return provenanceJobShape{}, fmt.Errorf("parseReleaseProvenanceJob: no top-level 'provenance:' job found")
	}

	otherJobRe := regexp.MustCompile(`^  [A-Za-z0-9_-]+:\s*$`)
	jobEnd := len(lines)
	for i := jobIdx + 1; i < len(lines); i++ {
		if otherJobRe.MatchString(lines[i]) {
			jobEnd = i
			break
		}
	}

	usesRe := regexp.MustCompile(`^\s*uses:\s*(\S+)\s*$`)
	runsOnRe := regexp.MustCompile(`^\s*runs-on:`)

	var shape provenanceJobShape
	for i := jobIdx + 1; i < jobEnd; i++ {
		line := lines[i]
		if m := usesRe.FindStringSubmatch(line); m != nil && shape.Uses == "" {
			shape.Uses = strings.Trim(m[1], `"'`)
		}
		if runsOnRe.MatchString(line) {
			shape.HasRunsOn = true
		}
	}
	if shape.Uses == "" {
		return provenanceJobShape{}, fmt.Errorf("parseReleaseProvenanceJob: provenance job has no uses: key")
	}
	return shape, nil
}

func mustReleaseProvenanceJob(t *testing.T, src string) provenanceJobShape {
	t.Helper()
	v, err := parseReleaseProvenanceJob(src)
	if err != nil {
		t.Fatalf("mustReleaseProvenanceJob: %v", err)
	}
	return v
}

// slsaGeneratorTaggedRe matches the SLSA generic generator's reusable-
// workflow reference ONLY when pinned by a full vX.Y.Z semver tag —
// slsa-verifier rejects both a SHA pin and a short tag (@v2) for this
// specific reusable workflow.
var slsaGeneratorTaggedRe = regexp.MustCompile(`^slsa-framework/slsa-github-generator/\.github/workflows/generator_generic_slsa3\.yml@v\d+\.\d+\.\d+`)

// --- tests ------------------------------------------------------------

// TestDarwinLegsBuildNatively is the D-08 mitigation for T-10-03-02: every
// darwin entry in release.yml's build matrix must build NATIVELY (never
// cross-linked via zig) on a real macOS-class runner. Demonstrated red
// (see 10-03-SUMMARY.md) by setting a darwin entry's needs_zig to true, and
// separately by pointing its runner at a linux profile label.
func TestDarwinLegsBuildNatively(t *testing.T) {
	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	src := string(data)

	entries := mustReleaseBuildMatrix(t, src)

	var darwinEntries []releaseMatrixEntry
	for _, e := range entries {
		if e.Goos == "darwin" {
			darwinEntries = append(darwinEntries, e)
		}
	}
	if len(darwinEntries) == 0 {
		t.Fatalf("TestDarwinLegsBuildNatively: no goos: darwin entries found in release.yml's build matrix")
	}

	for _, e := range darwinEntries {
		if e.NeedsZig {
			t.Errorf("darwin/%s: needs_zig = true, want false — a darwin leg must never cross-link via zig (D-08, libresolv/DNS-resolver risk)", e.Goarch)
		}
		if !isMacOSClassRunner(e.Runner) {
			t.Errorf("darwin/%s: runner = %q is not a recognized macOS-class runner label — a darwin leg must build on a real macOS host, never a linux profile", e.Goarch, e.Runner)
		}
	}
}

// TestProvenanceJobUsesTaggedSLSAGenerator is the D-07 mitigation for
// T-10-03-04: the provenance job must reference the SLSA generic generator
// by a full vX.Y.Z tag (never a SHA — slsa-verifier requires the tag form)
// and must declare no runs-on: of its own, since a reusable-workflow caller
// cannot override the callee's runner.
func TestProvenanceJobUsesTaggedSLSAGenerator(t *testing.T) {
	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	src := string(data)

	shape := mustReleaseProvenanceJob(t, src)

	if !slsaGeneratorTaggedRe.MatchString(shape.Uses) {
		t.Errorf("provenance job uses: %q, want the SLSA generic generator referenced by a full vX.Y.Z tag", shape.Uses)
	}
	if shape.HasRunsOn {
		t.Errorf("provenance job declares its own runs-on: — a reusable-workflow caller cannot override the callee's runner (D-07); this job must have none")
	}
}

// TestReleaseWorkflowFileMatchesPattern is the T-09-01-01 spoofing guard:
// it reads the real, on-disk release.yml, reconstructs the SAN a tag push
// would produce, and asserts releaseWorkflowRefPattern accepts it — plus
// two reject subtests proving the pattern still rejects a renamed-workflow
// SAN and a branch-ref SAN. Non-vacuous in both directions (see this task's
// SUMMARY for the observed break-observe-restore output).
func TestReleaseWorkflowFileMatchesPattern(t *testing.T) {
	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	src := string(data)

	re := regexp.MustCompile(releaseWorkflowRefPattern)
	baseName := filepath.Base(releaseWorkflowPath)
	san := "https://github.com/" + releaseRepoSlug + "/.github/workflows/" + baseName + "@refs/tags/v1.2.3"

	// A drift in the on-disk name: value is reported here, non-fatally, so
	// the reconstructed SAN below is always visible in the failure output —
	// diagnosable from test output alone, per the plan's non-vacuity
	// requirement.
	if name, nameErr := parseWorkflowTopLevelName(src); nameErr != nil {
		t.Errorf("parseWorkflowTopLevelName: %v (reconstructed SAN: %q)", nameErr, san)
	} else if name != "release" {
		t.Errorf("release.yml's top-level name: = %q, want %q (reconstructed SAN: %q)", name, "release", san)
	}

	t.Run("accept", func(t *testing.T) {
		if !re.MatchString(san) {
			t.Fatalf("releaseWorkflowRefPattern must accept release.yml's real tag-push SAN, got no match for: %q", san)
		}
	})

	t.Run("reject_renamed_workflow", func(t *testing.T) {
		san := "https://github.com/" + releaseRepoSlug + "/.github/workflows/release-please.yml@refs/tags/v1.2.3"
		if re.MatchString(san) {
			t.Fatalf("releaseWorkflowRefPattern must reject a different workflow filename in the same repo at a tag ref, matched: %q", san)
		}
	})

	t.Run("reject_branch_ref", func(t *testing.T) {
		san := "https://github.com/" + releaseRepoSlug + "/.github/workflows/" + baseName + "@refs/heads/main"
		if re.MatchString(san) {
			t.Fatalf("releaseWorkflowRefPattern must reject release.yml at a branch ref, matched: %q", san)
		}
	})
}

// TestReleaseWorkflowTriggerIsTagPushOnly is the mechanical enforcement of
// release.yml's header claim that it triggers only on tag pushes. D-03's
// documented workflow_dispatch fallback trigger cannot be added without this
// test going red — it would have to land in the same change as an update to
// this test and to release.yml's header comment (RESEARCH Pitfall 3).
func TestReleaseWorkflowTriggerIsTagPushOnly(t *testing.T) {
	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	src := string(data)

	keys := mustWorkflowOnKeys(t, src)
	if len(keys) != 1 || keys[0] != "push" {
		t.Fatalf("release.yml's on: block top-level keys = %v, want exactly [push]", keys)
	}

	patterns := mustWorkflowPushTagPatterns(t, src)
	if len(patterns) != 1 || patterns[0] != "v[0-9]*" {
		t.Fatalf("release.yml's on.push.tags patterns = %v, want exactly [v[0-9]*]", patterns)
	}
}

// TestWorkflowSourceHelpersFailLoudly is the T-09-01-07 repudiation guard:
// every pure parse core must return a non-nil error — never a usable zero
// value — when its target is absent from the source, across all four
// helper pairs.
func TestWorkflowSourceHelpersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "parseWorkflowTopLevelName: missing column-0 name: key",
			fn: func() error {
				_, err := parseWorkflowTopLevelName("on:\n  push:\n    branches: [main]\n")
				return err
			},
		},
		{
			name: "parseWorkflowOnKeys: missing on: key entirely",
			fn: func() error {
				_, err := parseWorkflowOnKeys("name: example\njobs:\n  build:\n    runs-on: ubuntu-latest\n")
				return err
			},
		},
		{
			name: "parseWorkflowPushTagPatterns: on: block has no push: child",
			fn: func() error {
				_, err := parseWorkflowPushTagPatterns("name: example\non:\n  pull_request:\n")
				return err
			},
		},
		{
			name: "parseWorkflowPushTagPatterns: tags: list has zero items",
			fn: func() error {
				_, err := parseWorkflowPushTagPatterns("name: example\non:\n  push:\n    tags:\njobs:\n  build:\n    runs-on: ubuntu-latest\n")
				return err
			},
		},
		{
			name: "parseWorkflowStepRunBlock: named step exists but has no run block",
			fn: func() error {
				src := "jobs:\n  build:\n    steps:\n      - name: Checkout\n        uses: actions/checkout@v6\n"
				_, err := parseWorkflowStepRunBlock(src, "Checkout")
				return err
			},
		},
		{
			name: "parseWorkflowStepRunBlock: step name does not exist at all",
			fn: func() error {
				src := "jobs:\n  build:\n    steps:\n      - name: Checkout\n        run: |\n          echo hi\n"
				_, err := parseWorkflowStepRunBlock(src, "Does Not Exist")
				return err
			},
		},
		{
			name: "parseReleaseBuildMatrix: no matrix include: entries found",
			fn: func() error {
				_, err := parseReleaseBuildMatrix("")
				return err
			},
		},
		{
			name: "parseReleaseProvenanceJob: no provenance: job found",
			fn: func() error {
				_, err := parseReleaseProvenanceJob("name: example\njobs:\n  build:\n    runs-on: ubuntu-latest\n")
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
