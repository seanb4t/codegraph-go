package githooks

import (
	"strings"
	"testing"
)

// TestMarkerBlock asserts markerBlock() produces the verbatim TS
// sync/git-hooks.js block bytes (D-03): the begin marker, the exact
// subshell-backgrounding guard line (Pitfall 5 — never simplify this), and
// the end marker, joined with "\n".
func TestMarkerBlock(t *testing.T) {
	want := strings.Join([]string{
		"# >>> codegraph sync hook >>>",
		"# Keeps the CodeGraph index fresh while the live file watcher is off",
		"# (e.g. WSL2 /mnt drives). Runs in the background so it never blocks git.",
		"# Managed by codegraph; remove with `codegraph uninit` or delete this block.",
		"if command -v codegraph >/dev/null 2>&1; then",
		"  ( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1",
		"fi",
		"# <<< codegraph sync hook <<<",
	}, "\n")

	got := markerBlock()
	if got != want {
		t.Fatalf("markerBlock() = %q, want %q", got, want)
	}
	if !strings.Contains(got, markerBegin) {
		t.Errorf("markerBlock() missing begin marker %q", markerBegin)
	}
	if !strings.Contains(got, "if command -v codegraph >/dev/null 2>&1; then") {
		t.Errorf("markerBlock() missing command -v guard line")
	}
	if !strings.Contains(got, "( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1") {
		t.Errorf("markerBlock() missing exact subshell-backgrounding snippet")
	}
	if !strings.Contains(got, markerEnd) {
		t.Errorf("markerBlock() missing end marker %q", markerEnd)
	}
}

func TestStripMarkerBlock_IndentedMarkerStripped(t *testing.T) {
	content := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		"  " + markerBegin, // indented — trimmed-line matching must still strip it
		"some block content",
		"  " + markerEnd,
		"echo after",
	}, "\n")

	got := stripMarkerBlock(content)
	want := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		"echo after",
	}, "\n")
	if got != want {
		t.Fatalf("stripMarkerBlock(indented) = %q, want %q", got, want)
	}
}

func TestStripMarkerBlock_NoMarkerPassthrough(t *testing.T) {
	content := "#!/bin/sh\necho hi\n"
	got := stripMarkerBlock(content)
	if got != content {
		t.Fatalf("stripMarkerBlock(no marker) = %q, want unchanged %q", got, content)
	}
}

func TestStripMarkerBlock_PreservesSurroundingContent(t *testing.T) {
	content := "#!/bin/sh\n\necho before\n\n" + markerBlock() + "\n\necho after\n"
	got := stripMarkerBlock(content)
	if strings.Contains(got, markerBegin) || strings.Contains(got, markerEnd) {
		t.Fatalf("stripMarkerBlock left markers in output: %q", got)
	}
	if !strings.Contains(got, "echo before") || !strings.Contains(got, "echo after") {
		t.Fatalf("stripMarkerBlock dropped surrounding content: %q", got)
	}
}

func TestIsEffectivelyEmpty_BlankAndShebangOnly(t *testing.T) {
	cases := []string{
		"",
		"\n\n",
		"#!/bin/sh\n",
		"#!/bin/sh\n\n\n",
		"  #!/bin/sh  \n",
	}
	for _, c := range cases {
		if !isEffectivelyEmpty(c) {
			t.Errorf("isEffectivelyEmpty(%q) = false, want true", c)
		}
	}
}

func TestIsEffectivelyEmpty_RealContentIsNotEmpty(t *testing.T) {
	cases := []string{
		"#!/bin/sh\necho hi\n",
		"echo hi\n",
		"#!/bin/sh\n  echo hi  \n",
	}
	for _, c := range cases {
		if isEffectivelyEmpty(c) {
			t.Errorf("isEffectivelyEmpty(%q) = true, want false", c)
		}
	}
}
