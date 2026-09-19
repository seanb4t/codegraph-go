package cli

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// wantConfigStyleLine is TestInstallPrintConfigStyle's TEST-LOCAL oracle:
// it builds one target's expected --print-config-style line directly from
// t.Capabilities(), independently of configStyleFields/printConfigStyle
// (the code under test) — the same D-00 discipline every other guard in
// this phase follows.
func wantConfigStyleLine(t agents.AgentTarget, loc agents.Location) string {
	caps := t.Capabilities()

	var scopes []string
	for _, l := range []agents.Location{agents.LocationGlobal, agents.LocationLocal} {
		if caps.Supports(l) {
			scopes = append(scopes, string(l))
		}
	}
	scopesCSV := strings.Join(scopes, ",")

	if !caps.Supports(loc) {
		return fmt.Sprintf("%s: scopes=%s (%s not supported)", t.ID(), scopesCSV, loc)
	}

	mcp, err := caps.MCPConfig(loc)
	if err != nil {
		return fmt.Sprintf("<wantConfigStyleLine: resolve mcp for %s: %v>", t.ID(), err)
	}
	instr, err := caps.InstructionsPath(loc)
	if err != nil {
		return fmt.Sprintf("<wantConfigStyleLine: resolve instructions for %s: %v>", t.ID(), err)
	}
	skill, err := caps.WrittenSkillDir(loc)
	if err != nil {
		return fmt.Sprintf("<wantConfigStyleLine: resolve skill for %s: %v>", t.ID(), err)
	}

	instrDisplay := "none"
	if instr != "" {
		instrDisplay = instr
	}
	skillDisplay := "none"
	if skill != "" {
		skillDisplay = skill
	}

	return fmt.Sprintf("%s: scopes=%s mcp=%s format=%s instructions=%s skill=%s hooks=%s",
		t.ID(), scopesCSV, mcp, caps.ConfigFormat, instrDisplay, skillDisplay, caps.Hooks)
}

// splitNonEmptyLines splits s on "\n" and drops the trailing empty segment
// a "\n"-terminated string always produces — never drops a genuinely empty
// line in the MIDDLE of s.
func splitNonEmptyLines(s string) []string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// TestInstallPrintConfigStyle asserts `install --print-config-style`
// prints exactly one line per registered target, in agents.AllTargetIDs()
// order, each equal to wantConfigStyleLine's independent oracle — and
// pins the Claude line literally (D-04).
func TestInstallPrintConfigStyle(t *testing.T) {
	home := fakeHome(t)

	out, _, err := execCmd("install", "--print-config-style", "--location", "global")
	if err != nil {
		t.Fatalf("install --print-config-style: %v", err)
	}

	lines := splitNonEmptyLines(out)
	ids := agents.AllTargetIDs()
	if len(lines) != len(ids) {
		t.Fatalf("stdout has %d lines, want %d (one per registered target):\n%s", len(lines), len(ids), out)
	}

	for i, id := range ids {
		target, ok := agents.GetTarget(id)
		if !ok {
			t.Fatalf("GetTarget(%s): not registered", id)
		}
		want := wantConfigStyleLine(target, agents.LocationGlobal)
		if lines[i] != want {
			t.Errorf("line %d = %q, want %q", i, lines[i], want)
		}
	}

	wantClaudeLine := fmt.Sprintf(
		"claude: scopes=global,local mcp=%s format=json instructions=%s skill=%s hooks=claude-json",
		filepath.Join(home, ".claude.json"),
		filepath.Join(home, ".claude", "CLAUDE.md"),
		filepath.Join(home, ".claude", "skills", "codegraph"),
	)
	var gotClaudeLine string
	for _, l := range lines {
		if strings.HasPrefix(l, "claude: ") {
			gotClaudeLine = l
		}
	}
	if gotClaudeLine != wantClaudeLine {
		t.Fatalf("claude line = %q, want %q", gotClaudeLine, wantClaudeLine)
	}
}

// dirEntryCount recursively counts every filesystem entry under root,
// returning 0 for a root that doesn't exist at all.
func dirEntryCount(t *testing.T, root string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root {
			count++
		}
		return nil
	})
	if err != nil {
		return 0
	}
	return count
}

