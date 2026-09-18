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
//
// RED-phase placeholder (05-02 Task 1): every declaration below compiles
// with a no-op body so skillshared_test.go's failing tests exercise real
// call sites rather than a build error. Real logic lands in this task's
// GREEN commit.
package agents

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

func sharedSkillDirPath(loc Location) (string, error) { return "", nil }

func sharedSkillDirs(loc Location) ([]string, error) { return nil, nil }

func skillManifestPath(dir string) string { return "" }

func skillDirIsForeign(dir string) (bool, error) { return false, nil }

func writeSkillFile(result *WriteResult, dir string, policy unmanifestedPolicy) ([]byte, bool) {
	return nil, false
}

func recordSkillManifest(result *WriteResult, dir string, loc Location, requester TargetID, ownFiles map[string]string) {
}

func installSkillPackage(result *WriteResult, dir string, loc Location, requester TargetID, policy unmanifestedPolicy) {
}

func uninstallSkillPackage(result *WriteResult, dir string, requester TargetID, exclusiveKeys []string, policy unmanifestedPolicy) {
}
