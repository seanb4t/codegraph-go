package upgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildBrewTree creates <root>/<prefixRel>/<tree>/<token>/<version>/codegraph
// (the payload) and, if withReceipt, the INSTALL_RECEIPT.json at the
// location that tree implies. wrongAncestor (Caskroom only) instead writes
// the receipt directly in the versioned directory — a location the
// Caskroom detector never checks — so the row genuinely distinguishes
// "shape present" from "shape present AND receipt at the RIGHT ancestor"
// (D-03). Returns the payload path.
func buildBrewTree(t *testing.T, root, prefixRel, tree, token, version string, withReceipt, wrongAncestor bool) string {
	t.Helper()

	tokenDir := filepath.Join(root, prefixRel, tree, token)
	versionDir := filepath.Join(tokenDir, version)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatalf("mkdir version dir: %v", err)
	}
	payload := filepath.Join(versionDir, "codegraph")
	if err := os.WriteFile(payload, []byte("payload"), 0o755); err != nil {
		t.Fatalf("seed payload: %v", err)
	}

	if withReceipt {
		var receiptPath string
		switch {
		case tree == "Caskroom" && wrongAncestor:
			receiptPath = filepath.Join(versionDir, brewReceiptFilename)
		case tree == "Caskroom":
			metadataDir := filepath.Join(tokenDir, ".metadata")
			if err := os.MkdirAll(metadataDir, 0o755); err != nil {
				t.Fatalf("mkdir metadata dir: %v", err)
			}
			receiptPath = filepath.Join(metadataDir, brewReceiptFilename)
		default: // Cellar
			receiptPath = filepath.Join(versionDir, brewReceiptFilename)
		}
		if err := os.WriteFile(receiptPath, []byte(`{}`), 0o644); err != nil {
			t.Fatalf("seed receipt: %v", err)
		}
	}

	return payload
}

// buildPlainFixture creates <root>/<prefixRel>/bin/codegraph — a payload
// with no Caskroom or Cellar segment anywhere in its ancestry.
func buildPlainFixture(t *testing.T, root, prefixRel string) string {
	t.Helper()

	dir := filepath.Join(root, prefixRel, "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}
	payload := filepath.Join(dir, "codegraph")
	if err := os.WriteFile(payload, []byte("payload"), 0o755); err != nil {
		t.Fatalf("seed payload: %v", err)
	}
	return payload
}

// buildTooFewSegmentsFixture creates <root>/<prefixRel>/Caskroom/codegraph
// with the binary directly inside — no separate token/version pair below
// Caskroom — plus a real receipt beside it, so the row proves the
// detector's segment-count requirement rejects it rather than a missing
// receipt.
func buildTooFewSegmentsFixture(t *testing.T, root, prefixRel string) string {
	t.Helper()

	tokenDir := filepath.Join(root, prefixRel, "Caskroom", "codegraph")
	if err := os.MkdirAll(tokenDir, 0o755); err != nil {
		t.Fatalf("mkdir token dir: %v", err)
	}
	payload := filepath.Join(tokenDir, "codegraph")
	if err := os.WriteFile(payload, []byte("payload"), 0o755); err != nil {
		t.Fatalf("seed payload: %v", err)
	}
	metadataDir := filepath.Join(tokenDir, ".metadata")
	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		t.Fatalf("mkdir metadata dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(metadataDir, brewReceiptFilename), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("seed receipt: %v", err)
	}
	return payload
}

// buildDanglingSymlinkFixture creates a Caskroom/<token>/<version>/
// tree with a REAL receipt at the correct ancestor, but the payload
// position itself is a symlink whose target has been removed. This is the
// executing regression fixture for the EvalSymlinks-error contract
// (T-04-03, review cycle 1 MEDIUM): the row fails under "fall back to the
// unresolved path and keep scanning" (shape + receipt both present) and
// passes only under "error means not-detected".
func buildDanglingSymlinkFixture(t *testing.T, root, prefixRel, tree, token, version string) string {
	t.Helper()

	tokenDir := filepath.Join(root, prefixRel, tree, token)
	versionDir := filepath.Join(tokenDir, version)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatalf("mkdir version dir: %v", err)
	}
	metadataDir := filepath.Join(tokenDir, ".metadata")
	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		t.Fatalf("mkdir metadata dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(metadataDir, brewReceiptFilename), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("seed receipt: %v", err)
	}

	symlinkPath := filepath.Join(versionDir, "codegraph")
	target := filepath.Join(versionDir, "codegraph-target-removed")
	if err := os.WriteFile(target, []byte("temp"), 0o755); err != nil {
		t.Fatalf("seed dangling target: %v", err)
	}
	if err := os.Symlink(target, symlinkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove dangling target: %v", err)
	}
	return symlinkPath
}

