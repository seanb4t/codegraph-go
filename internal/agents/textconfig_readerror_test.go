package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTextConfig_ReadFailureNeverDestroysContent pins issue #304: the Codex
// and Hermes writers read a text configuration file, splice their own block
// and write the result. A file that exists and cannot be read must stop
// the write. Each cell asserts the error, the bytes and the mode, because
// an error with a rewritten file is the defect and an untouched file with
// no error hides it.
func TestTextConfig_ReadFailureNeverDestroysContent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a 0o000 file, so no read error occurs and the test proves nothing")
	}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	cases := []struct {
		name    string
		relPath string
		fixture string
		op      func() WriteResult
	}{
		{
			name:    "codex install",
			relPath: filepath.Join(".codex", "config.toml"),
			fixture: "approval_policy = \"never\"\n",
			op:      func() WriteResult { return codexTarget{}.Install(LocationGlobal, opts) },
		},
		{
			name:    "codex uninstall",
			relPath: filepath.Join(".codex", "config.toml"),
			fixture: "approval_policy = \"never\"\n",
			op:      func() WriteResult { return codexTarget{}.Uninstall(LocationGlobal) },
		},
		{
			name:    "hermes install",
			relPath: filepath.Join(".hermes", "config.yaml"),
			fixture: "model: some-model\n",
			op:      func() WriteResult { return hermesTarget{}.Install(LocationGlobal, opts) },
		},
		{
			name:    "hermes uninstall",
			relPath: filepath.Join(".hermes", "config.yaml"),
			fixture: "model: some-model\n",
			op:      func() WriteResult { return hermesTarget{}.Uninstall(LocationGlobal) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := fakeHome(t)
			t.Setenv("HERMES_HOME", "")
			path := filepath.Join(home, tc.relPath)
			writeFile(t, path, tc.fixture)
			if err := os.Chmod(path, 0o000); err != nil {
				t.Fatalf("chmod %s to 0o000: %v", path, err)
			}
			t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

			result := tc.op()

			named := false
			for _, e := range result.Errors {
				if strings.Contains(e.Error(), path) {
					named = true
				}
			}
			if !named {
				t.Fatalf("no error names %s: %v", path, result.Errors)
			}
			for _, f := range result.Files {
				if f.Path == path {
					t.Fatalf("%s is reported as %q, want no record for a file that was not read", path, f.Action)
				}
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat %s after the operation: %v", path, err)
			}
			if mode := info.Mode().Perm(); mode != 0 {
				t.Fatalf("mode of %s is %o, want 0: the file was replaced", path, mode)
			}
			if err := os.Chmod(path, 0o644); err != nil {
				t.Fatalf("restore the mode of %s: %v", path, err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s after the operation: %v", path, err)
			}
			if string(got) != tc.fixture {
				t.Fatalf("content of %s changed:\nbefore=%q\nafter=%q", path, tc.fixture, got)
			}
		})
	}
}
