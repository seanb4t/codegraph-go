package agents

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestManifest_RoundTrips: writing a manifest and reading it back yields
// the same version, location, and file-hash map.
func TestManifest_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".codegraph-manifest.json")

	want := skillManifest{
		SchemaVersion:    manifestSchemaVersion,
		CodegraphVersion: "v0.10.0",
		Location:         string(LocationGlobal),
		Files: map[string]string{
			manifestKeySkillMD:   "sha256:aaaa",
			manifestKeyScript:    "sha256:bbbb",
			manifestKeyHooksFrag: "sha256:cccc",
		},
	}

	fr, err := writeManifest(path, want)
	if err != nil {
		t.Fatalf("writeManifest: %v", err)
	}
	if fr.Action != ActionCreated {
		t.Fatalf("writeManifest action = %q, want %q", fr.Action, ActionCreated)
	}

	got, present, err := readManifest(path)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if !present {
		t.Fatalf("readManifest reported not present after write")
	}
	if got.CodegraphVersion != want.CodegraphVersion {
		t.Fatalf("CodegraphVersion = %q, want %q", got.CodegraphVersion, want.CodegraphVersion)
	}
	if got.Location != want.Location {
		t.Fatalf("Location = %q, want %q", got.Location, want.Location)
	}
	if !stringMapEqual(got.Files, want.Files) {
		t.Fatalf("Files = %#v, want %#v", got.Files, want.Files)
	}
}

// TestManifest_HashIsSha256Hex: hashContent of known bytes equals the
// "sha256:" + lowercase-hex of sha256.Sum256 of those same bytes, computed
// independently in the test — not by calling hashContent twice.
func TestManifest_HashIsSha256Hex(t *testing.T) {
	input := []byte("codegraph skill package content")
	sum := sha256.Sum256(input)
	want := "sha256:" + hex.EncodeToString(sum[:])

	got := hashContent(input)
	if got != want {
		t.Fatalf("hashContent = %q, want %q", got, want)
	}
}

// TestManifest_OwnedHookBlockHashIsStable: hashing the same block list
// twice yields the same value; hashing a block list whose JSON was built
// with keys inserted in a different order yields the same value; hashing a
// block list with a different command yields a different value.
func TestManifest_OwnedHookBlockHashIsStable(t *testing.T) {
	blockA := []any{
		map[string]any{
			"matcher": "startup",
			"hooks": []any{
				map[string]any{"type": "command", "command": "/abs/session-nudge.sh"},
			},
		},
	}
	// Same content, built with map keys inserted in a different order.
	entryDifferentOrder := map[string]any{}
	entryDifferentOrder["command"] = "/abs/session-nudge.sh"
	entryDifferentOrder["type"] = "command"
	blockKeyOrderVaried := []any{
		map[string]any{
			"hooks":   []any{entryDifferentOrder},
			"matcher": "startup",
		},
	}
	blockDifferentCommand := []any{
		map[string]any{
			"matcher": "startup",
			"hooks": []any{
				map[string]any{"type": "command", "command": "/abs/other-script.sh"},
			},
		},
	}

	h1, err := hashOwnedHookBlocks(blockA)
	if err != nil {
		t.Fatalf("hashOwnedHookBlocks(blockA): %v", err)
	}
	h1Again, err := hashOwnedHookBlocks(blockA)
	if err != nil {
		t.Fatalf("hashOwnedHookBlocks(blockA) second call: %v", err)
	}
	if h1 != h1Again {
		t.Fatalf("hashing the same block list twice yielded different hashes: %q vs %q", h1, h1Again)
	}

	h2, err := hashOwnedHookBlocks(blockKeyOrderVaried)
	if err != nil {
		t.Fatalf("hashOwnedHookBlocks(blockKeyOrderVaried): %v", err)
	}
	if h1 != h2 {
		t.Fatalf("hash differed for key-order-varied but content-identical blocks: %q vs %q", h1, h2)
	}

	h3, err := hashOwnedHookBlocks(blockDifferentCommand)
	if err != nil {
		t.Fatalf("hashOwnedHookBlocks(blockDifferentCommand): %v", err)
	}
	if h1 == h3 {
		t.Fatalf("hash was identical for blocks with different commands: %q", h1)
	}
}

// TestManifest_OwnedHookBlockHashIgnoresUnrelatedSettings: the hash
// computed for a given block list is unaffected by what else is present in
// the containing settings.json — proven by hashing the blocks alone versus
// hashing the blocks extracted from a settings.json carrying unrelated
// events and keys.
func TestManifest_OwnedHookBlockHashIgnoresUnrelatedSettings(t *testing.T) {
	blocks := []any{
		map[string]any{
			"matcher": "startup",
			"hooks": []any{
				map[string]any{"type": "command", "command": "/abs/session-nudge.sh"},
			},
		},
	}

	bareHash, err := hashOwnedHookBlocks(blocks)
	if err != nil {
		t.Fatalf("hashOwnedHookBlocks(bare blocks): %v", err)
	}

	// Simulate extracting the exact same blocks value from a settings.json
	// that also carries unrelated events and top-level keys — the caller
	// (writeHookEntry) only ever passes hashOwnedHookBlocks the owned
	// partition, so this proves the hash function itself has no visibility
	// into, or dependence on, the surrounding file.
	settingsShaped := map[string]any{
		"hooks": map[string]any{
			"SessionStart": blocks,
			"PreToolUse": []any{
				map[string]any{"matcher": "Bash", "hooks": []any{
					map[string]any{"type": "command", "command": "/unrelated/guard.sh"},
				}},
			},
		},
		"someUnrelatedTopLevelKey": true,
	}
	extractedBlocks, _ := settingsShaped["hooks"].(map[string]any)["SessionStart"].([]any)

	extractedHash, err := hashOwnedHookBlocks(extractedBlocks)
	if err != nil {
		t.Fatalf("hashOwnedHookBlocks(extracted blocks): %v", err)
	}

	if bareHash != extractedHash {
		t.Fatalf("hash differed depending on surrounding settings.json content: bare=%q extracted=%q", bareHash, extractedHash)
	}
}

