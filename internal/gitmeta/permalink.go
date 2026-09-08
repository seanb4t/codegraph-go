// permalink.go derives the pieces GetPermalink (D-06) needs to construct a
// GitHub blob URL pinned to the indexed commit: the remote's owner/repo
// (D-08: GitHub only, everything else is an explicit no-link) and whether
// that commit is observably present on a remote-tracking branch (D-07: the
// check is sound in one direction only, so "could not check" and "checked,
// not there" must never collapse to the same value). D-09's line-range
// anchor is assembled by the caller (internal/uiserver/permalink.go) from
// values this file returns.
//
// Every function here follows the package's existing degrade-to-a-value
// contract (see the package doc in worktree.go): no error is ever
// returned. But the VALUE degraded to RECORDS WHICH failure occurred —
// RemotePresenceUnknown rather than a bare false, and a populated
// GitHubRemote.Reason rather than an empty struct — because a caller that
// cannot distinguish "I checked and found nothing" from "I could not check"
// cannot honestly report either (cycle-1 review, Codex, HIGH).
package gitmeta

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// gitExecLookPath is this file's test-only executable-lookup seam,
// mirroring internal/indexer/commit.go's gitExecLookPath: an unexported
// variable with no exported setter, reassigned only by tests to simulate
// "no git binary on PATH" deterministically. Every function in this file
// consults it before constructing any *exec.Cmd, so "git is absent" is a
// testable branch (cycle-1 review, Codex, MEDIUM) rather than an
// environment condition nothing in this package could previously drive.
var gitExecLookPath = exec.LookPath

// GitHubRemote is the outcome of parsing a repository's origin remote and
// classifying it against D-08 (GitHub only). On success Owner and Repo are
// populated and Reason is empty. On any failure — no origin, unsupported
// host, malformed remote, non-repo directory, or git absent — Owner and
// Repo are empty, Host names whatever host WAS parsed (empty if none
// could be), and Reason is a short, credential-free, user-facing sentence
// naming the SPECIFIC cause. A bare (owner, repo) pair can only say "no";
// this struct says WHICH no (cycle-1 review, Codex, MEDIUM).
type GitHubRemote struct {
	Owner  string
	Repo   string
	Host   string
	Reason string
}

// RemoteURL returns dir's origin remote URL with any
// `url.<base>.insteadOf` rewrite already applied, or "" if dir has no
// origin remote, isn't a git repository, or git is unavailable. Uses
// `git ls-remote --get-url origin`, which git's own documentation states
// "exit[s] without talking to the remote" — this must never be swapped
// for a plain `git remote get-url` / config read, which returns the
// UNREWRITTEN value and would silently disagree with what git itself
// resolves for a developer with an insteadOf rule configured.
//
// Note: `git ls-remote --get-url <name>` does not fail when <name> has no
// configured remote — it echoes the literal argument back unchanged
// (verified empirically). Callers must treat that echo, not a non-zero
// exit, as "no such remote".
func RemoteURL(ctx context.Context, dir string) string {
	if _, err := gitExecLookPath("git"); err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--get-url", "origin")
	cmd.Dir = dir
	cmd.Stdin = nil // git must never be able to block on an interactive prompt
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" || trimmed == "origin" {
		return ""
	}
	return trimmed
}

// RemoteGitHubRepo parses dir's origin remote and classifies it against
// D-08. Every failure path populates Reason with a distinct, actionable
// sentence rather than collapsing to one undifferentiated refusal
// (cycle-1 review, Codex, MEDIUM): git absent, non-repo directory, no
// origin configured, and an unsupported/lookalike host are each named
// separately.
func RemoteGitHubRepo(ctx context.Context, dir string) GitHubRemote {
	if _, err := gitExecLookPath("git"); err != nil {
		return GitHubRemote{Reason: "git is not installed or not on PATH"}
	}
	if !IsGitRepo(ctx, dir) {
		return GitHubRemote{Reason: "not a git repository"}
	}

	raw := RemoteURL(ctx, dir)
	if raw == "" {
		return GitHubRemote{Reason: "no origin remote is configured for this repository"}
	}

	host, owner, repo := parseGitHubRemote(raw)
	if host != "github.com" {
		if host == "" {
			return GitHubRemote{Reason: "origin remote is not a recognized URL"}
		}
		return GitHubRemote{Host: host, Reason: fmt.Sprintf("origin remote host %q is not github.com", host)}
	}
	if owner == "" || repo == "" {
		return GitHubRemote{Host: host, Reason: "could not parse an owner/repo path from the origin remote"}
	}
	return GitHubRemote{Owner: owner, Repo: repo, Host: host}
}

