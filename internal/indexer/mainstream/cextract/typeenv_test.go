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
