package indexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// TestJavaCallsResolveInTheirSourceRoot covers two source roots that
// declare the same package, as guava/src and android/guava/src do. A call
// resolves to the declaration in the source root of the calling file,
// whatever its arity, through a static import and through a qualified
// call.
func TestJavaCallsResolveInTheirSourceRoot(t *testing.T) {
	pre := `package com.example.base;

public final class Preconditions {
  public static <T> T checkNotNull(T ref) { return ref; }
  public static <T> T checkNotNull(T ref, Object msg) { return ref; }
  public static void checkArgument(boolean ok) {}
  public static void checkArgument(boolean ok, Object msg) {}
}
`
	caller := `package com.example.base;

import static com.example.base.Preconditions.checkNotNull;

public final class Joiner {
  Joiner(String sep) {
    checkNotNull(sep);
    checkNotNull(sep, "sep");
    Preconditions.checkArgument(sep != null);
    Preconditions.checkArgument(sep != null, "sep");
  }
}
`
	root := testtree.Write(t, map[string]string{
		"guava/src/com/example/base/Preconditions.java":         pre,
		"guava/src/com/example/base/Joiner.java":                caller,
		"android/guava/src/com/example/base/Preconditions.java": pre,
		"android/guava/src/com/example/base/Joiner.java":        caller,
	})
	r, _ := runAndSnapshot(t, root)
	targets := map[string][]string{}
	it, err := r.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer it.Close()
	for it.Next() {
		e := it.Edge()
		if e.Kind != goextract.RefKindCalls {
			continue
		}
		src, err := r.GetNode(e.Source)
		if err != nil {
			t.Fatalf("GetNode(%s): %v", e.Source, err)
		}
		dst, err := r.GetNode(e.Target)
		if err != nil {
			t.Fatalf("GetNode(%s): %v", e.Target, err)
		}
		targets[src.FilePath] = append(targets[src.FilePath], fmt.Sprintf("%d %s", e.Line, dst.FilePath))
	}
	for _, tree := range []string{"guava/src", "android/guava/src"} {
		got := targets[tree+"/com/example/base/Joiner.java"]
		if len(got) != 4 {
			t.Errorf("%s: Joiner has %d calls %v, want 4", tree, len(got), got)
		}
		for _, g := range got {
			if !strings.HasSuffix(g, " "+tree+"/com/example/base/Preconditions.java") {
				t.Errorf("%s: call %s resolves outside its source root", tree, g)
			}
		}
	}
}
