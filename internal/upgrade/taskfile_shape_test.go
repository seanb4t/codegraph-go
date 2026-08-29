package upgrade

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
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
	// golangciModfilePath is the fourth isolated tool modfile
	// (03-10-PLAN.md Task 1), registered with TestToolModfilesRemainIsolated
	// below so a new modfile absent from that guard's iterated set is not
	// silently inspected by nothing (the same shape TestWorkflowRunBodiesInvokeTask
	// guards for CI jobs — see inScopeJobs).
	golangciModfilePath = "../../go.tool-golangci.mod"
	// protoModfilePath was a PRE-EXISTING gap of the identical shape —
	// absent from this guard's iterated set since 01-01-PLAN.md — closed
	// alongside golangciModfilePath's registration rather than left as a
	// second recorded-but-unfixed instance of the same defect (03-10-PLAN.md
	// Task 1; see 03-10-SUMMARY.md for why this went beyond the plan's own
	// stated scope).
	protoModfilePath = "../../go.tool-proto.mod"
	taskfilePath     = "../../Taskfile.yml"
	goreleaserPath   = "../../.goreleaser.yaml"
	checkCrossTaskID = "check:cross"

	// releasePathWorkflowPath is an alias for releaseWorkflowPath
	// (release_workflow_shape_test.go), declared here too so the BLD-07
	// release-path fixture block below reads as self-contained beside
	// goreleaserPath, its sibling ROOT.
	releasePathWorkflowPath = releaseWorkflowPath

	// ciWorkflowPath is the on-disk path to the workflow BLD-01's
	// no-mutable-JS-cache invariant (Task 3) scans. Deliberately NOT
	// releasePathWorkflowPath: ci.yml is not part of the signed release
	// path — it is the PR/push gate whose `test` job D-13 folds the JS
	// gates into.
	ciWorkflowPath = "../../.github/workflows/ci.yml"

	// releasePathRepoRoot is the on-disk path (relative to this package)
	// to the repository root — used only by the BLD-07 unmodelled-edge
	// tripwire to test whether a candidate word inside a run:/cmds:/hooks:
	// scalar names a file that actually exists in the worktree.
	releasePathRepoRoot = "../.."
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

// forbiddenToolPackages are the build-tool import paths that must live
// ONLY in the isolated tool modfiles (go.tool.mod / go.tool-lint.mod /
// go.tool-proto.mod / go.tool-golangci.mod), never as a tool directive or
// a require line in the root go.mod (D-03). google.golang.org/protobuf and
// connectrpc.com/connect are deliberately NOT here even though
// go.tool-proto.mod also pins their cmd/ tool binaries: both are
// legitimate RUNTIME dependencies of the main module (the generated
// .pb.go/.connect.go files import them), so root go.mod requiring them is
// correct, not a D-03 violation — only the buf CLI itself is pure build
// tooling with no runtime import anywhere in this module.
var forbiddenToolPackages = []string{
	"github.com/go-task/task",
	"github.com/goreleaser/goreleaser",
	"github.com/rhysd/actionlint",
	"github.com/golangci/golangci-lint",
	"github.com/bufbuild/buf",
}

// forbiddenTaskfileGateKeys are the two go-task fields that silently SKIP
// a task instead of failing it: status: (up-to-date short-circuit) and
// platforms: (host-OS restriction). D-11 rejects both by name — only
// preconditions: with a non-empty msg: is the sanctioned cross-toolchain
// gating mechanism (GOLDEN-01 silent-skip failure class).
var forbiddenTaskfileGateKeys = []string{"status", "platforms"}

// crossToolchainTokens are command-line tokens that mark a task as
// requiring a non-host toolchain — any task whose command text references
// one of these MUST carry a preconditions: entry with a non-empty msg:.
// mingw-w64's x86_64-w64-mingw32-gcc left this list with native Windows
// support (quick task 260807-gho); zig remains for the linux/arm64 cross.
var crossToolchainTokens = []string{"zig"}

// taskWrapperExpectedLegs is the literal D-10 fixture for the `test`
// wrapper's five host-only legs, compared as a sorted set against the
// wrapper's actual cmds: list in TestTaskfileWrapperIsSerial — so both a
// missing leg and an extra one fail the guard.
var taskWrapperExpectedLegs = []string{
	"test:daemon",
	"test:golden",
	"test:integration",
	"test:race",
	"test:unit",
	"test:wireoracle",
}

// inScopeJob names one job, by its YAML map key (not its `name:` display
// string), whose steps' run: bodies TestWorkflowRunBodiesInvokeTask holds to
// the single-definition property (D-01/D-02): every step's run: body must
// be exactly `task <target>` once comments and blank lines are stripped,
// unless the step is named in runBodyExceptions.
type inScopeJob struct {
	Workflow string // filename under workflowsDir, e.g. "ci.yml"
	JobID    string // job's YAML map key, e.g. "test"
}

// inScopeJobs is the literal D-01/D-02 fixture: every job this guard binds,
// by (workflow file, job ID). bench.yml and release.yml are deliberately
// NOT here — both carry their own documented D-01 exceptions decided in
// earlier plans of this phase (bench.yml's rebless/publish/diagnostic
// jobs' inline `go run ./tools/bench/runner` invocations, commented in-file
// above the rebless job; release.yml's native build matrix, D-08).
// Including either file here would fail this test for reasons the project
// already decided, not for a real regression. govulncheck is also excluded
// — it runs via `uses: golang/govulncheck-action`, an action with no run:
// body at all, so it has nothing for this guard to check.
var inScopeJobs = []inScopeJob{
	{Workflow: "ci.yml", JobID: "test"},
	{Workflow: "ci.yml", JobID: "actionlint"},
	{Workflow: "ci.yml", JobID: "goreleaser-check"},
	{Workflow: "ci.yml", JobID: "reproducibility"},
	{Workflow: "ci.yml", JobID: "perf-regression"},
	{Workflow: "ci.yml", JobID: "transcript-freeze"},
	{Workflow: "ci.yml", JobID: "tool-vuln"},
	{Workflow: "release-please.yml", JobID: "pretag-gate"},
	{Workflow: "corpora.yml", JobID: "corpora"},
	{Workflow: "corpora.yml", JobID: "golden"},
}

// runBodyException is one literal, reasoned carve-out from the
// single-definition property — a step whose run: body is legitimately not
// a `task <target>` call. T-10-07-01: every entry MUST carry a non-empty
// reason, and an entry naming a step that no longer exists in its
// (workflow, job) fails TestWorkflowRunBodiesInvokeTask — a stale exception
// silently widening the allowlist is exactly the failure mode this fixture
// shape closes.
type runBodyException struct {
	Workflow string
	Job      string
	Step     string
	Reason   string
}

// runBodyExceptions is the literal, exhaustive exception list for the
// in-scope jobs above. One entry, in ci.yml:
//   - "Compute determinism inputs" (job reproducibility) writes to the CI
//     step-output file ($GITHUB_OUTPUT) via `id: repro`; it has no meaning
//     outside a runner and produces no artifact a contributor would ever
//     invoke directly.
//
// A second entry ("Install mingw-w64", job test) retired with native
// Windows support (quick task 260807-gho) — the apt step it excepted no
// longer exists, and the loop at the bottom of
// TestWorkflowRunStepsInvokeTaskTargets fails any exception that matches
// no real step, so a stale entry cannot silently widen this allowlist.
var runBodyExceptions = []runBodyException{
	{
		Workflow: "ci.yml",
		Job:      "reproducibility",
		Step:     "Compute determinism inputs",
		Reason:   "writes to the CI step-output file ($GITHUB_OUTPUT) via id: repro; has no meaning outside a runner",
	},
}

// taskCallLineRe matches a run: body, once comments/blanks are stripped,
// that is EXACTLY a single `task <target>` invocation — the shape D-01
// mandates for every rewired step.
var taskCallLineRe = regexp.MustCompile(`^task\s+[A-Za-z0-9:_-]+$`)

// forbiddenGoInvocationRe matches a direct go build/test/vet/run/install
// invocation as a whole word — the specific command shapes D-01 moved into
// Taskfile.yml targets and forbids from reappearing inline in a rewired
// job's run: body.
var forbiddenGoInvocationRe = regexp.MustCompile(`\bgo (build|test|vet|run|install)\b`)

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

// parseWorkflowEnvValue returns the value of key from a workflow YAML
// source's WORKFLOW-LEVEL `env:` block only — a job-level or step-level
// `env:` entry with the same key (both indented deeper than column 0) must
// not satisfy it, since the pin this guards (MAINT-03) is a workflow-level
// value. Surrounding quotes are stripped. Returns a non-nil error naming
// key whenever the workflow declares no top-level env: block at all, the
// block exists but does not declare key, or declares key with an empty
// value — never a usable zero value on any of those misses (the CR-01
// defect class every parser in this file guards against).
func parseWorkflowEnvValue(src, key string) (string, error) {
	envBlockRe := regexp.MustCompile(`^env:\s*$`)
	keyRe := regexp.MustCompile(`^  (\S+):\s*(.*)$`)

	inBlock := false
	for _, line := range strings.Split(src, "\n") {
		if envBlockRe.MatchString(line) {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		// A non-blank line at column 0 (no leading whitespace) ends the
		// workflow-level env: block — e.g. the following `jobs:` key.
		if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
		if m := keyRe.FindStringSubmatch(line); m != nil && m[1] == key {
			v := strings.TrimSpace(m[2])
			v = strings.Trim(v, `"'`)
			if v == "" {
				return "", fmt.Errorf("parseWorkflowEnvValue: workflow-level env: key %q found but its value is empty", key)
			}
			return v, nil
		}
	}
	if !inBlock {
		return "", fmt.Errorf("parseWorkflowEnvValue: no workflow-level env: block found")
	}
	return "", fmt.Errorf("parseWorkflowEnvValue: workflow-level env: block found but no key %q present", key)
}

func mustWorkflowEnvValue(t *testing.T, src, key string) string {
	t.Helper()
	v, err := parseWorkflowEnvValue(src, key)
	if err != nil {
		t.Fatalf("mustWorkflowEnvValue: %v", err)
	}
	return v
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

// taskNameLineRe matches a top-level task-name key line under the
// `tasks:` section of Taskfile.yml: exactly two leading spaces, an
// identifier (letters, digits, colons, hyphens, underscores — covers
// namespaced names like test:unit), and a trailing colon with nothing
// else on the line.
var taskNameLineRe = regexp.MustCompile(`^  ([A-Za-z0-9:_-]+):\s*$`)

// parseTaskBlocks splits Taskfile.yml source src into per-task line
// blocks keyed by task name — everything from a top-level task-name line
// (see taskNameLineRe) up to, but not including, the next one. Returns a
// non-nil error if zero top-level task blocks are found under `tasks:` —
// the DESIRED failure mode for an empty, malformed, or task-less file,
// never a silently-empty map (the same CR-01 defect class every parser in
// this file guards against).
func parseTaskBlocks(src string) (map[string]string, error) {
	lines := strings.Split(src, "\n")
	blocks := make(map[string]string)

	inTasks := false
	curName := ""
	var curLines []string
	flush := func() {
		if curName != "" {
			blocks[curName] = strings.Join(curLines, "\n")
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "tasks:") {
			inTasks = true
			continue
		}
		if !inTasks {
			continue
		}
		if m := taskNameLineRe.FindStringSubmatch(line); m != nil {
			flush()
			curName = m[1]
			curLines = nil
			continue
		}
		if curName != "" {
			curLines = append(curLines, line)
		}
	}
	flush()

	if len(blocks) == 0 {
		return nil, fmt.Errorf("parseTaskBlocks: no top-level task block found under tasks:")
	}
	return blocks, nil
}

func mustParseTaskBlocks(t *testing.T, src string) map[string]string {
	t.Helper()
	v, err := parseTaskBlocks(src)
	if err != nil {
		t.Fatalf("mustParseTaskBlocks: %v", err)
	}
	return v
}

// descKeyLineRe matches a `desc:` key line within a task block (as produced
// by parseTaskBlocks), capturing its own leading indentation and whatever
// follows the colon on the same line — either an inline value or a YAML
// block-scalar indicator (>-, |-, >, |).
var descKeyLineRe = regexp.MustCompile(`^(\s*)desc:\s*(.*)$`)

// parseTaskDescription returns a task block's desc: value, handling both
// the inline form (`desc: some text`) and the folded/literal block-scalar
// form (`desc: >-` or `desc: |-` followed by lines indented deeper than
// the desc: key itself, joined with single spaces per YAML folding —
// exactly the form every multi-line desc: in this Taskfile uses). Returns
// a non-nil error when block declares no desc: key at all, or declares one
// whose resulting text is empty — never a usable zero value on either
// miss.
func parseTaskDescription(block string) (string, error) {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		m := descKeyLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent, rest := m[1], strings.TrimSpace(m[2])

		if rest != "" && rest != ">-" && rest != "|-" && rest != ">" && rest != "|" {
			v := strings.Trim(rest, `"'`)
			v = strings.TrimSpace(v)
			if v == "" {
				return "", fmt.Errorf("parseTaskDescription: desc: key found but its inline value is empty")
			}
			return v, nil
		}

		var cont []string
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			trimmed := strings.TrimSpace(next)
			if trimmed == "" {
				continue
			}
			nextIndent := len(next) - len(strings.TrimLeft(next, " "))
			if nextIndent <= len(indent) {
				break
			}
			cont = append(cont, trimmed)
		}
		v := strings.TrimSpace(strings.Join(cont, " "))
		if v == "" {
			return "", fmt.Errorf("parseTaskDescription: desc: key found but its block-scalar value is empty")
		}
		return v, nil
	}
	return "", fmt.Errorf("parseTaskDescription: no desc: key found in task block")
}

func mustTaskDescription(t *testing.T, block string) string {
	t.Helper()
	v, err := parseTaskDescription(block)
	if err != nil {
		t.Fatalf("mustTaskDescription: %v", err)
	}
	return v
}

// blockDeclaresKey reports whether block contains a line declaring the
// given YAML key at any indentation depth (leading whitespace, the
// literal key name, then a colon) — used for status:/platforms:/deps:
// detection regardless of nesting depth within the task.
func blockDeclaresKey(block, key string) bool {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `:\s*`)
	return re.MatchString(block)
}

// blockReferencesToken reports whether block's text contains token as a
// whole word (not merely a substring of a longer identifier).
func blockReferencesToken(block, token string) bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(token) + `\b`)
	return re.MatchString(block)
}

// parsePreconditionMessages returns every non-empty inline msg: value
// found inside block, in block order. Returns a non-nil error if block
// declares no preconditions: key at all, or declares one with zero
// non-empty msg: values — the DESIRED failure state for "this task needs
// an actionable message and does not have one," never a silently-empty
// slice.
func parsePreconditionMessages(block string) ([]string, error) {
	if !blockDeclaresKey(block, "preconditions") {
		return nil, fmt.Errorf("parsePreconditionMessages: no preconditions: key found")
	}
	msgRe := regexp.MustCompile(`(?m)^\s*msg:\s*(.+)$`)
	var msgs []string
	for _, m := range msgRe.FindAllStringSubmatch(block, -1) {
		v := strings.TrimSpace(m[1])
		v = strings.Trim(v, `"'`)
		v = strings.TrimSpace(v)
		if v != "" {
			msgs = append(msgs, v)
		}
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("parsePreconditionMessages: preconditions: key present but no non-empty inline msg: value found")
	}
	return msgs, nil
}

// parseTaskCallList returns, in block order, every sub-task name named by
// a `- task: <name>` cmds: entry within block. Returns a non-nil error if
// block names zero sub-tasks this way — the DESIRED failure state for a
// wrapper task whose cmds: list was emptied or converted to raw shell
// commands, never a silently-empty slice.
func parseTaskCallList(block string) ([]string, error) {
	callRe := regexp.MustCompile(`(?m)^\s*-\s+task:\s*(\S+)\s*$`)
	var calls []string
	for _, m := range callRe.FindAllStringSubmatch(block, -1) {
		calls = append(calls, m[1])
	}
	if len(calls) == 0 {
		return nil, fmt.Errorf("parseTaskCallList: no '- task: <name>' cmds entries found")
	}
	return calls, nil
}

func mustParseTaskCallList(t *testing.T, block string) []string {
	t.Helper()
	v, err := parseTaskCallList(block)
	if err != nil {
		t.Fatalf("mustParseTaskCallList: %v", err)
	}
	return v
}

// goreleaserBuildEntry mirrors the shape of one .goreleaser.yaml `builds:`
// list entry, capturing only the two fields TestCheckCrossMatchesGoreleaserTargets
// needs. goos/goarch are declared in INLINE flow-sequence form in this
// repo's .goreleaser.yaml (`goos: [linux]`, not block-sequence form) — a
// real YAML decoder handles both; a raw-text sequence-marker regex would
// silently match neither and produce a vacuously empty pair set (this
// project's own house rule against exactly that failure class).
type goreleaserBuildEntry struct {
	GOOS   []string `yaml:"goos"`
	GOARCH []string `yaml:"goarch"`
}

type goreleaserConfig struct {
	Builds []goreleaserBuildEntry `yaml:"builds"`
}

// parseGoreleaserCrossPairs decodes .goreleaser.yaml source src with a real
// YAML decoder and returns every GOOS/GOARCH pair named across its builds:
// entries, as "goos/goarch" strings. Returns a non-nil error if src fails
// to parse as YAML, declares zero builds: entries, or declares builds:
// entries none of which name both a goos and a goarch — never a usable
// zero value on any of those misses (the CR-01 defect class every parser
// in this file guards against).
func parseGoreleaserCrossPairs(src string) ([]string, error) {
	var cfg goreleaserConfig
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserCrossPairs: %w", err)
	}
	if len(cfg.Builds) == 0 {
		return nil, fmt.Errorf("parseGoreleaserCrossPairs: no builds: entries found")
	}
	var pairs []string
	for _, b := range cfg.Builds {
		for _, goos := range b.GOOS {
			for _, goarch := range b.GOARCH {
				pairs = append(pairs, goos+"/"+goarch)
			}
		}
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("parseGoreleaserCrossPairs: builds: entries present but none declare both goos: and goarch:")
	}
	return pairs, nil
}

func mustParseGoreleaserCrossPairs(t *testing.T, src string) []string {
	t.Helper()
	v, err := parseGoreleaserCrossPairs(src)
	if err != nil {
		t.Fatalf("mustParseGoreleaserCrossPairs: %v", err)
	}
	return v
}

// checkCrossForLineRe matches the `for pair in ...; do` line inside
// check:cross's command body — the single source of truth for which
// GOOS/GOARCH pairs the sweep covers.
var checkCrossForLineRe = regexp.MustCompile(`for pair in (.+); do`)

