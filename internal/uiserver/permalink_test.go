package uiserver

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// newGitBackedGofixture git-inits copyGofixture's tree (the pkga/pkga.go-
// bearing fixture 03-05-PLAN.md's own <behavior> cases name explicitly),
// commits everything, indexes it, and returns (dir, sha) — sha via an
// independent `git rev-parse HEAD`, mirroring handlers_test.go's
// newGitIndexedFixture. Indexing runs AFTER the commit so
// resolveHeadCommitSHA (internal/indexer) observes the SAME HEAD sha this
// helper returns.
func newGitBackedGofixture(t *testing.T) (dir, sha string) {
	t.Helper()
	dir = copyGofixture(t)
	runGitFixtureCmd(t, dir, "init", "-q", ".")
	runGitFixtureCmd(t, dir, "config", "user.email", "permalinktest@example.com")
	runGitFixtureCmd(t, dir, "config", "user.name", "permalink test")
	runGitFixtureCmd(t, dir, "add", "-A")
	runGitFixtureCmd(t, dir, "commit", "-q", "-m", "init")
	sha = gitRevParseHEAD(t, dir)
	indexGofixture(t, dir)
	return dir, sha
}

// setOriginRemote configures dir's origin remote to remoteURL. Test-only
// fixture-building helper — never the code path under test.
func setOriginRemote(t *testing.T, dir, remoteURL string) {
	t.Helper()
	runGitFixtureCmd(t, dir, "remote", "add", "origin", remoteURL)
}

// markCommitObservedOnRemoteTrackingBranch synthesizes a LOCAL
// remote-tracking ref pointing at sha via `git update-ref`, WITHOUT ever
// creating a real remote or performing any network operation.
// `git branch -r --contains` (internal/gitmeta.CommitOnRemoteTrackingBranch)
// only reads local refs under refs/remotes/ — it has no way to tell a
// synthesized ref from one a real `git fetch` would have created, which
// is exactly what makes this a faithful, hermetic way to drive the
// Observed case without a bare "remote" repository or any push/fetch.
func markCommitObservedOnRemoteTrackingBranch(t *testing.T, dir, sha string) {
	t.Helper()
	runGitFixtureCmd(t, dir, "update-ref", "refs/remotes/origin/main", sha)
}

// --- LINKABLE: line/end_line/no-line anchor shapes (D-09) ---

func TestGetPermalink_LinkableSingleLineAnchor(t *testing.T) {
	dir, sha := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")
	markCommitObservedOnRemoteTrackingBranch(t, dir, sha)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	line := int32(3)
	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{
		Path: "pkga/pkga.go",
		Line: &line,
	}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	if got := resp.Msg.GetAvailability(); got != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE {
		t.Fatalf("availability = %v, want LINKABLE", got)
	}
	want := "https://github.com/owner/repo/blob/" + sha + "/pkga/pkga.go#L3"
	if resp.Msg.GetUrl() != want {
		t.Fatalf("url = %q, want %q", resp.Msg.GetUrl(), want)
	}
}

func TestGetPermalink_LinkableRangeAnchor(t *testing.T) {
	dir, sha := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")
	markCommitObservedOnRemoteTrackingBranch(t, dir, sha)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	line, endLine := int32(3), int32(9)
	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{
		Path:    "pkga/pkga.go",
		Line:    &line,
		EndLine: &endLine,
	}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	want := "https://github.com/owner/repo/blob/" + sha + "/pkga/pkga.go#L3-L9"
	if resp.Msg.GetUrl() != want {
		t.Fatalf("url = %q, want %q", resp.Msg.GetUrl(), want)
	}
}

func TestGetPermalink_LinkableNoLineAnchorAtAll(t *testing.T) {
	dir, sha := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")
	markCommitObservedOnRemoteTrackingBranch(t, dir, sha)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{
		Path: "pkga/pkga.go",
	}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	if strings.Contains(resp.Msg.GetUrl(), "#") {
		t.Fatalf("url = %q, want no fragment at all when line is unset", resp.Msg.GetUrl())
	}
	want := "https://github.com/owner/repo/blob/" + sha + "/pkga/pkga.go"
	if resp.Msg.GetUrl() != want {
		t.Fatalf("url = %q, want %q", resp.Msg.GetUrl(), want)
	}
}

// --- line/end_line validation (IN-10) ---

func TestGetPermalink_RejectsLineBelowOne(t *testing.T) {
	dir, _ := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	line := int32(-1)
	_, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{
		Path: "pkga/pkga.go",
		Line: &line,
	}))
	if err == nil {
		t.Fatalf("GetPermalink(line=-1): expected a non-nil error, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("GetPermalink(line=-1): code = %v, want CodeInvalidArgument", code)
	}
}

