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

	// D-17 / RESEARCH Pitfall 2: a dangling directory symlink (dir exists
	// as a link, but its target does not yet exist — e.g. a user's
	// `npx skills`-managed `~/.claude/skills/codegraph ->
	// ../../.agents/skills/codegraph`) makes writeEmbeddedFile's own
	// os.MkdirAll(filepath.Dir(skillPath), ...) fail: the mkdir syscall
	// sees an existing directory ENTRY (the symlink itself) at that path
	// and refuses to create anything there, even though Stat-following
	// the link reports "does not exist." Pre-creating the resolved target
	// directory keeps the user's link working instead of erroring. The
	// FileResult path recorded below still uses the caller's own dir (the
	// link) — not the resolved target — for both a non-link and a live
	// link, and this branch changes nothing about it either.
	if info, lerr := os.Lstat(dir); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
		if _, statErr := os.Stat(dir); statErr != nil && os.IsNotExist(statErr) {
			target, rerr := resolveSkillDir(dir)
			if rerr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, rerr))
				return nil, false
			}
			if merr := os.MkdirAll(target, 0o755); merr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, merr))
				return nil, false
			}
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
//
// recordSkillManifest itself always assumes Claude on an unreadable
// manifest, preserving every existing caller's behavior exactly (Claude's
// own directory, and every direct test call, which always exercises the
// shared directory where that assumption is justified). A harness-
// exclusive directory (CR-01, 05-REVIEW.md) must use
// recordSkillManifestWithFallback directly instead, with a fallback that
// does not invent Claude as a phantom co-owner.
func recordSkillManifest(result *WriteResult, dir string, loc Location, requester TargetID, ownFiles map[string]string) {
	recordSkillManifestWithFallback(result, dir, loc, requester, ownFiles, []TargetID{Claude})
}

// recordSkillManifestWithFallback is recordSkillManifest's general form,
// taking the "unreadable manifest" fallback explicit at the call site
// (CR-01) instead of hard-coding Claude unconditionally.
func recordSkillManifestWithFallback(result *WriteResult, dir string, loc Location, requester TargetID, ownFiles map[string]string, unreadableFallback []TargetID) {
	manifestPath := skillManifestPath(dir)
	existing, present, rerr := readManifest(manifestPath)

	requesters := manifestRequesters(existing, present, rerr, unreadableFallback)
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
//
// D-17's "both writers compare" requirement, the shared-writer side: for
// any requester other than Claude, dir may coincide with Claude's own
// skill directory (a symlink — the `npx skills` convention). When it does,
// a single advisory note is appended naming both paths, so a user who only
// ever looks at (say) Cursor's install output still learns that one
// physical package now serves both agents. This is purely informational —
// it changes nothing about what gets written; claude.go's own
// claudeSkillPolicy is what actually governs correctness on Claude's side
// of the comparison. A comparison error is recorded in result.Errors
// rather than silently dropped.
//
// installSkillPackage always assumes Claude on an unreadable manifest
// (recordSkillManifest's default), matching every existing caller: it is
// only ever invoked directly at the shared directory (Cursor's and
// opencode's declared skill directory IS the shared path), where that
// assumption is justified. installDeclaredSkill uses
// installSkillPackageWithFallback directly for CR-01's harness-exclusive
// case.
func installSkillPackage(result *WriteResult, dir string, loc Location, requester TargetID, policy unmanifestedPolicy) {
	installSkillPackageWithFallback(result, dir, loc, requester, policy, []TargetID{Claude})
}

// installSkillPackageWithFallback is installSkillPackage's general form,
// taking the "unreadable manifest" fallback explicit at the call site
// (CR-01) instead of hard-coding Claude unconditionally.
func installSkillPackageWithFallback(result *WriteResult, dir string, loc Location, requester TargetID, policy unmanifestedPolicy, unreadableFallback []TargetID) {
	if requester != Claude {
		if claudeDir, err := claudeSkillDirPath(loc); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, err))
		} else if same, serr := sameSkillDir(dir, claudeDir); serr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, serr))
		} else if same {
			result.Notes = append(result.Notes, fmt.Sprintf(
				"%s is the same directory as Claude Code's %s — one skill package, one manifest listing every agent that installed it",
				dir, claudeDir))
		}
	}

	content, ok := writeSkillFile(result, dir, policy)
	if !ok {
		return
	}
	recordSkillManifestWithFallback(result, dir, loc, requester, map[string]string{
		manifestKeySkillMD: hashContent(content),
	}, unreadableFallback)
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
//
// uninstallSkillPackage always assumes Claude on an unreadable manifest,
// matching installSkillPackage's default and every existing caller (see
// its doc comment). uninstallDeclaredSkill uses
// uninstallSkillPackageWithFallback directly for CR-01's harness-exclusive
// case.
func uninstallSkillPackage(result *WriteResult, dir string, requester TargetID, exclusiveKeys []string, policy unmanifestedPolicy) {
	uninstallSkillPackageWithFallback(result, dir, requester, exclusiveKeys, policy, []TargetID{Claude})
}