// parseCheckCrossPairs reads Taskfile.yml source src, locates its
// check:cross task block, and returns the GOOS/GOARCH pairs named on that
// block's `for pair in ...; do` line, in the order they appear. Returns a
// non-nil error if src has no check:cross task at all (naming it), or the
// task exists but its command text has no `for pair in ...; do` line, or
// that line names zero pairs — never a usable zero value on any of those
// misses.
func parseCheckCrossPairs(src string) ([]string, error) {
	blocks, err := parseTaskBlocks(src)
	if err != nil {
		return nil, fmt.Errorf("parseCheckCrossPairs: %w", err)
	}
	block, ok := blocks[checkCrossTaskID]
	if !ok {
		return nil, fmt.Errorf("parseCheckCrossPairs: no %q task found in Taskfile.yml", checkCrossTaskID)
	}
	m := checkCrossForLineRe.FindStringSubmatch(block)
	if m == nil {
		return nil, fmt.Errorf("parseCheckCrossPairs: %q task has no 'for pair in ...; do' line", checkCrossTaskID)
	}
	pairs := strings.Fields(m[1])
	if len(pairs) == 0 {
		return nil, fmt.Errorf("parseCheckCrossPairs: %q task's 'for pair in' line names zero pairs", checkCrossTaskID)
	}
	return pairs, nil
}

func mustParseCheckCrossPairs(t *testing.T, src string) []string {
	t.Helper()
	v, err := parseCheckCrossPairs(src)
	if err != nil {
		t.Fatalf("mustParseCheckCrossPairs: %v", err)
	}
	return v
}

// sortedPairSet returns pairs as a single comma-joined, sorted string —
// per this project's own house rule (~/.claude/rules/grepping.md) against
// exit-status/count-based comparison for exact multi-value equality:
// comparing one sorted string in one assertion carries "no more" and "no
// fewer" simultaneously, so both an omission and an addition fail.
func sortedPairSet(pairs []string) string {
	sorted := append([]string(nil), pairs...)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

// gateStanceWords are the two gate-stance words this repo's guards state
// explicitly in prose (Phase 3's D-03 transcript-freeze pattern,
// generalized by VULN-03): a job `name:` or step `name:` is a short,
// single-purpose display string, so naming neither word or naming both is
// not a legible single stance statement for that site.
var gateStanceWords = []string{"advisory", "blocking"}

// hasStanceWord reports whether text contains word as a case-insensitive
// whole word — matching the word itself, not a whole formatted string, so
// a reworded but still-honest name does not fail a stance guard while a
// deleted stance does.
func hasStanceWord(text, word string) bool {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`)
	return re.MatchString(text)
}

// parseGateStanceWord returns whichever of gateStanceWords appears in text
// as a case-insensitive whole word. Returns a non-nil error if text names
// neither word, or names both (an ambiguous, self-contradicting single-line
// stance statement) — never a usable zero value on either miss. Intended
// for short display strings (job/step name: values), not free-form prose
// like a task desc:, which may legitimately narrate a stance's history
// using both words.
func parseGateStanceWord(text string) (string, error) {
	var found []string
	for _, w := range gateStanceWords {
		if hasStanceWord(text, w) {
			found = append(found, w)
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("parseGateStanceWord: text states neither %q nor %q: %q", gateStanceWords[0], gateStanceWords[1], text)
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("parseGateStanceWord: text states both %v — an ambiguous, self-contradicting stance statement: %q", found, text)
	}
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

// TestGoreleaserPinParity is the MAINT-03 pin-parity guard: go.tool.mod's
// goreleaser require line and release.yml's workflow-level
// GORELEASER_VERSION must name the same version. Two independent pin sites
// drifted apart silently for five weeks before this test existed (19 days
// between ee258d9, which set release.yml's v2.17.0, and 82ffd60, which
// created go.tool.mod already carrying v2.17.1) — this test fails the
// build on any future divergence, naming both sites and both versions so
// the offending pair is immediately actionable.
func TestGoreleaserPinParity(t *testing.T) {
	toolModData, err := os.ReadFile(toolModfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", toolModfilePath, err)
	}
	toolVersion, err := parseGoModRequireVersion(string(toolModData), "github.com/goreleaser/goreleaser")
	if err != nil {
		t.Fatalf("%s: %v", toolModfilePath, err)
	}

	releaseData, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	releaseVersion := mustWorkflowEnvValue(t, string(releaseData), "GORELEASER_VERSION")

	toolNorm := strings.TrimPrefix(toolVersion, "v")
	releaseNorm := strings.TrimPrefix(releaseVersion, "v")

	if toolNorm != releaseNorm {
		t.Fatalf("GoReleaser pin mismatch (MAINT-03): %s requires goreleaser@%s but %s sets GORELEASER_VERSION=%s — the two pin sites must name the same version", toolModfilePath, toolVersion, releaseWorkflowPath, releaseVersion)
	}
}

// TestGateStancesStated is the VULN-03 stance guard.
//
// NOTE (D-04 superseded 2026-08-06 at 04-01's Task 1 checkpoint — see
// 04-CONTEXT.md and 04-01-SUMMARY.md): this test was originally designed
// to assert a DELIBERATE blocking-versus-advisory divergence between the
// tool-modfile scan (Taskfile.yml's `vuln` task / ci.yml's `tool-vuln`
// job) and the D-03 transcript-freeze guard (Phase 3's sibling
// advisory-stance pattern). Implementing the tool-modfile scan surfaced a
// real, permanently-unfixed, symbol-reachable vulnerability in
// goreleaser's own binary (GO-2026-5932) that would have made a blocking
// gate permanently red from its first CI run, so the maintainer demoted it
// to ADVISORY — now MATCHING transcript-freeze's stance rather than
// deliberately differing from it. This test is re-derived accordingly: it
// asserts all three sites state ADVISORY and that the tool-vuln job and
// transcript-freeze's advisory-guard step still agree with each other, so
// a future silent flip of either site back to BLOCKING — re-introducing an
// unstated stance mismatch — still fails loudly, naming the offending
// site.
func TestGateStancesStated(t *testing.T) {
	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(taskfileData))
	vulnBlock, ok := blocks["vuln"]
	if !ok {
		t.Fatalf("Taskfile.yml declares no top-level %q task", "vuln")
	}
	vulnDesc := mustTaskDescription(t, vulnBlock)
	if !hasStanceWord(vulnDesc, "advisory") {
		t.Errorf("Taskfile.yml vuln task's desc: does not state the ADVISORY stance (VULN-03): %q", vulnDesc)
	}

	ciPath := filepath.Join(workflowsDir, "ci.yml")
	ciData, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read %s: %v", ciPath, err)
	}
	ciSrc := string(ciData)

	jobNames := mustWorkflowJobNames(t, ciSrc)
	var toolVulnName string
	for _, n := range jobNames {
		if strings.Contains(n, "tool-vuln") {
			toolVulnName = n
			break
		}
	}
	if toolVulnName == "" {
		t.Fatalf("ci.yml declares no job name: containing %q (VULN-03)", "tool-vuln")
	}
	toolVulnStance, err := parseGateStanceWord(toolVulnName)
	if err != nil {
		t.Errorf("ci.yml tool-vuln job's name: %v", err)
	}

	tfSteps, err := parseWorkflowJobSteps(ciSrc, "transcript-freeze")
	if err != nil {
		t.Fatalf("ci.yml transcript-freeze: %v", err)
	}
	var tfStepName string
	for _, s := range tfSteps {
		if hasStanceWord(s.Name, "advisory") {
			tfStepName = s.Name
			break
		}
	}
	if tfStepName == "" {
		t.Fatalf("ci.yml transcript-freeze job has no step whose name: states the ADVISORY stance (D-03)")
	}
	tfStance, err := parseGateStanceWord(tfStepName)
	if err != nil {
		t.Errorf("ci.yml transcript-freeze step %q: %v", tfStepName, err)
	}

	if toolVulnStance != "" && toolVulnStance != "advisory" {
		t.Errorf("ci.yml tool-vuln job's name: states %q, want %q: %q", toolVulnStance, "advisory", toolVulnName)
	}
	if tfStance != "" && tfStance != "advisory" {
		t.Errorf("ci.yml transcript-freeze step %q states %q, want %q", tfStepName, tfStance, "advisory")
	}
	if toolVulnStance != "" && tfStance != "" && toolVulnStance != tfStance {
		t.Errorf("tool-vuln job states %q but transcript-freeze's advisory-guard step states %q — the two gates' stances no longer agree (D-04 superseded: they are now deliberately equal, not deliberately different)", toolVulnStance, tfStance)
	}
}

// TestToolModfilesRemainIsolated is the D-03 isolation guard: go.tool.mod
// and go.tool-lint.mod must exist as two distinct files, the root go.mod
// must declare no tool directive and require none of the three build-tool
// packages, and each tool modfile's header comment must be non-empty and
// state the isolation rationale — without that header the two files read
// as an accident and someone merges them.
// isolatedModfilePaths is every tool modfile TestToolModfilesRemainIsolated
// holds to the isolation property (D-03). 03-10-PLAN.md's own review
// (constraint: "hardcoded iteration set") flagged that this guard
// previously iterated a two-element slice hardcoded in-line — a new
// modfile absent from it was inspected by NOTHING and the test stayed
// green, which is why go.tool-proto.mod (present on disk since
// 01-01-PLAN.md) was never actually checked. Declaring the set here, as a
// named slice checked against disk by TestToolModfilesPopulationMatchesDisk
// below, closes both the golangci-lint gap this plan adds AND the
// pre-existing proto gap in the same change, rather than recording the
// latter as accepted debt.
var isolatedModfilePaths = []string{toolModfilePath, lintModfilePath, protoModfilePath, golangciModfilePath}

func TestToolModfilesRemainIsolated(t *testing.T) {
	infos := make([]os.FileInfo, len(isolatedModfilePaths))
	for i, path := range isolatedModfilePaths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		infos[i] = info
	}
	for i := range infos {
		for j := i + 1; j < len(infos); j++ {
			if os.SameFile(infos[i], infos[j]) {
				t.Fatalf("%s and %s resolve to the same file — they must be distinct modfiles (D-03)", isolatedModfilePaths[i], isolatedModfilePaths[j])
			}
		}
	}

	rootData, err := os.ReadFile(rootGoModPath)
	if err != nil {
		t.Fatalf("read %s: %v", rootGoModPath, err)
	}
	rootSrc := string(rootData)

	if pkgs, toolErr := parseGoModToolPackages(rootSrc); toolErr == nil {
		t.Fatalf("root go.mod declares a tool directive %v — build tools must live only in the isolated tool modfiles (D-03)", pkgs)
	}

	for _, pkg := range forbiddenToolPackages {
		if version, reqErr := parseGoModRequireVersion(rootSrc, pkg); reqErr == nil {
			t.Fatalf("root go.mod requires %s@%s directly — build tools must live only in the isolated tool modfiles (D-03)", pkg, version)
		}
	}

	for _, path := range isolatedModfilePaths {
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

// TestToolModfilesPopulationMatchesDisk is the positive-count guard
// 03-10-PLAN.md's review demanded: TestToolModfilesRemainIsolated's own
// exit status cannot report its own blindness to a modfile absent from
// isolatedModfilePaths (a new go.tool-*.mod that's simply never added to
// that slice is inspected by nothing, and the isolation test still
// passes). This test instead globs the actual population on disk and
// asserts the count matches exactly, so a future go.tool-*.mod landing
// without a matching isolatedModfilePaths entry fails LOUDLY here rather
// than silently falling behind — the same population-vs-assertion gap
// this plan closed for go.tool-proto.mod, guarded from recurring.
func TestToolModfilesPopulationMatchesDisk(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(rootGoModPath), "go.tool*.mod"))
	if err != nil {
		t.Fatalf("glob go.tool*.mod: %v", err)
	}
	if len(matches) != len(isolatedModfilePaths) {
		t.Fatalf("found %d go.tool*.mod file(s) on disk (%v) but isolatedModfilePaths names %d (%v) — a new tool modfile was added without registering it here, or a registered one no longer exists on disk", len(matches), matches, len(isolatedModfilePaths), isolatedModfilePaths)
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
		{
			name: "parseWorkflowEnvValue: empty input",
			fn: func() error {
				_, err := parseWorkflowEnvValue("", "GORELEASER_VERSION")
				return err
			},
		},
		{
			name: "parseTaskDescription: empty input",
			fn: func() error {
				_, err := parseTaskDescription("")
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

// TestParseWorkflowEnvValue_NoWorkflowLevelEnvBlockIsError is the edge case
// named by this task's own <behavior>: a workflow source that declares
// jobs: but no workflow-level env: block at all must produce a non-nil
// error, never an empty string that would compare equal to nothing.
func TestParseWorkflowEnvValue_NoWorkflowLevelEnvBlockIsError(t *testing.T) {
	src := "name: x\non:\n  push:\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - name: s\n        run: task build\n"
	if _, err := parseWorkflowEnvValue(src, "GORELEASER_VERSION"); err == nil {
		t.Fatalf("parseWorkflowEnvValue(%q, %q): expected a non-nil error for a workflow with jobs but no workflow-level env: block, got nil", src, "GORELEASER_VERSION")
	}
}

// TestTaskfileGatesFailLoud is the D-11 repudiation guard: it reads the
// real, on-disk Taskfile.yml and asserts (1) no task declares status: or
// platforms: — both silently skip a task on a non-matching condition
// instead of failing it, the exact GOLDEN-01 failure class this repo has
// a documented Critical-severity history with — and (2) every task whose
// command text references a cross-toolchain binary (mingw-w64's
// x86_64-w64-mingw32-gcc, zig) declares at least one preconditions: entry
// with a non-empty msg:, so a missing toolchain fails loud with an
// actionable instruction rather than skipping silently.
func TestTaskfileGatesFailLoud(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(data))

	for name, block := range blocks {
		for _, key := range forbiddenTaskfileGateKeys {
			if blockDeclaresKey(block, key) {
				t.Errorf("task %q declares a %q key — both status: and platforms: silently skip a task instead of failing it (D-11, GOLDEN-01 failure class); use preconditions: with msg: instead", name, key)
			}
		}

		var referenced []string
		for _, token := range crossToolchainTokens {
			if blockReferencesToken(block, token) {
				referenced = append(referenced, token)
			}
		}
		if len(referenced) == 0 {
			continue
		}
		if _, msgErr := parsePreconditionMessages(block); msgErr != nil {
			t.Errorf("task %q references cross-toolchain token(s) %v but has no preconditions: entry with a non-empty msg: (%v) — a missing toolchain would fail without an actionable message, or worse, silently skip", name, referenced, msgErr)
		}
	}
}

// TestTaskfileGatesFailLoud_EmptyFileIsError is the edge case: a Taskfile
// source with no `tasks:` section at all (or an empty one) must produce a
// non-nil error from parseTaskBlocks, never a vacuous pass over zero
// tasks.
func TestTaskfileGatesFailLoud_EmptyFileIsError(t *testing.T) {
	cases := []string{
		"",
		"version: \"3\"\n",
		"version: \"3\"\ntasks:\n",
	}
	for _, src := range cases {
		if _, err := parseTaskBlocks(src); err == nil {
			t.Fatalf("parseTaskBlocks(%q): expected a non-nil error for a task-less Taskfile source, got nil", src)
		}
	}
}

// TestTaskfileWrapperIsSerial is the D-10 wrapper guard: the `test`
// wrapper task must declare no deps: key (go-task runs deps: entries
// concurrently, which would reintroduce exactly the cross-test contention
// test:daemon's and test:race's -p 1 flags exist to prevent) and its
// cmds: list must name exactly the five host-only legs, compared as a
// sorted set against the literal D-10 fixture so both a missing leg and
// an extra one fail this test.
func TestTaskfileWrapperIsSerial(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(data))

	block, ok := blocks["test"]
	if !ok {
		t.Fatalf("Taskfile.yml declares no top-level %q task", "test")
	}

	if blockDeclaresKey(block, "deps") {
		t.Fatalf("task %q declares a deps: key — go-task runs deps: entries concurrently, which would reintroduce the cross-test contention test:daemon/test:race's -p 1 flags exist to prevent; use a serial cmds: list of '- task: <name>' entries instead", "test")
	}

	actual := mustParseTaskCallList(t, block)
	gotSorted := append([]string(nil), actual...)
	sort.Strings(gotSorted)
	wantSorted := append([]string(nil), taskWrapperExpectedLegs...)
	sort.Strings(wantSorted)

	if !reflect.DeepEqual(gotSorted, wantSorted) {
		t.Fatalf("task %q's cmds: leg set = %v, want (sorted-set-equal to) %v — a missing or extra leg changes the coverage of a contributor's `task test` run", "test", gotSorted, wantSorted)
	}
}

// TestTaskfileWrapperIsSerial_MissingWrapperIsError is the edge case: a
// Taskfile source that parses (has at least one top-level task) but has
// no `test` task at all must be distinguishable from "test wrapper exists
// but its leg list is wrong" — parseTaskBlocks must not synthesize a
// zero-value block for an absent key.
func TestTaskfileWrapperIsSerial_MissingWrapperIsError(t *testing.T) {
	src := "version: \"3\"\ntasks:\n  build:\n    cmds:\n      - go build ./...\n"
	blocks := mustParseTaskBlocks(t, src)
	if _, ok := blocks["test"]; ok {
		t.Fatalf("parseTaskBlocks(%q): expected no %q key in the returned map, got one", src, "test")
	}
}

// TestCheckCrossMatchesGoreleaserTargets is the D-15/T-10-05-03 divergence
// guard: the set of GOOS/GOARCH pairs Taskfile.yml's check:cross target
// sweeps must be set-equal to the set of {goos, goarch} pairs across
// .goreleaser.yaml's builds: entries. Compared as one sorted-string
// equality so both a pair present in one and absent from the other fails,
// in either direction, with the offending pair(s) named in the failure
// message — adding a release target to .goreleaser.yaml without adding it
// to the pre-tag sweep (or vice versa) now fails this test.
func TestCheckCrossMatchesGoreleaserTargets(t *testing.T) {
	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	taskPairs := mustParseCheckCrossPairs(t, string(taskfileData))

	goreleaserData, err := os.ReadFile(goreleaserPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserPath, err)
	}
	grPairs := mustParseGoreleaserCrossPairs(t, string(goreleaserData))

	gotSorted := sortedPairSet(taskPairs)
	wantSorted := sortedPairSet(grPairs)
	if gotSorted != wantSorted {
		t.Fatalf("check:cross pairs %v are not set-equal to .goreleaser.yaml builds: pairs %v — got sorted set %q, want %q (a pair present in one and absent from the other, in either direction, is a coverage divergence between the pre-tag sweep and the release build matrix)", taskPairs, grPairs, gotSorted, wantSorted)
	}
}

// TestCheckCrossMatchesGoreleaserTargets_EmptyBuildsIsError is the edge
// case named by the plan's own acceptance criteria: a .goreleaser.yaml
// with zero builds: entries must produce a non-nil error from
// parseGoreleaserCrossPairs, never an empty-set match against an equally
// empty (or absent) sweep — an empty-vs-empty comparison would pass
// vacuously without asserting anything.
func TestCheckCrossMatchesGoreleaserTargets_EmptyBuildsIsError(t *testing.T) {
	src := "version: 2\nproject_name: codegraph\nbuilds: []\n"
	if _, err := parseGoreleaserCrossPairs(src); err == nil {
		t.Fatalf("parseGoreleaserCrossPairs(%q): expected a non-nil error for zero builds: entries, got nil", src)
	}
}

// TestCheckCrossMatchesGoreleaserTargets_MissingCheckCrossIsError is the
// edge case named by the plan's own acceptance criteria: a Taskfile.yml
// with no check:cross target must produce a non-nil error from
// parseCheckCrossPairs naming the missing target, never a silently-empty
// pair slice.
func TestCheckCrossMatchesGoreleaserTargets_MissingCheckCrossIsError(t *testing.T) {
	src := "version: \"3\"\ntasks:\n  build:\n    cmds:\n      - go build ./...\n"
	_, err := parseCheckCrossPairs(src)
	if err == nil {
		t.Fatalf("parseCheckCrossPairs(%q): expected a non-nil error for a Taskfile with no check:cross task, got nil", src)
	}
	if !strings.Contains(err.Error(), checkCrossTaskID) {
		t.Fatalf("parseCheckCrossPairs(%q) error does not name the missing task %q: %v", src, checkCrossTaskID, err)
	}
}

// TestParseGoreleaserCrossPairs_InlineFlowSequence proves the parser is
// reading .goreleaser.yaml's ACTUAL syntax, not a syntax this repo doesn't
// use: goos/goarch are declared in inline flow-sequence form (`goos:
// [linux]`), not block-sequence form, in the real file. A parser that
// silently yields an empty set here would make the whole
// TestCheckCrossMatchesGoreleaserTargets comparison vacuous — this repo
// has already shipped one mutation test that no-opped for exactly this
// reason (Task 3's own read_first note).
func TestParseGoreleaserCrossPairs_InlineFlowSequence(t *testing.T) {
	src := "builds:\n  - id: example\n    goos: [linux]\n    goarch: [amd64]\n"
	pairs := mustParseGoreleaserCrossPairs(t, src)
	if len(pairs) == 0 {
		t.Fatalf("parseGoreleaserCrossPairs(%q): expected a non-empty pair set for inline flow-sequence goos/goarch, got empty", src)
	}
	want := "linux/amd64"
	found := false
	for _, p := range pairs {
		if p == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("parseGoreleaserCrossPairs(%q) = %v, want to contain %q", src, pairs, want)
	}
}

// TestCheckCrossParsersFailLoudly extends TestTaskfileShapeHelpersFailLoudly
// coverage to this test's two new parsers: every pure parse core must
// return a non-nil error — never a usable zero value — when fed empty
// input.
func TestCheckCrossParsersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "parseGoreleaserCrossPairs: empty input",
			fn: func() error {
				_, err := parseGoreleaserCrossPairs("")
				return err
			},
		},
		{
			name: "parseCheckCrossPairs: empty input",
			fn: func() error {
				_, err := parseCheckCrossPairs("")
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

// --- TestWorkflowRunBodiesInvokeTask: workflow step decoding -----------

// workflowRunStep is the subset of a GitHub Actions workflow step this
// guard needs: its display name and its run: body (empty for a `uses:`
// step, which this guard does not constrain — D-01 only binds run: bodies).
type workflowRunStep struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
	Uses string `yaml:"uses"`
}

type workflowJobYAML struct {
	Steps []workflowRunStep `yaml:"steps"`
}

type workflowFileYAML struct {
	Jobs map[string]workflowJobYAML `yaml:"jobs"`
}

// parseWorkflowJobSteps decodes workflow YAML source src with a real YAML
// decoder and returns the ordered steps of the job keyed jobID. Returns a
// non-nil error if src fails to parse as YAML, declares no jobs: at all,
// names no job matching jobID, or that job declares zero steps — never a
// usable empty slice on any of those misses (the CR-01 defect class every
// parser in this file guards against).
func parseWorkflowJobSteps(src, jobID string) ([]workflowRunStep, error) {
	var wf workflowFileYAML
	if err := yaml.Unmarshal([]byte(src), &wf); err != nil {
		return nil, fmt.Errorf("parseWorkflowJobSteps: %w", err)
	}
	if len(wf.Jobs) == 0 {
		return nil, fmt.Errorf("parseWorkflowJobSteps: no jobs: found in workflow source")
	}
	job, ok := wf.Jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("parseWorkflowJobSteps: no job %q found in workflow source", jobID)
	}
	if len(job.Steps) == 0 {
		return nil, fmt.Errorf("parseWorkflowJobSteps: job %q declares zero steps", jobID)
	}
	return job.Steps, nil
}

// stripRunBodyNoise removes blank lines and full-line shell comments from a
// step's run: body, returning the remaining lines rejoined with "\n". Only
// a line whose first non-whitespace character is `#` is treated as a
// comment — a `#` inside a quoted string or a `${var#pattern}` shell
// expansion is NOT stripped. The risk here runs the opposite direction from
// this guard's own house rule about comments masquerading as invocations:
// over-stripping would silently eat a real command line rather than a
// comment, which is exactly as dangerous to a guard's correctness.
func stripRunBodyNoise(body string) string {
	var kept []string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		kept = append(kept, trimmed)
	}
	return strings.Join(kept, "\n")
}

