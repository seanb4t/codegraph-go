package agents

import (
	"os"
	"path/filepath"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// TestSharedSkillPackage_WritesOnceAndAccumulatesTargets is the tracer test
// for the shared writer (D-05, D-07, AGENT-09): installing a shared skill
// directory for one requester writes SKILL.md and a manifest recording
// that requester; installing again for a second requester leaves SKILL.md
// unchanged (raw-byte idempotency, D-07 of v0.10.0) but accumulates the
// second requester into the manifest's targets set — never overwriting the
// first. A byte-clean re-run for either requester reports every file
// unchanged.
func TestSharedSkillPackage_WritesOnceAndAccumulatesTargets(t *testing.T) {
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			if loc == LocationLocal {
				scratch := t.TempDir()
				oldwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Getwd: %v", err)
				}
				if err := os.Chdir(scratch); err != nil {
					t.Fatalf("Chdir: %v", err)
				}
				t.Cleanup(func() { _ = os.Chdir(oldwd) })
			}

			dir, err := sharedSkillDirPath(loc)
			if err != nil {
				t.Fatalf("sharedSkillDirPath: %v", err)
			}
			wantContent, err := claudeassets.SkillMarkdown()
			if err != nil {
				t.Fatalf("claudeassets.SkillMarkdown: %v", err)
			}

			var r1 WriteResult
			installSkillPackage(&r1, dir, loc, Cursor, refuseUnmanifested)
			if len(r1.Errors) != 0 {
				t.Fatalf("first install errors: %v", r1.Errors)
			}
			gotSkillMD, err := os.ReadFile(filepath.Join(dir, skillFileName))
			if err != nil {
				t.Fatalf("read SKILL.md after first install: %v", err)
			}
			if string(gotSkillMD) != string(wantContent) {
				t.Fatalf("installed SKILL.md does not match embedded content")
			}
			m1, present1, err := readManifest(skillManifestPath(dir))
			if err != nil || !present1 {
				t.Fatalf("readManifest after first install: present=%v err=%v", present1, err)
			}
			if m1.SchemaVersion != manifestSchemaVersion {
				t.Fatalf("SchemaVersion = %d, want %d", m1.SchemaVersion, manifestSchemaVersion)
			}
			if !targetSetEqual(m1.Targets, []TargetID{Cursor}) {
				t.Fatalf("Targets after first install = %v, want [cursor]", m1.Targets)
			}
			if m1.Files[manifestKeySkillMD] != hashContent(wantContent) {
				t.Fatalf("Files[manifestKeySkillMD] = %q, want hash of embedded content", m1.Files[manifestKeySkillMD])
			}
			assertFileResult(t, r1.Files, filepath.Join(dir, skillFileName), ActionCreated)
			assertFileResult(t, r1.Files, skillManifestPath(dir), ActionCreated)

			var r2 WriteResult
			installSkillPackage(&r2, dir, loc, Opencode, refuseUnmanifested)
			if len(r2.Errors) != 0 {
				t.Fatalf("second install errors: %v", r2.Errors)
			}
			assertFileResult(t, r2.Files, filepath.Join(dir, skillFileName), ActionUnchanged)
			m2, present2, err := readManifest(skillManifestPath(dir))
			if err != nil || !present2 {
				t.Fatalf("readManifest after second install: present=%v err=%v", present2, err)
			}
			if !targetSetEqual(m2.Targets, []TargetID{Cursor, Opencode}) {
				t.Fatalf("Targets after second install = %v, want [cursor opencode]", m2.Targets)
			}

			before, err := os.ReadFile(skillManifestPath(dir))
			if err != nil {
				t.Fatalf("read manifest bytes: %v", err)
			}
			var r3, r4 WriteResult
			installSkillPackage(&r3, dir, loc, Cursor, refuseUnmanifested)
			installSkillPackage(&r4, dir, loc, Opencode, refuseUnmanifested)
			for i, r := range []WriteResult{r3, r4} {
				for _, fr := range r.Files {
					if fr.Action != ActionUnchanged {
						t.Fatalf("re-run %d reported non-unchanged action for %s: %q", i, fr.Path, fr.Action)
					}
				}
			}
			after, err := os.ReadFile(skillManifestPath(dir))
			if err != nil {
				t.Fatalf("read manifest bytes after re-run: %v", err)
			}
			if string(before) != string(after) {
				t.Fatalf("manifest bytes changed on a byte-clean re-run")
			}
		})
	}
}

