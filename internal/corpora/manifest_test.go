package corpora

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const validSHA = "0123456789abcdef0123456789abcdef01234567"

func validEntry() Entry {
	return Entry{
		Repo:     "gohugoio/hugo",
		SHA:      validSHA,
		License:  "Apache-2.0",
		Language: "go",
		Locked:   false,
		Note:     "roadmap shortlist",
	}
}

// TestManifestLoadRoundTrip proves a well-formed manifest decodes and
// validates cleanly, and that every field survives the JSON round trip
// unchanged.
func TestManifestLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	const doc = `{
		"note": "test manifest",
		"corpora": [
			{
				"repo": "gohugoio/hugo",
				"sha": "0123456789abcdef0123456789abcdef01234567",
				"license": "Apache-2.0",
				"language": "go",
				"locked": true,
				"note": "roadmap shortlist"
			}
		]
	}`
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(m.Corpora) != 1 {
		t.Fatalf("len(m.Corpora) = %d, want 1", len(m.Corpora))
	}
	e := m.Corpora[0]
	if e.Repo != "gohugoio/hugo" || e.SHA != validSHA || e.License != "Apache-2.0" ||
		e.Language != "go" || !e.Locked || e.Note != "roadmap shortlist" {
		t.Fatalf("round-tripped entry mismatch: %+v", e)
	}
	if m.Note != "test manifest" {
		t.Fatalf("m.Note = %q, want %q", m.Note, "test manifest")
	}
}

// TestManifestRejectsMalformedEntry proves Validate accepts a well-formed
// entry and rejects a sha that is empty, too short, too long, or carries
// an uppercase hex letter — each naming the offending entry.
func TestManifestRejectsMalformedEntry(t *testing.T) {
	good := Manifest{Corpora: []Entry{validEntry()}}
	if err := Validate(good); err != nil {
		t.Fatalf("Validate(well-formed) = %v, want nil", err)
	}

	cases := []struct {
		name string
		sha  string
	}{
		{"empty", ""},
		{"too short", validSHA[:39]},
		{"too long", validSHA + "0"},
		{"uppercase hex letter", "0123456789ABCDEF0123456789abcdef01234567"[:40]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := validEntry()
			e.SHA = tc.sha
			m := Manifest{Corpora: []Entry{e}}
			err := Validate(m)
			if err == nil {
				t.Fatalf("Validate(sha=%q) = nil, want error", tc.sha)
			}
			if !errors.Is(err, ErrInvalidSHA) {
				t.Errorf("Validate(sha=%q) = %v, want ErrInvalidSHA", tc.sha, err)
			}
		})
	}
}

// TestManifestRejectsShellMetacharacters drives Validate from a table of
// hostile repo payloads — one subtest per payload, so a single "rejects
// bad input" case cannot pass while most payloads slip through.
func TestManifestRejectsShellMetacharacters(t *testing.T) {
	payloads := []struct {
		name string
		repo string
	}{
		{"semicolon", "org/name;rm"},
		{"backtick", "org/name`whoami`"},
		{"dollar sign", "org/name$HOME"},
		{"space", "org/name here"},
		{"newline", "org/name\nrm -rf"},
		{"single quote", "org/name'"},
		{"double quote", "org/name\""},
		{"pipe", "org/name|cat"},
		{"ampersand", "org/name&sleep"},
		{"command substitution opener", "org/name$(whoami)"},
	}
	if len(payloads) < 9 {
		t.Fatalf("payload table has only %d entries, want at least 9", len(payloads))
	}
	for _, p := range payloads {
		t.Run(p.name, func(t *testing.T) {
			e := validEntry()
			e.Repo = p.repo
			m := Manifest{Corpora: []Entry{e}}
			err := Validate(m)
			if err == nil {
				t.Fatalf("Validate(repo=%q) = nil, want error", p.repo)
			}
			if !errors.Is(err, ErrInvalidRepo) {
				t.Errorf("Validate(repo=%q) = %v, want ErrInvalidRepo", p.repo, err)
			}
		})
	}
}

// TestManifestRejectsUnknownLicense proves the MIT/Apache-2.0 bar is
// enforced, including a BSD-3-Clause case — the exact licence
// tools/bench/realcorpus's manifest carries for cockroachdb-pebble, which
// this stricter bar deliberately rejects.
func TestManifestRejectsUnknownLicense(t *testing.T) {
	cases := []string{"BSD-3-Clause", "GPL-3.0", "", "mit"}
	for _, lic := range cases {
		t.Run(lic, func(t *testing.T) {
			e := validEntry()
			e.License = lic
			m := Manifest{Corpora: []Entry{e}}
			if err := Validate(m); err == nil {
				t.Fatalf("Validate(license=%q) = nil, want error", lic)
			}
		})
	}
}

