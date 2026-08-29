// permalink.go implements GetPermalink (D-06/D-07/D-08/D-09): it turns a
// repo-relative path and an optional line/end_line anchor into a GitHub
// blob URL pinned to the commit the INDEX was built at, never HEAD, so
// the remote view matches what the UI just showed.
//
// Everything this handler produces from git introspection is an ANSWER,
// never an error: no remote, a non-GitHub forge, an unpushed commit, or
// an absent commit_sha all render as a SUCCESSFUL response carrying
// PERMALINK_AVAILABILITY_NO_LINK or PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED
// with a populated reason — the same "empty is a successful response"
// discipline Explore already follows (D-07). The one case that IS an
// error is an invalid path argument, handled by (*query.Engine).
// ValidateRepoRelativePath (SRV-05) below.
package uiserver

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// noCommitSHAReason is GetPermalink's NO_LINK reason when the index's own
// Meta carries no commit SHA at all (D-05): a pre-upgrade graph that
// predates ENG-04's commit_sha field. Named the remedy explicitly —
// re-indexing — rather than leaving the caller to guess.
const noCommitSHAReason = "this index has no recorded commit SHA (a pre-upgrade graph) — re-index to enable permalinks"

// notObservedReason and checkUnknownReason are GetPermalink's
// LINKABLE_UNVERIFIED reasons (D-07). Both cases share ONE availability
// value on the wire — from the caller's perspective the link is equally
// uncertain either way — but the text differs, so an operator reading
// `reason` can still tell "ran and found nothing" from "could not run
// the check at all" without conflating them into the SAME string.
const (
	notObservedReason  = "this commit is not observed on any remote-tracking branch; it may be unpushed, and the link may 404 until it is pushed"
	checkUnknownReason = "could not verify whether this commit is on a remote-tracking branch (git unavailable, not a repository, or the check timed out); the link may 404"
)

