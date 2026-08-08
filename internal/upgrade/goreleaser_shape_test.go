package upgrade

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
	"text/template"

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

// ---------------------------------------------------------------------
// Task 2: binary_signs:/sboms: — declarative sidecar names matching
// today's hand-rolled loops (D-14, D-17).
// ---------------------------------------------------------------------

// releasePairs is the pinned 4-platform release matrix, mirroring
// TestReleaseAssetNameMatchesGoReleaser's independent-literal discipline
// (verify_release_e2e_test.go). Declared once here and reused by every
// Task 2/3 test in this file that needs to resolve a per-platform template.
var releasePairs = []struct{ goos, goarch string }{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
}

// pinnedReleaseTag is the same kind of independently-pinned literal
// TestReleaseAssetNameMatchesGoReleaser uses — never derived from
// goReleaserAssetName or releaseAssetName, so a bug shared by both cannot
// hide from a resolve-and-compare assertion.
const pinnedReleaseTag = "v1.2.3"

// goreleaserBinarySign mirrors one .goreleaser.yaml `binary_signs:` list
// entry.
type goreleaserBinarySign struct {
	Cmd       string   `yaml:"cmd"`
	Args      []string `yaml:"args"`
	Signature string   `yaml:"signature"`
	Artifacts string   `yaml:"artifacts"`
}

type goreleaserBinarySignsConfig struct {
	BinarySigns []goreleaserBinarySign `yaml:"binary_signs"`
}

// parseGoreleaserBinarySigns decodes .goreleaser.yaml source src with a
// real YAML decoder and returns every binary_signs: entry. Returns a
// non-nil error — never a usable empty slice — when src fails to parse, or
// when binary_signs: is absent or contains no entries.
func parseGoreleaserBinarySigns(src string) ([]goreleaserBinarySign, error) {
	var cfg goreleaserBinarySignsConfig
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserBinarySigns: %w", err)
	}
	if len(cfg.BinarySigns) == 0 {
		return nil, fmt.Errorf("parseGoreleaserBinarySigns: no binary_signs: entries found")
	}
	return cfg.BinarySigns, nil
}

func mustGoreleaserBinarySigns(t *testing.T, src string) []goreleaserBinarySign {
	t.Helper()
	v, err := parseGoreleaserBinarySigns(src)
	if err != nil {
		t.Fatalf("mustGoreleaserBinarySigns: %v", err)
	}
	return v
}

// goreleaserSBOM mirrors one .goreleaser.yaml `sboms:` list entry.
type goreleaserSBOM struct {
	ID        string   `yaml:"id"`
	Cmd       string   `yaml:"cmd"`
	Args      []string `yaml:"args"`
	Documents []string `yaml:"documents"`
	Artifacts string   `yaml:"artifacts"`
}

type goreleaserSBOMsConfig struct {
	SBOMs []goreleaserSBOM `yaml:"sboms"`
}

// parseGoreleaserSBOMs decodes .goreleaser.yaml source src with a real YAML
// decoder and returns every sboms: entry. Returns a non-nil error — never a
// usable empty slice — when src fails to parse, or when sboms: is absent or
// contains no entries.
func parseGoreleaserSBOMs(src string) ([]goreleaserSBOM, error) {
	var cfg goreleaserSBOMsConfig
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserSBOMs: %w", err)
	}
	if len(cfg.SBOMs) == 0 {
		return nil, fmt.Errorf("parseGoreleaserSBOMs: no sboms: entries found")
	}
	return cfg.SBOMs, nil
}

func mustGoreleaserSBOMs(t *testing.T, src string) []goreleaserSBOM {
	t.Helper()
	v, err := parseGoreleaserSBOMs(src)
	if err != nil {
		t.Fatalf("mustGoreleaserSBOMs: %v", err)
	}
	return v
}

// resolveGoreleaserFieldTemplate executes a .goreleaser.yaml Go-template
// string (the {{ .Field }} form GoReleaser resolves via its internal/tmpl
// package's text/template engine, once any ${VAR} env substitution has
// already happened) against the given fields. Used to prove what a
// template ACTUALLY resolves to for a given artifact, rather than
// string-matching its literal source — the "assert the property, not the
// literal" discipline cycle-3 review required for sboms.documents (HIGH-B)
// and this plan's own binary_signs.signature finding (see the config
// comment above binary_signs: in .goreleaser.yaml).
func resolveGoreleaserFieldTemplate(tplSrc string, fields map[string]any) (string, error) {
	tpl, err := template.New("goreleaser-field-template").Option("missingkey=error").Parse(tplSrc)
	if err != nil {
		return "", fmt.Errorf("resolveGoreleaserFieldTemplate: parse %q: %w", tplSrc, err)
	}
	var buf strings.Builder
	if err := tpl.Execute(&buf, fields); err != nil {
		return "", fmt.Errorf("resolveGoreleaserFieldTemplate: execute %q: %w", tplSrc, err)
	}
	return buf.String(), nil
}

