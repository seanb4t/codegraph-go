// golangci_shape_test.go binds .golangci.yml's own `# enabled-linters: N`
// header comment to the config it annotates (IN-15).
//
// That header is described in the file's own comment as "machine-read by
// 03-10-PLAN.md Task 2's <verify> block" — a one-shot, plan-time check,
// not a repository test. Nothing in internal/upgrade/*_test.go parsed it.
// Enabling a sixth linter (or a formatter) without updating the comment
// produced no failure anywhere, so the header could drift from the config
// it annotates with no guard noticing — mirroring
// TestUIProtoFieldNumbersAreStableAndUnique's pinned-length pattern
// (internal/upgrade already has that convention for a different frozen
// count), applied here to a comment instead of a proto field list.
package upgrade

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// golangciConfigPath is .golangci.yml's on-disk path relative to this
// package, mirroring workflowsDir/rootGoModPath's own "../.." convention
// (this package's directory is internal/upgrade/, two hops from the repo
// root).
const golangciConfigPath = "../../.golangci.yml"

var enabledLintersHeaderRE = regexp.MustCompile(`(?m)^#\s*enabled-linters:\s*(\d+)\s*$`)

// golangciConfigYAML is the subset of .golangci.yml's v2 schema this
// guard needs: the two enable lists the header comment's count is a sum
// over (linters.enable + formatters.enable) — deliberately NOT
// linters.exclusions or issues:, which shape HOW the enabled set reports,
// not WHICH linters are enabled, exactly as the header comment itself
// states.
type golangciConfigYAML struct {
	Linters struct {
		Enable []string `yaml:"enable"`
	} `yaml:"linters"`
	Formatters struct {
		Enable []string `yaml:"enable"`
	} `yaml:"formatters"`
}

// compareEnabledLintersHeader is the ONE comparison both
// TestGolangciEnabledLintersHeaderMatchesConfig (the real guard, driven
// against .golangci.yml on disk) and
// TestGolangciEnabledLintersHeaderDiscriminates (the planted positive
// control, driven against synthetic fixtures) call — WR-05: an earlier
// version of the discriminator re-implemented this exact sequence (regex
// match, strconv.Atoi, yaml.Unmarshal, the count comparison, even the
// failure message) against local fixtures instead of calling the real
// guard's own code, so it proved a RE-IMPLEMENTATION could fail, not that
// the guard could. A single shared function makes that vacuity
// impossible: both tests exercise the SAME code path, so a change to the
// real guard's logic (e.g. dropping `+ len(cfg.Formatters.Enable)`) fails
// BOTH tests, not just the one running against the real file.
//
// data is the raw .golangci.yml bytes (or a synthetic fixture in the same
// shape). Returns the header's claimed count, the actually-enabled count,
// the parsed config (so a caller building a failure message never
// re-parses the same bytes), and a non-nil err only for a structurally
// malformed input (no header comment found, header not an integer, or
// YAML that fails to parse) — headerCount != actual is NOT itself an
// error return; callers compare the two counts themselves, exactly as
// TestGolangciEnabledLintersHeaderMatchesConfig already did before this
// extraction.
func compareEnabledLintersHeader(data []byte) (headerCount, actual int, cfg golangciConfigYAML, err error) {
	m := enabledLintersHeaderRE.FindSubmatch(data)
	if m == nil {
		return 0, 0, cfg, fmt.Errorf("no `# enabled-linters: N` header comment found — the parse may have silently matched nothing")
	}
	headerCount, convErr := strconv.Atoi(string(m[1]))
	if convErr != nil {
		return 0, 0, cfg, fmt.Errorf("header comment %q does not parse as an integer: %w", m[0], convErr)
	}

	if yamlErr := yaml.Unmarshal(data, &cfg); yamlErr != nil {
		return 0, 0, cfg, fmt.Errorf("parse config: %w", yamlErr)
	}

	actual = len(cfg.Linters.Enable) + len(cfg.Formatters.Enable)
	return headerCount, actual, cfg, nil
}