func TestGetPermalink_RejectsEndLineBelowLine(t *testing.T) {
	dir, _ := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	line, endLine := int32(42), int32(0)
	_, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{
		Path:    "pkga/pkga.go",
		Line:    &line,
		EndLine: &endLine,
	}))
	if err == nil {
		t.Fatalf("GetPermalink(line=42, end_line=0): expected a non-nil error, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("GetPermalink(line=42, end_line=0): code = %v, want CodeInvalidArgument", code)
	}
}

func TestGetPermalink_RejectsEndLineWithoutLine(t *testing.T) {
	dir, _ := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	endLine := int32(9)
	_, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{
		Path:    "pkga/pkga.go",
		EndLine: &endLine,
	}))
	if err == nil {
		t.Fatalf("GetPermalink(end_line=9, line unset): expected a non-nil error, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("GetPermalink(end_line=9, line unset): code = %v, want CodeInvalidArgument", code)
	}
}

// --- LINKABLE_UNVERIFIED: commit not observed on any remote-tracking branch ---

func TestGetPermalink_LinkableUnverifiedWhenCommitNotObserved(t *testing.T) {
	dir, sha := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")
	// Deliberately NOT calling markCommitObservedOnRemoteTrackingBranch:
	// no remote-tracking ref exists at all, so the containment query runs
	// and finds nothing — the ordinary "local dev tool, unpushed commit"
	// case D-06/D-07 exist for.

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: "pkga/pkga.go"}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	if got := resp.Msg.GetAvailability(); got != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED {
		t.Fatalf("availability = %v, want LINKABLE_UNVERIFIED", got)
	}
	if resp.Msg.GetUrl() == "" {
		t.Fatalf("url is empty, want it populated even though unverified")
	}
	want := "https://github.com/owner/repo/blob/" + sha + "/pkga/pkga.go"
	if resp.Msg.GetUrl() != want {
		t.Fatalf("url = %q, want %q", resp.Msg.GetUrl(), want)
	}
	if resp.Msg.GetReason() == "" {
		t.Fatalf("reason is empty, want it to name the uncertainty")
	}
}

// --- NO_LINK: non-GitHub remote, no remote, no commit sha ---

func TestGetPermalink_NoLinkNonGitHubRemote(t *testing.T) {
	dir, _ := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://gitlab.com/owner/repo.git")

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: "pkga/pkga.go"}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	if got := resp.Msg.GetAvailability(); got != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK {
		t.Fatalf("availability = %v, want NO_LINK", got)
	}
	if resp.Msg.GetUrl() != "" {
		t.Fatalf("url = %q, want empty", resp.Msg.GetUrl())
	}
	if !strings.Contains(resp.Msg.GetReason(), "gitlab.com") {
		t.Fatalf("reason = %q, want it to name the host", resp.Msg.GetReason())
	}
}

func TestGetPermalink_NoLinkNoRemote(t *testing.T) {
	dir, _ := newGitBackedGofixture(t)
	// No origin configured at all.

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: "pkga/pkga.go"}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	if got := resp.Msg.GetAvailability(); got != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK {
		t.Fatalf("availability = %v, want NO_LINK", got)
	}
	if resp.Msg.GetUrl() != "" {
		t.Fatalf("url = %q, want empty", resp.Msg.GetUrl())
	}
	if resp.Msg.GetReason() == "" {
		t.Fatalf("reason is empty, want a reason")
	}
}

func TestGetPermalink_NoLinkNoCommitSHA(t *testing.T) {
	// copyGofixture alone (no git init at all): the indexer's own
	// resolveHeadCommitSHA degrades to "" for a non-git repoRoot, so
	// Meta.CommitSha is empty — the D-05 pre-upgrade-graph case.
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: "pkga/pkga.go"}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	if got := resp.Msg.GetAvailability(); got != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK {
		t.Fatalf("availability = %v, want NO_LINK", got)
	}
	if resp.Msg.GetUrl() != "" {
		t.Fatalf("url = %q, want empty", resp.Msg.GetUrl())
	}
	if !strings.Contains(resp.Msg.GetReason(), "re-index") {
		t.Fatalf("reason = %q, want it to name re-indexing as the remedy", resp.Msg.GetReason())
	}
}

