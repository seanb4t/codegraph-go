package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"
)

// opencodeSchemaURL is the $schema value opencode's own generated configs
// carry (https://opencode.ai/docs/config, https://opencode.ai/docs/mcp-servers).
const opencodeSchemaURL = "https://opencode.ai/config.json"

// opencodeTarget implements AgentTarget for opencode (D-05a, D-07). Both
// scopes are supported: global at <cfgdir>/opencode/opencode.jsonc (or
// .json, XDG_CONFIG_HOME/~/.config resolution — never %APPDATA%, Pitfall
// 4), local at ./opencode.jsonc (or ./opencode.json). Config edits go
// through github.com/tailscale/hujson (Parse -> Patch (RFC 6902) -> Format
// -> Pack) so comments in the user's existing JSONC survive — plain
// encoding/json round-tripping would silently drop them (T-06-03-01).
// opencode's mcp.codegraph.command is a combined [binary, ...args] array,
// unlike every other JSON-shaped target's separate command+args fields.
type opencodeTarget struct{}

func init() {
	registerTarget(opencodeTarget{})
}

func (opencodeTarget) ID() TargetID                   { return Opencode }
func (opencodeTarget) DisplayName() string            { return "opencode" }
func (opencodeTarget) SupportsLocation(Location) bool { return true }

// resolveOpencodeConfigDir returns the base config directory opencode
// itself resolves to: XDG_CONFIG_HOME if set and non-empty, else
// ~/.config — unconditionally on every OS including Windows. opencode
// never reads %APPDATA% (Pitfall 4, TS issue #535); do not add a
// runtime.GOOS branch here.
func resolveOpencodeConfigDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// opencodeConfigPath returns the config file to edit for loc: an existing
// opencode.jsonc wins, else an existing opencode.json, else opencode.jsonc
// is the default target for a not-yet-existing file.
func opencodeConfigPath(loc Location) (string, error) {
	var dir string
	if loc == LocationLocal {
		dir = "."
	} else {
		cfgDir, err := resolveOpencodeConfigDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cfgDir, "opencode")
	}

	jsonc := filepath.Join(dir, "opencode.jsonc")
	if fileExists(jsonc) {
		return jsonc, nil
	}
	if jsonPath := filepath.Join(dir, "opencode.json"); fileExists(jsonPath) {
		return jsonPath, nil
	}
	return jsonc, nil
}

// opencodeInstructionsPath: global is <cfgdir>/opencode/AGENTS.md, local is
// the project root ./AGENTS.md.
func opencodeInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return "AGENTS.md", nil
	}
	cfgDir, err := resolveOpencodeConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "opencode", "AGENTS.md"), nil
}

// opencodeMcpEntry builds the mcp.codegraph entry — a combined
// [binary, ...args] command array is opencode's own convention, unlike
// every other JSON-shaped target in this package.
func opencodeMcpEntry(execPath string) map[string]any {
	return map[string]any{
		"type":    "local",
		"command": []string{execPath, "serve", "--mcp"},
		"enabled": true,
	}
}

// writeOpencodeEntry surgically patches path's JSONC to set $schema (if
// absent) and mcp.codegraph (if absent or different), preserving every
// comment and unrelated key via hujson (T-06-03-01). Writes only if
// something actually changed (D-07 idempotency).
func writeOpencodeEntry(path, execPath string) (FileResult, error) {
	existed := fileExists(path)

	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return FileResult{}, err
		}
		raw = nil
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		raw = []byte("{}")
	}

	v, perr := hujson.Parse(raw)
	if perr != nil {
		// Malformed existing config — defensive empty-object fallback
		// (V5), mirroring readJSONFile's empty-map fallback for the
		// other JSON-shaped targets.
		v, perr = hujson.Parse([]byte("{}"))
		if perr != nil {
			return FileResult{}, perr
		}
	}

	desiredEntry := opencodeMcpEntry(execPath)
	desiredJSON, err := json.Marshal(desiredEntry)
	if err != nil {
		return FileResult{}, err
	}

	var ops []string
	if v.Find("/$schema") == nil {
		schemaJSON, _ := json.Marshal(opencodeSchemaURL)
		ops = append(ops, `{"op":"add","path":"/$schema","value":`+string(schemaJSON)+`}`)
	}
	if v.Find("/mcp") == nil {
		ops = append(ops, `{"op":"add","path":"/mcp","value":{}}`)
	}

	needsEntry := true
	if cur := v.Find("/mcp/codegraph"); cur != nil {
		if std, serr := hujson.Standardize(cur.Pack()); serr == nil {
			var curVal any
			if json.Unmarshal(std, &curVal) == nil {
				normDesired, nerr := normalizeJSON(desiredEntry)
				if nerr == nil && jsonDeepEqual(curVal, normDesired) {
					needsEntry = false
				}
			}
		}
	}
	if needsEntry {
		ops = append(ops, `{"op":"add","path":"/mcp/codegraph","value":`+string(desiredJSON)+`}`)
	}

	if len(ops) == 0 {
		return FileResult{Path: path, Action: ActionUnchanged}, nil
	}

	patch := []byte("[" + strings.Join(ops, ",") + "]")
	if err := v.Patch(patch); err != nil {
		return FileResult{}, err
	}
	v.Format()

	if err := atomicWriteFile(path, string(v.Pack())); err != nil {
		return FileResult{}, err
	}
	action := ActionUpdated
	if !existed {
		action = ActionCreated
	}
	return FileResult{Path: path, Action: action}, nil
}

