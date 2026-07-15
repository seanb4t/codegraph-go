package gitmeta

import "testing"

func TestMismatchNilReceiverSafety(t *testing.T) {
	if got := (*Mismatch)(nil).Notice(); got != "" {
		t.Fatalf("Notice() on nil receiver = %q, want \"\"", got)
	}
	if got := (*Mismatch)(nil).Warning(); got != "" {
		t.Fatalf("Warning() on nil receiver = %q, want \"\"", got)
	}
}

func TestMismatchWarningVerbatim(t *testing.T) {
	m := &Mismatch{WorktreeRoot: "/w", IndexRoot: "/i"}
	want := "This CodeGraph index belongs to a different git working tree.\n" +
		"  Running in: /w\n" +
		"  Index from: /i\n" +
		"Results reflect that tree's code (often a different branch), not this worktree — " +
		"symbols changed only here are missing. Run \"codegraph init -i\" in this worktree " +
		"for a worktree-local index."
	if got := m.Warning(); got != want {
		t.Fatalf("Warning() =\n%q\nwant\n%q", got, want)
	}
}

func TestMismatchNoticeVerbatim(t *testing.T) {
	m := &Mismatch{WorktreeRoot: "/w", IndexRoot: "/i"}
	want := "⚠ CodeGraph results below come from a different git worktree (/i), " +
		"not where you're working (/w) — they may reflect another branch, " +
		"and symbols changed only here are missing. Run \"codegraph init -i\" here for a " +
		"worktree-local index."
	got := m.Notice()
	if got != want {
		t.Fatalf("Notice() =\n%q\nwant\n%q", got, want)
	}

	b := []byte(got)
	if len(b) < 4 || b[0] != 0xe2 || b[1] != 0x9a || b[2] != 0xa0 {
		t.Fatalf("Notice() leading bytes = % x, want e2 9a a0", b[:min(4, len(b))])
	}
	if b[3] == 0xef {
		t.Fatalf("Notice() 4th byte = %x, looks like the start of a U+FE0F variation selector (ef b8 8f)", b[3])
	}
}