// runBodyExceptionKey builds the composite lookup key runBodyExceptions and
// TestWorkflowRunBodiesInvokeTask's scan loop both use to match a step to
// its exception entry.
func runBodyExceptionKey(workflow, job, step string) string {
	return workflow + "\x00" + job + "\x00" + step
}

// checkStepInvokesTask validates one step from an in-scope job: a step with
// no run: body (a `uses:` step) is unconstrained; an excepted step (present
// in exceptions) is unconstrained; every other step's run: body must equal
// a single `task <target>` line once stripRunBodyNoise runs. Returns a
// non-nil error naming workflow, job, and step on violation — with the
// specific offending `go <verb>` invocation named when one is present,
// since that is the actionable fact a contributor reading the failure
// needs to remove.
func checkStepInvokesTask(workflow, job string, step workflowRunStep, exceptions map[string]string) error {
	if step.Run == "" {
		return nil
	}
	if _, excepted := exceptions[runBodyExceptionKey(workflow, job, step.Name)]; excepted {
		return nil
	}
	stripped := stripRunBodyNoise(step.Run)
	if taskCallLineRe.MatchString(stripped) {
		return nil
	}
	if verb := forbiddenGoInvocationRe.FindString(stripped); verb != "" {
		return fmt.Errorf("%s job %q step %q invokes %q directly instead of a task target — single-definition property violated (D-01)", workflow, job, step.Name, verb)
	}
	return fmt.Errorf("%s job %q step %q's run: body is not a single 'task <target>' call after stripping comments/blanks (D-01): %q", workflow, job, step.Name, stripped)
}

// validateRunBodyExceptions asserts every entry in exceptions carries a
// non-empty reason. Returns a non-nil error naming the first offending
// entry — never a silent skip (T-10-07-01: a reasonless exception is
// indistinguishable from an accidental allowlist widening).
func validateRunBodyExceptions(exceptions []runBodyException) error {
	for i, exc := range exceptions {
		if strings.TrimSpace(exc.Reason) == "" {
			return fmt.Errorf("validateRunBodyExceptions: exceptions[%d] (%s/%s/%q) has an empty reason", i, exc.Workflow, exc.Job, exc.Step)
		}
	}
	return nil
}

// --- TestWorkflowRunBodiesInvokeTask ------------------------------------

// TestWorkflowRunBodiesInvokeTask is the D-01/D-02 single-definition guard:
// across inScopeJobs, every step's run: body must be exactly `task
// <target>` once comments/blanks are stripped, unless the step is named in
// runBodyExceptions (every entry of which must carry a non-empty reason and
// must actually match a real step — see the two checks below). This is
// what makes DEV-01 durable rather than a one-time cleanup: without it, the
// next contributor adds an inline `go test` to a workflow and the
// single-definition property dies silently.
func TestWorkflowRunBodiesInvokeTask(t *testing.T) {
	if err := validateRunBodyExceptions(runBodyExceptions); err != nil {
		t.Fatalf("runBodyExceptions: %v", err)
	}
	if len(inScopeJobs) == 0 {
		t.Fatalf("inScopeJobs fixture is empty — this guard would vacuously pass over zero jobs")
	}

	exceptionSet := make(map[string]string, len(runBodyExceptions))
	for _, exc := range runBodyExceptions {
		exceptionSet[runBodyExceptionKey(exc.Workflow, exc.Job, exc.Step)] = exc.Reason
	}
	matched := make(map[string]bool, len(runBodyExceptions))

	for _, ij := range inScopeJobs {
		path := filepath.Join(workflowsDir, ij.Workflow)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		steps, err := parseWorkflowJobSteps(string(data), ij.JobID)
		if err != nil {
			t.Fatalf("%s job %q: %v", ij.Workflow, ij.JobID, err)
		}
		for _, step := range steps {
			key := runBodyExceptionKey(ij.Workflow, ij.JobID, step.Name)
			if _, excepted := exceptionSet[key]; excepted {
				matched[key] = true
			}
			if err := checkStepInvokesTask(ij.Workflow, ij.JobID, step, exceptionSet); err != nil {
				t.Error(err)
			}
		}
	}

	for _, exc := range runBodyExceptions {
		key := runBodyExceptionKey(exc.Workflow, exc.Job, exc.Step)
		if !matched[key] {
			t.Errorf("exception %s/%s/%q was never matched against a real step in an in-scope job — a stale exception silently widens the allowlist (T-10-07-01); fix the step name or remove the entry", exc.Workflow, exc.Job, exc.Step)
		}
	}
}

// usesOnlyJobException names one job, by (workflow, job ID), that is
// legitimately absent from inScopeJobs because every one of its steps is
// a `uses:` step with no run: body at all — nothing for
// TestWorkflowRunBodiesInvokeTask to check. Mirrors runBodyException's
// own carve-out shape (IN-05) so a stale entry — the job no longer
// exists, or has gained a real run: step — fails loudly here rather than
// silently widening the population TestInScopeJobsPopulationMatchesDisk
// trusts.
type usesOnlyJobException struct {
	Workflow string
	JobID    string
	Reason   string
}

var usesOnlyJobExceptions = []usesOnlyJobException{
	{
		Workflow: "ci.yml",
		JobID:    "govulncheck",
		Reason:   "runs via `uses: golang/govulncheck-action`, an action with no run: body at all",
	},
	{
		Workflow: "release-please.yml",
		JobID:    "release-please",
		Reason:   "every step is `uses:` (create-github-app-token, release-please-action) — no run: body at all",
	},
}

// inScopeWorkflowFiles is the SAME set of workflow files inScopeJobs
// already covers — bench.yml and release.yml carry their own documented
// D-01 exceptions decided in earlier plans (see inScopeJobs's own doc
// comment) and are deliberately excluded from this population check too,
// for the identical reason.
var inScopeWorkflowFiles = []string{"ci.yml", "release-please.yml", "corpora.yml"}

// validateUsesOnlyJobExceptions proves every usesOnlyJobExceptions entry
// still exists on disk, still carries a non-empty reason, and still
// declares ZERO run: steps — mirroring validateRunBodyExceptions's own
// "a stale exception cannot silently widen" discipline. A job that gains
// even one run: step must move to inScopeJobs and be held to
// TestWorkflowRunBodiesInvokeTask like every other in-scope job, not stay
// carved out here.
func validateUsesOnlyJobExceptions(excs []usesOnlyJobException) error {
	for _, exc := range excs {
		if strings.TrimSpace(exc.Reason) == "" {
			return fmt.Errorf("%s/%s: empty reason", exc.Workflow, exc.JobID)
		}
		path := filepath.Join(workflowsDir, exc.Workflow)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s/%s: read %s: %w", exc.Workflow, exc.JobID, path, err)
		}
		var wf workflowFileYAML
		if err := yaml.Unmarshal(data, &wf); err != nil {
			return fmt.Errorf("%s/%s: parse %s: %w", exc.Workflow, exc.JobID, path, err)
		}
		job, ok := wf.Jobs[exc.JobID]
		if !ok {
			return fmt.Errorf("%s/%s: job no longer exists in %s", exc.Workflow, exc.JobID, path)
		}
		if len(job.Steps) == 0 {
			return fmt.Errorf("%s/%s: declares zero steps", exc.Workflow, exc.JobID)
		}
		for _, step := range job.Steps {
			if strings.TrimSpace(step.Run) != "" {
				return fmt.Errorf("%s/%s: step %q now has a run: body — this job is no longer uses:-only and must move to inScopeJobs", exc.Workflow, exc.JobID, step.Name)
			}
		}
	}
	return nil
}

