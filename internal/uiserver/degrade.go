package uiserver

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// degradeKind classifies an openEngine failure into one of the states
// SRV-04's degrade path must render distinctly (D-14, D-16): a
// repository that was never indexed at all (degradeNotInitialized)
// versus one whose store is locked past graphstore.Open's bounded retry
// budget (degradeIndexingInProgress). Neither state is surfaced to the
// human as an ordinary error: GetStatus answers both from filesystem
// facts (degradedStatus), and every other handler maps the locked case
// to a typed CodeUnavailable (errIndexingInProgress, wired through
// mapEngineError).
type degradeKind int

const (
	// degradeNone means classifyDegrade found neither sentinel in err's
	// chain: err is nil, or unrelated to either degrade condition. The
	// caller treats this as an ordinary failure — mapEngineError's
	// default CodeInternal path for non-Status handlers, or an
	// unmodified propagation for GetStatus.
	degradeNone degradeKind = iota
	// degradeNotInitialized means query.ErrNotInitialized is present in
	// err's chain: no .codegraph/ directory exists anywhere up
	// query.ResolveCodegraphDir's walk.
	degradeNotInitialized
	// degradeIndexingInProgress means graphstore.ErrStoreLocked is
	// present in err's chain: a .codegraph/ directory exists, but its
	// store stayed locked past graphstore.Open's bounded retry budget
	// (D-15: no second retry layer is added above that budget — this
	// classification fires only after Open's own loop has already given
	// up).
	degradeIndexingInProgress
)

// classifyDegrade is an EXPLICIT ORDERED CHECK: query.ErrNotInitialized
// is checked FIRST, graphstore.ErrStoreLocked SECOND, and anything else
// (including nil) falls through to degradeNone. Not-initialized outranks
// locked because a repository with no .codegraph/ cannot be mid-index —
// reporting "indexing in progress" there would be actively misleading.
//
// On a real filesystem the two conditions cannot co-occur:
// query.ResolveCodegraphDir (internal/query/resolve.go) returns
// ErrNotInitialized only after walking to the filesystem root without
// finding a .codegraph/ directory anywhere — and if no such directory
// exists, there is no store directory whose lock could be contended.
// query.OpenAt calls ResolveCodegraphDir BEFORE ever attempting
// graphstore.Open, so a lock-held failure can only occur once a
// .codegraph/ directory has already been found. The precedence is
// encoded anyway because this classifier is the one place a future
// caller could hand it an error chain carrying both sentinels (e.g. a
// constructed multi-target error, or a future error path this plan does
// not anticipate), and because an ordered check is cheaper to read than
// an argument about which branch happens to run first.
// TestClassifyDegradePrecedence drives this directly with such a
// constructed error rather than attempting to build an impossible
// repository on disk — a test that cannot reach its own precondition
// proves nothing.
//
// Classification is errors.Is against the two exported sentinels ONLY —
// never a broader check or a string match. internal/cli/serve.go's own
// comment on its ErrStoreLocked branch explains why: a permission error
// anywhere in an error chain must never masquerade as lock contention.
func classifyDegrade(err error) degradeKind {
	if err == nil {
		return degradeNone
	}
	switch {
	case errors.Is(err, query.ErrNotInitialized):
		return degradeNotInitialized
	case errors.Is(err, graphstore.ErrStoreLocked):
		return degradeIndexingInProgress
	default:
		return degradeNone
	}
}

// indexingInProgressMessage is the ONE fixed, generic sentence every
// degraded non-GetStatus RPC's IndexingInProgress detail carries (D-14,
// T-01-06). It names the condition and suggests a retry, and nothing
// else: no lock-file path, no holding process id, no store directory —
// this message crosses into an unauthenticated loopback browser caller.
// TestIndexingInProgressMessageLeaksNothing asserts exact equality
// against this constant and that it carries no path separator of either
// kind.
const indexingInProgressMessage = "The index is being rebuilt. Please retry shortly."

