package upgrade

import (
	"fmt"
	"os"
	"sort"
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

// ---------------------------------------------------------------------
// Task 1: archives:/checksum: — dual archive shapes keyed by id (REL-09,
// D-12, D-15, D-16).
// ---------------------------------------------------------------------

// goreleaserArchive mirrors the subset of one .goreleaser.yaml `archives:`
// list entry this file cares about.
type goreleaserArchive struct {
	ID           string   `yaml:"id"`
	Formats      []string `yaml:"formats"`
	IDs          []string `yaml:"ids"`
	NameTemplate string   `yaml:"name_template"`
}

type goreleaserArchivesConfig struct {
	Archives []goreleaserArchive `yaml:"archives"`
}

// parseGoreleaserArchives decodes .goreleaser.yaml source src with a real
// YAML decoder (yaml.Unmarshal into typed structs — never a hand-written
// line/indentation/folded-scalar scanner, per this plan's <parser_strategy>)
// and returns every archives: entry. Returns a non-nil error — never a
// usable empty slice — when src fails to parse, or when archives: is absent
// or contains no entries.
func parseGoreleaserArchives(src string) ([]goreleaserArchive, error) {
	var cfg goreleaserArchivesConfig
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserArchives: %w", err)
	}
	if len(cfg.Archives) == 0 {
		return nil, fmt.Errorf("parseGoreleaserArchives: no archives: entries found")
	}
	return cfg.Archives, nil
}

func mustGoreleaserArchives(t *testing.T, src string) []goreleaserArchive {
	t.Helper()
	v, err := parseGoreleaserArchives(src)
	if err != nil {
		t.Fatalf("mustGoreleaserArchives: %v", err)
	}
	return v
}

// findArchiveByID returns the archives: entry with the given id, or nil if
// none matches.
func findArchiveByID(archives []goreleaserArchive, id string) *goreleaserArchive {
	for i := range archives {
		if archives[i].ID == id {
			return &archives[i]
		}
	}
	return nil
}

// sortedJoin returns ss as a single comma-joined, sorted string, per this
// project's own house rule (~/.claude/rules/grepping.md) against
// order-dependent set comparisons: two lists naming the same set of
// elements in different orders must compare equal.
func sortedJoin(ss []string) string {
	sorted := make([]string, len(ss))
	copy(sorted, ss)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

// wantBuildIDs is the 4 build ids every archives: entry (and checksum
// scope) in this project's release matrix must name — pinned independently
// of .goreleaser.yaml's builds: list, so deleting a build entry there
// cannot silently make an archives: assertion vacuous.
var wantBuildIDs = []string{
	"codegraph-linux-amd64",
	"codegraph-linux-arm64",
	"codegraph-darwin-amd64",
	"codegraph-darwin-arm64",
}

// parseGoreleaserTopLevelBlock decodes .goreleaser.yaml source src with a
// real YAML decoder and returns the named top-level MAPPING block (e.g.
// "checksum", "release") as a generic map. Returns a non-nil error — never
// a usable nil map — when src fails to parse, when no top-level key named
// key is present, or when that key's value is not itself a mapping (use
// the typed list parsers in Task 2 for sequence-valued top-level keys like
// binary_signs:/sboms:).
func parseGoreleaserTopLevelBlock(src, key string) (map[string]any, error) {
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		return nil, fmt.Errorf("parseGoreleaserTopLevelBlock: %w", err)
	}
	raw, ok := doc[key]
	if !ok {
		return nil, fmt.Errorf("parseGoreleaserTopLevelBlock: no top-level %q key found", key)
	}
	block, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parseGoreleaserTopLevelBlock: top-level %q is not a mapping (got %T)", key, raw)
	}
	return block, nil
}

func mustGoreleaserTopLevelBlock(t *testing.T, src, key string) map[string]any {
	t.Helper()
	v, err := parseGoreleaserTopLevelBlock(src, key)
	if err != nil {
		t.Fatalf("mustGoreleaserTopLevelBlock: %v", err)
	}
	return v
}

