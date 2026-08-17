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

	out, err := engine.Node("", "main.go", nil)
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

	_, err = engine.Node("", "escape-link/secret.txt", nil)
	if err == nil {
		t.Fatal("Node: expected an error for a path escaping the repo root via a symlink, got nil")
	}
}

// TestIsGeneratedFile pins D-07's byte-for-byte transcription of the
// documented GENERATED_PATTERNS regex list (RESEARCH §7) — every pattern gets one matching and (where
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
// tie-break (RESEARCH Pattern 2 — the documented row order is non-deterministic
// across re-indexes, so this is an intentional Go-side determinism
// improvement, not a byte-for-byte transcription).
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

// TestNarrowNeverEmpty pins NODE-03: file/line narrowing is a PURE
// in-memory filter over an already-enumerated node slice (RESEARCH §9)
// that never empties the working set — a fileHint or lineHint that
// matches nothing simply leaves the set (or the prior stage's result)
// unchanged rather than erroring or returning zero matches.
func TestNarrowNeverEmpty(t *testing.T) {
	mk := func(id, file string, start, end int32) *schema.Node {
		return &schema.Node{Id: id, Name: "Dup", FilePath: file, StartLine: start, EndLine: end}
	}
	a := mk("a", "pkg/a.go", 10, 20)
	b := mk("b", "pkg/b.go", 30, 40)
	c := mk("c", "pkg/c.go", 50, 60)
	matches := []*schema.Node{a, b, c}

	t.Run("fileHint matching exactly one narrows to that one", func(t *testing.T) {
		got := narrowNodeMatches(matches, "pkg/b.go", nil)
		if len(got) != 1 || got[0] != b {
			t.Fatalf("narrowNodeMatches(fileHint=pkg/b.go): got %d matches, want [b]", len(got))
		}
	})

	t.Run("fileHint matching zero retains all matches (never empties)", func(t *testing.T) {
		got := narrowNodeMatches(matches, "nope/nowhere.go", nil)
		if len(got) != 3 {
			t.Fatalf("narrowNodeMatches(fileHint=no-match): got %d matches, want 3 (never empty)", len(got))
		}
	})

	t.Run("lineHint inside a def's [startLine,endLine] narrows to that def", func(t *testing.T) {
		line := 35
		got := narrowNodeMatches(matches, "", &line)
		if len(got) != 1 || got[0] != b {
			t.Fatalf("narrowNodeMatches(lineHint=35): got %d matches, want [b]", len(got))
		}
	})

	t.Run("lineHint inside no def falls back to the single nearest by startLine distance", func(t *testing.T) {
		line := 22 // |10-22|=12 (a), |30-22|=8 (b), |50-22|=28 (c) -> nearest is b
		got := narrowNodeMatches(matches, "", &line)
		if len(got) != 1 || got[0] != b {
			t.Fatalf("narrowNodeMatches(lineHint=22, no containing def): got %d matches, want nearest [b]", len(got))
		}
	})

	t.Run("both hints absent returns the full set unchanged", func(t *testing.T) {
		got := narrowNodeMatches(matches, "", nil)
		if len(got) != 3 {
			t.Fatalf("narrowNodeMatches(no hints): got %d matches, want 3", len(got))
		}
	})

	t.Run("a single match is returned unchanged regardless of hints", func(t *testing.T) {
		got := narrowNodeMatches([]*schema.Node{a}, "no/such/file.go", nil)
		if len(got) != 1 || got[0] != a {
			t.Fatalf("narrowNodeMatches(single match): got %d matches, want the single match unchanged", len(got))
		}
	})
}
