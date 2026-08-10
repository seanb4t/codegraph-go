package upgrade

import (
	"fmt"
	"os"
	"regexp"
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

// goreleaserSign mirrors one .goreleaser.yaml `signs:` list entry (D-18,
// LOCKED, maintainer 2026-08-09: renamed from the pre-ruling build-scoped
// `binary_signs:` block — see TestSignsSidecarMatchesUpgradeContract's doc
// comment for the full rationale). ids: is the NEW D-18 field: signs[].ids
// filters ARCHIVE ids (this project's raw/zip pair), a DIFFERENT namespace
// from notarize.macos[].ids' BUILD ids — confusing the two is how the zip
// archive shape could silently acquire its own cosign signature.
type goreleaserSign struct {
	Cmd       string   `yaml:"cmd"`
	IDs       []string `yaml:"ids"`
	Args      []string `yaml:"args"`
	Signature string   `yaml:"signature"`
	Artifacts string   `yaml:"artifacts"`
}

type goreleaserSignsConfig struct {
	Signs []goreleaserSign `yaml:"signs"`
}

// parseGoreleaserSigns decodes .goreleaser.yaml source src with a real YAML
// decoder and returns every signs: entry. Returns a non-nil error — never a
// usable empty slice — when src fails to parse, or when signs: is absent or
// contains no entries.
func parseGoreleaserSigns(src string) ([]goreleaserSign, error) {
	var cfg goreleaserSignsConfig
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserSigns: %w", err)
	}
	if len(cfg.Signs) == 0 {
		return nil, fmt.Errorf("parseGoreleaserSigns: no signs: entries found")
	}
	return cfg.Signs, nil
}

func mustGoreleaserSigns(t *testing.T, src string) []goreleaserSign {
	t.Helper()
	v, err := parseGoreleaserSigns(src)
	if err != nil {
		t.Fatalf("mustGoreleaserSigns: %v", err)
	}
	return v
}

// goreleaserNotarizeMacos mirrors one .goreleaser.yaml `notarize.macos` list
// entry. sign/notarize are decoded as map[string]any (not typed structs) so
// TestNotarizeMacosOmitsEntitlements can assert on KEY PRESENCE — an unset
// entitlements: key is distinguishable from one present with an empty
// value, which a typed string field's zero value could not tell apart.
type goreleaserNotarizeMacos struct {
	Enabled  string         `yaml:"enabled"`
	IDs      []string       `yaml:"ids"`
	Sign     map[string]any `yaml:"sign"`
	Notarize map[string]any `yaml:"notarize"`
}

type goreleaserNotarizeConfig struct {
	Notarize struct {
		Macos []goreleaserNotarizeMacos `yaml:"macos"`
	} `yaml:"notarize"`
}

// parseGoreleaserNotarizeMacos decodes .goreleaser.yaml source src with a
// real YAML decoder and returns the notarize.macos: list. Returns a
// non-nil error — never a usable value — when src fails to parse, when the
// list is absent or empty, OR when it carries more than one entry: a
// second entry changes notary.MacOS's own Run from a sequential per-binary
// loop inside one entry to a parallelized loop across entries (a semaphore
// error group), which would invalidate every timeout/sequencing statement
// this phase's config comments make (see TestNotarizeMacosHasExactlyOneEntry)
// — so "read the first and ignore the rest" is never an acceptable outcome
// here.
func parseGoreleaserNotarizeMacos(src string) ([]goreleaserNotarizeMacos, error) {
	var cfg goreleaserNotarizeConfig
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserNotarizeMacos: %w", err)
	}
	switch len(cfg.Notarize.Macos) {
	case 0:
		return nil, fmt.Errorf("parseGoreleaserNotarizeMacos: no notarize.macos: entries found")
	case 1:
		return cfg.Notarize.Macos, nil
	default:
		return nil, fmt.Errorf("parseGoreleaserNotarizeMacos: notarize.macos: has %d entries, want exactly 1 (a second entry changes the pipe's parallelism)", len(cfg.Notarize.Macos))
	}
}

