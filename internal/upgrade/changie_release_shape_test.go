// changie_release_shape_test.go binds the changie release chain's CI wiring:
// .github/workflows/release-pr.yml (REL-10, D-01 to D-04, D-07) and the
// Taskfile.yml `release:batch` target it runs.
//
// These tests read this repository's YAML, Taskfile bodies and config only.
// They never exercise how changie, peter-evans/create-pull-request or GitHub
// behave (rule eyva2svrcz).
//
// Plan 05-02 adds the tag half to this file: release-tag.yml (REL-11, D-06,
// D-07), the Taskfile.yml `release:tag` target, and the GRD-16 scan that holds
// release:tag as the repository's only tag-creating command. Plan 05-03
// extends it further, so the YAML decoder below is general and every
// package-level identifier carries the releaseChain or tagCreation prefix
// (package upgrade is shared by many test files).
package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	releaseChainPinAppToken = "actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0"
	releaseChainPinCreatePR = "peter-evans/create-pull-request@5f6978faf089d4d20b00c7766989d076bb2fc7f1 # v8.1.1"

	releaseChainSetupTools = setupToolsAction
)

// releaseChainExpectedPins is the exact set of pinned (non-local) `uses:`
// values a release workflow in this plan may carry, in sorted order.
var releaseChainExpectedPins = []string{
	releaseChainPinCheckout,
	releaseChainPinAppToken,
	releaseChainPinCreatePR,
}

// releaseChainUsesLineRe matches a `uses:` line in raw workflow source,
// capturing the value that follows it up to end of line.
var releaseChainUsesLineRe = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*(.+?)\s*$`)

// releaseChainGoToolChangieRe matches a changie invocation that goes through
// the go command (`go tool`, `go run`, `go build`, `go install`, or a
// -modfile build) instead of the changie mise.toml pins.
var releaseChainGoToolChangieRe = regexp.MustCompile(`\bgo (tool|run|build|install)\b|-modfile`)

func TestReleaseChainGoToolChangieRe(t *testing.T) {
	cases := map[string]bool{
		"        changie batch auto":                                        false,
		"        version=$(changie latest)":                                 false,
		"        go tool changie latest":                                    true,
		"        go run github.com/miniscruff/changie latest":               true,
		"        go install github.com/miniscruff/changie@latest":           true,
		"        GOWORK=off go build -modfile=x.mod -o bin/changie changie": true,
	}
	for line, want := range cases {
		if got := releaseChainGoToolChangieRe.MatchString(line); got != want {
			t.Errorf("releaseChainGoToolChangieRe.MatchString(%q) = %v, want %v", line, got, want)
		}
	}
}

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
		{Name: "Set up tools", Uses: releaseChainSetupTools, With: map[string]string{"go-cache": "true"}},
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
		{Name: "Set up tools", Uses: releaseChainSetupTools, With: map[string]string{"go-cache": "true"}},
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
	// line is the local setup-tools action or a full-SHA pin with a version
	// comment, and the distinct pins are exactly the expected set.
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
		if v == releaseChainSetupTools {
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
	{"batch", `changie batch auto`},
	{"merge", `changie merge`},
	{"latest", `changie latest`},
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
	} else if !strings.Contains(strings.Join(msgs, "\n"), "mise install") {
		t.Errorf("release:batch precondition messages %q do not say to run mise install", msgs)
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

	// changie is the mise-pinned binary on PATH, never `go tool` or `go run`
	// (CHG-01's one install path). Scoped to this block.
	invocations := 0
	for _, line := range releaseChainNonCommentLines(cmds) {
		if !releaseChainChangieVerbRe.MatchString(line) {
			continue
		}
		if strings.Contains(line, "echo ") || strings.Contains(line, "::error::") {
			continue
		}
		invocations++
		if releaseChainGoToolChangieRe.MatchString(line) {
			t.Errorf("release:batch invokes changie through the go command, not the mise-pinned changie: %q", strings.TrimSpace(line))
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
		t.Errorf("release:batch invokes changie %d times, want at least 3 (batch, merge, latest)", invocations)
	}
	t.Logf("release:batch: %d of %d ordered tokens found; %d changie invocations; %d workflow files scanned", tokens, len(releaseChainBatchTokens), invocations, scanned)
}

// --- release-tag.yml (REL-11, D-06, D-07) ------------------------------------

const releaseChainTagWorkflowPath = "../../.github/workflows/release-tag.yml"

// releaseChainTagExpectedPins is the exact set of pinned (non-local) `uses:`
// values release-tag.yml may carry, in sorted order: no create-pull-request,
// because the tag half opens no PR.
var releaseChainTagExpectedPins = []string{
	releaseChainPinCheckout,
	releaseChainPinAppToken,
}

// releaseChainAppTokenOutput is the only credential the tag job's checkout
// may carry (T-05-06): a tag pushed with the default workflow token never
// starts release.yml.
const releaseChainAppTokenOutput = "${{ steps.app-token.outputs.token }}"

// releaseChainSeqValues returns the scalar values of a YAML sequence node.
func releaseChainSeqValues(t *testing.T, what string, n *yaml.Node) []string {
	t.Helper()
	if n.Kind != yaml.SequenceNode {
		t.Fatalf("%s is not a sequence (kind=%v)", what, n.Kind)
	}
	var out []string
	for _, e := range n.Content {
		out = append(out, e.Value)
	}
	return out
}

// releaseChainPinnedUses reads every non-comment `uses:` line of nonComment,
// reports a failure for any non-local value that is not a full 40-hex SHA pin
// followed by ` # vX.Y.Z`, and returns the number of uses: lines and the
// sorted distinct pins.
func releaseChainPinnedUses(t *testing.T, nonComment []string) (int, []string) {
	t.Helper()
	usesLines := 0
	pins := map[string]bool{}
	for _, line := range nonComment {
		m := releaseChainUsesLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		usesLines++
		v := m[1]
		if v == releaseChainSetupTools {
			continue
		}
		if !releaseChainPinnedUsesRe.MatchString(v) {
			t.Errorf("uses: %q is not a full 40-hex SHA pin followed by ` # vX.Y.Z`", v)
			continue
		}
		pins[v] = true
	}
	var got []string
	for p := range pins {
		got = append(got, p)
	}
	sort.Strings(got)
	return usesLines, got
}

