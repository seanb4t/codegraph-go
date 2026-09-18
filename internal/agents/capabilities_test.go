package agents

import (
	"path/filepath"
	"testing"
)

// TestCapabilitiesDeclared pins, per registered target, an explicit
// expected row (D-01, D-02) — an independent oracle, not read back from
// the code. Also asserts MCPConfig resolves without error at every
// declared scope, and Antigravity's migration-aware MCPConfig resolution
// under a fresh fake home vs. an unmigrated legacy-file-present home.
func TestCapabilitiesDeclared(t *testing.T) {
	home := fakeHome(t)
	project := t.TempDir()
	t.Chdir(project)

	cases := []struct {
		id     TargetID
		scopes []Location
		format ConfigFormat
		hooks  HookMechanism
		// instructions maps Location -> expected path ("" = none).
		instructions map[Location]string
		// skillDir maps Location -> expected written skill dir ("" = none).
		skillDir map[Location]string
	}{
		{
			id:     Antigravity,
			scopes: []Location{LocationGlobal},
			format: ConfigFormatJSON,
			hooks:  HooksNone,
			instructions: map[Location]string{
				LocationGlobal: "",
			},
			skillDir: map[Location]string{
				LocationGlobal: "",
			},
		},
		{
			id:     Claude,
			scopes: []Location{LocationGlobal, LocationLocal},
			format: ConfigFormatJSON,
			hooks:  HooksClaudeJSON,
			instructions: map[Location]string{
				LocationGlobal: filepath.Join(home, ".claude", "CLAUDE.md"),
				LocationLocal:  filepath.Join(".claude", "CLAUDE.md"),
			},
			skillDir: map[Location]string{
				LocationGlobal: filepath.Join(home, ".claude", "skills", "codegraph"),
				LocationLocal:  filepath.Join(".claude", "skills", "codegraph"),
			},
		},
		{
			id:     Codex,
			scopes: []Location{LocationGlobal},
			format: ConfigFormatTOML,
			hooks:  HooksNone,
			instructions: map[Location]string{
				LocationGlobal: filepath.Join(home, ".codex", "AGENTS.md"),
			},
			skillDir: map[Location]string{
				LocationGlobal: "",
			},
		},
		{
			id:     Cursor,
			scopes: []Location{LocationGlobal, LocationLocal},
			format: ConfigFormatJSON,
			hooks:  HooksNone,
			instructions: map[Location]string{
				LocationGlobal: "",
				LocationLocal:  "",
			},
			skillDir: map[Location]string{
				LocationGlobal: "",
				LocationLocal:  "",
			},
		},
		{
			id:     Gemini,
			scopes: []Location{LocationGlobal, LocationLocal},
			format: ConfigFormatJSON,
			hooks:  HooksNone,
			instructions: map[Location]string{
				LocationGlobal: filepath.Join(home, ".gemini", "GEMINI.md"),
				LocationLocal:  "GEMINI.md",
			},
			skillDir: map[Location]string{
				LocationGlobal: "",
				LocationLocal:  "",
			},
		},
		{
			id:     Hermes,
			scopes: []Location{LocationGlobal},
			format: ConfigFormatYAML,
			hooks:  HooksNone,
			instructions: map[Location]string{
				LocationGlobal: "",
			},
			skillDir: map[Location]string{
				LocationGlobal: "",
			},
		},
		{
			id:     Kiro,
			scopes: []Location{LocationGlobal, LocationLocal},
			format: ConfigFormatJSON,
			hooks:  HooksNone,
			instructions: map[Location]string{
				LocationGlobal: "",
				LocationLocal:  "",
			},
			skillDir: map[Location]string{
				LocationGlobal: "",
				LocationLocal:  "",
			},
		},
		{
			id:     Opencode,
			scopes: []Location{LocationGlobal, LocationLocal},
			format: ConfigFormatJSONC,
			hooks:  HooksNone,
			instructions: map[Location]string{
				LocationGlobal: filepath.Join(home, ".config", "opencode", "AGENTS.md"),
				LocationLocal:  "AGENTS.md",
			},
			skillDir: map[Location]string{
				LocationGlobal: "",
				LocationLocal:  "",
			},
		},
	}

	if len(cases) != 8 {
		t.Fatalf("test table has %d entries, want 8 (one per registered target)", len(cases))
	}

	for _, tc := range cases {
		t.Run(string(tc.id), func(t *testing.T) {
			target, ok := GetTarget(tc.id)
			if !ok {
				t.Fatalf("GetTarget(%s): not registered", tc.id)
			}
			caps := target.Capabilities()

			if len(caps.Scopes) != len(tc.scopes) {
				t.Fatalf("Scopes = %v, want %v", caps.Scopes, tc.scopes)
			}
			for _, loc := range tc.scopes {
				if !caps.Supports(loc) {
					t.Errorf("Supports(%s) = false, want true (Scopes=%v)", loc, caps.Scopes)
				}
			}

			if caps.ConfigFormat != tc.format {
				t.Errorf("ConfigFormat = %q, want %q", caps.ConfigFormat, tc.format)
			}
			if caps.Hooks != tc.hooks {
				t.Errorf("Hooks = %q, want %q", caps.Hooks, tc.hooks)
			}

			for _, loc := range tc.scopes {
				if caps.MCPConfig == nil {
					t.Fatalf("MCPConfig is nil for a supported location %s", loc)
				}
				if _, err := caps.MCPConfig(loc); err != nil {
					t.Errorf("MCPConfig(%s) = error %v, want no error", loc, err)
				}

				gotInstr, err := caps.InstructionsPath(loc)
				if err != nil {
					t.Errorf("InstructionsPath(%s) = error %v, want no error", loc, err)
				}
				if wantInstr, ok := tc.instructions[loc]; ok && gotInstr != wantInstr {
					t.Errorf("InstructionsPath(%s) = %q, want %q", loc, gotInstr, wantInstr)
				}

				gotSkill, err := caps.WrittenSkillDir(loc)
				if err != nil {
					t.Errorf("WrittenSkillDir(%s) = error %v, want no error", loc, err)
				}
				if wantSkill, ok := tc.skillDir[loc]; ok && gotSkill != wantSkill {
					t.Errorf("WrittenSkillDir(%s) = %q, want %q", loc, gotSkill, wantSkill)
				}
			}
		})
	}
}

