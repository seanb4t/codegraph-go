package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
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

// releaseJobShape is the subset of one top-level release.yml job's YAML
// this package cares about post-collapse (D-11, D-13): its runner, whether
// it holds the OIDC signing permission, and whether it is the job that
// invokes goreleaser. Introduced by plan 01-03 to replace the deleted
// per-matrix-leg releaseMatrixEntry now that the build matrix is gone.
type releaseJobShape struct {
	ID                   string
	Name                 string
	RunsOn               string
	HasIDTokenWrite      bool
	HasAttestationsWrite bool
	InvokesGoreleaser    bool
}

var releaseJobsBlockRe = regexp.MustCompile(`^jobs:\s*$`)
var releaseTopLevelJobRe = regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)
var releaseJobNameRe = regexp.MustCompile(`^    name:\s*(.+)$`)
var releaseJobRunsOnRe = regexp.MustCompile(`^    runs-on:\s*(.+)$`)
var releaseJobIDTokenWriteRe = regexp.MustCompile(`id-token:\s*write`)
var releaseJobAttestationsWriteRe = regexp.MustCompile(`attestations:\s*write`)

// releaseJobInvokesGoreleaserRe matches either the Taskfile indirection
// (`task release:goreleaser`) or a bare `goreleaser ` invocation, so this
// guard does not silently pass if the Taskfile indirection is later
// removed. Deliberately case-sensitive and whitespace-anchored so it does
// not false-positive on prose like "Install GoReleaser" or on the
// goreleaser-action `uses:` reference (`goreleaser/goreleaser-action`,
// where "goreleaser" is never followed by whitespace).
var releaseJobInvokesGoreleaserRe = regexp.MustCompile(`task release:goreleaser|\bgoreleaser\s`)

// parseReleaseJobShapes walks every top-level job under release.yml's
// `jobs:` key and returns each job's releaseJobShape, using a
// job-boundary-scanning technique (find a job's start via a column-2
// `<jobname>:` line, find its end via the next such line or EOF) shared by
// parseAttestStep below. Returns a non-nil error — never a usable empty
// slice — when the file declares no `jobs:` key at all, or when `jobs:` has
// zero top-level job entries.
func parseReleaseJobShapes(src string) ([]releaseJobShape, error) {
	lines := strings.Split(src, "\n")

	jobsIdx := -1
	for i, line := range lines {
		if releaseJobsBlockRe.MatchString(line) {
			jobsIdx = i
			break
		}
	}
	if jobsIdx == -1 {
		return nil, fmt.Errorf("parseReleaseJobShapes: no top-level jobs: key found")
	}

	var starts []int
	var ids []string
	for i := jobsIdx + 1; i < len(lines); i++ {
		if m := releaseTopLevelJobRe.FindStringSubmatch(lines[i]); m != nil {
			starts = append(starts, i)
			ids = append(ids, m[1])
		}
	}
	if len(starts) == 0 {
		return nil, fmt.Errorf("parseReleaseJobShapes: jobs: key has no top-level job entries")
	}

	shapes := make([]releaseJobShape, 0, len(starts))
	for idx, start := range starts {
		end := len(lines)
		if idx+1 < len(starts) {
			end = starts[idx+1]
		}
		shape := releaseJobShape{ID: ids[idx]}
		for i := start + 1; i < end; i++ {
			line := lines[i]
			if m := releaseJobNameRe.FindStringSubmatch(line); m != nil && shape.Name == "" {
				shape.Name = strings.Trim(strings.TrimSpace(m[1]), `"'`)
			}
			if m := releaseJobRunsOnRe.FindStringSubmatch(line); m != nil && shape.RunsOn == "" {
				shape.RunsOn = strings.Trim(strings.TrimSpace(m[1]), `"'`)
			}
			if releaseJobIDTokenWriteRe.MatchString(line) {
				shape.HasIDTokenWrite = true
			}
			if releaseJobAttestationsWriteRe.MatchString(line) {
				shape.HasAttestationsWrite = true
			}
			if releaseJobInvokesGoreleaserRe.MatchString(line) {
				shape.InvokesGoreleaser = true
			}
		}
		shapes = append(shapes, shape)
	}
	return shapes, nil
}

func mustReleaseJobShapes(t *testing.T, src string) []releaseJobShape {
	t.Helper()
	v, err := parseReleaseJobShapes(src)
	if err != nil {
		t.Fatalf("mustReleaseJobShapes: %v", err)
	}
	return v
}

