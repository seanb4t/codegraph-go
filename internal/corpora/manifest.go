// Package corpora is the sole pin authority for the FIXT-01/FIXT-02
// third-party measurement corpora: a strictly validated manifest naming
// every candidate repository's pinned commit, license and lock state, plus
// the collision-free out-of-tree destination path each entry fetches into.
//
// This is deliberately a SEPARATE package from tools/bench/realcorpus,
// this repository's other pinned-corpus manifest, rather than an extension
// of it. Two of the four reasons recorded in 01-04-PLAN.md's objective are
// load-bearing here: realcorpus performs no network I/O and only reports a
// path a caller must already have fetched (the opposite of what a
// Taskfile-driven fetch target needs), and realcorpus deliberately carries
// a BSD-3-Clause entry while Validate below enforces a strictly narrower
// MIT/Apache-2.0 bar — sharing one type would force a policy-parameterised
// validator or silently widen that bar. A reader who lands here from
// realcorpus's package doc, or vice versa, should find this paragraph.
//
// Every value this package reads from a manifest file is untrusted input:
// it is interpolated into a `git remote add` / `git fetch` shell
// invocation by the Taskfile fetch targets, so Validate applies a strict
// allowlist to every field that reaches a shell, never a blocklist.
package corpora

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// ErrInvalidSHA is returned by Validate when an entry's SHA is not exactly
// 40 lowercase hex characters. Tests assert on this sentinel's identity
// (errors.Is) rather than on message text, so the check can be reworded
// without breaking callers.
var ErrInvalidSHA = errors.New("corpora: invalid sha")

// ErrInvalidRepo is returned by Validate when an entry's Repo is not a
// strict single-slash org/name form drawn from [A-Za-z0-9._-]. Tests
// assert on this sentinel's identity rather than on message text.
var ErrInvalidRepo = errors.New("corpora: invalid repo")

// shaPattern and repoPattern are the two field allowlists Validate
// enforces, compiled once so the accepted character set is stated a
// single time and stays greppable. shaPattern accepts only exactly 40
// lowercase hex characters — no uppercase, no short/long form.
// repoPattern accepts exactly one slash separating two non-empty runs of
// [A-Za-z0-9._-] — no leading/trailing slash, no second slash, and no
// character (semicolon, backtick, dollar sign, space, newline, quote,
// pipe, ampersand, command-substitution opener) outside that set.
var (
	shaPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	repoPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)
)

// validLicenses is the FIXT-01 licence bar: MIT and Apache-2.0 only. This
// is deliberately NARROWER than tools/bench/realcorpus's manifest, which
// carries a BSD-3-Clause entry (cockroachdb-pebble) — see the package doc
// for why the two manifests are not merged over that difference.
var validLicenses = map[string]bool{
	"MIT":        true,
	"Apache-2.0": true,
}

// Entry is one candidate corpus: a repository this phase measures or has
// measured, whether or not it ends up locked into the final set.
type Entry struct {
	// Repo is the GitHub "org/name" slug — the sole allowed form is a
	// single slash separating two [A-Za-z0-9._-] runs (repoPattern).
	Repo string `json:"repo"`
	// SHA is the full 40-character lowercase hex commit this entry is
	// pinned to. Never a branch or tag name — see D-11's shallow-fetch
	// discipline in 01-04-PLAN.md.
	SHA string `json:"sha"`
	// License is this entry's SPDX identifier at SHA, resolved live from
	// the GitHub API licence endpoint — never a README badge, never
	// recalled from memory. Must be MIT or Apache-2.0 (validLicenses).
	License string `json:"license"`
	// Language records the primary language this candidate is nominated
	// to close a coverage gap for.
	Language string `json:"language"`
	// Locked is true once this entry has been promoted into the final
	// measured-and-justified corpus set. False entries remain in the
	// manifest — including candidates rejected before measurement,
	// per D-09 — recorded rather than deleted.
	Locked bool `json:"locked"`
	// Note carries free-form provenance: for a locked entry, the
	// coverage gap it closes; for an unlocked one, the reason it was
	// not selected.
	Note string `json:"note,omitempty"`
}

// Manifest is the full corpora/manifest.json document: the sole pin
// authority every reader (the Taskfile fetch loop, the CI cache path, the
// coverage drift guard) consults, per D-09.
type Manifest struct {
	// Note is a top-level provenance record: that this file is the sole
	// pin authority for this phase's corpora, the date its SHAs and
	// licences were resolved live, and a pointer to the OTHER pinned
	// manifest in this repository (tools/bench/realcorpus) so a reader
	// of one is led to the other.
	Note string `json:"note,omitempty"`
	// Corpora is every candidate entry, locked or not, in manifest
	// order.
	Corpora []Entry `json:"corpora"`
}

