package upgrade

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// --- BLD-04: proto:gen / proto:drift Taskfile shape guards ----------------
//
// Companion Go-level shape guard for plan 01-07's proto codegen drift guard,
// matching taskfile_shape_test.go's literal-fixture-plus-parse-and-assert
// idiom rather than inventing a new one (this task's own <read_first>).
// Reuses taskfilePath and the workflow-parsing helpers
// (workflowFileYAML/workflowRunStep/stripRunBodyNoise) already declared in
// taskfile_shape_test.go — same package, same on-disk fixtures.

// protoTaskYAML mirrors just the fields of one Taskfile.yml task entry these
// guards need: desc:, cmds: (joined into one string per task below), and
// preconditions: (reusing gatekeeperPrecondition's sh:/msg: shape, already
// declared in taskfile_shape_test.go). cmds: is decoded as []yaml.Node
// rather than []string: go-task's cmds: list allows BOTH a plain string
// entry and a map entry (`cmd:`/`for:`/`vars:` sub-task calls), and several
// unrelated tasks elsewhere in Taskfile.yml use the map form — a []string
// field would fail to decode the WHOLE document over one of THOSE tasks'
// shape, even though this guard only ever reads proto:gen/proto:drift's own
// (plain-string) cmds:. Deferring the string decode to joinScalarCmds below,
// per node, isolates that unrelated shape from this guard entirely.
type protoTaskYAML struct {
	Desc          string                   `yaml:"desc"`
	Cmds          []yaml.Node              `yaml:"cmds"`
	Preconditions []gatekeeperPrecondition `yaml:"preconditions"`
}

// protoTaskfileRootRaw mirrors just the tasks: map of Taskfile.yml at the
// yaml.Node level — deliberately NOT typed per-task here, for the same
// reason protoTaskYAML.Cmds is []yaml.Node: some tasks elsewhere in the file
// have shapes (e.g. map-form cmds: entries) this guard does not need to
// understand, and typing the WHOLE tasks: map strictly would fail the parse
// over one of them. Only the ONE named task this guard actually wants is
// decoded further, by parseProtoTask below.
type protoTaskfileRootRaw struct {
	Tasks map[string]yaml.Node `yaml:"tasks"`
}

// parseProtoTask decodes Taskfile.yml source src with the real YAML decoder
// and returns the named task's entry. Returns a non-nil error when src fails
// to parse as YAML, or when no task named name exists under tasks: — never a
// usable zero-value protoTaskYAML on a parse miss (the CR-01 defect class
// every parser in this file guards against).
func parseProtoTask(src, name string) (protoTaskYAML, error) {
	var root protoTaskfileRootRaw
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		return protoTaskYAML{}, fmt.Errorf("parseProtoTask(%q): %w", name, err)
	}
	rawTask, ok := root.Tasks[name]
	if !ok {
		return protoTaskYAML{}, fmt.Errorf("parseProtoTask(%q): no task found under tasks:", name)
	}
	var task protoTaskYAML
	if err := rawTask.Decode(&task); err != nil {
		return protoTaskYAML{}, fmt.Errorf("parseProtoTask(%q): decode task node: %w", name, err)
	}
	return task, nil
}

// joinScalarCmds returns the scalar (plain-string) entries of a task's
// cmds: list, in order, joined with "\n". Non-scalar entries (a map-form
// cmd:/for:/vars: sub-task call) are skipped rather than erroring — neither
// proto:gen nor proto:drift uses that form, so skipping only ever discards
// nothing for the tasks this guard actually reads.
func joinScalarCmds(nodes []yaml.Node) string {
	lines := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if n.Kind != yaml.ScalarNode {
			continue
		}
		lines = append(lines, n.Value)
	}
	return strings.Join(lines, "\n")
}