// TestManifestRejectsDuplicateRepo proves two entries naming the same
// repository fail validation.
func TestManifestRejectsDuplicateRepo(t *testing.T) {
	a := validEntry()
	b := validEntry()
	b.SHA = "fedcba9876543210fedcba9876543210fedcba9"
	m := Manifest{Corpora: []Entry{a, b}}
	if err := Validate(m); err == nil {
		t.Fatal("Validate(duplicate repo) = nil, want error")
	}
}

// TestEntryDirIsCollisionFree proves the "a-b/c" versus "a/b-c" pair —
// both permitted by the repo grammar, both producing the identical Slug
// "a-b-c" — resolve to DIFFERENT directories at the same pinned SHA.
func TestEntryDirIsCollisionFree(t *testing.T) {
	e1 := Entry{Repo: "a-b/c", SHA: validSHA}
	e2 := Entry{Repo: "a/b-c", SHA: validSHA}

	if e1.Slug() != e2.Slug() {
		t.Fatalf("test premise violated: Slug()s differ (%q vs %q) — collision case not exercised", e1.Slug(), e2.Slug())
	}

	root := "/corpus-root"
	d1 := e1.Dir(root)
	d2 := e2.Dir(root)
	if d1 == d2 {
		t.Fatalf("Entry.Dir collided: %q == %q for distinct repos %q and %q", d1, d2, e1.Repo, e2.Repo)
	}
}

// TestEntryDirEmbedsPinnedSHA proves a pin bump changes the destination
// path, so a stale tree at a superseded pin can never be mistaken for the
// current one.
func TestEntryDirEmbedsPinnedSHA(t *testing.T) {
	e := validEntry()
	root := "/corpus-root"
	before := e.Dir(root)

	e.SHA = "fedcba9876543210fedcba9876543210fedcba9"
	after := e.Dir(root)

	if before == after {
		t.Fatalf("Entry.Dir did not change after a SHA bump: %q", before)
	}
}

// TestCorpusRootHonorsXDGAndOverride proves all three CorpusRoot
// resolution branches: explicit override, XDG cache-home fallback, and
// the bare home-directory fallback.
func TestCorpusRootHonorsXDGAndOverride(t *testing.T) {
	t.Run("explicit override wins", func(t *testing.T) {
		t.Setenv("CODEGRAPH_CORPUS_DIR", "/override/corpora")
		t.Setenv("XDG_CACHE_HOME", "/xdg/cache")
		got, err := CorpusRoot()
		if err != nil {
			t.Fatalf("CorpusRoot: %v", err)
		}
		if got != "/override/corpora" {
			t.Errorf("CorpusRoot = %q, want %q", got, "/override/corpora")
		}
	})

	t.Run("XDG_CACHE_HOME fallback", func(t *testing.T) {
		t.Setenv("CODEGRAPH_CORPUS_DIR", "")
		t.Setenv("XDG_CACHE_HOME", "/xdg/cache")
		got, err := CorpusRoot()
		if err != nil {
			t.Fatalf("CorpusRoot: %v", err)
		}
		want := filepath.Join("/xdg/cache", "codegraph", "corpora")
		if got != want {
			t.Errorf("CorpusRoot = %q, want %q", got, want)
		}
	})

	t.Run("home directory fallback", func(t *testing.T) {
		t.Setenv("CODEGRAPH_CORPUS_DIR", "")
		t.Setenv("XDG_CACHE_HOME", "")
		home := t.TempDir()
		t.Setenv("HOME", home)
		got, err := CorpusRoot()
		if err != nil {
			t.Fatalf("CorpusRoot: %v", err)
		}
		want := filepath.Join(home, ".cache", "codegraph", "corpora")
		if got != want {
			t.Errorf("CorpusRoot = %q, want %q", got, want)
		}
	})
}

// TestLockedEntriesFiltersAndPreservesOrder proves LockedEntries returns
// only locked entries, in manifest order.
func TestLockedEntriesFiltersAndPreservesOrder(t *testing.T) {
	a := validEntry()
	a.Repo = "org/a"
	a.Locked = true
	b := validEntry()
	b.Repo = "org/b"
	b.Locked = false
	c := validEntry()
	c.Repo = "org/c"
	c.Locked = true

	m := Manifest{Corpora: []Entry{a, b, c}}
	got := LockedEntries(m)
	if len(got) != 2 {
		t.Fatalf("LockedEntries returned %d entries, want 2: %+v", len(got), got)
	}
	if got[0].Repo != "org/a" || got[1].Repo != "org/c" {
		t.Fatalf("LockedEntries order = [%s, %s], want [org/a, org/c]", got[0].Repo, got[1].Repo)
	}
}