// Load reads path, decodes it as JSON into a Manifest, and Validates the
// result before returning it. A manifest with a duplicate repo, a
// malformed field, or an unknown license returns a non-nil error and a
// zero Manifest — never a partially populated one — so a caller can never
// observe a manifest that decoded but did not pass validation.
func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("corpora: read %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("corpora: decode %s: %w", path, err)
	}
	if err := Validate(m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// Validate rejects any manifest entry whose fields do not pass the strict
// allowlists above. This is the control that keeps a manifest-derived
// value safe to interpolate into the Taskfile fetch targets' shell
// commands: Validate runs before any value reaches a shell, and the
// Taskfile targets re-validate independently at the actual interpolation
// point as a second, last-line-of-defence check.
//
// The MIT/Apache-2.0 licence bar enforced here is deliberately NARROWER
// than tools/bench/realcorpus's manifest, which carries a BSD-3-Clause
// entry (cockroachdb-pebble) for the PERF-01 benchmark corpus — that
// difference in policy is exactly why the two manifests are not merged
// into one type; see the package doc for the other three reasons.
func Validate(m Manifest) error {
	seenRepo := make(map[string]bool, len(m.Corpora))
	for _, e := range m.Corpora {
		if !shaPattern.MatchString(e.SHA) {
			return fmt.Errorf("%w: entry %q: sha %q must be exactly 40 lowercase hex characters", ErrInvalidSHA, e.Repo, e.SHA)
		}
		if !repoPattern.MatchString(e.Repo) {
			return fmt.Errorf("%w: repo %q must be a single-slash org/name of [A-Za-z0-9._-] characters", ErrInvalidRepo, e.Repo)
		}
		if !validLicenses[e.License] {
			return fmt.Errorf("corpora: entry %q: license %q must be MIT or Apache-2.0", e.Repo, e.License)
		}
		if seenRepo[e.Repo] {
			return fmt.Errorf("corpora: duplicate repo %q", e.Repo)
		}
		seenRepo[e.Repo] = true
	}
	return nil
}

// Slug returns e's readable display form: Repo with its slash replaced by
// a hyphen. Slug is NOT collision-free on its own — "a-b/c" and "a/b-c"
// both produce "a-b-c" — which is exactly why Dir does not use Slug alone
// as a destination path component.
func (e Entry) Slug() string {
	out := make([]byte, len(e.Repo))
	for i := 0; i < len(e.Repo); i++ {
		if e.Repo[i] == '/' {
			out[i] = '-'
		} else {
			out[i] = e.Repo[i]
		}
	}
	return string(out)
}

// Dir returns e's destination directory under root: e's readable Slug, a
// separator, the first 8 hex characters of a SHA-256 digest over the
// canonical Repo string, an at-sign, and the pinned SHA. The digest makes
// the path collision-free by construction — "a-b/c" and "a/b-c" hash to
// different digests despite sharing a Slug — where a hand-reasoned
// "these characters cannot collide" argument over the slug alone would
// not be, and was the shape of bug this construction exists to avoid.
// Embedding SHA also means a pin bump changes the path outright, so a
// stale tree at a superseded pin can never be mistaken for the current
// one.
func (e Entry) Dir(root string) string {
	digest := sha256.Sum256([]byte(e.Repo))
	short := hex.EncodeToString(digest[:])[:8]
	return filepath.Join(root, fmt.Sprintf("%s-%s@%s", e.Slug(), short, e.SHA))
}

// CorpusRoot resolves the base directory fetched corpora land under: the
// caller's explicit override verbatim when set and non-empty; otherwise
// the XDG cache-home variable joined with "codegraph/corpora" when that
// is set and non-empty; otherwise the caller's home directory joined with
// ".cache/codegraph/corpora". This formula is applied literally and
// identically on every operating system: it does NOT use the standard
// library's own cache-directory resolver, because that resolver returns a
// platform-native location on Darwin instead of the formula above, and no
// per-operating-system branch belongs in this function — the same
// no-branch discipline internal/agents/opencode.go's config-directory
// resolution already applies to XDG_CONFIG_HOME.
func CorpusRoot() (string, error) {
	if v := os.Getenv("CODEGRAPH_CORPUS_DIR"); v != "" {
		return v, nil
	}
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "codegraph", "corpora"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("corpora: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".cache", "codegraph", "corpora"), nil
}

// LockedEntries returns only the entries of m whose Locked flag is true,
// in manifest order.
func LockedEntries(m Manifest) []Entry {
	out := make([]Entry, 0, len(m.Corpora))
	for _, e := range m.Corpora {
		if e.Locked {
			out = append(out, e)
		}
	}
	return out
}
