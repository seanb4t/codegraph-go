// changie_shape_test.go binds .changie.yaml, the .changes/ baseline seed
// layout, and the isolated go.tool-changie.mod pin to CHG-01/CHG-02's
// locked vocabulary (01-01-PLAN.md, D-06, D-10, D-11). These guards land
// RED first per rule x1cjy9vyhq: committed before .changie.yaml/.changes/
// exist, so the resulting failure is a real assertion failure, not a
// build error. Never gate on `gsd-tools check tdd-red-evidence` — it
// parses TAP only and always reports INVALID_RED for `go test`; the RED
// evidence is this commit's position on main..HEAD plus the pasted
// `--- FAIL` transcript in the plan's SUMMARY.
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
	"golang.org/x/mod/semver"
)

// --- fixture paths and locked vocabulary (D-06, D-10) -----------------------

const (
	changieConfigPath      = "../../.changie.yaml"
	changieChangesDir      = "../../.changes"
	changieChangelogPath   = "../../CHANGELOG.md"
	changieModfilePath     = "../../go.tool-changie.mod"
	changieToolPackage     = "github.com/miniscruff/changie"
	changieMinVersion      = "v1.26.0"
	changieBaselineVersion = "v0.14.0"
	changieSeedFloor       = 14
)

// changieVersionFormat, changieKindFormat and changieChangeFormat are
// copied byte for byte from .planning/notes/changie-release-management.md
// lines 79-81 (D-10) — the locked design-note vocabulary this phase
// copies, never redesigns. versionFormat contains an em dash: these are
// the literal bytes, not retyped.
const changieVersionFormat = `## [{{.Version}}](https://github.com/seanb4t/codegraph-go/releases/tag/{{.Version}}) — {{.Time.Format "2006-01-02"}}`
const changieKindFormat = `### {{.Kind}}`
const changieChangeFormat = `- {{.Body}}{{if .Custom.PR}} ([#{{.Custom.PR}}](https://github.com/seanb4t/codegraph-go/pull/{{.Custom.PR}})){{end}}`

// changieFragmentFileFormat adds sub-second precision to changie's
// default (seconds-only) fragment filename template — 01-01-PLAN.md Task
// 1 step 5's own measured collision rationale: changie v1.26.0's default
// silently overwrote a same-kind fragment written in the same second.
const changieFragmentFileFormat = `{{.Kind}}-{{.Time.Format "20060102-150405.000000000"}}`

// --- parseX/mustX helper pairs -----------------------------------------
//
// Following internal/upgrade's own house style (rule 84d1gfpywd,
// taskfile_shape_test.go's parseX/mustX idiom): every parser here returns
// a non-nil error on its "nothing found" branch, never a silently-usable
// zero value.

// parseChangieConfig unmarshals src as a YAML document and returns its
// root mapping node. Returns a non-nil error for empty input, a
// non-mapping root, or a missing/empty kinds sequence.
func parseChangieConfig(src []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("parseChangieConfig: unmarshal: %w", err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("parseChangieConfig: empty document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parseChangieConfig: root is not a mapping node (kind=%v)", root.Kind)
	}
	kindsNode, ok := yamlMappingValue(root, "kinds")
	if !ok || kindsNode.Kind != yaml.SequenceNode || len(kindsNode.Content) == 0 {
		return nil, fmt.Errorf("parseChangieConfig: missing or empty kinds sequence")
	}
	return root, nil
}

func mustChangieConfig(t *testing.T, src []byte) *yaml.Node {
	t.Helper()
	n, err := parseChangieConfig(src)
	if err != nil {
		t.Fatalf("mustChangieConfig: %v", err)
	}
	return n
}

// yamlMappingKeys returns n's mapping keys, sorted.
func yamlMappingKeys(n *yaml.Node) []string {
	var keys []string
	for i := 0; i+1 < len(n.Content); i += 2 {
		keys = append(keys, n.Content[i].Value)
	}
	sort.Strings(keys)
	return keys
}

// yamlMappingValue returns n's value node for key, and whether it was
// found.
func yamlMappingValue(n *yaml.Node, key string) (*yaml.Node, bool) {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1], true
		}
	}
	return nil, false
}