// TestInScopeJobsPopulationMatchesDisk is IN-05's disk-binding fix,
// mirroring TestToolModfilesPopulationMatchesDisk's own pattern (that
// test's own doc comment is this one's precedent, named directly in
// 03-REVIEW.md's IN-05 finding). Before this fix, inScopeJobs was a
// hand-enumerated fixture with only a non-empty guard
// (TestWorkflowRunBodiesInvokeTask's `len(inScopeJobs) == 0` check) —
// nothing asserted that every job actually declared in
// inScopeWorkflowFiles appears in it. A new CI job added to ci.yml,
// release-please.yml, or corpora.yml without a matching inScopeJobs entry
// was bound by nothing and the suite stayed green — the memory-v4zqxrz6b3
// shape: a hand-enumerated population narrows silently because a new
// subject passes by being absent.
func TestInScopeJobsPopulationMatchesDisk(t *testing.T) {
	if err := validateUsesOnlyJobExceptions(usesOnlyJobExceptions); err != nil {
		t.Fatalf("usesOnlyJobExceptions: %v", err)
	}

	want := make(map[string]bool, len(inScopeJobs))
	for _, ij := range inScopeJobs {
		want[ij.Workflow+"/"+ij.JobID] = true
	}
	excluded := make(map[string]bool, len(usesOnlyJobExceptions))
	for _, exc := range usesOnlyJobExceptions {
		excluded[exc.Workflow+"/"+exc.JobID] = true
	}

	var onDisk []string
	for _, wf := range inScopeWorkflowFiles {
		path := filepath.Join(workflowsDir, wf)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var parsed workflowFileYAML
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if len(parsed.Jobs) == 0 {
			t.Fatalf("%s declares no jobs: at all — this guard would vacuously pass over zero jobs", path)
		}
		for jobID := range parsed.Jobs {
			onDisk = append(onDisk, wf+"/"+jobID)
		}
	}
	sort.Strings(onDisk)

	var missing, extra []string
	seen := make(map[string]bool, len(onDisk))
	for _, key := range onDisk {
		seen[key] = true
		if excluded[key] {
			continue
		}
		if !want[key] {
			missing = append(missing, key)
		}
	}
	for key := range want {
		if !seen[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 {
		t.Errorf("job(s) on disk in %v not present in inScopeJobs and not in usesOnlyJobExceptions: %v — a new CI job was added without registering it here", inScopeWorkflowFiles, missing)
	}
	if len(extra) > 0 {
		t.Errorf("inScopeJobs names job(s) that no longer exist on disk: %v", extra)
	}
}

// TestRunBodyExceptionsHaveReasons_EmptyReasonIsError proves
// validateRunBodyExceptions actually catches what it claims to: a blank
// reason must produce a non-nil error, never a silent pass.
func TestRunBodyExceptionsHaveReasons_EmptyReasonIsError(t *testing.T) {
	bad := []runBodyException{{Workflow: "ci.yml", Job: "test", Step: "some step", Reason: "  "}}
	if err := validateRunBodyExceptions(bad); err == nil {
		t.Fatalf("validateRunBodyExceptions: expected a non-nil error for an exception with a blank reason, got nil")
	}
}

// TestParseWorkflowJobSteps_MissingJobIsError is the edge case: a job ID
// absent from the workflow's jobs: map must produce a non-nil error naming
// it, never a silently-empty step slice.
func TestParseWorkflowJobSteps_MissingJobIsError(t *testing.T) {
	src := "jobs:\n  build:\n    steps:\n      - name: X\n        run: task build\n"
	_, err := parseWorkflowJobSteps(src, "does-not-exist")
	if err == nil {
		t.Fatalf("parseWorkflowJobSteps(%q): expected a non-nil error for a missing job, got nil", "does-not-exist")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("parseWorkflowJobSteps error does not name the missing job: %v", err)
	}
}

// TestParseWorkflowJobSteps_NoJobsIsError is the edge case named directly
// in this task's <behavior>: a workflow with zero jobs: at all must return
// a non-nil error, never a vacuous pass over an empty job set.
func TestParseWorkflowJobSteps_NoJobsIsError(t *testing.T) {
	src := "name: empty\non:\n  push:\n"
	if _, err := parseWorkflowJobSteps(src, "test"); err == nil {
		t.Fatalf("parseWorkflowJobSteps(%q, %q): expected a non-nil error for a workflow with no jobs: at all, got nil", src, "test")
	}
}

// TestParseWorkflowJobSteps_ZeroStepsIsError is the edge case: a job that
// exists but declares zero steps must produce a non-nil error, never an
// empty-but-valid step slice.
func TestParseWorkflowJobSteps_ZeroStepsIsError(t *testing.T) {
	src := "jobs:\n  build:\n    runs-on: ubuntu-latest\n"
	if _, err := parseWorkflowJobSteps(src, "build"); err == nil {
		t.Fatalf("parseWorkflowJobSteps(%q, %q): expected a non-nil error for a job with zero steps, got nil", src, "build")
	}
}

// TestCheckStepInvokesTask_ForbiddenGoInvocationNamesVerb proves the
// forbidden-invocation branch actually names the offending verb, not just a
// generic shape mismatch — the failure message a contributor reads should
// point at exactly what to remove.
func TestCheckStepInvokesTask_ForbiddenGoInvocationNamesVerb(t *testing.T) {
	step := workflowRunStep{Name: "inline build", Run: "set -euo pipefail\ngo build ./...\n"}
	err := checkStepInvokesTask("ci.yml", "test", step, map[string]string{})
	if err == nil {
		t.Fatalf("checkStepInvokesTask: expected a non-nil error for an inline go build invocation, got nil")
	}
	if !strings.Contains(err.Error(), "go build") {
		t.Fatalf("checkStepInvokesTask error does not name the offending invocation: %v", err)
	}
}

// TestCheckStepInvokesTask_ExceptedStepPasses proves an excepted step is
// unconstrained regardless of its run: body's shape.
func TestCheckStepInvokesTask_ExceptedStepPasses(t *testing.T) {
	step := workflowRunStep{Name: "apt install", Run: "sudo apt-get install -y gcc-mingw-w64-x86-64"}
	exceptions := map[string]string{runBodyExceptionKey("ci.yml", "test", "apt install"): "installs a system package"}
	if err := checkStepInvokesTask("ci.yml", "test", step, exceptions); err != nil {
		t.Fatalf("checkStepInvokesTask: expected a nil error for an excepted step, got %v", err)
	}
}

// TestCheckStepInvokesTask_UsesStepPasses proves a `uses:` step (empty
// run: body) is unconstrained — this guard only binds run: bodies.
func TestCheckStepInvokesTask_UsesStepPasses(t *testing.T) {
	step := workflowRunStep{Name: "Checkout", Run: ""}
	if err := checkStepInvokesTask("ci.yml", "test", step, map[string]string{}); err != nil {
		t.Fatalf("checkStepInvokesTask: expected a nil error for a uses: step (empty run: body), got %v", err)
	}
}

// TestStripRunBodyNoise_KeepsHashInsideCommand proves stripRunBodyNoise
// does not over-strip: a `#` that is not the first non-whitespace character
// on a line (e.g. inside `${var#prefix}`) must be preserved, never treated
// as a comment marker.
func TestStripRunBodyNoise_KeepsHashInsideCommand(t *testing.T) {
	body := "# a real comment\ntask build\nVAR=${x#prefix}\n"
	got := stripRunBodyNoise(body)
	if strings.Contains(got, "a real comment") {
		t.Fatalf("stripRunBodyNoise(%q) = %q, still contains the comment line", body, got)
	}
	if !strings.Contains(got, `${x#prefix}`) {
		t.Fatalf("stripRunBodyNoise(%q) = %q, over-stripped a non-comment line containing '#'", body, got)
	}
}

// --- CONTRIBUTING.md task target references (DEV-01) ---------------------

// contributingPath is the on-disk path to CONTRIBUTING.md relative to this
// package.
const contributingPath = "../../CONTRIBUTING.md"

// parseContributingTaskTargets extracts every `task <target>` reference from
// CONTRIBUTING.md source src, matching only backtick-enclosed patterns (to
// avoid false positives from prose like "the task system").
// Returns a non-nil error if zero targets are found — a CONTRIBUTING.md that
// stops naming any task target silently violates DEV-01, so empty must FAIL,
// never pass vacuously.
func parseContributingTaskTargets(src string) ([]string, error) {
	// Match backtick-enclosed `task <target>` patterns.
	// The regex captures the target name after the word "task " within backticks.
	re := regexp.MustCompile("`task\\s+([A-Za-z0-9:_-]+)`")
	matches := re.FindAllStringSubmatch(src, -1)

	targetSet := make(map[string]bool)
	for _, m := range matches {
		target := m[1]
		targetSet[target] = true
	}

	// Extract unique targets in sorted order for deterministic output.
	var targets []string
	for t := range targetSet {
		targets = append(targets, t)
	}
	sort.Strings(targets)

	if len(targets) == 0 {
		return nil, fmt.Errorf("parseContributingTaskTargets: no backtick-enclosed `task <target>` patterns found")
	}
	return targets, nil
}

func mustParseContributingTaskTargets(t *testing.T, src string) []string {
	t.Helper()
	v, err := parseContributingTaskTargets(src)
	if err != nil {
		t.Fatalf("mustParseContributingTaskTargets: %v", err)
	}
	return v
}

// contains reports whether slice contains s as an element.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// TestContributingReferencesRealTaskTargets is the DEV-01 documentation-drift
// guard: it reads CONTRIBUTING.md and asserts every `task <target>` reference
// names a real, top-level task in Taskfile.yml. A pointer to a non-existent
// target silently rotts — this test fails the build on any such drift, naming
// the offending target and explaining the DEV-01 consequence.
func TestContributingReferencesRealTaskTargets(t *testing.T) {
	contributingData, err := os.ReadFile(contributingPath)
	if err != nil {
		t.Fatalf("read %s: %v", contributingPath, err)
	}
	targets := mustParseContributingTaskTargets(t, string(contributingData))

	if len(targets) == 0 {
		t.Fatalf("parseContributingTaskTargets returned an empty list; non-empty set should have been enforced by the parser")
	}

	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(taskfileData))

	var missing []string
	for _, target := range targets {
		if _, ok := blocks[target]; !ok {
			missing = append(missing, target)
		}
	}
	sort.Strings(missing)

	if len(missing) > 0 {
		t.Fatalf("CONTRIBUTING.md names task target(s) that do not exist in Taskfile.yml: %v — documentation drift violates DEV-01 (CONTRIBUTING.md points contributors at the targets). Update CONTRIBUTING.md to remove or correct the reference.", missing)
	}
}

// TestContributingReferencesRealTaskTargets_UnknownTargetIsError is the
// non-vacuity companion: it feeds parseContributingTaskTargets a synthetic
// input naming a target that does not exist in the real Taskfile.yml and
// verifies the main test would catch it. This proves the guard can actually
// FIRE on a real regression.
func TestContributingReferencesRealTaskTargets_UnknownTargetIsError(t *testing.T) {
	// Synthetic CONTRIBUTING.md source mentioning a real target and a fake one.
	syntheticContributing := `## Building

You need a C toolchain. Every command is a task target.

- ` + "`task build`" + ` — build everything
- ` + "`task fake-nonexistent-target-xyz`" + ` — this target does not exist

See Taskfile.yml for the complete list.
`

	// Read the real Taskfile.yml to get its actual target set.
	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(taskfileData))

	// Parse the synthetic CONTRIBUTING — this should succeed because it finds
	// at least one target (build, fake-nonexistent-target-xyz).
	targets, err := parseContributingTaskTargets(syntheticContributing)
	if err != nil {
		t.Fatalf("parseContributingTaskTargets(%q): expected success on synthetic input with multiple targets, got %v", syntheticContributing, err)
	}

	if len(targets) == 0 {
		t.Fatalf("parseContributingTaskTargets returned empty list for synthetic input with targets; expected at least two targets")
	}

	// Now verify that a check against the real Taskfile would CATCH the unknown
	// target. This proves that TestContributingReferencesRealTaskTargets, when
	// run on the real files, would fail if CONTRIBUTING.md named an unknown
	// target.
	var missing []string
	for _, target := range targets {
		if _, ok := blocks[target]; !ok {
			missing = append(missing, target)
		}
	}

	// We expect fake-nonexistent-target-xyz to be in the missing set.
	if len(missing) == 0 {
		t.Fatalf("TestContributingReferencesRealTaskTargets_UnknownTargetIsError: synthetic input named a fake target, but the guard found no missing targets — the guard cannot fire, so it would be vacuous")
	}

	if !contains(missing, "fake-nonexistent-target-xyz") {
		t.Fatalf("TestContributingReferencesRealTaskTargets_UnknownTargetIsError: expected 'fake-nonexistent-target-xyz' in missing targets, got %v", missing)
	}
}

// --- plan 02-01: verify:gatekeeper precondition-set guard ------------------

// gatekeeperPrecondition mirrors one Taskfile.yml preconditions: list entry
// (sh:/msg: pair).
type gatekeeperPrecondition struct {
	Sh  string `yaml:"sh"`
	Msg string `yaml:"msg"`
}

// gatekeeperTaskYAML mirrors just the preconditions: field of one Taskfile.yml
// task entry — the only field the guard below needs.
type gatekeeperTaskYAML struct {
	Preconditions []gatekeeperPrecondition `yaml:"preconditions"`
}

// gatekeeperTaskfileRoot mirrors just the tasks: map of Taskfile.yml, decoded
// with the REAL YAML decoder (go.yaml.in/yaml/v3) — never a line scanner.
// Taskfile.yml's preconditions: entries are an ordinary YAML list-of-maps;
// a real decoder handles that shape unambiguously, the same way
// parseGoreleaserCrossPairs above decodes .goreleaser.yaml's builds: entries
// rather than regex-scanning them.
type gatekeeperTaskfileRoot struct {
	Tasks map[string]gatekeeperTaskYAML `yaml:"tasks"`
}

// gatekeeperExpectedPreconditionShs is the exact set of sh: values
// verify:gatekeeper must declare: one per named input (TAG, REPO, GOOS,
// GOARCH, GH_TOKEN, GATEKEEPER_EXPECT presence, GATEKEEPER_EXPECT value
// validation), one per required tool (gh, jq, xattr, spctl), and the
// darwin-host check — 12 total. Compared as an exact set, both directions: a
// missing entry OR an unexpected extra entry both fail the guard.
var gatekeeperExpectedPreconditionShs = []string{
	`[ -n "${TAG:-}" ]`,
	`[ -n "${REPO:-}" ]`,
	`[ -n "${GOOS:-}" ]`,
	`[ -n "${GOARCH:-}" ]`,
	`[ -n "${GH_TOKEN:-}" ]`,
	`[ -n "${GATEKEEPER_EXPECT:-}" ]`,
	`[ "${GATEKEEPER_EXPECT:-}" = "accepted" ] || [ "${GATEKEEPER_EXPECT:-}" = "rejected" ]`,
	`[ "$(go env GOHOSTOS)" = "darwin" ]`,
	`command -v gh`,
	`command -v jq`,
	`command -v xattr`,
	`command -v spctl`,
}

// gatekeeperPreconditionNameChecks maps a subset of the sh: values above to
// the substring their msg: must contain — proving the precondition halts BY
// NAME (D-09's discipline) rather than with a generic message.
var gatekeeperPreconditionNameChecks = map[string]string{
	`[ -n "${TAG:-}" ]`:               "TAG",
	`[ -n "${REPO:-}" ]`:              "REPO",
	`[ -n "${GOOS:-}" ]`:              "GOOS",
	`[ -n "${GOARCH:-}" ]`:            "GOARCH",
	`[ -n "${GH_TOKEN:-}" ]`:          "GH_TOKEN",
	`[ -n "${GATEKEEPER_EXPECT:-}" ]`: "GATEKEEPER_EXPECT",
	`command -v gh`:                   "gh",
	`command -v jq`:                   "jq",
	`command -v xattr`:                "xattr",
	`command -v spctl`:                "spctl",
}

// parseGatekeeperPreconditions decodes Taskfile.yml source src with the real
// YAML decoder and returns verify:gatekeeper's preconditions: list. Returns
// a non-nil error when src fails to parse as YAML, when no verify:gatekeeper
// task exists under tasks:, or when that task declares zero preconditions:
// entries — never a usable empty slice on any of those misses (the CR-01
// defect class every parser in this file guards against).
func parseGatekeeperPreconditions(src string) ([]gatekeeperPrecondition, error) {
	var root gatekeeperTaskfileRoot
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		return nil, fmt.Errorf("parseGatekeeperPreconditions: %w", err)
	}
	task, ok := root.Tasks["verify:gatekeeper"]
	if !ok {
		return nil, fmt.Errorf("parseGatekeeperPreconditions: no verify:gatekeeper task found under tasks:")
	}
	if len(task.Preconditions) == 0 {
		return nil, fmt.Errorf("parseGatekeeperPreconditions: verify:gatekeeper declares zero preconditions:")
	}
	return task.Preconditions, nil
}

// TestVerifyGatekeeperDeclaresNamedPreconditions pins plan 02-01 Task 1's
// precondition contract for verify:gatekeeper: it hard-fails by name (D-09's
// discipline) on every missing input and missing tool, and GATEKEEPER_EXPECT
// additionally carries a value-validation precondition (review concern pi
// LOW) rather than silently accepting anything. Fails if the target is
// absent.
func TestVerifyGatekeeperDeclaresNamedPreconditions(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	preconditions, err := parseGatekeeperPreconditions(string(data))
	if err != nil {
		t.Fatalf("parseGatekeeperPreconditions: %v", err)
	}

	gotShs := make([]string, 0, len(preconditions))
	msgBySh := make(map[string]string, len(preconditions))
	for _, p := range preconditions {
		if strings.TrimSpace(p.Msg) == "" {
			t.Errorf("verify:gatekeeper precondition sh=%q has an empty msg: — every precondition must hard-fail by name, not by a bare non-zero exit", p.Sh)
		}
		gotShs = append(gotShs, p.Sh)
		msgBySh[p.Sh] = p.Msg
	}

	wantShs := append([]string(nil), gatekeeperExpectedPreconditionShs...)
	sort.Strings(gotShs)
	sort.Strings(wantShs)
	if !reflect.DeepEqual(gotShs, wantShs) {
		t.Fatalf("verify:gatekeeper preconditions: sh: values are not the expected exact set.\ngot:  %v\nwant: %v", gotShs, wantShs)
	}

	for sh, name := range gatekeeperPreconditionNameChecks {
		msg, ok := msgBySh[sh]
		if !ok {
			continue // already reported above as part of the exact-set mismatch
		}
		if !strings.Contains(msg, name) {
			t.Errorf("verify:gatekeeper precondition sh=%q has msg %q, which does not name %q — a precondition must halt BY NAME", sh, msg, name)
		}
	}

	expectValueSh := `[ "${GATEKEEPER_EXPECT:-}" = "accepted" ] || [ "${GATEKEEPER_EXPECT:-}" = "rejected" ]`
	if msg, ok := msgBySh[expectValueSh]; ok {
		if !strings.Contains(msg, "accepted") || !strings.Contains(msg, "rejected") {
			t.Errorf("verify:gatekeeper's GATEKEEPER_EXPECT value-validation precondition msg %q must name both allowed values (accepted, rejected)", msg)
		}
	}
}

// TestVerifyGatekeeperDeclaresNamedPreconditions_MissingTargetIsError is the
// non-vacuity companion: it feeds parseGatekeeperPreconditions synthetic
// Taskfile.yml source with no verify:gatekeeper task and verifies the parser
// — and therefore the main test above — would fail rather than pass
// vacuously if the target were ever removed or renamed.
func TestVerifyGatekeeperDeclaresNamedPreconditions_MissingTargetIsError(t *testing.T) {
	synthetic := `tasks:
  build:
    cmds:
      - go build ./...
`
	if _, err := parseGatekeeperPreconditions(synthetic); err == nil {
		t.Fatalf("parseGatekeeperPreconditions: expected an error for Taskfile.yml source with no verify:gatekeeper task, got nil")
	}
}

// --- plan 02-04: release:rehearse-notarize credential-precondition guard ---

// rehearseNotarizeExpectedCredentialVars is the exact set of Apple credential
// environment variables release:rehearse-notarize must declare a named
// precondition for — D-09's literal content: one precondition per
// credential, each hard-failing BY NAME before any network round-trip to
// Apple.
var rehearseNotarizeExpectedCredentialVars = []string{
	"MACOS_SIGN_P12",
	"MACOS_SIGN_PASSWORD",
	"MACOS_NOTARY_ISSUER_ID",
	"MACOS_NOTARY_KEY_ID",
	"MACOS_NOTARY_KEY",
}

// parseRehearseNotarizePreconditions decodes Taskfile.yml source src with the
// real YAML decoder (reusing the gatekeeperTaskfileRoot/gatekeeperTaskYAML/
// gatekeeperPrecondition structs plan 02-01 defined — their shape is generic
// to any task's preconditions: list, not gatekeeper-specific) and returns
// release:rehearse-notarize's preconditions: list. Mirrors
// parseGatekeeperPreconditions's non-vacuity discipline: a missing task or a
// task with zero preconditions is a returned error, never a usable empty
// slice.
func parseRehearseNotarizePreconditions(src string) ([]gatekeeperPrecondition, error) {
	var root gatekeeperTaskfileRoot
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		return nil, fmt.Errorf("parseRehearseNotarizePreconditions: %w", err)
	}
	task, ok := root.Tasks["release:rehearse-notarize"]
	if !ok {
		return nil, fmt.Errorf("parseRehearseNotarizePreconditions: no release:rehearse-notarize task found under tasks:")
	}
	if len(task.Preconditions) == 0 {
		return nil, fmt.Errorf("parseRehearseNotarizePreconditions: release:rehearse-notarize declares zero preconditions:")
	}
	return task.Preconditions, nil
}

// TestRehearseNotarizeDeclaresCredentialPreconditions pins plan 02-04 Task
// 1's credential-precondition contract for release:rehearse-notarize: one
// named precondition per Apple credential variable (D-09's literal content),
// each with a non-empty msg: that names the variable — halting BY NAME
// rather than after a network round-trip to Apple. Fails if the target is
// absent or if any one credential precondition is missing.
func TestRehearseNotarizeDeclaresCredentialPreconditions(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	preconditions, err := parseRehearseNotarizePreconditions(string(data))
	if err != nil {
		t.Fatalf("parseRehearseNotarizePreconditions: %v", err)
	}

	if len(preconditions) < len(rehearseNotarizeExpectedCredentialVars) {
		t.Fatalf("release:rehearse-notarize declares %d preconditions, want at least %d (one per Apple credential variable, plus host/tool/worktree guards)", len(preconditions), len(rehearseNotarizeExpectedCredentialVars))
	}

	joined := make([]string, 0, len(preconditions))
	for _, p := range preconditions {
		if strings.TrimSpace(p.Msg) == "" {
			t.Errorf("release:rehearse-notarize precondition sh=%q has an empty msg: — every precondition must hard-fail by name, not by a bare non-zero exit", p.Sh)
		}
		joined = append(joined, p.Sh+" :: "+p.Msg)
	}
	haystack := strings.Join(joined, "\n")

	for _, v := range rehearseNotarizeExpectedCredentialVars {
		if !strings.Contains(haystack, v) {
			t.Errorf("release:rehearse-notarize declares no precondition naming %q (sh: or msg:) — D-09 requires one named precondition per Apple credential variable", v)
		}
	}
}

// TestRehearseNotarizeDeclaresCredentialPreconditions_MissingTargetIsError is
// the non-vacuity companion: parseRehearseNotarizePreconditions must fail —
// not pass vacuously — against synthetic Taskfile.yml source with no
// release:rehearse-notarize task.
func TestRehearseNotarizeDeclaresCredentialPreconditions_MissingTargetIsError(t *testing.T) {
	synthetic := `tasks:
  build:
    cmds:
      - go build ./...
`
	if _, err := parseRehearseNotarizePreconditions(synthetic); err == nil {
		t.Fatalf("parseRehearseNotarizePreconditions: expected an error for Taskfile.yml source with no release:rehearse-notarize task, got nil")
	}
}

// --- plan 02-06 Task 3: verify:notarized-suite precondition-set guard -----

// notarizedSuiteExpectedPreconditionShs is the exact set of sh: values
// verify:notarized-suite must declare: one per named input (TAG, REPO,
// GOOS, GOARCH, GH_TOKEN), and one per required tool (gh, jq, cosign) — 8
// total. Compared as an exact set, both directions, mirroring
// gatekeeperExpectedPreconditionShs's own discipline.
var notarizedSuiteExpectedPreconditionShs = []string{
	`[ -n "${TAG:-}" ]`,
	`[ -n "${REPO:-}" ]`,
	`[ -n "${GOOS:-}" ]`,
	`[ -n "${GOARCH:-}" ]`,
	`[ -n "${GH_TOKEN:-}" ]`,
	`command -v gh`,
	`command -v jq`,
	`command -v cosign`,
}

