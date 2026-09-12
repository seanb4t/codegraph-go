package cli

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/pflag"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

func TestUICommandFlagSetIsExactlyTheFourUIFlags(t *testing.T) {
	cmd := newUiCmd()

	got := make(map[string]struct{})
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		got[f.Name] = struct{}{}
	})

	want := map[string]struct{}{"path": {}, "no-open": {}, "editor-url": {}, "no-editor-url": {}}
	if len(got) != len(want) {
		t.Fatalf("ui command flag set = %v, want %v", got, want)
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Fatalf("ui command has an unexpected flag %q, want only %v", name, want)
		}
	}
}

func TestShouldOpenBrowserSuppression(t *testing.T) {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	cases := []struct {
		name   string
		noOpen bool
		ci     string
		out    io.Writer
		want   bool
	}{
		{name: "--no-open suppresses", noOpen: true, out: devNull, want: false},
		{name: "CI set suppresses", ci: "1", out: devNull, want: false},
		{name: "non-character-device stdout suppresses", out: &bytes.Buffer{}, want: false},
		{name: "none of those: opens", out: devNull, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CI", tc.ci)

			cmd := newUiCmd()
			cmd.SetOut(tc.out)

			got := shouldOpenBrowser(tc.noOpen, cmd)
			if got != tc.want {
				t.Fatalf("shouldOpenBrowser(noOpen=%v) with CI=%q = %v, want %v", tc.noOpen, tc.ci, got, tc.want)
			}
		})
	}
}

// copyGofixtureForCLI and indexGofixtureForCLI mirror
// internal/query/engine_test.go's copyFixture/indexFixture byte-for-byte
// (03-PATTERNS.md's "test scaffolding" convention), reproduced here
// (rather than in internal/uiserver, which has its own copy) because
// each package's copy is unexported and this repo's own precedent
// duplicates this exact helper per-package instead of exporting it.
func copyGofixtureForCLI(t *testing.T) string {
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

func indexGofixtureForCLI(t *testing.T, dir string) {
	t.Helper()

	storeDir := filepath.Join(dir, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
}

// TestUICommandPrintsConnectableURLBeforeServing proves the whole
// Listen-then-publish-then-serve order end to end through the real
// cobra command: the printed URL is usable the instant it appears on
// stdout, and cancelling the command's context makes RunE return nil
// within the shutdown budget.
func TestUICommandPrintsConnectableURLBeforeServing(t *testing.T) {
	dir := copyGofixtureForCLI(t)
	indexGofixtureForCLI(t, dir)

	// io.Pipe, not a plain bytes.Buffer: RunE's Fprintln happens on the
	// command's own goroutine while this test reads on a different one,
	// and a plain buffer would race under -race. io.Pipe is a genuine
	// synchronized channel of bytes.
	pr, pw := io.Pipe()

	cmd := newUiCmd()
	cmd.SetOut(pw)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--path", dir, "--no-open"})

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)
	go func() {
		runErr <- cmd.ExecuteContext(ctx)
	}()

	scanner := bufio.NewScanner(pr)
	if !scanner.Scan() {
		t.Fatalf("no URL line printed on stdout: %v", scanner.Err())
	}
	printedURL := strings.TrimSpace(scanner.Text())
	if !strings.HasPrefix(printedURL, "http://127.0.0.1:") {
		t.Fatalf("printed URL = %q, want a http://127.0.0.1:<port> line", printedURL)
	}

	// Connect at the EXACT printed URL while the command is still
	// running — a URL that could not be connected to fails this test,
	// which is the whole point of the Listen/Serve split.
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, printedURL)
	if _, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{})); err != nil {
		t.Fatalf("GetStatus at the printed URL %q while the command is still running: %v", printedURL, err)
	}

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("cmd.ExecuteContext returned %v after cancellation, want nil", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("cmd.ExecuteContext did not return within the shutdown budget after context cancellation")
	}
}

