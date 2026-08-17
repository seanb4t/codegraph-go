package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// bench_workflow_shape_test.go is the publish-job-scoped D-06 verifier
// (review HIGH 1/2/3, cycle-2 MEDIUM 2/3, LOW 1/2): every D-06 property is
// asserted against the PARSED jobs.publish subtree — never by a file-wide
// grep another job (rebless's identical if-no-files-found: error, for
// instance) could already satisfy on the publish job's behalf. It reuses
// this package's `../../.github/workflows/bench.yml` relative-path and
// yaml.Unmarshal convention (see taskfile_shape_test.go's workflowsDir and
// release_workflow_shape_test.go's decodeFullWorkflowDoc), with its own
// bespoke struct set — the shared workflowRunStep/fullWorkflowJob types
// deliberately aren't extended here, since widening a shared parsing type
// for one file's bespoke needs (uses:/if:/runs-on: together in one place)
// risks silently changing what every OTHER guard in this package sees.

// benchWorkflowPath is the on-disk path (relative to this package) to the
// single workflow file both shape tests in this file parse.
const benchWorkflowPath = "../../.github/workflows/bench.yml"

// benchStep is the subset of a GitHub Actions workflow step both shape
// tests in this file need: its display name, its uses: action reference
// (empty for a run: step), its run: body (empty for a uses: step), and its
// with: parameter map. With: is map[string]any, not map[string]string,
// because a real with: block mixes scalar types (e.g. Set up Go's
// `cache: false` is a YAML bool sitting next to `go-version-file: go.mod`,
// a string) — decoding straight into map[string]string would fail the
// WHOLE document's Unmarshal on that bool, not just the one field.
type benchStep struct {
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

// benchJob is the subset of a GitHub Actions job both shape tests need.
type benchJob struct {
	RunsOn      string            `yaml:"runs-on"`
	If          string            `yaml:"if"`
	Env         map[string]string `yaml:"env"`
	Permissions map[string]string `yaml:"permissions"`
	Steps       []benchStep       `yaml:"steps"`
}

// benchDispatchInput mirrors one workflow_dispatch.inputs.<name> entry —
// only the two fields TestBenchPublishJobShape needs.
type benchDispatchInput struct {
	Default string   `yaml:"default"`
	Options []string `yaml:"options"`
}

// benchWorkflowDispatch mirrors on.workflow_dispatch.
type benchWorkflowDispatch struct {
	Inputs map[string]benchDispatchInput `yaml:"inputs"`
}

// benchWorkflowOn mirrors the on: key's workflow_dispatch sub-key only.
// Declared as a plain struct (not a generic map[string]any) specifically
// because struct-tag decoding does not hit YAML 1.1's implicit
// on/off-as-bool resolution that bites map[string]any key decoding — the
// identical property release_workflow_shape_test.go's fullWorkflowDoc.On
// doc comment documents; that file uses yaml.Node instead only because it
// also needs to resolve scalar/sequence/mapping trigger shapes generically,
// a need this file does not have (bench.yml's on: is always a mapping).
type benchWorkflowOn struct {
	WorkflowDispatch benchWorkflowDispatch `yaml:"workflow_dispatch"`
}

// benchWorkflowDoc is the top-level decode target for bench.yml.
type benchWorkflowDoc struct {
	On          benchWorkflowOn     `yaml:"on"`
	Permissions map[string]string   `yaml:"permissions"`
	Jobs        map[string]benchJob `yaml:"jobs"`
}

// decodeBenchWorkflowDoc reads and decodes bench.yml, failing the test
// loudly — never returning a usable zero value — on a read failure, a YAML
// parse failure, or a file declaring zero jobs:.
func decodeBenchWorkflowDoc(t *testing.T) benchWorkflowDoc {
	t.Helper()
	data, err := os.ReadFile(benchWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", benchWorkflowPath, err)
	}
	var doc benchWorkflowDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("yaml.Unmarshal(%s): %v", benchWorkflowPath, err)
	}
	if len(doc.Jobs) == 0 {
		t.Fatalf("decodeBenchWorkflowDoc(%s): declares zero jobs:", benchWorkflowPath)
	}
	return doc
}

