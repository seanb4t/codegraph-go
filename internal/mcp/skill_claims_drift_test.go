package mcp

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// GUARD-01 (05-CONTEXT.md, T-06-05): SKILL.md is the third surface in this
// repository's documented wire-claim-drift bug class — after tools.go's
// SURF-01 (a "default 5" claim advertised for a whole phase after the
// engine constant became 2) and the 2026-08-08 misdirection incident (an
// agent misled by a stale instructions string that named only one of three
// tool-visibility mechanisms). Phase 6's own notes make SKILL.md explicitly
// subject to the same GUARD-01 discipline the resource docs already carry.
// Every checker below reads BOTH the claim SKILL.md makes and the expected
// value from its real source — allToolNames(), resourceURIFor,
// companionNames, allowlistEnvName — never a hand-typed literal on either
// side, following the shape resources_schema_drift_test.go and
// instructions_contract_test.go already established.

// skillPath is the repository's dogfooded agent skill, reached from
// internal/mcp. Mirrors instructions_contract_test.go's readmePath
// convention: a relative path read directly, with a missing file treated
// as a loud failure (t.Fatalf) rather than a skip.
const skillPath = "../../.claude/skills/codegraph/SKILL.md"

// skillDirName is the agentskills.io rule that a skill's frontmatter name
// must equal the containing directory's name. This literal IS that
// directory name (".claude/skills/codegraph/") — there is no runtime value
// to derive it from, since the directory name is itself the fact under
// test.
const skillDirName = "codegraph"

// skillBodyMaxBytes is the four-bytes-per-token approximation of
// agentskills.io's ~5000-token body budget. RESEARCH.md Open Question 2
// resolves the token budget as the spec-authoritative ceiling and the line
// count below as a community-guidance proxy checked alongside it, not
// instead of it. The byte gate is the mechanically checkable stand-in
// because this repository carries no tokenizer and this milestone adds no
// dependency to acquire one.
const skillBodyMaxBytes = 20000

// skillBodyMaxLines is the practical ~500-line proxy repeated across
// third-party skill-authoring guidance, checked alongside skillBodyMaxBytes
// rather than as a substitute for it.
const skillBodyMaxLines = 500

// skillDescriptionMaxChars is agentskills.io's stated maximum length for a
// skill's frontmatter description field.
const skillDescriptionMaxChars = 1024

// skillWorkedExampleCount is D-05's locked worked-example count (06-CONTEXT.md:
// "Three worked examples, in this order") — a locked user decision, not a
// value derivable from code. Recorded as a named constant, with this doc
// comment stating its source, so the number cannot be mistaken for a
// source-derived one the way every other checker in this file compares
// against a real value.
const skillWorkedExampleCount = 3

// nudgeScriptPath reaches the SessionStart nudge script 06-01 shipped, from
// internal/mcp. This file's checks on its content are the SECOND layer over
// the nudge text: internal/agents/hookpackage_test.go's nudgeLine constant
// pins the emitted bytes exactly (the FIRST layer, proving nothing about
// their honesty); the checks here pin the same text against the live tool
// roster (allToolNames()) and the real environment-variable name
// (allowlistEnvName), so a future edit to the message cannot pass both
// layers while naming something that does not exist.
const nudgeScriptPath = "../../.claude/hooks/session-nudge.sh"

// workedExampleHeadingRe matches a level-3 markdown heading line, anchored
// to the start of a line via the (?m) multiline flag. The trailing space in
// the pattern is what excludes a level-4 ("#### ") heading from matching.
var workedExampleHeadingRe = regexp.MustCompile(`(?m)^### `)

