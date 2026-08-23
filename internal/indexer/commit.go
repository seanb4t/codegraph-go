package indexer

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// gitExecLookPath is resolveHeadCommitSHA's executable-lookup seam,
// initialised to exec.LookPath, mirroring internal/graphstore's
// openLockRetrySleep pattern: an unexported, test-only control point with
// no exported setter and no production behavior change. Tests reassign it
// to simulate "no git binary on PATH" deterministically, without emptying
// the process's real PATH — emptying PATH would also break any helper
// process the test itself needs to spawn, making the test's own failure
// mode ambiguous about its cause.
var gitExecLookPath = exec.LookPath

// resolveHeadCommitGitTimeout bounds the git subprocess resolveHeadCommitSHA
// shells out to, so a hung or misbehaving git can never stall an index run
// (T-01-21).
const resolveHeadCommitGitTimeout = 5 * time.Second

// gitSHA1HexLen and gitSHA256HexLen are the two lowercase-hex lengths
// resolveHeadCommitSHA accepts: 40 characters for Git's SHA-1 object
// format, 64 for SHA-256. Both are legitimate `git rev-parse HEAD` outputs
// depending on the repository's object format; accepting only 40 would
// silently record an ABSENT commit on a SHA-256 repository (T-01-19),
// which surfaces downstream as a permalink that simply never renders
// rather than as any visible error. Named here, once, rather than as two
// bare numeric literals at the comparison site.
const (
	gitSHA1HexLen   = 40
	gitSHA256HexLen = 64
)

// resolveHeadCommitSHA resolves the git commit HEAD points at for the
// repository rooted at repoPath (ENG-04, D-05), for stamping into
// schema.Meta.commit_sha. It deliberately returns a bare string and NEVER
// an error: indexing must never fail because a commit could not be
// resolved (T-01-21). Every failure path — git absent from PATH, repoPath
// not being a git checkout, a repository with no commits yet, a detached
// or broken HEAD, or the resolveHeadCommitGitTimeout firing — returns the
// empty string, which schema.IndexedCommitSHA treats identically to a
// pre-upgrade graph that has never carried this field. This "return a
// string, never an error" signature is unusual enough that it would
// otherwise look like a swallowed error; it is not one, it is the D-05
// degrade-gracefully contract made structural.
//
// The command is invoked with a fixed argument vector via
// exec.CommandContext — never a shell string — with repoPath passed as a
// `-C` argument rather than interpolated into anything git parses as
// shell syntax (T-01-19). Output is accepted only when it is exactly
// gitSHA1HexLen or gitSHA256HexLen characters of lowercase hex; any other
// length, or any uppercase character, yields the empty string just as
// firmly as git failing outright.
func resolveHeadCommitSHA(repoPath string) string {
	if _, err := gitExecLookPath("git"); err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), resolveHeadCommitGitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	sha := strings.TrimSpace(string(out))
	if !isLowercaseHexCommitSHA(sha) {
		return ""
	}
	return sha
}

// isLowercaseHexCommitSHA reports whether s is a well-formed git commit
// object id: exactly gitSHA1HexLen or gitSHA256HexLen characters, every
// one of them a lowercase hex digit. Widening from one accepted length to
// two is an ENUMERATED SET, not a relaxation — any other length (39, 41,
// 63, 65, ...) and any uppercase hex character are rejected exactly as
// firmly as before.
func isLowercaseHexCommitSHA(s string) bool {
	if len(s) != gitSHA1HexLen && len(s) != gitSHA256HexLen {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}
