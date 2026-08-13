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
	manifestKeySkillMD    = "skills/codegraph/SKILL.md"
	manifestKeyScript     = "hooks/session-nudge.sh"
	manifestKeyHooksFrag  = "settings.json#hooks.SessionStart"
	manifestSchemaVersion = 1
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
func writeManifest(path string, m skillManifest) (FileResult, error) {
	existing, existedBefore, err := readManifest(path)
	if err != nil {
		return FileResult{}, err
	}
	if existedBefore &&
		existing.SchemaVersion == m.SchemaVersion &&
		existing.CodegraphVersion == m.CodegraphVersion &&
		existing.Location == m.Location &&
		stringMapEqual(existing.Files, m.Files) {
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

// ConfiguredSkillLocations reports exactly the locations that currently
// carry a readable codegraph manifest for id, by probing the two fixed
// candidate manifest paths — never by walking the filesystem. Exported
// because Plan 04's CLI-layer upgrade refresh needs it. Returns nil for
// any target id other than Claude, since this phase is Claude-only by
// scope: discovery is two stat calls for a phase deliberately narrowed to
// one agent, and anything more general is unneeded generality here.
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
		_, present, err := readManifest(path)
		if err != nil || !present {
			continue
		}
		locs = append(locs, loc)
	}
	return locs
}