// countSkillWorkedExamples locates the worked-examples "## " section — the
// first "## " heading whose text contains "example" case-insensitively —
// bounds it at the next "## " heading or the end of the document, and
// counts workedExampleHeadingRe matches strictly within those bounds. The
// bounding matters: an unbounded count would silently absorb a "### "
// heading added to any other section later (e.g. a future reference or
// troubleshooting section with its own sub-headings), overcounting what the
// worked-examples section itself actually carries. Returns 0 if no
// worked-examples heading is found at all.
func countSkillWorkedExamples(body string) int {
	lines := strings.Split(body, "\n")

	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") && strings.Contains(strings.ToLower(line), "example") {
			start = i
			break
		}
	}
	if start == -1 {
		return 0
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}

	section := strings.Join(lines[start+1:end], "\n")
	return len(workedExampleHeadingRe.FindAllString(section, -1))
}

// resourceURIRe matches a codegraph:// resource URI. The character class
// deliberately excludes trailing punctuation (a period, comma, or closing
// paren at the end of a sentence) so a URI mentioned at the end of prose
// still matches exactly, with nothing borrowed from the following
// punctuation.
var resourceURIRe = regexp.MustCompile(`codegraph://[a-z0-9][a-z0-9/_-]*`)

// tableSeparatorRe matches a markdown table separator row: a line made
// entirely of whitespace, pipes, hyphens and colons. Callers additionally
// require the line to contain at least one pipe, since the character class
// alone would also accept an all-hyphen or all-whitespace line that is not
// a table row at all.
var tableSeparatorRe = regexp.MustCompile(`^[\s|:-]+$`)

// splitSkillFrontmatter requires doc to begin with a "---" fence line,
// finds the next "---" fence line, and returns the text between them as
// front and everything after the closing fence as body. It returns a
// descriptive error if doc does not begin with the opening fence or the
// fence is never closed. No YAML dependency is added — this milestone adds
// zero Go modules, and the frontmatter shape this package needs (flat
// "key: value" lines) does not require a parser.
func splitSkillFrontmatter(doc string) (front, body string, err error) {
	lines := strings.Split(doc, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return "", "", fmt.Errorf("document does not begin with a --- frontmatter fence")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			front = strings.Join(lines[1:i], "\n")
			body = strings.Join(lines[i+1:], "\n")
			return front, body, nil
		}
	}
	return "", "", fmt.Errorf("document's frontmatter fence is never closed (no second --- line found)")
}

// skillFrontmatterField scans front's lines for the exact prefix
// key + ":" at column zero and returns the trimmed remainder of the first
// matching line. The exact-prefix rule (the colon is part of the prefix
// being matched, not just the key) is what lets a field like "name" be
// looked up without accidentally matching a line for a differently-named
// field whose name happens to start with the same characters, such as
// "namespace".
func skillFrontmatterField(front, key string) (string, bool) {
	prefix := key + ":"
	for _, line := range strings.Split(front, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(line[len(prefix):]), true
		}
	}
	return "", false
}

// decisionTablePrecedesSecondSection implements SKILL-01's structural
// property: body must have at least two "## " headings, and the first
// markdown table separator row found anywhere in body must fall strictly
// after the first heading and strictly before the second. Every failure
// path names the line indices involved, so a failing test tells the
// reader exactly where the table and the headings landed relative to each
// other.
func decisionTablePrecedesSecondSection(body string) error {
	lines := strings.Split(body, "\n")

	var headingIdx []int
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			headingIdx = append(headingIdx, i)
		}
	}
	if len(headingIdx) < 2 {
		return fmt.Errorf("body has %d '## ' heading(s) (indices %v), need at least 2 to place a decision table before the second", len(headingIdx), headingIdx)
	}
	first, second := headingIdx[0], headingIdx[1]

	tableIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "|") && tableSeparatorRe.MatchString(line) {
			tableIdx = i
			break
		}
	}
	if tableIdx == -1 {
		return fmt.Errorf("no markdown table separator row found in body (first '## ' heading at line %d, second at line %d)", first, second)
	}
	if !(tableIdx > first && tableIdx < second) {
		return fmt.Errorf("table separator row at line %d does not fall strictly between the first '## ' heading (line %d) and the second (line %d)", tableIdx, first, second)
	}
	return nil
}