// overwriteCommitSHA rewrites an already-indexed store's Meta.commit_sha
// directly, bypassing internal/indexer's own write-time
// isLowercaseHexCommitSHA validation entirely — simulating a store built
// or edited by something other than this binary's own indexer (WR-07:
// the milestone-2 "CI-distributed indexes" shape .claude/CLAUDE.md names
// as this project's own architecture target).
func overwriteCommitSHA(t *testing.T, dir, sha string) {
	t.Helper()
	storeDir := filepath.Join(dir, ".codegraph", "store")
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	meta, err := snap.GetMeta()
	if err != nil {
		snap.Close()
		t.Fatalf("get meta: %v", err)
	}
	snap.Close()

	meta.CommitSha = sha
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("put meta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// TestGetPermalink_NoLinkMalformedCommitSHA reproduces WR-07: a Meta
// record carrying a commit_sha that is not well-formed hex (e.g. written
// by something other than this binary's own indexer) must degrade to
// NO_LINK, not reach buildGitHubBlobURL (which would splice it raw into
// the rendered URL) or gitmeta.CommitOnRemoteTrackingBranch (a git CLI
// argument).
func TestGetPermalink_NoLinkMalformedCommitSHA(t *testing.T) {
	dir, sha := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")
	markCommitObservedOnRemoteTrackingBranch(t, dir, sha)
	overwriteCommitSHA(t, dir, "../../attacker/attacker-repo/blob/main")

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: "pkga/pkga.go"}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	if got := resp.Msg.GetAvailability(); got != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK {
		t.Fatalf("availability = %v, want NO_LINK", got)
	}
	if resp.Msg.GetUrl() != "" {
		t.Fatalf("url = %q, want empty — a malformed commit_sha must never reach buildGitHubBlobURL", resp.Msg.GetUrl())
	}
	if !strings.Contains(resp.Msg.GetReason(), "re-index") {
		t.Fatalf("reason = %q, want it to name re-indexing as the remedy", resp.Msg.GetReason())
	}
}

// --- Percent-encoding (T-03-21) ---

func TestGetPermalink_PercentEncodesURLSignificantCharacters(t *testing.T) {
	dir, sha := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")
	markCommitObservedOnRemoteTrackingBranch(t, dir, sha)

	const rawName = "odd file#1?.txt"
	if err := os.WriteFile(filepath.Join(dir, rawName), []byte("hi\n"), 0o644); err != nil {
		t.Fatalf("write odd-named file: %v", err)
	}

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: rawName}))
	if err != nil {
		t.Fatalf("GetPermalink: %v", err)
	}
	const wantEncoded = "odd%20file%231%3F.txt"
	if !strings.Contains(resp.Msg.GetUrl(), wantEncoded) {
		t.Fatalf("url = %q, want it to contain the percent-encoded segment %q", resp.Msg.GetUrl(), wantEncoded)
	}
	if strings.Contains(resp.Msg.GetUrl(), rawName) {
		t.Fatalf("url = %q, contains the RAW unencoded name %q — the significant characters were not encoded", resp.Msg.GetUrl(), rawName)
	}
}

// --- Confinement (SRV-05, T-03-01b): the SAME gate GetNodeDetail uses,
// proven at this endpoint's own boundary with a passing control. ---

func TestGetPermalinkPathConfinementAtRPCBoundary(t *testing.T) {
	dir, sha := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")
	markCommitObservedOnRemoteTrackingBranch(t, dir, sha)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	// Positive control FIRST: a service broken for every input must fail
	// here, not read as successful refusals below it.
	t.Run("in-repo control", func(t *testing.T) {
		resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: "pkga/pkga.go"}))
		if err != nil {
			t.Fatalf("GetPermalink(in-repo control): %v", err)
		}
		if resp.Msg.GetUrl() == "" {
			t.Fatalf("GetPermalink(in-repo control): url is empty, want a populated permalink")
		}
	})

	cases := []struct {
		name     string
		path     string
		wantFrag string
	}{
		{name: "escape", path: "../outside.txt", wantFrag: "escapes the repo root"},
		{name: "absolute", path: "/etc/passwd"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: c.path}))
			if err == nil {
				t.Fatalf("GetPermalink(path=%q) succeeded, want a refusal", c.path)
			}
			if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
				t.Fatalf("GetPermalink(path=%q): code = %v, want CodeInvalidArgument (err=%q)", c.path, code, err.Error())
			}
			if c.wantFrag != "" && !strings.Contains(err.Error(), c.wantFrag) {
				t.Fatalf("GetPermalink(path=%q): error = %q, want it to contain %q", c.path, err.Error(), c.wantFrag)
			}
		})
	}
}