// TestBinarySignsSidecarMatchesUpgradeContract holds D-14: the
// binary_signs: block's signature: template must resolve, for every one of
// the 4 shipped platforms, to exactly the literal string internal/upgrade
// downloads (assetName + ".sigstore.json"). This is the one contract whose
// breakage bricks every user's codegraph upgrade after the binary has
// already been downloaded.
//
// binary_signs: signs the RAW build-output artifact (type Binary), not the
// renamed archives: release asset — confirmed against the pinned
// goreleaser/v2@v2.17.1 module's internal/pipe/sign/sign_binary.go (filters
// artifact.ByType(artifact.Binary)) and internal/pipe/sign/sign.go's
// signone() (the PUBLISHED signature artifact's Name is computed from
// env["artifact"] = art.Name, not art.Path). A raw Binary artifact's .Name
// is this project's literal `binary: codegraph` value for every platform,
// so a $artifact-based (env-var) template collides to one name; the
// template must instead use Go-template FIELDS (.ProjectName/.Tag/.Os/
// .Arch) bound from that same raw artifact's correct per-platform
// Goos/Goarch. This test RESOLVES the shipped template — it does not
// string-match its literal source — so a reversion to the colliding
// $artifact form is caught by a failing resolved-name comparison, not by a
// brittle text diff.
func TestBinarySignsSidecarMatchesUpgradeContract(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	signs := mustGoreleaserBinarySigns(t, src)
	if len(signs) != 1 {
		t.Fatalf("binary_signs: has %d entries, want exactly 1", len(signs))
	}
	bs := signs[0]

	if bs.Cmd != "cosign" {
		t.Errorf("binary_signs[0].cmd = %q, want %q", bs.Cmd, "cosign")
	}
	if bs.Artifacts != "binary" {
		t.Errorf("binary_signs[0].artifacts = %q, want %q", bs.Artifacts, "binary")
	}
	wantArgs := []string{"sign-blob", "--bundle=${signature}", "${artifact}", "--yes"}
	if len(bs.Args) != len(wantArgs) {
		t.Fatalf("binary_signs[0].args = %v, want %v", bs.Args, wantArgs)
	}
	for i, want := range wantArgs {
		if bs.Args[i] != want {
			t.Errorf("binary_signs[0].args[%d] = %q, want %q", i, bs.Args[i], want)
		}
	}

	seen := map[string]bool{}
	hostMatched := false
	for _, p := range releasePairs {
		got, err := resolveGoreleaserFieldTemplate(bs.Signature, map[string]any{
			"ProjectName": "codegraph",
			"Tag":         pinnedReleaseTag,
			"Os":          p.goos,
			"Arch":        p.goarch,
		})
		if err != nil {
			t.Fatalf("resolve binary_signs[0].signature for %s/%s: %v", p.goos, p.goarch, err)
		}
		want := goReleaserAssetName(pinnedReleaseTag, p.goos, p.goarch) + ".sigstore.json"
		if got != want {
			t.Errorf("binary_signs[0].signature resolved for %s/%s = %q, want %q", p.goos, p.goarch, got, want)
		}
		if seen[got] {
			t.Errorf("binary_signs[0].signature resolved to a NON-DISTINCT name %q for %s/%s — this is D-14's collision failure mode", got, p.goos, p.goarch)
		}
		seen[got] = true

		if p.goos == runtime.GOOS && p.goarch == runtime.GOARCH {
			hostMatched = true
			wantHost := releaseAssetName(pinnedReleaseTag) + ".sigstore.json"
			if got != wantHost {
				t.Errorf("binary_signs[0].signature resolved for host %s/%s = %q, want %q (must agree with releaseAssetName)", p.goos, p.goarch, got, wantHost)
			}
		}
	}
	if !hostMatched {
		t.Fatalf("host os/arch (%s/%s) is not one of the 4 pinned release pairs", runtime.GOOS, runtime.GOARCH)
	}
}