// TestSkillFrontmatterIsSpecCompliant is SKILL-01's frontmatter half: the
// document starts with a --- fence, carries a name field equal to
// skillDirName (agentskills.io's rule that name matches the containing
// directory), and carries a trigger-shaped description field — non-empty,
// within skillDescriptionMaxChars, no embedded newline, and containing
// "when" case-insensitively, since a description that only summarizes what
// the skill does never triggers on the shape of question it answers.
func TestSkillFrontmatterIsSpecCompliant(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v (SKILL.md is this phase's own deliverable — a missing file is the RED state, not a skip)", skillPath, err)
	}
	doc := string(data)

	front, _, err := splitSkillFrontmatter(doc)
	if err != nil {
		t.Fatalf("splitSkillFrontmatter(%s): %v", skillPath, err)
	}

	name, ok := skillFrontmatterField(front, "name")
	if !ok {
		t.Fatalf("%s frontmatter has no name field", skillPath)
	}
	if name != skillDirName {
		t.Errorf("%s frontmatter name = %q, want %q (agentskills.io requires name to equal the containing directory's name)", skillPath, name, skillDirName)
	}

	description, ok := skillFrontmatterField(front, "description")
	if !ok {
		t.Fatalf("%s frontmatter has no description field", skillPath)
	}
	if description == "" {
		t.Errorf("%s frontmatter description is empty", skillPath)
	}
	if len(description) > skillDescriptionMaxChars {
		t.Errorf("%s frontmatter description is %d characters, over the %d-character agentskills.io maximum", skillPath, len(description), skillDescriptionMaxChars)
	}
	if strings.ContainsAny(description, "\n\r") {
		t.Errorf("%s frontmatter description contains a newline; it must be a single line", skillPath)
	}
	if !strings.Contains(strings.ToLower(description), "when") {
		t.Errorf("%s frontmatter description never contains \"when\" — the description is the ONLY part of the skill always resident in context, so it must be trigger-shaped (state when to use the skill), not merely a summary of what it does", skillPath)
	}
}

// TestSkillLeadsWithDecisionTable is SKILL-01's load-bearing structural
// property: the decision procedure comes first structurally, not merely in
// intent. See decisionTablePrecedesSecondSection's own non-vacuity coverage
// in TestSkillStructureCheckersAreNotVacuous.
func TestSkillLeadsWithDecisionTable(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	_, body, err := splitSkillFrontmatter(string(data))
	if err != nil {
		t.Fatalf("splitSkillFrontmatter(%s): %v", skillPath, err)
	}
	if err := decisionTablePrecedesSecondSection(body); err != nil {
		t.Errorf("%s does not lead with a decision table: %v", skillPath, err)
	}
}

// TestSkillStaysWithinBudget enforces both the byte-budget approximation of
// agentskills.io's token ceiling and the line-count community proxy,
// reporting both actuals against both limits in one message so a failure
// names the full picture at once.
func TestSkillStaysWithinBudget(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	_, body, err := splitSkillFrontmatter(string(data))
	if err != nil {
		t.Fatalf("splitSkillFrontmatter(%s): %v", skillPath, err)
	}
	byteLen := len(body)
	lineCount := strings.Count(body, "\n") + 1
	if byteLen > skillBodyMaxBytes || lineCount > skillBodyMaxLines {
		t.Errorf("%s body is %d bytes (limit %d, the 4-bytes-per-token approximation of agentskills.io's ~5000-token body budget) and %d lines (limit %d, the community-guidance proxy checked alongside the byte gate, not instead of it)",
			skillPath, byteLen, skillBodyMaxBytes, lineCount, skillBodyMaxLines)
	}
}