// GetPermalink answers BRW-09 over the wire. path is confined by
// (*query.Engine).ValidateRepoRelativePath — a one-line delegating
// wrapper over the SAME resolveSourcePath gate GetNodeDetail and Explore
// already share (SRV-05); this handler adds no second confinement
// implementation.
//
// ONE case reclassifies that gate's own error for presentation, per the
// maintainer's Task 1 checkpoint decision (point 5, disposition
// deleted-classify-in-handler): a repo-relative, non-escaping path that
// does not EXIST in the working tree. resolveSourcePath's
// filepath.EvalSymlinks step returns that case as a raw *fs.PathError
// (unlike every OTHER refusal in that function, which is wrapped with
// invalidArgumentf), so mapEngineError's default arm would otherwise
// scrub it to an opaque CodeInternal "an internal error occurred" —
// materially worse than actionable for BRW-09's single most valuable
// case, a permalink to a file that existed at the indexed commit and has
// since been deleted or renamed. This reclassification happens ONLY in
// this handler, matches on errors.Is(err, fs.ErrNotExist), and builds its
// message from the CALLER'S OWN repo-relative path — never from the
// underlying *fs.PathError text, which carries the absolute host path of
// the checkout (WR-01/T-01-06). Confinement itself is unchanged: every
// escape, absolute-path and symlink-escape refusal is still produced by
// the same ValidateRepoRelativePath call, unclassified, falling through
// to mapEngineError exactly as before. This branch reclassifies the
// PRESENTATION of one already-refused case; it does not add a second
// confinement decision.
func (s *uiService) GetPermalink(ctx context.Context, req *connect.Request[uiv1.GetPermalinkRequest]) (*connect.Response[uiv1.GetPermalinkResponse], error) {
	var resp *uiv1.GetPermalinkResponse
	// classifiedErr carries the since-deleted-file reclassification
	// (disposition B) OUTSIDE withEngine's own error-mapping path.
	// withEngine unconditionally re-maps ANY error fn returns through
	// mapEngineError — including an already-built *connect.Error — so
	// returning connect.NewError(...) directly from fn here would be
	// re-wrapped a second time into an opaque CodeInternal, exactly the
	// outcome this reclassification exists to avoid. Recording it in this
	// outer variable and returning nil from fn instead lets GetPermalink
	// return it verbatim, once withEngine itself has returned successfully.
	var classifiedErr error
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		path := req.Msg.GetPath()
		if verr := eng.ValidateRepoRelativePath(path); verr != nil {
			if errors.Is(verr, fs.ErrNotExist) {
				classifiedErr = connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(
					"path %q does not exist in the working tree — it may have existed at the indexed commit and been deleted or renamed since; re-index or check the commit history",
					path,
				))
				return nil
			}
			return verr
		}

		meta, err := eng.IndexMeta()
		if err != nil {
			return err
		}
		sha, ok := schema.IndexedCommitSHA(meta)
		if !ok || sha == "" {
			resp = &uiv1.GetPermalinkResponse{
				Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK,
				Reason:       noCommitSHAReason,
			}
			return nil
		}
		// WR-07: internal/indexer validates a commit SHA with
		// schema.IsCommitSHA before ever stamping it into Meta at WRITE
		// time (internal/indexer/commit.go), but IndexedCommitSHA above
		// returns whatever string a Meta record on disk happens to carry
		// — the read side previously skipped that validator entirely. sha
		// is used below both spliced raw into a rendered GitHub blob URL
		// (buildGitHubBlobURL — every OTHER component of that URL is
		// escaped; this one was not) and as a git CLI argument
		// (gitmeta.CommitOnRemoteTrackingBranch). A store built by
		// anything other than this binary's own indexer — the
		// milestone-2 "CI-distributed indexes" shape this project's own
		// architecture targets — is not bound by the write-time
		// guarantee, so re-validate once here, at the one place every
		// path to those two sinks passes through.
		if !schema.IsCommitSHA(sha) {
			resp = &uiv1.GetPermalinkResponse{
				Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK,
				Reason:       noCommitSHAReason,
			}
			return nil
		}

		remote := gitmeta.RemoteGitHubRepo(ctx, s.repoPath)
		if remote.Owner == "" || remote.Repo == "" {
			resp = &uiv1.GetPermalinkResponse{
				Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK,
				Reason:       remote.Reason,
			}
			return nil
		}

		blobURL := buildGitHubBlobURL(remote.Owner, remote.Repo, sha, path, req.Msg.Line, req.Msg.EndLine)

		switch gitmeta.CommitOnRemoteTrackingBranch(ctx, s.repoPath, sha) {
		case gitmeta.RemotePresenceObserved:
			resp = &uiv1.GetPermalinkResponse{
				Url:          blobURL,
				Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE,
			}
		case gitmeta.RemotePresenceUnknown:
			resp = &uiv1.GetPermalinkResponse{
				Url:          blobURL,
				Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED,
				Reason:       checkUnknownReason,
			}
		default: // gitmeta.RemotePresenceNotObserved
			resp = &uiv1.GetPermalinkResponse{
				Url:          blobURL,
				Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED,
				Reason:       notObservedReason,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if classifiedErr != nil {
		return nil, classifiedErr
	}
	return connect.NewResponse(resp), nil
}

// buildGitHubBlobURL assembles a GitHub permalink from its parts (D-09).
// Each path segment is percent-encoded independently, via url.PathEscape
// on the SPLIT segments rather than on the joined string, so a literal
// "/" inside a filename can never be misinterpreted as a path separator
// and no URL-significant character (#, ?, a space) in a path segment
// silently produces a link to a different location than requested. The
// anchor is a range when endLine is set, a single line when only line is
// set, and absent entirely when line itself is unset.
func buildGitHubBlobURL(owner, repo, sha, path string, line, endLine *int32) string {
	var b strings.Builder
	b.WriteString("https://github.com/")
	b.WriteString(owner)
	b.WriteString("/")
	b.WriteString(repo)
	b.WriteString("/blob/")
	b.WriteString(sha)
	b.WriteString("/")
	b.WriteString(percentEncodeRepoPath(path))
	if line != nil {
		if endLine != nil {
			fmt.Fprintf(&b, "#L%d-L%d", *line, *endLine)
		} else {
			fmt.Fprintf(&b, "#L%d", *line)
		}
	}
	return b.String()
}

// percentEncodeRepoPath percent-encodes each "/"-separated segment of a
// repo-relative path independently and rejoins them with a literal "/" —
// never escaping the separator itself.
func percentEncodeRepoPath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}
