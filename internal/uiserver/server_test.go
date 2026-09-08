package uiserver

import (
	"context"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// copyGofixture and indexGofixture mirror internal/query/engine_test.go's
// copyFixture/indexFixture byte-for-byte (03-PATTERNS.md's "test
// scaffolding" convention) — reproduced here rather than imported because
// the originals are unexported in package query.
func copyGofixture(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "indexer", "testdata", "gofixture"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	dst := t.TempDir()

	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return dst
}

func indexGofixture(t *testing.T, dir string) {
	t.Helper()

	storeDir := filepath.Join(dir, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
}

// mustListen builds a live server over an indexed fixture rooted at
// repoPath, failing the test immediately on any Listen error.
func mustListen(t *testing.T, repoPath string) *Server {
	t.Helper()
	srv, err := Listen(Options{RepoPath: repoPath})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	return srv
}

// waitForConnectable retries a lightweight TCP dial against addr until it
// succeeds or the timeout elapses, absorbing the small scheduling gap
// between starting Serve in a goroutine and its Accept loop actually
// running.
func waitForConnectable(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("server at %q never became connectable", addr)
}

func TestListenPublishesABoundPortBeforeServe(t *testing.T) {
	srv := mustListen(t, t.TempDir())
	defer srv.Close()

	u, err := url.Parse(srv.URL())
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", srv.URL(), err)
	}
	if u.Scheme != "http" {
		t.Fatalf("URL scheme = %q, want http", u.Scheme)
	}
	if u.Hostname() != "127.0.0.1" {
		t.Fatalf("URL host = %q, want 127.0.0.1", u.Hostname())
	}
	if u.Port() == "" || u.Port() == "0" {
		t.Fatalf("URL port = %q, want a non-zero port", u.Port())
	}

	// No Serve call anywhere above this point: Listen alone must have
	// already made the port connectable.
	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatalf("net.Dial(%q) before Serve was ever called: %v", u.Host, err)
	}
	conn.Close()
}

func TestServeReturnsNilOnContextCancellation(t *testing.T) {
	srv := mustListen(t, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ctx)
	}()

	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v after cancellation, want nil", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("Serve did not return within the 5-second shutdown budget after cancellation — the serve goroutine may have leaked")
	}
}

// TestServeStopsPublisherOnAbnormalExit is WR-01's regression (06-REVIEW.md):
// Serve's errCh branch — taken when the underlying http.Server.Serve
// returns on its own, for a reason OTHER than ctx being cancelled by the
// caller — must still stop the live publisher, exactly like the ctx.Done()
// branch does. Before the fix, this branch returned directly with no call
// to stopPublisher(), leaking the fsnotify watcher, the debouncer, and the
// watch-loop goroutine.
func TestServeStopsPublisherOnAbnormalExit(t *testing.T) {
	srv := mustListen(t, t.TempDir())

	// Close the raw listener directly — NOT through Serve/Shutdown/Close —
	// so s.srv.Serve(s.ln) returns an error other than http.ErrServerClosed
	// entirely on its own, taking Serve's errCh branch rather than its
	// ctx.Done() branch. ctx is never cancelled, so only errCh can fire.
	if err := srv.ln.Close(); err != nil {
		t.Fatalf("srv.ln.Close(): %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(context.Background())
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after its listener closed on its own")
	}

	// Positive assertion that the publisher was actually stopped (rule
	// 84d1gfpywd — an upper bound or "nothing bad happened" proves
	// nothing): liveRegistry.Subscribe's own doc comment states a STOPPED
	// registry hands back a channel that is already closed. A live
	// (unstopped) registry would instead hand back an open, non-yielding
	// channel here, which the select below would time out on.
	ch, unsubscribe := srv.publisher.Subscribe(context.Background())
	defer unsubscribe()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("Subscribe's channel yielded a value instead of being closed — the publisher's registry was not stopped")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Subscribe's channel was neither closed nor received from — stopPublisher was never called on Serve's abnormal-exit branch (WR-01)")
	}
}