// TestSkillNamesOnlyRealTools is GUARD-01's tool-name half for SKILL.md: it
// reuses the package's own toolNameTokenRe (resources_schema_drift_test.go,
// non-vacuity already proven there) rather than reimplementing it, scanning
// the whole document since a stray tool token in the frontmatter would be
// just as wrong as one in the body.
func TestSkillNamesOnlyRealTools(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	doc := string(data)

	known := make(map[string]bool, len(allToolNames()))
	for _, name := range allToolNames() {
		known[name] = true
	}

	matches := toolNameTokenRe.FindAllString(doc, -1)
	if len(matches) == 0 {
		t.Fatal("found zero codegraph_<name> tokens in SKILL.md — this test would verify nothing")
	}
	for _, m := range matches {
		if !known[m] {
			t.Errorf("%s names %s, which is not a member of allToolNames() — a renamed or removed tool left behind in the skill", skillPath, m)
		}
	}
}

// TestSkillResourceURIsResolve is GUARD-01's resource-pointer half: every
// codegraph:// URI SKILL.md names must be a value resourceURIFor actually
// serves — a dead pointer here sends an agent to a resource read that
// fails, the exact broken-promise class this milestone retires.
func TestSkillResourceURIsResolve(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	doc := string(data)

	known := make(map[string]bool, len(resourceURIFor))
	for _, uri := range resourceURIFor {
		known[uri] = true
	}

	matches := resourceURIRe.FindAllString(doc, -1)
	if len(matches) == 0 {
		t.Fatal("found zero codegraph:// URIs in SKILL.md — this test would verify nothing")
	}
	for _, m := range matches {
		if !known[m] {
			t.Errorf("%s names %s, which is not a value in resourceURIFor — the skill points at a resource the server does not serve", skillPath, m)
		}
	}
}

// TestSkillDefersNumericFactsToResources is Phase 5 D-01's anti-duplication
// decision extended to SKILL.md: the body must state zero default-or-max
// numeric claims, deferring every such fact to the tool's own
// codegraph://tools/<stem> resource. Scoped to body (not the whole
// document) since the frontmatter description is prose about triggering,
// not a place a numeric claim would ever legitimately belong either way.
// Reuses numericClaimsMultiset (resources_schema_drift_test.go), whose own
// non-vacuity is proven by TestMCPResourceNumericClaimsMatchToolSchemas'
// use of it — not duplicated here.
func TestSkillDefersNumericFactsToResources(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	_, body, err := splitSkillFrontmatter(string(data))
	if err != nil {
		t.Fatalf("splitSkillFrontmatter(%s): %v", skillPath, err)
	}
	claims := numericClaimsMultiset(body)
	if len(claims) != 0 {
		t.Errorf("%s body states numeric claim(s) %v — every default or maximum belongs on the tool's own codegraph://tools/<stem> resource, not in SKILL.md (Phase 5 D-01 anti-duplication)", skillPath, claims)
	}
}

// TestSkillCountClaimsMatchSourceSets is GUARD-01's count half for
// SKILL.md: every companion-tagged claim in the body must equal
// len(companionNames) and every tool-tagged claim must equal
// len(allToolNames()). Scoped to body per the plan's own behavior
// specification. Reuses countClaimsIn (resources_schema_drift_test.go);
// its own non-vacuity is proven by TestResourceCountCheckerIsNotVacuous,
// not duplicated here.
func TestSkillCountClaimsMatchSourceSets(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	_, body, err := splitSkillFrontmatter(string(data))
	if err != nil {
		t.Fatalf("splitSkillFrontmatter(%s): %v", skillPath, err)
	}
	claims := countClaimsIn(body)
	if len(claims) == 0 {
		t.Fatal("found zero count claims in SKILL.md body — this test would verify nothing")
	}
	for _, claim := range claims {
		switch claim.noun {
		case "companion":
			if claim.value != len(companionNames) {
				t.Errorf("%s claims %d companion tool(s), but len(companionNames) is %d", skillPath, claim.value, len(companionNames))
			}
		case "tool":
			if claim.value != len(allToolNames()) {
				t.Errorf("%s claims %d tool(s) in total, but len(allToolNames()) is %d", skillPath, claim.value, len(allToolNames()))
			}
		}
	}
}