// assertFileResult fails the test unless files contains an entry for path
// with the given action.
func assertFileResult(t *testing.T, files []FileResult, path string, action FileAction) {
	t.Helper()
	for _, fr := range files {
		if fr.Path == path {
			if fr.Action != action {
				t.Fatalf("Files[%s].Action = %q, want %q", path, fr.Action, action)
			}
			return
		}
	}
	t.Fatalf("result.Files missing entry for %s (want action %q): %+v", path, action, files)
}

// TestSharedSkillPackage_UninstallRemovesOnlyRequester (D-08): removing one
// requester from a two-requester package leaves the other requester's
// package intact, byte-identical, and reports the manifest updated (its
// targets set shrank) while SKILL.md is merely kept (still needed).
func TestSharedSkillPackage_UninstallRemovesOnlyRequester(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")

	var r1, r2 WriteResult
	installSkillPackage(&r1, dir, LocationGlobal, Cursor, refuseUnmanifested)
	installSkillPackage(&r2, dir, LocationGlobal, Opencode, refuseUnmanifested)
	if len(r1.Errors) != 0 || len(r2.Errors) != 0 {
		t.Fatalf("setup installs failed: %v %v", r1.Errors, r2.Errors)
	}

	var u WriteResult
	uninstallSkillPackage(&u, dir, Cursor, nil, refuseUnmanifested)
	if len(u.Errors) != 0 {
		t.Fatalf("uninstall errors: %v", u.Errors)
	}

	m, present, err := readManifest(skillManifestPath(dir))
	if err != nil || !present {
		t.Fatalf("manifest not present after partial uninstall: present=%v err=%v", present, err)
	}
	if !targetSetEqual(m.Targets, []TargetID{Opencode}) {
		t.Fatalf("Targets = %v, want [opencode]", m.Targets)
	}

	wantContent, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		t.Fatalf("read SKILL.md after partial uninstall: %v", err)
	}
	if string(got) != string(wantContent) {
		t.Fatalf("SKILL.md content changed after partial uninstall")
	}
	assertFileResult(t, u.Files, filepath.Join(dir, skillFileName), ActionKept)
	assertFileResult(t, u.Files, skillManifestPath(dir), ActionUpdated)
}

// TestSharedSkillPackage_LastRequesterDeletesPackage (D-08): once every
// requester has uninstalled, both the manifest and SKILL.md are removed,
// and the directory itself is gone.
func TestSharedSkillPackage_LastRequesterDeletesPackage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")

	var r1, r2 WriteResult
	installSkillPackage(&r1, dir, LocationGlobal, Cursor, refuseUnmanifested)
	installSkillPackage(&r2, dir, LocationGlobal, Opencode, refuseUnmanifested)
	if len(r1.Errors) != 0 || len(r2.Errors) != 0 {
		t.Fatalf("setup installs failed: %v %v", r1.Errors, r2.Errors)
	}

	var u1 WriteResult
	uninstallSkillPackage(&u1, dir, Cursor, nil, refuseUnmanifested)
	if len(u1.Errors) != 0 {
		t.Fatalf("uninstall Cursor errors: %v", u1.Errors)
	}

	var u2 WriteResult
	uninstallSkillPackage(&u2, dir, Opencode, nil, refuseUnmanifested)
	if len(u2.Errors) != 0 {
		t.Fatalf("uninstall Opencode errors: %v", u2.Errors)
	}
	assertFileResult(t, u2.Files, skillManifestPath(dir), ActionRemoved)
	assertFileResult(t, u2.Files, filepath.Join(dir, skillFileName), ActionRemoved)
	if fileExists(dir) {
		t.Fatalf("skill dir still exists after last requester removed")
	}
}

