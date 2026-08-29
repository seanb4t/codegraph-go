package gitmeta

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- RemoteGitHubRepo: transport-shape parsing ---

func TestRemoteGitHubRepo_HTTPSWithGitSuffix(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Fatalf("RemoteGitHubRepo = %+v, want Owner=owner Repo=repo", got)
	}
	if got.Reason != "" {
		t.Fatalf("RemoteGitHubRepo.Reason = %q, want empty on success", got.Reason)
	}
}

func TestRemoteGitHubRepo_HTTPSNoGitSuffix(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "remote", "add", "origin", "https://github.com/owner/repo")

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Fatalf("RemoteGitHubRepo = %+v, want Owner=owner Repo=repo", got)
	}
}

func TestRemoteGitHubRepo_SCPLikeSyntax(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "remote", "add", "origin", "git@github.com:owner/repo.git")

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Fatalf("RemoteGitHubRepo = %+v, want Owner=owner Repo=repo", got)
	}
}

// TestRemoteGitHubRepo_SCPLikeSyntaxNoUser reproduces WR-06: git's own
// documented scp-like grammar is `[user@]host.xz:path/to/repo.git/` — the
// user is OPTIONAL. `git remote add origin github.com:owner/repo.git` is a
// perfectly ordinary, git-accepted remote with no "@" anywhere in it.
func TestRemoteGitHubRepo_SCPLikeSyntaxNoUser(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "remote", "add", "origin", "github.com:owner/repo.git")

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Fatalf("RemoteGitHubRepo = %+v, want Owner=owner Repo=repo", got)
	}
	if got.Reason != "" {
		t.Fatalf("RemoteGitHubRepo.Reason = %q, want empty on success", got.Reason)
	}
}

// TestParseGitHubRemote_BareLocalPathWithColonNotMisparsedAsSCP reproduces
// IN-04: WR-06 made the scp-like syntax's user@ prefix optional, which
// also removed the implicit "@ must be present" gate that had kept bare
// filesystem paths out of that branch. A Windows-style path with a colon
// before its first path separator (a drive letter) previously parsed as
// host="D" — a single-letter host that is never a valid git remote host —
// echoing a fragment of a local path into RemoteGitHubRepo's Reason text
// instead of the more accurate "not a recognized URL".
func TestParseGitHubRemote_BareLocalPathWithColonNotMisparsedAsSCP(t *testing.T) {
	cases := []string{
		`D:\src\myrepo`,
		`C:/src/myrepo`,
		`c:\Users\me\repo`,
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			host, owner, repo := parseGitHubRemote(raw)
			if host != "" || owner != "" || repo != "" {
				t.Fatalf("parseGitHubRemote(%q) = host=%q owner=%q repo=%q, want all empty (not misparsed as scp-like)", raw, host, owner, repo)
			}
		})
	}
}

// TestParseGitHubRemote_ShortHostSCPStillParses is the positive control
// for the guard above (rule 84d1gfpywd): a genuinely short but VALID
// scp-like host (2+ characters, no path separator) must still parse,
// proving the fix rejects single-character/path-separator candidates
// specifically, not short hosts in general.
func TestParseGitHubRemote_ShortHostSCPStillParses(t *testing.T) {
	host, owner, repo := parseGitHubRemote("ab:owner/repo.git")
	if host != "ab" || owner != "owner" || repo != "repo" {
		t.Fatalf("parseGitHubRemote(%q) = host=%q owner=%q repo=%q, want host=ab owner=owner repo=repo", "ab:owner/repo.git", host, owner, repo)
	}
}

func TestRemoteGitHubRepo_SSHScheme(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "remote", "add", "origin", "ssh://git@github.com/owner/repo.git")

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Fatalf("RemoteGitHubRepo = %+v, want Owner=owner Repo=repo", got)
	}
}

// TestRemoteGitHubRepo_CredentialsStripped pairs the negative containment
// check (the token appears in no returned field) with a positive assertion
// that owner/repo WERE extracted, so the absence check is proven to be
// inspecting a populated result rather than an empty one (rule 84d1gfpywd).
func TestRemoteGitHubRepo_CredentialsStripped(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	const token = "sometoken12345"
	runGit(t, dir, "remote", "add", "origin", "https://someone:"+token+"@github.com/owner/repo.git")

	got := RemoteGitHubRepo(context.Background(), dir)

	// Positive: the extraction actually happened.
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Fatalf("RemoteGitHubRepo = %+v, want Owner=owner Repo=repo (positive control for the absence check below)", got)
	}
	// Negative: the credential appears in no returned field.
	all := got.Owner + "|" + got.Repo + "|" + got.Host + "|" + got.Reason
	if strings.Contains(all, token) {
		t.Fatalf("RemoteGitHubRepo leaked credential token into a returned field: %+v", got)
	}
}

