package semanticeval

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Position encodings a reference can report.
const (
	EncodingUTF8  = "utf-8"
	EncodingUTF16 = "utf-16"
)

// Occurrence is one symbol occurrence a reference reports.
type Occurrence struct {
	Path       string `json:"path"`
	At         Position
	Symbol     string `json:"symbol"`
	Definition bool   `json:"definition"`
	Enclosing  *Span
}

// Reference is one independent source of facts.
type Reference interface {
	Name() string
	// Version returns the exact tool version. An error means the tool
	// cannot run, which is an infrastructure failure.
	Version() (string, error)
	// Establishes names the facts this reference can prove.
	Establishes() []string
	// Index returns the occurrences for the source tree in dir.
	Index(dir, scratch string, c *Case) ([]Occurrence, error)
}

type scipOccurrence struct {
	Range          []int           `json:"range"`
	EnclosingRange []int           `json:"enclosing_range"`
	TypedRange     json.RawMessage `json:"TypedRange"`
	TypedEnclosing json.RawMessage `json:"TypedEnclosingRange"`
	Symbol         string          `json:"symbol"`
	Roles          int             `json:"symbol_roles"`
}

type scipIndex struct {
	Documents []struct {
		Path        string           `json:"relative_path"`
		Occurrences []scipOccurrence `json:"occurrences"`
	} `json:"documents"`
}

// rawRange is a SCIP range as 0-based line and encoding-unit columns.
type rawRange struct{ startLine, startCol, endLine, endCol int }

func arrayRange(v []int) (rawRange, bool) {
	switch len(v) {
	case 3:
		return rawRange{v[0], v[1], v[0], v[2]}, true
	case 4:
		return rawRange{v[0], v[1], v[2], v[3]}, true
	}
	return rawRange{}, false
}

// typedRange reads the oneof wrappers `scip print --json` writes.
func typedRange(raw json.RawMessage) (rawRange, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return rawRange{}, false
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return rawRange{}, false
	}
	for name, body := range wrapper {
		var v struct {
			Line      int `json:"line"`
			StartLine int `json:"start_line"`
			EndLine   int `json:"end_line"`
			Start     int `json:"start_character"`
			End       int `json:"end_character"`
		}
		if err := json.Unmarshal(body, &v); err != nil {
			return rawRange{}, false
		}
		if strings.HasPrefix(name, "Single") {
			return rawRange{v.Line, v.Start, v.Line, v.End}, true
		}
		return rawRange{v.StartLine, v.Start, v.EndLine, v.End}, true
	}
	return rawRange{}, false
}

func occurrenceRange(o scipOccurrence) (rawRange, bool) {
	if r, ok := arrayRange(o.Range); ok {
		return r, true
	}
	return typedRange(o.TypedRange)
}

func enclosingRange(o scipOccurrence) (rawRange, bool) {
	if r, ok := arrayRange(o.EnclosingRange); ok {
		return r, true
	}
	return typedRange(o.TypedEnclosing)
}

// byteColumn converts a column in encoding to a UTF-8 byte offset.
func byteColumn(line string, col int, encoding string) int {
	if encoding == EncodingUTF8 {
		return col
	}
	units, offset := 0, 0
	for units < col && offset < len(line) {
		r, size := utf8.DecodeRuneInString(line[offset:])
		units++
		if r >= 0x10000 {
			units++
		}
		offset += size
	}
	return offset
}

// ParseSCIP converts `scip print --json` output into occurrences with
// positions in UTF-8 bytes. encoding is what the producing indexer uses.
func ParseSCIP(data []byte, dir, encoding string) ([]Occurrence, error) {
	var idx scipIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("decode scip json: %w", err)
	}
	var out []Occurrence
	for _, doc := range idx.Documents {
		raw, err := os.ReadFile(filepath.Join(dir, doc.Path))
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(raw), "\n")
		lineAt := func(n int) string {
			if n < len(lines) {
				return lines[n]
			}
			return ""
		}
		for _, o := range doc.Occurrences {
			r, ok := occurrenceRange(o)
			if !ok {
				return nil, fmt.Errorf("%s: occurrence of %s has no range", doc.Path, o.Symbol)
			}
			occ := Occurrence{
				Path:       doc.Path,
				At:         Position{Line: r.startLine + 1, Col: byteColumn(lineAt(r.startLine), r.startCol, encoding)},
				Symbol:     o.Symbol,
				Definition: o.Roles&1 != 0,
			}
			if e, ok := enclosingRange(o); ok {
				occ.Enclosing = &Span{
					Start: Position{Line: e.startLine + 1, Col: byteColumn(lineAt(e.startLine), e.startCol, encoding)},
					End:   Position{Line: e.endLine + 1, Col: byteColumn(lineAt(e.endLine), e.endCol, encoding)},
				}
			}
			out = append(out, occ)
		}
	}
	return out, nil
}

