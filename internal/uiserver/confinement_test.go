package uiserver

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"connectrpc.com/connect"

	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// SRV-05's actual gap was never the confinement gate itself — that
// already exists and is already shared (internal/query/node.go:33-79's
// resolveSourcePath, reused via readSourceFile/SourceFor by both the MCP
// path and internal/uiserver/handlers.go's GetNodeDetail). The gap was
// that nothing in this package asserted the refusal crosses the wire:
// before this file, internal/uiserver/*_test.go had zero occurrences of
// "confinement" or "escapes the repo root" (03-CONTEXT.md D-02).
//
// Per rule 84d1gfpywd, a guard that only asserts refusal passes
// vacuously the moment the RPC starts refusing everything — so
// TestGetNodeDetailPathConfinementAtRPCBoundary pairs every refusal case
// with an in-repo positive control, asserted FIRST, from the SAME live
// service.

// setupSymlinkEscape creates a symlink inside repoDir pointing at a fresh
// t.TempDir() OUTSIDE repoDir, writes a file under that outside target,
// and returns the repo-relative path a client would send to reach it
// through the link. This is the ONLY case in this file that exercises
// resolveSourcePath's post-symlink-resolution re-verification (WR-03):
// the string-level Clean/Rel check alone sees only the in-repo link text
// and would let it through.
//
// ok is false ONLY when os.Symlink failed for a reason that means the
// platform genuinely cannot create a symlink here: an EPERM or
// ErrUnsupported result, or — on windows specifically — the standard
// unprivileged-account rejection. Any OTHER failure, and every failure at
// all on linux or darwin (the two platforms this repo's CI runs), is
// t.Fatalf: this is the only RPC-boundary coverage of WR-03 and a
// permissive skip would let the most important case quietly disappear on
// the platforms that matter.
func setupSymlinkEscape(t *testing.T, repoDir string) (relPath string, ok bool) {
	t.Helper()

	outsideDir := t.TempDir()
	target := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(target, []byte("outside content — must never be reachable through the repo\n"), 0o644); err != nil {
		t.Fatalf("write outside target file: %v", err)
	}

	const linkName = "escape-link"
	linkPath := filepath.Join(repoDir, linkName)
	err := os.Symlink(outsideDir, linkPath)
	if err == nil {
		return filepath.ToSlash(filepath.Join(linkName, "secret.txt")), true
	}

	skippable := errors.Is(err, syscall.EPERM) || errors.Is(err, errors.ErrUnsupported)
	if !skippable && runtime.GOOS == "windows" {
		skippable = strings.Contains(err.Error(), "A required privilege is not held by the client")
	}
	if skippable {
		t.Skipf("os.Symlink(%q, %q) unsupported on %s: %v", outsideDir, linkPath, runtime.GOOS, err)
		return "", false
	}
	t.Fatalf("os.Symlink(%q, %q): %v — this platform (%s) is expected to support symlinks; the WR-03 post-symlink-escape case must not be silently skipped here", outsideDir, linkPath, runtime.GOOS, err)
	return "", false // unreachable
}

