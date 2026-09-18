// Package agents (this file): the single manifest-owned writer for EVERY
// codegraph skill directory (05-02 assumption-delta decision: a written
// skill directory's manifest `targets` set IS its ownership identity — a
// single-requester directory, like Claude's own non-shared
// `.claude/skills/codegraph/`, is the one-element case of that same shape,
// not a second code path). Ownership of a directory is manifest-file
// PRESENCE only, never a SKILL.md's name or content — the exact
// differential closed by commit 242ec0a, which reverted a
// matcher-and-shape recovery heuristic after security review found it let
// codegraph silently claim and overwrite an unrelated, user-authored hook
// block whenever a codegraph manifest happened to already exist at that
// install location. No target calls this package's writer yet: 05-03 moves
// Claude onto it, 05-04/05-05 wire the remaining harnesses.
package agents

import (
	"fmt"
	"os"
	"path/filepath"

	claudeassets "github.com/seanb4t/codegraph-go"
	"github.com/seanb4t/codegraph-go/internal/version"
)

// ActionKeptForeign reports that a codegraph-named skill directory exists
// but carries no codegraph manifest — it is foreign, never overwritten or
// removed (D-14).
const ActionKeptForeign FileAction = "kept (foreign)"

// unmanifestedPolicy governs what installSkillPackage/uninstallSkillPackage
// do when a skill directory carries no codegraph manifest.
type unmanifestedPolicy int

const (
	// refuseUnmanifested treats an unmanifested, non-empty directory as
	// foreign: never written, never removed (D-14). Every target except
	// Claude's own non-shared skill dir uses this policy.
	refuseUnmanifested unmanifestedPolicy = iota
	// adoptUnmanifested claims an unmanifested directory's SKILL.md as
	// codegraph's own (the v0.10.0 Claude behaviour, reserved for Claude's
	// own non-shared skill dir, D-05).
	adoptUnmanifested
)

// sharedSkillDirPath resolves the shared skill package directory every
// non-Claude target (and, once 05-03 lands, Claude itself when its own
// directory coincides via D-17) writes into: `.agents/skills/codegraph`
// (local) / `<home>/.agents/skills/codegraph` (global).
func sharedSkillDirPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".agents", "skills", "codegraph"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".agents", "skills", "codegraph"), nil
}

// sharedSkillDirs wraps sharedSkillDirPath as a PathsFunc (the shape
// Capabilities.SkillDirs requires) — the shared directory is always the
// sole, written entry; there is no second read-only path.
func sharedSkillDirs(loc Location) ([]string, error) {
	dir, err := sharedSkillDirPath(loc)
	if err != nil {
		return nil, err
	}
	return []string{dir}, nil
}

// skillManifestPath is dir joined with the sidecar manifest filename.
func skillManifestPath(dir string) string {
	return filepath.Join(dir, skillManifestFileName)
}

// skillDirIsForeign reports whether dir is a codegraph-named directory
// codegraph does not own (D-14): a non-existent dir is never foreign (there
// is nothing to protect); an existing dir carrying a codegraph manifest —
// however stale, corrupt, or drifted — is never foreign, since manifest
// PRESENCE alone is the ownership signal (never a SKILL.md's name or
// shape, the 242ec0a differential); an existing dir with no manifest is
// foreign only if it is non-empty (an empty dir is unclaimed territory,
// not someone else's content).
func skillDirIsForeign(dir string) (bool, error) {
	if !fileExists(dir) {
		return false, nil
	}
	if fileExists(skillManifestPath(dir)) {
		return false, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

// writeSkillFile writes the embedded SKILL.md into dir under policy: under
// refuseUnmanifested, a foreign dir (D-14) is left completely untouched
// and reported `kept (foreign)`; otherwise the embedded content is written
// through writeEmbeddedFile/recordFile exactly like every other embedded
// artifact this package writes, so a hand-edited own file is silently
// rewritten (D-16) rather than treated as tamper. Returns the written
// content and whether the write succeeded — installSkillPackage only
// records a manifest hash when ok is true (CR-01: never claim a write that
// did not happen).
func writeSkillFile(result *WriteResult, dir string, policy unmanifestedPolicy) ([]byte, bool) {
	if policy == refuseUnmanifested {
		foreign, err := skillDirIsForeign(dir)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, err))
			return nil, false
		}
		if foreign {
			result.Files = append(result.Files, FileResult{Path: dir, Action: ActionKeptForeign})
			return nil, false
		}
	}

	content, err := claudeassets.SkillMarkdown()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, err))
		return nil, false
	}

	skillPath := filepath.Join(dir, skillFileName)
	fr, werr := writeEmbeddedFile(skillPath, string(content), false)
	recordFile(result, skillPath, fr, werr)
	return content, werr == nil
}

// containsTarget reports whether id is present in ids.
func containsTarget(ids []TargetID, id TargetID) bool {
	for _, t := range ids {
		if t == id {
			return true
		}
	}
	return false
}

