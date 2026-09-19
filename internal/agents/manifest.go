// Package agents (this file): the sidecar manifest written next to
// SKILL.md at install time (D-03, D-04). The manifest records which
// binary version wrote the installed skill package and a content hash for
// every artifact codegraph wrote, so "which version is installed" is
// observable from the installed files rather than inferred from whatever
// binary happens to be on $PATH, and so codegraph upgrade has something to
// discover (AGENT-03).
//
// The manifest's hash is a drift signal only — it is NOT a
// tamper-detection or authenticity control. It protects a file any local
// process can freely rewrite, and carries no signature: a hash mismatch
// means "codegraph's own content was hand-edited since the last write,"
// nothing more. D-05 fixes the response to a mismatch as a silent
// overwrite, deliberately with no prompt, no warning, and no new flag. No
// code in this package may treat a hash mismatch as a security event.
package agents

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Stable manifest file keys — artifact identities, not absolute machine
// paths, so the same manifest schema reads sensibly regardless of
// --location. The third key's trailing "#hooks.SessionStart" fragment is
// deliberate: it names a portion of settings.json, since the hooks
// registration is not a whole physical file codegraph owns (RESEARCH
// Pitfall 5) — hashing the whole shared file would report drift on every
// unrelated user edit.
const (
	manifestKeySkillMD   = "skills/codegraph/SKILL.md"
	manifestKeyScript    = "hooks/session-nudge.sh"
	manifestKeyHooksFrag = "settings.json#hooks.SessionStart"
	// manifestSchemaVersion history: 1 = v0.10.0 Phase 7 through v0.14.0
	// Phase 4 — a single, Claude-only writer with no requester set. 2 =
	// the requester set added by 05-02 (D-07): skillManifest.Targets, the
	// ownership identity a shared skill directory needs once more than one
	// target can request it. Bumping this is flagged `costly` in
	// 05-02-PLAN.md: a released binary older than this change reads (and,
	// if it writes, rewrites) a schema-2 manifest at schema 1, silently
	// dropping Targets — self-healed on the next new-binary install via
	// manifestRequesters' "nil Targets read as [claude]" rule below, with
	// the loss mode confined to D-17's symlinked layouts.
	manifestSchemaVersion = 2
)

// skillManifest is the on-disk shape of <skillDir>/.codegraph-manifest.json.
// location is kept even though the manifest's own path already encodes it
// (RESEARCH Open Question #2) — it costs nothing, keeps the record
// self-describing for any future `codegraph install --status` surface, and
// avoids a bug class where discovery logic that no longer knows which
// candidate path matched would have to re-derive it.
type skillManifest struct {
	SchemaVersion    int               `json:"schema_version"`
	CodegraphVersion string            `json:"codegraph_version"`
	InstalledAt      string            `json:"installed_at"`
	Location         string            `json:"location"`
	Files            map[string]string `json:"files"`
	// Targets is the requester set that owns this manifest's skill
	// directory (D-07, 05-02) — the ownership identity a shared skill
	// package (AGENT-09) needs, since more than one target can request the
	// same directory. omitempty keeps a manifest written by a target that
	// never populates this field (none yet — 05-03 moves Claude onto the
	// shared writer) byte-identical to schema_version 1's shape.
	Targets []TargetID `json:"targets,omitempty"`
}

// manifestRequesters folds readManifest's three possible outcomes into the
// single non-destructive reading D-07's planner amendment requires: an
// unreadable manifest (readErr != nil), or a present-and-decodable one
// whose Targets field is nil (a genuine schema_version 1 write, or any
// hand-authored manifest predating D-07), returns unreadableFallback
// unchanged — the caller decides what "unreadable" means at its own
// directory, since that reading is only justified where a pre-phase
// manifest could genuinely exist (code review CR-01, 05-REVIEW.md):
// Claude's own directory, and the shared `.agents/skills/codegraph`
// directory reached through D-17's symlink-aware path, where "assume
// Claude" avoids a later uninstall deleting a package Claude still
// legitimately owns out from under it. A harness-exclusive directory
// (Gemini, Kiro, Antigravity) Claude never wrote to under any schema has
// no such ambiguity, so its caller passes a fallback that self-heals to
// the actual requester instead of inventing Claude as a phantom co-owner
// that can never legitimately relinquish ownership. An absent manifest
// (present == false) has no requesters at all — there is nothing to own.
// Only a present manifest that already carries a Targets field returns
// that set, copied so a caller mutating the returned slice can never
// corrupt the manifest's own backing array.
func manifestRequesters(m skillManifest, present bool, readErr error, unreadableFallback []TargetID) []TargetID {
	if readErr != nil {
		return unreadableFallback
	}
	if !present {
		return nil
	}
	if m.Targets == nil {
		return unreadableFallback
	}
	out := make([]TargetID, len(m.Targets))
	copy(out, m.Targets)
	return out
}

// targetSetEqual reports whether a and b contain the same TargetIDs,
// ignoring order and duplicate count (D-07: targets is written in
// first-install order but compared as a SET, so a re-run in any install
// order is a byte-level no-op).
func targetSetEqual(a, b []TargetID) bool {
	setA := make(map[TargetID]bool, len(a))
	for _, t := range a {
		setA[t] = true
	}
	setB := make(map[TargetID]bool, len(b))
	for _, t := range b {
		setB[t] = true
	}
	if len(setA) != len(setB) {
		return false
	}
	for t := range setA {
		if !setB[t] {
			return false
		}
	}
	return true
}

