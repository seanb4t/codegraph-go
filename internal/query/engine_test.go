package query

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// copyFixture copies internal/indexer/testdata/gofixture into a fresh
// t.TempDir(), mirroring internal/cli/cli_test.go's helper of the same
// name byte-for-byte (03-PATTERNS.md §"Test scaffolding") so query-package
// tests run against a normal directory with its own go.mod rather than
// mutating the checked-in testdata tree. Exported behavior only — kept
// package-private like its cli sibling since 03-03/04/05 live in this same
// package and can call it directly.
func copyFixture(t *testing.T) string {
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

// indexFixture builds a real graph from the fixture rooted at dir into
// <dir>/.codegraph/store via indexer.Run — the same from-scratch pipeline
// `codegraph init` drives — so OpenAt and every subsequent query-verb test
// (03-03/04/05) exercises a real Pebble-backed store rather than a mock.
// It fails the test immediately on any indexing error.
func indexFixture(t *testing.T, dir string) {
	t.Helper()

	storeDir := filepath.Join(dir, codegraphDirName, storeSubdir)
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
}

func TestResolveCodegraphDir(t *testing.T) {
	t.Run("finds .codegraph at start", func(t *testing.T) {
		dir := copyFixture(t)
		indexFixture(t, dir)

		got, err := ResolveCodegraphDir(dir)
		if err != nil {
			t.Fatalf("ResolveCodegraphDir: unexpected error: %v", err)
		}
		want, err := filepath.Abs(dir)
		if err != nil {
			t.Fatalf("abs: %v", err)
		}
		if got != want {
			t.Fatalf("ResolveCodegraphDir: got %q, want %q", got, want)
		}
	})

	t.Run("walks upward from a nested start path", func(t *testing.T) {
		dir := copyFixture(t)
		indexFixture(t, dir)

		nested := filepath.Join(dir, "nested")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir nested: %v", err)
		}

		got, err := ResolveCodegraphDir(nested)
		if err != nil {
			t.Fatalf("ResolveCodegraphDir: unexpected error: %v", err)
		}
		want, err := filepath.Abs(dir)
		if err != nil {
			t.Fatalf("abs: %v", err)
		}
		if got != want {
			t.Fatalf("ResolveCodegraphDir: got %q, want %q", got, want)
		}
	})

	t.Run("returns ErrNotInitialized when no .codegraph exists", func(t *testing.T) {
		dir := t.TempDir()

		_, err := ResolveCodegraphDir(dir)
		if !errors.Is(err, ErrNotInitialized) {
			t.Fatalf("ResolveCodegraphDir: got err %v, want ErrNotInitialized", err)
		}
	})
}

func TestOpenAt(t *testing.T) {
	t.Run("returns a usable Engine and closer for an init'd repo", func(t *testing.T) {
		dir := copyFixture(t)
		indexFixture(t, dir)

		engine, closer, err := OpenAt(dir)
		if err != nil {
			t.Fatalf("OpenAt: unexpected error: %v", err)
		}
		if engine == nil {
			t.Fatal("OpenAt: got nil Engine")
		}
		if closer == nil {
			t.Fatal("OpenAt: got nil closer")
		}
		if engine.reader == nil {
			t.Fatal("OpenAt: Engine has a nil reader")
		}

		if err := closer.Close(); err != nil {
			t.Fatalf("closer.Close (1st): unexpected error: %v", err)
		}
		if err := closer.Close(); err != nil {
			t.Fatalf("closer.Close (2nd): expected idempotent no-op, got error: %v", err)
		}
	})

	t.Run("returns ErrNotInitialized for a fresh directory", func(t *testing.T) {
		dir := t.TempDir()

		_, _, err := OpenAt(dir)
		if !errors.Is(err, ErrNotInitialized) {
			t.Fatalf("OpenAt: got err %v, want ErrNotInitialized", err)
		}
	})
}