// --- TestReleaseTagWorkflowShape (REL-11, D-06, D-07) ------------------------

func TestReleaseTagWorkflowShape(t *testing.T) {
	wf, src := mustReleaseChainWorkflow(t, releaseChainTagWorkflowPath)
	prWf, _ := mustReleaseChainWorkflow(t, releaseChainPRWorkflowPath)
	checked := 0

	// The display name must never be `release`: post-release-verify.yml keys
	// on workflow_run of "release".
	if wf.Name != "release-tag" {
		t.Errorf("top-level name = %q, want %q", wf.Name, "release-tag")
	}
	checked++

	// Trigger (D-06): changie's documented release trigger, a push to main
	// that changes CHANGELOG.md, and nothing else.
	onKeys := releaseChainMappingKeys(t, "on:", &wf.On)
	if !slices.Equal(onKeys, []string{"push"}) {
		t.Errorf("on: keys = %v, want exactly [push]", onKeys)
	} else {
		pushNode, _ := yamlMappingValue(&wf.On, "push")
		pushKeys := releaseChainMappingKeys(t, "on.push", pushNode)
		if !slices.Equal(pushKeys, []string{"branches", "paths"}) {
			t.Errorf("on.push keys = %v, want exactly [branches paths]", pushKeys)
		} else {
			branches, _ := yamlMappingValue(pushNode, "branches")
			if got := releaseChainSeqValues(t, "on.push.branches", branches); !slices.Equal(got, []string{"main"}) {
				t.Errorf("on.push.branches = %v, want exactly [main]", got)
			}
			paths, _ := yamlMappingValue(pushNode, "paths")
			if got := releaseChainSeqValues(t, "on.push.paths", paths); !slices.Equal(got, []string{"CHANGELOG.md"}) {
				t.Errorf("on.push.paths = %v, want exactly [CHANGELOG.md] (changie's documented release trigger, D-06)", got)
			}
		}
	}
	checked++

	if !mapsEqualReleaseChain(wf.Permissions, map[string]string{"contents": "read"}) {
		t.Errorf("top-level permissions = %v, want exactly {contents: read}", wf.Permissions)
	}
	checked++

	// A tag push is never cancelled mid-flight.
	if !mapsEqualReleaseChain(wf.Concurrency, map[string]string{"group": "release-tag", "cancel-in-progress": "false"}) {
		t.Errorf("concurrency = %v, want {group: release-tag, cancel-in-progress: false}", wf.Concurrency)
	}
	checked++

	var jobIDs []string
	for id := range wf.Jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)
	if !slices.Equal(jobIDs, []string{"pretag-gate", "tag"}) {
		t.Fatalf("jobs = %v, want exactly [pretag-gate tag]", jobIDs)
	}
	checked++

	// D-07: pretag-gate is the same job release-pr.yml runs, decoded equal.
	gate, prGate := wf.Jobs["pretag-gate"], prWf.Jobs["pretag-gate"]
	if len(gate.Steps) == 0 || len(prGate.Steps) == 0 {
		t.Fatalf("pretag-gate has %d steps here and %d in release-pr.yml — this comparison would vacuously pass", len(gate.Steps), len(prGate.Steps))
	}
	if !reflect.DeepEqual(gate, prGate) {
		t.Errorf("pretag-gate differs from release-pr.yml's (D-07 moves it verbatim):\n  release-tag.yml: %+v\n  release-pr.yml:  %+v", gate, prGate)
	}
	if last := gate.Steps[len(gate.Steps)-1]; stripRunBodyNoise(last.Run) != "task check:cross" {
		t.Errorf("pretag-gate last step run body = %q, want %q", stripRunBodyNoise(last.Run), "task check:cross")
	}
	checked += len(gate.Steps) + 1

	// The tag job: needs pretag-gate and runs the five steps in order, with
	// the App-token mint before the checkout that persists its credential.
	tag := wf.Jobs["tag"]
	if got := releaseChainNeeds(tag.Needs); !slices.Equal(got, []string{"pretag-gate"}) {
		t.Errorf("tag needs = %v, want [pretag-gate]", got)
	}
	if tag.RunsOn != "ubuntu-latest" {
		t.Errorf("tag runs-on = %q, want ubuntu-latest", tag.RunsOn)
	}
	if len(tag.Permissions) != 0 {
		t.Errorf("tag declares job-level permissions %v, want none", tag.Permissions)
	}
	checked += 3
	checked += releaseChainCheckSteps(t, "tag", tag.Steps, []releaseChainStepWanted{
		{
			Name: "Mint GitHub App installation token",
			ID:   "app-token",
			Uses: strings.SplitN(releaseChainPinAppToken, " #", 2)[0],
			With: map[string]string{"app-id": "${{ secrets.APP_ID }}", "private-key": "${{ secrets.APP_PRIVATE_KEY }}"},
		},
		{
			// The credential `git push` uses. persist-credentials stays at its
			// default: a With key for it would fail this comparison.
			Name: "Checkout",
			Uses: strings.SplitN(releaseChainPinCheckout, " #", 2)[0],
			With: map[string]string{"token": releaseChainAppTokenOutput},
		},
		{Name: "Set up tools", Uses: releaseChainSetupTools, With: map[string]string{"go-cache": "true"}},
		{Name: "Tag the release", Run: "task release:tag"},
	})

	nonComment := releaseChainNonCommentLines(src)
	usesLines, gotPins := releaseChainPinnedUses(t, nonComment)
	if usesLines == 0 {
		t.Fatalf("%s: found zero uses: lines — this guard would vacuously pass", releaseChainTagWorkflowPath)
	}
	if !slices.Equal(gotPins, releaseChainTagExpectedPins) {
		t.Errorf("distinct pinned uses = %v, want exactly %v", gotPins, releaseChainTagExpectedPins)
	}
	checked += usesLines

	// Run bodies: exactly two, each a single task call, no expression inside.
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
	if !slices.Equal(runBodies, []string{"task check:cross", "task release:tag"}) {
		t.Errorf("run bodies = %v, want exactly [task check:cross task release:tag]", runBodies)
	}
	checked += len(runBodies)

	// T-05-06: no non-comment line names the default token, and none touches
	// persist-credentials (comments may explain both).
	for i, line := range nonComment {
		if strings.Contains(line, "GITHUB_TOKEN") || strings.Contains(line, "github.token") {
			t.Errorf("non-comment line %d names the default token (REL-11): %q", i+1, line)
		}
		if strings.Contains(line, "persist-credentials") {
			t.Errorf("non-comment line %d overrides persist-credentials (it must stay at its default): %q", i+1, line)
		}
	}
	checked += len(nonComment)

	if checked == 0 {
		t.Fatalf("checked zero items — this guard would vacuously pass")
	}
	t.Logf("%s: %d fields, lines and uses checked (%d uses: lines, %d distinct pins, %d run bodies)", releaseChainTagWorkflowPath, checked, usesLines, len(gotPins), len(runBodies))
}