// TestInstallPrintConfigStyle_ReadOnly asserts `install --print-config-style`
// opens no picker and writes nothing — with or without --yes — even under
// a forced-interactive TTY (D-04).
func TestInstallPrintConfigStyle_ReadOnly(t *testing.T) {
	home := fakeHome(t)
	project, err := absCwd(t)
	if err != nil {
		t.Fatalf("resolve project cwd: %v", err)
	}

	withStubbedPicker(t, func(*cobra.Command, agents.Location) ([]agents.AgentTarget, error) {
		t.Fatal("runAgentPicker must never be called for --print-config-style")
		return nil, nil
	})

	if _, _, err := execCmd("install", "--print-config-style"); err != nil {
		t.Fatalf("install --print-config-style: %v", err)
	}
	if _, _, err := execCmd("install", "--print-config-style", "--yes"); err != nil {
		t.Fatalf("install --print-config-style --yes: %v", err)
	}

	if n := dirEntryCount(t, home); n != 0 {
		t.Fatalf("fake home has %d entries after --print-config-style, want 0", n)
	}
	if n := dirEntryCount(t, project); n != 0 {
		t.Fatalf("project dir has %d entries after --print-config-style, want 0", n)
	}
}

// absCwd resolves the current working directory as an absolute path,
// failing the test on error.
func absCwd(t *testing.T) (string, error) {
	t.Helper()
	return filepath.Abs(".")
}

// TestInstallPrintConfigStyle_StyledStripsToPlain mirrors
// TestInstall_StyledOutputStripsToPlain (install_test.go): `--color=always`
// output contains an ESC byte and, SGR-stripped, equals the
// `--color=never` output byte for byte (D-04).
func TestInstallPrintConfigStyle_StyledStripsToPlain(t *testing.T) {
	plainHome := fakeHome(t)
	plain, _, err := execCmd("install", "--print-config-style", "--color=never")
	if err != nil {
		t.Fatalf("install --print-config-style --color=never: %v", err)
	}
	plainNorm := strings.ReplaceAll(plain, plainHome, "<HOME>")

	styledHome := fakeHome(t)
	styled, _, err := execCmd("install", "--print-config-style", "--color=always")
	if err != nil {
		t.Fatalf("install --print-config-style --color=always: %v", err)
	}
	if !strings.Contains(styled, "\x1b[") {
		t.Fatalf("expected styled output to contain an ESC byte, got:\n%q", styled)
	}
	styledNorm := strings.ReplaceAll(stripInstallSGR(styled), styledHome, "<HOME>")
	if styledNorm != plainNorm {
		t.Fatalf("stripped+normalized styled output does not equal plain:\nplain:  %q\nstyled: %q", plainNorm, styledNorm)
	}
}

// TestInstallPrintConfigStyle_Filters asserts --print-config-style honours
// -t/--target and -l/--location exactly like the rest of install: a
// two-id CSV at local prints exactly those two lines (one of them
// "not supported" for a global-only target); "none" prints "no agents
// selected"; an unknown id errors before any target line prints (D-04).
func TestInstallPrintConfigStyle_Filters(t *testing.T) {
	fakeHome(t)

	out, _, err := execCmd("install", "--print-config-style", "-t", "claude,hermes", "-l", "local")
	if err != nil {
		t.Fatalf("install --print-config-style -t claude,hermes -l local: %v", err)
	}
	lines := splitNonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("expected exactly 2 lines, got %d:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "claude: scopes=") {
		t.Errorf("line 0 = %q, want a claude line", lines[0])
	}
	if lines[1] != "hermes: scopes=global (local not supported)" {
		t.Errorf("line 1 = %q, want %q", lines[1], "hermes: scopes=global (local not supported)")
	}

	out, _, err = execCmd("install", "--print-config-style", "-t", "none")
	if err != nil {
		t.Fatalf("install --print-config-style -t none: %v", err)
	}
	if strings.TrimRight(out, "\n") != "no agents selected" {
		t.Fatalf("output = %q, want %q", out, "no agents selected")
	}

	out, _, err = execCmd("install", "--print-config-style", "-t", "bogus")
	if err == nil {
		t.Fatalf("expected an error for an unknown target id, got none (output=%q)", out)
	}
	if !strings.Contains(err.Error(), "unknown agent target") {
		t.Fatalf("error = %v, want it to contain %q", err, "unknown agent target")
	}
	if strings.Contains(out, "scopes=") {
		t.Fatalf("expected no target line to print before the unknown-target error, got:\n%s", out)
	}
}
