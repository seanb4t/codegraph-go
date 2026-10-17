package upgrade

import (
	"encoding/json"
	"os"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// D-06R: the changie release workflow's App-token step (release-tag.yml job
// `tag`, which runs `task release:tag`) is the SOLE tag authority for this
// repository. GRD-16's TestTagCreationHasSingleAppTokenSite
// (changie_release_shape_test.go) holds that structurally; the two tests
// below pin the two config keys the decision depends on.
// Both were prose-only until now — recorded in PROJECT.md, ROADMAP.md, and
// STATE.md, and enforced by nothing. This file is the machine half, in the
// same idiom as TestGoreleaserPinParity and
// TestCheckCrossMatchesGoreleaserTargets: a documented invariant that only
// a human remembers is one refactor away from being silently untrue.

const gsdConfigPath = "../../.planning/config.json"

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
// release no release PR ever cut. ROADMAP.md and PROJECT.md both
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
		t.Fatalf("%s: git.create_tag is absent. GSD's gate is fail-open — a missing key means CREATE THE TAG (init.cjs: `readConfigJsonValue(...) !== false`), and a milestone tag matching release.yml's `v[0-9]*` trigger fires the release pipeline for a release no release PR cut (D-06R). Set \"create_tag\": false explicitly.", gsdConfigPath)
	}

	asBool, isBool := val.(bool)
	if !isBool {
		t.Fatalf("%s: git.create_tag is %T (%v), want a JSON boolean. GSD compares with `!== false`, so any non-boolean — including the string \"false\" — leaves tagging ENABLED while looking disabled.", gsdConfigPath, val, val)
	}

	if asBool {
		t.Fatalf("%s: git.create_tag is true, but the changie release workflow's App-token step is the sole tag authority (D-06R). GSD must not create milestone tags: a `v*` tag fires release.yml's `push: tags: \"v[0-9]*\"` trigger and publishes a release nobody cut. If this is a deliberate policy reversal, change D-06R in PROJECT.md first, then this test.", gsdConfigPath)
	}
}

// TestChangieBreakingBumpsMinorPreOne pins `auto: minor` on .changie.yaml's
// Breaking kind.
//
// changie derives the next version from the fragment kinds, so this one key is
// what keeps a single breaking change on a 0.x line from jumping straight to
// 1.0.0. That is exactly what happened on 2026-08-07, under the previous
// release automation: `feat(platform)!: drop native Windows support` produced
// a 1.0.0 release PR nobody asked for.
//
// The maintainer directive is explicit (PROJECT.md, D-06R, 2026-07-29):
// "We are not going to jump to 1.0". While the version is below 1.0.0, a
// Breaking fragment bumps MINOR; the version line keeps following the kinds,
// it just stops treating a breaking change as a command to leave 0.x.
//
// The kind is matched by label, not by position, and must occur exactly once:
// a duplicate would leave it unclear which entry wins, and a missing one would
// make the guard pass on nothing.
//
// Note what this does NOT do: it does not discourage recording real breaking
// changes. The kind stays honest about compatibility; only the version
// arithmetic changes.
func TestChangieBreakingBumpsMinorPreOne(t *testing.T) {
	raw, err := os.ReadFile(changieConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", changieConfigPath, err)
	}

	root := mustChangieConfig(t, raw)
	kinds, ok := yamlMappingValue(root, "kinds")
	if !ok {
		t.Fatalf("%s: no kinds key", changieConfigPath)
	}

	var breaking []*yaml.Node
	for _, kind := range kinds.Content {
		if kind.Kind != yaml.MappingNode {
			continue
		}
		if label, found := yamlMappingValue(kind, "label"); found && label.Value == "Breaking" {
			breaking = append(breaking, kind)
		}
	}
	if len(breaking) != 1 {
		t.Fatalf("%s: found %d kinds labelled Breaking, want exactly 1. changie derives the version from this kind, so a missing or duplicated entry leaves the pre-1.0 rule (D-06R: \"We are not going to jump to 1.0\") unpinned.", changieConfigPath, len(breaking))
	}

	auto, found := yamlMappingValue(breaking[0], "auto")
	if !found {
		t.Fatalf("%s: the Breaking kind has no auto key. Without it changie does not bump the version for a breaking change at all; D-06R wants `auto: minor` until 1.0.", changieConfigPath)
	}
	if auto.Value != "minor" {
		t.Fatalf("%s: the Breaking kind has auto: %q, want \"minor\". Any other value breaks D-06R (\"We are not going to jump to 1.0\"): major leaves the 0.x line on one breaking fragment. If the project genuinely intends to reach 1.0 now, that is a maintainer decision: record it in PROJECT.md first, then update this test deliberately rather than flipping the key alone.", changieConfigPath, auto.Value)
	}

	t.Logf("scanned %d kinds; Breaking auto = %s", len(kinds.Content), auto.Value)
}
