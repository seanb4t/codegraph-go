package wireoracle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveTestBinPath mirrors test/integration/binpath_test.go's table
// test of the same name exactly, over this package's own deliberate
// second copy of resolveTestBinPath (main_test.go) — Go test helpers are
// not importable across packages, matching the precedent this repo
// already set for its four runGit* helpers. Every case also re-checks the
// "no third outcome" invariant inline: useEnv=true never pairs with a
// non-nil error, and a non-empty raw value never resolves to useEnv=false
// with a nil error.
func TestResolveTestBinPath(t *testing.T) {
	dir := t.TempDir()

	validExec := filepath.Join(dir, "valid-exec")
	if err := os.WriteFile(validExec, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Chmod(validExec, 0o755); err != nil {
		t.Fatalf("chmod fixture executable: %v", err)
	}

	nonExecFile := filepath.Join(dir, "not-exec")
	if err := os.WriteFile(nonExecFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Chmod(nonExecFile, 0o644); err != nil {
		t.Fatalf("chmod fixture non-executable: %v", err)
	}

	missingPath := filepath.Join(dir, "does-not-exist")

	cases := []struct {
		name       string
		raw        string
		wantUseEnv bool
		wantErr    bool
		wantErrSub []string // every substring must appear in the error message
		wantAbs    bool     // if wantUseEnv, path must equal filepath.Abs(raw)
	}{
		{
			name:       "empty falls through to build path",
			raw:        "",
			wantUseEnv: false,
			wantErr:    false,
		},
		{
			name:       "existing executable regular file resolves with useEnv true",
			raw:        validExec,
			wantUseEnv: true,
			wantErr:    false,
			wantAbs:    true,
		},
		{
			name:       "nonexistent path aborts naming the variable and the path",
			raw:        missingPath,
			wantUseEnv: false,
			wantErr:    true,
			wantErrSub: []string{testBinEnvVar, missingPath},
		},
		{
			name:       "directory aborts naming the variable and not-a-regular-file",
			raw:        dir,
			wantUseEnv: false,
			wantErr:    true,
			wantErrSub: []string{testBinEnvVar, "not a regular file"},
		},
		{
			name:       "existing non-executable regular file aborts naming the variable and not-executable",
			raw:        nonExecFile,
			wantUseEnv: false,
			wantErr:    true,
			wantErrSub: []string{testBinEnvVar, "not executable"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path, useEnv, err := resolveTestBinPath(c.raw)

			if useEnv != c.wantUseEnv {
				t.Errorf("useEnv = %v, want %v", useEnv, c.wantUseEnv)
			}
			if c.wantErr && err == nil {
				t.Fatalf("err = nil, want an error")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			for _, sub := range c.wantErrSub {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("err = %q, want it to contain %q", err.Error(), sub)
				}
			}
			if !useEnv && path != "" {
				t.Errorf("path = %q, want empty when useEnv is false", path)
			}
			if c.wantAbs {
				wantAbs, absErr := filepath.Abs(c.raw)
				if absErr != nil {
					t.Fatalf("filepath.Abs(%q): %v", c.raw, absErr)
				}
				if path != wantAbs {
					t.Errorf("path = %q, want %q", path, wantAbs)
				}
				if !filepath.IsAbs(path) {
					t.Errorf("path = %q, want an absolute path", path)
				}
			}

			// No third outcome (see resolveTestBinPath's doc comment):
			if useEnv && err != nil {
				t.Errorf("useEnv=true with non-nil err=%v (third outcome forbidden)", err)
			}
			if c.raw != "" && err == nil && !useEnv {
				t.Errorf("non-empty input resolved useEnv=false with nil err (silent fallback forbidden)")
			}
		})
	}

	t.Run("relative path to an existing executable resolves to an absolute path", func(t *testing.T) {
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(oldWd) })
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("Chdir(%q): %v", dir, err)
		}

		path, useEnv, err := resolveTestBinPath("valid-exec")
		if err != nil {
			t.Fatalf(`resolveTestBinPath("valid-exec") returned error: %v`, err)
		}
		if !useEnv {
			t.Fatalf("useEnv = false, want true")
		}
		if !filepath.IsAbs(path) {
			t.Fatalf("path = %q, want an absolute path", path)
		}
		// Derive the expectation via filepath.Abs("valid-exec") — the
		// same os.Getwd()-based mechanism resolveTestBinPath itself uses —
		// rather than joining the pre-Chdir dir value: on macOS /var is a
		// symlink to /private/var, so os.Getwd() after Chdir(dir) returns
		// the resolved /private/var/... form while dir (captured before
		// Chdir) does not, making a naive filepath.Join(dir, ...) compare
		// unequal to a correct absolute path pointing at the same file.
		wantAbs, err := filepath.Abs("valid-exec")
		if err != nil {
			t.Fatalf("filepath.Abs: %v", err)
		}
		if path != wantAbs {
			t.Fatalf("path = %q, want %q", path, wantAbs)
		}
	})
}