// mustProtoTaskCmds decodes Taskfile.yml source src, requires task name to
// exist with a non-empty desc: and at least one cmds: entry, and returns its
// cmds: joined with "\n" — the single string this file's structural
// assertions grep against. Fails the test on any parse miss or empty field,
// never silently continuing with a zero value.
func mustProtoTaskCmds(t *testing.T, src, name string) string {
	t.Helper()
	task, err := parseProtoTask(src, name)
	if err != nil {
		t.Fatalf("mustProtoTaskCmds(%q): %v", name, err)
	}
	if strings.TrimSpace(task.Desc) == "" {
		t.Fatalf("task %q has an empty desc:", name)
	}
	if len(task.Cmds) == 0 {
		t.Fatalf("task %q declares zero cmds:", name)
	}
	return joinScalarCmds(task.Cmds)
}

// TestProtoTasksExist parses the real Taskfile.yml and asserts both proto:gen
// (plan 01-01's deliverable, reused rather than re-authored here) and
// proto:drift (this task's own deliverable) are present with a non-empty
// desc: and at least one cmds: entry each.
func TestProtoTasksExist(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	src := string(data)

	genCmds := mustProtoTaskCmds(t, src, "proto:gen")
	if !strings.Contains(genCmds, "buf generate") {
		t.Errorf("proto:gen's cmds: do not invoke buf generate: %q", genCmds)
	}

	driftCmds := mustProtoTaskCmds(t, src, "proto:drift")
	if !strings.Contains(driftCmds, "buf generate") {
		t.Errorf("proto:drift's cmds: do not invoke buf generate: %q", driftCmds)
	}
}

// protoDriftComparedCountRe matches proto:drift's compared-count echo line.
// It is anchored to the ${nfiles} shell variable name this task's own
// implementation uses — not a bare [0-9]+ presence check — because the
// SOURCE text never contains a literal digit; the count is only numeric at
// RUNTIME. This test asserts the STRUCTURAL property (the guard computes and
// echoes a count variable before comparing), which is what
// TestProtoDriftGuardReportsAComparedCount is scoped to; the numeric floor
// itself is asserted at runtime by this plan's <automated> gate against the
// task's actual stdout, not by this test.
var protoDriftComparedCountRe = regexp.MustCompile(`compared\s+\$\{nfiles\}\s+generated files`)

// protoDriftZeroCountFailureRe matches proto:drift's zero/short-count
// failure branch: a numeric comparison against the same ${nfiles} variable
// the count echo above reports, guarding a failing exit.
//
// The floor literal is pinned here on purpose, so that moving proto:drift's
// floor is a deliberate, paired edit rather than a silent widening. It is 4
// because four generated files are committed and compared: graph.pb.go,
// ui.pb.go, ui.connect.go (Go) and ui_pb.ts (TypeScript). To re-verify, run
// `git ls-files -- 'internal/schema/*.pb.go' 'internal/uiproto/uiv1/*.pb.go'
// 'internal/uiproto/uiv1/uiv1connect/*.connect.go' 'web/src/lib/gen/*.ts'`
// and count the result. Per rule 84d1gfpywd the floor moves to the CORRECT
// number, never merely upward: 4 holds only while buf.gen.ts.yaml sets
// `opt: target=ts`, since the protoc-gen-es default of js+dts would emit two
// files and make 5 the wrong-but-plausible number.
var protoDriftZeroCountFailureRe = regexp.MustCompile(`\$\{nfiles\}"\s+-lt\s+4`)

