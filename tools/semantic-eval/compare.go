package semanticeval

import (
	"fmt"
	"sort"
)

// Mismatch kinds. A mismatch string starts with its kind.
const (
	MissingDeclaration    = "missing-declaration"
	ExtraDeclaration      = "extra-declaration"
	WrongSpan             = "wrong-span"
	AmbiguousDeclaration  = "ambiguous-declaration"
	MissingRelationship   = "missing-relationship"
	ExtraRelationship     = "extra-relationship"
	ForbiddenRelationship = "forbidden-relationship"
	MissingQueryFact      = "missing-query-fact"
	ForbiddenQueryFact    = "forbidden-query-fact"
	TruncatedQueryFact    = "truncated-query-fact"
	WrongExclusion        = "wrong-exclusion"
)

// Counts holds the accuracy tally of one fact kind.
type Counts struct {
	TruePositives  int `json:"truePositives"`
	FalsePositives int `json:"falsePositives"`
	FalseNegatives int `json:"falseNegatives"`
	Unknown        int `json:"unknown"`
	WrongSpans     int `json:"wrongSpans"`
}

func (c *Counts) add(o Counts) {
	c.TruePositives += o.TruePositives
	c.FalsePositives += o.FalsePositives
	c.FalseNegatives += o.FalseNegatives
	c.Unknown += o.Unknown
	c.WrongSpans += o.WrongSpans
}

// Comparison is the result of comparing one case with its observations.
type Comparison struct {
	Counts     map[string]Counts
	Mismatches []string
}

// DeclarationFacts is the fact-kind key of declaration counts.
const DeclarationFacts = "declaration"

func relationshipFacts(kind string) string { return "relationship:" + kind }

func declFact(d ExpectedDeclaration) string {
	return fmt.Sprintf("%s/%s/%s/%s@%d:%d", d.Path, d.Owner, d.Role, d.Name, d.At.Line, d.At.Col)
}

func contains(s Span, p Position) bool {
	after := s.Start.Line < p.Line || (s.Start.Line == p.Line && s.Start.Col <= p.Col)
	before := p.Line < s.End.Line || (p.Line == s.End.Line && p.Col <= s.End.Col)
	return after && before
}