// TestGetPermalinkRefusesSinceDeletedFile pins the maintainer's Task 1
// checkpoint decision (point 5: deleted-classify-in-handler) with a fixed
// test name so <verify> does not need to branch on which disposition was
// chosen — only what this test ASSERTS differs between the two.
//
// Under deleted-classify-in-handler (chosen here): a repo-relative,
// non-escaping path that does not exist in the working tree is refused
// as CodeInvalidArgument, with a message built from the CALLER's own
// repo-relative path — and the message must NOT contain the fixture's
// absolute host checkout path, which the underlying *fs.PathError
// carries and WR-01 exists to keep off the wire. The negative
// containment check is paired with a positive one (the message DOES
// name the repo-relative path), per rule 84d1gfpywd: a bare negative
// check passes trivially against an empty message.
func TestGetPermalinkRefusesSinceDeletedFile(t *testing.T) {
	dir, sha := newGitBackedGofixture(t)
	setOriginRemote(t, dir, "https://github.com/owner/repo.git")
	markCommitObservedOnRemoteTrackingBranch(t, dir, sha)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	const relPath = "pkga/this-file-never-existed.go"
	_, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: relPath}))
	if err == nil {
		t.Fatalf("GetPermalink(path=%q) succeeded, want a refusal (path does not exist in the working tree)", relPath)
	}

	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("GetPermalink(path=%q): code = %v, want CodeInvalidArgument", relPath, code)
	}
	msg := err.Error()
	if msg == "" {
		t.Fatalf("GetPermalink(path=%q): refusal message is empty", relPath)
	}
	// Positive: the message names the caller's own repo-relative path.
	if !strings.Contains(msg, relPath) {
		t.Fatalf("GetPermalink(path=%q): message %q does not contain the caller's own repo-relative path — the negative host-path check below would be unproven without this", relPath, msg)
	}
	// Negative: the message does NOT leak the fixture's absolute host
	// checkout path, which the underlying *fs.PathError carries.
	if strings.Contains(msg, dir) {
		t.Fatalf("GetPermalink(path=%q): message %q contains the fixture's absolute host checkout path %q", relPath, msg, dir)
	}
}

// --- remotePresenceResponse: the default arm's safe direction (IN-08) ---

// TestRemotePresenceResponse_KnownValues proves the two currently-real
// RemotePresence values still map to their documented outcome after the
// switch was reordered — the reordering must not change observable
// behavior for any value that actually exists today.
func TestRemotePresenceResponse_KnownValues(t *testing.T) {
	const blobURL = "https://github.com/owner/repo/blob/deadbeef/pkga/pkga.go"

	observed := remotePresenceResponse(blobURL, gitmeta.RemotePresenceObserved)
	if observed.GetAvailability() != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE {
		t.Fatalf("RemotePresenceObserved: availability = %v, want LINKABLE", observed.GetAvailability())
	}
	if observed.GetReason() != "" {
		t.Fatalf("RemotePresenceObserved: reason = %q, want empty", observed.GetReason())
	}

	notObserved := remotePresenceResponse(blobURL, gitmeta.RemotePresenceNotObserved)
	if notObserved.GetAvailability() != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED {
		t.Fatalf("RemotePresenceNotObserved: availability = %v, want LINKABLE_UNVERIFIED", notObserved.GetAvailability())
	}
	if notObserved.GetReason() != notObservedReason {
		t.Fatalf("RemotePresenceNotObserved: reason = %q, want %q", notObserved.GetReason(), notObservedReason)
	}

	unknown := remotePresenceResponse(blobURL, gitmeta.RemotePresenceUnknown)
	if unknown.GetAvailability() != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED {
		t.Fatalf("RemotePresenceUnknown: availability = %v, want LINKABLE_UNVERIFIED", unknown.GetAvailability())
	}
	if unknown.GetReason() != checkUnknownReason {
		t.Fatalf("RemotePresenceUnknown: reason = %q, want %q", unknown.GetReason(), checkUnknownReason)
	}
}

// TestRemotePresenceResponse_UnrecognizedValueDegradesToUnknown reproduces
// IN-08: a RemotePresence value this switch does not recognize at all —
// simulating a future member added to the gitmeta package's enum without
// a corresponding case here — must degrade to the SAFE "could not check"
// wording (checkUnknownReason), never to notObservedReason's positive
// "this commit is not observed" claim, which D-07 forbids stating without
// having actually checked.
func TestRemotePresenceResponse_UnrecognizedValueDegradesToUnknown(t *testing.T) {
	const blobURL = "https://github.com/owner/repo/blob/deadbeef/pkga/pkga.go"

	future := gitmeta.RemotePresence(99) // a value no case in the switch names
	got := remotePresenceResponse(blobURL, future)

	if got.GetAvailability() != uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED {
		t.Fatalf("unrecognized RemotePresence(99): availability = %v, want LINKABLE_UNVERIFIED", got.GetAvailability())
	}
	if got.GetReason() != checkUnknownReason {
		t.Fatalf("unrecognized RemotePresence(99): reason = %q, want the SAFE checkUnknownReason %q — never notObservedReason's positive claim", got.GetReason(), checkUnknownReason)
	}
}