// TestProtoDriftGuardReportsAComparedCount asserts the proto:drift command
// body contains a count-reporting step (protoDriftComparedCountRe) AND a
// zero/short-count failure branch (protoDriftZeroCountFailureRe) that exits
// non-zero. That property is what makes the guard non-vacuous per rule
// 84d1gfpywd, so it is itself guarded here — a no-op proto:drift body (one
// that only prints a static "no diff" message) would fail both matches.
func TestProtoDriftGuardReportsAComparedCount(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	cmds := mustProtoTaskCmds(t, string(data), "proto:drift")

	if !protoDriftComparedCountRe.MatchString(cmds) {
		t.Errorf("proto:drift's cmds: do not echo a compared-count line matching %q:\n%s", protoDriftComparedCountRe.String(), cmds)
	}
	if !protoDriftZeroCountFailureRe.MatchString(cmds) {
		t.Errorf("proto:drift's cmds: do not guard a zero/short-count failure branch matching %q:\n%s", protoDriftZeroCountFailureRe.String(), cmds)
	}
	if !strings.Contains(cmds, "exit 1") {
		t.Errorf("proto:drift's cmds: contain no exit 1 — a count-reporting guard that never fails is vacuous")
	}
}

// TestProtoDriftGeneratesIntoATemporaryTree asserts the proto:drift command
// body creates a temporary output root (mktemp -d), cleans it up on exit
// INCLUDING on failure (a trap on EXIT, not a final-line-only rm), and reads
// the freshly-regenerated comparison target from inside that temporary root
// rather than comparing against files regenerated in place. This is the
// structural half of the no-data-loss property (T-01-32): a developer's
// uncommitted edit to a committed generated file must survive a guard run.
func TestProtoDriftGeneratesIntoATemporaryTree(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	cmds := mustProtoTaskCmds(t, string(data), "proto:drift")

	if !strings.Contains(cmds, "mktemp -d") {
		t.Errorf("proto:drift's cmds: do not create a temporary output root via mktemp -d:\n%s", cmds)
	}
	if !regexp.MustCompile(`trap\s+'rm -rf "\$\{scratch\}"'\s+EXIT`).MatchString(cmds) {
		t.Errorf("proto:drift's cmds: do not clean up the temporary root on EXIT (including on failure):\n%s", cmds)
	}
	// The comparison must read the freshly-generated file FROM the
	// temporary root ("${scratch}/..."), never regenerate over the
	// committed path directly.
	if !regexp.MustCompile(`fresh="\$\{scratch\}/gen/\$\{f\}"`).MatchString(cmds) {
		t.Errorf("proto:drift's cmds: comparison does not read from the temporary root:\n%s", cmds)
	}
	if !strings.Contains(cmds, `cmp -s "${f}" "${fresh}"`) {
		t.Errorf("proto:drift's cmds: does not byte-compare the committed file against its temporary-tree counterpart:\n%s", cmds)
	}
	// buf generate must be pointed AT the temporary root via -o, never
	// left to regenerate at its default in-place output location.
	if !regexp.MustCompile(`buf generate\s+-o\s+"\$\{scratch\}/gen"`).MatchString(cmds) {
		t.Errorf("proto:drift's cmds: buf generate is not directed at the temporary root via -o:\n%s", cmds)
	}
}

// TestProtoDriftTaskIsInvokedByCI parses the REAL .github/workflows/ci.yml
// from disk — not a fixture copy — and asserts at least one step across all
// jobs invokes `task proto:drift` (after stripRunBodyNoise strips
// comments/blanks), the same single-definition property
// TestWorkflowRunBodiesInvokeTask already enforces for every in-scope job's
// steps. Scans every job rather than hardcoding the job this task placed the
// step in, so a future reorganisation of ci.yml's job layout does not
// silently break this guard by moving the step.
func TestProtoDriftTaskIsInvokedByCI(t *testing.T) {
	path := workflowsDir + "/ci.yml"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var wf workflowFileYAML
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if len(wf.Jobs) == 0 {
		t.Fatalf("%s declares zero jobs: — this guard would vacuously pass over an empty workflow", path)
	}

	for jobID, job := range wf.Jobs {
		for _, step := range job.Steps {
			if step.Run == "" {
				continue
			}
			if stripRunBodyNoise(step.Run) == "task proto:drift" {
				return
			}
		}
		_ = jobID // keep the loop variable meaningfully named for readability
	}
	t.Fatalf("%s: no step in any job invokes exactly 'task proto:drift' after stripping comments/blanks", path)
}