// notarizedSuitePreconditionNameChecks maps each sh: value above to the
// substring its msg: must contain — proving the precondition halts BY NAME
// (D-09's discipline) rather than with a generic message.
var notarizedSuitePreconditionNameChecks = map[string]string{
	`[ -n "${TAG:-}" ]`:      "TAG",
	`[ -n "${REPO:-}" ]`:     "REPO",
	`[ -n "${GOOS:-}" ]`:     "GOOS",
	`[ -n "${GOARCH:-}" ]`:   "GOARCH",
	`[ -n "${GH_TOKEN:-}" ]`: "GH_TOKEN",
	`command -v gh`:          "gh",
	`command -v jq`:          "jq",
	`command -v cosign`:      "cosign",
}

// parseNotarizedSuitePreconditions decodes Taskfile.yml source src with the
// real YAML decoder (reusing gatekeeperTaskfileRoot/gatekeeperTaskYAML/
// gatekeeperPrecondition — generic to any task's preconditions: list) and
// returns verify:notarized-suite's preconditions: list. Returns a non-nil
// error when src fails to parse as YAML, when no verify:notarized-suite
// task exists under tasks:, or when that task declares zero preconditions:
// entries — never a usable empty slice on any of those misses.
func parseNotarizedSuitePreconditions(src string) ([]gatekeeperPrecondition, error) {
	var root gatekeeperTaskfileRoot
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		return nil, fmt.Errorf("parseNotarizedSuitePreconditions: %w", err)
	}
	task, ok := root.Tasks["verify:notarized-suite"]
	if !ok {
		return nil, fmt.Errorf("parseNotarizedSuitePreconditions: no verify:notarized-suite task found under tasks:")
	}
	if len(task.Preconditions) == 0 {
		return nil, fmt.Errorf("parseNotarizedSuitePreconditions: verify:notarized-suite declares zero preconditions:")
	}
	return task.Preconditions, nil
}

// TestVerifyNotarizedSuiteDeclaresNamedPreconditions pins plan 02-06 Task
// 3's precondition contract for verify:notarized-suite: it hard-fails by
// name (D-09's discipline) on every missing input and missing tool. Fails
// if the target is absent, or if the sh: set is not exactly
// notarizedSuiteExpectedPreconditionShs.
func TestVerifyNotarizedSuiteDeclaresNamedPreconditions(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	preconditions, err := parseNotarizedSuitePreconditions(string(data))
	if err != nil {
		t.Fatalf("parseNotarizedSuitePreconditions: %v", err)
	}

	gotShs := make([]string, 0, len(preconditions))
	msgBySh := make(map[string]string, len(preconditions))
	for _, p := range preconditions {
		if strings.TrimSpace(p.Msg) == "" {
			t.Errorf("verify:notarized-suite precondition sh=%q has an empty msg: — every precondition must hard-fail by name, not by a bare non-zero exit", p.Sh)
		}
		gotShs = append(gotShs, p.Sh)
		msgBySh[p.Sh] = p.Msg
	}

	wantShs := append([]string(nil), notarizedSuiteExpectedPreconditionShs...)
	sort.Strings(gotShs)
	sort.Strings(wantShs)
	if !reflect.DeepEqual(gotShs, wantShs) {
		t.Fatalf("verify:notarized-suite preconditions: sh: values are not the expected exact set.\ngot:  %v\nwant: %v", gotShs, wantShs)
	}

	for sh, name := range notarizedSuitePreconditionNameChecks {
		msg, ok := msgBySh[sh]
		if !ok {
			continue // already reported above as part of the exact-set mismatch
		}
		if !strings.Contains(msg, name) {
			t.Errorf("verify:notarized-suite precondition sh=%q has msg %q, which does not name %q — a precondition must halt BY NAME", sh, msg, name)
		}
	}
}

// TestVerifyNotarizedSuiteDeclaresNamedPreconditions_MissingTargetIsError is
// the non-vacuity companion: it feeds parseNotarizedSuitePreconditions
// synthetic Taskfile.yml source with no verify:notarized-suite task and
// verifies the parser — and therefore the main test above — would fail
// rather than pass vacuously if the target were ever removed or renamed.
func TestVerifyNotarizedSuiteDeclaresNamedPreconditions_MissingTargetIsError(t *testing.T) {
	synthetic := `tasks:
  build:
    cmds:
      - go build ./...
`
	if _, err := parseNotarizedSuitePreconditions(synthetic); err == nil {
		t.Fatalf("parseNotarizedSuitePreconditions: expected an error for Taskfile.yml source with no verify:notarized-suite task, got nil")
	}
}

// --- Task 2 (04-05): one release-identity policy, proved by boundary-case
// parity across every restatement ------------------------------------------

// cosignIdentityFiles is the FIVE files this project restates
// releaseWorkflowRefPattern's --certificate-identity-regexp literal in,
// published or executed. Missing either of the two files the cycle-1 review
// list overlooked (SECURITY.md, docs/RELEASE-PROCEDURES.md) would make this
// plan's own "every identity literal" must-have false again.
var cosignIdentityFiles = []string{
	taskfilePath,
	"../../README.md",
	"../../docs/RELEASE.md",
	"../../SECURITY.md",
	"../../docs/RELEASE-PROCEDURES.md",
}

// cosignIdentityLiteral is one --certificate-identity-regexp literal
// extracted from a restatement file, together with its source location —
// carrying File and Line is what makes the misattribution guard in
// TestCosignIdentityPolicyBoundaryParityWithCompiledPattern possible: a
// literal extracted twice from one file, or credited to the wrong file,
// would otherwise be indistinguishable from correct output.
type cosignIdentityLiteral struct {
	File  string
	Line  int // 1-based
	Value string
}

var cosignIdentitySingleQuotedRe = regexp.MustCompile(`'([^']*)'`)

// extractCosignIdentityLiterals scans contents for occurrences of the flag
// --certificate-identity-regexp and returns, for each occurrence, the
// single-quoted regexp literal found in its window, plus the count of
// occurrences for which no literal was found.
//
// Parse contract (review cycle 1, MEDIUM — stated explicitly so a future
// reformat of one of these files produces a named failure here rather than
// a silent one):
//   - The literal is a single-quoted string on a line AT OR AFTER a line
//     containing --certificate-identity-regexp, within a window of the next
//     3 lines (the flag's own line plus the following 3).
//   - Exactly one literal is taken per flag occurrence; the search stops at
//     the first single-quoted string found in the window.
//   - A flag occurrence with NO single-quoted string anywhere in its window
//     is NOT silently skipped: it increments the returned unmatched count,
//     so a reformat that breaks this parse is visible as a mismatch between
//     flag occurrences and literals returned, not as a quietly smaller
//     literal set.
//   - Anything else — a double-quoted literal, a literal on the flag line
//     itself, a heredoc — is out of contract; extending this parser to
//     cover such a shape must be a deliberate change, not inferred.
//
// Returns a nil slice (never panics) on unparseable or empty input — the
// caller decides whether zero is an error, which is what makes the vacuity
// self-test possible.
func extractCosignIdentityLiterals(path, contents string) ([]cosignIdentityLiteral, int) {
	if contents == "" {
		return nil, 0
	}
	lines := strings.Split(contents, "\n")
	var out []cosignIdentityLiteral
	unmatched := 0
	for i, line := range lines {
		if !strings.Contains(line, "--certificate-identity-regexp") {
			continue
		}
		found := false
		for j := i; j < len(lines) && j <= i+3; j++ {
			if m := cosignIdentitySingleQuotedRe.FindStringSubmatch(lines[j]); m != nil {
				out = append(out, cosignIdentityLiteral{File: path, Line: j + 1, Value: m[1]})
				found = true
				break
			}
		}
		if !found {
			unmatched++
		}
	}
	return out, unmatched
}

// cosignIdentitySANCorpus is the fixed boundary-case corpus every extracted
// literal is checked against. Seeded from the SANs verify_test.go and
// release_workflow_shape_test.go already use for the same compiled pattern
// (TestReleaseWorkflowRefPattern_AcceptsReleaseWorkflowTagRef,
// TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo,
// TestReleaseWorkflowFileMatchesPattern) — not a fresh, unrelated set.
var cosignIdentitySANCorpus = []struct {
	name   string
	san    string
	accept bool
}{
	{
		name:   "release.yml tag-push SAN",
		san:    "https://github.com/" + releaseRepoSlug + "/.github/workflows/release.yml@refs/tags/v1.2.3",
		accept: true,
	},
	{
		name:   "release.yaml tag-push SAN",
		san:    "https://github.com/" + releaseRepoSlug + "/.github/workflows/release.yaml@refs/tags/v1.2.3",
		accept: true,
	},
	{
		name:   "different workflow filename, same repo, tag ref",
		san:    "https://github.com/" + releaseRepoSlug + "/.github/workflows/release-please.yml@refs/tags/v1.2.3",
		accept: false,
	},
	{
		name:   "release.yml at a branch ref",
		san:    "https://github.com/" + releaseRepoSlug + "/.github/workflows/release.yml@refs/heads/main",
		accept: false,
	},
	{
		name:   "different repository owner, tag ref",
		san:    "https://github.com/some-other-org/some-other-repo/.github/workflows/release.yml@refs/tags/v1.2.3",
		accept: false,
	},
	{
		name:   "trailing space-plus-token after the tag",
		san:    "https://github.com/" + releaseRepoSlug + "/.github/workflows/release.yml@refs/tags/v1.2.3 extra-token",
		accept: false,
	},
}

// cosignIdentityLineForByteOffset converts a byte offset in contents to a
// 1-based line number — the same slicing technique Task 1's <verify>
// ordering assertion uses on Taskfile.yml, reused here so the region-scoped
// check below computes the identical region.
func cosignIdentityLineForByteOffset(contents string, offset int) int {
	return strings.Count(contents[:offset], "\n") + 1
}

// TestCosignIdentityPolicyBoundaryParityWithCompiledPattern is the T-04-16
// spoofing guard (04-05-PLAN.md). The name is deliberately NOT
// "MatchesCompiledPattern" (review cycle 1, MEDIUM): both
// releaseWorkflowRefPattern and its POSIX restatements are regular
// languages, and agreement on a finite hand-picked SAN corpus proves parity
// on those probed boundaries, not language equivalence.
func TestCosignIdentityPolicyBoundaryParityWithCompiledPattern(t *testing.T) {
	type found struct {
		lits      []cosignIdentityLiteral
		unmatched int
	}
	perFile := make(map[string]found, len(cosignIdentityFiles))
	var all []cosignIdentityLiteral
	for _, rel := range cosignIdentityFiles {
		data, err := os.ReadFile(rel)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		lits, unmatched := extractCosignIdentityLiterals(rel, string(data))
		perFile[rel] = found{lits: lits, unmatched: unmatched}
		all = append(all, lits...)
	}

	// Check 1: TOTAL floor. 7 is the post-Task-1 count, derived from the
	// measured pre-task baseline of 6 recorded in 04-05-PLAN.md's
	// <measured_baseline> — a floor at or below the pre-task count would be
	// satisfied before the task meant to raise it, exactly the cycle-1
	// pre-satisfied-floor defect this replaces. This check catches
	// WHOLESALE LOSS (literals deleted anywhere in the repository); it does
	// NOT catch relocation — check 3 below does, and neither check subsumes
	// the other (review cycle 2, HIGH).
	if len(all) < 7 {
		breakdown := make([]string, 0, len(cosignIdentityFiles))
		for _, rel := range cosignIdentityFiles {
			breakdown = append(breakdown, fmt.Sprintf("%s=%d", rel, len(perFile[rel].lits)))
		}
		t.Fatalf("extracted %d cosign identity literals across %v, want at least 7 (measured pre-task baseline: 6). Per-file breakdown: %v", len(all), cosignIdentityFiles, breakdown)
	}

	// Check 2: per-file set membership, one file at a time — a total floor
	// alone could be met by one file carrying everything.
	for _, rel := range cosignIdentityFiles {
		if len(perFile[rel].lits) == 0 {
			t.Errorf("file %s yielded zero cosign identity literals — every one of the five restatement files must carry at least one", rel)
		}
	}

	// Check 3: region-scoped requirement — at least one literal must come
	// from the verify:self-upgrade region of Taskfile.yml specifically, the
	// same region Task 1's ordering assertion slices. This is the
	// assertion that goes RED if Task 1's cosign step is dropped, MOVED OUT
	// of the target, or its flag is misspelled — the property the cycle-1
	// floor lacked. Its message names the region delimiters and is
	// unambiguously distinguishable from check 1's message when it fires.
	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	taskfileSrc := string(taskfileData)
	regionStart := strings.Index(taskfileSrc, "verify:self-upgrade:")
	regionEnd := strings.Index(taskfileSrc, "verify:gatekeeper:")
	if regionStart < 0 || regionEnd < 0 || regionEnd <= regionStart {
		t.Fatalf("could not locate the verify:self-upgrade: ... verify:gatekeeper: region in %s", taskfilePath)
	}
	startLine := cosignIdentityLineForByteOffset(taskfileSrc, regionStart)
	endLine := cosignIdentityLineForByteOffset(taskfileSrc, regionEnd)
	foundInRegion := false
	for _, lit := range perFile[taskfilePath].lits {
		if lit.Line >= startLine && lit.Line <= endLine {
			foundInRegion = true
			break
		}
	}
	if !foundInRegion {
		t.Fatalf("no cosign identity literal found inside the verify:self-upgrade region (lines %d-%d of %s) — Task 1's cosign step must restate the identity literal inside verify:self-upgrade itself", startLine, endLine, taskfilePath)
	}

	// Check 4: misattribution guard — every extracted literal's reported
	// line must actually contain the value it was credited with.
	for _, lit := range all {
		data, err := os.ReadFile(lit.File)
		if err != nil {
			t.Fatalf("read %s: %v", lit.File, err)
		}
		lines := strings.Split(string(data), "\n")
		if lit.Line-1 < 0 || lit.Line-1 >= len(lines) {
			t.Errorf("literal from %s:%d is out of range for that file's %d lines", lit.File, lit.Line, len(lines))
			continue
		}
		if !strings.Contains(lines[lit.Line-1], lit.Value) {
			t.Errorf("literal attributed to %s:%d does not appear on that line: line=%q value=%q", lit.File, lit.Line, lines[lit.Line-1], lit.Value)
		}
	}

	// Check 5: the corpus itself must not be degenerate.
	acceptCount, rejectCount := 0, 0
	for _, c := range cosignIdentitySANCorpus {
		if c.accept {
			acceptCount++
		} else {
			rejectCount++
		}
	}
	if acceptCount < 2 || rejectCount < 4 {
		t.Fatalf("cosignIdentitySANCorpus has %d accepted and %d rejected SANs, want at least 2 accepted and 4 rejected", acceptCount, rejectCount)
	}

	// Behavioural equality, not string equality, is deliberate: the Go
	// pattern spells the non-space class one way and the shell/POSIX
	// restatements spell it another, and both are correct. A literal
	// string comparison would fail on a legitimate difference and would
	// still miss a semantic drift written in the same spelling.
	compiled := regexp.MustCompile(releaseWorkflowRefPattern)
	for _, lit := range all {
		litRe, err := regexp.Compile(lit.Value)
		if err != nil {
			t.Errorf("literal from %s:%d does not compile as a Go regexp: %v (value=%q)", lit.File, lit.Line, err, lit.Value)
			continue
		}
		for _, c := range cosignIdentitySANCorpus {
			got := litRe.MatchString(c.san)
			want := compiled.MatchString(c.san)
			if got != want {
				t.Errorf("boundary-case parity mismatch: %s:%d literal %q vs releaseWorkflowRefPattern on SAN %q (%s): literal=%v compiled=%v", lit.File, lit.Line, lit.Value, c.san, c.name, got, want)
			}
		}
	}

	t.Logf("cosign identity literals: %d total across %d files (post-Task-1 floor: 7, pre-task baseline: 6)", len(all), len(cosignIdentityFiles))
	for _, rel := range cosignIdentityFiles {
		t.Logf("  %s: %d literal(s), %d unmatched flag occurrence(s)", rel, len(perFile[rel].lits), perFile[rel].unmatched)
	}
}

// TestCosignIdentityPolicyBoundaryParity_ZeroLiteralsIsError is the vacuity
// self-test, in the same shape as TestTaskfileGatesFailLoud_EmptyFileIsError:
// pins that the main test's t.Fatalf guard is reachable (empty input yields
// zero literals) and that the flag alone never manufactures a phantom
// literal (flag present, no quoted literal anywhere in its window).
func TestCosignIdentityPolicyBoundaryParity_ZeroLiteralsIsError(t *testing.T) {
	if lits, unmatched := extractCosignIdentityLiterals("empty.txt", ""); len(lits) != 0 || unmatched != 0 {
		t.Fatalf("extractCosignIdentityLiterals(empty) = (%v, %d), want (nil, 0)", lits, unmatched)
	}

	flagOnly := "  --certificate-identity-regexp \\\n  some text with no quotes\n  more text\n  still more\n"
	lits, unmatched := extractCosignIdentityLiterals("flag-only.txt", flagOnly)
	if len(lits) != 0 {
		t.Fatalf("extractCosignIdentityLiterals(flag with no quoted literal) returned %d literal(s), want 0 — the flag alone must never manufacture a phantom literal: %v", len(lits), lits)
	}
	if unmatched != 1 {
		t.Fatalf("extractCosignIdentityLiterals(flag with no quoted literal): unmatched = %d, want 1", unmatched)
	}
}

// --- quick 260811-s5o: cosign-installer coverage guard ---------------------
//
// Release v0.9.0, post-release-verify.yml run 31549287269: the self-upgrade
// job invoked `task verify:self-upgrade`, which declares a `command -v
// cosign` precondition (Taskfile.yml, added by phase 4 plan 04-05) — but the
// job never installed cosign, so the precondition failed closed exactly as
// designed ("task: cosign not found ... precondition not met"). The
// precondition was correct; the installer was missing. This guard derives,
// at runtime, every Taskfile target with a `command -v cosign` precondition
// and every job in post-release-verify.yml, then asserts each job that
// invokes such a target installs cosign via sigstore/cosign-installer
// exactly once, pinned to one uniform SHA+version comment across the whole
// file — so the next Taskfile target that grows a cosign precondition
// cannot silently ship the same defect.

// parseCosignRequiringTaskTargets decodes Taskfile source src with the real
// YAML decoder (reusing gatekeeperTaskfileRoot) and returns the set of task
// names whose preconditions: list contains an entry whose sh: equals
// "command -v cosign" after strings.TrimSpace. Returns a non-nil error when
// src fails to parse as YAML, when tasks: is empty, or when zero targets
// match — never a usable empty map on any of those misses (the CR-01 defect
// class every parser in this file guards against).
func parseCosignRequiringTaskTargets(src string) (map[string]bool, error) {
	var root gatekeeperTaskfileRoot
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		return nil, fmt.Errorf("parseCosignRequiringTaskTargets: %w", err)
	}
	if len(root.Tasks) == 0 {
		return nil, fmt.Errorf("parseCosignRequiringTaskTargets: no tasks: found in Taskfile source")
	}
	targets := make(map[string]bool)
	for name, task := range root.Tasks {
		for _, p := range task.Preconditions {
			if strings.TrimSpace(p.Sh) == "command -v cosign" {
				targets[name] = true
				break
			}
		}
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("parseCosignRequiringTaskTargets: zero task targets declare a 'command -v cosign' precondition")
	}
	return targets, nil
}

