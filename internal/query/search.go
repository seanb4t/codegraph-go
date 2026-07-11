package query

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Location is the lightweight locations-only projection Search returns
// (D-06) — name/kind/filePath/startLine only, no source body/signature.
// It is exported so 03-04's callers/callees/impact commands can reuse the
// same shape (callers.json/callees.json/impact.json all wrap this exact
// field set, per RESEARCH's Code Examples).
type Location struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FilePath  string `json:"filePath"`
	StartLine int32  `json:"startLine"`
}

// lexicalTier ranks how a term matched a node's name/qualifiedName
// (D-06): exact-name beats prefix beats substring/token. Lower is
// better; lexicalTierNone means no match at all.
type lexicalTier int

const (
	lexicalTierExact lexicalTier = iota
	lexicalTierPrefix
	lexicalTierSubstring
	lexicalTierNone
)

// lexicalMatchTier returns the best (lowest) tier at which term matches
// name or qualifiedName: exact equality, then prefix, then substring.
// This is a plain case-sensitive substring/token match (D-06) — no
// embeddings, no fuzzy scoring, no FTS5/BM25 (EMBED-01 deferred).
func lexicalMatchTier(term, name, qualifiedName string) lexicalTier {
	best := lexicalTierNone
	for _, field := range []string{name, qualifiedName} {
		var t lexicalTier
		switch {
		case field == term:
			t = lexicalTierExact
		case strings.HasPrefix(field, term):
			t = lexicalTierPrefix
		case strings.Contains(field, term):
			t = lexicalTierSubstring
		default:
			t = lexicalTierNone
		}
		if t < best {
			best = t
		}
	}
	return best
}

// rankedNode pairs a matched node with the lexical tier it matched at, so
// matchNodes can sort once and both Query and Search reuse the ranking.
type rankedNode struct {
	node *schema.Node
	tier lexicalTier
}

// matchNodes scans every node once (reader.IterateNodes(), D-03 — no
// per-kind key scan, RESEARCH Pitfall 1), applies an in-memory --kind
// filter and the D-06 lexical matcher, and returns matches ranked by
// lexicalTier with a stable secondary sort by QualifiedName then Id so
// ties (and therefore --json output) are deterministic. Callers are
// responsible for validating kind/limit before calling this — matchNodes
// itself always scans once it is invoked.
func (e *Engine) matchNodes(term, kind string) ([]rankedNode, error) {
	it, err := e.reader.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var results []rankedNode
	for it.Next() {
		n := it.Node()
		if kind != "" && n.Kind != kind {
			continue
		}
		tier := lexicalMatchTier(term, n.Name, n.QualifiedName)
		if tier == lexicalTierNone {
			continue
		}
		results = append(results, rankedNode{node: n, tier: tier})
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].tier != results[j].tier {
			return results[i].tier < results[j].tier
		}
		if results[i].node.QualifiedName != results[j].node.QualifiedName {
			return results[i].node.QualifiedName < results[j].node.QualifiedName
		}
		return results[i].node.Id < results[j].node.Id
	})

	return results, nil
}

// Query returns full node records whose name or qualifiedName matches
// term (D-06), optionally filtered by kind and capped at limit. kind is
// validated via ValidateKind and limit via validateLimit before any scan
// runs (V5, T-03-03-Kind, T-03-03-DoS) — an unknown kind or an
// out-of-range limit returns an error without touching the store.
func (e *Engine) Query(term, kind string, limit int) ([]*schema.Node, error) {
	if err := ValidateKind(kind); err != nil {
		return nil, err
	}
	if err := validateLimit(limit); err != nil {
		return nil, err
	}

	ranked, err := e.matchNodes(term, kind)
	if err != nil {
		return nil, err
	}

	n := len(ranked)
	if limit > 0 && limit < n {
		n = limit
	}
	nodes := make([]*schema.Node, n)
	for i := 0; i < n; i++ {
		nodes[i] = ranked[i].node
	}
	return nodes, nil
}

