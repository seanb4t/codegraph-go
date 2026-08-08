package upgrade

import (
	"fmt"
	"os"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// goreleaserConfigPath is the on-disk path (relative to this package) to
// .goreleaser.yaml, mirroring releaseWorkflowPath's off-disk-read idiom
// (release_workflow_shape_test.go).
const goreleaserConfigPath = "../../.goreleaser.yaml"

// goreleaserEnvBuildEntry mirrors the subset of one .goreleaser.yaml
// `builds:` list entry this file cares about: its id and its env: list.
// Named distinctly from taskfile_shape_test.go's goreleaserBuildEntry
// (which carries goos/goarch, for TestCheckCrossMatchesGoreleaserTargets) to
// avoid a same-package type collision.
type goreleaserEnvBuildEntry struct {
	ID  string   `yaml:"id"`
	Env []string `yaml:"env"`
}

type goreleaserEnvConfig struct {
	Builds []goreleaserEnvBuildEntry `yaml:"builds"`
}

// parseGoreleaserBuildEnv decodes .goreleaser.yaml source src with a real
// YAML decoder (per this plan's <parser_strategy> — never a hand-written
// line/indentation scanner; see parseGoreleaserCrossPairs in
// taskfile_shape_test.go for the established precedent) and returns the
// env: list of the builds: entry whose id equals buildID, as a map of
// VAR=value pairs split on the first '='. Returns a non-nil error — never a
// usable empty map — when src fails to parse, when builds: is empty, when
// no entry's id matches buildID, or when the matched entry has no env: list.
func parseGoreleaserBuildEnv(src, buildID string) (map[string]string, error) {
	var cfg goreleaserEnvConfig
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserBuildEnv: %w", err)
	}
	if len(cfg.Builds) == 0 {
		return nil, fmt.Errorf("parseGoreleaserBuildEnv: no builds: entries found")
	}
	for _, b := range cfg.Builds {
		if b.ID != buildID {
			continue
		}
		if len(b.Env) == 0 {
			return nil, fmt.Errorf("parseGoreleaserBuildEnv: build id %q has no env: list", buildID)
		}
		out := make(map[string]string, len(b.Env))
		for _, e := range b.Env {
			k, v, ok := strings.Cut(e, "=")
			if !ok {
				return nil, fmt.Errorf("parseGoreleaserBuildEnv: build id %q env entry %q has no '='", buildID, e)
			}
			out[k] = v
		}
		return out, nil
	}
	return nil, fmt.Errorf("parseGoreleaserBuildEnv: no builds: entry with id %q found", buildID)
}

func mustGoreleaserBuildEnv(t *testing.T, src, buildID string) map[string]string {
	t.Helper()
	v, err := parseGoreleaserBuildEnv(src, buildID)
	if err != nil {
		t.Fatalf("mustGoreleaserBuildEnv: %v", err)
	}
	return v
}

// TestLinuxBuildIdsCrossCompileViaZig is the REL-05 enabling-change guard:
// both linux build ids must carry a zig cc/zig c++ CC/CXX override naming
// their glibc target triple, and neither darwin build id may carry one —
// darwin must never cross-link via zig (the libresolv/DNS-resolver risk
// documented in .goreleaser.yaml's darwin comment and release.yml's header).
// Enumerates all four build ids explicitly so deleting a build entry cannot
// make this vacuously pass.
func TestLinuxBuildIdsCrossCompileViaZig(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	linuxAmd64Env := mustGoreleaserBuildEnv(t, src, "codegraph-linux-amd64")
	if cc, ok := linuxAmd64Env["CC"]; !ok || !strings.HasPrefix(cc, "zig cc") || !strings.Contains(cc, "x86_64-linux-gnu") {
		t.Errorf("codegraph-linux-amd64: CC = %q, want a zig cc override targeting x86_64-linux-gnu", cc)
	}
	if cxx, ok := linuxAmd64Env["CXX"]; !ok || !strings.HasPrefix(cxx, "zig c++") || !strings.Contains(cxx, "x86_64-linux-gnu") {
		t.Errorf("codegraph-linux-amd64: CXX = %q, want a zig c++ override targeting x86_64-linux-gnu", cxx)
	}

	linuxArm64Env := mustGoreleaserBuildEnv(t, src, "codegraph-linux-arm64")
	if cc, ok := linuxArm64Env["CC"]; !ok || !strings.HasPrefix(cc, "zig cc") || !strings.Contains(cc, "aarch64-linux-gnu") {
		t.Errorf("codegraph-linux-arm64: CC = %q, want a zig cc override targeting aarch64-linux-gnu", cc)
	}
	if cxx, ok := linuxArm64Env["CXX"]; !ok || !strings.HasPrefix(cxx, "zig c++") || !strings.Contains(cxx, "aarch64-linux-gnu") {
		t.Errorf("codegraph-linux-arm64: CXX = %q, want a zig c++ override targeting aarch64-linux-gnu", cxx)
	}

	for _, id := range []string{"codegraph-darwin-amd64", "codegraph-darwin-arm64"} {
		env, err := parseGoreleaserBuildEnv(src, id)
		if err != nil {
			// Darwin entries carry an env: list today (CGO_ENABLED=1), so
			// this should not fire. Fail loud rather than skip, so a future
			// change to the darwin entries' env: shape cannot silently
			// vacuously pass the CC/CXX-absence assertion below.
			t.Fatalf("%s: %v", id, err)
		}
		if cc, ok := env["CC"]; ok {
			t.Errorf("%s: has a CC=%q override, want none — darwin must never cross-link via zig (libresolv/DNS-resolver risk)", id, cc)
		}
		if cxx, ok := env["CXX"]; ok {
			t.Errorf("%s: has a CXX=%q override, want none — darwin must never cross-link via zig (libresolv/DNS-resolver risk)", id, cxx)
		}
	}
}

// TestParseGoreleaserBuildEnv_MissingBuildIDIsError is the non-vacuity
// companion: parseGoreleaserBuildEnv must return a non-nil error naming the
// missing build id, never a usable empty map, when asked for an id that
// does not exist in .goreleaser.yaml.
func TestParseGoreleaserBuildEnv_MissingBuildIDIsError(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	_, err = parseGoreleaserBuildEnv(src, "does-not-exist")
	if err == nil {
		t.Fatalf("parseGoreleaserBuildEnv(src, %q) = nil error, want non-nil naming the missing build id", "does-not-exist")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("parseGoreleaserBuildEnv error = %q, want it to name the missing build id %q", err.Error(), "does-not-exist")
	}
}