// hashContent returns "sha256:" followed by the lowercase hex encoding of
// sha256.Sum256(b) — the same algorithm this codebase already uses for
// binary-integrity hashing (internal/upgrade/upgrade.go), reused here
// rather than introducing a second hash convention into the same binary.
func hashContent(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// hashOwnedHookBlocks hashes only the codegraph-owned SessionStart blocks
// passed in — never the containing settings.json. blocks is run through
// normalizeJSON first so a value built from concrete Go types compares and
// marshals identically to one readJSONFileStrict decoded into generic
// map[string]any/[]any; encoding/json's own map-key sorting then makes the
// resulting byte sequence deterministic for a given structure regardless of
// original key-insertion order.
func hashOwnedHookBlocks(blocks []any) (string, error) {
	normalized, err := normalizeJSON(blocks)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return hashContent(data), nil
}

// readManifest parses path as a skillManifest, distinguishing three
// outcomes exactly like readJSONFileStrict's contract: an absent file
// yields a zero manifest and false with no error; an unreadable or
// undecodable file yields an error (never a zero-valued fallback — a
// manifest codegraph cannot read is a manifest whose recorded hashes
// cannot be trusted to decide anything); a present, decodable file yields
// the manifest and true.
func readManifest(path string) (skillManifest, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return skillManifest{}, false, nil
		}
		return skillManifest{}, false, fmt.Errorf("could not read existing manifest: %w", err)
	}
	var m skillManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return skillManifest{}, false, fmt.Errorf("%s: existing manifest is not valid JSON — fix or remove it manually: %w", path, err)
	}
	return m, true, nil
}

// stringMapEqual reports whether a and b contain the same keys mapped to
// the same values.
func stringMapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// writeManifest writes m to path, unless an existing manifest at path
// already has the same schema_version, codegraph_version, and files map —
// in which case it returns ActionUnchanged and writes nothing, deliberately
// preserving the stored installed_at rather than refreshing it. This is
// load-bearing: a manifest that stamped the current clock on every run
// would make a second `install` produce a different file and break
// AGENT-01's idempotency. installed_at's honest meaning is "when this
// content was installed," not "when install last ran."
//
// A corrupted or unreadable existing manifest is self-healed by
// overwriting, not treated as a blocking error: unlike settings.json
// (which this package only partially owns, so refusing to touch malformed
// content protects a third party's data), the manifest is a wholly
// codegraph-owned dot-prefixed sidecar with nothing to protect — the same
// category as SKILL.md/session-nudge.sh, both of which silently restore
// on drift per D-05 (code review WR-01). Leaving it permanently blocked
// would make every future install/upgrade report a spurious failure until
// a human manually deletes the file.
func writeManifest(path string, m skillManifest) (FileResult, error) {
	existing, existedBefore, err := readManifest(path)
	if err != nil {
		existing = skillManifest{}
		existedBefore = false
	}
	if existedBefore &&
		existing.SchemaVersion == m.SchemaVersion &&
		existing.CodegraphVersion == m.CodegraphVersion &&
		existing.Location == m.Location &&
		stringMapEqual(existing.Files, m.Files) &&
		targetSetEqual(existing.Targets, m.Targets) {
		return FileResult{Path: path, Action: ActionUnchanged}, nil
	}

	m.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return FileResult{}, err
	}
	if err := atomicWriteFile(path, string(out)+"\n"); err != nil {
		return FileResult{}, err
	}
	action := ActionCreated
	if existedBefore {
		action = ActionUpdated
	}
	return FileResult{Path: path, Action: action}, nil
}

// ConfiguredSkillLocations reports every location that carries evidence of
// a prior codegraph install FOR id — a readable manifest naming id among
// its requesters, OR one that exists but failed to parse — by probing the
// two fixed candidate manifest paths, never by walking the filesystem.
//
// D-17 changed what "a manifest exists at Claude's path" can mean: since a
// symlinked shared skill directory makes Claude's path and another
// target's shared directory the SAME physical file, a manifest can now
// exist there because Cursor or opencode alone requested the shared
// package — proof that THOSE agents were configured, not proof Claude
// was. `codegraph upgrade`'s refresh step must never install Claude at a
// location the user never asked it to configure (T-05-13), so manifest
// presence alone is no longer sufficient: id must actually be among the
// manifest's requesters (manifestRequesters).
//
// A present-but-corrupted manifest is still proof the location was
// configured before, exactly as much proof as a readable one naming id —
// excluding it would let a corrupted manifest silently drop that location
// from every future refresh with no warning anywhere (code review WR-04),
// even though writeManifest self-heals a corrupted manifest the moment
// Install() next runs there. A legacy manifest (schema_version 1, no
// targets key) reads as owned by [Claude] via manifestRequesters' D-07
// rule, so it is included too — both are read-error/nil-Targets cases
// manifestRequesters already folds into "assume Claude," and this
// function trusts that folding rather than re-deriving it. Only a
// genuinely absent manifest (no error, not present), or one present and
// decodable but naming OTHER requesters without id, means "never
// configured for id" and is excluded.
//
// Exported because Plan 04's CLI-layer upgrade refresh needs it. Returns
// nil for any target id other than Claude, since this phase is
// Claude-only by scope: discovery is two stat calls for a phase
// deliberately narrowed to one agent, and anything more general is
// unneeded generality here.
func ConfiguredSkillLocations(id TargetID) []Location {
	if id != Claude {
		return nil
	}
	var locs []Location
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		path, err := claudeManifestPath(loc)
		if err != nil {
			continue
		}
		m, present, rerr := readManifest(path)
		if rerr != nil {
			// WR-04: unreadable/corrupt is still proof of a prior
			// configuration — never silently dropped.
			locs = append(locs, loc)
			continue
		}
		if !present {
			continue
		}
		if containsTarget(manifestRequesters(m, present, rerr, []TargetID{Claude}), id) {
			locs = append(locs, loc)
		}
	}
	return locs
}