// Compare matches observations with the case's expected facts one to one.
func Compare(c *Case, obs *Observed) *Comparison {
	res := &Comparison{Counts: map[string]Counts{}}
	bump := func(key string, f func(*Counts)) {
		n := res.Counts[key]
		f(&n)
		res.Counts[key] = n
	}
	mismatch := func(kind, fact string) { res.Mismatches = append(res.Mismatches, kind+" "+fact) }

	// A node matches one expected identifier, so a node that spans several
	// names can never satisfy two of them.
	used := map[string]bool{}
	declOf := map[string]string{}
	var order []ExpectedDeclaration
	order = append(order, c.Declarations...)
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i].Span, order[j].Span
		if a.Start != b.Start {
			return a.Start.Line > b.Start.Line || (a.Start.Line == b.Start.Line && a.Start.Col > b.Start.Col)
		}
		return false
	})
	for _, want := range order {
		var exact, enclosing []ObservedDeclaration
		for _, o := range obs.Declarations {
			if used[o.NodeID] || o.Path != want.Path || o.Owner != want.Owner || o.Role != want.Role || o.Name != want.Name {
				continue
			}
			if !contains(o.Span, want.At) {
				continue
			}
			enclosing = append(enclosing, o)
			if o.Span == want.Span {
				exact = append(exact, o)
			}
		}
		switch {
		case len(exact) == 1:
			used[exact[0].NodeID] = true
			declOf[exact[0].NodeID] = want.ID
			bump(DeclarationFacts, func(n *Counts) { n.TruePositives++ })
		case len(exact) > 1:
			used[exact[0].NodeID] = true
			declOf[exact[0].NodeID] = want.ID
			mismatch(AmbiguousDeclaration, declFact(want))
			bump(DeclarationFacts, func(n *Counts) { n.FalsePositives++; n.FalseNegatives++ })
		case len(enclosing) > 0:
			used[enclosing[0].NodeID] = true
			declOf[enclosing[0].NodeID] = want.ID
			o := enclosing[0].Span
			mismatch(WrongSpan, fmt.Sprintf("%s rule=%s got=%d:%d-%d:%d", declFact(want), want.SpanRule,
				o.Start.Line, o.Start.Col, o.End.Line, o.End.Col))
			bump(DeclarationFacts, func(n *Counts) { n.WrongSpans++; n.FalsePositives++; n.FalseNegatives++ })
		default:
			mismatch(MissingDeclaration, declFact(want))
			bump(DeclarationFacts, func(n *Counts) { n.FalseNegatives++ })
		}
	}

	for _, o := range obs.Declarations {
		if used[o.NodeID] || !c.HasRole(o.Role) {
			continue
		}
		if c.Labeling == LabelingSampled {
			bump(DeclarationFacts, func(n *Counts) { n.Unknown++ })
			continue
		}
		mismatch(ExtraDeclaration, fmt.Sprintf("%s/%s/%s/%s@%d:%d", o.Path, o.Owner, o.Role, o.Name, o.Span.Start.Line, o.Span.Start.Col))
		bump(DeclarationFacts, func(n *Counts) { n.FalsePositives++ })
	}

	endpoint := func(nodeID string) string {
		if id, ok := declOf[nodeID]; ok {
			return id
		}
		return nodeEndpoint(nodeID, obs)
	}
	type triple struct{ s, k, t string }
	label := func(list []ExpectedRelationship) map[triple]bool {
		m := map[triple]bool{}
		for _, r := range list {
			m[triple{r.Source, r.Kind, r.Target}] = true
		}
		return m
	}
	expected, forbidden, unknown := label(c.Relationships), label(c.Forbidden), label(c.Unknown)

	// The store keys edges by endpoints and kind, so this set is the
	// graph-level denominator. Call-site counts stay outside it.
	emitted := map[triple]bool{}
	for _, e := range obs.Relationships {
		if c.HasKind(e.Kind) {
			emitted[triple{endpoint(e.Source), e.Kind, endpoint(e.Target)}] = true
		}
	}
	for t := range expected {
		key := relationshipFacts(t.k)
		if emitted[t] {
			bump(key, func(n *Counts) { n.TruePositives++ })
			continue
		}
		mismatch(MissingRelationship, fmt.Sprintf("%s %s %s", t.s, t.k, t.t))
		bump(key, func(n *Counts) { n.FalseNegatives++ })
	}
	for t := range emitted {
		key := relationshipFacts(t.k)
		switch {
		case expected[t]:
		case forbidden[t]:
			mismatch(ForbiddenRelationship, fmt.Sprintf("%s %s %s", t.s, t.k, t.t))
			bump(key, func(n *Counts) { n.FalsePositives++ })
		case unknown[t] || c.Labeling == LabelingSampled:
			bump(key, func(n *Counts) { n.Unknown++ })
		default:
			mismatch(ExtraRelationship, fmt.Sprintf("%s %s %s", t.s, t.k, t.t))
			bump(key, func(n *Counts) { n.FalsePositives++ })
		}
	}
	sort.Strings(res.Mismatches)
	return res
}

// nodeEndpoint names a node that matched no expected declaration. A file
// node is file:<path>, the source of a module-level statement.
func nodeEndpoint(nodeID string, obs *Observed) string {
	if path, ok := obs.Files[nodeID]; ok {
		return "file:" + path
	}
	for _, d := range obs.Declarations {
		if d.NodeID == nodeID {
			return "unmatched:" + d.Path + "/" + d.Owner + "/" + d.Name
		}
	}
	return "external:" + nodeID
}