func TestUIServerServesGetStatusEndToEnd(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := mustListen(t, dir)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Serve(ctx) }()
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{}))
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !resp.Msg.GetInitialized() {
		t.Fatal("GetStatus: initialized = false, want true")
	}

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt (independent verification path): %v", err)
	}
	defer closer.Close()
	want, err := eng.Status(context.Background())
	if err != nil {
		t.Fatalf("eng.Status: %v", err)
	}
	if resp.Msg.GetNodeCount() != want.NodeCount {
		t.Fatalf("GetStatus node_count = %d, want %d (independently reported by Engine.Status for the same fixture)", resp.Msg.GetNodeCount(), want.NodeCount)
	}
}

// capturingTransport wraps an http.RoundTripper to (a) optionally
// override the outgoing request's Host header, exercising the guard from
// the client side without hand-rolling the Connect wire protocol, and
// (b) record the resulting response's status and Content-Type for the
// test to assert on afterward — the generated Connect client already
// consumes/decodes the body, so this is the seam that lets a test
// inspect the raw HTTP response Connect's own codec would otherwise hide.
type capturingTransport struct {
	rt              http.RoundTripper
	hostOverride    string
	lastStatus      int
	lastContentType string
}

func (c *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if c.hostOverride != "" {
		req.Host = c.hostOverride
	}
	resp, err := c.rt.RoundTrip(req)
	if resp != nil {
		c.lastStatus = resp.StatusCode
		c.lastContentType = resp.Header.Get("Content-Type")
	}
	return resp, err
}

// bodyLooksLikeAConnectEnvelope reports whether a response with the
// given Content-Type looks like it was produced by the connect-go
// handler rather than a plain net/http rejection: Connect's unary codecs
// set Content-Type to "application/proto" (binary) or "application/json"
// (its JSON codec, used for both success and its own error-JSON shape).
// http.Error's plain-text rejection sets "text/plain; charset=utf-8" and
// never either of those.
func bodyLooksLikeAConnectEnvelope(contentType string) bool {
	return strings.HasPrefix(contentType, "application/proto") ||
		strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/connect")
}

func TestUIServerRejectsForeignHostEndToEnd(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := mustListen(t, dir)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Serve(ctx) }()
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))

	// Positive control (required before the absence claim can mean
	// anything): a genuine, successful SAME-ORIGIN response against this
	// same live server DOES satisfy bodyLooksLikeAConnectEnvelope. A
	// predicate that always reports false, or one whose framing
	// assumption is wrong, would otherwise pass the rejection assertion
	// below for the wrong reason.
	okTransport := &capturingTransport{rt: http.DefaultTransport}
	okClient := uiv1connect.NewUIServiceClient(&http.Client{Transport: okTransport}, srv.URL())
	if _, err := okClient.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{})); err != nil {
		t.Fatalf("baseline same-origin GetStatus failed: %v", err)
	}
	if !bodyLooksLikeAConnectEnvelope(okTransport.lastContentType) {
		t.Fatalf("positive control failed: a genuine successful response (Content-Type %q) does not satisfy bodyLooksLikeAConnectEnvelope — the predicate itself is broken, so the rejection assertion below would prove nothing", okTransport.lastContentType)
	}

	// The actual claim: a foreign Host is rejected with 403, and the
	// response body carries NO serialized Connect envelope — both
	// halves required.
	badTransport := &capturingTransport{rt: http.DefaultTransport, hostOverride: "evil.com"}
	badClient := uiv1connect.NewUIServiceClient(&http.Client{Transport: badTransport}, srv.URL())
	if _, err := badClient.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{})); err == nil {
		t.Fatal("GetStatus with Host: evil.com succeeded, want an error")
	}
	if badTransport.lastStatus != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", badTransport.lastStatus, http.StatusForbidden)
	}
	if bodyLooksLikeAConnectEnvelope(badTransport.lastContentType) {
		t.Fatalf("403 response (Content-Type %q) looks like a Connect envelope — the guard's rejection leaked a framed response body", badTransport.lastContentType)
	}
}

func TestUIServerHoldsNoStoreHandleBetweenCalls(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := mustListen(t, dir)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Serve(ctx) }()
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	for i := 0; i < 2; i++ {
		if _, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{})); err != nil {
			t.Fatalf("GetStatus call %d: %v", i+1, err)
		}
	}

	storeDir := filepath.Join(dir, ".codegraph", "store")
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open after two RPC calls: %v — the server is holding a store handle open between calls (SRV-04 violation)", err)
	}
	store.Close()
}

