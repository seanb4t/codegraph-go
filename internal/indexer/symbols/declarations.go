package symbols

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Declarations is what a Builder reads from the extraction result of one
// file. A full index keeps one value for each file for the whole run, and
// it holds much less than the result itself: no reference, no import and
// no edge except the contains edges inside the file.
type Declarations struct {
	ImportPath, RelPath string

	// Language is the language of the file. The builder does not read it.
	// Route detection does, to skip the results of other languages.
	Language string

	// Err is the extraction error of the file. A file with an error
	// declares no symbol.
	Err error

	// SyntaxErrors is true when the parser recovered from a syntax error
	// in the file.
	SyntaxErrors bool

	// Nodes are the nodes of the result, in the order the file emits them.
	// The view holds these nodes, not copies.
	Nodes []*schema.Node

	MethodArity map[string]int32

	// Contains are the contains edges among the IntraEdges of the result.
	Contains []*schema.Edge

	// DefaultVisibility is the visibility that the language gives a type
	// with no access modifier, or "" for a language without one. The merge
	// of the copies of one node id reads it.
	DefaultVisibility string
}

// DeclarationsOf returns the declarations of r. It is the only place that
// lists the fields of a FileResult that a Builder reads.
func DeclarationsOf(r goextract.FileResult, defaultVisibility string) Declarations {
	d := Declarations{
		ImportPath:        r.ImportPath,
		RelPath:           r.RelPath,
		Language:          r.Language,
		Err:               r.Err,
		SyntaxErrors:      r.SyntaxErrors,
		MethodArity:       r.MethodArity,
		DefaultVisibility: defaultVisibility,
	}
	if len(r.Nodes) > 0 {
		d.Nodes = make([]*schema.Node, len(r.Nodes))
		for i, en := range r.Nodes {
			d.Nodes[i] = en.Node
		}
	}
	for _, ie := range r.IntraEdges {
		if ie.Edge.Kind == goextract.RefKindContains {
			d.Contains = append(d.Contains, ie.Edge)
		}
	}
	return d
}

// mergeCopy folds another copy of a node id into merged, the node that the
// first copy gave. Several files emit one id for a C# partial type, and
// one file emits one id twice for a route that it registers twice.
//
// The merged node takes the first visibility that differs from the default
// of its language: C# applies an access modifier on any part to the whole
// type. It is exported if any copy is. Every other field stays that of the
// first copy.
func mergeCopy(merged, other *schema.Node, defaultVisibility string) {
	if merged.Visibility == defaultVisibility && other.Visibility != defaultVisibility {
		merged.Visibility = other.Visibility
	}
	if other.IsExported {
		merged.IsExported = true
	}
}