// Search returns the same matches as Query, but projected to the
// lightweight Location shape (D-06) — no source body, no signature.
// kind/limit are validated identically to Query, before any scan.
func (e *Engine) Search(term, kind string, limit int) ([]Location, error) {
	if err := ValidateKind(kind); err != nil {
		return nil, err
	}
	if err := validateLimit(limit); err != nil {
		return nil, err
	}

	ranked, err := e.matchNodes(term, kind)
	if err != nil {
		return nil, err
	}

	n := len(ranked)
	if limit > 0 && limit < n {
		n = limit
	}
	locations := make([]Location, n)
	for i := 0; i < n; i++ {
		node := ranked[i].node
		locations[i] = Location{
			Name:      node.Name,
			Kind:      node.Kind,
			FilePath:  node.FilePath,
			StartLine: node.StartLine,
		}
	}
	return locations, nil
}

// queryNodeJSON mirrors the golden query.json per-node shape (D-05):
// camelCase keys matching the TS capture, including the three TS-only
// boolean concepts (isAsync/isStatic/isAbstract) that have no Go analog —
// rendered as literal false to keep JSON shape parity rather than
// omitting the keys. There is deliberately no "score" key: D-06 drops
// TS's FTS5/BM25 score entirely as non-deterministic/volatile.
type queryNodeJSON struct {
	ID            string  `json:"id"`
	Kind          string  `json:"kind"`
	Name          string  `json:"name"`
	QualifiedName string  `json:"qualifiedName"`
	FilePath      string  `json:"filePath"`
	Language      string  `json:"language"`
	StartLine     int32   `json:"startLine"`
	EndLine       int32   `json:"endLine"`
	StartColumn   int32   `json:"startColumn"`
	EndColumn     int32   `json:"endColumn"`
	Signature     string  `json:"signature,omitempty"`
	Docstring     string  `json:"docstring,omitempty"`
	Visibility    *string `json:"visibility"`
	IsExported    bool    `json:"isExported"`
	ReturnType    string  `json:"returnType,omitempty"`
	IsAsync       bool    `json:"isAsync"`
	IsStatic      bool    `json:"isStatic"`
	IsAbstract    bool    `json:"isAbstract"`
}

// queryNodeEnvelope is the top-level array element shape the golden
// query.json corpus uses: {"node": {...}}, not a bare node object.
type queryNodeEnvelope struct {
	Node queryNodeJSON `json:"node"`
}

// renderQueryNodeJSON converts a schema.Node into its golden-shaped JSON
// projection. Visibility is rendered as a *string so an empty (never-set)
// value marshals to JSON null, matching the golden fixture's
// "visibility": null for every captured record.
func renderQueryNodeJSON(n *schema.Node) queryNodeJSON {
	var visibility *string
	if n.Visibility != "" {
		v := n.Visibility
		visibility = &v
	}
	return queryNodeJSON{
		ID:            n.Id,
		Kind:          n.Kind,
		Name:          n.Name,
		QualifiedName: n.QualifiedName,
		FilePath:      n.FilePath,
		Language:      n.Language,
		StartLine:     n.StartLine,
		EndLine:       n.EndLine,
		StartColumn:   n.StartCol,
		EndColumn:     n.EndCol,
		Signature:     n.Signature,
		Docstring:     n.Docstring,
		Visibility:    visibility,
		IsExported:    n.IsExported,
		ReturnType:    n.ReturnType,
		IsAsync:       false,
		IsStatic:      false,
		IsAbstract:    false,
	}
}

// MarshalQueryJSON renders nodes as the golden query --json shape: a
// top-level array of {"node": {...}} envelopes (D-05). Marshaling is
// deterministic given the same input slice — encoding/json orders struct
// fields by declaration, and Query/Search already return a stably
// ranked, tie-broken slice — so two consecutive calls for the same query
// produce byte-identical output (D-06).
func MarshalQueryJSON(nodes []*schema.Node) ([]byte, error) {
	envelopes := make([]queryNodeEnvelope, len(nodes))
	for i, n := range nodes {
		envelopes[i] = queryNodeEnvelope{Node: renderQueryNodeJSON(n)}
	}
	return json.Marshal(envelopes)
}