// parseWorkflowJobsAllSteps decodes workflow YAML source src with the real
// YAML decoder (reusing workflowFileYAML) and returns every job's full,
// ordered step list keyed by job id. Returns a non-nil error when src fails
// to parse as YAML or declares zero jobs: — never a usable empty map on
// either miss.
func parseWorkflowJobsAllSteps(src string) (map[string][]workflowRunStep, error) {
	var wf workflowFileYAML
	if err := yaml.Unmarshal([]byte(src), &wf); err != nil {
		return nil, fmt.Errorf("parseWorkflowJobsAllSteps: %w", err)
	}
	if len(wf.Jobs) == 0 {
		return nil, fmt.Errorf("parseWorkflowJobsAllSteps: no jobs: found in workflow source")
	}
	result := make(map[string][]workflowRunStep, len(wf.Jobs))
	for id, job := range wf.Jobs {
		result[id] = job.Steps
	}
	return result, nil
}

// cosignGuardTaskCallLineRe is a capturing variant of taskCallLineRe, used
// only by taskTargetsInRunBody below, to recover WHICH target a run: body
// invokes rather than merely confirming the `task <target>` shape.
var cosignGuardTaskCallLineRe = regexp.MustCompile(`^task\s+([A-Za-z0-9:_.-]+)$`)

// taskTargetsInRunBody runs stripRunBodyNoise first (so a `#`-prefixed
// comment line can never masquerade as an invocation — this file's existing
// house rule), then returns every task target invoked by a `task <target>`
// line in body. Pure helper, no error return.
func taskTargetsInRunBody(body string) []string {
	stripped := stripRunBodyNoise(body)
	if stripped == "" {
		return nil
	}
	var targets []string
	for _, line := range strings.Split(stripped, "\n") {
		if m := cosignGuardTaskCallLineRe.FindStringSubmatch(line); m != nil {
			targets = append(targets, m[1])
		}
	}
	return targets
}

// cosignInstallerPinRe matches a sigstore/cosign-installer `uses:` line
// pinned to a 40-hex commit SHA with a trailing `# vX.Y.Z` version comment,
// checked against RAW source rather than the YAML-decoded step (the YAML
// decoder strips the trailing comment, so pin-shape must be read from raw
// text — exactly as this package already does for
// actions/attest-build-provenance in release_workflow_shape_test.go). The
// (?m) flag is required for `$` to match end-of-LINE rather than only
// end-of-text, since this regex is matched with FindAllStringSubmatch
// against a multi-line file with more than one occurrence.
var cosignInstallerPinRe = regexp.MustCompile(`(?m)uses:\s*(sigstore/cosign-installer@[0-9a-f]{40})\s+#\s*(v\d+\.\d+\.\d+)\s*$`)

// TestCosignRequiringJobsInstallCosign is quick 260811-s5o's guard: it is
// the test that would have caught release v0.9.0 run 31549287269's
// self-upgrade failure ("task: cosign not found ... precondition not met")
// before it ever reached CI. It derives, at runtime — never from a
// hardcoded job or target list — every Taskfile target with a
// `command -v cosign` precondition and every job in
// post-release-verify.yml, then asserts an EXACT installer-step count of 1
// for every job that invokes a cosign-requiring target, plus pin uniformity
// across the whole file. A "no job is missing an installer" style
// negative-only assertion would pass vacuously the moment its anchor
// stopped matching (repo rule 84d1gfpywd) — this guard never states the
// property that way.
func TestCosignRequiringJobsInstallCosign(t *testing.T) {
	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	cosignTargets, err := parseCosignRequiringTaskTargets(string(taskfileData))
	if err != nil {
		t.Fatalf("parseCosignRequiringTaskTargets: %v", err)
	}

	workflowData, err := os.ReadFile(postReleaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", postReleaseWorkflowPath, err)
	}
	workflowSrc := string(workflowData)
	jobs, err := parseWorkflowJobsAllSteps(workflowSrc)
	if err != nil {
		t.Fatalf("parseWorkflowJobsAllSteps: %v", err)
	}

	jobIDs := make([]string, 0, len(jobs))
	for id := range jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)

	totalInstallerSteps := 0
	cosignRequiringJobCount := 0
	for _, jobID := range jobIDs {
		steps := jobs[jobID]

		requiringTargets := make(map[string]bool)
		for _, step := range steps {
			for _, target := range taskTargetsInRunBody(step.Run) {
				if cosignTargets[target] {
					requiringTargets[target] = true
				}
			}
		}

		installerCount := 0
		for _, step := range steps {
			if strings.HasPrefix(step.Uses, "sigstore/cosign-installer@") {
				installerCount++
			}
		}
		totalInstallerSteps += installerCount

		if len(requiringTargets) == 0 {
			continue
		}
		cosignRequiringJobCount++

		requiredList := make([]string, 0, len(requiringTargets))
		for target := range requiringTargets {
			requiredList = append(requiredList, target)
		}
		sort.Strings(requiredList)

		t.Logf("job %q invokes cosign-requiring target(s) %v, installer step count=%d", jobID, requiredList, installerCount)

		if installerCount != 1 {
			t.Errorf("job %q invokes cosign-requiring target(s) %v but declares %d sigstore/cosign-installer step(s) (want exactly 1) — a job whose run: body invokes a task target with a 'command -v cosign' precondition must install cosign exactly once", jobID, requiredList, installerCount)
		}
	}

	if cosignRequiringJobCount == 0 {
		t.Fatalf("derived cosign-requiring job set is empty — a broken run-body parse must go red, never pass silently")
	}

	pinMatches := cosignInstallerPinRe.FindAllStringSubmatch(workflowSrc, -1)
	if len(pinMatches) != totalInstallerSteps {
		t.Errorf("%s: found %d sigstore/cosign-installer pin(s) matching the SHA+version-comment shape via raw-source regex, but %d installer step(s) via YAML decode — an unpinned, branch-pinned, or comment-less installer step decodes as a step but matches no pin line", postReleaseWorkflowPath, len(pinMatches), totalInstallerSteps)
	}

	shas := make(map[string]bool)
	versions := make(map[string]bool)
	for _, m := range pinMatches {
		shas[m[1]] = true
		versions[m[2]] = true
	}
	if len(shas) != 1 {
		t.Errorf("%s: found %d distinct sigstore/cosign-installer SHA pin(s), want exactly 1: %v", postReleaseWorkflowPath, len(shas), shas)
	}
	if len(versions) != 1 {
		t.Errorf("%s: found %d distinct sigstore/cosign-installer version comment(s), want exactly 1: %v", postReleaseWorkflowPath, len(versions), versions)
	}
}

// TestCosignInstallerGuardParsersFailLoudly is the non-vacuity companion to
// TestCosignRequiringJobsInstallCosign, modelled on this file's other
// ..._FailLoudly / ..._MissingTargetIsError tests: every parser it exercises
// must return a non-nil error on a broken or unmatching input, never a
// usable empty zero value that would let the guard above pass silently.
func TestCosignInstallerGuardParsersFailLoudly(t *testing.T) {
	t.Run("parseCosignRequiringTaskTargets empty source", func(t *testing.T) {
		if _, err := parseCosignRequiringTaskTargets(""); err == nil {
			t.Fatalf("parseCosignRequiringTaskTargets(\"\"): expected a non-nil error, got nil")
		}
	})

	t.Run("parseCosignRequiringTaskTargets no cosign preconditions", func(t *testing.T) {
		synthetic := "tasks:\n  build:\n    preconditions:\n      - sh: '[ -n \"${TAG:-}\" ]'\n        msg: \"TAG not set\"\n"
		if _, err := parseCosignRequiringTaskTargets(synthetic); err == nil {
			t.Fatalf("parseCosignRequiringTaskTargets(no cosign precondition): expected a non-nil error, got nil")
		}
	})

	t.Run("parseWorkflowJobsAllSteps empty source", func(t *testing.T) {
		if _, err := parseWorkflowJobsAllSteps(""); err == nil {
			t.Fatalf("parseWorkflowJobsAllSteps(\"\"): expected a non-nil error, got nil")
		}
	})
}

// === BLD-07: release-path structural scanner (02-04 Tasks 1-2) ===========
//
// Proves the signed release path stays pure Go: a fixture-backed scanner
// over the transitive execution closure of the two ROOTS below finds zero
// JavaScript-toolchain invocations, and is itself proven able to find one
// in each reachable unit kind (TestReleasePathScanIsNonVacuous) while
// leaving this repository's own near-miss strings alone
// (TestReleasePathScanIgnoresNearMisses). See this plan's
// <recorded_limitations> for the four boundaries the model does not cover
// and which three are tripwired.

// releasePathScanRoots is the literal ROOTS fixture: the two documents the
// signed release path is defined by. 02-CONTEXT.md verified both clean on
// 2026-08-23 with a positive control (zero matches for node/npm/npx/pnpm,
// against 29/14 `go` hits in the same two files — proof the search itself
// works). The scanned SET is NOT these two paths alone —
// resolveReleasePathClosure derives the transitive closure from them: the
// local composite actions release.yml `uses:`, and the Taskfile.yml
// targets its `run:` bodies invoke via `task <target>`. A new EDGE from an
// existing root (a new `uses: ./…` or `task <target>` line added to
// release.yml) is picked up automatically — no fixture change required. A
// new ROOT (a second release workflow, a second GoReleaser config) DOES
// require adding it here explicitly.
var releasePathScanRoots = []string{goreleaserPath, releasePathWorkflowPath}

// forbiddenJSToolchainCommands are the four command names BLD-07's
// requirement text enumerates by name: the Node runtime, the npm CLI, the
// npm package runner, and the pnpm CLI. Matched by exact path-BASENAME
// equality (walkNodeForJSToolchain), never by raw substring — a substring
// search flags this repository's own `pnpm-lock.yaml`, `web/node_modules`
// and `protoc-gen-es` strings, all of which legitimately exist elsewhere in
// this tree (TestReleasePathScanIgnoresNearMisses proves the distinction).
var forbiddenJSToolchainCommands = []string{"node", "npm", "npx", "pnpm"}

// forbiddenJSToolchainActions are marketplace-action owner/repo prefixes
// that put a JavaScript runtime or package manager onto a runner WITHOUT
// naming any of forbiddenJSToolchainCommands — an action that installs a
// runtime is a JS-toolchain invocation even though a command-name scan
// alone would never see it. Compared against a `uses:` value's text before
// its first "@" version pin.
var forbiddenJSToolchainActions = []string{"actions/setup-node", "pnpm/action-setup"}

// unsupportedEdgeKeys names the two workflow job-level keys that introduce
// an execution context this scanner cannot see into: a job's own container
// image, or a service container's image. Neither names a command or a
// marketplace action a token scan can inspect — the image itself may ship
// a JS runtime. resolveReleasePathClosure refuses
// (errUnsupportedReachabilityEdge) the moment either key appears on a
// scanned workflow job, rather than scanning the rest of the document and
// reporting a clean zero over a job it cannot actually see into
// (recorded_limitations boundary 2).
var unsupportedEdgeKeys = []string{"container", "services"}

// executionBodyKeys names the three YAML keys whose scalar content is
// actually EXECUTED at release time — a workflow step's run:, a Taskfile
// task's cmds:, and a GoReleaser hooks: entry. The unmodelled-script
// tripwire (checkExecutionBodyWordsForScript) only walks content reached
// through one of these keys: a script path named under an unrelated key
// (e.g. .goreleaser.yaml's `main: ./cmd/codegraph`, a BUILD input, never
// executed as a shell command) is not an execution edge and must not trip
// the wire.
var executionBodyKeys = []string{"run", "cmds", "hooks"}

// errUnsupportedReachabilityEdge is the BLD-07 coverage-model tripwire
// (T-02-04-07): resolveReleasePathClosure returns it, wrapped with the
// offending unit and edge kind, when it meets an execution edge its model
// does not cover — a run:/cmds:/hooks: scalar invoking a repository-local
// executable script (recorded_limitations boundaries 1 and 4), or a
// scanned workflow job carrying a container:/services: key
// (recorded_limitations boundary 2) — rather than scanning past it and
// reporting a clean zero over a coverage model that just shrank. This is
// repo rule 84d1gfpywd applied to the scanner's OWN coverage model, not
// merely its output: a guard that meets something it cannot model must say
// so. The remedy when this fires is to widen resolveReleasePathClosure's
// model, sandbox the new content, or record an accepted risk in the threat
// register — never to add the offending edge to an allowlist and move on.
// Deliberately NOT tripwired: a JS runtime installed by a general-purpose
// command (`brew install node`, a piped `curl | sh` installer, or a
// marketplace action outside forbiddenJSToolchainActions) —
// recorded_limitations boundary 3, recognisable only by enumerating
// command spellings, so refusing on it would be a second curated denylist
// wearing a structural-guard costume, not a structural rule.
var errUnsupportedReachabilityEdge = errors.New("unsupported reachability edge")

// releasePathScanUnit is one resolved document in the release path's
// transitive execution closure, together with the provenance
// resolveReleasePathClosure reached it by — which root, and through which
// edge kind — so a finding or a resolver error can name the exact path
// from a root to the offending content.
type releasePathScanUnit struct {
	// UnitName is the human-readable identifier used in every finding and
	// error message: the on-disk path for a root or a local action, or
	// `Taskfile.yml task "<target>"` for a Taskfile-target unit (whose
	// Content is a sub-block of Taskfile.yml, not the whole file).
	UnitName string
	Root     string // which of releasePathScanRoots reached this unit
	EdgeKind string // "root" | "local-action" | "taskfile-target"
	Content  []byte // the actual bytes scanYAMLForJSToolchain parses
}

// releasePathClosureStats counts what resolveReleasePathClosure actually
// resolved — the positive assertion repo rule 84d1gfpywd requires. A
// resolver that silently resolves zero local actions or zero Taskfile
// targets must not read as "the closure is small and clean": release.yml
// demonstrably `uses: ./.github/actions/install-task` and runs `task
// release:goreleaser` / `task release:record-final-hashes` today
// (T-02-04-01), so a zero count means the resolver is broken, not that the
// closure is empty.
type releasePathClosureStats struct {
	Roots                int
	ResolvedLocalActions int
	ResolvedTaskTargets  int
}

// jsToolchainFinding is one JS-toolchain invocation scanYAMLForJSToolchain
// found: which unit, where in that unit's YAML structure, and the
// offending token — precise enough that a failure message identifies the
// exact line to fix without a human re-deriving it from a raw grep.
type jsToolchainFinding struct {
	Unit     string
	Location string // a YAML-path-shaped breadcrumb, e.g. "$.jobs.release.run"
	Token    string // the offending command basename or action owner/repo prefix
	Kind     string // "command" | "action"
}

func (f jsToolchainFinding) String() string {
	return fmt.Sprintf("%s at %s: forbidden %s %q", f.Unit, f.Location, f.Kind, f.Token)
}

// resolveLocalActionPath resolves a workflow uses: value beginning with
// "./" (a local composite action reference) to its action.yml or
// action.yaml file on disk, relative to this test package. Returns a
// non-nil error naming both candidate filenames tried when neither exists
// — a closure edge pointing at a missing local action is an error, never a
// silently-skipped unit.
func resolveLocalActionPath(usesValue string) (string, error) {
	dir := filepath.Join(releasePathRepoRoot, usesValue)
	var tried []string
	for _, name := range []string{"action.yml", "action.yaml"} {
		candidate := filepath.Join(dir, name)
		tried = append(tried, candidate)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("resolveLocalActionPath(%s): neither action.yml nor action.yaml found (tried %v)", usesValue, tried)
}

// extractTaskCallTargets reuses taskCallLineRe (this file's existing
// `task <target>` shape) to recover every task target a run: body invokes,
// after stripRunBodyNoise removes comments/blanks — the same two-step
// idiom checkStepInvokesTask already applies. Returns nil for a run: body
// that invokes no task at all (a `uses:`-only step, or a step whose run:
// body is not a bare `task <target>` line — re-validating D-01's
// single-definition property is TestWorkflowRunBodiesInvokeTask's job, not
// this resolver's).
func extractTaskCallTargets(runBody string) []string {
	stripped := stripRunBodyNoise(runBody)
	if stripped == "" {
		return nil
	}
	var targets []string
	for _, line := range strings.Split(stripped, "\n") {
		if !taskCallLineRe.MatchString(line) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 2 {
			targets = append(targets, fields[1])
		}
	}
	return targets
}

// releasePathGenericWorkflow decodes a workflow file's jobs: as raw
// key-presence maps rather than a typed struct — the only way to detect a
// container:/services: key's mere PRESENCE (its value's shape is
// irrelevant to this check; only whether the key exists at all).
type releasePathGenericWorkflow struct {
	Jobs map[string]map[string]interface{} `yaml:"jobs"`
}

// checkWorkflowJobsForUnsupportedContainerEdges parses src as a workflow
// document and returns errUnsupportedReachabilityEdge, wrapped with the
// job's ID and the offending key, the moment any job declares a key in
// unsupportedEdgeKeys (recorded_limitations boundary 2). unitName
// identifies the unit in the returned error.
func checkWorkflowJobsForUnsupportedContainerEdges(unitName string, src []byte) error {
	var generic releasePathGenericWorkflow
	if err := yaml.Unmarshal(src, &generic); err != nil {
		return fmt.Errorf("checkWorkflowJobsForUnsupportedContainerEdges(%s): %w", unitName, err)
	}
	jobIDs := make([]string, 0, len(generic.Jobs))
	for id := range generic.Jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)
	for _, id := range jobIDs {
		job := generic.Jobs[id]
		for _, key := range unsupportedEdgeKeys {
			if _, ok := job[key]; ok {
				return fmt.Errorf("%w: %s job %q carries a %q key — its image is opaque to a structural token scan (recorded_limitations boundary 2)", errUnsupportedReachabilityEdge, unitName, id, key)
			}
		}
	}
	return nil
}

// shellWordSplitRe splits a run:/cmds:/hooks: scalar's text into candidate
// command words on whitespace and the shell metacharacters that separate
// distinct commands within one line. Used by both the forbidden-command
// scan and the unmodelled-script tripwire.
var shellWordSplitRe = regexp.MustCompile("[\\s;&|()<>`$]+")

// tokenizeShellWords splits s into words on shellWordSplitRe, trimming
// surrounding quote characters from each word and dropping empty results.
func tokenizeShellWords(s string) []string {
	var words []string
	for _, w := range shellWordSplitRe.Split(s, -1) {
		w = strings.Trim(w, `"'`)
		if w != "" {
			words = append(words, w)
		}
	}
	return words
}