// removeOpencodeEntry removes mcp.codegraph from path's JSONC, preserving
// comments and every unrelated key. If that empties the mcp object, mcp
// itself is removed too (D-07 keep-clean, mirrors removeMcpEntry). Never
// errors when there's nothing to remove.
func removeOpencodeEntry(path string) (FileResult, error) {
	if !fileExists(path) {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return FileResult{}, err
	}
	v, perr := hujson.Parse(raw)
	if perr != nil {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}

	mcpVal := v.Find("/mcp")
	if mcpVal == nil || v.Find("/mcp/codegraph") == nil {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}

	op := `{"op":"remove","path":"/mcp/codegraph"}`
	if obj, ok := mcpVal.Value.(*hujson.Object); ok && len(obj.Members) == 1 {
		op = `{"op":"remove","path":"/mcp"}`
	}

	if err := v.Patch([]byte("[" + op + "]")); err != nil {
		return FileResult{}, err
	}
	v.Format()
	if err := atomicWriteFile(path, string(v.Pack())); err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: path, Action: ActionRemoved}, nil
}

// opencodeMcpEntryPresent reports whether path's JSONC already has an
// mcp.codegraph entry, used by Detect's AlreadyConfigured field.
func opencodeMcpEntryPresent(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	v, perr := hujson.Parse(raw)
	if perr != nil {
		return false
	}
	return v.Find("/mcp/codegraph") != nil
}

// opencodeSweepStaleAppData self-heals a legacy %APPDATA%/opencode
// codegraph entry a broken pre-fix TS install may have left behind
// (Pitfall 4, TS issue #535) — only when APPDATA is set and resolves to a
// directory different from the correct XDG-resolved config dir, so this
// never touches the real config file on a normal run.
func opencodeSweepStaleAppData(resolvedCfgDir string) {
	appData := os.Getenv("APPDATA")
	if appData == "" || filepath.Clean(appData) == filepath.Clean(resolvedCfgDir) {
		return
	}
	for _, name := range []string{"opencode.jsonc", "opencode.json"} {
		stalePath := filepath.Join(appData, "opencode", name)
		if fileExists(stalePath) {
			removeOpencodeEntry(stalePath)
		}
	}
}

func (opencodeTarget) Detect(loc Location) DetectionResult {
	configPath, err := opencodeConfigPath(loc)
	if err != nil {
		return DetectionResult{}
	}
	installed := fileExists(configPath)
	if !installed {
		installed = fileExists(filepath.Dir(configPath))
	}
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: opencodeMcpEntryPresent(configPath),
		ConfigPath:        configPath,
	}
}

func (opencodeTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult

	if configPath, err := opencodeConfigPath(loc); err == nil {
		if fr, err := writeOpencodeEntry(configPath, opts.ExecPath); err == nil {
			result.Files = append(result.Files, fr)
		}
	}

	if instrPath, err := opencodeInstructionsPath(loc); err == nil {
		if fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody()); err == nil {
			result.Files = append(result.Files, fr)
		}
	}

	if loc == LocationGlobal {
		if cfgDir, err := resolveOpencodeConfigDir(); err == nil {
			opencodeSweepStaleAppData(cfgDir)
		}
	}

	return result
}

func (opencodeTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult

	if configPath, err := opencodeConfigPath(loc); err == nil {
		if fr, err := removeOpencodeEntry(configPath); err == nil {
			result.Files = append(result.Files, fr)
		}
	}

	if instrPath, err := opencodeInstructionsPath(loc); err == nil {
		if action, err := removeMarkedSection(instrPath, codegraphSectionStart, codegraphSectionEnd); err == nil {
			result.Files = append(result.Files, FileResult{Path: instrPath, Action: action})
		}
	}

	return result
}

func (opencodeTarget) DescribePaths(loc Location) []string {
	var paths []string
	if p, err := opencodeConfigPath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := opencodeInstructionsPath(loc); err == nil {
		paths = append(paths, p)
	}
	return paths
}