// TestSkillEnvVarNamesAreReal is GUARD-01's env-var half for SKILL.md: the
// only CODEGRAPH_-prefixed token the skill may name is allowlistEnvName.
// Reuses envVarTokenRe (resources_schema_drift_test.go); its own
// non-vacuity is proven by TestResourceEnvVarNamesAreReal, not duplicated
// here.
func TestSkillEnvVarNamesAreReal(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	doc := string(data)

	matches := envVarTokenRe.FindAllString(doc, -1)
	if len(matches) == 0 {
		t.Fatal("found zero CODEGRAPH_-prefixed tokens in SKILL.md — this test would verify nothing")
	}
	for _, m := range matches {
		if m != allowlistEnvName {
			t.Errorf("%s names environment variable %q, which is not %s", skillPath, m, allowlistEnvName)
		}
	}
}

// TestSkillCarriesNoHostFacts is T-06-04's mitigation: SKILL.md may not
// carry an absolute machine-specific filesystem path, mirroring
// server.go's own no-interpolation convention for the instructions
// constant. Reuses hostFactsIn (resources_schema_drift_test.go); its own
// non-vacuity is proven by TestResourceHostFactCheckerIsNotVacuous, not
// duplicated here.
func TestSkillCarriesNoHostFacts(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	if found := hostFactsIn(string(data)); len(found) > 0 {
		t.Errorf("%s carries host-specific path(s) %v — no agent-facing file may interpolate machine-specific filesystem layout", skillPath, found)
	}
}

// TestSkillNamesTheFilterWhenItNamesCompanions applies
// docNamesCompanionsWithoutTheFilter (instructions_contract_test.go) to
// SKILL.md: a document naming filterable companion tools without also
// naming the narrowing filter describes one configuration of the tool
// surface as though it were the only one. Its own non-vacuity is proven by
// TestREADMEGateCheckerIsNotVacuous, not duplicated here.
func TestSkillNamesTheFilterWhenItNamesCompanions(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	if err := docNamesCompanionsWithoutTheFilter(string(data)); err != nil {
		t.Errorf("%s %v", skillPath, err)
	}
}

// TestSkillCarriesExactlyThreeWorkedExamples is SKILL-02's count gate:
// countSkillWorkedExamples's real-file verdict must equal
// skillWorkedExampleCount exactly. Boundary behaviour is what matters here,
// not just the passing case — see TestSkillWorkedExampleCounterIsNotVacuous
// for the synthetic 2/3/4-example discrimination this depends on, and this
// plan's own summary for the verbatim failure text observed at both real
// boundaries (one example deleted, one duplicated).
func TestSkillCarriesExactlyThreeWorkedExamples(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	_, body, err := splitSkillFrontmatter(string(data))
	if err != nil {
		t.Fatalf("splitSkillFrontmatter(%s): %v", skillPath, err)
	}
	if got := countSkillWorkedExamples(body); got != skillWorkedExampleCount {
		t.Errorf("%s carries %d worked example(s), want %d (D-05 in 06-CONTEXT.md locks the count at three)", skillPath, got, skillWorkedExampleCount)
	}
}

