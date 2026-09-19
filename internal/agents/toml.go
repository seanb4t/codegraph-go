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
// "[tableName]" header line itself. end is the byte offset of the first
// following table header (at ANY indentation) whose dotted path is
// neither tableName itself nor prefixed by "tableName." (codegraph's own
// subtables stay inside its range), backed off to the start of the
// contiguous run of blank/comment-only lines immediately preceding that
// header — so a comment or blank run sitting just above the next table
// stays with what follows, never with codegraph's own block (D-07,
// CODEX-02's "comments preserved"). With no such following header, the
// same back-off applies from EOF. A "[" or "]" inside a multi-line
// basic/literal string or an unfinished multi-line array/inline table is
// never mistaken for a header (D-07). found is false if no line, with
// surrounding whitespace and any trailing "# comment" removed, is
// exactly "[tableName]".
func findTOMLTableRange(content, tableName string) (start, end int, found bool) {
	header := "[" + tableName + "]"
	subtablePrefix := tableName + "."
	lines := splitTOMLLines(content)

	var state tomlLineState
	headerIdx := -1
	for i, ln := range lines {
		isHeader := state.neutral() && isTOMLHeaderLine(ln.text)
		if isHeader && strings.TrimSpace(stripTOMLTrailingComment(ln.text)) == header {
			headerIdx = i
			start = ln.offset
			state.scan(ln.text)
			break
		}
		state.scan(ln.text)
	}
	if headerIdx == -1 {
		return 0, 0, false
	}

	endIdx := -1
	for i := headerIdx + 1; i < len(lines); i++ {
		ln := lines[i]
		isHeader := state.neutral() && isTOMLHeaderLine(ln.text)
		if isHeader {
			path := tomlHeaderPath(ln.text)
			if path != tableName && !strings.HasPrefix(path, subtablePrefix) {
				endIdx = i
				break
			}
		}
		state.scan(ln.text)
	}

	var endOffset int
	if endIdx == -1 {
		endOffset = len(content)
	} else {
		endOffset = lines[endIdx].offset
	}
	backoff := endIdx
	if backoff == -1 {
		backoff = len(lines)
	}
	for backoff-1 > headerIdx && isTOMLBlankOrCommentLine(lines[backoff-1].text) {
		backoff--
		endOffset = lines[backoff].offset
	}
	return start, endOffset, true
}

// tomlLine is one line of a TOML file's text (excluding its terminating
// "\n", but including any trailing "\r" so a CRLF file's carriage return
// never splits from its line) together with its byte offset in the
// original content.
type tomlLine struct {
	text   string
	offset int
}

// splitTOMLLines splits content into lines by "\n" only, recording each
// line's starting byte offset. Unlike strings.Split, a trailing "\n" at
// EOF produces no phantom empty final line — the empty element after a
// final newline is not a line (D-07 interfaces note) — so back-off
// scanning from EOF never mistakes "nothing after the last newline" for
// an extra blank line.
func splitTOMLLines(content string) []tomlLine {
	var lines []tomlLine
	offset := 0
	for {
		idx := strings.IndexByte(content[offset:], '\n')
		if idx == -1 {
			if offset < len(content) {
				lines = append(lines, tomlLine{text: content[offset:], offset: offset})
			}
			return lines
		}
		lines = append(lines, tomlLine{text: content[offset : offset+idx], offset: offset})
		offset += idx + 1
	}
}

// isTOMLHeaderLine reports whether line, after stripping only leading
// spaces/tabs, begins with "[" — the one predicate that recognizes a
// "[x]" or "[[x]]" table header at any indentation (D-07). This function
// is the file's only caller of the leading-whitespace trim it uses here.
func isTOMLHeaderLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "[")
}

// isTOMLBlankOrCommentLine reports whether line, trimmed of surrounding
// whitespace, is empty or starts with "#" — used to back a table's range
// end off before the contiguous run of blank/comment lines that precede
// the next header, so that run stays with what follows (D-07).
func isTOMLBlankOrCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