// TestManifest_ReadStrictSurfacesMalformed: a malformed manifest file
// returns an error rather than an empty manifest.
func TestManifest_ReadStrictSurfacesMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".codegraph-manifest.json")
	if err := os.WriteFile(path, []byte("{ not valid json"), 0o644); err != nil {
		t.Fatalf("seed malformed manifest: %v", err)
	}

	_, present, err := readManifest(path)
	if err == nil {
		t.Fatalf("readManifest on malformed file returned no error")
	}
	if present {
		t.Fatalf("readManifest on malformed file reported present=true")
	}
}

// TestConfiguredSkillLocations_ProbesFixedPaths: with a manifest present
// only at global scope, the function returns exactly [LocationGlobal];
// with neither present, it returns empty; with both, it returns both.
func TestConfiguredSkillLocations_ProbesFixedPaths(t *testing.T) {
	t.Run("neither present returns empty", func(t *testing.T) {
		fakeHome(t)
		locs := ConfiguredSkillLocations(Claude)
		if len(locs) != 0 {
			t.Fatalf("expected empty, got %v", locs)
		}
	})

	t.Run("global only", func(t *testing.T) {
		home := fakeHome(t)
		globalPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
		if _, err := writeManifest(globalPath, skillManifest{
			SchemaVersion:    manifestSchemaVersion,
			CodegraphVersion: "v0.10.0",
			Location:         string(LocationGlobal),
			Files:            map[string]string{},
		}); err != nil {
			t.Fatalf("seed global manifest: %v", err)
		}

		locs := ConfiguredSkillLocations(Claude)
		if len(locs) != 1 || locs[0] != LocationGlobal {
			t.Fatalf("expected [global], got %v", locs)
		}
	})

	t.Run("both present", func(t *testing.T) {
		home := fakeHome(t)
		globalPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
		if _, err := writeManifest(globalPath, skillManifest{
			SchemaVersion:    manifestSchemaVersion,
			CodegraphVersion: "v0.10.0",
			Location:         string(LocationGlobal),
			Files:            map[string]string{},
		}); err != nil {
			t.Fatalf("seed global manifest: %v", err)
		}
		localPath := filepath.Join(".claude", "skills", "codegraph", ".codegraph-manifest.json")
		if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
			t.Fatalf("mkdir local skill dir: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(filepath.Join(".claude")) })
		if _, err := writeManifest(localPath, skillManifest{
			SchemaVersion:    manifestSchemaVersion,
			CodegraphVersion: "v0.10.0",
			Location:         string(LocationLocal),
			Files:            map[string]string{},
		}); err != nil {
			t.Fatalf("seed local manifest: %v", err)
		}

		locs := ConfiguredSkillLocations(Claude)
		if len(locs) != 2 {
			t.Fatalf("expected both locations, got %v", locs)
		}
		seen := map[Location]bool{}
		for _, l := range locs {
			seen[l] = true
		}
		if !seen[LocationGlobal] || !seen[LocationLocal] {
			t.Fatalf("expected both global and local, got %v", locs)
		}
	})

	t.Run("other target ids return empty", func(t *testing.T) {
		fakeHome(t)
		for _, id := range []TargetID{Cursor, Codex, Opencode, Hermes, Gemini, Antigravity, Kiro} {
			if locs := ConfiguredSkillLocations(id); len(locs) != 0 {
				t.Fatalf("ConfiguredSkillLocations(%s) = %v, want empty", id, locs)
			}
		}
	})
}

// TestManifest_WriteIsIdempotentAndPreservesTimestamp: a second write of
// unchanged version/location/files leaves the file's raw bytes and
// modification time identical, and reports ActionUnchanged.
func TestManifest_WriteIsIdempotentAndPreservesTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".codegraph-manifest.json")
	m := skillManifest{
		SchemaVersion:    manifestSchemaVersion,
		CodegraphVersion: "v0.10.0",
		Location:         string(LocationGlobal),
		Files: map[string]string{
			manifestKeySkillMD: "sha256:aaaa",
		},
	}

	if _, err := writeManifest(path, m); err != nil {
		t.Fatalf("first writeManifest: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after first write: %v", err)
	}
	infoBefore, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after first write: %v", err)
	}

	fr, err := writeManifest(path, m)
	if err != nil {
		t.Fatalf("second writeManifest: %v", err)
	}
	if fr.Action != ActionUnchanged {
		t.Fatalf("second writeManifest action = %q, want %q", fr.Action, ActionUnchanged)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after second write: %v", err)
	}
	infoAfter, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after second write: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("manifest bytes changed on unchanged re-write:\nbefore=%q\nafter=%q", before, after)
	}
	if !infoBefore.ModTime().Equal(infoAfter.ModTime()) {
		t.Fatalf("manifest mtime changed on unchanged re-write: before=%v after=%v", infoBefore.ModTime(), infoAfter.ModTime())
	}
}
