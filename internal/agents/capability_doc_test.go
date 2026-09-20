// Package agents (this file): the AGENT-14 drift guard for
// docs/AGENT-CAPABILITIES.md (07-10-PLAN.md, D-27/D-28). Mirrors
// internal/indexer/capability/matrix_test.go's TestMatrix_DocMirrorsDescriptor
// shape, adjusted for this package: every code-derived cell in the
// published table must equal a value computed straight from
// AgentTarget.Capabilities() for all 8 targets x 2 scopes, and the
// hand-kept verification column must carry one of exactly two forms
// (never a silent blank, never an overclaim).
package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// capabilityDocRepoRoot locates the repository root from THIS SOURCE
// FILE's own package directory (internal/agents), two levels up — the
// same pattern internal/indexer/capability/matrix_test.go's repoRoot
// uses, adjusted for this package's shallower nesting (07-10-PLAN.md
// Notes). Resolved via runtime.Caller rather than a cwd-relative
// filepath.Abs(".."), because TestCapabilityDoc_MirrorsCapabilities
// t.Chdir()s into a scratch temp dir before calling this — a cwd-relative
// "../.." would silently resolve inside the temp dir instead of the repo
// (caught live: the RED run failed with "resolved repo root
// .../T does not contain go.mod" before this fix).
func capabilityDocRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repo root: runtime.Caller(0) failed")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolved repo root %q does not contain go.mod: %v", root, err)
	}
	return root
}

// capabilityDocIDCell matches the doc's first cell, e.g. "`codex`".
var capabilityDocIDCell = regexp.MustCompile("^`[a-z]+`$")

// parseCapabilityDocRows collects every capability-table data row from doc:
// a line starting with "| `" (skipping the header and separator rows,
// neither of which has a backtick immediately after the leading pipe),
// split on "|" into exactly 9 trimmed cells (07-10-PLAN.md's own row-shape
// choice: id, scope, mcp, format, instructions, skill, hooks, nudge,
// verification).
func parseCapabilityDocRows(t *testing.T, doc string) [][]string {
	t.Helper()
	var rows [][]string
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "| `") {
			continue
		}
		parts := strings.Split(trimmed, "|")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
			parts = parts[1:]
		}
		if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
			parts = parts[:len(parts)-1]
		}
		if len(parts) != 9 {
			continue
		}
		cells := make([]string, len(parts))
		for i, p := range parts {
			cells[i] = strings.TrimSpace(p)
		}
		if !capabilityDocIDCell.MatchString(cells[0]) {
			continue
		}
		rows = append(rows, cells)
	}
	return rows
}

// expectedCapabilityRow is one row's 7 code-derived cells, computed
// straight from Capabilities() — never a second hand-copied path.
type expectedCapabilityRow struct {
	id, scope, mcp, format, instructions, skill, hooks, nudge string
}

// renderCapabilityDocPath renders p the way docs/AGENT-CAPABILITIES.md
// itself renders a resolved path: "none" for an undeclared (empty) path;
// a fake-HOME-relative path rendered as `~/...`; anything else (a local,
// already-relative path) rendered as-is — both backticked. This mirrors
// 07-10-PLAN.md's interfaces note: "paths under the fake HOME rendered as
// ~/…, local paths relative."
func renderCapabilityDocPath(home, p string) string {
	if p == "" {
		return "none"
	}
	rendered := filepath.ToSlash(p)
	if home != "" {
		homeSlash := filepath.ToSlash(home)
		if strings.HasPrefix(rendered, homeSlash) {
			rendered = "~" + strings.TrimPrefix(rendered, homeSlash)
		}
	}
	return "`" + rendered + "`"
}

// expectedNudgeColumn derives the nudge column from Hooks alone (D-28: no
// new Capabilities field, no new --print-config-style field).
func expectedNudgeColumn(hooks HookMechanism) string {
	switch hooks {
	case HooksClaudeJSON:
		return "SessionStart + opt-in PreToolUse"
	case HooksCodexJSON:
		return "opt-in PreToolUse"
	default:
		return "none"
	}
}