func printSCIP(path string) ([]byte, error) {
	out, err := exec.Command("scip", "print", "--json", path).Output()
	if err != nil {
		return nil, fmt.Errorf("scip print %s: %w", path, err)
	}
	return out, nil
}

func toolVersion(name string, args ...string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("required reference tool %s is not installed: %w", name, err)
	}
	out, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// Qualification is the result of checking a case against its reference.
type Qualification struct {
	// Failures name disagreements between the manifest and the reference.
	Failures []string
	// Excluded counts reference facts by the status of their exclusion.
	Excluded map[string]int
}

func isLocalSymbol(s string) bool { return strings.HasPrefix(s, "local ") }

// isScopeSymbol is true for package, namespace, file and parameter symbols,
// which are not declarations in the evaluated contract.
func isScopeSymbol(s string) bool { return strings.HasSuffix(s, "/") || strings.Contains(s, ").(") }

// Qualify checks authored facts against occurrences in both directions.
func Qualify(c *Case, occs []Occurrence) (*Qualification, error) {
	q := &Qualification{Excluded: map[string]int{}}
	type exclusion struct {
		re     *regexp.Regexp
		status string
	}
	var exclusions []exclusion
	for _, x := range c.ReferenceExclusion {
		re, err := regexp.Compile(x.Symbol)
		if err != nil {
			return nil, fmt.Errorf("reference exclusion %q: %w", x.Symbol, err)
		}
		if x.Status != StatusUnsupported && x.Status != StatusUnevaluated {
			return nil, fmt.Errorf("reference exclusion %q: status %q must be unsupported or unevaluated", x.Symbol, x.Status)
		}
		exclusions = append(exclusions, exclusion{re, x.Status})
	}

	symbolOf := map[string]string{}
	authored := map[string]bool{}
	for _, d := range c.Declarations {
		if d.Evidence == "reviewed" {
			continue
		}
		found := false
		for _, o := range occs {
			if o.Definition && o.Path == d.Path && o.At == d.At && !isLocalSymbol(o.Symbol) {
				symbolOf[d.ID] = o.Symbol
				authored[o.Symbol] = true
				found = true
			}
		}
		if !found {
			q.Failures = append(q.Failures, fmt.Sprintf("reference does not define %s at %s:%d:%d", d.ID, d.Path, d.At.Line, d.At.Col))
		}
	}

	seen := map[string]bool{}
	for _, o := range occs {
		if !o.Definition || isLocalSymbol(o.Symbol) || isScopeSymbol(o.Symbol) || authored[o.Symbol] || seen[o.Symbol] {
			continue
		}
		seen[o.Symbol] = true
		matched := false
		for _, x := range exclusions {
			if x.re.MatchString(o.Symbol) {
				q.Excluded[x.status]++
				matched = true
				break
			}
		}
		if !matched {
			q.Failures = append(q.Failures, fmt.Sprintf("reference defines unlabeled %s at %s:%d:%d", o.Symbol, o.Path, o.At.Line, o.At.Col))
		}
	}

	for _, r := range c.Relationships {
		if r.Site == nil || r.Evidence == "reviewed" {
			continue
		}
		want, ok := symbolOf[r.Target]
		if !ok {
			q.Failures = append(q.Failures, fmt.Sprintf("relationship %s %s %s has a site but the reference has no target symbol", r.Source, r.Kind, r.Target))
			continue
		}
		found := false
		for _, o := range occs {
			if o.Path == r.Path && o.At == *r.Site && o.Symbol == want {
				found = true
			}
		}
		if !found {
			q.Failures = append(q.Failures, fmt.Sprintf("reference has no occurrence of %s at %s:%d:%d", r.Target, r.Path, r.Site.Line, r.Site.Col))
		}
	}
	return q, nil
}
