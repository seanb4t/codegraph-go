package symbols

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

func cFileNode(file string) *schema.Node {
	return &schema.Node{
		Id: nodeid.NodeID(goextract.KindFile, file, file), Kind: goextract.KindFile,
		Name: file, QualifiedName: file, FilePath: file, Language: "c",
	}
}

// cDecls returns the declarations of a C file with a file node and the
// quoted includes written.
func cDecls(file string, written ...string) Declarations {
	r := goextract.FileResult{
		ImportPath: file, RelPath: file, Language: "c",
		Nodes: []goextract.ExtractedNode{{Node: cFileNode(file)}},
	}
	for _, p := range written {
		r.Unresolved = append(r.Unresolved, goextract.UnresolvedRef{Kind: goextract.RefKindImports, Name: p, Quoted: true})
	}
	return DeclarationsOf(r, "")
}

func TestIncludesOf(t *testing.T) {
	type include struct{ file, written, want string }
	cases := []struct {
		name  string
		decls []Declarations
		check []include
	}{
		{
			name:  "a header beside the file",
			decls: []Declarations{cDecls("src/main.c", "add.h"), cDecls("src/add.h")},
			check: []include{{"src/main.c", "add.h", "src/add.h"}},
		},
		{
			name:  "a path from the root",
			decls: []Declarations{cDecls("src/main.c", "util/str.h"), cDecls("util/str.h")},
			check: []include{{"src/main.c", "util/str.h", "util/str.h"}},
		},
		{
			name:  "a missing header",
			decls: []Declarations{cDecls("src/main.c", "nope.h"), cDecls("src/add.h")},
			check: []include{{"src/main.c", "nope.h", ""}},
		},
		{
			name:  "an absolute path",
			decls: []Declarations{cDecls("src/main.c", "/usr/include/foo.h"), cDecls("src/usr/include/foo.h"), cDecls("usr/include/foo.h")},
			check: []include{{"src/main.c", "/usr/include/foo.h", ""}},
		},
		{
			name:  "a sibling include directory",
			decls: []Declarations{cDecls("src/main.c", "../include/add.h", "../../x.h"), cDecls("include/add.h"), cDecls("x.h")},
			check: []include{{"src/main.c", "../include/add.h", "include/add.h"}, {"src/main.c", "../../x.h", ""}},
		},
		{
			name: "a header whose result has an error",
			decls: func() []Declarations {
				failed := cDecls("src/add.h")
				failed.Err = errors.New("parse failed")
				return []Declarations{cDecls("src/main.c", "add.h"), failed}
			}(),
			check: []include{{"src/main.c", "add.h", ""}},
		},
		{
			name:  "a header that includes itself",
			decls: []Declarations{cDecls("a.h", "a.h")},
			check: []include{{"a.h", "a.h", ""}},
		},
		{
			name:  "a root path that names the includer",
			decls: []Declarations{cDecls("src/a.h", "src/a.h")},
			check: []include{{"src/a.h", "src/a.h", ""}},
		},
		{
			name:  "a near candidate that is the includer",
			decls: []Declarations{cDecls("src/a.h", "a.h"), cDecls("a.h")},
			check: []include{{"src/a.h", "a.h", ""}},
		},
		{
			name:  "inner dot segments and a trailing slash",
			decls: []Declarations{cDecls("src/main.c", "add.h/", "add.h/.", "nodir/../add.h", "./add.h"), cDecls("src/add.h")},
			check: []include{
				{"src/main.c", "add.h/", ""}, {"src/main.c", "add.h/.", ""},
				{"src/main.c", "nodir/../add.h", ""}, {"src/main.c", "./add.h", "src/add.h"},
			},
		},
		{
			name:  "mixed leading runs",
			decls: []Declarations{cDecls("src/sub/main.c", "./../add.h", ".././add.h", "../sub/../add.h"), cDecls("src/add.h")},
			check: []include{
				{"src/sub/main.c", "./../add.h", "src/add.h"}, {"src/sub/main.c", ".././add.h", "src/add.h"},
				{"src/sub/main.c", "../sub/../add.h", ""},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder()
			b.AddResults(tc.decls)
			v := b.Freeze()
			wantOf := make(map[string][]string)
			for _, c := range tc.check {
				if got := v.IncludeTarget(c.file, c.written); got != c.want {
					t.Errorf("IncludeTarget(%q, %q) = %q, want %q", c.file, c.written, got, c.want)
				}
				if c.want != "" && !slices.Contains(wantOf[c.file], c.want) {
					wantOf[c.file] = append(wantOf[c.file], c.want)
				}
			}
			for _, c := range tc.check {
				want := wantOf[c.file]
				slices.Sort(want)
				if got := v.IncludesOf(c.file); !slices.Equal(got, want) {
					t.Errorf("IncludesOf(%q) = %q, want %q", c.file, got, want)
				}
			}
		})
	}

	t.Run("an angle and a macro include", func(t *testing.T) {
		r := goextract.FileResult{
			ImportPath: "main.c", RelPath: "main.c", Language: "c",
			Nodes: []goextract.ExtractedNode{{Node: cFileNode("main.c")}},
			Unresolved: []goextract.UnresolvedRef{
				{Kind: goextract.RefKindImports, Name: "stdio.h"},
				{Kind: goextract.RefKindImports, Name: "HEADER"},
			},
		}
		b := NewBuilder()
		b.AddResults([]Declarations{DeclarationsOf(r, ""), cDecls("stdio.h"), cDecls("HEADER")})
		if got := b.Freeze().IncludesOf("main.c"); len(got) != 0 {
			t.Errorf("IncludesOf(main.c) = %q, want none", got)
		}
	})

	t.Run("a stored include", func(t *testing.T) {
		b := NewBuilder()
		for _, f := range []string{"src/main.c", "src/add.h", "src/src/add.h"} {
			b.AddNode(cFileNode(f))
		}
		b.AddStoredInclude(&schema.Edge{
			Source: cFileNode("src/main.c").Id, Target: cFileNode("src/add.h").Id, Kind: "imports",
		})
		if got := b.Freeze().IncludesOf("src/main.c"); !slices.Equal(got, []string{"src/add.h"}) {
			t.Errorf("IncludesOf(src/main.c) = %q, want [src/add.h]", got)
		}
	})
}