func mustGoreleaserNotarizeMacos(t *testing.T, src string) []goreleaserNotarizeMacos {
	t.Helper()
	v, err := parseGoreleaserNotarizeMacos(src)
	if err != nil {
		t.Fatalf("mustGoreleaserNotarizeMacos: %v", err)
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

// resolveGoreleaserFieldTemplateWithFuncs is resolveGoreleaserFieldTemplate
// plus an injectable template.FuncMap, so a predicate function (e.g. an
// isEnvSet stand-in for GoReleaser's own env-presence template function)
// can be controlled deterministically from a test rather than read from the
// real process environment. resolveGoreleaserFieldTemplate's own signature
// is left unchanged — every other test in this file depends on it.
func resolveGoreleaserFieldTemplateWithFuncs(tplSrc string, fields map[string]any, funcs template.FuncMap) (string, error) {
	tpl, err := template.New("goreleaser-field-template-funcs").Funcs(funcs).Option("missingkey=error").Parse(tplSrc)
	if err != nil {
		return "", fmt.Errorf("resolveGoreleaserFieldTemplateWithFuncs: parse %q: %w", tplSrc, err)
	}
	var buf strings.Builder
	if err := tpl.Execute(&buf, fields); err != nil {
		return "", fmt.Errorf("resolveGoreleaserFieldTemplateWithFuncs: execute %q: %w", tplSrc, err)
	}
	return buf.String(), nil
}

// isEnvSetFuncMap returns a template.FuncMap providing an "isEnvSet"
// function matching GoReleaser's own env-presence template predicate
// (goreleaser.com/customization/templates), backed by the given set of
// variable names considered "present" rather than the real process
// environment — so the resolved enabled: verdict can be tested under
// controlled environments without mutating the actual process environment.
// This is a LOCALLY CONSTRUCTED emulation of the predicate, not GoReleaser's
// own runtime template evaluation — see TestNotarizeMacosEnabledIsEnvGated's
// doc comment for what this does and does not prove.
func isEnvSetFuncMap(present map[string]bool) template.FuncMap {
	return template.FuncMap{
		"isEnvSet": func(name string) bool {
			return present[name]
		},
	}
}

// notarizeCredentialVars is the five credential env vars
// notarize.macos[0].enabled must gate on CONJUNCTIVELY (D-18's
// five-term-conjunction ruling in .goreleaser.yaml's notarize: comment) —
// pinned independently here so a future edit that drops one variable from
// the template is caught by this list, not discovered by re-reading the
// template.
var notarizeCredentialVars = []string{
	"MACOS_SIGN_P12",
	"MACOS_SIGN_PASSWORD",
	"MACOS_NOTARY_ISSUER_ID",
	"MACOS_NOTARY_KEY_ID",
	"MACOS_NOTARY_KEY",
}

// TestSignsSidecarMatchesUpgradeContract holds D-18 (LOCKED, maintainer,
// 2026-08-09): cosign now signs via the RELEASE-SCOPED signs: pipe, not the
// build-scoped binary_signs: pipe this test was originally named and
// pinned against (renamed here — the prior name described the
// build-scoped block and is retired along with it). The test moved
// deliberately, per this repo's own "a test pinning a broken invariant
// resists correction worse than no test" rule — it keeps asserting the
// SAME property (a distinct, download-contract-matching
// sidecar name per platform), not a new one.
//
// Why the block moved: GoReleaser's pipe execution order is a hardcoded Go
// slice (internal/pipeline/pipeline.go), not driven by .goreleaser.yaml's
// block order. sign.BinaryPipe{} (binary_signs:) and notary.MacOS{}
// (notarize:) are BOTH registered inside BuildPipeline, with
// sign.BinaryPipe{} first — so the pre-D-18 config cosign-signed bytes that
// notary.MacOS{} (a sign-AND-notarize pipe: quill.Sign rewrites the Mach-O's
// LC_CODE_SIGNATURE load command in place) then mutated afterward.
// sign.Pipe{} (signs:), by contrast, is registered well AFTER
// BuildPipeline entirely — and therefore after notarize — so moving cosign
// here makes the signed subject and the published subject describe the
// same, post-notarization bytes by construction. GoReleaser is not wrong
// and this is not an upstream bug: the pre-D-18 config was simply asking
// GoReleaser for the wrong thing (D-18, RESEARCH.md).
//
// The Path-vs-Name hazard D-14 (Phase 1) already found is UNCHANGED by this
// move: signone() (internal/pipe/sign/sign.go) still rebinds
// env["artifact"] = art.Name (not art.Path) for the publish-naming template
// pass, so the explicit Go-template signature: field below remains the
// mitigation and must never revert to a bare ${artifact} form —
// GoReleaser's own documented "${artifact}.sigstore.json" idiom collapses
// all 4 platforms to one name (D-14's original collision failure mode).
//
// ids: [raw] (the NEW D-18 field on this block) is the filter that
// actually keeps this pipe off the zip archive shape (D-15/D-04):
// signs[].ids filters ARCHIVE ids (this project's raw/zip pair), a
// DIFFERENT namespace from notarize.macos[].ids' BUILD ids — confusing the
// two is how the zip would silently acquire its own cosign signature. This
// test asserts ids as an exact set ([raw], nothing else), never
// containment, so an edit that adds "zip" alongside "raw" turns this red.
//
// This test RESOLVES the shipped signature: template for all 4 release
// platforms and asserts the 4 results are DISTINCT strings each matching
// internal/upgrade's download contract — it never string-matches the
// template's literal source.
func TestSignsSidecarMatchesUpgradeContract(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	signs := mustGoreleaserSigns(t, src)
	if len(signs) != 1 {
		t.Fatalf("signs: has %d entries, want exactly 1", len(signs))
	}
	s := signs[0]

	if s.Cmd != "cosign" {
		t.Errorf("signs[0].cmd = %q, want %q", s.Cmd, "cosign")
	}
	if s.Artifacts != "binary" {
		t.Errorf("signs[0].artifacts = %q, want %q", s.Artifacts, "binary")
	}
	wantSignIDs := sortedJoin([]string{"raw"})
	if got := sortedJoin(s.IDs); got != wantSignIDs {
		t.Errorf("signs[0].ids = %v, want exactly the set [raw] (D-15/D-04: this is the filter that keeps cosign off the zip archive shape; signs[].ids is an ARCHIVE-id filter, a different namespace from notarize.macos[].ids' BUILD-id filter)", s.IDs)
	}
	wantArgs := []string{"sign-blob", "--bundle=${signature}", "${artifact}", "--yes"}
	if len(s.Args) != len(wantArgs) {
		t.Fatalf("signs[0].args = %v, want %v", s.Args, wantArgs)
	}
	for i, want := range wantArgs {
		if s.Args[i] != want {
			t.Errorf("signs[0].args[%d] = %q, want %q", i, s.Args[i], want)
		}
	}

	seen := map[string]bool{}
	hostMatched := false
	for _, p := range releasePairs {
		got, err := resolveGoreleaserFieldTemplate(s.Signature, map[string]any{
			"ProjectName": "codegraph",
			"Tag":         pinnedReleaseTag,
			"Os":          p.goos,
			"Arch":        p.goarch,
		})
		if err != nil {
			t.Fatalf("resolve signs[0].signature for %s/%s: %v", p.goos, p.goarch, err)
		}
		want := goReleaserAssetName(pinnedReleaseTag, p.goos, p.goarch) + ".sigstore.json"
		if got != want {
			t.Errorf("signs[0].signature resolved for %s/%s = %q, want %q", p.goos, p.goarch, got, want)
		}
		if seen[got] {
			t.Errorf("signs[0].signature resolved to a NON-DISTINCT name %q for %s/%s — this is D-14's collision failure mode", got, p.goos, p.goarch)
		}
		seen[got] = true

		if p.goos == runtime.GOOS && p.goarch == runtime.GOARCH {
			hostMatched = true
			wantHost := releaseAssetName(pinnedReleaseTag) + ".sigstore.json"
			if got != wantHost {
				t.Errorf("signs[0].signature resolved for host %s/%s = %q, want %q (must agree with releaseAssetName)", p.goos, p.goarch, got, wantHost)
			}
		}
	}
	if !hostMatched {
		t.Fatalf("host os/arch (%s/%s) is not one of the 4 pinned release pairs", runtime.GOOS, runtime.GOARCH)
	}
}

// TestParseGoreleaserSigns_NoSignsBlockIsError is the non-vacuity
// companion: parseGoreleaserSigns("") must return a non-nil error, never an
// empty-but-usable slice.
func TestParseGoreleaserSigns_NoSignsBlockIsError(t *testing.T) {
	if _, err := parseGoreleaserSigns(""); err == nil {
		t.Fatalf("parseGoreleaserSigns(\"\") = nil error, want non-nil")
	}
}

// TestParseGoreleaserNotarize_NoNotarizeBlockIsError is the non-vacuity
// companion: parseGoreleaserNotarizeMacos("") must return a non-nil error,
// never an empty-but-usable slice.
func TestParseGoreleaserNotarize_NoNotarizeBlockIsError(t *testing.T) {
	if _, err := parseGoreleaserNotarizeMacos(""); err == nil {
		t.Fatalf("parseGoreleaserNotarizeMacos(\"\") = nil error, want non-nil")
	}
}

// wantDarwinBuildIDs is the two darwin build ids notarize.macos[0].ids must
// name exactly (T-02-05) — pinned independently of wantBuildIDs (which
// covers all 4 platforms), so this test cannot accidentally pass by
// asserting notarize.macos[0].ids against the same 4-element slice it is
// meant to exclude two elements from.
var wantDarwinBuildIDs = []string{
	"codegraph-darwin-amd64",
	"codegraph-darwin-arm64",
}

// TestNotarizeMacosIdsCoverDarwinBuildIDs holds T-02-05: notarize.macos[0].ids
// is exactly the set of this project's two darwin build ids — not a subset
// (the default is [ProjectName] = ["codegraph"], which matches neither and
// silently no-ops the pipe, exit 0, un-notarized binaries, no error
// anywhere in the log), and not a superset (a linux build id, or an
// archive id like "zip" smuggled into what is a BUILD-id filter). Both
// directions matter: per D-15 the raw Mach-O is the only notarized
// subject, and per D-04 notarizing the zip too was rejected as a redundant
// Apple round-trip.
func TestNotarizeMacosIdsCoverDarwinBuildIDs(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	entries := mustGoreleaserNotarizeMacos(t, src)
	entry := entries[0]

	want := sortedJoin(wantDarwinBuildIDs)
	if got := sortedJoin(entry.IDs); got != want {
		t.Errorf("notarize.macos[0].ids = %v, want exactly the set %v (matching too little silently skips notarization; matching too much notarizes a shape it should not, e.g. a linux build id or the zip archive id)", entry.IDs, wantDarwinBuildIDs)
	}
}

// TestNotarizeMacosHasExactlyOneEntry holds the plan's parallelism
// invariant: notarize.macos: must carry EXACTLY one entry. A second entry
// does not merely go untested — it changes notary.MacOS's own Run from a
// sequential per-binary loop inside one entry to a parallelized loop
// ACROSS entries (a semaphore error group), which would invalidate every
// timeout/sequencing statement this phase's config comments make. Proven
// both on the real config (must parse to exactly 1 entry) and by mutation
// (a synthetic 2-entry fixture must be REJECTED by the parser, not
// silently truncated to its first element).
func TestNotarizeMacosHasExactlyOneEntry(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	entries := mustGoreleaserNotarizeMacos(t, src)
	if len(entries) != 1 {
		t.Fatalf("notarize.macos: has %d entries, want exactly 1", len(entries))
	}

	// Mutation observation: a synthetic fixture naming a SECOND macos:
	// entry must be REJECTED by the parser (see
	// parseGoreleaserNotarizeMacos's "absent, empty, OR longer than one
	// entry" contract), never silently read as its first element.
	const twoEntryFixture = `
notarize:
  macos:
    - enabled: "true"
      ids: [codegraph-darwin-amd64]
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
    - enabled: "true"
      ids: [codegraph-darwin-arm64]
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
`
	if _, err := parseGoreleaserNotarizeMacos(twoEntryFixture); err == nil {
		t.Fatalf("parseGoreleaserNotarizeMacos(<2-entry fixture>) = nil error, want non-nil — a second macos: entry changes the pipe's parallelism and must be rejected, not silently read as its first element")
	}
}

// TestNotarizeMacosOmitsEntitlements holds D-03: notarize.macos[0].sign
// declares no entitlements key at all — quill embeds nothing when the key
// is unset (D-03's working hypothesis) while still applying the
// hardened-runtime flag unconditionally regardless. sign: is decoded as
// map[string]any specifically so this assertion is made on KEY PRESENCE,
// never on a zero value that an empty-string entitlements: key would also
// produce.
func TestNotarizeMacosOmitsEntitlements(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	entries := mustGoreleaserNotarizeMacos(t, src)
	entry := entries[0]

	if entry.Sign == nil {
		t.Fatalf("notarize.macos[0].sign is absent, want a mapping (certificate/password)")
	}
	if _, ok := entry.Sign["entitlements"]; ok {
		t.Errorf("notarize.macos[0].sign declares an entitlements: key, want none (D-03: a future addition must be a deliberate decision, not drift)")
	}
}

// TestNotarizeMacosEnabledIsEnvGated holds T-02-06: notarize.macos[0].enabled
// must resolve to the true literal ONLY when all five credential variables
// (notarizeCredentialVars) are present, and to the false literal when ANY
// subset — including zero — is missing. Proven by resolving the template
// under the all-present environment, the all-absent environment, and each
// single-credential-missing environment in turn (7 resolutions total: 1
// true, 6 false), never by reading the template source.
//
// This is a STATIC BACKSTOP over Go's text/template with a locally
// constructed isEnvSet FuncMap emulating GoReleaser's own env-presence
// template predicate — it proves the TEMPLATE's own logic (the conjunction
// is correctly five-term and correctly conjunctive), not GoReleaser's
// runtime template evaluation, which reads the real process environment
// through its own internal/tmpl package. Runtime behaviour — whether the
// pipe actually fires or skips under a real credential set in a real
// `goreleaser build`/`release` invocation — is established only by plan
// 02-04's rehearsal observing the pipe fire and not fire, never by this
// test alone.
func TestNotarizeMacosEnabledIsEnvGated(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	entries := mustGoreleaserNotarizeMacos(t, src)
	entry := entries[0]
	if strings.TrimSpace(entry.Enabled) == "" {
		t.Fatalf("notarize.macos[0].enabled is empty, want a non-empty template")
	}

	resolve := func(t *testing.T, present map[string]bool) string {
		t.Helper()
		got, err := resolveGoreleaserFieldTemplateWithFuncs(entry.Enabled, nil, isEnvSetFuncMap(present))
		if err != nil {
			t.Fatalf("resolve notarize.macos[0].enabled: %v", err)
		}
		return got
	}

	allPresent := map[string]bool{}
	for _, v := range notarizeCredentialVars {
		allPresent[v] = true
	}
	if got := resolve(t, allPresent); got != "true" {
		t.Errorf("notarize.macos[0].enabled resolved with all 5 credentials present = %q, want %q", got, "true")
	}

	if got := resolve(t, map[string]bool{}); got != "false" {
		t.Errorf("notarize.macos[0].enabled resolved with 0 credentials present = %q, want %q", got, "false")
	}

	for _, missing := range notarizeCredentialVars {
		present := map[string]bool{}
		for _, v := range notarizeCredentialVars {
			present[v] = v != missing
		}
		if got := resolve(t, present); got != "false" {
			t.Errorf("notarize.macos[0].enabled resolved with only %q missing = %q, want %q (a partial credential set must not enable the pipe)", missing, got, "false")
		}
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

// ---------------------------------------------------------------------
// Task 3 (plan 03-02): homebrew_casks: — property tests holding the
// block's non-obvious fields, each demonstrated RED against a real,
// confirmed-applied, byte-cleanly-reverted mutation of .goreleaser.yaml
// (see 03-02-SUMMARY.md for the mutation/message/revert record).
// ---------------------------------------------------------------------

// goreleaserHomebrewCasksTopLevel mirrors the top-level homebrew_casks:
// SEQUENCE. Decoded as []map[string]any (not a typed struct) deliberately:
// a typed struct silently drops an unrecognized key on decode, which would
// make TestHomebrewCaskHasNoURLKey's "absent vs present" distinction
// impossible to hold — a typed field's zero value looks identical whether
// the YAML key was omitted or present-but-empty.
type goreleaserHomebrewCasksTopLevel struct {
	Casks []map[string]any `yaml:"homebrew_casks"`
}

// parseGoreleaserHomebrewCasks decodes .goreleaser.yaml source src with a
// real YAML decoder and returns every homebrew_casks: entry as a raw map.
// Returns a non-nil error — never a usable empty slice — when src fails to
// parse, or when homebrew_casks: is absent or contains no entries.
func parseGoreleaserHomebrewCasks(src string) ([]map[string]any, error) {
	var cfg goreleaserHomebrewCasksTopLevel
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserHomebrewCasks: %w", err)
	}
	if len(cfg.Casks) == 0 {
		return nil, fmt.Errorf("parseGoreleaserHomebrewCasks: no homebrew_casks: entries found")
	}
	return cfg.Casks, nil
}

func mustGoreleaserHomebrewCask(t *testing.T, src string) map[string]any {
	t.Helper()
	v, err := parseGoreleaserHomebrewCasks(src)
	if err != nil {
		t.Fatalf("mustGoreleaserHomebrewCask: %v", err)
	}
	if len(v) != 1 {
		t.Fatalf("homebrew_casks: has %d entries, want exactly 1", len(v))
	}
	return v[0]
}

// TestParseGoreleaserCask_NoBlockIsError is the non-vacuity companion:
// parseGoreleaserHomebrewCasks("") must return a non-nil error, never an
// empty-but-usable slice.
func TestParseGoreleaserCask_NoBlockIsError(t *testing.T) {
	if _, err := parseGoreleaserHomebrewCasks(""); err == nil {
		t.Fatalf(`parseGoreleaserHomebrewCasks("") = nil error, want non-nil`)
	}
}

// TestHomebrewCaskIDsIsExactlyZipArchiveSet holds BREW-04/RESEARCH.md
// Pattern 1: homebrew_casks[0].ids is exactly the single-element set
// naming the zip archive entry. An unscoped or wrongly-scoped filter makes
// both archive shapes (raw, zip) match the same platform pairs and
// cask.Pipe{}'s dataFor() throws its own ErrMultipleArchivesSameOS ("one
// tap can handle only one archive of an OS/Arch combination").
func TestHomebrewCaskIDsIsExactlyZipArchiveSet(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	cask := mustGoreleaserHomebrewCask(t, string(data))

	rawIDs, ok := cask["ids"]
	if !ok {
		t.Fatalf("homebrew_casks[0] has no ids: key")
	}
	ids, err := toStringSlice(rawIDs)
	if err != nil {
		t.Fatalf("homebrew_casks[0].ids: %v", err)
	}
	want := sortedJoin([]string{"zip"})
	if got := sortedJoin(ids); got != want {
		t.Errorf("homebrew_casks[0].ids = %v, want exactly [zip] — naming the raw archive entry too (or instead) reintroduces cask.Pipe{}'s ErrMultipleArchivesSameOS", ids)
	}
}

// TestHomebrewCaskHasNoURLKey holds RESEARCH.md Pattern 2: homebrew_casks[0]
// declares no url: key at all — the unset default is GoReleaser's own
// Name-derived ReleaseURLTemplate, and a hand-written url: template is the
// Path-vs-Name collision class this file's signs:/sboms: blocks each
// independently document having hit before. Decoded as a raw map
// specifically so this test can distinguish "key absent" from "key present
// with an empty value" — a typed struct's zero value cannot tell those
// apart.
func TestHomebrewCaskHasNoURLKey(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	cask := mustGoreleaserHomebrewCask(t, string(data))

	if v, ok := cask["url"]; ok {
		t.Errorf("homebrew_casks[0] declares a url: key (value=%v) — want it entirely absent; a hand-written url.template reintroduces the Path-vs-Name collision class this file's signs:/sboms: comments document", v)
	}
}

// TestHomebrewCaskGeneratedCompletionsShellsIsExactSet holds BREW-03/D-06:
// generate_completions_from_executable.shells is exactly the three-element
// set {bash, zsh, fish} — narrower than Homebrew's own SUPPORTED_SHELLS
// default, which also includes pwsh, an unrequested shell whose generation
// failure would produce a warning nobody asked for.
func TestHomebrewCaskGeneratedCompletionsShellsIsExactSet(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	cask := mustGoreleaserHomebrewCask(t, string(data))

	raw, ok := cask["generate_completions_from_executable"]
	if !ok {
		t.Fatalf("homebrew_casks[0] has no generate_completions_from_executable: key")
	}
	block, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("homebrew_casks[0].generate_completions_from_executable is %T, not a mapping", raw)
	}
	rawShells, ok := block["shells"]
	if !ok {
		t.Fatalf("generate_completions_from_executable has no shells: key")
	}
	shells, err := toStringSlice(rawShells)
	if err != nil {
		t.Fatalf("generate_completions_from_executable.shells: %v", err)
	}
	want := sortedJoin([]string{"bash", "zsh", "fish"})
	if got := sortedJoin(shells); got != want {
		t.Errorf("generate_completions_from_executable.shells = %v, want exactly [bash zsh fish] — a superset (e.g. adding pwsh) reintroduces Homebrew's own unrequested default shell", shells)
	}
	if got, ok := block["shell_parameter_format"].(string); !ok || got != "cobra" {
		t.Errorf("generate_completions_from_executable.shell_parameter_format = %v, want %q", block["shell_parameter_format"], "cobra")
	}
}

// raiseKeywordPattern matches Ruby's `raise` keyword as a whole word, used
// only by countNonCommentRaiseStatements below.
var raiseKeywordPattern = regexp.MustCompile(`\braise\b`)

// countNonCommentRaiseStatements counts lines in a Ruby hook body
// containing the raise keyword, EXCLUDING comment lines (lines whose
// trimmed content starts with "#") — so this file's own dense inline
// commentary inside .goreleaser.yaml can never change the count. Used to
// assert a STRUCTURAL property (how many distinct failure paths exist)
// rather than pinning any raise message's literal text.
func countNonCommentRaiseStatements(body string) int {
	count := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if raiseKeywordPattern.MatchString(line) {
			count++
		}
	}
	return count
}

// TestHomebrewCaskHooksHaveStructuralProperties holds BREW-05/D-11: both
// hooks.post.install and hooks.post.uninstall are present and non-empty,
// and the install body contains at least two distinct raise-style failure
// paths — D-11's two positive assertions ("didn't run" / "ran but is the
// wrong artifact"). Asserts PROPERTIES only, never pins a raise message's
// literal text: a mutation that reworks a raise message's wording without
// removing the raise must stay green here, while a mutation that deletes a
// raise must turn this red. The raise count is derived with comment lines
// excluded, so prose in this hook's own dense inline commentary can never
// change the count.
func TestHomebrewCaskHooksHaveStructuralProperties(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	cask := mustGoreleaserHomebrewCask(t, string(data))

	rawHooks, ok := cask["hooks"]
	if !ok {
		t.Fatalf("homebrew_casks[0] has no hooks: key")
	}
	hooks, ok := rawHooks.(map[string]any)
	if !ok {
		t.Fatalf("homebrew_casks[0].hooks is %T, not a mapping", rawHooks)
	}
	rawPost, ok := hooks["post"]
	if !ok {
		t.Fatalf("homebrew_casks[0].hooks has no post: key")
	}
	post, ok := rawPost.(map[string]any)
	if !ok {
		t.Fatalf("homebrew_casks[0].hooks.post is %T, not a mapping", rawPost)
	}

	install, ok := post["install"].(string)
	if !ok || strings.TrimSpace(install) == "" {
		t.Fatalf("homebrew_casks[0].hooks.post.install is absent or empty")
	}
	uninstall, ok := post["uninstall"].(string)
	if !ok || strings.TrimSpace(uninstall) == "" {
		t.Fatalf("homebrew_casks[0].hooks.post.uninstall is absent or empty")
	}

	if n := countNonCommentRaiseStatements(install); n < 2 {
		t.Errorf("hooks.post.install contains %d non-comment raise statement(s), want at least 2 (D-11's two positive assertions)", n)
	}
}

// goreleaserArchivesRawTopLevel mirrors the top-level archives: sequence as
// raw maps — used only by TestHomebrewCaskArchivesHaveNoFilesKey, which
// needs to detect the PRESENCE of a files: key; the typed goreleaserArchive
// struct earlier in this file has no Files field and would silently drop
// it on decode, making that presence check impossible.
type goreleaserArchivesRawTopLevel struct {
	Archives []map[string]any `yaml:"archives"`
}

// parseGoreleaserArchivesRaw decodes .goreleaser.yaml source src with a
// real YAML decoder and returns every archives: entry as a raw map.
// Returns a non-nil error — never a usable empty slice — when src fails to
// parse, or when archives: is absent or contains no entries.
func parseGoreleaserArchivesRaw(src string) ([]map[string]any, error) {
	var cfg goreleaserArchivesRawTopLevel
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return nil, fmt.Errorf("parseGoreleaserArchivesRaw: %w", err)
	}
	if len(cfg.Archives) == 0 {
		return nil, fmt.Errorf("parseGoreleaserArchivesRaw: no archives: entries found")
	}
	return cfg.Archives, nil
}

