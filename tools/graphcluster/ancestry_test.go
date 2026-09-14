package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGitH mirrors internal/query/engine_worktree_test.go's runGitW:
// hermetic -c flags, t.Skipf (never t.Fatal) on any git failure —
// including git being absent from PATH — so ancestry proof, which
// depends on real repository history, degrades to skip rather than a
// hard failure on an environment that cannot run git.
func runGitH(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{
		"-c", "init.defaultBranch=main",
		"-c", "user.name=codegraph-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always",
	}
	full := append(append([]string{}, base...), args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("git %v failed (git missing or unsupported here): %v: %s", args, err, string(out))
	}
	return string(out)
}

// repoRoot resolves the checkout's top-level directory so these tests can
// run git commands against repo-relative paths regardless of the test
// binary's own working directory (tools/graphcluster).
func repoRoot(t *testing.T) string {
	t.Helper()
	return strings.TrimSpace(runGitH(t, ".", "rev-parse", "--show-toplevel"))
}

// linesNonEmpty splits git log's newline-delimited output into non-empty
// commit-hash lines.
func linesNonEmpty(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// TestClusterThresholdCommitIsAncestorOfEveryMeasurement (D-05/D-08): the
// commit that ADDED corpora/graph-cluster-threshold.json must be an
// ancestor (or the same commit, after a squash-merge) of every commit
// touching the observation file and the harness — proven by git history,
// not asserted in prose.
//
// mutation: widen max after measurement
func TestClusterThresholdCommitIsAncestorOfEveryMeasurement(t *testing.T) {
	root := repoRoot(t)

	if strings.TrimSpace(runGitH(t, root, "rev-parse", "--is-shallow-repository")) == "true" {
		t.Skip("shallow clone: ancestry cannot be proven without history (CI test job checks out fetch-depth 1); run on a full clone")
	}

	addedLines := linesNonEmpty(runGitH(t, root, "log", "--format=%H", "--diff-filter=A", "--", "corpora/graph-cluster-threshold.json"))
	if len(addedLines) != 1 {
		t.Fatalf("commits that ADDED corpora/graph-cluster-threshold.json = %d, want exactly 1: %v", len(addedLines), addedLines)
	}
	thr := addedLines[0]

	allTouchLines := linesNonEmpty(runGitH(t, root, "log", "--format=%H", "--", "corpora/graph-cluster-threshold.json"))
	if len(allTouchLines) != 1 {
		t.Fatalf("commits touching corpora/graph-cluster-threshold.json = %d, want exactly 1 (never modified after being added): %v", len(allTouchLines), allTouchLines)
	}

	measurements := linesNonEmpty(runGitH(t, root, "log", "--format=%H", "--", "corpora/graph-cluster-observations.json", "tools/graphcluster/main.go"))
	n := len(measurements)
	if n == 0 {
		t.Fatalf("commits touching corpora/graph-cluster-observations.json or tools/graphcluster/main.go = 0, want >= 1")
	}

	for _, c := range measurements {
		// A commit is its own ancestor, so a later squash-merge that folds
		// both the threshold and a measurement into one commit still
		// passes this check — deliberate, not a loophole: it remains true
		// that the threshold's content predates or is simultaneous with
		// the measurement inside that single commit.
		cmd := exec.Command("git", "-c", "protocol.file.allow=always", "merge-base", "--is-ancestor", thr, c)
		cmd.Dir = root
		if err := cmd.Run(); err != nil {
			t.Fatalf("git merge-base --is-ancestor %s %s: not an ancestor (err=%v)", thr, c, err)
		}
	}

	t.Logf("threshold commit %s; compared %d measurement commits", thr, n)
}

// TestClusterThresholdDigestMatchesCommittedObservation (never skips; no
// history needed): the committed observation's thresholdDigest must equal
// the digest of the CURRENTLY committed threshold bytes — a later edit to
// the threshold changes the digest and turns this test RED without any
// new commit (the instrument 11-05 family (b) mutates against).
func TestClusterThresholdDigestMatchesCommittedObservation(t *testing.T) {
	root := repoRoot(t)

	raw, err := os.ReadFile(filepath.Join(root, "corpora", "graph-cluster-threshold.json"))
	if err != nil {
		t.Fatalf("read committed threshold: %v", err)
	}
	digest := thresholdDigest(raw)

	obsRaw, err := os.ReadFile(filepath.Join(root, "corpora", "graph-cluster-observations.json"))
	if err != nil {
		t.Fatalf("read committed observation: %v", err)
	}
	var obs observation
	if err := json.Unmarshal(obsRaw, &obs); err != nil {
		t.Fatalf("decode committed observation: %v", err)
	}

	if obs.ThresholdDigest != digest {
		t.Fatalf("observation.thresholdDigest = %q, want %q (committed threshold digest) — the threshold was edited after measurement, or the observation is stale", obs.ThresholdDigest, digest)
	}
	if obs.Verdict != "PASS" && obs.Verdict != "FAIL" {
		t.Fatalf("observation.verdict = %q, want exactly PASS or FAIL", obs.Verdict)
	}
	if obs.ClusteringTimeMs.Verdict != obs.Verdict {
		t.Fatalf("clusteringTimeMs.verdict = %q, want %q (must equal the top-level verdict)", obs.ClusteringTimeMs.Verdict, obs.Verdict)
	}
	if len(obs.Runs) != 3 {
		t.Fatalf("len(runs) = %d, want 3", len(obs.Runs))
	}
	if obs.StoreSource != "manifest" {
		t.Fatalf("storeSource = %q, want manifest (the committed measurement must come from the resolved corpus, not an explicit test store)", obs.StoreSource)
	}
	if obs.Corpus.SHA != "94f39958baf7ad51ddf9c70e406ed6b188194daa" {
		t.Fatalf("corpus.sha = %q, want the pinned guava sha", obs.Corpus.SHA)
	}
	if bytes.Contains(obsRaw, []byte("/Users/")) || bytes.Contains(obsRaw, []byte("/home/")) {
		t.Fatalf("committed observation leaks a host-local path")
	}

	t.Logf("threshold digest %s matches; verdict %s; median %d ms", digest, obs.Verdict, obs.ClusteringTimeMs.Median)
}