// mustBenchJob selects job jobID from doc's parsed jobs: map, failing the
// test loudly if the key is absent. This is the anchor every other
// assertion in both shape tests below is scoped to — without it, every
// assertion is vacuous (review HIGH 1/2/3): a file-wide grep cannot tell
// the publish job's own properties from another job's, but a subtree
// selected by this function can.
func mustBenchJob(t *testing.T, doc benchWorkflowDoc, jobID string) benchJob {
	t.Helper()
	job, ok := doc.Jobs[jobID]
	if !ok {
		t.Fatalf("bench.yml declares no job %q", jobID)
	}
	return job
}

// benchActionIdentifier splits a uses: value into its action identifier
// (owner/repo, or a local ./path with no @ suffix at all) and its
// @-suffixed ref.
func benchActionIdentifier(uses string) (name, ref string) {
	idx := strings.LastIndex(uses, "@")
	if idx < 0 {
		return uses, ""
	}
	return uses[:idx], uses[idx+1:]
}

// sha40Re matches a 40-character lowercase-hex full commit SHA — the pin
// shape every remote uses: reference in this repo must carry.
var sha40Re = regexp.MustCompile(`^[0-9a-f]{40}$`)

// withString stringifies with[key], returning "" if key is absent. GitHub
// Actions with: values this file inspects (name/path/if-no-files-found) are
// always plain string scalars, so %v round-trips them exactly.
func withString(with map[string]any, key string) string {
	v, ok := with[key]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// withStrings stringifies every key in with — used for an exact map[string]string
// comparison against a literal fixture (TestBenchReblessJobShape), so an
// unexpected extra with: key fails the comparison too, not just a missing
// or wrong-valued one.
func withStrings(with map[string]any) map[string]string {
	out := make(map[string]string, len(with))
	for k, v := range with {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

// concatRunBodies joins every step's non-empty run: body in job, in step
// order, separated by a single newline.
func concatRunBodies(job benchJob) string {
	var bodies []string
	for _, s := range job.Steps {
		if s.Run != "" {
			bodies = append(bodies, s.Run)
		}
	}
	return strings.Join(bodies, "\n")
}

// countRunStepsContaining returns the number of steps in job whose run:
// body contains substr.
func countRunStepsContaining(job benchJob, substr string) int {
	n := 0
	for _, s := range job.Steps {
		if strings.Contains(s.Run, substr) {
			n++
		}
	}
	return n
}

// normalizeRunBody trims trailing whitespace from every line of body and
// drops trailing blank lines — the exact normalisation
// reblessRunBodyDigestFixture below was computed under.
func normalizeRunBody(body string) string {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// stepByName returns the first step in job named name, failing the test
// loudly (naming jobLabel and name) if none matches.
func stepByName(t *testing.T, job benchJob, jobLabel, name string) benchStep {
	t.Helper()
	for _, s := range job.Steps {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("%s job has no step named %q", jobLabel, name)
	return benchStep{}
}

// --- TestBenchPublishJobShape --------------------------------------------

// TestBenchPublishJobShape is the publish-job-scoped D-06 verifier (review
// HIGH 1/2/3, cycle-2 LOW 1/2). It selects jobs.publish and fails loudly if
// that key is absent (mustBenchJob) — the job key is the anchor, and every
// assertion below is vacuous without it.
func TestBenchPublishJobShape(t *testing.T) {
	doc := decodeBenchWorkflowDoc(t)
	job := mustBenchJob(t, doc, "publish")

	// D-06.1 — runner pinning + env contract, asserted exactly, scoped to
	// this job. CheckRegression refuses a cross-runner comparison
	// (internal/bench/regression.go:72), so both values are measurement
	// validity, not cosmetics.
	const wantRunner = "namespace-profile-linux-amd64-4x8"
	if job.RunsOn != wantRunner {
		t.Errorf("publish job runs-on = %q, want %q", job.RunsOn, wantRunner)
	}
	if got := job.Env["CODEGRAPH_BENCH_RUNNER"]; got != wantRunner {
		t.Errorf("publish job env.CODEGRAPH_BENCH_RUNNER = %q, want %q", got, wantRunner)
	}

	// The if: expression must carry all three live dispatch terms: the
	// scheduled-event term, the dispatch term naming this job id, and the
	// dispatch term inputs.job == 'both' (cycle-2 LOW 2) — dropping the
	// 'both' clause turns 'both' into a silently rebless-only option.
	for _, want := range []string{
		"github.event_name == 'schedule'",
		"inputs.job == 'publish'",
		"inputs.job == 'both'",
	} {
		if !strings.Contains(job.If, want) {
			t.Errorf("publish job if: %q does not contain %q", job.If, want)
		}
	}

	// D-06.2 — exactly one step invokes the runner inline via go run
	// ./tools/bench/runner, and that body carries the publish mode flag.
	runnerInvocations := 0
	for _, s := range job.Steps {
		if strings.Contains(s.Run, "go run ./tools/bench/runner") {
			runnerInvocations++
			if !strings.Contains(s.Run, "-mode publish") {
				t.Errorf("publish job step %q invokes the runner but its body does not carry -mode publish: %q", s.Name, s.Run)
			}
		}
	}
	if runnerInvocations != 1 {
		t.Errorf("publish job has %d step(s) invoking 'go run ./tools/bench/runner', want exactly 1", runnerInvocations)
	}

	// D-06.3 — the concatenation of every run: body is non-empty (the
	// positive assertion — a job with no run bodies must fail rather than
	// pass every prohibition vacuously) and contains none of the forbidden
	// substrings: a task bench invocation, the gate function name, the
	// regression mode flag, the baseline-overwriting flag, or a
	// second-runtime invocation (npm/npx/node).
	concat := concatRunBodies(job)
	if strings.TrimSpace(concat) == "" {
		t.Fatalf("publish job's concatenated run: bodies are empty — every prohibition below would pass vacuously")
	}
	forbidden := []string{"task bench", "CheckRegression", "-mode regression", "-rebless", "npm ", "npx ", "node "}
	for _, f := range forbidden {
		if strings.Contains(concat, f) {
			t.Errorf("publish job's run: bodies contain forbidden substring %q (D-06.3/BENCH-03: publish must never gate and must never invoke a second implementation)", f)
		}
	}

	// D-06.4 (review HIGH 2) — exactly one run body references the
	// job-summary variable.
	if n := countRunStepsContaining(job, "GITHUB_STEP_SUMMARY"); n != 1 {
		t.Errorf("publish job has %d run: bod(y/ies) referencing GITHUB_STEP_SUMMARY, want exactly 1", n)
	}

	// D-06.4 (review HIGH 2) — exactly one upload-artifact step, and its
	// OWN with: map carries the three required keys. Scoped to THIS job:
	// the untouched rebless upload at bench.yml:274 already carries
	// if-no-files-found: error and would otherwise satisfy a file-wide
	// check on the publish job's behalf.
	uploads := 0
	for _, s := range job.Steps {
		name, _ := benchActionIdentifier(s.Uses)
		if name != "actions/upload-artifact" {
			continue
		}
		uploads++
		want := map[string]string{
			"name":              "publish-results",
			"path":              "publish-results.json",
			"if-no-files-found": "error",
		}
		for k, v := range want {
			if got := withString(s.With, k); got != v {
				t.Errorf("publish job's upload-artifact step with.%s = %q, want %q", k, got, v)
			}
		}
	}
	if uploads != 1 {
		t.Errorf("publish job has %d actions/upload-artifact step(s), want exactly 1", uploads)
	}

	// No step uses a Node-setup action — BENCH-03 forbids installing a
	// second runtime anywhere in this job.
	for _, s := range job.Steps {
		name, _ := benchActionIdentifier(s.Uses)
		if strings.Contains(name, "setup-node") {
			t.Errorf("publish job step %q uses a Node-setup action (%s) — BENCH-03 forbids installing a second runtime", s.Name, name)
		}
	}

	// cycle-2 LOW 1 — the action identifier list is ORDERED, not a set,
	// and length-asserted: a set comparison would silently tolerate a
	// duplicated checkout or setup-go step.
	wantActions := []string{
		"actions/checkout",
		"actions/setup-go",
		"namespacelabs/nscloud-cache-action",
		"actions/upload-artifact",
	}
	var gotActions []string
	for _, s := range job.Steps {
		if s.Uses == "" {
			continue
		}
		name, ref := benchActionIdentifier(s.Uses)
		gotActions = append(gotActions, name)
		if !sha40Re.MatchString(ref) {
			t.Errorf("publish job step %q's uses: ref %q is not a 40-hex SHA pin", s.Name, ref)
		}
	}
	if len(gotActions) != len(wantActions) {
		t.Errorf("publish job action identifier list has %d entries %v, want %d entries %v", len(gotActions), gotActions, len(wantActions), wantActions)
	} else {
		for i := range wantActions {
			if gotActions[i] != wantActions[i] {
				t.Errorf("publish job action[%d] = %q, want %q (ordered list, not a set)", i, gotActions[i], wantActions[i])
			}
		}
	}

	// The job declares no permissions: key of its own, and the
	// workflow-level permissions: map is exactly {contents: read}.
	if len(job.Permissions) != 0 {
		t.Errorf("publish job declares its own permissions: %v, want none (inherits workflow-level contents: read)", job.Permissions)
	}
	wantWorkflowPerms := map[string]string{"contents": "read"}
	if !reflect.DeepEqual(doc.Permissions, wantWorkflowPerms) {
		t.Errorf("workflow-level permissions: = %v, want %v", doc.Permissions, wantWorkflowPerms)
	}

	// Dispatch identifier family: default/options must name 'publish' and
	// keep 'both' answered (cycle-2 LOW 2) — so the identifier family
	// cannot drift apart and 'both' cannot survive as a choice the publish
	// job no longer answers.
	jobInput, ok := doc.On.WorkflowDispatch.Inputs["job"]
	if !ok {
		t.Fatalf("workflow_dispatch.inputs declares no %q input", "job")
	}
	if jobInput.Default != "publish" {
		t.Errorf("workflow_dispatch.inputs.job.default = %q, want %q", jobInput.Default, "publish")
	}
	for _, want := range []string{"publish", "both"} {
		found := false
		for _, opt := range jobInput.Options {
			if opt == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("workflow_dispatch.inputs.job.options %v does not contain %q", jobInput.Options, want)
		}
	}
}

// --- TestBenchReblessJobShape ---------------------------------------------

// reblessIfFixture is the rebless job's if: expression, transcribed
// verbatim from the file at the recorded pre-plan SHA
// (06-02-PREPLAN-SHA.txt: c49ef77251d8cc65866f697fdf2b41a6ed7020e7) — this
// job is a deliberately-untouched region of this plan's edit.
const reblessIfFixture = "github.event_name == 'workflow_dispatch' && (inputs.job == 'rebless' || inputs.job == 'both')"

// reblessStepNamesFixture is the pre-edit rebless job's ordered step name:
// sequence — six steps, transcribed verbatim. A sequence comparison (not a
// count) catches an inserted, removed, or reordered step.
var reblessStepNamesFixture = []string{
	"Checkout",
	"Set up Go",
	"Record candidate baseline on this runner",
	"Publish candidate to job summary",
	"Upload candidate baseline artifact",
	"Verify the candidate survives the gate on this runner class",
}

// reblessRunBodyDigestFixture is the SHA-256 hex digest of the normalised
// (normalizeRunBody, joined by "\n") concatenation, in step order, of the
// rebless job's three run: bodies — "Record candidate baseline on this
// runner", "Publish candidate to job summary", and "Verify the candidate
// survives the gate on this runner class" — computed against the file at
// the recorded pre-plan SHA (06-02-PREPLAN-SHA.txt:
// c49ef77251d8cc65866f697fdf2b41a6ed7020e7). Cycle-2 MEDIUM 2: without this,
// the parsed-subtree fixture below covers structure and step names but not
// the load-bearing bodies themselves, including the -rebless invocation the
// D-13/D-01 exception exists to contain. A digest is the right shape here
// specifically because these bodies are not this phase's subject: the
// assertion is "unchanged", and nothing about their content needs to be
// readable in this test — the publish job's bodies are the opposite case
// and stay asserted by content above.
const reblessRunBodyDigestFixture = "edbd1b7f634aef3f85c812d7b7c0ccecc303a85f7d37c698e118173f43a9a1ae"

// TestBenchReblessJobShape is the parsed-subtree comparison review finding
// 06-02:165 asked for, in a form that keeps checking after this phase ends:
// it selects jobs.rebless and asserts it against a literal fixture
// transcribed from the pre-edit file.
func TestBenchReblessJobShape(t *testing.T) {
	doc := decodeBenchWorkflowDoc(t)
	job := mustBenchJob(t, doc, "rebless")

	if job.RunsOn != "ubuntu-latest" {
		t.Errorf("rebless job runs-on = %q, want %q", job.RunsOn, "ubuntu-latest")
	}
	if got := job.Env["CODEGRAPH_BENCH_RUNNER"]; got != "ubuntu-latest" {
		t.Errorf("rebless job env.CODEGRAPH_BENCH_RUNNER = %q, want %q", got, "ubuntu-latest")
	}
	if job.If != reblessIfFixture {
		t.Errorf("rebless job if: = %q, want %q", job.If, reblessIfFixture)
	}

	var gotNames []string
	for _, s := range job.Steps {
		gotNames = append(gotNames, s.Name)
	}
	if !reflect.DeepEqual(gotNames, reblessStepNamesFixture) {
		t.Errorf("rebless job step name: sequence = %v, want %v (ordered — an inserted, removed, or reordered step fails here)", gotNames, reblessStepNamesFixture)
	}

	uploadStep := stepByName(t, job, "rebless", "Upload candidate baseline artifact")
	wantWith := map[string]string{
		"name":              "baseline-candidate",
		"path":              "tools/bench/baseline.json",
		"if-no-files-found": "error",
	}
	if got := withStrings(uploadStep.With); !reflect.DeepEqual(got, wantWith) {
		t.Errorf("rebless job's upload step with: = %v, want %v", got, wantWith)
	}

	if len(job.Permissions) != 0 {
		t.Errorf("rebless job declares its own permissions: %v, want none", job.Permissions)
	}

	bodyNames := []string{
		"Record candidate baseline on this runner",
		"Publish candidate to job summary",
		"Verify the candidate survives the gate on this runner class",
	}
	var bodies []string
	for _, name := range bodyNames {
		s := stepByName(t, job, "rebless", name)
		bodies = append(bodies, normalizeRunBody(s.Run))
	}
	sum := sha256.Sum256([]byte(strings.Join(bodies, "\n")))
	gotDigest := hex.EncodeToString(sum[:])
	if gotDigest != reblessRunBodyDigestFixture {
		t.Errorf("rebless job's three run: bodies hash to %s, want %s (rebless must stay byte-unchanged by this plan's edit; the computed digest is printed here so a deliberate future change to rebless is a visible one-line fixture update, not a puzzle)", gotDigest, reblessRunBodyDigestFixture)
	}
}