// indexingInProgressError is a named error type carrying
// indexingInProgressMessage as its Error() text (IN-01). ST1005
// ("error strings should not be capitalized or end with punctuation")
// only inspects string literals passed to errors.New/fmt.Errorf; a
// literal passed through a custom Error() method is outside its scope
// entirely. This removes the trade-off a blanket `//nolint:staticcheck`
// carried on the errors.New call it replaces: that directive disabled
// ALL staticcheck classes on its line — SA correctness checks, S
// simplifications, QF quickfixes — not only ST1005, so a future genuine
// SA-class finding on the connect.NewError(...) call would have been
// silently suppressed alongside the one this project actually reasoned
// about.
type indexingInProgressError struct{}

func (indexingInProgressError) Error() string { return indexingInProgressMessage }

// errIndexingInProgress builds the SRV-04/D-14 degrade error:
// connect.CodeUnavailable — the Connect protocol reference's own
// "transient, back off and retry" code — carrying a typed
// IndexingInProgress detail with the fixed generic message above.
// connect.CodeFailedPrecondition is explicitly wrong here: the user
// fixes nothing and the condition resolves itself once the re-index
// completes. Every non-GetStatus handler's degraded response goes
// through this constructor, via mapEngineError, so there is exactly one
// place this error shape is built.
func errIndexingInProgress() error {
	// indexingInProgressMessage is deliberately punctuated, user-facing
	// prose crossing to an unauthenticated browser caller (see its own doc
	// comment) — never wrapped with fmt.Errorf/%w into a chained internal
	// error, so the "error strings should not be capitalized or end with
	// punctuation" Go convention ST1005 enforces does not apply here. It is
	// carried via indexingInProgressError{} above (a named error type)
	// rather than errors.New(indexingInProgressMessage), which sidesteps
	// ST1005 by construction rather than by directive.
	connErr := connect.NewError(connect.CodeUnavailable, indexingInProgressError{})
	if detail, detailErr := connect.NewErrorDetail(&uiv1.IndexingInProgress{Message: indexingInProgressMessage}); detailErr == nil {
		connErr.AddDetail(detail)
	}
	return connErr
}

// degradedStatus builds GetStatus's D-16 answer once openEngine has
// already failed for repoPath: new plumbing, not a reused fallback —
// Engine.Status() runs only after a successful Engine construction, so
// it never runs in this case today and there is nothing to fall back to
// (01-RESEARCH.md Pitfall 3).
//
// openErr is classified first (classifyDegrade); a degradeNone
// classification means this is NOT a degrade case at all, and
// degradedStatus reports that by returning openErr unchanged as its
// second return value alongside a nil response — the caller (GetStatus)
// must then fall back to mapEngineError instead of treating this as a
// successful degraded response.
//
// For an actual degrade, query.ResolveCodegraphDir is called
// independently on repoPath — the one additional filesystem read this
// path can safely make without ever touching graphstore.Open again
// (D-15): it only os.Stats the .codegraph/ directory, succeeding
// regardless of whether the store inside it is locked. Its result
// determines store_exists directly from the filesystem rather than by
// re-trusting openErr's own classification, which is what "answers from
// filesystem facts alone" means for this handler. Every graph-derived
// count (node_count, edge_count, file_count) is left at its zero value:
// there is no opened store to read them from.
func degradedStatus(repoPath string, openErr error) (*uiv1.GetStatusResponse, error) {
	kind := classifyDegrade(openErr)
	if kind == degradeNone {
		return nil, openErr
	}

	_, resolveErr := query.ResolveCodegraphDir(repoPath)
	storeExists := resolveErr == nil

	return &uiv1.GetStatusResponse{
		Initialized:        false,
		StoreExists:        storeExists,
		IndexingInProgress: kind == degradeIndexingInProgress,
	}, nil
}