// TestIncludeRules draws includes, held files and one added file, and
// checks the lookup rules and the waits of design D2.
func TestIncludeRules(t *testing.T) {
	segment := rapid.SampledFrom([]string{"a.h", "b.h", "src", "sub", ".", "..", ""})
	pool := []string{"a.h", "b.h", "src/a.h", "src/b.h", "src/sub/a.h", "sub/a.h", "src/src/a.h", "include/a.h"}
	rapid.Check(t, func(t *rapid.T) {
		file := rapid.SampledFrom([]string{"main.c", "src/main.c", "src/sub/main.c", "a.h", "src/a.h", "src/sub/a.h"}).Draw(t, "file")
		var written string
		switch rapid.IntRange(0, 3).Draw(t, "shape") {
		case 0:
			written = file[strings.LastIndex(file, "/")+1:]
		case 1:
			written = file
		default:
			segs := rapid.SliceOfN(segment, 1, 4).Draw(t, "segments")
			written = strings.Join(segs, "/")
			if rapid.Bool().Draw(t, "absolute") {
				written = "/" + written
			}
		}
		held := make(map[string]bool)
		for _, p := range rapid.SliceOfDistinct(rapid.SampledFrom(pool), rapid.ID[string]).Draw(t, "held") {
			held[p] = true
		}
		held[file] = true
		freeze := func(held map[string]bool) string {
			decls := []Declarations{cDecls(file, written)}
			for p := range held {
				if p != file {
					decls = append(decls, cDecls(p))
				}
			}
			b := NewBuilder()
			b.AddResults(decls)
			return b.Freeze().IncludeTarget(file, written)
		}
		target := freeze(held)
		if target == file {
			t.Fatalf("%q in %q links to its includer", written, file)
		}
		if target != "" && (!held[target] || strings.HasPrefix(target, "../")) {
			t.Fatalf("%q in %q links to %q, which is not a held path in the root", written, file, target)
		}
		if cands := IncludeCandidates(file, written); len(cands) > 0 && cands[0] != file && held[cands[0]] && target != cands[0] {
			t.Fatalf("%q in %q links to %q, want the held near candidate %q", written, file, target, cands[0])
		}

		added := rapid.SampledFrom(pool).Draw(t, "added")
		if held[added] {
			return
		}
		more := make(map[string]bool, len(held)+1)
		for p := range held {
			more[p] = true
		}
		more[added] = true
		if after := freeze(more); after != target && !slices.Contains(IncludeWaits(file, written, target), added) {
			t.Fatalf("adding %q moves %q in %q from %q to %q, but it is not a wait path %q",
				added, written, file, target, after, IncludeWaits(file, written, target))
		}
	})
}
