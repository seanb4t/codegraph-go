// changie_release_shape_test.go binds the changie release chain's CI wiring:
// .github/workflows/release-pr.yml (REL-10, D-01 to D-04, D-07) and the
// Taskfile.yml `release:batch` target it runs. The guards land RED first per
// rule x1cjy9vyhq: committed before release-pr.yml and release:batch exist, so
// the failure is a real assertion or read failure, not a build error. Never
// gate on `gsd-tools check tdd-red-evidence` — it parses TAP only and always
// reports INVALID_RED for `go test`; the RED evidence is the test(05-01)
// commit that precedes the ci(05-01) commit plus the pasted `--- FAIL`
// transcript in the plan's SUMMARY and 05-MUTATION-LOG.md.
//
// These tests read this repository's YAML, Taskfile bodies and config only.
// They never exercise how changie, peter-evans/create-pull-request or GitHub
// behave (rule eyva2svrcz).
//
// Plans 05-02 and 05-03 extend this file, so the YAML decoder below is
// general and every package-level identifier carries the releaseChain prefix
// (package upgrade is shared by many test files).
package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// --- fixture paths and pins (D-03, D-07, T-05-SC) ---------------------------

const releaseChainPRWorkflowPath = "../../.github/workflows/release-pr.yml"

const (
	releaseChainPinCheckout = "actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3"
	releaseChainPinSetupGo  = "actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0"
	releaseChainPinAppToken = "actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0"
	releaseChainPinCreatePR = "peter-evans/create-pull-request@5f6978faf089d4d20b00c7766989d076bb2fc7f1 # v8.1.1"

	releaseChainInstallTask = "./.github/actions/install-task"
)

// releaseChainExpectedPins is the exact set of pinned (non-local) `uses:`
// values a release workflow in this plan may carry, in sorted order.
var releaseChainExpectedPins = []string{
	releaseChainPinCheckout,
	releaseChainPinAppToken,
	releaseChainPinSetupGo,
	releaseChainPinCreatePR,
}

// releaseChainUsesLineRe matches a `uses:` line in raw workflow source,
// capturing the value that follows it up to end of line.
var releaseChainUsesLineRe = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*(.+?)\s*$`)

// releaseChainPinnedUsesRe is the only accepted shape for a non-local uses
// value: owner/repo@<40 hex>, a space, `#`, a space, vX.Y.Z.
var releaseChainPinnedUsesRe = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+@[0-9a-f]{40} # v[0-9]+\.[0-9]+\.[0-9]+$`)

// --- YAML decoding -----------------------------------------------------------