func TestUIServiceHoldsNoStoreTypedField(t *testing.T) {
	// uiserver.Options carries EXACTLY RepoPath and Addr — asserted here
	// (rather than as a separate function/subtest) to keep this task's
	// PASS-line floor at exactly 6, zero headroom by design. A reflected
	// field-NAME set equality with a length-2 check, so a reintroduced
	// browser field (review MEDIUM 01-06: ownership of the browser
	// launch) fails this assertion.
	optsTyp := reflect.TypeOf(Options{})
	gotOptsFields := make(map[string]struct{}, optsTyp.NumField())
	for i := 0; i < optsTyp.NumField(); i++ {
		gotOptsFields[optsTyp.Field(i).Name] = struct{}{}
	}
	wantOptsFields := map[string]struct{}{"RepoPath": {}, "Addr": {}}
	if len(gotOptsFields) != len(wantOptsFields) {
		t.Fatalf("Options field set = %v, want %v", gotOptsFields, wantOptsFields)
	}
	for name := range gotOptsFields {
		if _, ok := wantOptsFields[name]; !ok {
			t.Fatalf("Options has an unexpected field %q, want only %v", name, wantOptsFields)
		}
	}

	typ := reflect.TypeOf(uiService{})

	got := make(map[string]struct{}, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		got[typ.Field(i).Type.String()] = struct{}{}
	}

	want := map[string]struct{}{"string": {}, "*uiserver.livePublisher": {}}
	if len(got) != len(want) {
		t.Fatalf("uiService field type set = %v, want %v", got, want)
	}
	for typeName := range got {
		if _, ok := want[typeName]; !ok {
			t.Fatalf("uiService has an unexpected field of type %q, want only %v", typeName, want)
		}
	}

	forbidden := []string{"*query.Engine", "graphstore.GraphStore", "graphstore.Reader"}
	for _, f := range forbidden {
		if _, ok := got[f]; ok {
			t.Fatalf("uiService retains a field of type %q — SRV-04 forbids caching a store handle across calls", f)
		}
	}
}

// TestListenSetsEveryServerTimeout proves WR-03's mitigation (gosec
// G112): the http.Server Listen builds has all four timeouts set to the
// named constants, none left at net/http's "no limit" zero value.
//
// The set is asserted by NAME and by VALUE, one subtest per field, so a
// future edit that drops one — the realistic regression, since three
// correct fields make the fourth easy to miss — fails on that field
// rather than passing because the other three are still set. The
// ordering assertion below is the reason writeTimeout is not merely
// "some positive duration": a write budget at or under the read budget
// would cut off a legitimate slow response.
func TestListenSetsEveryServerTimeout(t *testing.T) {
	srv := mustListen(t, t.TempDir())
	defer srv.Close()

	cases := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"ReadHeaderTimeout", srv.srv.ReadHeaderTimeout, readHeaderTimeout},
		{"ReadTimeout", srv.srv.ReadTimeout, readTimeout},
		{"WriteTimeout", srv.srv.WriteTimeout, writeTimeout},
		{"IdleTimeout", srv.srv.IdleTimeout, idleTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got == 0 {
				t.Fatalf("http.Server.%s is 0 — net/http's zero value means NO LIMIT, so a client that never completes its request pins a goroutine and an fd indefinitely (gosec G112)", tc.name)
			}
			if tc.got != tc.want {
				t.Fatalf("http.Server.%s = %v, want the named constant %v", tc.name, tc.got, tc.want)
			}
		})
	}

	if writeTimeout <= readTimeout {
		t.Fatalf("writeTimeout (%v) is not greater than readTimeout (%v) — the response budget must exceed the request budget, or a legitimate slow response is cut off", writeTimeout, readTimeout)
	}
	if readHeaderTimeout >= readTimeout {
		t.Fatalf("readHeaderTimeout (%v) is not less than readTimeout (%v) — the header budget is a subset of the whole-request budget", readHeaderTimeout, readTimeout)
	}
}