func TestRemoteGitHubRepo_InsteadOfRewriteIsApplied(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "config", "url.https://github.com/.insteadOf", "gh:")
	runGit(t, dir, "remote", "add", "origin", "gh:owner/repo.git")

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Fatalf("RemoteGitHubRepo with insteadOf rewrite = %+v, want Owner=owner Repo=repo", got)
	}
}

// --- RemoteGitHubRepo: no-owner/repo cases, each with a distinct Reason ---

func TestRemoteGitHubRepo_NonGitHubHost(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "remote", "add", "origin", "https://gitlab.com/owner/repo.git")

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "" || got.Repo != "" {
		t.Fatalf("RemoteGitHubRepo(gitlab.com) = %+v, want empty Owner/Repo", got)
	}
	if got.Reason == "" {
		t.Fatalf("RemoteGitHubRepo(gitlab.com).Reason is empty, want a reason naming the host")
	}
	if !strings.Contains(got.Reason, "gitlab.com") {
		t.Fatalf("RemoteGitHubRepo(gitlab.com).Reason = %q, want it to name the host", got.Reason)
	}
}

func TestRemoteGitHubRepo_LookalikeHostRejected(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "remote", "add", "origin", "https://github.example.com/owner/repo.git")

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "" || got.Repo != "" {
		t.Fatalf("RemoteGitHubRepo(github.example.com) = %+v, want empty Owner/Repo (host must be an EXACT match, not a suffix/substring match)", got)
	}
	if !strings.Contains(got.Reason, "github.example.com") {
		t.Fatalf("RemoteGitHubRepo(github.example.com).Reason = %q, want it to name the lookalike host", got.Reason)
	}
}

func TestRemoteGitHubRepo_NoOriginConfigured(t *testing.T) {
	dir := initRepo(t, t.TempDir())

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "" || got.Repo != "" {
		t.Fatalf("RemoteGitHubRepo(no origin) = %+v, want empty Owner/Repo", got)
	}
	if got.Reason == "" {
		t.Fatalf("RemoteGitHubRepo(no origin).Reason is empty, want a reason")
	}
}

func TestRemoteGitHubRepo_NonGitDirectory(t *testing.T) {
	dir := t.TempDir()

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "" || got.Repo != "" {
		t.Fatalf("RemoteGitHubRepo(non-repo dir) = %+v, want empty Owner/Repo", got)
	}
	if got.Reason == "" {
		t.Fatalf("RemoteGitHubRepo(non-repo dir).Reason is empty, want a reason")
	}
}

// TestRemoteGitHubRepo_GitAbsent drives the injected gitExecLookPath seam to
// simulate git being entirely absent from PATH, against a fixture that DOES
// have a real, resolvable GitHub origin remote. If the seam were not
// consulted before the subprocess was constructed, the real "owner"/"repo"
// values would come back; asserting the ZERO value here — against a fixture
// that could only produce a non-zero answer if a subprocess actually ran —
// proves no subprocess was attempted, not merely that the result was empty.
// Mirrors internal/indexer/commit_test.go's TestResolveHeadCommitSHAWithNoGitBinary.
func TestRemoteGitHubRepo_GitAbsent(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")

	orig := gitExecLookPath
	defer func() { gitExecLookPath = orig }()
	gitExecLookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}

	got := RemoteGitHubRepo(context.Background(), dir)
	if got.Owner != "" || got.Repo != "" {
		t.Fatalf("RemoteGitHubRepo with git absent = %+v, want empty Owner/Repo (a real subprocess would have found owner/repo)", got)
	}
	if got.Reason == "" {
		t.Fatalf("RemoteGitHubRepo with git absent has empty Reason, want a reason naming git's absence")
	}
}

// TestRemoteGitHubRepo_ReasonsArePairwiseDistinct asserts every no-owner/repo
// Reason string above is DIFFERENT from every other — not merely
// individually non-empty (rule 84d1gfpywd: a guard must carry a positive
// assertion that it did its work, and "each is non-empty" alone would pass
// even if every branch returned the same generic string).
func TestRemoteGitHubRepo_ReasonsArePairwiseDistinct(t *testing.T) {
	ctx := context.Background()

	gitlabDir := initRepo(t, t.TempDir())
	runGit(t, gitlabDir, "remote", "add", "origin", "https://gitlab.com/owner/repo.git")

	lookalikeDir := initRepo(t, t.TempDir())
	runGit(t, lookalikeDir, "remote", "add", "origin", "https://github.example.com/owner/repo.git")

	noOriginDir := initRepo(t, t.TempDir())

	nonRepoDir := t.TempDir()

	reasons := map[string]string{
		"gitlab-host":    RemoteGitHubRepo(ctx, gitlabDir).Reason,
		"lookalike-host": RemoteGitHubRepo(ctx, lookalikeDir).Reason,
		"no-origin":      RemoteGitHubRepo(ctx, noOriginDir).Reason,
		"non-repo-dir":   RemoteGitHubRepo(ctx, nonRepoDir).Reason,
	}

	orig := gitExecLookPath
	gitExecLookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	reasons["git-absent"] = RemoteGitHubRepo(ctx, initRepo(t, t.TempDir())).Reason
	gitExecLookPath = orig

	seen := map[string]string{}
	for label, reason := range reasons {
		if reason == "" {
			t.Fatalf("case %q produced an empty Reason", label)
		}
		if other, dup := seen[reason]; dup {
			t.Fatalf("case %q and case %q produced the SAME Reason %q, want pairwise-distinct reasons", label, other, reason)
		}
		seen[reason] = label
	}
}