// enabledLintersHeaderMismatch renders compareEnabledLintersHeader's two
// counts into the same failure message both tests below assert against —
// extracted alongside the comparison itself so the message text lives in
// exactly one place too.
func enabledLintersHeaderMismatch(headerCount, actual int, cfg golangciConfigYAML) string {
	return fmt.Sprintf(
		"header comment claims %d enabled linters, but linters.enable (%v, %d) + formatters.enable (%v, %d) = %d — the header has drifted from the config it annotates",
		headerCount, cfg.Linters.Enable, len(cfg.Linters.Enable), cfg.Formatters.Enable, len(cfg.Formatters.Enable), actual,
	)
}

// TestGolangciEnabledLintersHeaderMatchesConfig parses BOTH the `#
// enabled-linters: N` header comment and the real linters.enable +
// formatters.enable lists from the same file, and asserts N equals their
// combined length — so enabling a sixth linter without updating the
// comment (or vice versa: updating the comment without actually changing
// the enabled set) fails here rather than drifting silently forever.
func TestGolangciEnabledLintersHeaderMatchesConfig(t *testing.T) {
	data, err := os.ReadFile(golangciConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", golangciConfigPath, err)
	}

	headerCount, actual, cfg, cmpErr := compareEnabledLintersHeader(data)
	if cmpErr != nil {
		t.Fatalf("%s: %v", golangciConfigPath, cmpErr)
	}
	if len(cfg.Linters.Enable) == 0 {
		t.Fatalf("%s: linters.enable parsed zero entries — the parse may have silently matched nothing", golangciConfigPath)
	}

	if headerCount != actual {
		t.Errorf("%s: %s", golangciConfigPath, enabledLintersHeaderMismatch(headerCount, actual, cfg))
	}
}

// TestGolangciEnabledLintersHeaderDiscriminates is the planted positive
// control (rule 84d1gfpywd): proves compareEnabledLintersHeader — the
// SAME function the real guard above calls — can actually fail, in both
// directions, rather than having only ever been run against a matching
// pair. WR-05: this must drive compareEnabledLintersHeader itself, not a
// second, parallel implementation of the same steps — see that
// function's doc comment for why.
func TestGolangciEnabledLintersHeaderDiscriminates(t *testing.T) {
	cases := []struct {
		name        string
		src         string
		wantErr     bool
		errContains string
	}{
		{
			name: "header matches config",
			src: "# enabled-linters: 2\n" +
				"linters:\n  enable:\n    - errcheck\nformatters:\n  enable:\n    - gofmt\n",
			wantErr: false,
		},
		{
			name: "header UNDER-counts a real linter",
			src: "# enabled-linters: 1\n" +
				"linters:\n  enable:\n    - errcheck\nformatters:\n  enable:\n    - gofmt\n",
			wantErr:     true,
			errContains: "claims 1",
		},
		{
			name: "header OVER-counts (a linter was removed without updating N)",
			src: "# enabled-linters: 3\n" +
				"linters:\n  enable:\n    - errcheck\nformatters:\n  enable:\n    - gofmt\n",
			wantErr:     true,
			errContains: "claims 3",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			headerCount, actual, cfg, err := compareEnabledLintersHeader([]byte(c.src))
			if err != nil {
				t.Fatalf("fixture %q: compareEnabledLintersHeader: %v", c.name, err)
			}

			gotErr := headerCount != actual
			if gotErr != c.wantErr {
				t.Fatalf("fixture %q: headerCount=%d actual=%d mismatch=%v, want mismatch=%v", c.name, headerCount, actual, gotErr, c.wantErr)
			}
			if c.wantErr {
				msg := enabledLintersHeaderMismatch(headerCount, actual, cfg)
				if !regexp.MustCompile(c.errContains).MatchString(msg) {
					t.Fatalf("fixture %q: constructed message %q does not contain expected fragment %q", c.name, msg, c.errContains)
				}
			}
		})
	}
}