func mustGoreleaserArchivesRaw(t *testing.T, src string) []map[string]any {
	t.Helper()
	v, err := parseGoreleaserArchivesRaw(src)
	if err != nil {
		t.Fatalf("mustGoreleaserArchivesRaw: %v", err)
	}
	return v
}

// TestParseGoreleaserArchivesRaw_NoArchivesBlockIsError is the non-vacuity
// companion: parseGoreleaserArchivesRaw("") must return a non-nil error,
// never an empty-but-usable slice.
func TestParseGoreleaserArchivesRaw_NoArchivesBlockIsError(t *testing.T) {
	if _, err := parseGoreleaserArchivesRaw(""); err == nil {
		t.Fatalf(`parseGoreleaserArchivesRaw("") = nil error, want non-nil`)
	}
}

// TestHomebrewCaskArchivesHaveNoFilesKey holds D-16: no archives: entry
// declares a files: key — a future attempt to ship completions or man
// pages inside the zip (or raw) archive must turn this test red instead
// of quietly creating a second, staler source alongside the install-time
// generation this phase relies on.
func TestHomebrewCaskArchivesHaveNoFilesKey(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	archives := mustGoreleaserArchivesRaw(t, string(data))
	for _, a := range archives {
		id, _ := a["id"].(string)
		if _, ok := a["files"]; ok {
			t.Errorf("archives[id=%s] declares a files: key — completions/man pages must be generated at install time (BREW-03/BREW-04), never shipped inside the archive", id)
		}
	}
}