// toStringSlice converts a YAML-decoded []any (yaml.Unmarshal's shape for a
// sequence nested inside a map[string]any) into a []string. Returns a
// non-nil error — never a usable empty slice — if v is not a []any, or if
// any element is not a string.
func toStringSlice(v any) ([]string, error) {
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("toStringSlice: value is %T, not a sequence", v)
	}
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		s, ok := e.(string)
		if !ok {
			return nil, fmt.Errorf("toStringSlice: element %v is %T, not a string", e, e)
		}
		out = append(out, s)
	}
	return out, nil
}

// TestRawArchiveEntryStaysBinaryFormat holds REL-09's success criterion 4 as
// a machine check: the archives: entry with id: raw declares formats:
// containing exactly the single value "binary". Fails if the entry is
// absent, if formats: names anything other than binary, or if it names more
// than one format.
func TestRawArchiveEntryStaysBinaryFormat(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	archives := mustGoreleaserArchives(t, src)
	raw := findArchiveByID(archives, "raw")
	if raw == nil {
		t.Fatalf("no archives: entry with id: raw found")
	}
	if len(raw.Formats) != 1 || raw.Formats[0] != "binary" {
		t.Fatalf("archives[id=raw].formats = %v, want exactly [\"binary\"]", raw.Formats)
	}
}

// TestZipArchiveSharesRawAssetStem holds D-15: the archives: entries with
// id: raw and id: zip declare byte-identical name_template values, the zip
// entry declares formats: containing exactly zip, and BOTH entries' ids:
// lists name all four build ids — so a platform cannot silently lose an
// archive shape.
func TestZipArchiveSharesRawAssetStem(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	archives := mustGoreleaserArchives(t, src)
	raw := findArchiveByID(archives, "raw")
	if raw == nil {
		t.Fatalf("no archives: entry with id: raw found")
	}
	zip := findArchiveByID(archives, "zip")
	if zip == nil {
		t.Fatalf("no archives: entry with id: zip found")
	}

	if raw.NameTemplate != zip.NameTemplate {
		t.Errorf("archives[id=raw].name_template = %q, archives[id=zip].name_template = %q, want byte-identical", raw.NameTemplate, zip.NameTemplate)
	}
	if len(zip.Formats) != 1 || zip.Formats[0] != "zip" {
		t.Errorf("archives[id=zip].formats = %v, want exactly [\"zip\"]", zip.Formats)
	}

	wantSet := sortedJoin(wantBuildIDs)
	if got := sortedJoin(raw.IDs); got != wantSet {
		t.Errorf("archives[id=raw].ids = %v, want the set %v", raw.IDs, wantBuildIDs)
	}
	if got := sortedJoin(zip.IDs); got != wantSet {
		t.Errorf("archives[id=zip].ids = %v, want the set %v", zip.IDs, wantBuildIDs)
	}
}

// TestChecksumCoversRawAndZipIdsOnly holds D-12: checksum.ids is exactly
// the set {raw, zip} — not a superset, not empty, not absent.
func TestChecksumCoversRawAndZipIdsOnly(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	block := mustGoreleaserTopLevelBlock(t, src, "checksum")
	rawIDs, ok := block["ids"]
	if !ok {
		t.Fatalf("checksum: block has no ids: key")
	}
	ids, err := toStringSlice(rawIDs)
	if err != nil {
		t.Fatalf("checksum.ids: %v", err)
	}
	want := sortedJoin([]string{"raw", "zip"})
	if got := sortedJoin(ids); got != want {
		t.Errorf("checksum.ids = %v, want exactly [raw zip]", ids)
	}
}

// TestParseGoreleaserArchives_NoArchivesBlockIsError is the non-vacuity
// companion: parseGoreleaserArchives("") must return a non-nil error, never
// an empty-but-usable slice.
func TestParseGoreleaserArchives_NoArchivesBlockIsError(t *testing.T) {
	if _, err := parseGoreleaserArchives(""); err == nil {
		t.Fatalf("parseGoreleaserArchives(\"\") = nil error, want non-nil")
	}
}
