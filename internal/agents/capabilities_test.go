package agents

import (
	"path/filepath"
	"testing"
)

// stringSetEqual reports whether a and b contain the same elements,
// ignoring order — used by TestCapabilitiesTableDrivesDerivations to
// compare an independently-built expected path set against
// DescribePaths(loc)'s actual output.
func stringSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]int, len(a))
	for _, s := range a {
		am[s]++
	}
	for _, s := range b {
		am[s]--
	}
	for _, n := range am {
		if n != 0 {
			return false
		}
	}
	return true
}

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
				// AGENT-07, maintainer decision 1A (05-07): the one dir agy
				// 1.2.6 was live-proven to read (05-LIVE-SESSIONS.md).
				LocationGlobal: filepath.Join(home, ".gemini", "config", "skills", "codegraph"),
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
				// D-06 (05-04): Cursor writes the SHARED skill package, not
				// a Cursor-specific directory.
				LocationGlobal: filepath.Join(home, ".agents", "skills", "codegraph"),
				LocationLocal:  filepath.Join(".agents", "skills", "codegraph"),
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
				// AGENT-10 (05-05): Gemini writes its own harness-specific
				// directory, not the shared .agents/skills/codegraph path.
				LocationGlobal: filepath.Join(home, ".gemini", "skills", "codegraph"),
				LocationLocal:  filepath.Join(".gemini", "skills", "codegraph"),
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
				// AGENT-11 (05-05): Kiro writes its own harness-specific
				// directory, not the shared .agents/skills/codegraph path.
				LocationGlobal: filepath.Join(home, ".kiro", "skills", "codegraph"),
				LocationLocal:  filepath.Join(".kiro", "skills", "codegraph"),
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
				// D-06 (05-04): opencode writes the SHARED skill package,
				// not an opencode-specific directory.
				LocationGlobal: filepath.Join(home, ".agents", "skills", "codegraph"),
				LocationLocal:  filepath.Join(".agents", "skills", "codegraph"),
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

