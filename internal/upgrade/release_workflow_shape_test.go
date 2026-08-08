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
// `jobs:` key and returns each job's releaseJobShape. Generalizes
// parseReleaseProvenanceJob's job-boundary-scanning technique (find a job's
// start via a column-2 `<jobname>:` line, find its end via the next such
// line or EOF) to every job in the file rather than one named job. Returns
// a non-nil error — never a usable empty slice — when the file declares no
// `jobs:` key at all, or when `jobs:` has zero top-level job entries.
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

// provenanceJobIDTokenAllowance is the ONE named, staleness-checked
// temporary exception to D-11's "exactly one job holds id-token: write":
// the reusable-workflow provenance: job still declares its own
// id-token: write until plan 01-04 removes both the job and this
// allowance together. A stale allowance (naming a job that no longer
// exists in release.yml) fails TestOIDCWriteScopedToSingleGoreleaserJob
// rather than silently widening it.
const provenanceJobIDTokenAllowance = "provenance"

// TestOIDCWriteScopedToSingleGoreleaserJob is the D-11 mitigation for
// T-01-11: scans every top-level job in release.yml and asserts that
// id-token: write is held ONLY by the job that invokes goreleaser and (as a
// named, temporary, staleness-checked allowance) the provenance: job — no
// third holder, and the goreleaser job itself must be a holder. Demonstrated
// red by adding id-token: write to a throwaway scratch job (a THIRD holder;
// post-collapse the file has only these two, so a second holder is the
// expected temporary allowance and would not go red).
func TestOIDCWriteScopedToSingleGoreleaserJob(t *testing.T) {
	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	src := string(data)

	shapes := mustReleaseJobShapes(t, src)
	goreleaserJob := mustGoreleaserInvokingJob(t, src)

	allowanceExists := false
	for _, s := range shapes {
		if s.ID == provenanceJobIDTokenAllowance {
			allowanceExists = true
			break
		}
	}
	if !allowanceExists {
		t.Fatalf("provenanceJobIDTokenAllowance names job %q, which no longer exists in release.yml — this allowance is stale and must be removed (plan 01-04)", provenanceJobIDTokenAllowance)
	}

	allowed := map[string]bool{goreleaserJob.ID: true, provenanceJobIDTokenAllowance: true}
	var holders []string
	var unexpected []string
	for _, s := range shapes {
		if !s.HasIDTokenWrite {
			continue
		}
		holders = append(holders, s.ID)
		if !allowed[s.ID] {
			unexpected = append(unexpected, s.ID)
		}
	}

	if len(holders) == 0 {
		t.Fatalf("no job in release.yml declares id-token: write — no signing identity is possible")
	}
	if len(unexpected) > 0 {
		t.Errorf("id-token: write held by unexpected job(s) %v — only the goreleaser job (%q) and the temporary provenance: allowance may hold it (D-11)", unexpected, goreleaserJob.ID)
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
			name: "parseReleaseJobShapes: no top-level jobs: key found",
			fn: func() error {
				_, err := parseReleaseJobShapes("")
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