// changelogHeadingRe matches a release-please-authored or changie-merged
// version heading. The optional "v" accepts the `## [v0.15.0](…)` shape
// changie batches write under the locked versionFormat (D-03) from this
// phase's first real release on, so the guard survives it.
var changelogHeadingRe = regexp.MustCompile(`^## \[v?(\d+\.\d+\.\d+)\]`)

// parseChangelogVersionHeadings returns every "## [x.y.z]" (or
// "## [vx.y.z]") heading in src, as "v"+capture, in file order. Returns a
// non-nil error on zero matches or a duplicated version.
func parseChangelogVersionHeadings(src string) ([]string, error) {
	seen := make(map[string]bool)
	var versions []string
	for _, line := range strings.Split(src, "\n") {
		m := changelogHeadingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		v := "v" + m[1]
		if seen[v] {
			return nil, fmt.Errorf("parseChangelogVersionHeadings: duplicated version heading %q", v)
		}
		seen[v] = true
		versions = append(versions, v)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("parseChangelogVersionHeadings: zero '## [x.y.z]' headings found")
	}
	return versions, nil
}

// listChangieVersionSeeds returns the sorted stems of every v*.md file in
// dir. Returns a non-nil error on zero matches.
func listChangieVersionSeeds(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "v*.md"))
	if err != nil {
		return nil, fmt.Errorf("listChangieVersionSeeds: glob %s: %w", dir, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("listChangieVersionSeeds: zero v*.md files found in %s", dir)
	}
	stems := make([]string, 0, len(matches))
	for _, m := range matches {
		stems = append(stems, strings.TrimSuffix(filepath.Base(m), ".md"))
	}
	sort.Strings(stems)
	return stems, nil
}

// --- TestChangieConfigShape (CHG-01, D-06) ---------------------------------