// basenameOf returns the path segment after the last "/" in word, or word
// itself if it names no path at all — the structural comparison unit for
// forbiddenJSToolchainCommands. A raw substring search would flag
// "pnpm-lock.yaml" and "web/node_modules"; comparing basenames does not.
func basenameOf(word string) string {
	if idx := strings.LastIndex(word, "/"); idx >= 0 {
		return word[idx+1:]
	}
	return word
}

// actionPrefix returns a `uses:` value's owner/repo prefix — everything
// before its first "@" version pin, or the whole value if it carries none.
func actionPrefix(uses string) string {
	if idx := strings.Index(uses, "@"); idx >= 0 {
		return uses[:idx]
	}
	return uses
}

// checkExecutionBodiesForUnsupportedEdges parses content as YAML and walks
// every run:/cmds:/hooks: scalar (executionBodyKeys) looking for a word
// that names a repository-local executable script — recorded_limitations
// boundaries 1 (a shelled-out script from a workflow/Taskfile run:/cmds:
// body) and 4 (a GoReleaser hook that is a script path rather than an
// inline command). A candidate word must BOTH look path-shaped (begin with
// "./" or "../") AND resolve to an existing, non-directory file relative
// to releasePathRepoRoot before this refuses — a bare word that merely
// LOOKS path-shaped (e.g. a template placeholder) must not produce a false
// refusal. unitName identifies the unit in the returned error.
func checkExecutionBodiesForUnsupportedEdges(unitName string, content []byte) error {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return fmt.Errorf("checkExecutionBodiesForUnsupportedEdges(%s): %w", unitName, err)
	}
	return walkForUnsupportedScriptEdge(unitName, &doc)
}

func walkForUnsupportedScriptEdge(unitName string, node *yaml.Node) error {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range node.Content {
			if err := walkForUnsupportedScriptEdge(unitName, c); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, val := node.Content[i], node.Content[i+1]
			if contains(executionBodyKeys, key.Value) {
				if err := checkExecutionBodyWordsForScript(unitName, val); err != nil {
					return err
				}
			}
			if err := walkForUnsupportedScriptEdge(unitName, val); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkExecutionBodyWordsForScript(unitName string, node *yaml.Node) error {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		for _, word := range tokenizeShellWords(node.Value) {
			if !strings.HasPrefix(word, "./") && !strings.HasPrefix(word, "../") {
				continue
			}
			candidate := filepath.Join(releasePathRepoRoot, word)
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return fmt.Errorf("%w: %s's execution body invokes repository-local script %q (recorded_limitations boundary 1/4)", errUnsupportedReachabilityEdge, unitName, word)
			}
		}
	case yaml.SequenceNode, yaml.MappingNode:
		for _, c := range node.Content {
			if err := checkExecutionBodyWordsForScript(unitName, c); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveReleasePathClosure walks the release path's two ROOTS
// (releasePathScanRoots) and returns every unit reachable from them
// through the two edge kinds the release path actually executes: a
// workflow step's `uses: ./…` local composite action, and a
// `run:`/`cmds:`/`hooks:` scalar's `task <target>` invocation into
// Taskfile.yml. The closure is DERIVED from release.yml's own content —
// never hardcoded — so a new edge added to release.yml widens the scan
// automatically; only a new ROOT requires a fixture change
// (releasePathScanRoots).
//
// Four boundaries remain outside this model, recorded in this plan's
// <recorded_limitations> and in T-02-04-05 (accepted risk):
//  1. A repository-local executable script a scanned scalar shells out to
//     (e.g. `run: ./scripts/foo.sh`) — the resolver sees the invocation
//     but does not open the script. TRIPWIRED via
//     checkExecutionBodiesForUnsupportedEdges.
//  2. A container image (`container:`/`services:`) that ships a JS
//     runtime baked in — opaque to a token scan. TRIPWIRED via
//     checkWorkflowJobsForUnsupportedContainerEdges.
//  3. A JS runtime installed by a general-purpose command (`brew install
//     node`, a piped `curl | sh` installer, or a marketplace action
//     outside forbiddenJSToolchainActions). NOT tripwired — recognisable
//     only by enumerating command spellings, which would make the
//     tripwire a second curated denylist rather than a structural rule.
//  4. A GoReleaser hook that is a script path rather than an inline
//     command — same shape as (1). TRIPWIRED.
//
// Every one of these is a deliberate boundary, stated here rather than
// papered over: widening to (1), (2) or (4) means executing or sandboxing
// untrusted content at test time, a larger change than BLD-07's
// requirement text asks for; widening to (3) means abandoning the
// structural property entirely. When (1), (2) or (4) fires, the remedy is
// to widen this model, sandbox the new content, or record an accepted
// risk — never to allowlist the edge and move on unchanged.
func resolveReleasePathClosure() ([]releasePathScanUnit, releasePathClosureStats, error) {
	var units []releasePathScanUnit
	var stats releasePathClosureStats

	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		return nil, stats, fmt.Errorf("resolveReleasePathClosure: read %s: %w", taskfilePath, err)
	}
	taskBlocks, err := parseTaskBlocks(string(taskfileData))
	if err != nil {
		return nil, stats, fmt.Errorf("resolveReleasePathClosure: %s: %w", taskfilePath, err)
	}

	seenActions := make(map[string]bool)
	seenTargets := make(map[string]bool)

	for _, root := range releasePathScanRoots {
		data, err := os.ReadFile(root)
		if err != nil {
			return nil, stats, fmt.Errorf("resolveReleasePathClosure: read root %s: %w", root, err)
		}
		stats.Roots++
		units = append(units, releasePathScanUnit{UnitName: root, Root: root, EdgeKind: "root", Content: data})

		if err := checkExecutionBodiesForUnsupportedEdges(root, data); err != nil {
			return nil, stats, err
		}

		if root != releasePathWorkflowPath {
			// .goreleaser.yaml has no jobs:/steps:/uses: shape to derive
			// further edges from — it IS the whole document, already added
			// above.
			continue
		}

		if err := checkWorkflowJobsForUnsupportedContainerEdges(root, data); err != nil {
			return nil, stats, err
		}

		jobsSteps, err := parseWorkflowJobsAllSteps(string(data))
		if err != nil {
			return nil, stats, fmt.Errorf("resolveReleasePathClosure: %s: %w", root, err)
		}
		jobIDs := make([]string, 0, len(jobsSteps))
		for id := range jobsSteps {
			jobIDs = append(jobIDs, id)
		}
		sort.Strings(jobIDs)

		for _, jobID := range jobIDs {
			for _, step := range jobsSteps[jobID] {
				if strings.HasPrefix(step.Uses, "./") {
					actionPath, resolveErr := resolveLocalActionPath(step.Uses)
					if resolveErr != nil {
						return nil, stats, fmt.Errorf("resolveReleasePathClosure: %s job %q step %q: %w", root, jobID, step.Name, resolveErr)
					}
					if !seenActions[actionPath] {
						seenActions[actionPath] = true
						actionData, readErr := os.ReadFile(actionPath)
						if readErr != nil {
							return nil, stats, fmt.Errorf("resolveReleasePathClosure: local action %s (uses: %s in %s job %q): %w", actionPath, step.Uses, root, jobID, readErr)
						}
						stats.ResolvedLocalActions++
						units = append(units, releasePathScanUnit{UnitName: actionPath, Root: root, EdgeKind: "local-action", Content: actionData})
						if err := checkExecutionBodiesForUnsupportedEdges(actionPath, actionData); err != nil {
							return nil, stats, err
						}
					}
				}

				for _, target := range extractTaskCallTargets(step.Run) {
					if seenTargets[target] {
						continue
					}
					seenTargets[target] = true
					block, ok := taskBlocks[target]
					if !ok {
						return nil, stats, fmt.Errorf("resolveReleasePathClosure: %s job %q step %q invokes task %q, which does not exist in %s", root, jobID, step.Name, target, taskfilePath)
					}
					unitName := fmt.Sprintf("%s task %q", taskfilePath, target)
					stats.ResolvedTaskTargets++
					units = append(units, releasePathScanUnit{UnitName: unitName, Root: root, EdgeKind: "taskfile-target", Content: []byte(block)})
					if err := checkExecutionBodiesForUnsupportedEdges(unitName, []byte(block)); err != nil {
						return nil, stats, err
					}
				}
			}
		}
	}

	return units, stats, nil
}

// scanYAMLForJSToolchain is the content-taking core of the BLD-07
// structural scanner. It parses content as a YAML document, walks every
// SCALAR NODE in the tree (comments are not part of a parsed node's value
// and are never examined — a comment mentioning "npm" is not an
// invocation), and for each one:
//   - tokenizes the scalar's text on whitespace/shell metacharacters and
//     compares each word's BASENAME against forbiddenJSToolchainCommands —
//     never a raw substring (TestReleasePathScanIgnoresNearMisses proves
//     the distinction matters: a substring search flags pnpm-lock.yaml,
//     web/node_modules, and protoc-gen-es, all real strings elsewhere in
//     this repository);
//   - when the scalar is a mapping value under the key "uses", ALSO
//     compares its owner/repo prefix against forbiddenJSToolchainActions —
//     an action that installs a runtime is a JS-toolchain invocation even
//     though it names none of the four forbidden commands.
//
// Returns a non-nil error when content fails to parse as YAML, or when
// zero scalar nodes were examined — an empty parse must fail loudly, never
// silently pass (the CR-01 defect class every parser in this file guards
// against).
func scanYAMLForJSToolchain(unitName string, content []byte) ([]jsToolchainFinding, int, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, 0, fmt.Errorf("scanYAMLForJSToolchain(%s): %w", unitName, err)
	}
	findings, examined := walkNodeForJSToolchain(unitName, &doc, "$")
	if examined == 0 {
		return nil, 0, fmt.Errorf("scanYAMLForJSToolchain(%s): parsed as YAML but examined zero scalar nodes — an empty document must fail loudly, not pass", unitName)
	}
	return findings, examined, nil
}

// scanForJSToolchain is the thin path-reading wrapper around
// scanYAMLForJSToolchain — the path itself doubles as the unit name.
// Returns a non-nil error for a path that does not exist, never a nil
// error with an empty finding slice.
func scanForJSToolchain(path string) ([]jsToolchainFinding, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("scanForJSToolchain: read %s: %w", path, err)
	}
	return scanYAMLForJSToolchain(path, data)
}

func walkNodeForJSToolchain(unitName string, node *yaml.Node, path string) ([]jsToolchainFinding, int) {
	if node == nil {
		return nil, 0
	}
	var findings []jsToolchainFinding
	examined := 0
	switch node.Kind {
	case yaml.DocumentNode:
		for _, c := range node.Content {
			f, e := walkNodeForJSToolchain(unitName, c, path)
			findings = append(findings, f...)
			examined += e
		}
	case yaml.SequenceNode:
		for i, c := range node.Content {
			f, e := walkNodeForJSToolchain(unitName, c, fmt.Sprintf("%s[%d]", path, i))
			findings = append(findings, f...)
			examined += e
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, val := node.Content[i], node.Content[i+1]
			childPath := fmt.Sprintf("%s.%s", path, key.Value)
			if key.Value == "uses" && val.Kind == yaml.ScalarNode {
				if prefix := actionPrefix(val.Value); contains(forbiddenJSToolchainActions, prefix) {
					findings = append(findings, jsToolchainFinding{Unit: unitName, Location: childPath, Token: prefix, Kind: "action"})
				}
			}
			f, e := walkNodeForJSToolchain(unitName, val, childPath)
			findings = append(findings, f...)
			examined += e
		}
	case yaml.ScalarNode:
		examined++
		for _, word := range tokenizeShellWords(node.Value) {
			base := basenameOf(word)
			if contains(forbiddenJSToolchainCommands, base) {
				findings = append(findings, jsToolchainFinding{Unit: unitName, Location: path, Token: base, Kind: "command"})
			}
		}
	}
	return findings, examined
}

// releasePathUnitNames extracts UnitName from every unit — used only by
// test assertions that need to search the closure by name.
func releasePathUnitNames(units []releasePathScanUnit) []string {
	names := make([]string, len(units))
	for i, u := range units {
		names[i] = u.UnitName
	}
	return names
}

// TestReleasePathClosureIsTransitive is the review-HIGH closure-derivation
// guard: resolving the closure from the two roots must name, BY NAME, the
// three execution edges release.yml demonstrably contains today —
// `.github/actions/install-task/action.yml`, `release:goreleaser`, and
// `release:record-final-hashes` — and the resolved-local-action and
// resolved-Taskfile-target counts must each clear a floor derived from
// what release.yml actually contains, not a guess. A closure whose
// local-action or Taskfile-target count is zero is a hard failure: those
// edges exist in the file, so zero means the resolver broke.
func TestReleasePathClosureIsTransitive(t *testing.T) {
	units, stats, err := resolveReleasePathClosure()
	if err != nil {
		t.Fatalf("resolveReleasePathClosure: %v", err)
	}

	if stats.Roots != 2 {
		t.Fatalf("resolveReleasePathClosure: stats.Roots = %d, want exactly 2 (%v)", stats.Roots, releasePathScanRoots)
	}
	if stats.ResolvedLocalActions < 1 {
		t.Fatalf("resolveReleasePathClosure: stats.ResolvedLocalActions = %d, want >= 1 — %s demonstrably `uses: ./.github/actions/install-task` today; zero means the resolver broke, not that the closure is small", stats.ResolvedLocalActions, releasePathWorkflowPath)
	}
	if stats.ResolvedTaskTargets < 2 {
		t.Fatalf("resolveReleasePathClosure: stats.ResolvedTaskTargets = %d, want >= 2 — %s demonstrably runs `task release:goreleaser` and `task release:record-final-hashes` today", stats.ResolvedTaskTargets, releasePathWorkflowPath)
	}

	names := releasePathUnitNames(units)
	wantGoreleaserTarget := fmt.Sprintf("%s task %q", taskfilePath, "release:goreleaser")
	wantRecordHashesTarget := fmt.Sprintf("%s task %q", taskfilePath, "release:record-final-hashes")

	var haveInstallTask, haveGoreleaserTarget, haveRecordHashesTarget bool
	for _, n := range names {
		if strings.HasSuffix(n, "/.github/actions/install-task/action.yml") {
			haveInstallTask = true
		}
		if n == wantGoreleaserTarget {
			haveGoreleaserTarget = true
		}
		if n == wantRecordHashesTarget {
			haveRecordHashesTarget = true
		}
	}
	if !haveInstallTask {
		t.Errorf("resolveReleasePathClosure: closure does not name .github/actions/install-task/action.yml by unit name — units: %v", names)
	}
	if !haveGoreleaserTarget {
		t.Errorf("resolveReleasePathClosure: closure does not name unit %q — units: %v", wantGoreleaserTarget, names)
	}
	if !haveRecordHashesTarget {
		t.Errorf("resolveReleasePathClosure: closure does not name unit %q — units: %v", wantRecordHashesTarget, names)
	}
}

// TestReleasePathHasNoJSToolchain scans every unit resolveReleasePathClosure
// returns and asserts zero JS-toolchain findings, that each unit
// contributed a non-zero examined-scalar count, and that the closure's
// TOTAL examined-scalar count strictly exceeds what the two roots alone
// contribute — the assertion that the transitive half is really being
// scanned rather than resolved and discarded. This zero-findings result
// means nothing on its own; TestReleasePathScanIsNonVacuous is its
// required pair.
func TestReleasePathHasNoJSToolchain(t *testing.T) {
	units, stats, err := resolveReleasePathClosure()
	if err != nil {
		t.Fatalf("resolveReleasePathClosure: %v", err)
	}
	if stats.Roots == 0 {
		t.Fatalf("resolveReleasePathClosure: zero roots resolved")
	}

	var allFindings []jsToolchainFinding
	totalExamined := 0
	rootExamined := 0
	for _, u := range units {
		findings, examined, scanErr := scanYAMLForJSToolchain(u.UnitName, u.Content)
		if scanErr != nil {
			t.Fatalf("scanYAMLForJSToolchain(%s): %v", u.UnitName, scanErr)
		}
		if examined == 0 {
			t.Errorf("scanYAMLForJSToolchain(%s): examined zero scalar nodes — this unit contributed nothing to the scan", u.UnitName)
		}
		allFindings = append(allFindings, findings...)
		totalExamined += examined
		if u.EdgeKind == "root" {
			rootExamined += examined
		}
	}

	if len(allFindings) > 0 {
		t.Fatalf("release path contains %d JS-toolchain invocation(s) — a genuine BLD-07 violation, not this guard's fault to fix: %v", len(allFindings), allFindings)
	}

	if totalExamined <= rootExamined {
		t.Fatalf("closure total examined-scalar count (%d) does not exceed the two roots' own total (%d) — the transitive units contributed nothing to the scan", totalExamined, rootExamined)
	}
}

// TestReleasePathMissingFileIsError is the CR-01 edge case: a path that
// does not exist on disk must produce a non-nil error, never a nil error
// with an empty finding slice — the same defect class every parser in
// this file is built to avoid. A closure edge pointing at a local action
// that does not exist on disk is also an error, not a silently-skipped
// unit.
func TestReleasePathMissingFileIsError(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "does-not-exist.yml")
	if _, _, err := scanForJSToolchain(missingPath); err == nil {
		t.Fatalf("scanForJSToolchain(%s): expected a non-nil error for a missing file, got nil", missingPath)
	}

	if _, err := resolveLocalActionPath("./this/local/action/does/not/exist"); err == nil {
		t.Fatalf("resolveLocalActionPath: expected a non-nil error for a closure edge pointing at a nonexistent local action, got nil")
	}
}