// --- TestReleaseTagTargetShape (REL-11, D-06, C3, C8) ------------------------

// releaseChainTagTokens are the tokens of release:tag's body, in the order
// they must appear: the tag, the origin lookup, the three case arms, the
// local tag, and the single-ref push.
var releaseChainTagTokens = []struct {
	label string
	token string
}{
	{"tag from changie latest", `tag=$(changie latest)`},
	{"origin lookup", `git ls-remote --exit-code --tags origin "refs/tags/${tag}"`},
	{"existing-tag arm", `0) echo "release:tag: ${tag} already exists on origin`},
	{"absent-tag arm", `2) ;;`},
	{"transport-error arm", `*) echo "::error::release:tag:`},
	{"local tag", `git tag "${tag}"`},
	{"single-ref push", `git push origin "refs/tags/${tag}"`},
}

func TestReleaseTagTargetShape(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(data))
	block, ok := blocks["release:tag"]
	if !ok {
		t.Fatalf("%s: no top-level %q task", taskfilePath, "release:tag")
	}

	if !regexp.MustCompile(`(?m)^    silent: true\s*$`).MatchString(block) {
		t.Errorf("release:tag does not declare `silent: true`:\n%s", block)
	}
	desc, err := parseTaskDescription(block)
	if err != nil {
		t.Errorf("parseTaskDescription(release:tag): %v", err)
	} else if !strings.Contains(desc, "REL-11") {
		t.Errorf("release:tag desc does not cite REL-11: %q", desc)
	}

	cmdsIdx := regexp.MustCompile(`(?m)^    cmds:\s*$`).FindStringIndex(block)
	if cmdsIdx == nil {
		t.Fatalf("release:tag block has no cmds: key:\n%s", block)
	}
	preconds, cmds := block[:cmdsIdx[0]], block[cmdsIdx[1]:]

	// C8: the target runs only under GitHub Actions (D-06R: no human git tag).
	if !strings.Contains(preconds, `sh: '[ "${GITHUB_ACTIONS:-}" = "true" ]'`) {
		t.Errorf("release:tag has no GITHUB_ACTIONS=true precondition (C8, D-06R):\n%s", preconds)
	}
	msgs, err := parsePreconditionMessages(block)
	if err != nil {
		t.Errorf("parsePreconditionMessages(release:tag): %v", err)
	} else if !strings.Contains(strings.Join(msgs, "\n"), "runs only in release-tag.yml") {
		t.Errorf("release:tag precondition messages %q do not say it runs only in release-tag.yml", msgs)
	}

	lines := releaseChainNonCommentLines(cmds)
	var body []string
	for _, l := range lines {
		// `- |` opens the block scalar; it is YAML, not part of the shell body.
		if tl := strings.TrimSpace(l); tl != "" && tl != "- |" {
			body = append(body, tl)
		}
	}
	if len(body) == 0 || body[0] != "set -euo pipefail" {
		t.Errorf("release:tag cmds do not start with `set -euo pipefail`: first line %q", firstOrEmpty(body))
	}

	// The tokens, at strictly increasing byte offsets.
	offset := 0
	found := 0
	for _, tk := range releaseChainTagTokens {
		i := strings.Index(cmds[offset:], tk.token)
		if i < 0 {
			t.Errorf("release:tag cmds: token %q (%s) not found at or after byte offset %d", tk.token, tk.label, offset)
			continue
		}
		offset += i + len(tk.token)
		found++
	}

	// Per-line contracts the ordered tokens cannot express.
	var lsRemote, existingArm, errorArm int
	var tagLines, pushLines []string
	for _, l := range body {
		switch {
		case strings.HasPrefix(l, "git ls-remote"):
			lsRemote++
			if !strings.HasSuffix(l, "|| rc=$?") {
				t.Errorf("release:tag ls-remote line does not capture its status with `|| rc=$?`: %q", l)
			}
		case strings.HasPrefix(l, "0) echo"):
			existingArm++
			if !strings.Contains(l, "already exists on origin") || !strings.HasSuffix(strings.TrimSpace(l), "exit 0 ;;") {
				t.Errorf("release:tag existing-tag arm must say `already exists on origin` and exit 0: %q", l)
			}
		case strings.HasPrefix(l, "*) echo"):
			errorArm++
			if !strings.Contains(l, "git ls-remote exited") || !strings.HasSuffix(strings.TrimSpace(l), "exit 1 ;;") {
				t.Errorf("release:tag transport-error arm must say `git ls-remote exited` and exit 1: %q", l)
			}
		}
		if regexp.MustCompile(`^git\s+tag\s`).MatchString(l) {
			tagLines = append(tagLines, l)
		}
		if regexp.MustCompile(`^git\s+push\s`).MatchString(l) {
			pushLines = append(pushLines, l)
		}
	}
	if lsRemote != 1 || existingArm != 1 || errorArm != 1 {
		t.Errorf("release:tag has %d ls-remote lines, %d existing-tag arms and %d transport-error arms, want 1 of each", lsRemote, existingArm, errorArm)
	}
	if len(tagLines) != 1 || len(pushLines) != 1 {
		t.Errorf("release:tag has %d `git tag` lines and %d `git push` lines, want exactly 1 of each: tag=%q push=%q", len(tagLines), len(pushLines), tagLines, pushLines)
	}
	for _, l := range pushLines {
		for _, flag := range []string{"--tags", "--follow-tags", "--all", "--mirror"} {
			if strings.Contains(l, flag) {
				t.Errorf("release:tag push widens past the one tag ref (%s): %q", flag, l)
			}
		}
	}

	// changie is the mise-pinned binary on PATH, never `go tool` or `go run`
	// (CHG-01's one install path).
	invocations := 0
	for _, l := range lines {
		if !releaseChainChangieVerbRe.MatchString(l) || strings.Contains(l, "echo ") || strings.Contains(l, "::error::") {
			continue
		}
		invocations++
		if releaseChainGoToolChangieRe.MatchString(l) {
			t.Errorf("release:tag invokes changie through the go command, not the mise-pinned changie: %q", strings.TrimSpace(l))
		}
	}
	if invocations != 1 {
		t.Errorf("release:tag invokes changie %d times, want exactly 1 (latest)", invocations)
	}
	t.Logf("release:tag: %d of %d ordered tokens found; %d body lines; %d changie invocation", found, len(releaseChainTagTokens), len(body), invocations)
}