// buildSymlinkLoopFixture creates two symlinks pointing at each other at a
// path shaped like Caskroom/<token>/<version>/codegraph. The detector must
// return not-detected rather than hanging or panicking (T-04-03).
func buildSymlinkLoopFixture(t *testing.T, root, prefixRel, tree, token, version string) string {
	t.Helper()

	versionDir := filepath.Join(root, prefixRel, tree, token, version)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatalf("mkdir version dir: %v", err)
	}
	a := filepath.Join(versionDir, "codegraph")
	b := filepath.Join(versionDir, "codegraph-loop-b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatalf("symlink a->b: %v", err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatalf("symlink b->a: %v", err)
	}
	return a
}

// TestDetectBrewManaged is the constructed-tree table: every prefix/tree
// combination UPGR-02 and D-04R require, plus the executing
// false-positive and EvalSymlinks-failure rows (D-03, T-04-01, T-04-03).
// Simulating a prefix as a relative subtree under t.TempDir() is
// deliberate — the real prefixes are unwritable in a test, and per the
// promote decision recorded in 04-01-PLAN.md, the detector must not care
// what the prefix is.
func TestDetectBrewManaged(t *testing.T) {
	type row struct {
		name         string
		build        func(t *testing.T, root string) string
		wantDetected bool
		wantTree     string
		wantToken    string
		wantVersion  string
	}

	prefixes := []struct {
		name string
		rel  string
	}{
		{"apple-silicon", filepath.Join("opt", "homebrew")},
		{"intel", filepath.Join("usr", "local")},
		{"custom", filepath.Join("opt", "custom-brew-prefix")},
		{"linuxbrew", filepath.Join("home", "linuxbrew", ".linuxbrew")},
	}

	var rows []row
	for _, p := range prefixes {
		p := p
		rows = append(rows,
			row{
				name: p.name + "-caskroom-detected",
				build: func(t *testing.T, root string) string {
					return buildBrewTree(t, root, p.rel, "Caskroom", "codegraph", "0.8.0", true, false)
				},
				wantDetected: true, wantTree: "Caskroom", wantToken: "codegraph", wantVersion: "0.8.0",
			},
			row{
				name: p.name + "-cellar-detected",
				build: func(t *testing.T, root string) string {
					return buildBrewTree(t, root, p.rel, "Cellar", "codegraph", "0.8.0", true, false)
				},
				wantDetected: true, wantTree: "Cellar", wantToken: "codegraph", wantVersion: "0.8.0",
			},
		)
	}

	appleSiliconRel := filepath.Join("opt", "homebrew")
	rows = append(rows,
		row{
			name: "caskroom-no-receipt-not-detected",
			build: func(t *testing.T, root string) string {
				return buildBrewTree(t, root, appleSiliconRel, "Caskroom", "codegraph", "0.8.0", false, false)
			},
			wantDetected: false,
		},
		row{
			name: "cellar-no-receipt-not-detected",
			build: func(t *testing.T, root string) string {
				return buildBrewTree(t, root, appleSiliconRel, "Cellar", "codegraph", "0.8.0", false, false)
			},
			wantDetected: false,
		},
		row{
			name: "caskroom-receipt-wrong-ancestor-not-detected",
			build: func(t *testing.T, root string) string {
				return buildBrewTree(t, root, appleSiliconRel, "Caskroom", "codegraph", "0.8.0", true, true)
			},
			wantDetected: false,
		},
		row{
			name: "no-tree-segment-not-detected",
			build: func(t *testing.T, root string) string {
				return buildPlainFixture(t, root, appleSiliconRel)
			},
			wantDetected: false,
		},
		row{
			name: "caskroom-too-few-segments-not-detected",
			build: func(t *testing.T, root string) string {
				return buildTooFewSegmentsFixture(t, root, appleSiliconRel)
			},
			wantDetected: false,
		},
		row{
			name: "reached-through-symlink-detected",
			build: func(t *testing.T, root string) string {
				payload := buildBrewTree(t, root, appleSiliconRel, "Caskroom", "codegraph", "0.8.0", true, false)
				binDir := filepath.Join(root, appleSiliconRel, "bin")
				if err := os.MkdirAll(binDir, 0o755); err != nil {
					t.Fatalf("mkdir bin dir: %v", err)
				}
				symlink := filepath.Join(binDir, "codegraph")
				if err := os.Symlink(payload, symlink); err != nil {
					t.Fatalf("symlink payload: %v", err)
				}
				return symlink
			},
			wantDetected: true, wantTree: "Caskroom", wantToken: "codegraph", wantVersion: "0.8.0",
		},
		row{
			name: "dangling-symlink-not-detected",
			build: func(t *testing.T, root string) string {
				return buildDanglingSymlinkFixture(t, root, appleSiliconRel, "Caskroom", "codegraph", "0.8.0")
			},
			wantDetected: false,
		},
		row{
			name: "symlink-loop-not-detected",
			build: func(t *testing.T, root string) string {
				return buildSymlinkLoopFixture(t, root, appleSiliconRel, "Caskroom", "codegraph", "0.8.0")
			},
			wantDetected: false,
		},
	)

	// Row-count guard (repo rule 84d1gfpywd): a table that silently loses
	// rows in a future edit must break loudly rather than shrink into a
	// green no-op. Both floors equal the exact inventory recorded in
	// 04-01-PLAN.md's Task 2 behavior block — deleting any single row
	// turns one of these RED.
	if len(rows) < 16 {
		t.Fatalf("row-count guard: table has %d rows, want at least 16 (a row was silently dropped)", len(rows))
	}
	notDetectedCount := 0
	for _, r := range rows {
		if !r.wantDetected {
			notDetectedCount++
		}
	}
	if notDetectedCount < 7 {
		t.Fatalf("not-detected row-count guard: table has %d not-detected rows, want at least 7 (a false-positive row was silently dropped)", notDetectedCount)
	}

	for _, tc := range rows {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := tc.build(t, root)

			inst, ok := detectBrewManaged(path)
			if ok != tc.wantDetected {
				t.Fatalf("detectBrewManaged(%q) ok = %v, want %v", path, ok, tc.wantDetected)
			}
			if !tc.wantDetected {
				if inst != (brewInstall{}) {
					t.Fatalf("detectBrewManaged(%q) returned a non-zero struct %+v for a not-detected row", path, inst)
				}
				return
			}

			if inst.Tree != tc.wantTree {
				t.Errorf("Tree = %q, want %q", inst.Tree, tc.wantTree)
			}
			if inst.Token != tc.wantToken {
				t.Errorf("Token = %q, want %q", inst.Token, tc.wantToken)
			}
			if inst.Version != tc.wantVersion {
				t.Errorf("Version = %q, want %q", inst.Version, tc.wantVersion)
			}
			if _, err := os.Stat(inst.Receipt); err != nil {
				t.Errorf("Receipt %q does not exist: %v", inst.Receipt, err)
			}
			wantSuffix := filepath.Join(tc.wantTree, tc.wantToken, tc.wantVersion)
			if !strings.HasSuffix(inst.InstallDir, wantSuffix) {
				t.Errorf("InstallDir = %q, want suffix %q", inst.InstallDir, wantSuffix)
			}
			// Cross-cutting regression assertion for every detected row
			// (review cycle 1, HIGH): InstallDir must be absolute — never
			// the relative path the split-and-rejoin defect would have
			// produced.
			if !filepath.IsAbs(inst.InstallDir) {
				t.Errorf("InstallDir = %q, want an absolute path", inst.InstallDir)
			}
		})
	}
}

