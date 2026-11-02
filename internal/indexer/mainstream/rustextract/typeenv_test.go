package rustextract

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
// type and argument types for a call through a struct field, self, a
// parameter, a typed let, a let from a struct expression, a literal, a
// path call and an unknown name, and that a lowercase struct and an
// uppercase local resolve by their declarations, never by their case
// (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `struct App {
    logger: Logger,
}

impl App {
    fn run(&self, s: &str, n: i32) {
        self.logger.log(s, n);
        self.close();
        let b: Builder = make();
        b.add(true);
        let h = helper { x: 1 };
        h.assist();
        let Upper = Sink::new();
        Upper.add("x");
        unknown.call();
        Factory::make(s);
        helper::assist();
    }
}
`
	p := newTestParser(t)
	res, err := Extract(p, "app", "app.rs", []byte(src))
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
		"log@7":     {"log", "", "Logger", []string{"str", "i32"}},
		"close@8":   {"close", "", "App", nil},
		"add@10":    {"add", "", "Builder", []string{"bool"}},
		"assist@12": {"assist", "", "helper", nil},
		"add@14":    {"add", "", goextract.DeferredTypePrefix + "13:20", []string{"str"}},
		"call@15":   {"call", "", "unknown", nil},
		"make@16":   {"make", "", "Factory", []string{"str"}},
		"assist@17": {"assist", "", "helper", nil},
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
