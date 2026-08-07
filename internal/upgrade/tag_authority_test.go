package upgrade

import (
	"encoding/json"
	"os"
	"testing"
)

// D-06R: release-please is the SOLE tag authority for this repository.
//
// The two tests below pin the two config keys that decision depends on.
// Both were prose-only until now — recorded in PROJECT.md, ROADMAP.md, and
// STATE.md, and enforced by nothing. This file is the machine half, in the
// same idiom as TestGoreleaserPinParity and
// TestCheckCrossMatchesGoreleaserTargets: a documented invariant that only
// a human remembers is one refactor away from being silently untrue.

const (
	gsdConfigPath           = "../../.planning/config.json"
	releasePleaseConfigPath = "../../release-please-config.json"
)

// TestGsdTagCreationIsDisabled pins `git.create_tag: false` in GSD's project
// config.
//
// Why a test and not a comment: GSD's own default for this key is TRUE, and
// its gate is FAIL-OPEN. The resolver is
//
//	readConfigJsonValue(cwd, ["git", "create_tag"]) !== false
//
// (gsd-core/bin/lib/init.cjs), whose own comment reads "missing
// `git.create_tag` key means 'create the tag'". So a dropped key, a null, a
// `gsd config` regeneration, or the STRING "false" all silently re-enable
// milestone tagging — none of which look like a change to tagging policy in
// review.
//
// The blast radius is not cosmetic. A `v*` tag matches release.yml's
// `push: tags: "v[0-9]*"` trigger, so an accidental milestone tag fires the
// full release pipeline — building, signing, SBOM'ing and publishing a
// release that release-please never cut. ROADMAP.md and PROJECT.md both
// state this consequence explicitly; this test is what makes them binding.
//
// The type assertion is load-bearing: `"create_tag": "false"` is a non-empty
// string, which is `!== false`, which means GSD tags. A bool check alone
// would pass a value that disables nothing.
func TestGsdTagCreationIsDisabled(t *testing.T) {
	raw, err := os.ReadFile(gsdConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", gsdConfigPath, err)
	}

	var cfg struct {
		Git map[string]any `json:"git"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse %s: %v", gsdConfigPath, err)
	}

	val, present := cfg.Git["create_tag"]
	if !present {
		t.Fatalf("%s: git.create_tag is absent. GSD's gate is fail-open — a missing key means CREATE THE TAG (init.cjs: `readConfigJsonValue(...) !== false`), and a milestone tag matching release.yml's `v[0-9]*` trigger fires the release pipeline for a release release-please never cut (D-06R). Set \"create_tag\": false explicitly.", gsdConfigPath)
	}

	asBool, isBool := val.(bool)
	if !isBool {
		t.Fatalf("%s: git.create_tag is %T (%v), want a JSON boolean. GSD compares with `!== false`, so any non-boolean — including the string \"false\" — leaves tagging ENABLED while looking disabled.", gsdConfigPath, val, val)
	}

	if asBool {
		t.Fatalf("%s: git.create_tag is true, but release-please is the sole tag authority (D-06R). GSD must not create milestone tags: a `v*` tag fires release.yml's `push: tags: \"v[0-9]*\"` trigger and publishes a release nobody cut. If this is a deliberate policy reversal, change D-06R in PROJECT.md first, then this test.", gsdConfigPath)
	}
}

// TestReleasePleaseStaysPreMajor pins `bump-minor-pre-major: true`.
//
// Without it, release-please applies standard semver and a single breaking
// change on a 0.x line jumps straight to 1.0.0. That is exactly what
// happened on 2026-08-07: `feat(platform)!: drop native Windows support`
// produced a 1.0.0 release PR nobody asked for.
//
// The maintainer directive is explicit (PROJECT.md, D-06R, 2026-07-29):
// "We are not going to jump to 1.0 … We'll follow things as release-please
// and conventional commits requires." With this flag, a breaking change
// below 1.0.0 bumps MINOR — the version line keeps following Conventional
// Commits, it just stops treating `!` as a command to leave 0.x.
//
// Note what this does NOT do: it does not discourage marking real breaking
// changes with `!`. The marker stays honest about compatibility; only the
// version arithmetic changes.
func TestReleasePleaseStaysPreMajor(t *testing.T) {
	raw, err := os.ReadFile(releasePleaseConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", releasePleaseConfigPath, err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse %s: %v", releasePleaseConfigPath, err)
	}

	val, present := cfg["bump-minor-pre-major"]
	if !present {
		t.Fatalf("%s: bump-minor-pre-major is absent. Without it a single `!` commit on a 0.x line bumps to 1.0.0 — the exact outcome D-06R rejects (\"We are not going to jump to 1.0\"). Add \"bump-minor-pre-major\": true.", releasePleaseConfigPath)
	}

	asBool, isBool := val.(bool)
	if !isBool {
		t.Fatalf("%s: bump-minor-pre-major is %T (%v), want a JSON boolean true.", releasePleaseConfigPath, val, val)
	}

	if !asBool {
		t.Fatalf("%s: bump-minor-pre-major is false, so a breaking change bumps MAJOR and leaves the 0.x line. D-06R says otherwise. If the project genuinely intends to reach 1.0 now, that is a maintainer decision: record it in PROJECT.md, then update this test deliberately rather than flipping the flag alone.", releasePleaseConfigPath)
	}
}