// parseGitHubRemote splits a raw git remote URL into (host, owner, repo).
// It handles the three transport shapes real repositories use: a URL with
// a scheme (https://, ssh://, git://), the scp-like `user@host:path` form
// git itself recognizes with no scheme present, and a bare local
// filesystem path (no host at all, returned as host="").
//
// Host is returned even on a parse "failure" of owner/repo, so a caller
// can name the lookalike/unsupported host back to the user. Host
// comparison against the literal "github.com" is the CALLER's job
// (RemoteGitHubRepo) and must remain exact equality — this function does
// no host filtering of its own.
func parseGitHubRemote(raw string) (host, owner, repo string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", ""
	}

	if idx := strings.Index(raw, "://"); idx == -1 {
		// No scheme. Recognize git's scp-like syntax ([user@]host:path) —
		// git's own documented grammar makes the user optional (WR-06:
		// `github.com:owner/repo.git`, with no "git@"/user prefix, is a
		// perfectly ordinary, git-accepted remote) — keying the decision
		// on a colon appearing before any slash. Strip a leading user@ if
		// present, then apply the colon-before-slash rule to what
		// remains; otherwise this is a bare local filesystem path, which
		// has no host at all.
		rest := raw
		if at := strings.Index(raw, "@"); at >= 0 {
			rest = raw[at+1:]
		}
		if colon := strings.Index(rest, ":"); colon >= 0 {
			slash := strings.Index(rest, "/")
			if slash == -1 || colon < slash {
				candidateHost := rest[:colon]
				// IN-04: reject a candidate host that is a single
				// character or contains a path separator. WR-06 made the
				// user@ prefix optional, which also removed the implicit
				// "@ must be present" gate that had kept bare filesystem
				// paths out of this branch — without this check,
				// `D:\src\myrepo` (or any local path with a colon before
				// its first "/") parses as host="D", misclassifying a
				// local path as a remote scp-like URL and echoing a
				// fragment of it into RemoteGitHubRepo's Reason text.
				// git's own scp-like grammar names a HOST here
				// ([user@]host.xz:path/to/repo.git/); a single-letter
				// drive designator or a string containing "\" or "/" is
				// never a valid one.
				if len(candidateHost) > 1 && !strings.ContainsAny(candidateHost, `\/`) {
					host = candidateHost
					owner, repo = splitOwnerRepo(rest[colon+1:])
					return host, owner, repo
				}
			}
		}
		return "", "", ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", "", ""
	}
	host = u.Hostname() // deliberately excludes any userinfo and port
	owner, repo = splitOwnerRepo(strings.TrimPrefix(u.Path, "/"))
	return host, owner, repo
}

// splitOwnerRepo extracts the last two `/`-separated segments of path as
// (owner, repo), stripping a trailing "/" and a trailing ".git" first.
// GitHub's own URL shape is flat (owner/repo); the last-two-segments rule
// tolerates a leading slash or trailing slash without over-fitting to a
// single exact input shape.
func splitOwnerRepo(path string) (owner, repo string) {
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", ""
	}
	owner = parts[len(parts)-2]
	repo = parts[len(parts)-1]
	if owner == "" || repo == "" {
		return "", ""
	}
	return owner, repo
}

// RemotePresence is the tri-state result of asking whether a commit is
// observably present on a remote-tracking branch. It is deliberately NOT
// a bool and NOT a (bool, error) pair: collapsing "the query ran and found
// nothing" and "the query could not run at all" into the same false value
// would make D-07's honesty requirement unimplementable at the only layer
// that could implement it (cycle-1 review, Codex, HIGH).
type RemotePresence int

const (
	// RemotePresenceUnknown is the zero value: the containment query could
	// not run at all — git absent, dir not a repository, a malformed sha,
	// or the gitTimeout firing. This is NOT a statement about the commit.
	RemotePresenceUnknown RemotePresence = iota
	// RemotePresenceObserved means the query ran and FOUND the commit on
	// at least one remote-tracking branch. Sound in this direction: a hit
	// proves the commit is on the remote.
	RemotePresenceObserved
	// RemotePresenceNotObserved means the query ran and did NOT find the
	// commit on any remote-tracking branch. NOT proof the commit is absent
	// from the remote — only that the last fetch did not see it.
	RemotePresenceNotObserved
)

// CommitOnRemoteTrackingBranch reports whether sha is reachable from any
// LOCAL remote-tracking branch (`git branch -r --contains <sha>`) — this
// function never fetches and performs no network I/O of any kind.
//
// The Observed answer is sound in exactly ONE direction: a hit PROVES the
// commit is on a remote-tracking branch. No hit proves only that the last
// fetch did not see it — someone else may have pushed it since, or the
// user may simply not have fetched. A read-only local tool must not fetch
// to find out, so the "not found" case degrades to RemotePresenceNotObserved
// (an honest answer about uncertainty), never to a claim that the commit
// is absent from the remote (D-07).
func CommitOnRemoteTrackingBranch(ctx context.Context, dir, sha string) RemotePresence {
	if _, err := gitExecLookPath("git"); err != nil {
		return RemotePresenceUnknown
	}

	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	// WR-07 raised passing "--" here to force git to treat sha as a
	// positional revision rather than an option. Empirically verified
	// (git 2.54.0) this is unnecessary AND WRONG for `branch --contains`
	// specifically: git's parse-options unconditionally consumes the very
	// next argv token as --contains's value regardless of a leading "-"
	// (`--contains -o`, `--contains --help`, and `--contains --` all fail
	// with "malformed object name <token>", never a flag reinterpretation)
	// — so a hostile sha already fails safely here with no "--" needed,
	// and inserting one breaks the command outright (git resolves the
	// literal "--" itself as the object name and errors). The real
	// hardening for a malformed/hostile commit_sha is validating it once
	// at the read boundary before it ever reaches this call — see
	// schema.IsCommitSHA and its caller in uiserver.GetPermalink.
	cmd := exec.CommandContext(ctx, "git", "branch", "-r", "--contains", sha)
	cmd.Dir = dir
	cmd.Stdin = nil // git must never be able to block on an interactive prompt
	out, err := cmd.Output()
	if err != nil {
		// Non-zero exit: malformed sha, non-repo directory, or a transient
		// failure. The query did not run to completion, so this is
		// "unknown", never "not observed".
		return RemotePresenceUnknown
	}
	if strings.TrimSpace(string(out)) == "" {
		return RemotePresenceNotObserved
	}
	return RemotePresenceObserved
}