// uninstallSkillPackageWithFallback is uninstallSkillPackage's general
// form, taking the "unreadable manifest" fallback explicit at the call
// site (CR-01) instead of hard-coding Claude unconditionally.
func uninstallSkillPackageWithFallback(result *WriteResult, dir string, requester TargetID, exclusiveKeys []string, policy unmanifestedPolicy, unreadableFallback []TargetID) {
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

	requesters := manifestRequesters(existing, present, rerr, unreadableFallback)
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

// maxSymlinkResolveDepth bounds resolveSkillDir's recursive symlink
// resolution so a self-referential link (or any link cycle) surfaces as
// an error rather than a hang (D-17).
const maxSymlinkResolveDepth = 8

// resolveSkillDir resolves dir to its canonical, symlink-free absolute
// path — even when dir does not exist yet, or exists only as a DANGLING
// symlink whose target has never been created (D-17, RESEARCH Pitfall 2:
// the maintainer's own
// `~/.claude/skills/codegraph -> ../../.agents/skills/codegraph`).
// filepath.EvalSymlinks alone cannot handle either case (it requires
// every path component to exist), so this recurses: if dir itself is a
// live symlink, EvalSymlinks succeeds directly; if dir is a DANGLING
// symlink, its link target is read and resolved in dir's own parent
// directory; if dir simply does not exist and is not a link at all, its
// PARENT is resolved the same way and dir's own base name is joined back
// on. Recursion is bounded to maxSymlinkResolveDepth so a self-referential
// link errors instead of looping forever.
func resolveSkillDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return resolveSkillDirDepth(abs, 0)
}

func resolveSkillDirDepth(abs string, depth int) (string, error) {
	if depth > maxSymlinkResolveDepth {
		return "", fmt.Errorf("agents: resolveSkillDir exceeded max depth (%d) at %s — possible symlink cycle", maxSymlinkResolveDepth, abs)
	}

	if resolved, evalErr := filepath.EvalSymlinks(abs); evalErr == nil {
		return resolved, nil
	} else if info, lerr := os.Lstat(abs); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
		// abs itself is a symlink (EvalSymlinks failed, so it must be
		// dangling) — read its target and resolve that, relative to
		// abs's own parent when the target itself is relative.
		target, rerr := os.Readlink(abs)
		if rerr != nil {
			return "", rerr
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(abs), target)
		}
		return resolveSkillDirDepth(filepath.Clean(target), depth+1)
	} else if os.IsNotExist(evalErr) {
		// abs does not exist and is not itself a link — resolve its
		// parent and join abs's own base name back on.
		parent := filepath.Dir(abs)
		if parent == abs {
			return abs, nil
		}
		resolvedParent, perr := resolveSkillDirDepth(parent, depth+1)
		if perr != nil {
			return "", perr
		}
		return filepath.Join(resolvedParent, filepath.Base(abs)), nil
	} else {
		return "", evalErr
	}
}

// sameSkillDir reports whether a and b are the same physical directory —
// resolving through any chain of symlinks, including a dangling one whose
// target does not exist yet (D-17).
func sameSkillDir(a, b string) (bool, error) {
	ra, err := resolveSkillDir(a)
	if err != nil {
		return false, err
	}
	rb, err := resolveSkillDir(b)
	if err != nil {
		return false, err
	}
	return ra == rb, nil
}