type releaseChainStep struct {
	Name string            `yaml:"name"`
	ID   string            `yaml:"id"`
	If   string            `yaml:"if"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]string `yaml:"with"`
}

type releaseChainJob struct {
	Name        string             `yaml:"name"`
	Needs       any                `yaml:"needs"`
	RunsOn      string             `yaml:"runs-on"`
	Permissions map[string]string  `yaml:"permissions"`
	Steps       []releaseChainStep `yaml:"steps"`
}

type releaseChainWorkflow struct {
	Name        string                     `yaml:"name"`
	On          yaml.Node                  `yaml:"on"`
	Permissions map[string]string          `yaml:"permissions"`
	Concurrency map[string]string          `yaml:"concurrency"`
	Jobs        map[string]releaseChainJob `yaml:"jobs"`
}

// parseReleaseChainWorkflow decodes workflow source. It returns a non-nil
// error for unparseable YAML, a document with no jobs, or a job with zero
// steps — never a usable zero value on a miss.
func parseReleaseChainWorkflow(src []byte) (*releaseChainWorkflow, error) {
	var wf releaseChainWorkflow
	if err := yaml.Unmarshal(src, &wf); err != nil {
		return nil, fmt.Errorf("parseReleaseChainWorkflow: unmarshal: %w", err)
	}
	if len(wf.Jobs) == 0 {
		return nil, fmt.Errorf("parseReleaseChainWorkflow: no jobs: found")
	}
	for id, job := range wf.Jobs {
		if len(job.Steps) == 0 {
			return nil, fmt.Errorf("parseReleaseChainWorkflow: job %q declares zero steps", id)
		}
	}
	return &wf, nil
}

func mustReleaseChainWorkflow(t *testing.T, path string) (*releaseChainWorkflow, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	wf, err := parseReleaseChainWorkflow(data)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return wf, string(data)
}

// releaseChainMappingKeys returns n's mapping keys, sorted. It fails the test
// when n is not a mapping node.
func releaseChainMappingKeys(t *testing.T, what string, n *yaml.Node) []string {
	t.Helper()
	if n.Kind != yaml.MappingNode {
		t.Fatalf("%s is not a mapping (kind=%v)", what, n.Kind)
	}
	return yamlMappingKeys(n)
}

// releaseChainNeeds normalizes a job's needs: value (a string or a list).
func releaseChainNeeds(needs any) []string {
	switch v := needs.(type) {
	case string:
		return []string{v}
	case []any:
		var out []string
		for _, e := range v {
			out = append(out, fmt.Sprint(e))
		}
		return out
	}
	return nil
}

// releaseChainNonCommentLines returns the lines of src whose first non-blank
// character is not `#`.
func releaseChainNonCommentLines(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// releaseChainStepWanted is one expected step of a job: positional fields
// the test asserts. An empty field means "must be empty".
type releaseChainStepWanted struct {
	Name string
	ID   string
	If   string
	Uses string
	Run  string
	With map[string]string
}

// releaseChainCheckSteps asserts job's steps equal want, in order. It returns
// the number of step fields it compared.
func releaseChainCheckSteps(t *testing.T, jobID string, got []releaseChainStep, want []releaseChainStepWanted) int {
	t.Helper()
	if len(got) != len(want) {
		names := make([]string, 0, len(got))
		for _, s := range got {
			names = append(names, s.Name)
		}
		t.Fatalf("job %q has %d steps %q, want %d", jobID, len(got), names, len(want))
	}
	checked := 0
	for i, w := range want {
		g := got[i]
		where := fmt.Sprintf("job %q step %d", jobID, i+1)
		if g.Name != w.Name {
			t.Errorf("%s: name = %q, want %q", where, g.Name, w.Name)
		}
		if g.ID != w.ID {
			t.Errorf("%s (%s): id = %q, want %q", where, w.Name, g.ID, w.ID)
		}
		if g.If != w.If {
			t.Errorf("%s (%s): if = %q, want %q", where, w.Name, g.If, w.If)
		}
		if g.Uses != w.Uses {
			t.Errorf("%s (%s): uses = %q, want %q", where, w.Name, g.Uses, w.Uses)
		}
		if gotRun := stripRunBodyNoise(g.Run); gotRun != w.Run {
			t.Errorf("%s (%s): run body = %q, want %q", where, w.Name, gotRun, w.Run)
		}
		wantWith := w.With
		if wantWith == nil {
			wantWith = map[string]string{}
		}
		gotWith := g.With
		if gotWith == nil {
			gotWith = map[string]string{}
		}
		if !mapsEqualReleaseChain(gotWith, wantWith) {
			t.Errorf("%s (%s): with = %v, want %v", where, w.Name, gotWith, wantWith)
		}
		checked += 6
	}
	return checked
}

func mapsEqualReleaseChain(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// --- TestReleasePRWorkflowShape (REL-10, D-01 to D-04, D-07) -----------------

func TestReleasePRWorkflowShape(t *testing.T) {
	wf, src := mustReleaseChainWorkflow(t, releaseChainPRWorkflowPath)
	checked := 0

	// The display name must never be `release`: post-release-verify.yml keys
	// on workflow_run of "release".
	if wf.Name != "release-pr" {
		t.Errorf("top-level name = %q, want %q", wf.Name, "release-pr")
	}
	checked++

	// Trigger: a push to main only (D-02); no paths filter, no pull_request.
	onKeys := releaseChainMappingKeys(t, "on:", &wf.On)
	if !slices.Equal(onKeys, []string{"push"}) {
		t.Errorf("on: keys = %v, want exactly [push]", onKeys)
	} else {
		pushNode, _ := yamlMappingValue(&wf.On, "push")
		pushKeys := releaseChainMappingKeys(t, "on.push", pushNode)
		if !slices.Equal(pushKeys, []string{"branches"}) {
			t.Errorf("on.push keys = %v, want exactly [branches] (no paths filter)", pushKeys)
		} else {
			branches, _ := yamlMappingValue(pushNode, "branches")
			var got []string
			for _, b := range branches.Content {
				got = append(got, b.Value)
			}
			if !slices.Equal(got, []string{"main"}) {
				t.Errorf("on.push.branches = %v, want exactly [main]", got)
			}
		}
	}
	checked++

	if !mapsEqualReleaseChain(wf.Permissions, map[string]string{"contents": "read"}) {
		t.Errorf("top-level permissions = %v, want exactly {contents: read}", wf.Permissions)
	}
	checked++

	if !mapsEqualReleaseChain(wf.Concurrency, map[string]string{"group": "release-pr", "cancel-in-progress": "false"}) {
		t.Errorf("concurrency = %v, want {group: release-pr, cancel-in-progress: false}", wf.Concurrency)
	}
	checked++

	var jobIDs []string
	for id := range wf.Jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)
	if !slices.Equal(jobIDs, []string{"pretag-gate", "release-pr"}) {
		t.Fatalf("jobs = %v, want exactly [pretag-gate release-pr]", jobIDs)
	}
	checked++

	// D-07: the pretag-gate job runs the four canonical steps.
	gate := wf.Jobs["pretag-gate"]
	if gate.RunsOn != "ubuntu-latest" {
		t.Errorf("pretag-gate runs-on = %q, want ubuntu-latest", gate.RunsOn)
	}
	if got := releaseChainNeeds(gate.Needs); len(got) != 0 {
		t.Errorf("pretag-gate needs = %v, want none", got)
	}
	checked += 2
	checked += releaseChainCheckSteps(t, "pretag-gate", gate.Steps, []releaseChainStepWanted{
		{Name: "Checkout", Uses: strings.SplitN(releaseChainPinCheckout, " #", 2)[0]},
		{Name: "Set up Go", Uses: strings.SplitN(releaseChainPinSetupGo, " #", 2)[0], With: map[string]string{"go-version-file": "go.mod"}},
		{Name: "Install Task", Uses: releaseChainInstallTask},
		{Name: "Run check:cross sweep", Run: "task check:cross"},
	})

	// The release-pr job: needs pretag-gate and runs the six steps in order.
	rel := wf.Jobs["release-pr"]
	if got := releaseChainNeeds(rel.Needs); !slices.Equal(got, []string{"pretag-gate"}) {
		t.Errorf("release-pr needs = %v, want [pretag-gate]", got)
	}
	if rel.RunsOn != "ubuntu-latest" {
		t.Errorf("release-pr runs-on = %q, want ubuntu-latest", rel.RunsOn)
	}
	if len(rel.Permissions) != 0 {
		t.Errorf("release-pr declares job-level permissions %v, want none", rel.Permissions)
	}
	checked += 3
	checked += releaseChainCheckSteps(t, "release-pr", rel.Steps, []releaseChainStepWanted{
		{
			Name: "Mint GitHub App installation token",
			ID:   "app-token",
			Uses: strings.SplitN(releaseChainPinAppToken, " #", 2)[0],
			With: map[string]string{"app-id": "${{ secrets.APP_ID }}", "private-key": "${{ secrets.APP_PRIVATE_KEY }}"},
		},
		{Name: "Checkout", Uses: strings.SplitN(releaseChainPinCheckout, " #", 2)[0]},
		{Name: "Set up Go", Uses: strings.SplitN(releaseChainPinSetupGo, " #", 2)[0], With: map[string]string{"go-version-file": "go.mod"}},
		{Name: "Install Task", Uses: releaseChainInstallTask},
		{Name: "Batch fragments into a release", ID: "batch", Run: "task release:batch"},
		{
			Name: "Open or update the release PR",
			If:   "steps.batch.outputs.version != ''",
			Uses: strings.SplitN(releaseChainPinCreatePR, " #", 2)[0],
			With: map[string]string{
				// D-03: the App token, so the PR's pull_request checks run.
				"token":          "${{ steps.app-token.outputs.token }}",
				"branch":         "changie/release",
				"title":          "chore(main): release ${{ steps.batch.outputs.version }}",
				"commit-message": "chore(main): release ${{ steps.batch.outputs.version }}",
				"body-path":      ".changes/${{ steps.batch.outputs.version }}.md",
				// D-04: close-draft-prs.yml closes App-authored drafts.
				"draft": "false",
			},
		},
	})

	// The pinned uses set, read from raw source: every non-comment `uses:`
	// line is the local install-task action or a full-SHA pin with a version
	// comment, and the distinct pins are exactly the four expected.
	nonComment := releaseChainNonCommentLines(src)
	usesLines := 0
	pins := map[string]bool{}
	for _, line := range nonComment {
		m := releaseChainUsesLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		usesLines++
		v := m[1]
		if v == releaseChainInstallTask {
			continue
		}
		if !releaseChainPinnedUsesRe.MatchString(v) {
			t.Errorf("uses: %q is not a full 40-hex SHA pin followed by ` # vX.Y.Z`", v)
			continue
		}
		pins[v] = true
	}
	if usesLines == 0 {
		t.Fatalf("%s: found zero uses: lines — this guard would vacuously pass", releaseChainPRWorkflowPath)
	}
	var gotPins []string
	for p := range pins {
		gotPins = append(gotPins, p)
	}
	sort.Strings(gotPins)
	if !slices.Equal(gotPins, releaseChainExpectedPins) {
		t.Errorf("distinct pinned uses = %v, want exactly %v", gotPins, releaseChainExpectedPins)
	}
	checked += usesLines

	// Run bodies: exactly two, each a single task call, and no expression
	// inside any run body (fragment text must never reach a shell, T-05-03).
	var runBodies []string
	for _, id := range jobIDs {
		for _, s := range wf.Jobs[id].Steps {
			if s.Run == "" {
				continue
			}
			runBodies = append(runBodies, stripRunBodyNoise(s.Run))
			if strings.Contains(s.Run, "${{") {
				t.Errorf("job %q step %q: run body contains a ${{ }} expression: %q", id, s.Name, s.Run)
			}
		}
	}
	sort.Strings(runBodies)
	if !slices.Equal(runBodies, []string{"task check:cross", "task release:batch"}) {
		t.Errorf("run bodies = %v, want exactly [task check:cross task release:batch]", runBodies)
	}
	checked += len(runBodies)

	// D-03 / T-05-01: the default token must not appear on any non-comment
	// line (comments may explain the hazard).
	for i, line := range nonComment {
		if strings.Contains(line, "GITHUB_TOKEN") || strings.Contains(line, "github.token") {
			t.Errorf("non-comment line %d names the default token (D-03): %q", i+1, line)
		}
	}
	checked += len(nonComment)

	if checked == 0 {
		t.Fatalf("checked zero items — this guard would vacuously pass")
	}
	t.Logf("%s: %d fields, lines and uses checked (%d uses: lines, %d distinct pins, %d run bodies)", releaseChainPRWorkflowPath, checked, usesLines, len(gotPins), len(runBodies))
}

// --- TestReleaseBatchTargetShape (REL-10, D-02, C1, C2) ---------------------

// releaseChainBatchTokens are the seven tokens of release:batch's body, in
// the order they must appear (C1, C2, D-02).
var releaseChainBatchTokens = []struct {
	label string
	token string
}{
	{"fragment count", `find .changes/unreleased -maxdepth 1 -type f -name '*.yaml'`},
	{"zero-count exit 0", `exit 0`},
	{"batch", `{{.GO_TOOL_CHANGIE}} changie batch auto`},
	{"merge", `{{.GO_TOOL_CHANGIE}} changie merge`},
	{"latest", `{{.GO_TOOL_CHANGIE}} changie latest`},
	{"skip-ci phrase scan", `\[(skip ci|ci skip|no ci|skip actions|actions skip)\]`},
	{"version output", `"version=${version}" >> "${GITHUB_OUTPUT}"`},
}

// releaseChainChangieVerbRe matches a changie invocation of a verb release:batch
// may use.
var releaseChainChangieVerbRe = regexp.MustCompile(`\bchangie\s+(batch|merge|latest|new|next)\b`)

func TestReleaseBatchTargetShape(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(data))
	block, ok := blocks["release:batch"]
	if !ok {
		t.Fatalf("%s: no top-level %q task", taskfilePath, "release:batch")
	}

	if !regexp.MustCompile(`(?m)^    silent: true\s*$`).MatchString(block) {
		t.Errorf("release:batch does not declare `silent: true`:\n%s", block)
	}
	desc, err := parseTaskDescription(block)
	if err != nil {
		t.Errorf("parseTaskDescription(release:batch): %v", err)
	} else if !strings.Contains(desc, "REL-10") {
		t.Errorf("release:batch desc does not cite REL-10: %q", desc)
	}
	msgs, err := parsePreconditionMessages(block)
	if err != nil {
		t.Errorf("parsePreconditionMessages(release:batch): %v", err)
	} else if !strings.Contains(strings.Join(msgs, "\n"), "go.tool-changie.mod") {
		t.Errorf("release:batch precondition messages %q do not name go.tool-changie.mod", msgs)
	}

	// Scope the body to the cmds: section so the desc prose is not scanned.
	cmdsIdx := regexp.MustCompile(`(?m)^    cmds:\s*$`).FindStringIndex(block)
	if cmdsIdx == nil {
		t.Fatalf("release:batch block has no cmds: key:\n%s", block)
	}
	cmds := block[cmdsIdx[1]:]

	// The seven tokens, at strictly increasing byte offsets.
	offset := 0
	tokens := 0
	for _, tk := range releaseChainBatchTokens {
		i := strings.Index(cmds[offset:], tk.token)
		if i < 0 {
			t.Errorf("release:batch cmds: token %q (%s) not found at or after byte offset %d", tk.token, tk.label, offset)
			continue
		}
		offset += i + len(tk.token)
		tokens++
	}

	// The skip-ci scan reads the rendered notes file.
	for _, line := range strings.Split(cmds, "\n") {
		if strings.Contains(line, releaseChainBatchTokens[5].token) && !strings.Contains(line, `".changes/${version}.md"`) {
			t.Errorf("release:batch skip-ci scan line does not read \".changes/${version}.md\": %q", line)
		}
	}

	// changie is reached only through GO_TOOL_CHANGIE (CHG-01's one install
	// path). Scoped to this block: check:changie legs call a scratch binary.
	invocations := 0
	for _, line := range releaseChainNonCommentLines(cmds) {
		if !releaseChainChangieVerbRe.MatchString(line) {
			continue
		}
		if strings.Contains(line, "echo ") || strings.Contains(line, "::error::") {
			continue
		}
		invocations++
		if !strings.Contains(line, "{{.GO_TOOL_CHANGIE}} changie ") {
			t.Errorf("release:batch invokes changie without the {{.GO_TOOL_CHANGIE}} prefix: %q", strings.TrimSpace(line))
		}
	}

	// No workflow file names the marketplace changie action (D-02).
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("read %s: %v", workflowsDir, err)
	}
	scanned := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		wfData, err := os.ReadFile(filepath.Join(workflowsDir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		scanned++
		for _, line := range releaseChainNonCommentLines(string(wfData)) {
			if strings.Contains(line, "miniscruff/changie-action") {
				t.Errorf("%s names the marketplace changie action (D-02 forbids a second install path): %q", e.Name(), strings.TrimSpace(line))
			}
		}
	}
	if scanned == 0 {
		t.Fatalf("%s: scanned zero workflow files — this guard would vacuously pass", workflowsDir)
	}
	if invocations < 3 {
		t.Errorf("release:batch invokes changie through GO_TOOL_CHANGIE %d times, want at least 3 (batch, merge, latest)", invocations)
	}
	t.Logf("release:batch: %d of %d ordered tokens found; %d changie invocations; %d workflow files scanned", tokens, len(releaseChainBatchTokens), invocations, scanned)
}
