package cextract

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

type callRecord struct {
	Name, PkgAlias, ReceiverType string
	ArgTypes                     []string
}

// TestCallsRecordReceiverAndArgumentTypes checks the recorded receiver
// type and argument types in C++ for a call through a field, this, a
// parameter, a typed local, a pointer from `new`, a literal, a static
// call and an unknown name, and that a lowercase class and an uppercase
// local resolve by their declarations, never by their case (design.md
// D17, D18). A declared type loses its pointer star in the environment
// (goextract.TypeName), so `const char *s` gives `const char`.
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `class App {
    Logger logger;
    helper h;

    void run(const char *s, int n) {
        logger.log(s, n);
        this->close();
        Builder *b = new Builder();
        b->add(1);
        h.assist();
        Sink Upper;
        Upper.add(2.5);
        unknown.call();
        Factory::make(s);
        helper::assist();
    }
};
`
	p := newCppTestParser(t)
	res, err := Extract(p, "app", "app.cpp", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	calls := make(map[string]callRecord)
	for _, r := range res.Unresolved {
		if r.Kind == goextract.RefKindCalls {
			calls[fmt.Sprintf("%s@%d", r.Name, r.Line)] = callRecord{r.Name, r.PkgAlias, r.ReceiverType, r.ArgTypes}
		}
	}
	want := map[string]callRecord{
		"log@6":     {"log", "", "Logger", []string{"const char", "int"}},
		"close@7":   {"close", "", "App", nil},
		"add@9":     {"add", "", "Builder", []string{"int"}},
		"assist@10": {"assist", "", "helper", nil},
		"add@12":    {"add", "", "Sink", []string{"double"}},
		"call@13":   {"call", "", "unknown", nil},
		"make@14":   {"make", "", "Factory", []string{"const char"}},
		"assist@15": {"assist", "", "helper", nil},
	}
	for key, w := range want {
		got, ok := calls[key]
		if !ok {
			t.Errorf("no recorded call %s; recorded %v", key, calls)
			continue
		}
		if diff := cmp.Diff(w, got); diff != "" {
			t.Errorf("%s (-want +got):\n%s", key, diff)
		}
	}
}

// TestReferenceReturningBodiesTypeParametersThisAndLocals checks the
// body environment of a free function, an inline method and an
// out-of-line method that return a reference: parameters keep their
// types, `this` and the fields have the class type in both methods, and
// a reference local keeps its declared type.
func TestReferenceReturningBodiesTypeParametersThisAndLocals(t *testing.T) {
	src := `class Foo {
    Logger log;
    void bar();
    int &get(int i);
    int &inl(int j) { log.write(j); this->bar(); static int x; return x; }
};
int &Foo::get(int i) {
    log.write(i);
    this->bar();
    Foo &self = *this;
    self.bar();
    static int x;
    return x;
}
int &free(int k) { use(k); static int x; return x; }
`
	res, err := Extract(newCppTestParser(t), "app", "foo.cpp", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	calls := make(map[string]callRecord)
	for _, r := range res.Unresolved {
		if r.Kind == goextract.RefKindCalls {
			calls[fmt.Sprintf("%s@%d", r.Name, r.Line)] = callRecord{r.Name, r.PkgAlias, r.ReceiverType, r.ArgTypes}
		}
	}
	want := map[string]callRecord{
		"write@5": {"write", "", "Logger", []string{"int"}},
		"bar@5":   {"bar", "", "Foo", nil},
		"write@8": {"write", "", "Logger", []string{"int"}},
		"bar@9":   {"bar", "", "Foo", nil},
		"bar@11":  {"bar", "", "Foo", nil},
		"use@15":  {"use", "", "", []string{"int"}},
	}
	for key, w := range want {
		got, ok := calls[key]
		if !ok {
			t.Errorf("no recorded call %s; recorded %v", key, calls)
			continue
		}
		if diff := cmp.Diff(w, got); diff != "" {
			t.Errorf("%s (-want +got):\n%s", key, diff)
		}
	}
}