// --- CommitOnRemoteTrackingBranch: the tri-state ---

// newPushedCommitFixture builds a local repo with origin pointed at a real
// bare "remote" repository, pushes the initial commit (so it lands on
// origin/main), then makes ONE further commit that is never pushed.
// Returns the local checkout dir, the pushed commit's sha, and the
// unpushed commit's sha.
func newPushedCommitFixture(t *testing.T) (dir, pushedSHA, unpushedSHA string) {
	t.Helper()
	tmp := t.TempDir()
	bare := filepath.Join(tmp, "remote.git")
	runGit(t, tmp, "init", "--bare", bare)

	local := initRepo(t, filepath.Join(tmp, "local"))
	runGit(t, local, "remote", "add", "origin", bare)
	runGit(t, local, "push", "origin", "HEAD:main")
	pushedSHA = strings.TrimSpace(runGit(t, local, "rev-parse", "HEAD"))

	runGit(t, local, "commit", "--allow-empty", "-m", "unpushed")
	unpushedSHA = strings.TrimSpace(runGit(t, local, "rev-parse", "HEAD"))

	return local, pushedSHA, unpushedSHA
}

func TestCommitOnRemoteTrackingBranch_Observed(t *testing.T) {
	dir, pushedSHA, _ := newPushedCommitFixture(t)

	got := CommitOnRemoteTrackingBranch(context.Background(), dir, pushedSHA)
	if got != RemotePresenceObserved {
		t.Fatalf("CommitOnRemoteTrackingBranch(pushed commit) = %v, want RemotePresenceObserved", got)
	}
}

func TestCommitOnRemoteTrackingBranch_NotObserved(t *testing.T) {
	dir, _, unpushedSHA := newPushedCommitFixture(t)

	got := CommitOnRemoteTrackingBranch(context.Background(), dir, unpushedSHA)
	if got != RemotePresenceNotObserved {
		t.Fatalf("CommitOnRemoteTrackingBranch(unpushed commit) = %v, want RemotePresenceNotObserved (no network access should have occurred)", got)
	}
}

// TestCommitOnRemoteTrackingBranch_UnknownIsNotNotObserved is the tri-state
// proof required by Task 2: it asserts RemotePresenceUnknown !=
// RemotePresenceNotObserved AND that the git-absent path produces Unknown
// while a real unpushed commit produces NotObserved, in the same block —
// demonstrating three states, not a boolean wearing three names.
func TestCommitOnRemoteTrackingBranch_UnknownIsNotNotObserved(t *testing.T) {
	if RemotePresenceUnknown == RemotePresenceNotObserved {
		t.Fatalf("RemotePresenceUnknown must not equal RemotePresenceNotObserved")
	}

	dir, _, unpushedSHA := newPushedCommitFixture(t)

	notObserved := CommitOnRemoteTrackingBranch(context.Background(), dir, unpushedSHA)
	if notObserved != RemotePresenceNotObserved {
		t.Fatalf("CommitOnRemoteTrackingBranch(unpushed) = %v, want RemotePresenceNotObserved", notObserved)
	}

	orig := gitExecLookPath
	defer func() { gitExecLookPath = orig }()
	gitExecLookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	unknown := CommitOnRemoteTrackingBranch(context.Background(), dir, unpushedSHA)
	if unknown != RemotePresenceUnknown {
		t.Fatalf("CommitOnRemoteTrackingBranch with git absent = %v, want RemotePresenceUnknown", unknown)
	}

	if unknown == notObserved {
		t.Fatalf("RemotePresenceUnknown (%v) must differ from RemotePresenceNotObserved (%v) — tri-state collapsed to a boolean", unknown, notObserved)
	}
}

func TestCommitOnRemoteTrackingBranch_MalformedSHAIsUnknown(t *testing.T) {
	dir := initRepo(t, t.TempDir())

	got := CommitOnRemoteTrackingBranch(context.Background(), dir, "not-a-real-sha")
	if got != RemotePresenceUnknown {
		t.Fatalf("CommitOnRemoteTrackingBranch(malformed sha) = %v, want RemotePresenceUnknown", got)
	}
}