// expectedDeclaredPaths independently builds the set of paths
// TestCapabilitiesTableDrivesDerivations expects DescribePaths(loc) to
// return, WITHOUT calling describeDeclaredPaths — it re-derives the same
// property from Capabilities' own fields, so this test does not merely
// re-run the code under test against itself (D-00/D-03).
func expectedDeclaredPaths(t *testing.T, caps Capabilities, loc Location) []string {
	t.Helper()
	var want []string
	if caps.MCPConfig != nil {
		if p, err := caps.MCPConfig(loc); err == nil && p != "" {
			want = append(want, p)
		}
	}
	if caps.Instructions != nil {
		if p, err := caps.Instructions(loc); err == nil && p != "" {
			want = append(want, p)
		}
	}
	if caps.Hooks == HooksClaudeJSON {
		if settingsPath, err := claudeSettingsPath(loc); err == nil {
			want = append(want, settingsPath)
		}
		if scriptPath, err := claudeHooksScriptPath(loc); err == nil {
			want = append(want, scriptPath)
		}
	}
	if caps.SkillDirs != nil {
		if dirs, err := caps.SkillDirs(loc); err == nil && len(dirs) > 0 {
			want = append(want, filepath.Join(dirs[0], skillFileName), filepath.Join(dirs[0], skillManifestFileName))
		}
	}
	// Deduplicate, matching describeDeclaredPaths' own dedup contract.
	seen := make(map[string]bool, len(want))
	out := want[:0]
	for _, p := range want {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// TestCapabilitiesTableDrivesDerivations is the D-03 guard: for every
// registered target x {global, local}, SupportsLocation, DescribePaths and
// Detect's ConfigPath are all consistent with an independently-computed
// expectation built from Capabilities() alone — never by calling
// describeDeclaredPaths itself. Family (a1) in 05-MUTATION-LOG.md proves
// this guard RED against a planted divergence.
func TestCapabilitiesTableDrivesDerivations(t *testing.T) {
	leaves := 0
	for _, target := range AllTargets() {
		for _, loc := range []Location{LocationGlobal, LocationLocal} {
			target, loc := target, loc
			t.Run(string(target.ID())+"/"+string(loc), func(t *testing.T) {
				fakeHome(t)
				t.Chdir(t.TempDir())
				leaves++

				caps := target.Capabilities()
				wantSupported := caps.Supports(loc)
				if got := target.SupportsLocation(loc); got != wantSupported {
					t.Fatalf("SupportsLocation(%s) = %v, want %v (Scopes=%v)", loc, got, wantSupported, caps.Scopes)
				}

				if !wantSupported {
					if paths := target.DescribePaths(loc); len(paths) != 0 {
						t.Errorf("DescribePaths(%s) for an unsupported location = %v, want empty", loc, paths)
					}
					if got := target.Detect(loc); got != (DetectionResult{}) {
						t.Errorf("Detect(%s) for an unsupported location = %+v, want the zero value", loc, got)
					}
					return
				}

				want := expectedDeclaredPaths(t, caps, loc)
				got := target.DescribePaths(loc)
				if !stringSetEqual(want, got) {
					t.Errorf("DescribePaths(%s) = %v, want set-equal to %v", loc, got, want)
				}
				seen := make(map[string]bool, len(got))
				for _, p := range got {
					if seen[p] {
						t.Errorf("DescribePaths(%s) has a duplicate entry %q: %v", loc, p, got)
					}
					seen[p] = true
				}

				wantConfigPath, err := caps.MCPConfig(loc)
				if err != nil {
					t.Fatalf("MCPConfig(%s): %v", loc, err)
				}
				if detected := target.Detect(loc); detected.ConfigPath != wantConfigPath {
					t.Errorf("Detect(%s).ConfigPath = %q, want %q", loc, detected.ConfigPath, wantConfigPath)
				}
			})
		}
	}

	if leaves < 16 {
		t.Fatalf("executed %d leaf subtests, want at least 16 (8 targets x 2 locations)", leaves)
	}
}

// TestCapabilitiesMatchInstallWrites is the D-03 guard's other direction:
// after a real Install call, every path the table declares (DescribePaths,
// evaluated AFTER install so Antigravity's migration-aware resolution
// reflects what Install just wrote) must appear among the files Install
// reported, and every created/updated/unchanged file Install reported must
// be declared — the sole named exception is Antigravity's ".migrated"
// marker, which is migration bookkeeping, not configuration. Family (a2)
// in 05-MUTATION-LOG.md proves this guard RED against a planted omission.
func TestCapabilitiesMatchInstallWrites(t *testing.T) {
	leaves := 0
	for _, target := range AllTargets() {
		for _, loc := range []Location{LocationGlobal, LocationLocal} {
			target, loc := target, loc
			if !target.Capabilities().Supports(loc) {
				continue
			}
			t.Run(string(target.ID())+"/"+string(loc), func(t *testing.T) {
				fakeHome(t)
				t.Chdir(t.TempDir())
				leaves++

				result := target.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
				if len(result.Errors) != 0 {
					t.Fatalf("Install(%s) returned errors: %v", loc, result.Errors)
				}

				declared := make(map[string]bool)
				for _, p := range target.DescribePaths(loc) {
					declared[p] = true
				}

				written := make(map[string]bool)
				for _, fr := range result.Files {
					switch fr.Action {
					case ActionCreated, ActionUpdated, ActionUnchanged:
						written[fr.Path] = true
					}
				}

				for p := range declared {
					if !written[p] {
						t.Errorf("declared path %q was not among Install's created/updated/unchanged files: %v", p, result.Files)
					}
				}
				for p := range written {
					if declared[p] {
						continue
					}
					// Antigravity's ".migrated" marker is migration
					// bookkeeping, never configuration the table declares
					// (D-02) — the sole named exception both directions of
					// this guard exempt.
					if target.ID() == Antigravity && filepath.Base(p) == ".migrated" {
						continue
					}
					t.Errorf("Install wrote/kept %q, which the table does not declare: declared=%v", p, target.DescribePaths(loc))
				}
			})
		}
	}

	if leaves < 13 {
		t.Fatalf("executed %d leaf subtests, want at least 13 (5 global+local targets x 2, 3 global-only targets x 1)", leaves)
	}
}

// fakeHooksCodexJSONTarget is a minimal AgentTarget stub reaching only
// describeDeclaredPaths's HookFiles branch: it declares Hooks:
// HooksCodexJSON, the mechanism HookFiles has no case for yet (Phase 7
// owns Codex's hooks literal). Every other method is an unused stub —
// WR-01's regression test never calls them.
type fakeHooksCodexJSONTarget struct{}

func (fakeHooksCodexJSONTarget) ID() TargetID        { return TargetID("fake-codex-hooks") }
func (fakeHooksCodexJSONTarget) DisplayName() string { return "Fake Codex Hooks Target" }
func (fakeHooksCodexJSONTarget) SupportsLocation(loc Location) bool {
	return loc == LocationGlobal
}
func (fakeHooksCodexJSONTarget) Detect(Location) DetectionResult { return DetectionResult{} }
func (fakeHooksCodexJSONTarget) Install(Location, InstallOptions) WriteResult {
	return WriteResult{}
}
func (fakeHooksCodexJSONTarget) Uninstall(Location) WriteResult { return WriteResult{} }
func (t fakeHooksCodexJSONTarget) DescribePaths(loc Location) []string {
	return describeDeclaredPaths(t, loc)
}
func (fakeHooksCodexJSONTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal},
		ConfigFormat: ConfigFormatJSON,
		Hooks:        HooksCodexJSON,
		MCPConfig:    func(Location) (string, error) { return "/fake/config.json", nil },
	}
}

// TestDescribeDeclaredPaths_PanicsOnUndeclaredHookFiles is WR-01's
// regression test (code review 05-REVIEW.md): errHookFilesUndeclared's and
// HookFiles's doc comments both promise that a target declaring
// HooksCodexJSON without a HookFiles case "must fail loudly here ...
// rather than silently describing no hook files at all." Before the fix,
// describeDeclaredPaths discards every HookFiles error unconditionally,
// so DescribePaths returns an incomplete-but-successful path list instead
// of failing loudly.
func TestDescribeDeclaredPaths_PanicsOnUndeclaredHookFiles(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("describeDeclaredPaths did not panic for a target declaring HooksCodexJSON with no HookFiles case — the documented loud-failure contract was not honored")
		}
	}()
	fakeHooksCodexJSONTarget{}.DescribePaths(LocationGlobal)
}
