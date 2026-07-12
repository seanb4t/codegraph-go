package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hermesCliToolsetName is the entry appended to platform_toolsets.cli so
// Hermes CLI profiles don't filter codegraph's tools out of normal
// sessions (D-06, Corrected Per-Agent Parity Table).
const hermesCliToolsetName = "mcp-codegraph"

// hermesTarget implements AgentTarget for Hermes Agent (D-06, D-07).
// Global scope only — $HERMES_HOME/config.yaml (default ~/.hermes).
// Edited via hand-rolled YAML line-range surgery (no YAML library, same
// minimal-deps reasoning as Codex's TOML splice): a top-level
// mcp_servers.codegraph block plus an indent-matched append to
// platform_toolsets.cli. Hermes has no AGENTS.md-equivalent instructions
// convention TS integrates with — never write one here (parity
// regression, Pitfall 2).
type hermesTarget struct{}

func init() {
	registerTarget(hermesTarget{})
}

func (hermesTarget) ID() TargetID                       { return Hermes }
func (hermesTarget) DisplayName() string                { return "Hermes Agent" }
func (hermesTarget) SupportsLocation(loc Location) bool { return loc == LocationGlobal }

// hermesConfigPath resolves $HERMES_HOME/config.yaml, defaulting
// HERMES_HOME to ~/.hermes when unset.
func hermesConfigPath() (string, error) {
	if v := os.Getenv("HERMES_HOME"); v != "" {
		return filepath.Join(v, "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hermes", "config.yaml"), nil
}

// yamlBlockRange locates [start, end) of the block beginning at the line
// consisting of exactly `indent` leading spaces followed by header (e.g.
// header="mcp_servers:", indent=0 for a top-level mapping key, or
// header="codegraph:", indent=2 for a child one level deeper). The block
// ends at the next non-blank line whose own indentation is <= indent (a
// sibling or ancestor key), or at len(content) if none follows. found is
// false if no line exactly matches the indented header.
//
// Every comparison strips a trailing "\r" first (CR-03): a CRLF-line-ended
// config.yaml (a realistic case for a project that ships Windows release
// binaries) would otherwise never match headerLine, and a bare "\r" blank
// line would otherwise be mistaken for a non-blank sibling/ancestor line
// ending the block — both mirror toml.go's TrimSpace-based CRLF safety.
func yamlBlockRange(content, header string, indent int) (start, end int, found bool) {
	headerLine := strings.Repeat(" ", indent) + header
	lines := strings.Split(content, "\n")

	offset := 0
	headerIdx := -1
	for i, line := range lines {
		if strings.TrimRight(line, "\r") == headerLine {
			headerIdx = i
			start = offset
			break
		}
		offset += len(line) + 1
	}
	if headerIdx == -1 {
		return 0, 0, false
	}

	scanOffset := start + len(lines[headerIdx]) + 1
	for i := headerIdx + 1; i < len(lines); i++ {
		line := lines[i]
		bare := strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(bare, " ")
		lineIndent := len(bare) - len(trimmed)
		if trimmed != "" && lineIndent <= indent {
			return start, scanOffset, true
		}
		scanOffset += len(line) + 1
	}
	return start, len(content), true
}

// yamlListBlockRange locates [start, end) of a "key:" line at exactly
// `indent` leading spaces plus every immediately-following line that is a
// "- item" continuation of ITS list — a line qualifies as a continuation
// if its trimmed text starts with "- " and its own indentation is >=
// indent (PyYAML's default block-sequence style emits list items at the
// SAME indent as their parent key, not deeper — Pitfall 5; hand-authored
// YAML often indents deeper, which >= also accepts). The block ends at
// the first line that isn't a qualifying continuation, or at
// len(content).
func yamlListBlockRange(content, header string, indent int) (start, end int, found bool) {
	headerLine := strings.Repeat(" ", indent) + header
	lines := strings.Split(content, "\n")

	offset := 0
	headerIdx := -1
	for i, line := range lines {
		if strings.TrimRight(line, "\r") == headerLine {
			headerIdx = i
			start = offset
			break
		}
		offset += len(line) + 1
	}
	if headerIdx == -1 {
		return 0, 0, false
	}

	scanOffset := start + len(lines[headerIdx]) + 1
	for i := headerIdx + 1; i < len(lines); i++ {
		line := lines[i]
		bare := strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(bare, " ")
		lineIndent := len(bare) - len(trimmed)
		if strings.HasPrefix(trimmed, "- ") && lineIndent >= indent {
			scanOffset += len(line) + 1
			continue
		}
		return start, scanOffset, true
	}
	return start, len(content), true
}

// yamlRemoveRange deletes content[start:end], collapsing surrounding
// blank lines the same way stripTOMLTable/removeMarkedSection do, so
// removing a block this package itself appended restores the exact
// pre-append bytes.
func yamlRemoveRange(content string, start, end int) string {
	before := strings.TrimRight(content[:start], "\n")
	after := strings.TrimLeft(content[end:], "\n")
	switch {
	case before == "" && after == "":
		return ""
	case before == "":
		if !strings.HasSuffix(after, "\n") {
			return after + "\n"
		}
		return after
	case after == "":
		return before + "\n"
	default:
		return before + "\n\n" + after
	}
}

// hermesCodegraphChildBlock renders the "  codegraph:" child block body
// (4-space indent for its own scalar/list keys) for execPath.
func hermesCodegraphChildBlock(execPath string) string {
	var b strings.Builder
	b.WriteString("  codegraph:\n")
	b.WriteString("    command: " + tomlString(execPath) + "\n")
	b.WriteString("    args:\n")
	b.WriteString("      - serve\n")
	b.WriteString("      - --mcp\n")
	b.WriteString("    timeout: 120\n")
	b.WriteString("    connect_timeout: 60\n")
	b.WriteString("    enabled: true\n")
	return b.String()
}

// hermesSpliceMcpServersBlock upserts the top-level mcp_servers.codegraph
// block into content, preserving every other top-level key and, when
// mcp_servers already exists, every sibling server under it (T-06-03-01).
func hermesSpliceMcpServersBlock(content, execPath string) string {
	childBlock := hermesCodegraphChildBlock(execPath)

	parentStart, parentEnd, parentFound := yamlBlockRange(content, "mcp_servers:", 0)
	if !parentFound {
		block := "mcp_servers:\n" + childBlock
		trimmed := strings.TrimRight(content, "\n")
		if trimmed == "" {
			return block
		}
		return trimmed + "\n\n" + block
	}

	parentBlock := content[parentStart:parentEnd]
	childStart, childEnd, childFound := yamlBlockRange(parentBlock, "codegraph:", 2)
	if !childFound {
		newParentBlock := strings.TrimRight(parentBlock, "\n") + "\n" + childBlock
		return content[:parentStart] + newParentBlock + content[parentEnd:]
	}

	if parentBlock[childStart:childEnd] == childBlock {
		return content
	}
	newParentBlock := parentBlock[:childStart] + childBlock + parentBlock[childEnd:]
	return content[:parentStart] + newParentBlock + content[parentEnd:]
}

// hermesStripMcpServersBlock removes the codegraph child from
// mcp_servers, removing the whole mcp_servers block too if codegraph was
// its only child (D-07 keep-clean). A missing block is a no-op.
func hermesStripMcpServersBlock(content string) string {
	parentStart, parentEnd, parentFound := yamlBlockRange(content, "mcp_servers:", 0)
	if !parentFound {
		return content
	}
	parentBlock := content[parentStart:parentEnd]
	childStart, childEnd, childFound := yamlBlockRange(parentBlock, "codegraph:", 2)
	if !childFound {
		return content
	}

	remainder := parentBlock[:childStart] + parentBlock[childEnd:]
	afterHeader := strings.TrimPrefix(remainder, "mcp_servers:\n")
	if strings.TrimSpace(afterHeader) == "" {
		return yamlRemoveRange(content, parentStart, parentEnd)
	}
	return content[:parentStart] + remainder + content[parentEnd:]
}

// hermesAppendCliToolset appends hermesCliToolsetName to
// platform_toolsets.cli, detecting and matching the list's ACTUAL
// existing item indent (Pitfall 5) rather than assuming a fixed depth.
// If the entry is already present, content is returned unchanged (D-07).
func hermesAppendCliToolset(content string) string {
	parentStart, parentEnd, parentFound := yamlBlockRange(content, "platform_toolsets:", 0)
	if !parentFound {
		block := "platform_toolsets:\n  cli:\n  - " + hermesCliToolsetName + "\n"
		trimmed := strings.TrimRight(content, "\n")
		if trimmed == "" {
			return block
		}
		return trimmed + "\n\n" + block
	}

	parentBlock := content[parentStart:parentEnd]
	// yamlListBlockRange (not yamlBlockRange) is required here: PyYAML's
	// default block-sequence style emits "cli:"'s list items at the SAME
	// indent as "cli:" itself, which yamlBlockRange's map-child boundary
	// rule (indent <= parent) would mistake for the end of the block.
	cliStart, cliEnd, cliFound := yamlListBlockRange(parentBlock, "cli:", 2)
	if !cliFound {
		newParentBlock := strings.TrimRight(parentBlock, "\n") + "\n  cli:\n  - " + hermesCliToolsetName + "\n"
		return content[:parentStart] + newParentBlock + content[parentEnd:]
	}
	cliBlock := parentBlock[cliStart:cliEnd]

	itemIndent := -1
	for _, line := range strings.Split(cliBlock, "\n")[1:] {
		trimmed := strings.TrimLeft(line, " ")
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		if itemIndent == -1 {
			itemIndent = len(line) - len(trimmed)
		}
		if strings.TrimSpace(strings.TrimPrefix(trimmed, "-")) == hermesCliToolsetName {
			return content // already present — no duplicate (D-07).
		}
	}
	if itemIndent == -1 {
		// No existing items to detect from — PyYAML's own default is to
		// emit list items at the SAME indent as their parent key.
		itemIndent = 2
	}

	newItemLine := strings.Repeat(" ", itemIndent) + "- " + hermesCliToolsetName + "\n"
	newCliBlock := strings.TrimRight(cliBlock, "\n") + "\n" + newItemLine
	newParentBlock := parentBlock[:cliStart] + newCliBlock + parentBlock[cliEnd:]
	return content[:parentStart] + newParentBlock + content[parentEnd:]
}

// hermesRemoveCliToolset removes the single "- mcp-codegraph" list item
// line this package's own hermesAppendCliToolset would have added,
// regardless of its indent — the inverse of hermesAppendCliToolset. The
// search is scoped to the platform_toolsets.cli block via
// yamlListBlockRange (WR-07), matching hermesAppendCliToolset's own
// scoping — an unscoped whole-file search would delete the first line
// anywhere that happens to equal "- mcp-codegraph", including an
// unrelated list elsewhere in the user's config.
func hermesRemoveCliToolset(content string) string {
	parentStart, parentEnd, parentFound := yamlBlockRange(content, "platform_toolsets:", 0)
	if !parentFound {
		return content
	}
	parentBlock := content[parentStart:parentEnd]
	cliStart, cliEnd, cliFound := yamlListBlockRange(parentBlock, "cli:", 2)
	if !cliFound {
		return content
	}
	cliBlock := parentBlock[cliStart:cliEnd]

	lines := strings.Split(cliBlock, "\n")
	removed := false
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !removed && strings.TrimSpace(line) == "- "+hermesCliToolsetName {
			removed = true
			continue
		}
		out = append(out, line)
	}
	if !removed {
		return content
	}
	newCliBlock := strings.Join(out, "\n")
	newParentBlock := parentBlock[:cliStart] + newCliBlock + parentBlock[cliEnd:]
	return content[:parentStart] + newParentBlock + content[parentEnd:]
}

