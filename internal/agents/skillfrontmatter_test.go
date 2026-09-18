package agents

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// skillNamePattern is the name-must-match-folder rule Cursor, opencode and
// Kiro each independently enforce in their own skill-loading docs
// (RESEARCH.md Pitfall 4, all fetched 2026-09-18):
//   - https://cursor.com/docs/skills.md
//   - https://opencode.ai/docs/skills.md
//   - https://kiro.dev/docs/skills/
//
// All three require a skill's frontmatter `name` to be lowercase,
// hyphen-separated, 1-64 characters, and equal to the base name of the
// directory it ships in.
var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// parseSkillFrontmatter extracts name/description from a SKILL.md's leading
// "---" frontmatter block using the standard library only (line split +
// "key: value" cut) — no YAML dependency for a two-field read.
func parseSkillFrontmatter(t *testing.T, content []byte) (name, description string) {
	t.Helper()
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		t.Fatalf("SKILL.md does not start with a --- frontmatter block")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		t.Fatalf("SKILL.md frontmatter block has no closing ---")
	}
	for _, line := range lines[1:end] {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			name = strings.TrimSpace(value)
		case "description":
			description = strings.TrimSpace(value)
		}
	}
	return name, description
}

// TestSkillFrontmatterMatchesEveryWrittenDir (AGENT-06, D-00): the shipped
// SKILL.md is valid for every harness that validates a skill's frontmatter
// against its own folder — this asserts OUR bytes only; whether a harness
// actually LOADS the skill is 05-06's live evidence, never a go test (D-00).
func TestSkillFrontmatterMatchesEveryWrittenDir(t *testing.T) {
	content, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	name, description := parseSkillFrontmatter(t, content)

	if name != "codegraph" {
		t.Fatalf("frontmatter name = %q, want %q", name, "codegraph")
	}
	if !skillNamePattern.MatchString(name) {
		t.Fatalf("frontmatter name %q does not match %s", name, skillNamePattern.String())
	}
	if len(name) < 1 || len(name) > 64 {
		t.Fatalf("frontmatter name length = %d, want 1-64", len(name))
	}
	if description == "" {
		t.Fatalf("frontmatter description is empty")
	}
	if len(description) > 1024 {
		t.Fatalf("frontmatter description length = %d, want <= 1024", len(description))
	}

	fakeHome(t)
	checked := 0
	for _, target := range AllTargets() {
		for _, loc := range []Location{LocationGlobal, LocationLocal} {
			if !target.SupportsLocation(loc) {
				continue
			}
			dir, err := target.Capabilities().WrittenSkillDir(loc)
			if err != nil {
				t.Fatalf("%s WrittenSkillDir(%s): %v", target.ID(), loc, err)
			}
			if dir == "" {
				continue
			}
			checked++
			if got := filepath.Base(dir); got != name {
				t.Fatalf("%s/%s written skill dir base name = %q, want %q (frontmatter name must match its folder)", target.ID(), loc, got, name)
			}
		}
	}
	if checked < 6 {
		t.Fatalf("checked %d written skill dirs, want at least 6 (claude, cursor, opencode x 2 scopes) — the floor rises as 05-05 wires more", checked)
	}
}