// buildExpectedCapabilityRows computes every expected row, in AllTargets()
// x [global, local] order — the exact order the drift test requires the
// doc to follow.
func buildExpectedCapabilityRows(t *testing.T, home string) []expectedCapabilityRow {
	t.Helper()
	var rows []expectedCapabilityRow
	for _, target := range AllTargets() {
		caps := target.Capabilities()
		for _, loc := range []Location{LocationGlobal, LocationLocal} {
			row := expectedCapabilityRow{id: string(target.ID()), scope: string(loc)}
			if !caps.Supports(loc) {
				row.mcp = "not supported"
				row.format = "not supported"
				row.instructions = "not supported"
				row.skill = "not supported"
				row.hooks = "not supported"
				row.nudge = "not supported"
				rows = append(rows, row)
				continue
			}

			if caps.MCPConfig == nil {
				t.Fatalf("%s/%s: Supports(%s)=true but MCPConfig is nil", target.ID(), loc, loc)
			}
			mcpPath, err := caps.MCPConfig(loc)
			if err != nil {
				t.Fatalf("%s/%s: resolve MCP path: %v", target.ID(), loc, err)
			}
			row.mcp = renderCapabilityDocPath(home, mcpPath)
			row.format = string(caps.ConfigFormat)

			instrPath, err := caps.InstructionsPath(loc)
			if err != nil {
				t.Fatalf("%s/%s: resolve instructions path: %v", target.ID(), loc, err)
			}
			row.instructions = renderCapabilityDocPath(home, instrPath)

			skillPath, err := caps.WrittenSkillDir(loc)
			if err != nil {
				t.Fatalf("%s/%s: resolve skill path: %v", target.ID(), loc, err)
			}
			row.skill = renderCapabilityDocPath(home, skillPath)

			row.hooks = string(caps.Hooks)
			row.nudge = expectedNudgeColumn(caps.Hooks)
			rows = append(rows, row)
		}
	}
	return rows
}

// TestCapabilityDoc_MirrorsCapabilities proves docs/AGENT-CAPABILITIES.md
// — the human-readable half of AGENT-14 — carries the EXACT same
// code-derived cells (scope, mcp, format, instructions, skill, hooks,
// nudge) as Capabilities() for every one of the 8 registered targets at
// both scopes, in AllTargets() x [global, local] order. A missing, extra,
// reordered, or mismatching row/cell fails this test — the doc cannot
// silently drift from the code (D-27).
func TestCapabilityDoc_MirrorsCapabilities(t *testing.T) {
	home := fakeHome(t)
	t.Chdir(t.TempDir())

	root := capabilityDocRepoRoot(t)
	docPath := filepath.Join(root, "docs", "AGENT-CAPABILITIES.md")
	raw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}

	rows := parseCapabilityDocRows(t, string(raw))
	expected := buildExpectedCapabilityRows(t, home)

	if len(expected) != 16 {
		t.Fatalf("internal error: expected 16 computed rows from AllTargets() x [global,local], got %d", len(expected))
	}
	if len(rows) != 16 {
		t.Fatalf("docs/AGENT-CAPABILITIES.md: expected exactly 16 capability rows (8 targets x 2 scopes), got %d", len(rows))
	}

	for i, exp := range expected {
		got := rows[i]
		label := fmt.Sprintf("%s/%s (row %d)", exp.id, exp.scope, i)

		wantID := "`" + exp.id + "`"
		if got[0] != wantID {
			t.Errorf("%s: id column: doc=%s want=%s (reordered or missing row?)", label, got[0], wantID)
		}
		if got[1] != exp.scope {
			t.Errorf("%s: scope column: doc=%q want=%q (reordered row?)", label, got[1], exp.scope)
		}
		if got[2] != exp.mcp {
			t.Errorf("%s: mcp column: doc=%q want=%q", label, got[2], exp.mcp)
		}
		if got[3] != exp.format {
			t.Errorf("%s: format column: doc=%q want=%q", label, got[3], exp.format)
		}
		if got[4] != exp.instructions {
			t.Errorf("%s: instructions column: doc=%q want=%q", label, got[4], exp.instructions)
		}
		if got[5] != exp.skill {
			t.Errorf("%s: skill column: doc=%q want=%q", label, got[5], exp.skill)
		}
		if got[6] != exp.hooks {
			t.Errorf("%s: hooks column: doc=%q want=%q", label, got[6], exp.hooks)
		}
		if got[7] != exp.nudge {
			t.Errorf("%s: nudge column: doc=%q want=%q", label, got[7], exp.nudge)
		}
	}
}