// parseGoreleaserInvokingJob returns the one job whose InvokesGoreleaser is
// true. Returns a non-nil error — never a usable zero value — when zero
// jobs invoke goreleaser (no signing identity possible) or when more than
// one does (the topology this plan's collapse is supposed to prevent).
func parseGoreleaserInvokingJob(src string) (releaseJobShape, error) {
	shapes, err := parseReleaseJobShapes(src)
	if err != nil {
		return releaseJobShape{}, fmt.Errorf("parseGoreleaserInvokingJob: %w", err)
	}
	var matches []releaseJobShape
	for _, s := range shapes {
		if s.InvokesGoreleaser {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return releaseJobShape{}, fmt.Errorf("parseGoreleaserInvokingJob: no job invokes goreleaser (neither 'task release:goreleaser' nor a bare 'goreleaser ' invocation found in any job)")
	case 1:
		return matches[0], nil
	default:
		var ids []string
		for _, m := range matches {
			ids = append(ids, m.ID)
		}
		return releaseJobShape{}, fmt.Errorf("parseGoreleaserInvokingJob: %d jobs invoke goreleaser (%v), want exactly 1", len(matches), ids)
	}
}

func mustGoreleaserInvokingJob(t *testing.T, src string) releaseJobShape {
	t.Helper()
	v, err := parseGoreleaserInvokingJob(src)
	if err != nil {
		t.Fatalf("mustGoreleaserInvokingJob: %v", err)
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

// attestStepShape is the subset of the `actions/attest-build-provenance`
// step's YAML this package cares about (plan 01-04, D-09/D-11/D-12): its
// full `uses:` reference (for SHA-pin verification), the id of the job it
// lives in (must be the goreleaser-invoking job), and its
// `subject-checksums:` input (cross-file compared against
// .goreleaser.yaml's checksum.name_template — review Codex Plan-04 LOW,
// C8). Replaces the prior provenance-job shape type now that the SLSA
// generic generator's reusable-workflow `provenance:` job is gone entirely.
type attestStepShape struct {
	Uses             string
	JobID            string
	SubjectChecksums string
}

var attestStepUsesRe = regexp.MustCompile(`uses:\s*(actions/attest-build-provenance\S*)`)
var attestStepSubjectChecksumsRe = regexp.MustCompile(`^\s*subject-checksums:\s*(.+)$`)

// attestActionPinnedRe asserts actions/attest-build-provenance is pinned to
// a full 40-hex commit SHA — never a bare tag, never a branch — mirroring
// the style of the prior SLSA-generator tag-pin regex it replaces
// (regex-on-a-uses:-string) but asserting a SHA pin instead of a tag pin:
// this Action follows the SAME SHA-pinning convention as every other
// third-party Action in release.yml, now unconditional (D-09 — the SLSA
// generator's documented tag-pin exception left with it).
var attestActionPinnedRe = regexp.MustCompile(`^actions/attest-build-provenance@[0-9a-f]{40}$`)

// parseAttestStep scans every top-level job in release.yml (the same
// job-boundary technique parseReleaseJobShapes uses: find a job's start via
// a column-2 `<jobname>:` line, its end via the next such line or EOF) for
// the ONE step whose `uses:` references actions/attest-build-provenance,
// and returns its `uses:` reference, the id of the job it lives in, and its
// `subject-checksums:` input value. Returns a non-nil error — never a
// usable zero value — when no such step exists anywhere in the file, or
// when more than one does.
func parseAttestStep(src string) (attestStepShape, error) {
	lines := strings.Split(src, "\n")

	jobsIdx := -1
	for i, line := range lines {
		if releaseJobsBlockRe.MatchString(line) {
			jobsIdx = i
			break
		}
	}
	if jobsIdx == -1 {
		return attestStepShape{}, fmt.Errorf("parseAttestStep: no top-level jobs: key found")
	}

	var starts []int
	var ids []string
	for i := jobsIdx + 1; i < len(lines); i++ {
		if m := releaseTopLevelJobRe.FindStringSubmatch(lines[i]); m != nil {
			starts = append(starts, i)
			ids = append(ids, m[1])
		}
	}
	if len(starts) == 0 {
		return attestStepShape{}, fmt.Errorf("parseAttestStep: jobs: key has no top-level job entries")
	}

	var matches []attestStepShape
	for idx, start := range starts {
		end := len(lines)
		if idx+1 < len(starts) {
			end = starts[idx+1]
		}
		for i := start + 1; i < end; i++ {
			m := attestStepUsesRe.FindStringSubmatch(lines[i])
			if m == nil {
				continue
			}
			shape := attestStepShape{Uses: m[1], JobID: ids[idx]}
			// Scan forward within this job for the step's
			// subject-checksums: input, bounded by the next step's `- `
			// start or this job's end.
			for j := i + 1; j < end; j++ {
				line := lines[j]
				if sm := attestStepSubjectChecksumsRe.FindStringSubmatch(line); sm != nil {
					shape.SubjectChecksums = strings.Trim(strings.TrimSpace(sm[1]), `"'`)
				}
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "- name:") || strings.HasPrefix(trimmed, "- uses:") {
					break // next step started
				}
			}
			matches = append(matches, shape)
		}
	}

	switch len(matches) {
	case 0:
		return attestStepShape{}, fmt.Errorf("parseAttestStep: no step uses: actions/attest-build-provenance found in release.yml")
	case 1:
		return matches[0], nil
	default:
		var uses []string
		for _, m := range matches {
			uses = append(uses, m.Uses)
		}
		return attestStepShape{}, fmt.Errorf("parseAttestStep: %d steps use actions/attest-build-provenance (%v), want exactly 1", len(matches), uses)
	}
}

func mustAttestStep(t *testing.T, src string) attestStepShape {
	t.Helper()
	v, err := parseAttestStep(src)
	if err != nil {
		t.Fatalf("mustAttestStep: %v", err)
	}
	return v
}

// --- tests ------------------------------------------------------------

// TestDarwinLegsBuildNatively is the D-08/D-13 mitigation for T-10-03-02,
// REWRITTEN IN PLACE (name preserved per D-13) for the post-collapse
// single-job topology: the job that invokes goreleaser must run on a
// macOS-class runner (darwin builds natively, never cross-linked), and
// both linux build ids in .goreleaser.yaml must carry a zig cc override
// (delegated to plan 01-01's package-level parseGoreleaserBuildEnv, so
// there is one implementation of that half of the invariant). Demonstrated
// red by flipping the goreleaser job's runs-on: to a linux profile label.
func TestDarwinLegsBuildNatively(t *testing.T) {
	releaseData, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	releaseSrc := string(releaseData)

	goreleaserJob := mustGoreleaserInvokingJob(t, releaseSrc)
	if !isMacOSClassRunner(goreleaserJob.RunsOn) {
		t.Errorf("the goreleaser-invoking job (%q) runs on %q, which is not a recognized macOS-class runner label — darwin must build natively, never cross-linked (D-08)", goreleaserJob.ID, goreleaserJob.RunsOn)
	}

	goreleaserData, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	goreleaserSrc := string(goreleaserData)

	for _, id := range []string{"codegraph-linux-amd64", "codegraph-linux-arm64"} {
		env := mustGoreleaserBuildEnv(t, goreleaserSrc, id)
		if cc, ok := env["CC"]; !ok || !strings.HasPrefix(cc, "zig cc") {
			t.Errorf("%s: CC = %q, want a zig cc override (D-01/D-02)", id, cc)
		}
	}
}

// TestOIDCWriteScopedToSingleGoreleaserJob is the D-11 mitigation for
// T-01-11: scans every top-level job in release.yml and asserts that
// id-token: write is held ONLY by the job that invokes goreleaser — no
// second holder, no allowance, and the goreleaser job itself must be a
// holder. Plan 01-03's temporary provenance:-job allowance is REMOVED here
// (plan 01-04): the `provenance:` job it named is deleted entirely, so the
// allowance would now be permanently stale. Demonstrated red by adding
// id-token: write to a throwaway scratch job (a SECOND holder; post-01-04
// the file has only one, so any second holder is unexpected).
func TestOIDCWriteScopedToSingleGoreleaserJob(t *testing.T) {
	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	src := string(data)

	shapes := mustReleaseJobShapes(t, src)
	goreleaserJob := mustGoreleaserInvokingJob(t, src)

	var holders []string
	var unexpected []string
	for _, s := range shapes {
		if !s.HasIDTokenWrite {
			continue
		}
		holders = append(holders, s.ID)
		if s.ID != goreleaserJob.ID {
			unexpected = append(unexpected, s.ID)
		}
	}

	if len(holders) == 0 {
		t.Fatalf("no job in release.yml declares id-token: write — no signing identity is possible")
	}
	if len(unexpected) > 0 {
		t.Errorf("id-token: write held by unexpected job(s) %v — only the goreleaser job (%q) may hold it, with no allowance (D-11)", unexpected, goreleaserJob.ID)
	}

	goreleaserJobHolds := false
	for _, h := range holders {
		if h == goreleaserJob.ID {
			goreleaserJobHolds = true
			break
		}
	}
	if !goreleaserJobHolds {
		t.Errorf("the goreleaser-invoking job %q does not declare id-token: write — it must be the job that mints the signing OIDC token", goreleaserJob.ID)
	}
}

// stripFullLineShellComments removes every line whose first non-whitespace
// character is '#' — load-bearing for TestNoHandRolledChecksumStepInReleaseWorkflow:
// without it, this file's own explanatory prose about the removed step
// would keep the check permanently red.
func stripFullLineShellComments(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// TestNoHandRolledChecksumStepInReleaseWorkflow is the REL-07 mitigation:
// after stripping full-line comments, release.yml must contain no
// sha256sum or shasum -a 256 invocation — GoReleaser's checksum: pipe (this
// plan's Task 2 + plan 01-02's Task 3, JOINT criterion) is the only writer
// of the checksums file. Demonstrated red by re-inserting a
// `sha256sum codegraph_* > out.txt` line.
func TestNoHandRolledChecksumStepInReleaseWorkflow(t *testing.T) {
	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	stripped := stripFullLineShellComments(string(data))

	shaRe := regexp.MustCompile(`sha256sum|shasum -a 256`)
	if shaRe.MatchString(stripped) {
		t.Errorf("release.yml contains a hand-rolled checksum invocation (sha256sum or shasum -a 256) after comment-stripping — GoReleaser's checksum: pipe must be the ONLY writer of the checksums file (REL-07)")
	}
}

// TestNoGoreleaserHooksInReleaseConfig is the review HIGH-3 mitigation:
// .goreleaser.yaml, decoded with the real YAML decoder (per plan 01-01's
// <parser_strategy>), must declare no top-level hooks: key, no top-level
// before: key, and no hooks: key on any builds: entry. After the collapse,
// .goreleaser.yaml executes inside the ONE job able to mint an OIDC token
// whose SAN internal/upgrade/verify.go:44 unconditionally trusts — a
// hooks.before entry would be an arbitrary shell command inside that
// signing boundary. Demonstrated red by adding a top-level
// `hooks:\n  before:\n    - go mod tidy` block.
func TestNoGoreleaserHooksInReleaseConfig(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal(%s): %v", goreleaserConfigPath, err)
	}

	if _, ok := cfg["hooks"]; ok {
		t.Errorf("%s declares a top-level hooks: key — arbitrary commands must not execute inside the OIDC-bearing job (review HIGH-3)", goreleaserConfigPath)
	}
	if _, ok := cfg["before"]; ok {
		t.Errorf("%s declares a top-level before: key — arbitrary commands must not execute inside the OIDC-bearing job (review HIGH-3)", goreleaserConfigPath)
	}

	buildsRaw, ok := cfg["builds"]
	if !ok {
		t.Fatalf("%s declares no builds: key", goreleaserConfigPath)
	}
	builds, ok := buildsRaw.([]any)
	if !ok {
		t.Fatalf("%s builds: is not a list", goreleaserConfigPath)
	}
	for i, b := range builds {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		if _, ok := bm["hooks"]; ok {
			t.Errorf("%s builds[%d] declares a hooks: key — arbitrary commands must not execute inside the OIDC-bearing job (review HIGH-3)", goreleaserConfigPath, i)
		}
	}
}

// TestParseReleaseJobShapes_NoJobsIsError is the non-vacuity companion:
// parseReleaseJobShapes must return a non-nil error, never a usable empty
// slice, when its source declares no jobs: key at all.
func TestParseReleaseJobShapes_NoJobsIsError(t *testing.T) {
	_, err := parseReleaseJobShapes("")
	if err == nil {
		t.Fatalf(`parseReleaseJobShapes("") = nil error, want non-nil`)
	}
}

// TestProvenanceAttestorIsPinnedNativeAction is the D-09/D-11/D-12
// mitigation REPLACING TestProvenanceJobUsesTaggedSLSAGenerator: the SLSA
// generic generator's reusable workflow and its `provenance:` job are gone
// entirely; actions/attest-build-provenance runs as a SHA-pinned step
// inside the SAME job that invokes goreleaser; that job declares
// attestations: write; and the step's subject-checksums: input names the
// EXACT SAME concrete file .goreleaser.yaml's checksum.name_template
// resolves to — proven by resolving both sides against one pinned tag
// literal with the real template engine, not by a loose pattern match
// (review Codex Plan-04 LOW, C8). Demonstrated red by three mutations
// recorded in the plan SUMMARY: a bare-tag uses: pin, the attest step moved
// to a non-goreleaser job, and a mismatched .goreleaser.yaml checksum stem.
func TestProvenanceAttestorIsPinnedNativeAction(t *testing.T) {
	releaseData, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	releaseSrc := string(releaseData)
	strippedRelease := stripFullLineShellComments(releaseSrc)

	if strings.Contains(strippedRelease, "slsa-framework/slsa-github-generator") {
		t.Errorf("release.yml still references slsa-framework/slsa-github-generator after comment-stripping — the third-party reusable workflow must be gone entirely (D-09)")
	}

	provenanceJobRe := regexp.MustCompile(`(?m)^  provenance:\s*$`)
	if provenanceJobRe.MatchString(strippedRelease) {
		t.Errorf("release.yml still declares a top-level provenance: job — it must be deleted entirely, replaced by a step inside the goreleaser job (D-09)")
	}

	step := mustAttestStep(t, releaseSrc)

	if !attestActionPinnedRe.MatchString(step.Uses) {
		t.Errorf("attest step uses: %q, want actions/attest-build-provenance pinned to a full 40-hex commit SHA — never a bare tag or branch", step.Uses)
	}

	// Confirm the pin also carries the trailing version-tag comment
	// convention every other third-party Action in this file follows (the
	// header's SHA-pinning statement is now unconditional — D-09's note
	// that the SLSA generator's tag-pin exception left with it). The
	// attestStepUsesRe capture is space-terminated and never includes
	// trailing comment text, so this reads the raw source directly instead.
	pinCommentRe := regexp.MustCompile(`uses:\s*actions/attest-build-provenance@[0-9a-f]{40}\s*#\s*v\d+\.\d+\.\d+`)
	if !pinCommentRe.MatchString(releaseSrc) {
		t.Errorf("release.yml's actions/attest-build-provenance step has no trailing # vX.Y.Z version comment, unlike every other pinned Action in this file")
	}

	goreleaserJob, err := parseGoreleaserInvokingJob(releaseSrc)
	if err != nil {
		t.Fatalf("parseGoreleaserInvokingJob: %v", err)
	}
	if step.JobID != goreleaserJob.ID {
		t.Errorf("attest step lives in job %q, want the goreleaser-invoking job %q (D-11: one job, one signing identity)", step.JobID, goreleaserJob.ID)
	}
	if !goreleaserJob.HasAttestationsWrite {
		t.Errorf("job %q (the goreleaser job) does not declare attestations: write — required by GitHub's Attestations API for actions/attest-build-provenance", goreleaserJob.ID)
	}

	if step.SubjectChecksums == "" {
		t.Fatalf("attest step has no subject-checksums: input")
	}

	// Cross-file (review Codex Plan-04 LOW, C8): resolve the workflow's
	// subject-checksums: value and .goreleaser.yaml's checksum.name_template
	// against the SAME pinned tag literal, using the real template engine
	// (resolveGoreleaserFieldTemplate, shared with goreleaser_shape_test.go's
	// "assert the property, not the literal" discipline), and assert they
	// name the SAME concrete file. .goreleaser.yaml is the source of truth;
	// the workflow is the side that can silently drift.
	goreleaserData, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	goreleaserSrc := string(goreleaserData)

	checksumBlock := mustGoreleaserTopLevelBlock(t, goreleaserSrc, "checksum")
	nameTemplateRaw, ok := checksumBlock["name_template"]
	if !ok {
		t.Fatalf("%s's checksum: block has no name_template: key", goreleaserConfigPath)
	}
	nameTemplate, ok := nameTemplateRaw.(string)
	if !ok {
		t.Fatalf("%s's checksum.name_template is %T, not a string", goreleaserConfigPath, nameTemplateRaw)
	}

	wantFilename, err := resolveGoreleaserFieldTemplate(nameTemplate, map[string]any{
		"ProjectName": "codegraph",
		"Tag":         pinnedReleaseTag,
	})
	if err != nil {
		t.Fatalf("resolve checksum.name_template: %v", err)
	}

	// The workflow side uses GitHub Actions expression syntax
	// (${{ github.ref_name }}), not Go-template syntax — substitute the
	// SAME pinned tag literal, then compare basenames so a "./dist/" prefix
	// difference (this step's path vs. GoReleaser's own dist/-relative
	// output path) cannot cause a false mismatch.
	resolvedSubject := strings.ReplaceAll(step.SubjectChecksums, "${{ github.ref_name }}", pinnedReleaseTag)
	gotFilename := filepath.Base(resolvedSubject)

	if gotFilename != wantFilename {
		t.Errorf("attest step's subject-checksums: resolves to %q, .goreleaser.yaml's checksum.name_template resolves to %q — want the SAME file (review Codex Plan-04 LOW / C8)", gotFilename, wantFilename)
	}
}

// TestParseAttestStep_NoAttestStepIsError is the non-vacuity companion:
// parseAttestStep must return a non-nil error, never a usable zero value,
// when its source declares no actions/attest-build-provenance step at all.
func TestParseAttestStep_NoAttestStepIsError(t *testing.T) {
	_, err := parseAttestStep("")
	if err == nil {
		t.Fatalf(`parseAttestStep("") = nil error, want non-nil`)
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
			name: "parseReleaseJobShapes: no top-level jobs: key found",
			fn: func() error {
				_, err := parseReleaseJobShapes("")
				return err
			},
		},
		{
			name: "parseAttestStep: no attest step found",
			fn: func() error {
				_, err := parseAttestStep("name: example\njobs:\n  build:\n    runs-on: ubuntu-latest\n")
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

// --- plan 02-06 Task 1: Apple secrets scoping ------------------------------
//
// appleCredentialNames is the exact set of five secret names T-02-16 scopes
// to exactly one job in exactly one workflow (release.yml's release job).
// Defined once so the literal set is not duplicated across this test's
// traversal and Taskfile.yml's own precondition guards.
var appleCredentialNames = []string{
	"MACOS_SIGN_P12",
	"MACOS_SIGN_PASSWORD",
	"MACOS_NOTARY_ISSUER_ID",
	"MACOS_NOTARY_KEY_ID",
	"MACOS_NOTARY_KEY",
}

// fullWorkflowStep, fullWorkflowJob and fullWorkflowDoc decode a workflow
// file's full shape with the REAL YAML decoder (never a line scanner) —
// richer than workflowRunStep/workflowJobYAML/workflowFileYAML
// (taskfile_shape_test.go), which only capture name:/run:. These add env:,
// permissions:, uses:, and with: (specifically with.ref:), and use
// map[string]any rather than map[string]string for env:/permissions:/with:
// values because a workflow env: value or a checkout `fetch-depth: 0` is
// not always a YAML string scalar — this test only ever inspects KEYS
// (env:/permissions:) or stringifies one known scalar (with.ref:), never
// requires a typed value.
//
// On: is decoded as a raw yaml.Node, not resolved into a Go bool/string:
// struct-field decoding matches the literal source key "on" against the
// `yaml:"on"` tag directly (verified empirically against this exact
// decoder before use — struct-tag matching does not hit YAML 1.1's
// implicit on/off-as-bool resolution, which only bites map[string]any key
// decoding).
type fullWorkflowStep struct {
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	Env  map[string]any `yaml:"env"`
	With map[string]any `yaml:"with"`
}

type fullWorkflowJob struct {
	Env         map[string]any     `yaml:"env"`
	Permissions map[string]any     `yaml:"permissions"`
	Steps       []fullWorkflowStep `yaml:"steps"`
}

type fullWorkflowDoc struct {
	On          yaml.Node                 `yaml:"on"`
	Env         map[string]any             `yaml:"env"`
	Permissions map[string]any             `yaml:"permissions"`
	Jobs        map[string]fullWorkflowJob `yaml:"jobs"`
}

// decodeFullWorkflowDoc reads and decodes one workflow YAML file at path.
// Returns a non-nil error — never a usable zero value — on a read failure,
// a YAML parse failure, or a file declaring zero jobs: entries (the CR-01
// defect class every parser in this package guards against).
func decodeFullWorkflowDoc(path string) (fullWorkflowDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fullWorkflowDoc{}, err
	}
	var doc fullWorkflowDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fullWorkflowDoc{}, fmt.Errorf("yaml.Unmarshal(%s): %w", path, err)
	}
	if len(doc.Jobs) == 0 {
		return fullWorkflowDoc{}, fmt.Errorf("decodeFullWorkflowDoc(%s): declares zero jobs:", path)
	}
	return doc, nil
}

// workflowOnTriggers returns the set of trigger-event names named directly
// under a workflow's on: key, handling all three shapes GitHub Actions
// allows there: a bare scalar (`on: push`), a flow/block sequence
// (`on: [push, pull_request]`), and a mapping
// (`on:\n  push:\n  pull_request:`).
func workflowOnTriggers(on yaml.Node) map[string]bool {
	triggers := map[string]bool{}
	switch on.Kind {
	case yaml.ScalarNode:
		triggers[on.Value] = true
	case yaml.SequenceNode:
		for _, item := range on.Content {
			triggers[item.Value] = true
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(on.Content); i += 2 {
			triggers[on.Content[i].Value] = true
		}
	}
	return triggers
}

// credentialReference names one place a credential name in
// appleCredentialNames was found: which file, which job (empty for a
// workflow-level env: reference), which step (empty for job- or
// workflow-level), and the scope it was found at.
type credentialReference struct {
	File     string
	JobID    string
	StepName string
	Scope    string // "workflow", "job", or "step"
}

// findCredentialReferences scans doc (decoded from file) for every
// reference to any name in credentials, at all three env: scopes:
// workflow-level, job-level, and step-level.
func findCredentialReferences(file string, doc fullWorkflowDoc, credentials []string) []credentialReference {
	var refs []credentialReference
	for _, cred := range credentials {
		if _, ok := doc.Env[cred]; ok {
			refs = append(refs, credentialReference{File: file, Scope: "workflow"})
		}
	}
	for jobID, job := range doc.Jobs {
		for _, cred := range credentials {
			if _, ok := job.Env[cred]; ok {
				refs = append(refs, credentialReference{File: file, JobID: jobID, Scope: "job"})
			}
		}
		for _, step := range job.Steps {
			for _, cred := range credentials {
				if _, ok := step.Env[cred]; ok {
					refs = append(refs, credentialReference{File: file, JobID: jobID, StepName: step.Name, Scope: "step"})
				}
			}
		}
	}
	return refs
}

// jobHasIDTokenWrite reports whether a decoded permissions: mapping (job- or
// workflow-level) declares id-token: write. Values are stringified via
// fmt.Sprintf since GitHub Actions permission values are always the plain
// scalars read/write/none, but the decoded map is map[string]any (see the
// fullWorkflowStep doc comment for why).
func jobHasIDTokenWrite(perms map[string]any) bool {
	v, ok := perms["id-token"]
	if !ok {
		return false
	}
	return fmt.Sprintf("%v", v) == "write"
}

// workflowHasIDTokenWrite reports whether doc declares id-token: write at
// its workflow-level permissions: block or at any job-level permissions:
// block.
func workflowHasIDTokenWrite(doc fullWorkflowDoc) bool {
	if jobHasIDTokenWrite(doc.Permissions) {
		return true
	}
	for _, job := range doc.Jobs {
		if jobHasIDTokenWrite(job.Permissions) {
			return true
		}
	}
	return false
}

// TestAppleSecretsScopedToSingleReleaseJob is the T-02-16 mitigation:
// enumerates every workflow file in .github/workflows/ at RUNTIME (never a
// fixture list, so a newly added workflow is covered the day it lands —
// and fails loudly below if the directory scan finds zero files, which
// would make this whole test vacuous) and asserts, over every one of them:
//
//   - every reference to any of the five Apple credential names appears
//     ONLY in release.yml, and within it, only in the release job (the one
//     job that already holds id-token: write, per
//     TestOIDCWriteScopedToSingleGoreleaserJob);
//   - every such reference is under a STEP-level env:, never job-level or
//     workflow-level — step-level scoping is what keeps the values out of
//     every other step in the job (review suggestion, codex);
//   - no workflow whose triggers include pull_request OR
//     pull_request_target (review concern, pi LOW — BOTH are treated as
//     pull-request triggers: pull_request_target is the more dangerous
//     fork-reachable form because it runs with repository context, and
//     this repository has several workflows using it, so a detector
//     matching only the plain trigger would miss the worse case)
//     references any of the five names, or declares id-token: write;
//   - release.yml still declares id-token: write in exactly one job, so
//     this change cannot have widened T-01-11/D-11's existing invariant
//     (re-verified here independently of TestOIDCWriteScopedToSingleGoreleaserJob,
//     so this test does not silently depend on that one to catch a
//     widening).
//
// SCOPE OF PROOF (T-02-20, review concern codex MEDIUM): this test proves
// where credential NAMES are CONSUMED in workflow files — which workflow,
// which job, which scope. It does NOT and CANNOT prove the GitHub-side
// scope of the secrets themselves: whether they are configured as
// REPOSITORY-scoped, organization-scoped, or environment-scoped secrets in
// the GitHub dashboard, nor their access policies there. Those are
// dashboard facts, covered by this plan's user_setup, not by any test that
// runs against on-disk source. A reader must not over-trust a green result
// here as proof of repository-scoped secret configuration.
func TestAppleSecretsScopedToSingleReleaseJob(t *testing.T) {
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("os.ReadDir(%s): %v", workflowsDir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml") {
			files = append(files, filepath.Join(workflowsDir, e.Name()))
		}
	}
	if len(files) == 0 {
		t.Fatalf("workflow directory scan found zero files in %s — this would make the whole test vacuous", workflowsDir)
	}

	releaseSrc, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	releaseJob := mustGoreleaserInvokingJob(t, string(releaseSrc))

	releaseAbs, err := filepath.Abs(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("filepath.Abs(%s): %v", releaseWorkflowPath, err)
	}

	for _, path := range files {
		doc, err := decodeFullWorkflowDoc(path)
		if err != nil {
			t.Fatalf("decodeFullWorkflowDoc(%s): %v", path, err)
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			t.Fatalf("filepath.Abs(%s): %v", path, err)
		}
		isReleaseWorkflow := absPath == releaseAbs

		refs := findCredentialReferences(path, doc, appleCredentialNames)
		triggers := workflowOnTriggers(doc.On)
		isPRTriggerable := triggers["pull_request"] || triggers["pull_request_target"]

		for _, ref := range refs {
			if !isReleaseWorkflow {
				t.Errorf("%s references an Apple credential name outside release.yml (scope=%s job=%s step=%q) — T-02-16 requires exactly one workflow to hold these", ref.File, ref.Scope, ref.JobID, ref.StepName)
				continue
			}
			if ref.JobID != releaseJob.ID {
				t.Errorf("release.yml references an Apple credential name in job %q, want only the release job %q (scope=%s step=%q)", ref.JobID, releaseJob.ID, ref.Scope, ref.StepName)
			}
			if ref.Scope != "step" {
				t.Errorf("release.yml references an Apple credential name at %s-level env: (job=%s step=%q) — must be step-level only, so the values stay out of every other step in the job", ref.Scope, ref.JobID, ref.StepName)
			}
		}

		if isPRTriggerable {
			if len(refs) > 0 {
				t.Errorf("%s triggers on a pull-request event (pull_request or pull_request_target) AND references an Apple credential name — forbidden (T-02-16)", path)
			}
			if workflowHasIDTokenWrite(doc) {
				t.Errorf("%s triggers on a pull-request event (pull_request or pull_request_target) AND declares id-token: write — must never coincide", path)
			}
		}
	}

	shapes := mustReleaseJobShapes(t, string(releaseSrc))
	var holders []string
	for _, s := range shapes {
		if s.HasIDTokenWrite {
			holders = append(holders, s.ID)
		}
	}
	if len(holders) != 1 {
		t.Errorf("release.yml declares id-token: write in %d job(s) %v, want exactly 1", len(holders), holders)
	}
}

// TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError is the
// non-vacuity companion: decodeFullWorkflowDoc must return a non-nil error,
// never a usable zero value, for a workflow source with no jobs: entries.
func TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yml")
	if err := os.WriteFile(path, []byte("name: empty\non:\n  push:\njobs: {}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	if _, err := decodeFullWorkflowDoc(path); err == nil {
		t.Fatalf("decodeFullWorkflowDoc(%s): expected a non-nil error for a workflow with zero jobs:, got nil", path)
	}
}