// TestDetectBrewManaged_RealInstall is the manual-acceptance harness plan
// 04-06 leg 1 drives against a genuine Homebrew install. It is skipped by
// default (no test in this suite touches a real Homebrew installation —
// D-13) and only runs when CODEGRAPH_BREW_ACCEPTANCE_PATH is set.
func TestDetectBrewManaged_RealInstall(t *testing.T) {
	path := os.Getenv("CODEGRAPH_BREW_ACCEPTANCE_PATH")
	if path == "" {
		t.Skip("CODEGRAPH_BREW_ACCEPTANCE_PATH is unset; set it to the path of a real Homebrew-managed codegraph binary to run this acceptance check")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("CODEGRAPH_BREW_ACCEPTANCE_PATH=%q does not exist: %v", path, err)
	}

	inst, ok := detectBrewManaged(path)
	if !ok {
		t.Fatalf("detectBrewManaged(%q) reported not detected against a real Homebrew install", path)
	}
	t.Logf("resolved binary: %s", inst.ResolvedBinary)
	t.Logf("install dir: %s", inst.InstallDir)
	t.Logf("tree: %s", inst.Tree)
	t.Logf("token: %s", inst.Token)
	t.Logf("version: %s", inst.Version)
	t.Logf("receipt: %s", inst.Receipt)
}