// TestSharedSkillPackage_UninstallNonRequesterIsNoop (D-08): uninstalling a
// target that never requested the package changes nothing — both files
// report not-found and stay byte-identical.
func TestSharedSkillPackage_UninstallNonRequesterIsNoop(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")

	var r1 WriteResult
	installSkillPackage(&r1, dir, LocationGlobal, Opencode, refuseUnmanifested)
	if len(r1.Errors) != 0 {
		t.Fatalf("setup install failed: %v", r1.Errors)
	}

	beforeSkill, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		t.Fatalf("read SKILL.md before uninstall: %v", err)
	}
	beforeManifest, err := os.ReadFile(skillManifestPath(dir))
	if err != nil {
		t.Fatalf("read manifest before uninstall: %v", err)
	}

	var u WriteResult
	uninstallSkillPackage(&u, dir, Cursor, nil, refuseUnmanifested)
	if len(u.Errors) != 0 {
		t.Fatalf("uninstall non-requester errors: %v", u.Errors)
	}
	assertFileResult(t, u.Files, filepath.Join(dir, skillFileName), ActionNotFound)
	assertFileResult(t, u.Files, skillManifestPath(dir), ActionNotFound)

	afterSkill, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		t.Fatalf("read SKILL.md after uninstall: %v", err)
	}
	afterManifest, err := os.ReadFile(skillManifestPath(dir))
	if err != nil {
		t.Fatalf("read manifest after uninstall: %v", err)
	}
	if string(beforeSkill) != string(afterSkill) {
		t.Fatalf("SKILL.md changed on non-requester uninstall")
	}
	if string(beforeManifest) != string(afterManifest) {
		t.Fatalf("manifest changed on non-requester uninstall")
	}
}

// TestSharedSkillPackage_ForeignDirKeptForeign (D-14, the 242ec0a
// differential): a directory holding a foreign SKILL.md and no codegraph
// manifest is never touched by install or uninstall — both report exactly
// one `kept (foreign)` entry for the directory itself.
func TestSharedSkillPackage_ForeignDirKeptForeign(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	foreignContent := "# someone else's skill\n"
	if err := os.WriteFile(filepath.Join(dir, skillFileName), []byte(foreignContent), 0o644); err != nil {
		t.Fatalf("seed foreign SKILL.md: %v", err)
	}

	var r WriteResult
	installSkillPackage(&r, dir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(r.Errors) != 0 {
		t.Fatalf("install on foreign dir errors: %v", r.Errors)
	}
	if len(r.Files) != 1 || r.Files[0].Path != dir || r.Files[0].Action != ActionKeptForeign {
		t.Fatalf("expected exactly one {dir, kept (foreign)} entry, got %+v", r.Files)
	}
	got, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if string(got) != foreignContent {
		t.Fatalf("foreign SKILL.md bytes changed")
	}
	if fileExists(skillManifestPath(dir)) {
		t.Fatalf("manifest created for a foreign dir")
	}

	var u WriteResult
	uninstallSkillPackage(&u, dir, Cursor, nil, refuseUnmanifested)
	if len(u.Errors) != 0 {
		t.Fatalf("uninstall on foreign dir errors: %v", u.Errors)
	}
	if len(u.Files) != 1 || u.Files[0].Path != dir || u.Files[0].Action != ActionKeptForeign {
		t.Fatalf("expected exactly one {dir, kept (foreign)} entry on uninstall, got %+v", u.Files)
	}
	got2, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		t.Fatalf("read SKILL.md after uninstall: %v", err)
	}
	if string(got2) != foreignContent {
		t.Fatalf("foreign SKILL.md bytes changed after uninstall")
	}
}

// TestSharedSkillPackage_EmptyDirIsNotForeign (D-14): an existing but
// genuinely empty directory is not foreign — install writes into it
// normally.
func TestSharedSkillPackage_EmptyDirIsNotForeign(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var r WriteResult
	installSkillPackage(&r, dir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(r.Errors) != 0 {
		t.Fatalf("install on empty dir errors: %v", r.Errors)
	}
	assertFileResult(t, r.Files, filepath.Join(dir, skillFileName), ActionCreated)
	assertFileResult(t, r.Files, skillManifestPath(dir), ActionCreated)
}

// TestSharedSkillPackage_AdoptPolicyAdoptsUnmanifested (D-05): under
// adoptUnmanifested (Claude's own non-shared skill dir), a directory with
// no manifest is claimed rather than left foreign — its SKILL.md is
// rewritten to the embed and a fresh manifest is created naming the
// installing requester.
func TestSharedSkillPackage_AdoptPolicyAdoptsUnmanifested(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, skillFileName), []byte("# someone else's skill\n"), 0o644); err != nil {
		t.Fatalf("seed foreign SKILL.md: %v", err)
	}

	var r WriteResult
	installSkillPackage(&r, dir, LocationGlobal, Claude, adoptUnmanifested)
	if len(r.Errors) != 0 {
		t.Fatalf("adopt install errors: %v", r.Errors)
	}

	wantContent, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if string(got) != string(wantContent) {
		t.Fatalf("SKILL.md not rewritten to embedded content under adopt policy")
	}
	assertFileResult(t, r.Files, filepath.Join(dir, skillFileName), ActionUpdated)

	m, present, err := readManifest(skillManifestPath(dir))
	if err != nil || !present {
		t.Fatalf("manifest not created under adopt policy: present=%v err=%v", present, err)
	}
	if !targetSetEqual(m.Targets, []TargetID{Claude}) {
		t.Fatalf("Targets = %v, want [claude]", m.Targets)
	}
}

