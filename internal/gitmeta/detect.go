package gitmeta

import "context"

// Mismatch describes a detected "borrowed index" situation: startPath lives
// in one git working tree, but the resolved CodeGraph index belongs to a
// different one. The json tags match TS's `--json` object shape
// (`{worktreeRoot, indexRoot}`) so plan 02-04 can embed this directly into
// StatusResult.
type Mismatch struct {
	WorktreeRoot string `json:"worktreeRoot"`
	IndexRoot    string `json:"indexRoot"`
}

// DetectIndexMismatch detects when startPath lives in one git working tree
// but the resolved CodeGraph index (indexRoot) belongs to a DIFFERENT
// working tree — the silent "worktree queries the main branch's graph"
// correctness bug (WORK-01). Ported verbatim from TS sync/worktree.js's
// detectWorktreeIndexMismatch (D-02), gate order and polarity preserved
// exactly.
//
// Worst case this spawns four git subprocesses (two WorktreeRoot, two
// CommonDir) — gates 1-3 each short-circuit before reaching CommonDir,
// so most calls spawn one or two. This per-call cost is what motivates
// CachingDetector (Task 3): a long-lived MCP server must not re-pay it on
// every tool call.
//
// Returns nil ("nothing to warn about") — never an error, never a panic —
// on every degradation path, including git being absent, slow, or the path
// not being a git repo at all (WORK-03).
func DetectIndexMismatch(ctx context.Context, startPath, indexRoot string) *Mismatch {
	// Gate 1: startPath must itself be inside a git working tree. Not a
	// repo (or git unavailable) ⇒ no mismatch — there's nothing to compare.
	worktreeRoot := WorktreeRoot(ctx, startPath)
	if worktreeRoot == "" {
		return nil
	}

	// Gate 2: if the index already lives in startPath's own working tree,
	// there's no borrowing going on.
	resolvedIndexRoot := realpath(indexRoot)
	if worktreeRoot == resolvedIndexRoot {
		return nil
	}

	// Gate 3: the index root must ITSELF be a working-tree root. This is
	// what kills monorepo-subdir, plain-ancestor, and non-git false
	// positives — an index that merely sits in some parent directory (not a
	// git root at all) is not a "different working tree", it's just an
	// ancestor directory that happens to contain a .codegraph/.
	if WorktreeRoot(ctx, resolvedIndexRoot) != resolvedIndexRoot {
		return nil
	}

	// Gate 4 — ★ COUNTERINTUITIVE POLARITY, read carefully before touching:
	// if BOTH common dirs are non-empty AND they DIFFER, this is NOT a
	// mismatch — it is SUPPRESSED.
	//
	// A submodule or an embedded/nested clone is a DIFFERENT repository from
	// startPath's, one whose files the parent index ALREADY covers because
	// indexing a super-repo descends into its submodules and gitlinked
	// clones. The warning's entire premise — "results reflect a different
	// branch of the SAME repo; symbols changed only here are missing" — is
	// false in that case, and its "run codegraph init -i" advice would
	// needlessly fragment a unified workspace index. A submodule/embedded
	// clone reports its OWN git common dir (e.g. `.git/modules/<name>`, or
	// its own standalone `.git`), which DIFFERS from the parent's.
	//
	// A GENUINE borrowed worktree, by contrast, is the SAME repository as
	// the index root: linked worktrees of one repo SHARE a single common
	// dir. So: shared common dir ⇒ real mismatch, proceed to report it.
	// Differing common dir ⇒ a different repo the index already covers,
	// suppress. This is the inverse of what a naive reading of "detect a
	// mismatched worktree" suggests. Do NOT invert this in a future
	// "simplification" — see the fixture comments on newSubmoduleFixture /
	// newNestedCloneFixture in detect_test.go's sibling fixtures_test.go,
	// and CONTEXT.md D-02 / TS issues #1031, #1033.
	// ★ WR-03, deliberate D-02 divergence from TS: TS's own gate 4
	// (`if (worktreeCommon && indexCommon && worktreeCommon !== indexCommon)
	// return null;`) falls THROUGH to reporting a mismatch whenever either
	// CommonDir call fails (empty string) — CommonDir collapses every
	// failure (timeout, transient fork failure, safe.directory rejection)
	// into "". Gates 1-3 already proved git works in both directories, so
	// this is a narrow window, but it is a REAL fail-open: a degraded git
	// on this one call would silently defeat the submodule/embedded-clone
	// suppression and produce exactly the false-positive "nags users
	// constantly" failure mode worktree.go's own doc comment (D-02) commits
	// to never causing ("degrades to a safe zero value on ANY failure…
	// report 'no signal' rather than an error"). Go therefore intentionally
	// diverges here: an unavailable common dir on EITHER side degrades to
	// "no signal" (no mismatch), matching gates 1-3's own philosophy,
	// rather than replicating TS's narrow fail-open bug. Only a SHARED,
	// successfully-resolved common dir is treated as positive evidence of
	// a genuine borrowed worktree.
	worktreeCommon := CommonDir(ctx, worktreeRoot)
	indexCommon := CommonDir(ctx, resolvedIndexRoot)
	if worktreeCommon == "" || indexCommon == "" || worktreeCommon != indexCommon {
		return nil
	}

	return &Mismatch{WorktreeRoot: worktreeRoot, IndexRoot: resolvedIndexRoot}
}