func TestChangieConfigShape(t *testing.T) {
	data, err := os.ReadFile(changieConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", changieConfigPath, err)
	}
	root := mustChangieConfig(t, data)

	wantKeys := []string{
		"changeFormat", "changelogPath", "changesDir", "custom",
		"fragmentFileFormat", "headerPath", "kindFormat", "kinds",
		"newlines", "unreleasedDir", "versionExt", "versionFormat",
	}
	sort.Strings(wantKeys)
	if gotKeys := yamlMappingKeys(root); !slices.Equal(gotKeys, wantKeys) {
		t.Fatalf("%s: root mapping key set = %v, want exactly %v", changieConfigPath, gotKeys, wantKeys)
	}

	wantScalars := map[string]string{
		"changesDir":    ".changes",
		"unreleasedDir": "unreleased",
		"headerPath":    "header.tpl.md",
		"changelogPath": "CHANGELOG.md",
		"versionExt":    "md",
	}
	for _, key := range sortedMapKeys(wantScalars) {
		want := wantScalars[key]
		v, ok := yamlMappingValue(root, key)
		if !ok {
			t.Fatalf("%s: missing key %q", changieConfigPath, key)
		}
		if v.Value != want {
			t.Fatalf("%s: %s = %q, want %q", changieConfigPath, key, v.Value, want)
		}
	}

	wantFormats := map[string]string{
		"versionFormat":      changieVersionFormat,
		"kindFormat":         changieKindFormat,
		"changeFormat":       changieChangeFormat,
		"fragmentFileFormat": changieFragmentFileFormat,
	}
	for _, key := range sortedMapKeys(wantFormats) {
		want := wantFormats[key]
		v, ok := yamlMappingValue(root, key)
		if !ok {
			t.Fatalf("%s: missing key %q", changieConfigPath, key)
		}
		if v.Value != want {
			t.Fatalf("%s: %s = %q, want %q", changieConfigPath, key, v.Value, want)
		}
	}

	kindsNode, ok := yamlMappingValue(root, "kinds")
	if !ok || kindsNode.Kind != yaml.SequenceNode {
		t.Fatalf("%s: kinds is not a sequence", changieConfigPath)
	}
	type wantKind struct {
		label string
		auto  string
	}
	wantKinds := []wantKind{
		{"Breaking", "minor"},
		{"Features", "minor"},
		{"Fixes", "patch"},
		{"Performance", "patch"},
		{"Dependencies", "patch"},
	}
	if len(kindsNode.Content) != len(wantKinds) {
		t.Fatalf("%s: kinds has %d entries, want %d (by position: %v)", changieConfigPath, len(kindsNode.Content), len(wantKinds), wantKinds)
	}
	wantKindKeys := []string{"auto", "label"}
	for i, want := range wantKinds {
		kindNode := kindsNode.Content[i]
		if gotKeys := yamlMappingKeys(kindNode); !slices.Equal(gotKeys, wantKindKeys) {
			t.Fatalf("%s: kinds[%d] key set = %v, want exactly %v", changieConfigPath, i, gotKeys, wantKindKeys)
		}
		labelNode, _ := yamlMappingValue(kindNode, "label")
		autoNode, _ := yamlMappingValue(kindNode, "auto")
		if labelNode.Value != want.label {
			t.Fatalf("%s: kinds[%d].label = %q, want %q (position-order matters — it fixes the kind-section order of every future batch)", changieConfigPath, i, labelNode.Value, want.label)
		}
		if autoNode.Value != want.auto {
			t.Fatalf("%s: kinds[%d].auto = %q, want %q", changieConfigPath, i, autoNode.Value, want.auto)
		}
		if want.label == "Breaking" && !strings.Contains(autoNode.LineComment, "Flip to major at 1.0") {
			t.Fatalf("%s: kinds[%d] (Breaking) auto value's inline comment = %q, want it to contain %q", changieConfigPath, i, autoNode.LineComment, "Flip to major at 1.0")
		}
	}

	customNode, ok := yamlMappingValue(root, "custom")
	if !ok || customNode.Kind != yaml.SequenceNode || len(customNode.Content) != 1 {
		t.Fatalf("%s: custom must be a sequence of exactly one entry", changieConfigPath)
	}
	prNode := customNode.Content[0]
	wantCustomKeys := []string{"key", "minInt", "type"}
	sort.Strings(wantCustomKeys)
	if gotKeys := yamlMappingKeys(prNode); !slices.Equal(gotKeys, wantCustomKeys) {
		t.Fatalf("%s: custom[0] key set = %v, want exactly %v (an added optional key must fail — D-11)", changieConfigPath, gotKeys, wantCustomKeys)
	}
	keyNode, _ := yamlMappingValue(prNode, "key")
	typeNode, _ := yamlMappingValue(prNode, "type")
	minIntNode, _ := yamlMappingValue(prNode, "minInt")
	if keyNode.Value != "PR" {
		t.Fatalf("%s: custom[0].key = %q, want %q", changieConfigPath, keyNode.Value, "PR")
	}
	if typeNode.Value != "int" {
		t.Fatalf("%s: custom[0].type = %q, want %q", changieConfigPath, typeNode.Value, "int")
	}
	if minIntNode.Value != "1" {
		t.Fatalf("%s: custom[0].minInt = %q, want %q", changieConfigPath, minIntNode.Value, "1")
	}

	newlinesNode, ok := yamlMappingValue(root, "newlines")
	if !ok || newlinesNode.Kind != yaml.MappingNode {
		t.Fatalf("%s: newlines is not a mapping", changieConfigPath)
	}
	wantNewlinesKeys := []string{"afterChangelogHeader"}
	if gotKeys := yamlMappingKeys(newlinesNode); !slices.Equal(gotKeys, wantNewlinesKeys) {
		t.Fatalf("%s: newlines key set = %v, want exactly %v (D-02: no other newlines key — seeds already carry their own trailing blank-line separator)", changieConfigPath, gotKeys, wantNewlinesKeys)
	}
	afterNode, _ := yamlMappingValue(newlinesNode, "afterChangelogHeader")
	if afterNode.Value != "1" {
		t.Fatalf("%s: newlines.afterChangelogHeader = %q, want %q", changieConfigPath, afterNode.Value, "1")
	}
}

// sortedMapKeys returns m's keys, sorted — a small local helper so the
// tests above iterate map-typed fixtures deterministically without
// reaching for a package-level generic.
func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- TestChangieBaselineLayout (CHG-02, D-06, D-09) -------------------------

