package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// TestJavaDataReferences covers the Java references of the data-symbols
// contract. A field access through this reaches the field of the
// enclosing class, or of the nearest base class that declares it, and a
// field that the subclass declares shadows the base field. A private
// base field is out of reach. A bare field name reads the field through
// the implicit receiver unless a local shadows it, and that field may be
// one the class inherits from its base class or a constant of an
// interface it implements. A call of a method
// with a field's name reaches the method, and a bare call never reaches
// a field of another class. A field initializer records
// the calls it makes.
func TestJavaDataReferences(t *testing.T) {
	r, _ := runAndSnapshot(t, testtree.Write(t, map[string]string{
		"p/Base.java": `package p;

public class Base {
    protected int count = 0;
    private int secret = 1;
    public static final int LIMIT = 10;
    protected String label;
}
`,
		"p/Derived.java": `package p;

public class Derived extends Base implements Limits {
    protected String label;

    void bump() {
        this.count += 1;
        this.label = "x";
    }

    int peek() {
        return this.secret;
    }

    int bare() {
        return count;
    }

    int bareSecret() {
        return secret;
    }

    int bareShadow() {
        int count = 2;
        return count;
    }

    int max() {
        return MAX;
    }
}
`,
		"p/Limits.java": `package p;

public interface Limits {
    int MAX = 5;
}
`,
		"p/Helper.java": `package p;

public class Helper {
    static int compute() {
        return 1;
    }
}
`,
		"p/Box.java": `package p;

public class Box {
    private final int size = Helper.compute();
    private int total;

    public int size() {
        return size;
    }

    int read() {
        return total;
    }

    void write(int v) {
        total = v;
    }

    void inc() {
        total++;
    }

    int shadow() {
        int total = 3;
        return total;
    }

    int call() {
        return size();
    }

    int limit() {
        return Base.LIMIT;
    }
}

class Other {
    void poke() {
        total();
    }
}
`,
	}))
	refs := func(file, kind, src string) map[string]string {
		return dataEdges(t, r, file, kind, src, goextract.RefKindReferences)
	}
	calls := func(file, kind, src string) map[string]string {
		return dataEdges(t, r, file, kind, src, goextract.RefKindCalls)
	}
	const derived, box = "p/Derived.java", "p/Box.java"

	bump := refs(derived, goextract.KindMethod, "Derived.bump")
	if bump["field Base.count"] != "read_write" {
		t.Errorf("bump references %v, want read_write of the inherited Base.count", bump)
	}
	if bump["field Derived.label"] != "write" {
		t.Errorf("bump references %v, want a write of the shadowing Derived.label", bump)
	}
	if _, ok := bump["field Base.label"]; ok {
		t.Errorf("bump writes the shadowed Base.label: %v", bump)
	}
	if peek := refs(derived, goextract.KindMethod, "Derived.peek"); len(peek) != 0 {
		t.Errorf("peek references %v through the private Base.secret", peek)
	}
	if got := refs(derived, goextract.KindMethod, "Derived.bare")["field Base.count"]; got != "read" {
		t.Errorf("bare -> Base.count access = %q, want read of the inherited field by its bare name", got)
	}
	if s := refs(derived, goextract.KindMethod, "Derived.bareSecret"); len(s) != 0 {
		t.Errorf("bareSecret references %v through the private Base.secret", s)
	}
	if s := refs(derived, goextract.KindMethod, "Derived.bareShadow"); len(s) != 0 {
		t.Errorf("bareShadow references %v through its local count", s)
	}
	if got := refs(derived, goextract.KindMethod, "Derived.max")["field Limits.MAX"]; got != "read" {
		t.Errorf("max -> Limits.MAX access = %q, want read of the constant of the implemented interface", got)
	}

	if got := refs(box, goextract.KindMethod, "Box.size")["field Box.size"]; got != "read" {
		t.Errorf("Box.size() -> Box.size access = %q, want read through the implicit this", got)
	}
	if got := refs(box, goextract.KindMethod, "Box.read")["field Box.total"]; got != "read" {
		t.Errorf("read -> total access = %q, want read", got)
	}
	if got := refs(box, goextract.KindMethod, "Box.write")["field Box.total"]; got != "write" {
		t.Errorf("write -> total access = %q, want write", got)
	}
	if got := refs(box, goextract.KindMethod, "Box.inc")["field Box.total"]; got != "read_write" {
		t.Errorf("inc -> total access = %q, want read_write", got)
	}
	if s := refs(box, goextract.KindMethod, "Box.shadow"); len(s) != 0 {
		t.Errorf("shadow references %v through its local total", s)
	}
	c := calls(box, goextract.KindMethod, "Box.call")
	if _, ok := c["method Box.size"]; !ok || len(c) != 1 {
		t.Errorf("call calls %v, want only the method Box.size", c)
	}
	if p := refs(box, goextract.KindMethod, "Other.poke"); len(p) != 0 {
		t.Errorf("a bare call total() in Other references %v: a field is reached only through its owner", p)
	}
	if got := refs(box, goextract.KindMethod, "Box.limit")["field Base.LIMIT"]; got != "read" {
		t.Errorf("limit -> Base.LIMIT access = %q, want read", got)
	}
	if _, ok := calls(box, goextract.KindField, "Box.size")["method Helper.compute"]; !ok {
		t.Errorf("the initializer of Box.size calls %v, want Helper.compute", calls(box, goextract.KindField, "Box.size"))
	}
}