func firstOrEmpty(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

// --- GRD-16: one tag-creating site, holding the App-token credential ----------

// The four tag-creating command shapes (RESEARCH Pitfall 6). Each is matched
// against one logical line of an executed body, after full-line comments are
// dropped. A script invoked by path is not opened; this mirrors
// resolveReleasePathClosure's recorded boundary 1 (T-05-10, accepted). A
// non-`sh:` Taskfile var value is not scanned either, because only cmds, cmd,
// sh, deps and defer are executed positions.
var (
	tagCreationGitTagRe          = regexp.MustCompile(`(^|[;&|(\s])git(\s+-[cC]\s+\S+)*\s+tag(\s|$)`)
	tagCreationGitPushRe         = regexp.MustCompile(`(^|[;&|(\s])git(\s+-[cC]\s+\S+)*\s+push(\s|$)`)
	tagCreationGhReleaseCreateRe = regexp.MustCompile(`\bgh\s+release\s+create\b`)
	tagCreationGhAPIRefsRe       = regexp.MustCompile(`\bgh\s+api\b.*\bgit/(refs|tags)\b`)
)

var tagCreationPatterns = []*regexp.Regexp{
	tagCreationGitTagRe,
	tagCreationGitPushRe,
	tagCreationGhReleaseCreateRe,
	tagCreationGhAPIRefsRe,
}

// tagCreationPushWordRe picks a push command out of release:tag's site lines
// for the validator, independent of the scanner's own patterns.
var tagCreationPushWordRe = regexp.MustCompile(`\bpush\b`)

const (
	// tagCreationSoleUnit is the one unit that may create or push a tag.
	tagCreationSoleUnit = "task:release:tag"
	// tagCreationSoleInvoker is the one place that unit may be invoked from.
	tagCreationSoleInvoker = "release-tag.yml/tag/Tag the release"
)

type tagCreationSite struct {
	Unit string // "workflow:<file>/<job>/<step>", "action:<path>/<step>" or "task:<target>"
	Line string
}

type tagCreationScan struct {
	WorkflowFiles   int
	RunBodies       int
	LocalActions    int
	TaskTargets     int
	ExecutedScalars int
	Sites           []tagCreationSite
	// TaskInvocations maps a Taskfile target to every place that invokes it:
	// "<file>/<job>/<step>" for a workflow step, "Taskfile:<target>" for a
	// `task:` reference or `task <target>` line inside the Taskfile.
	TaskInvocations map[string][]string
}

func tagCreationLineCreates(line string) bool {
	for _, re := range tagCreationPatterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// scanBody records every tag-creating line of one executed body under unit,
// and every `task <target>` line under invoker.
func (s *tagCreationScan) scanBody(unit, invoker, body string) {
	for _, line := range strings.Split(stripRunBodyNoise(body), "\n") {
		if line != "" && tagCreationLineCreates(line) {
			s.Sites = append(s.Sites, tagCreationSite{Unit: unit, Line: line})
		}
	}
	for _, target := range taskTargetsInRunBody(body) {
		s.TaskInvocations[target] = append(s.TaskInvocations[target], invoker)
	}
}

func tagCreationStepLabel(name string, i int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("step-%d", i+1)
}

func tagCreationDeref(n *yaml.Node) *yaml.Node {
	for n != nil && n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n
}

// execTask records one executed Taskfile scalar attributed to task name.
func (s *tagCreationScan) execTask(name, body string) {
	s.ExecutedScalars++
	s.scanBody("task:"+name, "Taskfile:"+name, body)
}

func (s *tagCreationScan) invokeFromTask(name, target string) {
	s.TaskInvocations[target] = append(s.TaskInvocations[target], "Taskfile:"+name)
}

// cmdEntry handles one element of a `cmds` (or `defer`) value: a shell
// string, or a mapping holding `cmd`, `task` or `defer`.
func (s *tagCreationScan) cmdEntry(name string, n *yaml.Node) {
	n = tagCreationDeref(n)
	switch n.Kind {
	case yaml.ScalarNode:
		s.execTask(name, n.Value)
	case yaml.MappingNode:
		if v, ok := yamlMappingValue(n, "cmd"); ok {
			s.execTask(name, tagCreationDeref(v).Value)
		}
		if v, ok := yamlMappingValue(n, "task"); ok {
			s.invokeFromTask(name, tagCreationDeref(v).Value)
		}
		if v, ok := yamlMappingValue(n, "defer"); ok {
			s.cmdEntry(name, v)
		}
	}
}

// shEntry executes a mapping's `sh:` value, if it has one (a dynamic var or a
// precondition).
func (s *tagCreationScan) shEntry(name string, n *yaml.Node) {
	n = tagCreationDeref(n)
	switch n.Kind {
	case yaml.ScalarNode:
		s.execTask(name, n.Value) // precondition shorthand: a bare shell string
	case yaml.MappingNode:
		if v, ok := yamlMappingValue(n, "sh"); ok {
			s.execTask(name, tagCreationDeref(v).Value)
		}
	}
}

func (s *tagCreationScan) varsEntries(name string, n *yaml.Node) {
	n = tagCreationDeref(n)
	if n.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if v := tagCreationDeref(n.Content[i+1]); v.Kind == yaml.MappingNode {
			s.shEntry(name, v)
		}
	}
}

func (s *tagCreationScan) scanTask(name string, n *yaml.Node) {
	n = tagCreationDeref(n)
	switch n.Kind {
	case yaml.ScalarNode:
		s.execTask(name, n.Value)
	case yaml.SequenceNode:
		for _, e := range n.Content {
			s.cmdEntry(name, e)
		}
	case yaml.MappingNode:
		if v, ok := yamlMappingValue(n, "cmds"); ok {
			for _, e := range tagCreationDeref(v).Content {
				s.cmdEntry(name, e)
			}
		}
		if v, ok := yamlMappingValue(n, "cmd"); ok {
			s.execTask(name, tagCreationDeref(v).Value)
		}
		if v, ok := yamlMappingValue(n, "deps"); ok {
			for _, e := range tagCreationDeref(v).Content {
				e = tagCreationDeref(e)
				switch e.Kind {
				case yaml.ScalarNode:
					s.invokeFromTask(name, e.Value)
				case yaml.MappingNode:
					if tv, ok := yamlMappingValue(e, "task"); ok {
						s.invokeFromTask(name, tagCreationDeref(tv).Value)
					}
				}
			}
		}
		if v, ok := yamlMappingValue(n, "preconditions"); ok {
			for _, e := range tagCreationDeref(v).Content {
				s.shEntry(name, e)
			}
		}
		if v, ok := yamlMappingValue(n, "vars"); ok {
			s.varsEntries(name, v)
		}
	}
}

type tagCreationActionFile struct {
	Runs struct {
		Steps []struct {
			Name string `yaml:"name"`
			Run  string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"runs"`
}

// scanTagCreationSites reads every workflow run body, every local action run
// step and every command Task executes, and reports the tag-creating ones. It
// never reads `desc:` or `msg:` text and never a full-line comment. It
// returns an error for zero workflows, zero run bodies or zero task targets
// decoded: a scan of nothing must not pass (rule 84d1gfpywd).
func scanTagCreationSites(workflows map[string][]byte, actions map[string][]byte, taskfile []byte) (tagCreationScan, error) {
	scan := tagCreationScan{TaskInvocations: map[string][]string{}}

	files := make([]string, 0, len(workflows))
	for f := range workflows {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, file := range files {
		jobs, err := parseWorkflowJobsAllSteps(string(workflows[file]))
		if err != nil {
			return scan, fmt.Errorf("scanTagCreationSites: workflow %s: %w", file, err)
		}
		scan.WorkflowFiles++
		jobIDs := make([]string, 0, len(jobs))
		for id := range jobs {
			jobIDs = append(jobIDs, id)
		}
		sort.Strings(jobIDs)
		for _, id := range jobIDs {
			for i, step := range jobs[id] {
				if step.Run == "" {
					continue
				}
				scan.RunBodies++
				label := file + "/" + id + "/" + tagCreationStepLabel(step.Name, i)
				scan.scanBody("workflow:"+label, label, step.Run)
			}
		}
	}

	paths := make([]string, 0, len(actions))
	for p := range actions {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		var act tagCreationActionFile
		if err := yaml.Unmarshal(actions[p], &act); err != nil {
			return scan, fmt.Errorf("scanTagCreationSites: action %s: %w", p, err)
		}
		scan.LocalActions++
		for i, step := range act.Runs.Steps {
			if step.Run == "" {
				continue
			}
			scan.RunBodies++
			label := p + "/" + tagCreationStepLabel(step.Name, i)
			scan.scanBody("action:"+label, "action:"+label, step.Run)
		}
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(taskfile, &doc); err != nil {
		return scan, fmt.Errorf("scanTagCreationSites: Taskfile: %w", err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return scan, fmt.Errorf("scanTagCreationSites: Taskfile has no top-level mapping")
	}
	root := doc.Content[0]
	if v, ok := yamlMappingValue(root, "vars"); ok {
		scan.varsEntries("<taskfile>", v)
	}
	tasks, ok := yamlMappingValue(root, "tasks")
	if !ok || tagCreationDeref(tasks).Kind != yaml.MappingNode {
		return scan, fmt.Errorf("scanTagCreationSites: Taskfile has no tasks: mapping")
	}
	tasks = tagCreationDeref(tasks)
	for i := 0; i+1 < len(tasks.Content); i += 2 {
		scan.TaskTargets++
		scan.scanTask(tasks.Content[i].Value, tasks.Content[i+1])
	}

	switch {
	case scan.WorkflowFiles == 0:
		return scan, fmt.Errorf("scanTagCreationSites: decoded zero workflow files — this scan would vacuously pass")
	case scan.RunBodies == 0:
		return scan, fmt.Errorf("scanTagCreationSites: decoded zero run bodies — this scan would vacuously pass")
	case scan.TaskTargets == 0:
		return scan, fmt.Errorf("scanTagCreationSites: decoded zero Taskfile targets — this scan would vacuously pass")
	}
	return scan, nil
}

// tagCreationForbiddenPushFlags are the flags that would widen release:tag's
// push past the one tag it just made.
var tagCreationForbiddenPushFlags = []string{"--tags", "--follow-tags", "--all", "--mirror"}

// validateTagCreationScan enforces GRD-16: exactly one unit holds a
// tag-creating command, task:release:tag; its push names refs/tags/ and no
// widening flag; and the only invocation of release:tag is release-tag.yml's
// tag job. Every violation is reported, each naming the offending unit.
func validateTagCreationScan(scan tagCreationScan) error {
	var problems []string

	byUnit := map[string][]string{}
	var order []string
	for _, site := range scan.Sites {
		if _, seen := byUnit[site.Unit]; !seen {
			order = append(order, site.Unit)
		}
		byUnit[site.Unit] = append(byUnit[site.Unit], site.Line)
	}
	for _, unit := range order {
		if unit != tagCreationSoleUnit {
			problems = append(problems, fmt.Sprintf("unit %q holds a tag-creating command %q; only %s may (GRD-16, D-06R)", unit, byUnit[unit][0], tagCreationSoleUnit))
		}
	}

	lines, found := byUnit[tagCreationSoleUnit]
	if !found {
		problems = append(problems, fmt.Sprintf("found zero tag-creating commands in %s, want exactly that one unit", tagCreationSoleUnit))
	} else {
		pushes := 0
		for _, line := range lines {
			if !tagCreationPushWordRe.MatchString(line) {
				continue
			}
			pushes++
			if !strings.Contains(line, "refs/tags/") {
				problems = append(problems, fmt.Sprintf("unit %q pushes without naming refs/tags/: %q", tagCreationSoleUnit, line))
			}
			for _, flag := range tagCreationForbiddenPushFlags {
				if strings.Contains(line, flag) {
					problems = append(problems, fmt.Sprintf("unit %q push uses %s, which widens past the one tag: %q", tagCreationSoleUnit, flag, line))
				}
			}
		}
		if pushes == 0 {
			problems = append(problems, fmt.Sprintf("unit %q holds no push line to check", tagCreationSoleUnit))
		}
	}

	invokers := scan.TaskInvocations["release:tag"]
	if !slices.Equal(invokers, []string{tagCreationSoleInvoker}) {
		problems = append(problems, fmt.Sprintf("task release:tag is invoked from %q, want exactly [%q]", invokers, tagCreationSoleInvoker))
	}
	for _, inv := range invokers {
		if strings.HasPrefix(inv, "Taskfile:") {
			problems = append(problems, fmt.Sprintf("%s references release:tag; only a workflow step may", inv))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(problems, "; "))
}

// validateTagJobCredential checks that the tag job's checkout credential is
// the output of the App-token mint that precedes it (T-05-06).
func validateTagJobCredential(wf *releaseChainWorkflow) error {
	job, ok := wf.Jobs["tag"]
	if !ok {
		return fmt.Errorf("release-tag.yml has no job %q", "tag")
	}
	mint, checkout := -1, -1
	for i, s := range job.Steps {
		switch {
		case s.ID == "app-token" && strings.HasPrefix(s.Uses, "actions/create-github-app-token@"):
			mint = i
		case strings.HasPrefix(s.Uses, "actions/checkout@"):
			checkout = i
		}
	}
	if mint < 0 || checkout < 0 || mint > checkout {
		return fmt.Errorf("tag job: App-token mint at step %d must precede the checkout at step %d", mint, checkout)
	}
	if got := job.Steps[checkout].With["token"]; got != releaseChainAppTokenOutput {
		return fmt.Errorf("tag job checkout token = %q, want %q", got, releaseChainAppTokenOutput)
	}
	return nil
}

const releaseChainActionsDir = "../../.github/actions"

func TestTagCreationHasSingleAppTokenSite(t *testing.T) {
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("read %s: %v", workflowsDir, err)
	}
	workflows := map[string][]byte{}
	onDisk := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		onDisk++
		data, err := os.ReadFile(filepath.Join(workflowsDir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		workflows[e.Name()] = data
	}

	actions := map[string][]byte{}
	actionDirs, err := os.ReadDir(releaseChainActionsDir)
	if err != nil {
		t.Fatalf("read %s: %v", releaseChainActionsDir, err)
	}
	for _, d := range actionDirs {
		if !d.IsDir() {
			continue
		}
		for _, name := range []string{"action.yml", "action.yaml"} {
			data, err := os.ReadFile(filepath.Join(releaseChainActionsDir, d.Name(), name))
			if err == nil {
				actions[d.Name()+"/"+name] = data
				break
			}
		}
	}
	taskfile, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}

	scan, err := scanTagCreationSites(workflows, actions, taskfile)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Positive assertions (rule 84d1gfpywd): the scan saw a real population.
	if scan.WorkflowFiles != onDisk {
		t.Errorf("scanned %d workflow files, but %d *.yml files are on disk", scan.WorkflowFiles, onDisk)
	}
	if scan.RunBodies <= 0 {
		t.Errorf("scanned %d run bodies, want more than 0", scan.RunBodies)
	}
	if scan.LocalActions < 1 {
		t.Errorf("scanned %d local actions, want at least 1", scan.LocalActions)
	}
	if scan.TaskTargets <= 50 {
		t.Errorf("scanned %d Taskfile targets, want more than 50", scan.TaskTargets)
	}
	units := map[string]bool{}
	for _, s := range scan.Sites {
		units[s.Unit] = true
	}
	if len(units) != 1 {
		t.Errorf("found %d tag-creating units %v, want exactly 1 (%s)", len(units), units, tagCreationSoleUnit)
	}

	if err := validateTagCreationScan(scan); err != nil {
		t.Errorf("GRD-16: %v", err)
	}

	// The credential `git push` uses is the App token.
	if src, ok := workflows["release-tag.yml"]; !ok {
		t.Errorf("release-tag.yml is not among the %d workflow files on disk", onDisk)
	} else if wf, err := parseReleaseChainWorkflow(src); err != nil {
		t.Errorf("release-tag.yml: %v", err)
	} else if err := validateTagJobCredential(wf); err != nil {
		t.Errorf("T-05-06: %v", err)
	}

	t.Logf("tag-creation scan: %d workflow files (%d on disk), %d run bodies, %d local actions, %d Taskfile targets, %d executed Taskfile scalars, %d tag-creating lines in %d unit(s), release:tag invoked from %q",
		scan.WorkflowFiles, onDisk, scan.RunBodies, scan.LocalActions, scan.TaskTargets, scan.ExecutedScalars, len(scan.Sites), len(units), scan.TaskInvocations["release:tag"])
}

// --- TestTagCreationScanFlagsPlantedSites: the scanner over literal inputs ----

const (
	tagCreationPlantedTagWorkflow = `name: release-tag
jobs:
  tag:
    runs-on: ubuntu-latest
    steps:
      - name: Tag the release
        run: task release:tag
`
	tagCreationPlantedAction = `name: Set up tools
runs:
  using: composite
  steps:
    - name: Verify the tools are on PATH
      shell: bash
      run: task --version
`
)

// tagCreationPlantedCI returns a ci.yml whose `test` job holds one extra step
// named "planted" running body.
func tagCreationPlantedCI(body string) []byte {
	return []byte("name: ci\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - name: planted\n        run: |\n          " + body + "\n")
}

// tagCreationPlantedTaskfile returns a Taskfile whose release:tag runs
// releaseTagCmds and which carries extra verbatim after its tasks.
func tagCreationPlantedTaskfile(releaseTagCmds, extra string) []byte {
	return []byte(`version: "3"
tasks:
  release:tag:
    desc: tags the release
    cmds:
      - |
` + releaseTagCmds + `
  check:
    desc: a harmless target
    cmds:
      - echo hi
` + extra)
}

const tagCreationBaselineCmds = `        git tag "${tag}"
        git push origin "refs/tags/${tag}"`

func TestTagCreationScanFlagsPlantedSites(t *testing.T) {
	baselineWorkflows := func(extra map[string][]byte) map[string][]byte {
		m := map[string][]byte{"release-tag.yml": []byte(tagCreationPlantedTagWorkflow)}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	actions := map[string][]byte{"setup-tools/action.yml": []byte(tagCreationPlantedAction)}

	cases := []struct {
		name      string
		workflows map[string][]byte
		taskfile  []byte
		// wantErr is a substring the validator's error must contain; "" means
		// the scan must validate cleanly.
		wantErr string
		// wantUnits, for the cases that must validate, is the exact set of
		// tag-creating units the scan must have seen: positive evidence that
		// an ignored comment or desc was read and skipped, not unread.
		wantUnits []string
	}{
		{
			name:      "baseline_single_site",
			workflows: baselineWorkflows(nil),
			taskfile:  tagCreationPlantedTaskfile(tagCreationBaselineCmds, ""),
			wantUnits: []string{tagCreationSoleUnit},
		},
		{
			name:      "reject_second_workflow_push_tags",
			workflows: baselineWorkflows(map[string][]byte{"ci.yml": tagCreationPlantedCI("git push origin --tags")}),
			taskfile:  tagCreationPlantedTaskfile(tagCreationBaselineCmds, ""),
			wantErr:   "workflow:ci.yml/test/planted",
		},
		{
			name:      "reject_second_taskfile_git_tag",
			workflows: baselineWorkflows(nil),
			taskfile:  tagCreationPlantedTaskfile(tagCreationBaselineCmds, "  rogue:\n    cmds:\n      - git tag v9.9.9\n"),
			wantErr:   "task:rogue",
		},
		{
			name:      "reject_gh_release_create",
			workflows: baselineWorkflows(map[string][]byte{"ci.yml": tagCreationPlantedCI("gh release create v9.9.9 --notes x")}),
			taskfile:  tagCreationPlantedTaskfile(tagCreationBaselineCmds, ""),
			wantErr:   "workflow:ci.yml/test/planted",
		},
		{
			name:      "reject_second_release_tag_invocation",
			workflows: baselineWorkflows(map[string][]byte{"ci.yml": tagCreationPlantedCI("task release:tag")}),
			taskfile:  tagCreationPlantedTaskfile(tagCreationBaselineCmds, ""),
			wantErr:   "ci.yml/test/planted",
		},
		{
			name:      "reject_release_tag_push_widened",
			workflows: baselineWorkflows(nil),
			taskfile: tagCreationPlantedTaskfile(`        git tag "${tag}"
        git push origin "refs/tags/${tag}" --tags`, ""),
			wantErr: "task:release:tag",
		},
		{
			name:      "ignore_comment_git_tag",
			workflows: baselineWorkflows(map[string][]byte{"ci.yml": tagCreationPlantedCI("# git tag v1.0.0\n          echo done")}),
			taskfile: tagCreationPlantedTaskfile(`        # git push origin --tags
`+tagCreationBaselineCmds, ""),
			wantUnits: []string{tagCreationSoleUnit},
		},
		{
			name:      "ignore_desc_git_tag",
			workflows: baselineWorkflows(nil),
			taskfile:  tagCreationPlantedTaskfile(tagCreationBaselineCmds, "  docs:\n    desc: explains how to git tag a release\n    cmds:\n      - echo hi\n"),
			wantUnits: []string{tagCreationSoleUnit},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			scan, err := scanTagCreationSites(c.workflows, actions, c.taskfile)
			if err != nil {
				t.Fatalf("scanTagCreationSites: %v", err)
			}
			verr := validateTagCreationScan(scan)
			if c.wantErr == "" {
				if verr != nil {
					t.Errorf("validateTagCreationScan rejected a clean scan: %v", verr)
				}
				units := map[string]bool{}
				for _, s := range scan.Sites {
					units[s.Unit] = true
				}
				var got []string
				for u := range units {
					got = append(got, u)
				}
				sort.Strings(got)
				if !slices.Equal(got, c.wantUnits) {
					t.Errorf("tag-creating units = %v, want exactly %v", got, c.wantUnits)
				}
				return
			}
			if verr == nil {
				t.Fatalf("validateTagCreationScan returned no error, want one naming %q (sites: %v)", c.wantErr, scan.Sites)
			}
			if !strings.Contains(verr.Error(), c.wantErr) {
				t.Errorf("validateTagCreationScan error %q does not name the offending unit %q", verr, c.wantErr)
			}
		})
	}
}

// --- release notes: release:goreleaser and release:dry-run (REL-12, REL-13) ----
//
// Plan 05-03. These guards read this repository's Taskfile.yml and
// release.yml only. How GoReleaser renders a Release body from the file it is
// handed is GoReleaser's behavior and is not asserted (D-14, rule eyva2svrcz).

// releaseChainTaskBlock returns the named top-level task block and the text
// after its `cmds:` key. It fails the test when the target or its cmds key is
// absent, never returning an empty block.
func releaseChainTaskBlock(t *testing.T, name string) (block, cmds string) {
	t.Helper()
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(data))
	block, ok := blocks[name]
	if !ok {
		t.Fatalf("%s: no top-level %q task", taskfilePath, name)
	}
	idx := regexp.MustCompile(`(?m)^    cmds:\s*$`).FindStringIndex(block)
	if idx == nil {
		t.Fatalf("%s block has no cmds: key:\n%s", name, block)
	}
	return block, block[idx[1]:]
}

// releaseChainOrderedTokens asserts each token appears in text at a strictly
// increasing byte offset, reporting every miss. It returns how many it found.
func releaseChainOrderedTokens(t *testing.T, what, text string, tokens []struct{ label, token string }) int {
	t.Helper()
	offset, found := 0, 0
	for _, tk := range tokens {
		i := strings.Index(text[offset:], tk.token)
		if i < 0 {
			t.Errorf("%s: token %q (%s) not found at or after byte offset %d", what, tk.token, tk.label, offset)
			continue
		}
		offset += i + len(tk.token)
		found++
	}
	return found
}

// releaseChainGoreleaserTokens are the tokens of release:goreleaser's body that
// pass the changie version file to GoReleaser, in the order they must appear:
// the tag at HEAD, the notes path derived from it, the refusal of an absent or
// empty file, then the release call carrying the flag (REL-12, D-08, C6).
var releaseChainGoreleaserTokens = []struct{ label, token string }{
	{"tag at HEAD", `TAG="$(git describe --exact-match --tags HEAD)"`},
	{"notes path from the tag", `NOTES=".changes/${TAG}.md"`},
	{"empty-notes check", `[ ! -s "${NOTES}" ]`},
	{"empty-notes error", `::error::release:goreleaser: ${NOTES} is missing or empty`},
	{"empty-notes exit", `exit 1`},
	{"release call with notes", `"${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"`},
}

func TestReleaseGoreleaserPassesReleaseNotes(t *testing.T) {
	block, cmds := releaseChainTaskBlock(t, "release:goreleaser")
	lines := releaseChainNonCommentLines(cmds)
	body := strings.Join(lines, "\n")

	// The ordered tokens, over non-comment text so a comment cannot satisfy one.
	found := releaseChainOrderedTokens(t, "release:goreleaser cmds", body, releaseChainGoreleaserTokens)

	// Exactly one non-comment release invocation, and it is the flagged one (C6).
	var releaseCalls, tagDefs int
	for _, l := range lines {
		tl := strings.TrimSpace(l)
		if strings.Contains(tl, `"${GORELEASER_BIN}" release`) {
			releaseCalls++
			if tl != `"${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"` {
				t.Errorf("release:goreleaser release invocation is %q, want exactly `\"${GORELEASER_BIN}\" release --clean --release-notes \"${NOTES}\"`", tl)
			}
		}
		if regexp.MustCompile(`^TAG=`).MatchString(tl) {
			tagDefs++
		}
	}
	if releaseCalls != 1 {
		t.Errorf("release:goreleaser has %d non-comment release invocations, want exactly 1 (C6)", releaseCalls)
	}
	if tagDefs != 1 {
		t.Errorf("release:goreleaser defines TAG on %d lines, want exactly 1 (the hoisted definition the GITHUB_OUTPUT block reuses)", tagDefs)
	}

	desc, err := parseTaskDescription(block)
	if err != nil {
		t.Errorf("parseTaskDescription(release:goreleaser): %v", err)
	} else if !strings.Contains(desc, "REL-12") || !strings.Contains(desc, "--release-notes") {
		t.Errorf("release:goreleaser desc does not cite REL-12 and --release-notes: %q", desc)
	}

	// D-12: the flag lives here, so release.yml's run bodies stay exactly the
	// two task invocations (TestWorkflowRunBodiesInvokeTask holds the rule;
	// this pins the set so a literal goreleaser line cannot appear there).
	wfData, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	jobs, err := parseWorkflowJobsAllSteps(string(wfData))
	if err != nil {
		t.Fatalf("%s: %v", releaseWorkflowPath, err)
	}
	var runs []string
	for _, steps := range jobs {
		for _, s := range steps {
			if r := stripRunBodyNoise(s.Run); r != "" {
				runs = append(runs, r)
			}
		}
	}
	sort.Strings(runs)
	if want := []string{"task release:goreleaser", "task release:record-final-hashes"}; !slices.Equal(runs, want) {
		t.Errorf("%s run bodies = %q, want exactly %q", releaseWorkflowPath, runs, want)
	}
	t.Logf("release:goreleaser: %d of %d ordered tokens found; %d release invocation; %d TAG definition; %d release.yml run bodies", found, len(releaseChainGoreleaserTokens), releaseCalls, tagDefs, len(runs))
}