func hermesConfigured(content string) bool {
	parentStart, parentEnd, found := yamlBlockRange(content, "mcp_servers:", 0)
	if !found {
		return false
	}
	_, _, childFound := yamlBlockRange(content[parentStart:parentEnd], "codegraph:", 2)
	return childFound
}

func (hermesTarget) Detect(loc Location) DetectionResult {
	if loc != LocationGlobal {
		return DetectionResult{}
	}
	configPath, err := hermesConfigPath()
	if err != nil {
		return DetectionResult{}
	}
	installed := fileExists(configPath)
	if !installed {
		if dir := filepath.Dir(configPath); fileExists(dir) {
			installed = true
		}
	}
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: hermesConfigured(readFileOrEmpty(configPath)),
		ConfigPath:        configPath,
	}
}

func (hermesTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult
	if loc != LocationGlobal {
		return result
	}

	configPath, err := hermesConfigPath()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve hermes config path: %w", err))
		return result
	}
	existed := fileExists(configPath)
	existing := readFileOrEmpty(configPath)

	updated := hermesSpliceMcpServersBlock(existing, opts.ExecPath)
	updated = hermesAppendCliToolset(updated)

	if updated == existing {
		result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionUnchanged})
		return result
	}
	if err := atomicWriteFile(configPath, updated); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, err))
		return result
	}
	action := ActionUpdated
	if !existed {
		action = ActionCreated
	}
	result.Files = append(result.Files, FileResult{Path: configPath, Action: action})
	return result
}

func (hermesTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult
	if loc != LocationGlobal {
		return result
	}

	configPath, err := hermesConfigPath()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve hermes config path: %w", err))
		return result
	}
	if !fileExists(configPath) {
		result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionNotFound})
		return result
	}

	existing := readFileOrEmpty(configPath)
	updated := hermesStripMcpServersBlock(existing)
	updated = hermesRemoveCliToolset(updated)

	if updated == existing {
		result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionNotFound})
		return result
	}
	if err := atomicWriteFile(configPath, updated); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, err))
		return result
	}
	result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionRemoved})
	return result
}

func (hermesTarget) DescribePaths(loc Location) []string {
	if loc != LocationGlobal {
		return nil
	}
	configPath, err := hermesConfigPath()
	if err != nil {
		return nil
	}
	return []string{configPath}
}
