package agents

import (
	"errors"
	"fmt"
	"strings"
)

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
// unchanged (D-07 idempotency). CRLF content stays CRLF (tomlLineEnding).
// When tomlTableConflict reports an existing definition of tableName that
// cannot be safely edited, content is returned completely unchanged and
// the conflict error is discarded by the caller (spliceTOMLTable itself
// never returns an error — callers needing to surface the conflict call
// tomlTableConflict directly, per D-07's "refused, not duplicated" rule).
func spliceTOMLTable(content, tableName string, bodyLines []string) string {
	if tomlTableConflict(content, tableName) != nil {
		return content
	}

	ending := tomlLineEnding(content)
	var block strings.Builder
	block.WriteString("[" + tableName + "]" + ending)
	for _, line := range bodyLines {
		block.WriteString(line)
		block.WriteString(ending)
	}
	newBlock := block.String()

	start, end, found := findTOMLTableRange(content, tableName)
	if !found {
		trimmed := strings.TrimRight(content, "\r\n")
		if trimmed == "" {
			return newBlock
		}
		return trimmed + ending + ending + newBlock
	}

	if content[start:end] == newBlock {
		return content
	}
	return content[:start] + newBlock + content[end:]
}

// stripTOMLTable removes the "[tableName]" block (header through the next
// top-level "[...]" header or EOF) from content, restoring the file to its
// exact pre-splice bytes when applied after spliceTOMLTable (D-07/D-08). A
// missing table is a no-op — content is returned unchanged. CRLF content
// stays CRLF (tomlLineEnding). A conflicting definition of tableName
// (tomlTableConflict) also leaves content completely unchanged. A leading
// UTF-8 BOM with no other surviving content before or after the stripped
// table is dropped along with it (WR-03, 07-REVIEW-FIX.md pass 2) — see the
// beforeIsEmptyOrBOMOnly comment below. A BOM is still preserved byte for
// byte whenever real content survives either side (CR-01).
func stripTOMLTable(content, tableName string) string {
	if tomlTableConflict(content, tableName) != nil {
		return content
	}

	start, end, found := findTOMLTableRange(content, tableName)
	if !found {
		return content
	}

	ending := tomlLineEnding(content)
	before := strings.TrimRight(content[:start], "\r\n")
	after := strings.TrimLeft(content[end:], "\r\n")

	// A leading BOM with nothing else surviving on either side is
	// "effectively empty": once codegraph's table — the BOM'd file's only
	// real content — is gone, the BOM has no downstream reader left to
	// preserve it for, so it is dropped along with the rest rather than
	// left behind as a permanent BOM-plus-newline stub file. This keeps
	// stripTOMLTable's "" return the single "file is now empty" sentinel
	// every caller (today, only codexTarget.Uninstall) keys its
	// keep-clean removal off of, instead of every caller having to
	// separately special-case a BOM-only residual.
	beforeIsEmptyOrBOMOnly := before == "" || before == tomlBOM

	switch {
	case beforeIsEmptyOrBOMOnly && after == "":
		return ""
	case before == "":
		if !strings.HasSuffix(after, ending) {
			return after + ending
		}
		return after
	case after == "":
		return before + ending
	default:
		return before + ending + ending + after
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

// tomlBOM is the UTF-8 byte-order mark (U+FEFF, EF BB BF). A Windows-
// authored config.toml commonly opens with one (CR-01, 07-REVIEW.md).
const tomlBOM = "\xef\xbb\xbf"

// splitTOMLLines splits content into lines by "\n" only, recording each
// line's starting byte offset. Unlike strings.Split, a trailing "\n" at
// EOF produces no phantom empty final line — the empty element after a
// final newline is not a line (D-07 interfaces note) — so back-off
// scanning from EOF never mistakes "nothing after the last newline" for
// an extra blank line.
//
// When content's absolute first three bytes are a UTF-8 BOM, that BOM is
// emitted as its own synthetic zero-th "line" (offset 0, the 3 BOM bytes,
// no trailing newline) rather than folded into whatever real line follows
// it (CR-01, 07-REVIEW.md). This is the single point of entry
// findTOMLTableRange, tomlTableConflict and tomlBoolSetting all funnel
// through, so a BOM never defeats isTOMLHeaderLine's recognition of a
// header that immediately follows it — the common shape for a project-
// local Codex config.toml holding only the [mcp_servers.*] tables. The
// BOM pseudo-line is never itself eligible to be a header (isTOMLHeaderLine
// requires a "[" prefix) or to be spliced over, and every other line's
// offset is unaffected, so content[start:end] byte-slicing elsewhere in
// this file stays correct. Only content's true, absolute leading BOM is
// special-cased this way — a BOM byte sequence appearing on any later
// line is ordinary line content, never treated as leading whitespace to
// strip before recognizing a header.
func splitTOMLLines(content string) []tomlLine {
	var lines []tomlLine
	offset := 0
	if strings.HasPrefix(content, tomlBOM) {
		lines = append(lines, tomlLine{text: tomlBOM, offset: 0})
		offset = len(tomlBOM)
	}
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

// tomlLineEnding returns "\r\n" when content contains at least one CRLF
// line ending, else "\n" (D-07: a CRLF file must round-trip byte for
// byte through splice/strip).
func tomlLineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

// errTOMLTableConflict is wrapped by tomlTableConflict's returned error
// whenever content already defines tableName in a shape
// spliceTOMLTable/stripTOMLTable cannot safely edit (D-07).
var errTOMLTableConflict = errors.New("codegraph's TOML table conflicts with an existing entry")

// tomlTableConflict scans content for any existing definition of
// tableName that spliceTOMLTable/stripTOMLTable cannot safely edit: a
// quoted or spaced variant of codegraph's own header, a duplicate exact
// header, an "[[...]]" array-of-tables header naming the same path, an
// inline table or dotted key assigning into the same path from a
// different table, or one of codegraph's own subtable headers detached
// from its main header's contiguous range (D-07). On any conflict it
// returns a non-nil error wrapping errTOMLTableConflict naming the
// 1-based line; spliceTOMLTable/stripTOMLTable both leave content
// completely unchanged in that case, never producing a duplicate key.
func tomlTableConflict(content, tableName string) error {
	subtablePrefix := tableName + "."
	exactHeader := "[" + tableName + "]"
	lines := splitTOMLLines(content)

	rangeStart, rangeEnd, rangeFound := 0, 0, false
	if s, e, found := findTOMLTableRange(content, tableName); found {
		rangeStart, rangeEnd, rangeFound = s, e, true
	}

	var state tomlLineState
	currentTablePath := ""
	exactHeaderCount := 0

	for _, ln := range lines {
		if !state.neutral() {
			state.scan(ln.text)
			continue
		}

		if isTOMLHeaderLine(ln.text) {
			raw := strings.TrimSpace(stripTOMLTrailingComment(ln.text))
			isArrayHeader := strings.HasPrefix(raw, "[[") && strings.HasSuffix(raw, "]]")
			path := tomlNormalizedHeaderPath(ln.text)

			switch {
			case isArrayHeader:
				if path == tableName || strings.HasPrefix(path, subtablePrefix) {
					return tomlConflictError(lines, ln)
				}
			case path == tableName:
				exactHeaderCount++
				if raw != exactHeader || exactHeaderCount > 1 {
					return tomlConflictError(lines, ln)
				}
			case strings.HasPrefix(path, subtablePrefix):
				inOwnRange := rangeFound && ln.offset >= rangeStart && ln.offset < rangeEnd
				if !inOwnRange {
					return tomlConflictError(lines, ln)
				}
			}
			currentTablePath = path
			state.scan(ln.text)
			continue
		}

		if keyPath, ok := tomlKeyTablePath(ln.text); ok {
			effective := keyPath
			if currentTablePath != "" {
				effective = currentTablePath + "." + keyPath
			}
			ownTable := currentTablePath == tableName || strings.HasPrefix(currentTablePath, subtablePrefix)
			if !ownTable && (effective == tableName || strings.HasPrefix(effective, subtablePrefix)) {
				return tomlConflictError(lines, ln)
			}
		}
		state.scan(ln.text)
	}
	return nil
}

// tomlConflictError builds the errTOMLTableConflict-wrapping error for
// the conflicting line ln, naming its 1-based line number.
func tomlConflictError(lines []tomlLine, ln tomlLine) error {
	lineNo := 0
	for i, l := range lines {
		if l.offset == ln.offset {
			lineNo = i + 1
			break
		}
	}
	return fmt.Errorf("%w: line %d: %q", errTOMLTableConflict, lineNo, strings.TrimRight(ln.text, "\r"))
}

// tomlNormalizedHeaderPath returns the dotted table path named by a
// header line, like tomlHeaderPath, but additionally splits on "."
// outside quotes and unquotes/trims each segment (tomlSplitDottedPath),
// so a quoted (`["a"."b"]`) or spaced (`[a . b]`) header normalizes to
// the same path as its bare form `[a.b]` for conflict comparison.
func tomlNormalizedHeaderPath(line string) string {
	trimmed := strings.TrimSpace(stripTOMLTrailingComment(line))
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	return strings.Join(tomlSplitDottedPath(strings.TrimSpace(trimmed)), ".")
}

// tomlKeyTablePath reports the normalized dotted path named by the key
// side of a "key = value" line (the text before the first "=" outside
// any quoted segment), or ok=false if line carries no such unquoted "="
// (e.g. a blank/comment line or an array-continuation line).
func tomlKeyTablePath(line string) (path string, ok bool) {
	trimmed := strings.TrimSpace(stripTOMLTrailingComment(line))
	i := 0
	for i < len(trimmed) {
		switch trimmed[i] {
		case '=':
			keyText := strings.TrimSpace(trimmed[:i])
			if keyText == "" {
				return "", false
			}
			return strings.Join(tomlSplitDottedPath(keyText), "."), true
		case '"':
			i = skipTOMLBasicString(trimmed, i)
		case '\'':
			i = skipTOMLLiteralString(trimmed, i)
		default:
			i++
		}
	}
	return "", false
}

// tomlSplitDottedPath splits a dotted TOML key/table path on "." that
// occurs outside a quoted segment, trimming surrounding spaces and
// unquoting each segment (basic "…" or literal '…'), so a quoted or
// spaced variant of a path normalizes to the same segment list as its
// bare form.
func tomlSplitDottedPath(raw string) []string {
	var segments []string
	var cur strings.Builder
	i := 0
	n := len(raw)
	for i < n {
		switch raw[i] {
		case '.':
			segments = append(segments, strings.TrimSpace(cur.String()))
			cur.Reset()
			i++
		case '"':
			j := skipTOMLBasicString(raw, i)
			cur.WriteString(tomlUnquoteBasic(raw[i:j]))
			i = j
		case '\'':
			j := skipTOMLLiteralString(raw, i)
			cur.WriteString(tomlUnquoteLiteral(raw[i:j]))
			i = j
		default:
			cur.WriteByte(raw[i])
			i++
		}
	}
	segments = append(segments, strings.TrimSpace(cur.String()))
	return segments
}

// tomlUnquoteBasic strips the surrounding quotes from a single-line
// basic-string token (as returned by skipTOMLBasicString) and unescapes
// \" and \\, the two escapes tomlString ever produces.
func tomlUnquoteBasic(s string) string {
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// tomlUnquoteLiteral strips the surrounding quotes from a single-line
// literal-string token (as returned by skipTOMLLiteralString); literal
// strings have no escapes to undo.
func tomlUnquoteLiteral(s string) string {
	s = strings.TrimPrefix(s, `'`)
	return strings.TrimSuffix(s, `'`)
}

// tomlKeyValue splits a "key = value" line (after stripping a trailing
// comment) into its normalized dotted key path and raw value text — the
// same key-side parse tomlKeyTablePath performs, but also returning the
// value side, which tomlBoolSetting needs. ok is false under the same
// conditions tomlKeyTablePath reports false for (no unquoted "=", or an
// empty key).
func tomlKeyValue(line string) (keyPath, value string, ok bool) {
	trimmed := strings.TrimSpace(stripTOMLTrailingComment(line))
	i := 0
	for i < len(trimmed) {
		switch trimmed[i] {
		case '=':
			keyText := strings.TrimSpace(trimmed[:i])
			if keyText == "" {
				return "", "", false
			}
			return strings.Join(tomlSplitDottedPath(keyText), "."), strings.TrimSpace(trimmed[i+1:]), true
		case '"':
			i = skipTOMLBasicString(trimmed, i)
		case '\'':
			i = skipTOMLLiteralString(trimmed, i)
		default:
			i++
		}
	}
	return "", "", false
}

// tomlBoolLiteral reports whether s (already trimmed of surrounding
// whitespace by the caller) is exactly the bare TOML boolean literal
// "true" or "false" — any other value (a quoted string, a number, an
// array, an inline table) is reported not-a-bool, never guessed.
func tomlBoolLiteral(s string) (value, ok bool) {
	switch strings.TrimSpace(s) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

// splitTOMLInlineTableEntries splits the text between an inline table's "{"
// and "}" on "," that occurs outside a quoted segment — used only to
// recover a single "key = value" member for tomlInlineTableBoolValue below;
// it does not need to handle nested inline tables or arrays, since D-18's
// only inline-table shape is a flat `table = { key = value, ... }` on one
// line.
func splitTOMLInlineTableEntries(s string) []string {
	var entries []string
	var cur strings.Builder
	i := 0
	for i < len(s) {
		switch s[i] {
		case ',':
			entries = append(entries, cur.String())
			cur.Reset()
			i++
		case '"':
			j := skipTOMLBasicString(s, i)
			cur.WriteString(s[i:j])
			i = j
		case '\'':
			j := skipTOMLLiteralString(s, i)
			cur.WriteString(s[i:j])
			i = j
		default:
			cur.WriteByte(s[i])
			i++
		}
	}
	entries = append(entries, cur.String())
	return entries
}

// tomlInlineTableBoolValue reports the boolean value of key within raw, a
// single-line inline-table value text (e.g. "{ hooks = false }"). Returns
// ok=false when raw is not an inline table, or key is absent or not a bare
// boolean literal inside it.
func tomlInlineTableBoolValue(raw, key string) (value, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return false, false
	}
	inner := trimmed[1 : len(trimmed)-1]
	for _, part := range splitTOMLInlineTableEntries(inner) {
		kp, v, kvOK := tomlKeyValue(part)
		if !kvOK || kp != key {
			continue
		}
		return tomlBoolLiteral(v)
	}
	return false, false
}

// tomlBoolSetting reports the boolean value of table.key within content,
// recognizing three equivalent syntactic forms (D-18, used by
// codexHooksExplicitlyDisabled to read `[features] hooks` /
// `features.hooks` / `codex_hooks` from a Codex config.toml):
//
//   - a plain `key = true|false` line inside `[table]` (at any indentation,
//     any header form findTOMLTableRange itself already accepts);
//   - a root dotted `table.key = true|false` line OUTSIDE any table (Codex's
//     config.toml has no notion of "root table" beyond top-of-file, so this
//     form is only recognized before the first `[...]` header is seen);
//   - a single-line inline `table = { ... key = true|false ... }` value.
//
// Any other shape — a non-boolean literal, the key inside an unrelated
// table, or text inside a multi-line string/array (tomlLineState) — is
// reported unset (value=false, set=false), never guessed. The last matching
// occurrence in content wins, mirroring TOML's own "last key assignment
// wins" semantics for a file that (invalidly, but not this function's job to
// reject) repeats a key.
func tomlBoolSetting(content, table, key string) (value, set bool) {
	lines := splitTOMLLines(content)
	var state tomlLineState
	currentTablePath := ""

	for _, ln := range lines {
		if !state.neutral() {
			state.scan(ln.text)
			continue
		}

		if isTOMLHeaderLine(ln.text) {
			currentTablePath = tomlNormalizedHeaderPath(ln.text)
			state.scan(ln.text)
			continue
		}

		keyPath, kv, ok := tomlKeyValue(ln.text)
		if ok {
			switch {
			case currentTablePath == table && keyPath == key:
				if b, isBool := tomlBoolLiteral(kv); isBool {
					value, set = b, true
				}
			case currentTablePath == "" && keyPath == table+"."+key:
				if b, isBool := tomlBoolLiteral(kv); isBool {
					value, set = b, true
				}
			case currentTablePath == "" && keyPath == table:
				if b, isBool := tomlInlineTableBoolValue(kv, key); isBool {
					value, set = b, true
				}
			}
		}

		state.scan(ln.text)
	}
	return value, set
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