// TestSharedSkillPackage_HandEditedOwnFileRewritten (D-16): a hand-edited
// own SKILL.md is silently rewritten to the embed on the next install; the
// hash mismatch is a drift signal, not a security event, so the manifest
// (unaffected by the drift) reports unchanged.
func TestSharedSkillPackage_HandEditedOwnFileRewritten(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")

	var r1 WriteResult
	installSkillPackage(&r1, dir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(r1.Errors) != 0 {
		t.Fatalf("first install errors: %v", r1.Errors)
	}
	manifestBefore, err := os.ReadFile(skillManifestPath(dir))
	if err != nil {
		t.Fatalf("read manifest before hand-edit: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, skillFileName), []byte("edited\n"), 0o644); err != nil {
		t.Fatalf("hand-edit SKILL.md: %v", err)
	}

	var r2 WriteResult
	installSkillPackage(&r2, dir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(r2.Errors) != 0 {
		t.Fatalf("second install errors: %v", r2.Errors)
	}

	wantContent, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if string(got) != string(wantContent) {
		t.Fatalf("hand-edited SKILL.md was not rewritten to embedded content")
	}
	assertFileResult(t, r2.Files, filepath.Join(dir, skillFileName), ActionUpdated)

	manifestAfter, err := os.ReadFile(skillManifestPath(dir))
	if err != nil {
		t.Fatalf("read manifest after hand-edit rewrite: %v", err)
	}
	if string(manifestBefore) != string(manifestAfter) {
		t.Fatalf("manifest bytes changed even though its own content did not: before=%q after=%q", manifestBefore, manifestAfter)
	}
}

// TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude (D-07
// planner amendment): a manifest written before this phase (schema 1, no
// targets key) or an outright corrupt manifest is both read as owned
// solely by Claude — never as "unknown" — so installing a second
// requester accumulates onto [claude], and removing that second requester
// leaves Claude's ownership (and the package) intact.
func TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude(t *testing.T) {
	t.Run("legacy schema_version 1, no targets key", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "codegraph")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		wantContent, err := claudeassets.SkillMarkdown()
		if err != nil {
			t.Fatalf("claudeassets.SkillMarkdown: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, skillFileName), wantContent, 0o644); err != nil {
			t.Fatalf("seed SKILL.md: %v", err)
		}
		legacy := `{"schema_version":1,"codegraph_version":"v0.10.0","installed_at":"2026-01-01T00:00:00Z","location":"global","files":{"skills/codegraph/SKILL.md":"` + hashContent(wantContent) + `"}}`
		if err := os.WriteFile(skillManifestPath(dir), []byte(legacy), 0o644); err != nil {
			t.Fatalf("seed legacy manifest: %v", err)
		}

		var r WriteResult
		installSkillPackage(&r, dir, LocationGlobal, Cursor, refuseUnmanifested)
		if len(r.Errors) != 0 {
			t.Fatalf("install over legacy manifest errors: %v", r.Errors)
		}
		m, present, err := readManifest(skillManifestPath(dir))
		if err != nil || !present {
			t.Fatalf("manifest not present after install: present=%v err=%v", present, err)
		}
		if !targetSetEqual(m.Targets, []TargetID{Claude, Cursor}) {
			t.Fatalf("Targets after install over legacy manifest = %v, want [claude cursor]", m.Targets)
		}

		var u WriteResult
		uninstallSkillPackage(&u, dir, Cursor, nil, refuseUnmanifested)
		if len(u.Errors) != 0 {
			t.Fatalf("uninstall cursor errors: %v", u.Errors)
		}
		m2, present2, err := readManifest(skillManifestPath(dir))
		if err != nil || !present2 {
			t.Fatalf("manifest not present after uninstall: present=%v err=%v", present2, err)
		}
		if !targetSetEqual(m2.Targets, []TargetID{Claude}) {
			t.Fatalf("Targets after uninstall cursor = %v, want [claude]", m2.Targets)
		}
		if !fileExists(filepath.Join(dir, skillFileName)) {
			t.Fatalf("SKILL.md removed even though claude still requests the package")
		}
	})

	t.Run("corrupt manifest self-heals", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "codegraph")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		wantContent, err := claudeassets.SkillMarkdown()
		if err != nil {
			t.Fatalf("claudeassets.SkillMarkdown: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, skillFileName), wantContent, 0o644); err != nil {
			t.Fatalf("seed SKILL.md: %v", err)
		}
		if err := os.WriteFile(skillManifestPath(dir), []byte("{not json"), 0o644); err != nil {
			t.Fatalf("seed corrupt manifest: %v", err)
		}

		var r WriteResult
		installSkillPackage(&r, dir, LocationGlobal, Cursor, refuseUnmanifested)
		if len(r.Errors) != 0 {
			t.Fatalf("install over corrupt manifest errors: %v", r.Errors)
		}
		m, present, err := readManifest(skillManifestPath(dir))
		if err != nil || !present {
			t.Fatalf("manifest not present after self-heal: present=%v err=%v", present, err)
		}
		if !targetSetEqual(m.Targets, []TargetID{Claude, Cursor}) {
			t.Fatalf("Targets after self-heal install = %v, want [claude cursor]", m.Targets)
		}
	})
}

// TestSharedSkillPackage_UserFileKeepsDir (must_haves.prohibitions): a
// user-authored file in the skill directory survives the last-requester
// uninstall, and so does the directory itself — only codegraph's own two
// files are removed.
func TestSharedSkillPackage_UserFileKeepsDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")

	var r WriteResult
	installSkillPackage(&r, dir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(r.Errors) != 0 {
		t.Fatalf("install errors: %v", r.Errors)
	}
	notesPath := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(notesPath, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("seed user file: %v", err)
	}

	var u WriteResult
	uninstallSkillPackage(&u, dir, Cursor, nil, refuseUnmanifested)
	if len(u.Errors) != 0 {
		t.Fatalf("uninstall errors: %v", u.Errors)
	}

	if fileExists(filepath.Join(dir, skillFileName)) {
		t.Fatalf("SKILL.md still present after last requester removed")
	}
	if fileExists(skillManifestPath(dir)) {
		t.Fatalf("manifest still present after last requester removed")
	}
	if !fileExists(notesPath) {
		t.Fatalf("user file was removed")
	}
	if !fileExists(dir) {
		t.Fatalf("skill dir was removed even though a user file remains")
	}
}

// TestSharedSkillPackage_ExclusiveKeysDroppedWhenRequesterLeaves (D-08):
// when the departing requester is the only one whose manifest keys were
// exclusive to it (e.g. Claude's hook-fragment/script keys, which no other
// target writes), uninstall drops exactly those keys from the manifest's
// Files map while leaving the shared SKILL.md key for the remaining
// requester.
func TestSharedSkillPackage_ExclusiveKeysDroppedWhenRequesterLeaves(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codegraph")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var r1 WriteResult
	recordSkillManifest(&r1, dir, LocationGlobal, Claude, map[string]string{
		manifestKeySkillMD:   "sha256:aaaa",
		manifestKeyScript:    "sha256:bbbb",
		manifestKeyHooksFrag: "sha256:cccc",
	})
	if len(r1.Errors) != 0 {
		t.Fatalf("seed recordSkillManifest errors: %v", r1.Errors)
	}

	var r2 WriteResult
	installSkillPackage(&r2, dir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(r2.Errors) != 0 {
		t.Fatalf("installSkillPackage errors: %v", r2.Errors)
	}
	m, present, err := readManifest(skillManifestPath(dir))
	if err != nil || !present {
		t.Fatalf("manifest not present: present=%v err=%v", present, err)
	}
	if !targetSetEqual(m.Targets, []TargetID{Claude, Cursor}) {
		t.Fatalf("Targets = %v, want [claude cursor]", m.Targets)
	}
	for _, key := range []string{manifestKeySkillMD, manifestKeyScript, manifestKeyHooksFrag} {
		if _, ok := m.Files[key]; !ok {
			t.Fatalf("Files missing key %q: %#v", key, m.Files)
		}
	}

	var u WriteResult
	uninstallSkillPackage(&u, dir, Claude, []string{manifestKeyScript, manifestKeyHooksFrag}, refuseUnmanifested)
	if len(u.Errors) != 0 {
		t.Fatalf("uninstall errors: %v", u.Errors)
	}
	m2, present2, err := readManifest(skillManifestPath(dir))
	if err != nil || !present2 {
		t.Fatalf("manifest not present after uninstall: present=%v err=%v", present2, err)
	}
	if !targetSetEqual(m2.Targets, []TargetID{Cursor}) {
		t.Fatalf("Targets after uninstall claude = %v, want [cursor]", m2.Targets)
	}
	if len(m2.Files) != 1 {
		t.Fatalf("Files after exclusive-key drop = %#v, want exactly the SKILL key", m2.Files)
	}
	if _, ok := m2.Files[manifestKeySkillMD]; !ok {
		t.Fatalf("Files missing manifestKeySkillMD after exclusive-key drop: %#v", m2.Files)
	}
}

// skillSeqOp is one step in TestSharedSkillPackage_TargetsInvariantOverAllSequences's
// exhaustive sequence walk: install or uninstall one target.
type skillSeqOp struct {
	install bool
	target  TargetID
}

// TestSharedSkillPackage_TargetsInvariantOverAllSequences (the
// assumption-delta's invariant_test): every sequence of length 1..4 over
// {install, uninstall} x {Cursor, Opencode, Gemini} — 6+36+216+1296 = 1554
// sequences, each replayed from a fresh temp dir — must leave the package
// present if and only if the modelled requester set is non-empty, with the
// manifest's targets set exactly matching that model at every step.
func TestSharedSkillPackage_TargetsInvariantOverAllSequences(t *testing.T) {
	targets := []TargetID{Cursor, Opencode, Gemini}
	var alphabet []skillSeqOp
	for _, tgt := range targets {
		alphabet = append(alphabet, skillSeqOp{install: true, target: tgt})
		alphabet = append(alphabet, skillSeqOp{install: false, target: tgt})
	}

	executed := 0
	var walk func(seq []skillSeqOp)
	walk = func(seq []skillSeqOp) {
		if len(seq) >= 1 {
			runSkillSequence(t, seq)
			executed++
		}
		if len(seq) == 4 {
			return
		}
		for _, o := range alphabet {
			next := make([]skillSeqOp, len(seq)+1)
			copy(next, seq)
			next[len(seq)] = o
			walk(next)
		}
	}
	walk(nil)

	if executed != 1554 {
		t.Fatalf("executed %d sequences, want 1554", executed)
	}
	t.Logf("executed exactly %d install/uninstall sequences", executed)
}

func runSkillSequence(t *testing.T, seq []skillSeqOp) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "codegraph")
	model := map[TargetID]bool{}

	for _, o := range seq {
		var result WriteResult
		if o.install {
			installSkillPackage(&result, dir, LocationGlobal, o.target, refuseUnmanifested)
			model[o.target] = true
		} else {
			uninstallSkillPackage(&result, dir, o.target, nil, refuseUnmanifested)
			delete(model, o.target)
		}
		if len(result.Errors) != 0 {
			t.Fatalf("sequence %+v: step %+v produced errors: %v", seq, o, result.Errors)
		}

		wantPresent := len(model) > 0
		gotPresent := fileExists(filepath.Join(dir, skillFileName)) && fileExists(skillManifestPath(dir))
		if gotPresent != wantPresent {
			t.Fatalf("sequence %+v: after step %+v, package present=%v, want model non-empty=%v", seq, o, gotPresent, wantPresent)
		}
		if wantPresent {
			m, present, err := readManifest(skillManifestPath(dir))
			if err != nil || !present {
				t.Fatalf("sequence %+v: after step %+v, manifest unreadable: present=%v err=%v", seq, o, present, err)
			}
			want := make([]TargetID, 0, len(model))
			for tgt := range model {
				want = append(want, tgt)
			}
			if !targetSetEqual(m.Targets, want) {
				t.Fatalf("sequence %+v: after step %+v, Targets = %v, want set %v", seq, o, m.Targets, want)
			}
		}
	}
}