// TestSkillStructureCheckersAreNotVacuous proves the three NEW helpers this
// file declares — splitSkillFrontmatter, skillFrontmatterField,
// decisionTablePrecedesSecondSection — and the new resourceURIRe pattern
// each discriminate on synthetic inputs, never touching the real SKILL.md.
// It must pass even while every other TestSkill* test in this file is RED,
// because it operates entirely on inline literals.
func TestSkillStructureCheckersAreNotVacuous(t *testing.T) {
	t.Run("splitSkillFrontmatter", func(t *testing.T) {
		cases := []struct {
			name    string
			doc     string
			wantErr bool
		}{
			{"no fence", "# Just a heading\n\nSome body text.\n", true},
			{"unterminated fence", "---\nname: foo\ndescription: bar\n", true},
			{"well-formed fence", "---\nname: foo\ndescription: bar\n---\nBody here.\n", false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := splitSkillFrontmatter(tc.doc)
				if tc.wantErr && err == nil {
					t.Errorf("splitSkillFrontmatter(%q) = nil error, want an error", tc.doc)
				}
				if !tc.wantErr && err != nil {
					t.Errorf("splitSkillFrontmatter(%q) = %v, want nil error", tc.doc, err)
				}
			})
		}
	})

	t.Run("skillFrontmatterField", func(t *testing.T) {
		cases := []struct {
			name    string
			front   string
			key     string
			wantVal string
			wantOK  bool
		}{
			{"present field", "name: foo\ndescription: bar\n", "name", "foo", true},
			{"absent field", "name: foo\ndescription: bar\n", "missing", "", false},
			{
				"field name is a prefix of another field's name",
				"namespace: unrelated\nname: foo\n",
				"name", "foo", true,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				gotVal, gotOK := skillFrontmatterField(tc.front, tc.key)
				if gotOK != tc.wantOK || gotVal != tc.wantVal {
					t.Errorf("skillFrontmatterField(%q, %q) = %q, %v, want %q, %v", tc.front, tc.key, gotVal, gotOK, tc.wantVal, tc.wantOK)
				}
			})
		}
	})

	t.Run("decisionTablePrecedesSecondSection", func(t *testing.T) {
		cases := []struct {
			name    string
			body    string
			wantErr bool
		}{
			{
				"table before second heading",
				"## First\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n## Second\n\nMore text.\n",
				false,
			},
			{
				"table after second heading",
				"## First\n\nSome text.\n\n## Second\n\n| A | B |\n|---|---|\n| 1 | 2 |\n",
				true,
			},
			{
				"only one heading",
				"## First\n\n| A | B |\n|---|---|\n| 1 | 2 |\n",
				true,
			},
			{
				"no table",
				"## First\n\nSome text.\n\n## Second\n\nMore text.\n",
				true,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := decisionTablePrecedesSecondSection(tc.body)
				if tc.wantErr && err == nil {
					t.Errorf("decisionTablePrecedesSecondSection(%q) = nil, want an error", tc.body)
				}
				if !tc.wantErr && err != nil {
					t.Errorf("decisionTablePrecedesSecondSection(%q) = %v, want nil", tc.body, err)
				}
			})
		}
	})

	t.Run("resourceURIRe", func(t *testing.T) {
		cases := []struct {
			name string
			text string
			want string
		}{
			{"real URI", `read codegraph://tools/explore for details`, `codegraph://tools/explore`},
			{"URI with trailing punctuation", `see codegraph://tools/explore.`, `codegraph://tools/explore`},
			{"bare word", `codegraph is a tool`, ""},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				matches := resourceURIRe.FindAllString(tc.text, -1)
				if tc.want == "" {
					if len(matches) != 0 {
						t.Errorf("resourceURIRe.FindAllString(%q) = %v, want none", tc.text, matches)
					}
					return
				}
				if len(matches) != 1 || matches[0] != tc.want {
					t.Errorf("resourceURIRe.FindAllString(%q) = %v, want [%q]", tc.text, matches, tc.want)
				}
			})
		}
	})
}