// TestSbomsArePerBinaryWithSpdxNames holds D-17: the sboms: block declares
// artifacts: binary (explicitly present, not defaulted — the pipe's own
// default is archive, which would silently catalog the zips and break
// DIST-03's per-binary .spdx.json contract), cmd: syft, and a documents:
// list of exactly one NAME-derived template. It RESOLVES that template for
// all four platforms and asserts the four results are four DISTINCT
// strings, each equal to goReleaserAssetName(tag, goos, goarch) +
// ".spdx.json" — the property cycle-3 review's HIGH-B finding cares about,
// not the literal template text.
func TestSbomsArePerBinaryWithSpdxNames(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	sboms := mustGoreleaserSBOMs(t, src)
	if len(sboms) != 1 {
		t.Fatalf("sboms: has %d entries, want exactly 1", len(sboms))
	}
	sb := sboms[0]

	if sb.ID != "binary-sbom" {
		t.Errorf("sboms[0].id = %q, want %q", sb.ID, "binary-sbom")
	}
	if sb.Artifacts != "binary" {
		t.Errorf("sboms[0].artifacts = %q, want %q (absent defaults to \"archive\", which would break DIST-03)", sb.Artifacts, "binary")
	}
	if sb.Cmd != "syft" {
		t.Errorf("sboms[0].cmd = %q, want %q", sb.Cmd, "syft")
	}
	wantArgs := []string{"$artifact", "--output", "spdx-json=$document"}
	if len(sb.Args) != len(wantArgs) {
		t.Fatalf("sboms[0].args = %v, want %v", sb.Args, wantArgs)
	}
	for i, want := range wantArgs {
		if sb.Args[i] != want {
			t.Errorf("sboms[0].args[%d] = %q, want %q", i, sb.Args[i], want)
		}
	}
	if len(sb.Documents) != 1 {
		t.Fatalf("sboms[0].documents has %d elements, want exactly 1", len(sb.Documents))
	}

	seen := map[string]bool{}
	for _, p := range releasePairs {
		artifactName := goReleaserAssetName(pinnedReleaseTag, p.goos, p.goarch)
		got, err := resolveGoreleaserFieldTemplate(sb.Documents[0], map[string]any{
			"ArtifactName": artifactName,
		})
		if err != nil {
			t.Fatalf("resolve sboms[0].documents[0] for %s/%s: %v", p.goos, p.goarch, err)
		}
		want := artifactName + ".spdx.json"
		if got != want {
			t.Errorf("sboms[0].documents[0] resolved for %s/%s = %q, want %q", p.goos, p.goarch, got, want)
		}
		if seen[got] {
			t.Errorf("sboms[0].documents[0] resolved to a NON-DISTINCT name %q for %s/%s — this is HIGH-B's collision failure mode", got, p.goos, p.goarch)
		}
		seen[got] = true
	}
}

// ---------------------------------------------------------------------
// Task 3: release: — rerun idempotency and prerelease handling are config,
// not GoReleaser defaults (review HIGH-2).
// ---------------------------------------------------------------------

// TestReleaseBlockIsRerunIdempotent holds T-01-44: .goreleaser.yaml
// declares a top-level release: block, and within it
// replace_existing_artifacts is the boolean true. Fails if the release:
// block is absent entirely, if the key is absent (GoReleaser v2.17.1
// defaults it to false), or if it is present and false.
func TestReleaseBlockIsRerunIdempotent(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	block := mustGoreleaserTopLevelBlock(t, src, "release")
	v, ok := block["replace_existing_artifacts"]
	if !ok {
		t.Fatalf("release: block has no replace_existing_artifacts key (GoReleaser v2.17.1 defaults this to false)")
	}
	b, ok := v.(bool)
	if !ok || !b {
		t.Errorf("release.replace_existing_artifacts = %v (%T), want true", v, v)
	}
}

// TestReleaseBlockDoesNotRewriteReleaseBody holds T-01-30 / D-06R: within
// the release: block, prerelease is exactly the string "auto", and neither
// a name_template nor a header/footer/draft/disable key is declared —
// release-please authors the Release name and body, and GoReleaser must
// only add assets to it.
func TestReleaseBlockDoesNotRewriteReleaseBody(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	block := mustGoreleaserTopLevelBlock(t, src, "release")
	v, ok := block["prerelease"]
	if !ok {
		t.Fatalf("release: block has no prerelease key")
	}
	s, ok := v.(string)
	if !ok || s != "auto" {
		t.Errorf("release.prerelease = %v (%T), want %q", v, v, "auto")
	}

	for _, forbidden := range []string{"name_template", "header", "footer", "draft", "disable"} {
		if _, ok := block[forbidden]; ok {
			t.Errorf("release: block declares forbidden key %q — release-please owns Release authorship (D-06R)", forbidden)
		}
	}
}
