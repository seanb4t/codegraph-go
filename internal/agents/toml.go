package agents

import "strings"

// spliceTOMLTable and stripTOMLTable are a hand-rolled, ~100-line TOML
// single-table-block editor that uses the same per-agent TOML-splicing
// behavior the marker contract defines
// (D-05a): a full TOML parser/serializer dependency is unjustified for
// editing exactly one dotted-key table ("~50KB dependency for ~6 lines of
// output") — a minimal text-splice strategy instead of reaching for a
// general TOML library (minimal-deps constraint). This file is used only
// by Codex's config.toml edit.

// spliceTOMLTable returns content with the "[tableName]" block replaced
// (or, if absent, appended) so it contains exactly header + bodyLines,
// preserving every other byte of content verbatim (T-06-03-01). When the
// table is already present with byte-identical body, content is returned
// unchanged (D-07 idempotency).
func spliceTOMLTable(content, tableName string, bodyLines []string) string {
	var block strings.Builder
	block.WriteString("[" + tableName + "]\n")
	for _, line := range bodyLines {
		block.WriteString(line)
		block.WriteByte('\n')
	}
	newBlock := block.String()

	start, end, found := findTOMLTableRange(content, tableName)
	if !found {
		trimmed := strings.TrimRight(content, "\n")
		if trimmed == "" {
			return newBlock
		}
		return trimmed + "\n\n" + newBlock
	}

	if content[start:end] == newBlock {
		return content
	}
	return content[:start] + newBlock + content[end:]
}

// stripTOMLTable removes the "[tableName]" block (header through the next
// top-level "[...]" header or EOF) from content, restoring the file to its
// exact pre-splice bytes when applied after spliceTOMLTable (D-07/D-08). A
// missing table is a no-op — content is returned unchanged.
func stripTOMLTable(content, tableName string) string {
	start, end, found := findTOMLTableRange(content, tableName)
	if !found {
		return content
	}

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

// findTOMLTableRange locates the byte range [start, end) of the
// "[tableName]" block within content: start is the byte offset of the
// "[tableName]" header line itself, end is the byte offset of the next
// line beginning with "[" (a top-level table header) at column 0, or
// len(content) if none follows (EOF). found is false if no line
// (trimmed of surrounding whitespace) exactly equals "[tableName]".
func findTOMLTableRange(content, tableName string) (start, end int, found bool) {
	header := "[" + tableName + "]"
	lines := strings.Split(content, "\n")

	offset := 0
	headerLine := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == header {
			headerLine = i
			start = offset
			break
		}
		offset += len(line) + 1
	}
	if headerLine == -1 {
		return 0, 0, false
	}

	scanOffset := start + len(lines[headerLine]) + 1
	for i := headerLine + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "[") {
			return start, scanOffset, true
		}
		scanOffset += len(lines[i]) + 1
	}
	return start, len(content), true
}

// tomlString renders s as a TOML basic string, escaping backslashes and
// double quotes (the two characters that would otherwise break out of the
// quoted form) — matters on Windows where ExecPath contains backslashes.
func tomlString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// tomlStringArray renders items as a TOML array of basic strings, e.g.
// ["serve", "--mcp"].
func tomlStringArray(items []string) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(tomlString(item))
	}
	b.WriteByte(']')
	return b.String()
}
