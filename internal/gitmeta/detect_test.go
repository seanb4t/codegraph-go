package gitmeta

import (
	"context"
	"path/filepath"
	"testing"
)

// fixtureCase pairs a fixture builder with its expected D-15 verdict.
type fixtureCase struct {
	name         string
	build        func(t *testing.T) (startPath, indexRoot string)
	wantMismatch bool
}

// TestFixtureVerdicts is the D-15 verdict matrix: seven real-git layouts,
// each asserting DetectIndexMismatch's non-nil/nil verdict against the
// documented expectation. This test MUST fail to compile until
// DetectIndexMismatch exists (RED state, Task 1).
func TestFixtureVerdicts(t *testing.T) {
	cases := []fixtureCase{
		{"linked-worktree", newLinkedWorktreeFixture, true},
		{"claude-worktrees", newClaudeWorktreeFixture, true},
		// Gate 4 polarity: differing common dirs SUPPRESS the warning — see
		// the explanatory comment on newSubmoduleFixture.
		{"submodule", newSubmoduleFixture, false},
		// Same gate-4 polarity as submodule — see newNestedCloneFixture.
		{"nested-clone", newNestedCloneFixture, false},
		{"monorepo-subdir", newMonorepoSubdirFixture, false},
		{"symlinked", newSymlinkedFixture, false},
		{"non-git", newNonGitFixture, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			startPath, indexRoot := c.build(t)
			got := DetectIndexMismatch(context.Background(), startPath, indexRoot)

			if c.wantMismatch && got == nil {
				t.Fatalf("DetectIndexMismatch(%q, %q) = nil, want mismatch", startPath, indexRoot)
			}
			if !c.wantMismatch && got != nil {
				t.Fatalf("DetectIndexMismatch(%q, %q) = %+v, want no mismatch", startPath, indexRoot, got)
			}
			if !c.wantMismatch {
				return
			}

			// macOS's t.TempDir() returns a /var -> /private/var symlink, so
			// a naive string compare against the raw tmp path fails; resolve
			// the expected values through the same EvalSymlinks path the
			// implementation uses.
			wantWorktree := resolveExpected(t, startPath)
			wantIndex := resolveExpected(t, indexRoot)
			if got.WorktreeRoot != wantWorktree {
				t.Errorf("WorktreeRoot = %q, want %q", got.WorktreeRoot, wantWorktree)
			}
			if got.IndexRoot != wantIndex {
				t.Errorf("IndexRoot = %q, want %q", got.IndexRoot, wantIndex)
			}
		})
	}

	// Plain-ancestor variant of the monorepo-subdir case (D-15): an index
	// root that is a plain non-git ancestor directory (not itself a
	// worktree root) isolates gate 3 specifically, distinct from the table
	// row above where index=repo root is caught by gate 2 first.
	t.Run("monorepo-subdir-plain-ancestor", func(t *testing.T) {
		tmp := t.TempDir()
		repo := initRepo(t, filepath.Join(tmp, "repo"))
		got := DetectIndexMismatch(context.Background(), repo, tmp)
		if got != nil {
			t.Fatalf("DetectIndexMismatch(%q, %q) = %+v, want no mismatch (gate 3: tmp is not a worktree root)", repo, tmp, got)
		}
	})
}

func resolveExpected(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", p, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}