// stripTOMLTrailingComment returns line with any trailing "# comment"
// removed, honoring single-line basic ("...") and literal ('...') string
// boundaries so a '#' inside a quoted string is never mistaken for a
// comment start. Only single-quote forms are handled — a header line
// never carries a multi-line triple-quoted (basic or literal) string.
func stripTOMLTrailingComment(line string) string {
	i := 0
	for i < len(line) {
		switch line[i] {
		case '#':
			return line[:i]
		case '"':
			i = skipTOMLBasicString(line, i)
		case '\'':
			i = skipTOMLLiteralString(line, i)
		default:
			i++
		}
	}
	return line
}

// skipTOMLBasicString returns the index just past the closing '"' of the
// single-line basic string starting at line[start] (line[start] == '"'),
// honoring backslash escapes. If no closing quote is found, it returns
// len(line).
func skipTOMLBasicString(line string, start int) int {
	i := start + 1
	for i < len(line) {
		if line[i] == '\\' && i+1 < len(line) {
			i += 2
			continue
		}
		if line[i] == '"' {
			return i + 1
		}
		i++
	}
	return i
}

// skipTOMLLiteralString returns the index just past the closing quote of
// the single-line literal string starting at line[start] (line[start] is
// a literal-string quote); literal strings have no escapes. If no
// closing quote is found, it returns len(line).
func skipTOMLLiteralString(line string, start int) int {
	i := start + 1
	for i < len(line) && line[i] != '\'' {
		i++
	}
	if i < len(line) {
		return i + 1
	}
	return i
}

// tomlHeaderPath returns the dotted table path named by a header line
// (line must satisfy isTOMLHeaderLine): any trailing "# comment" and
// surrounding whitespace are removed, then one layer of "[" "]" is
// unwrapped for a plain header and two layers for an array-of-tables
// "[[...]]" header, so both forms compare on the same dotted text.
func tomlHeaderPath(line string) string {
	trimmed := strings.TrimSpace(stripTOMLTrailingComment(line))
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	return strings.TrimSpace(trimmed)
}

// tomlLineState tracks, across a run of lines, whether the scanner is
// inside a multi-line basic (triple-double-quoted) or literal
// (triple-single-quoted) string, and the current nesting depth of
// unfinished "[" "{" values (a multi-line array or inline table spanning
// several lines). A line may only be treated as a header when the state
// is neutral on entry to that line (D-07): a "[" or "]" inside either
// construct is never a header.
type tomlLineState struct {
	inBasicML    bool
	inLiteralML  bool
	bracketDepth int
}

// neutral reports whether state carries no open multi-line string and no
// unfinished array/inline-table nesting.
func (s tomlLineState) neutral() bool {
	return !s.inBasicML && !s.inLiteralML && s.bracketDepth == 0
}

// scan advances state past line's characters, honoring single- and
// multi-line basic/literal string boundaries and counting "[" "{" / "]"
// "}" outside any string and before a "#" comment. It is called exactly
// once per line, in order, so multi-line string/array state carries
// correctly from one line to the next.
func (s *tomlLineState) scan(line string) {
	i := 0
	n := len(line)
	for i < n {
		if s.inBasicML {
			if idx := strings.Index(line[i:], `"""`); idx >= 0 {
				i += idx + 3
				s.inBasicML = false
				continue
			}
			return
		}
		if s.inLiteralML {
			if idx := strings.Index(line[i:], `'''`); idx >= 0 {
				i += idx + 3
				s.inLiteralML = false
				continue
			}
			return
		}
		switch {
		case line[i] == '#':
			return
		case strings.HasPrefix(line[i:], `"""`):
			if idx := strings.Index(line[i+3:], `"""`); idx >= 0 {
				i += 3 + idx + 3
				continue
			}
			s.inBasicML = true
			return
		case strings.HasPrefix(line[i:], `'''`):
			if idx := strings.Index(line[i+3:], `'''`); idx >= 0 {
				i += 3 + idx + 3
				continue
			}
			s.inLiteralML = true
			return
		case line[i] == '"':
			i = skipTOMLBasicString(line, i)
		case line[i] == '\'':
			i = skipTOMLLiteralString(line, i)
		case line[i] == '[' || line[i] == '{':
			s.bracketDepth++
			i++
		case line[i] == ']' || line[i] == '}':
			if s.bracketDepth > 0 {
				s.bracketDepth--
			}
			i++
		default:
			i++
		}
	}
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