// TestUnsupportedReachabilityEdgeIsLoud is the T-02-04-07 tripwire proof:
// table-driven over synthetic in-memory documents, one row per unmodelled
// edge kind — a workflow run: scalar invoking a repository-local
// executable script, a Taskfile cmds: entry doing the same, a GoReleaser
// hook that is a script path, a job carrying container:, and a job
// carrying services:. Each row exercises the SAME two functions
// resolveReleasePathClosure itself calls
// (checkExecutionBodiesForUnsupportedEdges,
// checkWorkflowJobsForUnsupportedContainerEdges) — not a re-implementation
// — so the synthetic and real-repository code paths are provably
// identical. A sixth, negative-control row runs the real
// resolveReleasePathClosure() against this repository's actual content and
// asserts it returns no such error — proving the tripwire is not merely
// eager. Without that row, the other five rows would be equally consistent
// with a tripwire that refuses on everything.
//
// Deliberately does NOT exercise boundary 3 (a general-purpose installer)
// — see errUnsupportedReachabilityEdge's own doc comment for why that
// boundary is not tripwired at all.
func TestUnsupportedReachabilityEdgeIsLoud(t *testing.T) {
	cases := []struct {
		name    string
		unit    string
		src     string
		checkFn func(unitName string, content []byte) error
	}{
		{
			name:    "workflow run: invokes a repository-local script",
			unit:    "synthetic-workflow-run-script.yml",
			src:     "jobs:\n  x:\n    steps:\n      - run: ./Taskfile.yml\n",
			checkFn: checkExecutionBodiesForUnsupportedEdges,
		},
		{
			name:    "Taskfile cmds: invokes a repository-local script",
			unit:    "synthetic-taskfile-cmds-script",
			src:     "cmds:\n  - ./Taskfile.yml\n",
			checkFn: checkExecutionBodiesForUnsupportedEdges,
		},
		{
			name:    "GoReleaser hook is a script path",
			unit:    "synthetic-goreleaser-hook-script.yaml",
			src:     "before:\n  hooks:\n    - ./Taskfile.yml\n",
			checkFn: checkExecutionBodiesForUnsupportedEdges,
		},
		{
			name:    "job carries container:",
			unit:    "synthetic-workflow-container.yml",
			src:     "jobs:\n  x:\n    container: node:20\n    steps:\n      - run: echo hi\n",
			checkFn: checkWorkflowJobsForUnsupportedContainerEdges,
		},
		{
			name:    "job carries services:",
			unit:    "synthetic-workflow-services.yml",
			src:     "jobs:\n  x:\n    services:\n      redis:\n        image: redis\n    steps:\n      - run: echo hi\n",
			checkFn: checkWorkflowJobsForUnsupportedContainerEdges,
		},
	}

	for _, c := range cases {
		err := c.checkFn(c.unit, []byte(c.src))
		if err == nil {
			t.Errorf("%s: expected a non-nil errUnsupportedReachabilityEdge, got nil", c.name)
			continue
		}
		if !errors.Is(err, errUnsupportedReachabilityEdge) {
			t.Errorf("%s: error %v does not wrap errUnsupportedReachabilityEdge", c.name, err)
		}
		if !strings.Contains(err.Error(), c.unit) {
			t.Errorf("%s: error %v does not name the unit %q", c.name, err, c.unit)
		}
	}

	if _, _, err := resolveReleasePathClosure(); err != nil {
		t.Errorf("resolveReleasePathClosure: negative control failed — real repository content unexpectedly tripped the unmodelled-edge wire: %v (either a genuine new edge was introduced and the model must widen, or the tripwire itself is over-eager)", err)
	}
}

// TestReleasePathScanIsNonVacuous is the non-vacuity companion required
// before TestReleasePathHasNoJSToolchain's zero-findings result means
// anything (Taskfile.yml's vuln:selftest is this repository's canonical
// model: assert an exact result against a PLANTED case, never trust a
// clean run alone — "a transcript grep is a claim about the grep, not
// about the product", STATE.md). It plants each of the four forbidden
// command names in each of four reachable unit shapes — a workflow run:
// scalar, a GoReleaser hooks: list entry, a local composite action's run:
// scalar, and a Taskfile cmds: entry — the last two proving the
// TRANSITIVE half of the closure is genuinely scanned: a planted token in
// a Taskfile cmds: body or a local action's run: scalar is exactly the
// shape a two-file model could not see. It also plants both forbidden
// action prefixes in a version-pinned uses: position. Eighteen planted
// forms total, each asserted to produce exactly one finding.
func TestReleasePathScanIsNonVacuous(t *testing.T) {
	type row struct {
		name string
		src  string
	}
	var rows []row

	const workflowRunTemplate = "jobs:\n  x:\n    steps:\n      - name: bad\n        run: %s script.js\n"
	const goreleaserHookTemplate = "before:\n  hooks:\n    - %s script.js\n"
	const localActionRunTemplate = "runs:\n  using: composite\n  steps:\n    - shell: bash\n      run: %s script.js\n"
	const taskfileCmdsTemplate = "cmds:\n  - %s script.js\n"

	for _, cmd := range forbiddenJSToolchainCommands {
		rows = append(rows,
			row{name: fmt.Sprintf("workflow run: %s", cmd), src: fmt.Sprintf(workflowRunTemplate, cmd)},
			row{name: fmt.Sprintf("goreleaser hooks: %s", cmd), src: fmt.Sprintf(goreleaserHookTemplate, cmd)},
			row{name: fmt.Sprintf("local action run: %s", cmd), src: fmt.Sprintf(localActionRunTemplate, cmd)},
			row{name: fmt.Sprintf("taskfile cmds: %s", cmd), src: fmt.Sprintf(taskfileCmdsTemplate, cmd)},
		)
	}
	for _, action := range forbiddenJSToolchainActions {
		rows = append(rows, row{
			name: fmt.Sprintf("uses: %s pinned", action),
			src:  fmt.Sprintf("jobs:\n  x:\n    steps:\n      - uses: %s@v4\n", action),
		})
	}

	if len(rows) < 18 {
		t.Fatalf("TestReleasePathScanIsNonVacuous: only %d planted rows, want at least 18", len(rows))
	}

	for _, r := range rows {
		findings, examined, err := scanYAMLForJSToolchain(r.name, []byte(r.src))
		if err != nil {
			t.Errorf("scanYAMLForJSToolchain(%q): %v", r.name, err)
			continue
		}
		if examined == 0 {
			t.Errorf("scanYAMLForJSToolchain(%q): examined zero scalar nodes", r.name)
			continue
		}
		if len(findings) != 1 {
			t.Errorf("scanYAMLForJSToolchain(%q): got %d finding(s), want exactly 1: %v", r.name, len(findings), findings)
			continue
		}
		// Capture one row's failure-message shape for the SUMMARY: the
		// finding names the unit, the YAML location, and the token.
		if r.name == "workflow run: node" {
			t.Logf("planted-token finding (SUMMARY evidence): %s", findings[0].String())
		}
	}
}

// TestReleasePathScanIgnoresNearMisses proves scanYAMLForJSToolchain
// degrades gracefully rather than into a substring search: each row below
// is a REAL string that exists in this repository today (cited to its
// source, not invented) and MUST NOT be flagged. A row that fails here
// means the scanner would have to be weakened until it matched nothing —
// exactly the failure mode this test prevents.
func TestReleasePathScanIgnoresNearMisses(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		source string
	}{
		{
			name:   "pnpm lockfile filename",
			src:    "jobs:\n  x:\n    steps:\n      - run: cat web/pnpm-lock.yaml\n",
			source: "web/pnpm-lock.yaml, created by 02-01 (wave 1)",
		},
		{
			name:   "dependencies directory path",
			src:    "jobs:\n  x:\n    steps:\n      - run: du -sh web/node_modules\n",
			source: "web/node_modules, created by 02-01 (wave 1)",
		},
		{
			name:   "Go setup action owner/repo",
			src:    "jobs:\n  x:\n    steps:\n      - uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16\n",
			source: ".github/workflows/ci.yml's Go setup step, present on main today",
		},
		{
			name:   "cache action owner/repo",
			src:    "jobs:\n  x:\n    steps:\n      - uses: namespacelabs/nscloud-cache-action@c5f8dab7560444c4bf8dbc64f1b203431873c547\n",
			source: ".github/workflows/ci.yml's cache action, present on main today",
		},
		{
			name: "generated-plugin binary name (protoc-gen-es)",
			src:  "cmds:\n  - web/node_modules/.bin/protoc-gen-es --arg\n",
			source: "02-CONTEXT.md D-05: 02-03 lands this literal into Taskfile.yml/web/package.json " +
				"in this SAME wave; re-confirm against web/package.json once 02-03 has landed. Not read " +
				"from that file directly here — this plan must not depend on a sibling in its own wave.",
		},
	}

	for _, c := range cases {
		findings, examined, err := scanYAMLForJSToolchain(c.name, []byte(c.src))
		if err != nil {
			t.Errorf("scanYAMLForJSToolchain(%q): %v", c.name, err)
			continue
		}
		if examined == 0 {
			t.Errorf("scanYAMLForJSToolchain(%q): examined zero scalar nodes", c.name)
			continue
		}
		if len(findings) != 0 {
			t.Errorf("scanYAMLForJSToolchain(%q): got %d false-positive finding(s) for a real, legitimate string (%s): %v — the scanner degenerated into a substring search", c.name, len(findings), c.source, findings)
		}
	}
}

// === BLD-01: no mutable JS cache on ci.yml's install path (02-04 Task 3) ==
//
// D-13's fold-in of the JS gates into ci.yml's existing `test` job relies
// on the JS install being a pure function of web/pnpm-lock.yaml — a stale
// mutable cache producing a node_modules that differs from what the
// lockfile describes is exactly the divergence BLD-01/BLD-03 forbid. This
// makes that a structural invariant instead of an omission nothing
// enforces, authored one wave BEFORE 02-06 adds the Node setup step, so it
// cannot arrive with a cache input already attached.

// forbiddenCacheActions are marketplace-action owner/repo prefixes that
// restore a MUTABLE cache unconditionally, regardless of any input scoping
// — a general-purpose cache action whose actual scope is whatever its
// path: input says, which cannot be reasoned about structurally from the
// action name alone. This is a SCOPED invariant, not a blanket cache ban:
// ci.yml's test job legitimately caches the Go module/build cache today
// (namespacelabs/nscloud-cache-action with cache: go), which is explicitly
// ALLOWED — see scanContentForMutableCache.
var forbiddenCacheActions = []string{"actions/cache", "actions/cache/restore", "actions/cache/save"}

// cacheInputKeys are the with: input keys that turn a Node setup action
// into a cache restorer. A cache: input on actions/setup-node is a
// FINDING regardless of its value; the identical key on actions/setup-go
// (cache: false in ci.yml today) is a different action prefix entirely and
// is not matched by this fixture's use in the Node-setup case.
var cacheInputKeys = []string{"cache"}

// nodeSetupActionPrefix and nscloudCacheActionPrefix are the two `uses:`
// owner/repo prefixes scanContentForMutableCache classifies by NAME rather
// than by forbiddenCacheActions membership — the Node setup action gains
// cache behavior only via cacheInputKeys, and nscloud-cache-action's
// cache: input value decides JS-vs-Go, not the action's mere presence.
const (
	nodeSetupActionPrefix    = "actions/setup-node"
	nscloudCacheActionPrefix = "namespacelabs/nscloud-cache-action"
)

// jsEcosystemCacheValues are nscloud-cache-action's own cache: input
// vocabulary values that name a JavaScript-ecosystem cache. "go" is
// deliberately absent — that is the existing, allowed cache this
// invariant leaves alone (ci.yml's test job today).
var jsEcosystemCacheValues = []string{"pnpm", "npm", "node", "yarn"}

// mutableCacheFinding is one JS-scoped mutable-cache step
// scanContentForMutableCache found: which step, its uses: value, and why
// it was classified as a finding.
type mutableCacheFinding struct {
	Step string
	Uses string
	Kind string // "forbidden-action" | "node-setup-cache-input" | "js-ecosystem-cache-value"
}

// mutableCacheScanResult is scanContentForMutableCache's return shape:
// findings, plus the two positive counts (repo rule 84d1gfpywd) that make
// a zero-findings result mean something — steps genuinely examined, and
// cache-bearing steps genuinely classified ALLOWED (the existing Go
// cache), rather than the scanner never having reached the job at all.
type mutableCacheScanResult struct {
	Findings      []mutableCacheFinding
	StepsExamined int
	AllowedCaches int
}

type cacheAwareStep struct {
	Name string                 `yaml:"name"`
	Uses string                 `yaml:"uses"`
	With map[string]interface{} `yaml:"with"`
}

type cacheAwareJob struct {
	Steps []cacheAwareStep `yaml:"steps"`
}

type cacheAwareWorkflow struct {
	Jobs map[string]cacheAwareJob `yaml:"jobs"`
}

// hasAnyCacheInputKey reports whether with declares any key named in
// cacheInputKeys, regardless of that key's value.
func hasAnyCacheInputKey(with map[string]interface{}) bool {
	for _, key := range cacheInputKeys {
		if _, ok := with[key]; ok {
			return true
		}
	}
	return false
}

// scanContentForMutableCache is the content-taking core of the BLD-01
// no-mutable-JS-cache invariant. It classifies every cache-bearing step in
// job jobID into FINDING or ALLOWED:
//   - a step whose uses: owner/repo prefix matches forbiddenCacheActions is
//     a FINDING regardless of inputs;
//   - a step whose uses: owner/repo prefix is the Node setup action and
//     whose with: mapping carries any key in cacheInputKeys is a FINDING;
//   - a step whose uses: owner/repo prefix is nscloud-cache-action whose
//     cache: input names a JavaScript ecosystem (jsEcosystemCacheValues) is
//     a FINDING;
//   - a step whose uses: owner/repo prefix is nscloud-cache-action whose
//     cache: input is "go" is ALLOWED and COUNTED.
//
// Returns a non-nil error when content fails to parse, declares no jobs:,
// names no job jobID, or that job declares zero steps — never a usable
// empty result on any of those misses.
func scanContentForMutableCache(jobID string, content []byte) (mutableCacheScanResult, error) {
	var wf cacheAwareWorkflow
	if err := yaml.Unmarshal(content, &wf); err != nil {
		return mutableCacheScanResult{}, fmt.Errorf("scanContentForMutableCache: %w", err)
	}
	if len(wf.Jobs) == 0 {
		return mutableCacheScanResult{}, fmt.Errorf("scanContentForMutableCache: no jobs: found in workflow source")
	}
	job, ok := wf.Jobs[jobID]
	if !ok {
		return mutableCacheScanResult{}, fmt.Errorf("scanContentForMutableCache: no job %q found in workflow source", jobID)
	}
	if len(job.Steps) == 0 {
		return mutableCacheScanResult{}, fmt.Errorf("scanContentForMutableCache: job %q declares zero steps", jobID)
	}

	var result mutableCacheScanResult
	for _, step := range job.Steps {
		result.StepsExamined++
		prefix := actionPrefix(step.Uses)

		switch {
		case contains(forbiddenCacheActions, prefix):
			result.Findings = append(result.Findings, mutableCacheFinding{Step: step.Name, Uses: step.Uses, Kind: "forbidden-action"})

		case prefix == nodeSetupActionPrefix && hasAnyCacheInputKey(step.With):
			result.Findings = append(result.Findings, mutableCacheFinding{Step: step.Name, Uses: step.Uses, Kind: "node-setup-cache-input"})

		case prefix == nscloudCacheActionPrefix:
			cacheVal, _ := step.With["cache"].(string)
			switch {
			case contains(jsEcosystemCacheValues, cacheVal):
				result.Findings = append(result.Findings, mutableCacheFinding{Step: step.Name, Uses: step.Uses, Kind: "js-ecosystem-cache-value"})
			case cacheVal == "go":
				result.AllowedCaches++
			}
		}
	}
	return result, nil
}

// scanForMutableCache is the thin path-reading wrapper around
// scanContentForMutableCache, always reading ciWorkflowPath — the JS
// install path this invariant guards lives in ci.yml, not release.yml.
func scanForMutableCache(jobID string) (mutableCacheScanResult, error) {
	data, err := os.ReadFile(ciWorkflowPath)
	if err != nil {
		return mutableCacheScanResult{}, fmt.Errorf("scanForMutableCache: read %s: %w", ciWorkflowPath, err)
	}
	return scanContentForMutableCache(jobID, data)
}

// TestJSInstallPathHasNoMutableCache is the BLD-01 structural invariant:
// ci.yml's test job carries no mutable JS-scoped cache. The Node-setup
// clause (actions/setup-node with a cache: input) has NOTHING to match at
// this wave — 02-06 adds that step two waves later — and that is by
// design: authoring the invariant BEFORE the step exists means the step
// cannot arrive with a cache input already attached. What makes this
// pre-arrival zero mean something is AllowedCaches > 0: ci.yml's test job
// demonstrably contains a Go cache step (nscloud-cache-action, cache: go)
// today, so a zero AllowedCaches count would mean this scanner never
// reached it, and its clean verdict would mean nothing.
func TestJSInstallPathHasNoMutableCache(t *testing.T) {
	result, err := scanForMutableCache("test")
	if err != nil {
		t.Fatalf("scanForMutableCache(%q): %v", "test", err)
	}
	if result.StepsExamined == 0 {
		t.Fatalf("scanForMutableCache(%q): examined zero steps", "test")
	}
	if result.AllowedCaches == 0 {
		t.Fatalf("scanForMutableCache(%q): AllowedCaches = 0 — ci.yml's test job demonstrably contains a Go cache step today; a zero count means this scanner never reached it, so its clean verdict means nothing", "test")
	}
	if len(result.Findings) != 0 {
		t.Fatalf("scanForMutableCache(%q): found %d mutable-cache finding(s) on the JS install path — a stale cache producing a node_modules that differs from web/pnpm-lock.yaml is exactly the divergence BLD-01/BLD-03 forbid: %+v", "test", len(result.Findings), result.Findings)
	}
}

// TestMutableCacheScanIsNonVacuous plants each of the three forbidden
// forms in synthetic in-memory YAML, one row each, and asserts exactly one
// finding per row. A fourth row plants the two EXISTING Go forms this
// repository actually uses (nscloud-cache-action with cache: go,
// actions/setup-go with cache: false) and asserts zero findings — pinning
// the boundary so a future widening of the matcher breaks THIS test
// instead of quietly failing TestJSInstallPathHasNoMutableCache against
// the real workflow.
func TestMutableCacheScanIsNonVacuous(t *testing.T) {
	cases := []struct {
		name         string
		src          string
		wantFindings int
	}{
		{
			name:         "actions/cache step",
			src:          "jobs:\n  test:\n    steps:\n      - name: bad\n        uses: actions/cache@v4\n        with:\n          path: web/node_modules\n          key: x\n",
			wantFindings: 1,
		},
		{
			name:         "nscloud-cache-action with a JS ecosystem value",
			src:          "jobs:\n  test:\n    steps:\n      - name: bad\n        uses: namespacelabs/nscloud-cache-action@v1.6.1\n        with:\n          cache: pnpm\n",
			wantFindings: 1,
		},
		{
			name:         "Node setup action with a cache: input",
			src:          "jobs:\n  test:\n    steps:\n      - name: bad\n        uses: actions/setup-node@v4\n        with:\n          cache: pnpm\n",
			wantFindings: 1,
		},
		{
			name:         "existing Go forms stay allowed",
			src:          "jobs:\n  test:\n    steps:\n      - name: go cache\n        uses: namespacelabs/nscloud-cache-action@v1.6.1\n        with:\n          cache: go\n      - name: setup go\n        uses: actions/setup-go@v6\n        with:\n          cache: false\n",
			wantFindings: 0,
		},
	}

	for _, c := range cases {
		result, err := scanContentForMutableCache("test", []byte(c.src))
		if err != nil {
			t.Errorf("scanContentForMutableCache(%q): %v", c.name, err)
			continue
		}
		if len(result.Findings) != c.wantFindings {
			t.Errorf("scanContentForMutableCache(%q): got %d finding(s), want %d: %v", c.name, len(result.Findings), c.wantFindings, result.Findings)
		}
	}
}
