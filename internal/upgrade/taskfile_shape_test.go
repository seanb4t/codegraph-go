package upgrade

import (
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
	rootGoModPath    = "../../go.mod"
	toolModfilePath  = "../../go.tool.mod"
	lintModfilePath  = "../../go.tool-lint.mod"
	taskfilePath     = "../../Taskfile.yml"
	goreleaserPath   = "../../.goreleaser.yaml"
	checkCrossTaskID = "check:cross"
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
// earlier plans of this phase (bench.yml's rebless/headtohead/diagnostic
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
