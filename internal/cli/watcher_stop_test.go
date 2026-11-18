package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/daemon"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// refusalFixture returns a directory with an empty .codegraph, enough for
// daemon.New, and the refusal a watcher returns when that index changes.
func refusalFixture(t *testing.T) (string, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, codegraphDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, &query.IndexOutsideRootError{
		Link:   filepath.Join(dir, codegraphDirName),
		Target: "/elsewhere",
		Root:   dir,
	}
}

// TestServeWatchStartPrintsIndexChange covers "serve keeps serving after
// its watcher stops": the stop goes to stderr, the watcher goroutine ends,
// and nothing cancels the context of the watcher until serve itself calls
// cancel.
func TestServeWatchStartPrintsIndexChange(t *testing.T) {
	dir, refusal := refusalFixture(t)
	var watchCtx context.Context
	saved := runDaemonWithRetry
	t.Cleanup(func() { runDaemonWithRetry = saved })
	runDaemonWithRetry = func(ctx context.Context, _ *daemon.Daemon, _ time.Duration, _ func()) error {
		watchCtx = ctx
		return refusal
	}

	var stderr bytes.Buffer
	cancel, done := serveWatchStart(dir, true, nil, false, true, indexer.Options{Quiet: true}, &stderr, nil)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the watcher goroutine did not end after its watcher stopped")
	}

	if want := "codegraph serve: watcher: " + refusal.Error() + "\n"; !strings.Contains(stderr.String(), want) {
		t.Fatalf("stderr = %q, want the line %q", stderr.String(), want)
	}
	if watchCtx == nil || watchCtx.Err() != nil {
		t.Fatalf("the watcher's context ended (%v) before serve cancelled it", watchCtx)
	}
	cancel()
	if watchCtx.Err() == nil {
		t.Fatal("cancel did not end the watcher's context")
	}
}

// TestDaemonStartReturnsIndexChange covers "daemon start reports the
// stop": the command returns the watcher's error, which main prints before
// it exits 1.
func TestDaemonStartReturnsIndexChange(t *testing.T) {
	dir, refusal := refusalFixture(t)
	saved := runDaemon
	t.Cleanup(func() { runDaemon = saved })
	runDaemon = func(*daemon.Daemon, context.Context) error { return refusal }

	_, _, err := execCmd("daemon", "start", "-p", dir)
	if !errors.Is(err, query.ErrIndexOutsideRoot) {
		t.Fatalf("daemon start = %v, want query.ErrIndexOutsideRoot", err)
	}
}