// TestSkillWorkedExampleCounterIsNotVacuous proves countSkillWorkedExamples
// discriminates on synthetic inputs, never touching the real SKILL.md: the
// 0/1/2/3/4-heading counts, a heading placed outside the worked-examples
// section (must not be counted), and a level-4 heading (must not be
// mistaken for a level-3 one).
func TestSkillWorkedExampleCounterIsNotVacuous(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"zero examples", "## Worked examples\n\nNo sub-headings here.\n\n## Full reference\n", 0},
		{"one example", "## Worked examples\n\n### First\n\nBody.\n\n## Full reference\n", 1},
		{"two examples", "## Worked examples\n\n### First\n\n### Second\n\n## Full reference\n", 2},
		{"three examples", "## Worked examples\n\n### First\n\n### Second\n\n### Third\n\n## Full reference\n", 3},
		{"four examples", "## Worked examples\n\n### First\n\n### Second\n\n### Third\n\n### Fourth\n\n## Full reference\n", 4},
		{
			"heading outside the worked-examples section is not counted",
			"## Before\n\n### Not an example\n\n## Worked examples\n\n### First\n\n## Full reference\n",
			1,
		},
		{
			"a #### heading is not counted as ###",
			"## Worked examples\n\n### First\n\n#### Not a worked example\n\n## Full reference\n",
			1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countSkillWorkedExamples(tc.body)
			if got != tc.want {
				t.Errorf("countSkillWorkedExamples(%q) = %d, want %d", tc.body, got, tc.want)
			}
		})
	}
}

// TestNudgeTextNamesOnlyRealTools is GUARD-01's tool-name half for the
// nudge script: every codegraph_<name> token the emitted text carries must
// resolve against allToolNames(), same as TestSkillNamesOnlyRealTools does
// for SKILL.md. Additionally asserts docNamesCompanionsWithoutTheFilter
// returns nil for the script's text — the nudge names only
// codegraph_explore, the one tool no filter can remove, so it makes no
// claim CODEGRAPH_MCP_TOOLS could falsify.
func TestNudgeTextNamesOnlyRealTools(t *testing.T) {
	data, err := os.ReadFile(nudgeScriptPath)
	if err != nil {
		t.Fatalf("read %s: %v", nudgeScriptPath, err)
	}
	doc := string(data)

	known := make(map[string]bool, len(allToolNames()))
	for _, name := range allToolNames() {
		known[name] = true
	}

	matches := toolNameTokenRe.FindAllString(doc, -1)
	if len(matches) == 0 {
		t.Fatal("found zero codegraph_<name> tokens in the nudge script — this test would verify nothing")
	}
	for _, m := range matches {
		if !known[m] {
			t.Errorf("%s names %s, which is not a member of allToolNames() — a renamed or removed tool left behind in the nudge text", nudgeScriptPath, m)
		}
	}

	if err := docNamesCompanionsWithoutTheFilter(doc); err != nil {
		t.Errorf("%s %v", nudgeScriptPath, err)
	}
}

// TestNudgeTextCarriesNoUnpinnedFacts is GUARD-01's remaining half for the
// nudge script: no host-specific path, no default-or-max numeric claim, no
// count claim, and the only CODEGRAPH_-prefixed token permitted (if any) is
// allowlistEnvName.
func TestNudgeTextCarriesNoUnpinnedFacts(t *testing.T) {
	data, err := os.ReadFile(nudgeScriptPath)
	if err != nil {
		t.Fatalf("read %s: %v", nudgeScriptPath, err)
	}
	doc := string(data)

	if found := hostFactsIn(doc); len(found) > 0 {
		t.Errorf("%s carries host-specific path(s) %v — no agent-facing file may interpolate machine-specific filesystem layout", nudgeScriptPath, found)
	}
	if claims := numericClaimsMultiset(doc); len(claims) != 0 {
		t.Errorf("%s states numeric claim(s) %v — no default or maximum belongs in the nudge text", nudgeScriptPath, claims)
	}
	if claims := countClaimsIn(doc); len(claims) != 0 {
		t.Errorf("%s states count claim(s) %v — no tool/companion count belongs in the nudge text", nudgeScriptPath, claims)
	}
	for _, m := range envVarTokenRe.FindAllString(doc, -1) {
		if m != allowlistEnvName {
			t.Errorf("%s names environment variable %q, which is not %s", nudgeScriptPath, m, allowlistEnvName)
		}
	}
}