func TestChangieBaselineLayout(t *testing.T) {
	headerPath := filepath.Join(changieChangesDir, "header.tpl.md")
	data, err := os.ReadFile(headerPath)
	if err != nil {
		t.Fatalf("read %s: %v", headerPath, err)
	}
	if string(data) != "# Changelog\n" {
		t.Fatalf("%s = %q, want exactly %q (D-09: no preamble, no extra bytes)", headerPath, string(data), "# Changelog\n")
	}

	gitkeepPath := filepath.Join(changieChangesDir, "unreleased", ".gitkeep")
	info, err := os.Stat(gitkeepPath)
	if err != nil {
		t.Fatalf("stat %s: %v", gitkeepPath, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("%s is not a regular file (mode=%v)", gitkeepPath, info.Mode())
	}

	baselinePath := filepath.Join(changieChangesDir, changieBaselineVersion+".md")
	baselineData, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read %s: %v", baselinePath, err)
	}
	firstLine := strings.SplitN(string(baselineData), "\n", 2)[0]
	const wantPrefix = "## [0.14.0]("
	if !strings.HasPrefix(firstLine, wantPrefix) {
		t.Fatalf("%s: first line = %q, want a prefix of %q (CHG-02 baseline, pinned permanently)", baselinePath, firstLine, wantPrefix)
	}
}

// --- TestChangieVersionSeedsMatchChangelog (CHG-02) -------------------------

func TestChangieVersionSeedsMatchChangelog(t *testing.T) {
	seeds, err := listChangieVersionSeeds(changieChangesDir)
	if err != nil {
		t.Fatalf("listChangieVersionSeeds(%s): %v", changieChangesDir, err)
	}
	changelogData, err := os.ReadFile(changieChangelogPath)
	if err != nil {
		t.Fatalf("read %s: %v", changieChangelogPath, err)
	}
	headings, err := parseChangelogVersionHeadings(string(changelogData))
	if err != nil {
		t.Fatalf("parseChangelogVersionHeadings(%s): %v", changieChangelogPath, err)
	}

	t.Logf("%s: %d seed files; %s: %d version headings", changieChangesDir, len(seeds), changieChangelogPath, len(headings))

	if len(seeds) < changieSeedFloor {
		t.Fatalf("%s: %d seed files, want at least %d", changieChangesDir, len(seeds), changieSeedFloor)
	}
	if len(headings) < changieSeedFloor {
		t.Fatalf("%s: %d version headings, want at least %d", changieChangelogPath, len(headings), changieSeedFloor)
	}

	seedSet := make(map[string]bool, len(seeds))
	for _, s := range seeds {
		seedSet[s] = true
	}
	headingSet := make(map[string]bool, len(headings))
	for _, h := range headings {
		headingSet[h] = true
	}

	var onlyInSeeds, onlyInHeadings []string
	for s := range seedSet {
		if !headingSet[s] {
			onlyInSeeds = append(onlyInSeeds, s)
		}
	}
	for h := range headingSet {
		if !seedSet[h] {
			onlyInHeadings = append(onlyInHeadings, h)
		}
	}
	sort.Strings(onlyInSeeds)
	sort.Strings(onlyInHeadings)

	if len(onlyInSeeds) > 0 || len(onlyInHeadings) > 0 {
		t.Fatalf("%s vs %s: set mismatch — versions only in seeds: %v, versions only in changelog headings: %v", changieChangesDir, changieChangelogPath, onlyInSeeds, onlyInHeadings)
	}
}

// --- TestChangieToolPinnedInIsolatedModfile (CHG-01, D-04) ------------------