// TestUICommandEditorURLFlagReachesGetEditorLink proves --editor-url and
// --no-editor-url are wired all the way through resolveEditorLink into
// uiserver.Options.EditorLink, observed via a real GetEditorLink call
// against the running command.
func TestUICommandEditorURLFlagReachesGetEditorLink(t *testing.T) {
	t.Run("--editor-url sets FLAG provenance", func(t *testing.T) {
		dir := copyGofixtureForCLI(t)
		indexGofixtureForCLI(t, dir)

		pr, pw := io.Pipe()
		cmd := newUiCmd()
		cmd.SetOut(pw)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"--path", dir, "--no-open", "--editor-url", "vscode://file/{path}:{line}:{col}"})

		ctx, cancel := context.WithCancel(context.Background())
		runErr := make(chan error, 1)
		go func() { runErr <- cmd.ExecuteContext(ctx) }()

		scanner := bufio.NewScanner(pr)
		if !scanner.Scan() {
			t.Fatalf("no URL line printed on stdout: %v", scanner.Err())
		}
		printedURL := strings.TrimSpace(scanner.Text())

		client := uiv1connect.NewUIServiceClient(http.DefaultClient, printedURL)
		resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go"}))
		if err != nil {
			t.Fatalf("GetEditorLink at %q: %v", printedURL, err)
		}
		if got := resp.Msg.GetAvailability(); got != uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_BUILDABLE {
			t.Fatalf("availability = %v, want BUILDABLE", got)
		}
		if got := resp.Msg.GetDefaultSource(); got != uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_FLAG {
			t.Fatalf("default_source = %v, want FLAG", got)
		}
		if got := resp.Msg.GetUrl(); !strings.HasPrefix(got, "vscode://file/") {
			t.Fatalf("url = %q, want prefix %q", got, "vscode://file/")
		}

		cancel()
		select {
		case err := <-runErr:
			if err != nil {
				t.Fatalf("cmd.ExecuteContext returned %v after cancellation, want nil", err)
			}
		case <-time.After(6 * time.Second):
			t.Fatal("cmd.ExecuteContext did not return within the shutdown budget after context cancellation")
		}
	})

	t.Run("--no-editor-url disables with the operator reason", func(t *testing.T) {
		dir := copyGofixtureForCLI(t)
		indexGofixtureForCLI(t, dir)

		pr, pw := io.Pipe()
		cmd := newUiCmd()
		cmd.SetOut(pw)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"--path", dir, "--no-open", "--no-editor-url"})

		ctx, cancel := context.WithCancel(context.Background())
		runErr := make(chan error, 1)
		go func() { runErr <- cmd.ExecuteContext(ctx) }()

		scanner := bufio.NewScanner(pr)
		if !scanner.Scan() {
			t.Fatalf("no URL line printed on stdout: %v", scanner.Err())
		}
		printedURL := strings.TrimSpace(scanner.Text())

		client := uiv1connect.NewUIServiceClient(http.DefaultClient, printedURL)
		resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go"}))
		if err != nil {
			t.Fatalf("GetEditorLink at %q: %v", printedURL, err)
		}
		if got := resp.Msg.GetAvailability(); got != uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_NO_TEMPLATE {
			t.Fatalf("availability = %v, want NO_TEMPLATE", got)
		}
		if got := resp.Msg.GetReason(); !strings.Contains(got, "disabled by operator") {
			t.Fatalf("reason = %q, want it to contain %q", got, "disabled by operator")
		}
		if got := resp.Msg.GetDefaultSource(); got != uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_DISABLED {
			t.Fatalf("default_source = %v, want DISABLED", got)
		}

		cancel()
		select {
		case err := <-runErr:
			if err != nil {
				t.Fatalf("cmd.ExecuteContext returned %v after cancellation, want nil", err)
			}
		case <-time.After(6 * time.Second):
			t.Fatal("cmd.ExecuteContext did not return within the shutdown budget after context cancellation")
		}
	})
}

// TestUICommandRefusesMalformedEditorURLBeforeBinding proves D-17
// through the real command: a malformed --editor-url value makes
// ExecuteContext return an error naming --editor-url and the offending
// scheme, and prints NOTHING to stdout — the port is never bound.
func TestUICommandRefusesMalformedEditorURLBeforeBinding(t *testing.T) {
	dir := copyGofixtureForCLI(t)
	indexGofixtureForCLI(t, dir)

	pr, pw := io.Pipe()
	cmd := newUiCmd()
	cmd.SetOut(pw)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--path", dir, "--no-open", "--editor-url", "javascript:{path}"})

	runErr := make(chan error, 1)
	go func() { runErr <- cmd.ExecuteContext(context.Background()) }()

	var err error
	select {
	case err = <-runErr:
	case <-time.After(6 * time.Second):
		t.Fatal("cmd.ExecuteContext did not return within the budget")
	}
	if err == nil {
		t.Fatal("cmd.ExecuteContext returned nil, want a refusal naming --editor-url")
	}
	if !strings.Contains(err.Error(), "--editor-url") || !strings.Contains(strings.ToLower(err.Error()), "scheme") {
		t.Fatalf("error = %q, want it to contain %q and %q", err.Error(), "--editor-url", "scheme")
	}

	// Confirm zero bytes were ever written to stdout: close the write
	// end now that RunE has returned, then drain the read end with a
	// bounded read — a hung read would mean something was written
	// without a trailing newline the scanner-based tests above rely on.
	pw.Close()
	buf := make([]byte, 1)
	n, readErr := pr.Read(buf)
	if n != 0 {
		t.Fatalf("stdout carried %d byte(s), want 0 (the port must never be bound on a startup refusal)", n)
	}
	if readErr == nil {
		t.Fatal("expected io.EOF from the closed pipe, got nil")
	}
}