// TestGetNodeDetailPathConfinementAtRPCBoundary drives GetNodeDetail
// through a real uiv1connect client against a real listener — never by
// calling the handler struct directly, because the point of this test is
// the boundary, not the function.
//
// The positive control (an in-repo path that returns real source,
// byte-equal to an independent os.ReadFile) is asserted FIRST: a service
// broken for every input must fail there, not read as four successful
// refusals below it.
func TestGetNodeDetailPathConfinementAtRPCBoundary(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	t.Run("in-repo control", func(t *testing.T) {
		const relPath = "main.go"

		want, err := os.ReadFile(filepath.Join(dir, relPath))
		if err != nil {
			t.Fatalf("independent os.ReadFile(%q): %v", relPath, err)
		}

		resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{File: relPath}))
		if err != nil {
			t.Fatalf("GetNodeDetail(file=%q): %v", relPath, err)
		}
		if got := resp.Msg.GetMode(); got != uiv1.NodeDetailMode_NODE_DETAIL_MODE_FILE {
			t.Fatalf("GetNodeDetail(file=%q) mode = %v, want NODE_DETAIL_MODE_FILE", relPath, got)
		}
		src := resp.Msg.GetSource()
		if src == nil || len(src.GetContent()) == 0 {
			t.Fatalf("GetNodeDetail(file=%q): source is nil or has no content, want the real file bytes", relPath)
		}
		if !bytes.Equal(src.GetContent(), want) {
			t.Fatalf("GetNodeDetail(file=%q) source.content (%d bytes) does not byte-equal the independent os.ReadFile (%d bytes) — the refusals below would prove nothing if this service cannot serve a legitimate file", relPath, len(src.GetContent()), len(want))
		}
	})

	type refusalCase struct {
		name     string
		file     string
		wantFrag string
	}

	cases := []refusalCase{
		{name: "escape", file: "../outside.txt", wantFrag: "escapes the repo root"},
		{name: "absolute", file: "/etc/passwd", wantFrag: "is not allowed"},
		// empty: no message fragment asserted here — code alone, per this
		// task's own behavior spec. Task 2 asserts what the message DOES
		// and does NOT contain.
		{name: "empty", file: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{File: c.file}))
			if err == nil {
				t.Fatalf("GetNodeDetail(file=%q) succeeded, want a refusal", c.file)
			}
			if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
				t.Fatalf("GetNodeDetail(file=%q): code = %v, want CodeInvalidArgument (err=%q)", c.file, code, err.Error())
			}
			if c.wantFrag != "" && !strings.Contains(err.Error(), c.wantFrag) {
				t.Fatalf("GetNodeDetail(file=%q): error = %q, want it to contain %q", c.file, err.Error(), c.wantFrag)
			}
		})
	}

	// The symlink case gets its own t.Run rather than joining the table
	// above: setupSymlinkEscape may call t.Skipf, and that must localize
	// to this one subtest, not abort the whole parent test function
	// before the table above has a chance to run.
	t.Run("symlink", func(t *testing.T) {
		relPath, ok := setupSymlinkEscape(t, dir)
		if !ok {
			return // setupSymlinkEscape already called t.Skipf or t.Fatalf
		}
		_, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{File: relPath}))
		if err == nil {
			t.Fatalf("GetNodeDetail(file=%q) succeeded, want a refusal", relPath)
		}
		if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
			t.Fatalf("GetNodeDetail(file=%q): code = %v, want CodeInvalidArgument (err=%q)", relPath, code, err.Error())
		}
		if !strings.Contains(err.Error(), "escapes the repo root") {
			t.Fatalf("GetNodeDetail(file=%q): error = %q, want it to contain %q", relPath, err.Error(), "escapes the repo root")
		}
	})
}

// TestGetNodeDetailConfinementRefusalDoesNotLeakHostPath asserts T-03-06
// (03-CONTEXT.md, threat register): no confinement refusal crossing the
// wire names the fixture's absolute host checkout path.
//
// A negative containment check is meaningless on its own — it passes
// trivially against an empty string (rule 84d1gfpywd applied to an
// information-disclosure guard). Every case here is paired with a
// positive containment check proving the message was actually inspected:
// the caller's own submitted path value must be present in the same
// message the host path must be absent from.
//
// This test builds its own fixture/server/client rather than sharing
// Task 1's (03-02-PLAN.md Task 2's declined-merge rationale): the two
// tests assert different properties — Task 1 the code and mode, this one
// what the message may and may not contain — and merging them would make
// a single failure ambiguous about which invariant broke.
func TestGetNodeDetailConfinementRefusalDoesNotLeakHostPath(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	assertNoHostLeak := func(t *testing.T, file string) {
		t.Helper()

		_, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{File: file}))
		if err == nil {
			t.Fatalf("GetNodeDetail(file=%q) succeeded, want a refusal", file)
		}
		msg := err.Error()
		if msg == "" {
			t.Fatalf("GetNodeDetail(file=%q): refusal message is empty", file)
		}
		if strings.Contains(msg, dir) {
			t.Fatalf("GetNodeDetail(file=%q): refusal message %q contains the fixture's absolute host checkout path %q", file, msg, dir)
		}

		if file == "" {
			// The submitted value is the empty string, and every string
			// contains the empty string — a positive containment check
			// against it would be a tautology (this task's own action
			// text). Assert instead that the message is populated and
			// names the file-path validation it hit.
			if !strings.Contains(msg, "file path") {
				t.Fatalf("GetNodeDetail(file=%q): refusal message %q does not mention the empty file path", file, msg)
			}
			return
		}
		if !strings.Contains(msg, file) {
			t.Fatalf("GetNodeDetail(file=%q): refusal message %q does not contain the caller's own submitted path — the negative host-path check above would be unproven without this", file, msg)
		}
	}

	t.Run("escape", func(t *testing.T) { assertNoHostLeak(t, "../outside.txt") })
	t.Run("absolute", func(t *testing.T) { assertNoHostLeak(t, "/etc/passwd") })
	t.Run("empty", func(t *testing.T) { assertNoHostLeak(t, "") })
	t.Run("symlink", func(t *testing.T) {
		relPath, ok := setupSymlinkEscape(t, dir)
		if !ok {
			return // setupSymlinkEscape already called t.Skipf or t.Fatalf
		}
		assertNoHostLeak(t, relPath)
	})
}
