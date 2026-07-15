package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestResolveSourcePathAllowsRegularFileInRepo is the control case for
// TestResolveSourcePathRejectsSymlinkEscape: a plain, non-symlinked
// in-repo file must still be readable via Node's file mode.
func TestResolveSourcePathAllowsRegularFileInRepo(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	out, err := engine.Node("", "main.go")
	if err != nil {
		t.Fatalf("Node: unexpected error reading a regular in-repo file: %v", err)
	}
	if out == "" {
		t.Fatal("Node: expected non-empty rendered source for main.go")
	}
}

// TestResolveSourcePathRejectsSymlinkEscape pins WR-03: a symlink inside
// the repo root that points at a file outside it must be rejected, not
// silently followed. The string-level Clean/Rel check alone cannot catch
// this — the symlink's own path text is entirely inside the repo — so
// this specifically exercises the filepath.EvalSymlinks re-verification
// step.
func TestResolveSourcePathRejectsSymlinkEscape(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	// Create a directory OUTSIDE the repo root with a secret file, then a
	// symlink INSIDE the repo root pointing at it.
	outside := t.TempDir()
	secretPath := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("outside-repo-secret"), 0o644); err != nil {
		t.Fatalf("write secret file: %v", err)
	}
	linkPath := filepath.Join(dir, "escape-link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err = engine.Node("", "escape-link/secret.txt")
	if err == nil {
		t.Fatal("Node: expected an error for a path escaping the repo root via a symlink, got nil")
	}
}

// TestIsGeneratedFile pins D-07's verbatim port of TS's GENERATED_PATTERNS
// regex list (RESEARCH §7) — every pattern gets one matching and (where
// meaningful) one non-matching case, so a future edit that drops or
// mistypes a pattern fails loudly here rather than silently degrading
// NODE-01's generated-files-last sort.
func TestIsGeneratedFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"foo.pb.go", true},
		{"foo.pulsar.go", true},
		{"foo_grpc.pb.go", true},
		{"foo_mock.go", true},
		{"foo_mocks.go", true},
		{"mock_foo.go", true},
		{"internal/mock_foo.go", false}, // ^ anchors the whole path, not the basename
		{"foo.generated.js", true},
		{"foo.generated.jsx", true},
		{"foo.generated.ts", true},
		{"foo.generated.tsx", true},
		{"foo.gen.js", true},
		{"foo.gen.tsx", true},
		{"foo.pb.js", true},
		{"foo.pb.ts", true},
		{"foo_pb.js", true},
		{"foo_pb.ts", true},
		{"foo_grpc_pb.js", true},
		{"foo.min.js", true},
		{"foo.min.mjs", true},
		{"foo_pb2.py", true},
		{"foo_pb2_grpc.py", true},
		{"foo_pb2.pyi", true},
		{"foo.pb.cc", true},
		{"foo.pb.h", true},
		{"foo.g.cs", true},
		{"FooGrpc.cs", true},
		{"FooOuterClass.java", true},
		{"FooGrpc.java", true},
		{"foo.pb.swift", true},
		{"foo.g.dart", true},
		{"foo.freezed.dart", true},
		{"foo.pb.dart", true},
		{"foo.pbgrpc.dart", true},
		{"foo.chopper.dart", true},
		{"foo.generated.rs", true},
		{"handler.go", false},
		{"handler_test.go", false},
		{"internal/cli/finish.go", false},
	}

	for _, c := range cases {
		if got := isGeneratedFile(c.path); got != c.want {
			t.Errorf("isGeneratedFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// TestNodeMultiDef pins NODE-01: enumerateSymbolDefs collects EVERY node
// whose Name matches (a full scan, D-03's base), sorted generated-files-
// last (D-07, primary key) with lowest-Id as the documented secondary
// tie-break (RESEARCH Pattern 2 — TS's own row order is non-deterministic
// across re-indexes, so this is an intentional Go-side determinism
// improvement, not a byte-for-byte TS port).
func TestNodeMultiDef(t *testing.T) {
	t.Run("two same-named nodes in different non-generated files are both returned", func(t *testing.T) {
		nodes := map[string]*schema.Node{
			"h1": {Id: "h1", Name: "Handler", Kind: "function", FilePath: "a/handler.go"},
			"h2": {Id: "h2", Name: "Handler", Kind: "function", FilePath: "b/handler.go"},
		}
		e := New(&traverseFakeReader{nodes: nodes})

		got, err := e.enumerateSymbolDefs("Handler")
		if err != nil {
			t.Fatalf("enumerateSymbolDefs(Handler): unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("enumerateSymbolDefs(Handler): got %d matches, want 2", len(got))
		}
	})

	t.Run("a generated-file definition sorts after a plain-file definition with the same name", func(t *testing.T) {
		nodes := map[string]*schema.Node{
			"a-gen":   {Id: "a-gen", Name: "Handler", Kind: "function", FilePath: "b/handler.pb.go"},
			"z-plain": {Id: "z-plain", Name: "Handler", Kind: "function", FilePath: "a/handler.go"},
		}
		e := New(&traverseFakeReader{nodes: nodes})

		got, err := e.enumerateSymbolDefs("Handler")
		if err != nil {
			t.Fatalf("enumerateSymbolDefs(Handler): unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("enumerateSymbolDefs(Handler): got %d matches, want 2", len(got))
		}
		if got[0].FilePath != "a/handler.go" || got[1].FilePath != "b/handler.pb.go" {
			t.Fatalf("enumerateSymbolDefs(Handler): got order [%s, %s], want plain file first, generated file last", got[0].FilePath, got[1].FilePath)
		}
	})

	t.Run("a single node named X resolves to a single-element slice", func(t *testing.T) {
		nodes := map[string]*schema.Node{
			"x1": {Id: "x1", Name: "X", Kind: "function", FilePath: "a.go"},
		}
		e := New(&traverseFakeReader{nodes: nodes})

		got, err := e.enumerateSymbolDefs("X")
		if err != nil {
			t.Fatalf("enumerateSymbolDefs(X): unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("enumerateSymbolDefs(X): got %d matches, want 1", len(got))
		}
	})

	t.Run("a tie among non-generated files breaks on the lowest Id", func(t *testing.T) {
		nodes := map[string]*schema.Node{
			"z9": {Id: "z9", Name: "Dup", Kind: "function", FilePath: "a.go"},
			"a1": {Id: "a1", Name: "Dup", Kind: "function", FilePath: "b.go"},
		}
		e := New(&traverseFakeReader{nodes: nodes})

		got, err := e.enumerateSymbolDefs("Dup")
		if err != nil {
			t.Fatalf("enumerateSymbolDefs(Dup): unexpected error: %v", err)
		}
		if len(got) != 2 || got[0].Id != "a1" {
			t.Fatalf("enumerateSymbolDefs(Dup): got first match Id %q, want lowest-Id \"a1\" first", got[0].Id)
		}
	})
}