// releaseChainDryRunTokens are the tokens of release:dry-run's body that
// rehearse the notes flag, in the order they must appear: the tag from changie,
// the notes path, the refusal of an absent or empty file, the evidence line,
// then the non-snapshot invocation (REL-13, D-14, Pitfall 2).
var releaseChainDryRunTokens = []struct{ label, token string }{
	{"tag from changie latest", `TAG="$(changie latest)"`},
	{"notes path from the tag", `NOTES=".changes/${TAG}.md"`},
	{"empty-notes check", `[ ! -s "${NOTES}" ]`},
	{"empty-notes error", `::error::release:dry-run: ${NOTES} is missing or empty`},
	{"evidence line", `RELEASE-NOTES-EVIDENCE tag=${TAG} file=${NOTES} bytes=`},
	{"evidence hash", `sha256=`},
	{"non-snapshot invocation", `GORELEASER_CURRENT_TAG="${TAG}" "${GORELEASER_BIN}" release --skip=publish,sign,validate --clean --release-notes "${NOTES}"`},
}

func TestReleaseDryRunRehearsesReleaseNotes(t *testing.T) {
	block, cmds := releaseChainTaskBlock(t, "release:dry-run")
	lines := releaseChainNonCommentLines(cmds)
	body := strings.Join(lines, "\n")

	found := releaseChainOrderedTokens(t, "release:dry-run cmds", body, releaseChainDryRunTokens)

	// GoReleaser skips its changelog pipe under --snapshot, so a snapshot run
	// would ignore --release-notes (Pitfall 2): no executed line may carry it.
	for _, l := range lines {
		if strings.Contains(l, "--snapshot") {
			t.Errorf("release:dry-run runs a --snapshot line, which skips the notes pipe: %q", strings.TrimSpace(l))
		}
	}

	// The four-binary assertions stay.
	for _, keep := range []string{"BINARY_COUNT", "want exactly 4"} {
		if !strings.Contains(body, keep) {
			t.Errorf("release:dry-run cmds lost the four-binary assertion token %q", keep)
		}
	}

	// changie is the mise-pinned binary on PATH, never `go tool` or `go run`
	// (CHG-01's one install path).
	invocations := 0
	for _, l := range lines {
		if !releaseChainChangieVerbRe.MatchString(l) || strings.Contains(l, "echo ") || strings.Contains(l, "::error::") {
			continue
		}
		invocations++
		if releaseChainGoToolChangieRe.MatchString(l) {
			t.Errorf("release:dry-run invokes changie through the go command, not the mise-pinned changie: %q", strings.TrimSpace(l))
		}
	}
	if invocations != 1 {
		t.Errorf("release:dry-run invokes changie %d times, want exactly 1 (latest)", invocations)
	}

	desc, err := parseTaskDescription(block)
	if err != nil {
		t.Errorf("parseTaskDescription(release:dry-run): %v", err)
	} else if !strings.Contains(desc, "REL-13") {
		t.Errorf("release:dry-run desc does not cite REL-13: %q", desc)
	}
	t.Logf("release:dry-run: %d of %d ordered tokens found; %d changie invocation", found, len(releaseChainDryRunTokens), invocations)
}