func TestClampDepth(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"non-positive uses default", 0, defaultDepth},
		{"negative uses default", -5, defaultDepth},
		{"in-range passes through", 10, 10},
		{"above ceiling clamps to MaxDepth", MaxDepth + 100, MaxDepth},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampDepth(tc.in); got != tc.want {
				t.Fatalf("clampDepth(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestValidateLimit(t *testing.T) {
	if err := validateLimit(10); err != nil {
		t.Fatalf("validateLimit(10): unexpected error: %v", err)
	}
	if err := validateLimit(MaxLimit); err != nil {
		t.Fatalf("validateLimit(MaxLimit): unexpected error: %v", err)
	}
	if err := validateLimit(MaxLimit + 1); err == nil {
		t.Fatal("validateLimit(MaxLimit+1): expected error, got nil")
	}
	if err := validateLimit(0); err != nil {
		t.Fatalf("validateLimit(0): unexpected error: %v (0 must mean \"no explicit limit\")", err)
	}
	// WR-02: negative values are rejected outright, consistent with
	// validateMaxFiles/validateDepth/validateFilesDepth, rather than
	// silently treated the same as 0/"unset".
	if err := validateLimit(-1); err == nil {
		t.Fatal("validateLimit(-1): expected error, got nil (WR-02: negative must be rejected, not treated as unset)")
	}
}

func TestValidateMaxFiles(t *testing.T) {
	if err := validateMaxFiles(10); err != nil {
		t.Fatalf("validateMaxFiles(10): unexpected error: %v", err)
	}
	if err := validateMaxFiles(MaxFiles); err != nil {
		t.Fatalf("validateMaxFiles(MaxFiles): unexpected error: %v", err)
	}
	if err := validateMaxFiles(MaxFiles + 1); err == nil {
		t.Fatal("validateMaxFiles(MaxFiles+1): expected error, got nil")
	}
	if err := validateMaxFiles(0); err != nil {
		t.Fatalf("validateMaxFiles(0): unexpected error: %v (0 must mean \"use the default\")", err)
	}
	// WR-02: negative values are rejected outright, matching
	// validateLimit/validateDepth/validateFilesDepth.
	if err := validateMaxFiles(-1); err == nil {
		t.Fatal("validateMaxFiles(-1): expected error, got nil (WR-02: negative must be rejected, not treated as unset)")
	}
}

// TestValidateDepth pins WR-02's cross-command negative-value convention
// for Impact's --depth: negative is rejected outright (previously
// clampDepth silently treated it identically to 0/"unset"), while 0
// still means "use the default" and an above-MaxDepth value is left to
// clampDepth's existing silent-clamp behavior (TestImpact's "absurdly
// large depth is clamped, not unbounded").
func TestValidateDepth(t *testing.T) {
	if err := validateDepth(0); err != nil {
		t.Fatalf("validateDepth(0): unexpected error: %v (0 must mean \"use the default\")", err)
	}
	if err := validateDepth(10); err != nil {
		t.Fatalf("validateDepth(10): unexpected error: %v", err)
	}
	if err := validateDepth(-1); err == nil {
		t.Fatal("validateDepth(-1): expected error, got nil (WR-02: negative must be rejected)")
	}
}

func TestValidateKind(t *testing.T) {
	t.Run("empty string accepts (no filter)", func(t *testing.T) {
		if err := ValidateKind(""); err != nil {
			t.Fatalf("ValidateKind(\"\"): unexpected error: %v", err)
		}
	})

	t.Run("known kinds accept", func(t *testing.T) {
		for _, kind := range []string{"function", "method", "type_alias", "package", "file", "struct", "interface", "constant", "variable"} {
			if err := ValidateKind(kind); err != nil {
				t.Fatalf("ValidateKind(%q): unexpected error: %v", kind, err)
			}
		}
	})

	t.Run("unknown kind rejects with allowed-set error", func(t *testing.T) {
		err := ValidateKind("banana")
		if err == nil {
			t.Fatal("ValidateKind(\"banana\"): expected error, got nil")
		}
	})
}