// recordSkillManifest reads dir's existing manifest (if any), folds
// requester into its requester set via manifestRequesters (D-07's
// legacy/corrupt-reads-as-Claude rule applies here), merges ownFiles into
// the existing Files map without disturbing any other requester's
// recorded keys, and writes the result through writeManifest/recordFile.
// The manifest is always read fresh — never memoized across calls in the
// same install run — so the SECOND and later targets in one run correctly
// see and extend the FIRST target's requester set (RESEARCH "Anti-Patterns
// to Avoid": a package-level "already wrote this run" flag would silently
// drop every requester after the first).
func recordSkillManifest(result *WriteResult, dir string, loc Location, requester TargetID, ownFiles map[string]string) {
	manifestPath := skillManifestPath(dir)
	existing, present, rerr := readManifest(manifestPath)

	requesters := manifestRequesters(existing, present, rerr)
	if !containsTarget(requesters, requester) {
		requesters = append(requesters, requester)
	}

	files := make(map[string]string, len(existing.Files)+len(ownFiles))
	for k, v := range existing.Files {
		files[k] = v
	}
	for k, v := range ownFiles {
		files[k] = v
	}

	m := skillManifest{
		SchemaVersion:    manifestSchemaVersion,
		CodegraphVersion: version.Info().Version,
		Location:         string(loc),
		Files:            files,
		Targets:          requesters,
	}
	fr, werr := writeManifest(manifestPath, m)
	recordFile(result, manifestPath, fr, werr)
}

// installSkillPackage is the single manifest-owned install path every
// skill-writing target funnels through (D-05): write the shared SKILL.md
// (or refuse on a foreign dir), then — only if that write actually
// succeeded — record requester's ownership of the manifestKeySkillMD hash
// in the sidecar manifest. A write that did not happen never earns a
// recorded hash (CR-01's have-flag rule, mirrored from claude.go's own
// Install).
func installSkillPackage(result *WriteResult, dir string, loc Location, requester TargetID, policy unmanifestedPolicy) {
	content, ok := writeSkillFile(result, dir, policy)
	if !ok {
		return
	}
	recordSkillManifest(result, dir, loc, requester, map[string]string{
		manifestKeySkillMD: hashContent(content),
	})
}

// uninstallSkillPackage is installSkillPackage's mirror (D-08): a manifest
// absent entirely defers to policy (refuse: foreign dirs are kept
// untouched, everything else reports not-found; adopt: SKILL.md is removed
// the way v0.10.0's Claude-only uninstall always did, then the dir is swept
// if now empty). A manifest present but not naming requester is a no-op
// reporting not-found for both files. Removing requester from a manifest
// naming MULTIPLE requesters rewrites the manifest with the remaining set
// and drops requester's exclusiveKeys from Files, keeping SKILL.md (report
// `kept` — still needed). Removing the LAST requester deletes the manifest
// first (so the directory-empty sweep below sees it already gone), then
// SKILL.md, then sweeps the directory via removeSkillDirIfEmpty — never a
// recursive delete, so a user's own file in the directory survives.
func uninstallSkillPackage(result *WriteResult, dir string, requester TargetID, exclusiveKeys []string, policy unmanifestedPolicy) {
	manifestPath := skillManifestPath(dir)
	skillPath := filepath.Join(dir, skillFileName)

	existing, present, rerr := readManifest(manifestPath)

	if !present && rerr == nil {
		if policy == refuseUnmanifested {
			foreign, ferr := skillDirIsForeign(dir)
			if ferr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, ferr))
				return
			}
			if foreign {
				result.Files = append(result.Files, FileResult{Path: dir, Action: ActionKeptForeign})
				return
			}
			result.Files = append(result.Files,
				FileResult{Path: manifestPath, Action: ActionNotFound},
				FileResult{Path: skillPath, Action: ActionNotFound},
			)
			return
		}
		// adoptUnmanifested: the v0.10.0 Claude-only behaviour — a
		// manifest-less directory's SKILL.md is still codegraph's own to
		// remove, then swept if it leaves the directory empty.
		fr, serr := removeEmbeddedFile(skillPath)
		recordFile(result, skillPath, fr, serr)
		if err := removeSkillDirIfEmpty(dir); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, err))
		}
		return
	}

	requesters := manifestRequesters(existing, present, rerr)
	if !containsTarget(requesters, requester) {
		result.Files = append(result.Files,
			FileResult{Path: manifestPath, Action: ActionNotFound},
			FileResult{Path: skillPath, Action: ActionNotFound},
		)
		return
	}

	remaining := make([]TargetID, 0, len(requesters))
	for _, r := range requesters {
		if r != requester {
			remaining = append(remaining, r)
		}
	}

	if len(remaining) == 0 {
		mfr, merr := removeEmbeddedFile(manifestPath)
		recordFile(result, manifestPath, mfr, merr)
		sfr, serr := removeEmbeddedFile(skillPath)
		recordFile(result, skillPath, sfr, serr)
		if err := removeSkillDirIfEmpty(dir); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, err))
		}
		return
	}

	exclusive := make(map[string]bool, len(exclusiveKeys))
	for _, k := range exclusiveKeys {
		exclusive[k] = true
	}
	files := make(map[string]string, len(existing.Files))
	for k, v := range existing.Files {
		if !exclusive[k] {
			files[k] = v
		}
	}
	m := skillManifest{
		SchemaVersion:    manifestSchemaVersion,
		CodegraphVersion: version.Info().Version,
		Location:         existing.Location,
		Files:            files,
		Targets:          remaining,
	}
	fr, werr := writeManifest(manifestPath, m)
	recordFile(result, manifestPath, fr, werr)
	result.Files = append(result.Files, FileResult{Path: skillPath, Action: ActionKept})
}