func TestChangieToolPinnedInIsolatedModfile(t *testing.T) {
	data, err := os.ReadFile(changieModfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", changieModfilePath, err)
	}
	src := string(data)

	pkgs, err := parseGoModToolPackages(src)
	if err != nil {
		t.Fatalf("parseGoModToolPackages(%s): %v", changieModfilePath, err)
	}
	if len(pkgs) != 1 || pkgs[0] != changieToolPackage {
		t.Fatalf("%s: tool packages = %v, want exactly [%s]", changieModfilePath, pkgs, changieToolPackage)
	}

	version, err := parseGoModRequireVersion(src, changieToolPackage)
	if err != nil {
		t.Fatalf("parseGoModRequireVersion(%s, %s): %v", changieModfilePath, changieToolPackage, err)
	}
	if !semver.IsValid(version) {
		t.Fatalf("%s: required %s version %q is not valid semver", changieModfilePath, changieToolPackage, version)
	}
	if semver.Compare(version, changieMinVersion) < 0 {
		t.Fatalf("%s: required %s version %q is older than the pinned floor %q", changieModfilePath, changieToolPackage, version, changieMinVersion)
	}

	if !slices.Contains(isolatedModfilePaths, changieModfilePath) {
		t.Fatalf("isolatedModfilePaths %v does not contain %s (D-04)", isolatedModfilePaths, changieModfilePath)
	}
	if !slices.Contains(forbiddenToolPackages, changieToolPackage) {
		t.Fatalf("forbiddenToolPackages %v does not contain %s (D-04)", forbiddenToolPackages, changieToolPackage)
	}

	header := mustToolModfileHeaderComment(t, src)
	lowerHeader := strings.ToLower(header)
	for _, want := range []string{"isolat", "gowork=off", "task changie"} {
		if !strings.Contains(lowerHeader, want) {
			t.Fatalf("%s: header comment does not mention %q, got: %q", changieModfilePath, want, header)
		}
	}
}

// --- TestChangieWrapperTaskRecordsInstallPath (CHG-01, D-05) ----------------

func TestChangieWrapperTaskRecordsInstallPath(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	src := string(data)

	const wantVarLine = "  GO_TOOL_CHANGIE: GOWORK=off go tool -modfile=go.tool-changie.mod"
	if !strings.Contains(src, wantVarLine) {
		t.Fatalf("%s: does not contain the exact line %q", taskfilePath, wantVarLine)
	}

	blocks := mustParseTaskBlocks(t, src)
	block, ok := blocks["changie"]
	if !ok {
		t.Fatalf("%s: no top-level %q task", taskfilePath, "changie")
	}

	desc, err := parseTaskDescription(block)
	if err != nil {
		t.Fatalf("parseTaskDescription(changie): %v", err)
	}
	if !strings.Contains(desc, "go.tool-changie.mod") {
		t.Fatalf("changie task desc %q does not mention go.tool-changie.mod (D-05)", desc)
	}

	if !blockDeclaresKey(block, "silent") {
		t.Fatalf("changie task block does not declare silent: — stdout must carry only changie's own output:\n%s", block)
	}

	const wantCmd = "{{.GO_TOOL_CHANGIE}} changie {{.CLI_ARGS}}"
	if !strings.Contains(block, wantCmd) {
		t.Fatalf("changie task block does not contain the exact cmds entry %q, got:\n%s", wantCmd, block)
	}
}

// --- TestChangieShapeParsersFailLoudly ---------------------------------

func TestChangieShapeParsersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "parseChangieConfig: empty input",
			fn: func() error {
				_, err := parseChangieConfig([]byte(""))
				return err
			},
		},
		{
			name: "parseChangieConfig: empty kinds sequence",
			fn: func() error {
				_, err := parseChangieConfig([]byte("kinds: []\n"))
				return err
			},
		},
		{
			name: "parseChangelogVersionHeadings: empty input",
			fn: func() error {
				_, err := parseChangelogVersionHeadings("")
				return err
			},
		},
		{
			name: "parseChangelogVersionHeadings: no headings",
			fn: func() error {
				_, err := parseChangelogVersionHeadings("# Changelog\n")
				return err
			},
		},
		{
			name: "parseChangelogVersionHeadings: duplicated heading",
			fn: func() error {
				_, err := parseChangelogVersionHeadings("## [0.1.0](x)\n\n## [0.1.0](x)\n")
				return err
			},
		},
		{
			name: "listChangieVersionSeeds: empty dir",
			fn: func() error {
				_, err := listChangieVersionSeeds(t.TempDir())
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

// --- TestChangieCheckWiredIntoCI (CHG-03, D-07, D-08) -----------------------

// TestChangieCheckWiredIntoCI asserts ci.yml's test job runs `task
// check:changie` immediately after `task docs:cli:drift` (D-08), and that
// Taskfile.yml's check:changie block declares actionable preconditions and
// builds from the pinned go.tool-changie.mod (D-07).
func TestChangieCheckWiredIntoCI(t *testing.T) {
	ciData, err := os.ReadFile(ciWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", ciWorkflowPath, err)
	}
	steps, err := parseWorkflowJobSteps(string(ciData), "test")
	if err != nil {
		t.Fatalf("parseWorkflowJobSteps(%s, %q): %v", ciWorkflowPath, "test", err)
	}

	docsDriftIdx, checkChangieIdx := -1, -1
	var stripped []string
	for i, step := range steps {
		s := stripRunBodyNoise(step.Run)
		stripped = append(stripped, s)
		switch s {
		case "task docs:cli:drift":
			if docsDriftIdx != -1 {
				t.Fatalf("%s job %q: more than one step's run: body strips to %q (indices %d and %d): %v", ciWorkflowPath, "test", "task docs:cli:drift", docsDriftIdx, i, stripped)
			}
			docsDriftIdx = i
		case "task check:changie":
			if checkChangieIdx != -1 {
				t.Fatalf("%s job %q: more than one step's run: body strips to %q (indices %d and %d): %v", ciWorkflowPath, "test", "task check:changie", checkChangieIdx, i, stripped)
			}
			checkChangieIdx = i
		}
	}
	if docsDriftIdx == -1 {
		t.Fatalf("%s job %q: no step's run: body strips to %q — observed steps: %v", ciWorkflowPath, "test", "task docs:cli:drift", stripped)
	}
	if checkChangieIdx == -1 {
		t.Fatalf("%s job %q: no step's run: body strips to %q — observed steps: %v", ciWorkflowPath, "test", "task check:changie", stripped)
	}
	if checkChangieIdx != docsDriftIdx+1 {
		t.Fatalf("%s job %q: %q is at index %d, want %d (immediately after %q at index %d, D-08) — observed steps: %v", ciWorkflowPath, "test", "task check:changie", checkChangieIdx, docsDriftIdx+1, "task docs:cli:drift", docsDriftIdx, stripped)
	}

	taskfileData, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	src := string(taskfileData)
	blocks := mustParseTaskBlocks(t, src)
	block, ok := blocks["check:changie"]
	if !ok {
		t.Fatalf("%s: no top-level %q task", taskfilePath, "check:changie")
	}
	msgs, err := parsePreconditionMessages(block)
	if err != nil {
		t.Fatalf("parsePreconditionMessages(check:changie): %v", err)
	}
	if len(msgs) == 0 {
		t.Fatalf("check:changie task block declares preconditions: but no non-empty msg: values")
	}
	const wantModfileToken = "-modfile=go.tool-changie.mod"
	if !strings.Contains(block, wantModfileToken) {
		t.Fatalf("check:changie task block does not contain %q (D-07: must build from the pinned modfile)", wantModfileToken)
	}
}

// --- TestChangieBinaryInToolVulnScan (CHG-01) -------------------------------

// changieVulnForLoopRe matches the vuln target's for-loop name list line
// naming changie as a whole word — the shape TestChangieBinaryInToolVulnScan
// requires so a future edit that drops changie from the loop, or renames it
// to something ambiguous, fails loudly here.
var changieVulnForLoopRe = regexp.MustCompile(`(?m)^\s*for name in .*\bchangie\b.*; do$`)

func TestChangieBinaryInToolVulnScan(t *testing.T) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", taskfilePath, err)
	}
	blocks := mustParseTaskBlocks(t, string(data))
	block, ok := blocks["vuln"]
	if !ok {
		t.Fatalf("%s: no top-level %q task", taskfilePath, "vuln")
	}

	const wantModfileToken = "-modfile=go.tool-changie.mod"
	if !strings.Contains(block, wantModfileToken) {
		t.Fatalf("vuln task block does not contain %q — changie is not built from the pinned modfile", wantModfileToken)
	}
	if !strings.Contains(block, changieToolPackage) {
		t.Fatalf("vuln task block does not contain %q — changie's package is not built for the scan", changieToolPackage)
	}
	if !changieVulnForLoopRe.MatchString(block) {
		t.Fatalf("vuln task block has no 'for name in ...changie...; do' loop line naming changie as a whole word:\n%s", block)
	}
}