// TestCapabilitiesDeclared_AntigravityMigrationAware pins D-02's migration
// narrowing separately from the main table above: under a fresh fake home
// (no Antigravity config at all) Capabilities().MCPConfig(global) must
// equal the UNIFIED path (the one Install actually writes on a fresh
// machine); once only the legacy file exists (unmigrated), it must equal
// the LEGACY path.
func TestCapabilitiesDeclared_AntigravityMigrationAware(t *testing.T) {
	home := fakeHome(t)
	target, ok := GetTarget(Antigravity)
	if !ok {
		t.Fatalf("GetTarget(antigravity): not registered")
	}

	wantUnified := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	got, err := target.Capabilities().MCPConfig(LocationGlobal)
	if err != nil {
		t.Fatalf("MCPConfig(global) on a fresh home: %v", err)
	}
	if got != wantUnified {
		t.Fatalf("MCPConfig(global) on a fresh home = %q, want unified %q", got, wantUnified)
	}

	legacy := filepath.Join(home, ".gemini", "antigravity", "mcp_config.json")
	writeFile(t, legacy, `{"mcpServers":{}}`)

	wantLegacy := legacy
	got, err = target.Capabilities().MCPConfig(LocationGlobal)
	if err != nil {
		t.Fatalf("MCPConfig(global) with only the legacy file present: %v", err)
	}
	if got != wantLegacy {
		t.Fatalf("MCPConfig(global) with only the legacy file present = %q, want legacy %q", got, wantLegacy)
	}
}