// capabilityDocVerifiedPattern/capabilityDocAssumedPattern are the two
// allowed forms of the hand-kept verification cell (D-27): a live-session
// evidence file, or a documented-but-unverified assumption naming its
// source and fetch date. Anything else — including a blank cell — fails.
var (
	capabilityDocVerifiedPattern = regexp.MustCompile(`^verified \d{4}-\d{2}-\d{2} \(.+\)$`)
	capabilityDocAssumedPattern  = regexp.MustCompile(`^\[ASSUMED\] \(.+, fetched \d{4}-\d{2}-\d{2}\)$`)
)

// TestCapabilityDoc_VerificationColumn proves the hand-kept verification
// column never overclaims: every supported row reads `verified <date>
// (<evidence file>)` or `[ASSUMED] (<source>, fetched <date>)`; every
// unsupported row reads `n/a`; Cursor, Gemini CLI and Kiro stay
// `[ASSUMED]` at both scopes (no live session ran for them); both Codex
// rows read `verified` via 07-LIVE-SESSIONS.md; and the doc never advises
// bypassing hook trust (D-10, D-19, D-27).
func TestCapabilityDoc_VerificationColumn(t *testing.T) {
	root := capabilityDocRepoRoot(t)
	docPath := filepath.Join(root, "docs", "AGENT-CAPABILITIES.md")
	raw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	doc := string(raw)

	rows := parseCapabilityDocRows(t, doc)
	if len(rows) != 16 {
		t.Fatalf("docs/AGENT-CAPABILITIES.md: expected exactly 16 capability rows, got %d", len(rows))
	}

	assumedCount := map[string]int{}
	for _, row := range rows {
		id := strings.Trim(row[0], "`")
		scope := row[1]
		verification := row[8]
		unsupported := row[2] == "not supported"

		switch {
		case unsupported:
			if verification != "n/a" {
				t.Errorf("%s/%s: unsupported scope must read verification=n/a, got %q", id, scope, verification)
			}
		case capabilityDocVerifiedPattern.MatchString(verification):
			if id == "cursor" || id == "gemini" || id == "kiro" {
				t.Errorf("%s/%s: must stay [ASSUMED] (no live session ran for this target), got %q", id, scope, verification)
			}
		case capabilityDocAssumedPattern.MatchString(verification):
			assumedCount[id]++
			if id == "codex" {
				t.Errorf("codex/%s: must be verified via 07-LIVE-SESSIONS.md, got %q", scope, verification)
			}
		default:
			t.Errorf("%s/%s: verification cell %q matches neither the verified nor [ASSUMED] form", id, scope, verification)
		}
	}

	for _, id := range []string{"cursor", "gemini", "kiro"} {
		if assumedCount[id] != 2 {
			t.Errorf("%s: expected [ASSUMED] at both scopes, got %d [ASSUMED] row(s)", id, assumedCount[id])
		}
	}

	for _, row := range rows {
		id := strings.Trim(row[0], "`")
		if id != "codex" {
			continue
		}
		if !capabilityDocVerifiedPattern.MatchString(row[8]) || !strings.Contains(row[8], "07-LIVE-SESSIONS.md") {
			t.Errorf("codex/%s: expected a verified cell naming 07-LIVE-SESSIONS.md, got %q", row[1], row[8])
		}
	}

	if !strings.Contains(doc, "/hooks") {
		t.Error("docs/AGENT-CAPABILITIES.md must mention /hooks (Codex's hook-trust review command)")
	}
	if !strings.Contains(doc, "--pretool-nudge") {
		t.Error("docs/AGENT-CAPABILITIES.md must mention --pretool-nudge (the opt-in flag)")
	}
	// Built by concatenation so this file's own source never contains the
	// literal flag string either (07-10-PLAN.md: "The test builds the
	// trust-bypass flag name by concatenation when it checks the doc for
	// it").
	trustBypassFlag := "--dangerously-" + "bypass-hook-trust"
	if strings.Contains(doc, trustBypassFlag) {
		t.Errorf("docs/AGENT-CAPABILITIES.md must never advise the trust-bypass flag %q (D-10, D-19)", trustBypassFlag)
	}
}
