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

	m := enabledLintersHeaderRE.FindSubmatch(data)
	if m == nil {
		t.Fatalf("%s: no `# enabled-linters: N` header comment found — the parse may have silently matched nothing", golangciConfigPath)
	}
	headerCount, convErr := strconv.Atoi(string(m[1]))
	if convErr != nil {
		t.Fatalf("%s: header comment %q does not parse as an integer: %v", golangciConfigPath, m[0], convErr)
	}

	var cfg golangciConfigYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse %s: %v", golangciConfigPath, err)
	}
	if len(cfg.Linters.Enable) == 0 {
		t.Fatalf("%s: linters.enable parsed zero entries — the parse may have silently matched nothing", golangciConfigPath)
	}

	actual := len(cfg.Linters.Enable) + len(cfg.Formatters.Enable)
	if headerCount != actual {
		t.Errorf(
			"%s: header comment claims %d enabled linters, but linters.enable (%v, %d) + formatters.enable (%v, %d) = %d — the header has drifted from the config it annotates",
			golangciConfigPath, headerCount, cfg.Linters.Enable, len(cfg.Linters.Enable), cfg.Formatters.Enable, len(cfg.Formatters.Enable), actual,
		)
	}
}

// TestGolangciEnabledLintersHeaderDiscriminates is the planted positive
// control (rule 84d1gfpywd): proves the comparison above can actually
// fail, in both directions, rather than having only ever been run
// against a matching pair.
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
			m := enabledLintersHeaderRE.FindSubmatch([]byte(c.src))
			if m == nil {
				t.Fatalf("fixture %q: header regexp did not match", c.name)
			}
			headerCount, err := strconv.Atoi(string(m[1]))
			if err != nil {
				t.Fatalf("fixture %q: header does not parse as int: %v", c.name, err)
			}

			var cfg golangciConfigYAML
			if err := yaml.Unmarshal([]byte(c.src), &cfg); err != nil {
				t.Fatalf("fixture %q: yaml parse: %v", c.name, err)
			}
			actual := len(cfg.Linters.Enable) + len(cfg.Formatters.Enable)

			gotErr := headerCount != actual
			if gotErr != c.wantErr {
				t.Fatalf("fixture %q: headerCount=%d actual=%d mismatch=%v, want mismatch=%v", c.name, headerCount, actual, gotErr, c.wantErr)
			}
			if c.wantErr {
				msg := fmt.Sprintf("header comment claims %d enabled linters, but linters.enable (%d) + formatters.enable (%d) = %d",
					headerCount, len(cfg.Linters.Enable), len(cfg.Formatters.Enable), actual)
				if !regexp.MustCompile(c.errContains).MatchString(msg) {
					t.Fatalf("fixture %q: constructed message %q does not contain expected fragment %q", c.name, msg, c.errContains)
				}
			}
		})
	}
}
