package swiftextract

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

type callRecord struct {
	Name, PkgAlias, ReceiverType string
	ArgTypes                     []string
	Labels                       []string
}

// TestCallsRecordReceiverAndArgumentTypes checks the recorded receiver
// type, argument types and argument labels for a call through a
// property, a parameter, a typed local, a local from a constructor call,
// a literal and an unknown name, and that a lowercase class and an
// uppercase local resolve by their declarations, never by their case
// (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `class App {
    let logger: Logger = Logger()

    func run(s: String, n: Int) {
        logger.log(s, level: n)
        let b: Builder = make()
        b.add(1)
        let h = helper()
        h.assist()
        let Upper: Sink = Sink()
        Upper.add(2.5)
        unknown.call()
        self.logger.log(s)
        helper.assist()
    }
}
`
	p := newTestParser(t)
	res, err := Extract(p, "app", "App.swift", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	calls := make(map[string]callRecord)
	for _, r := range res.Unresolved {
		if r.Kind == goextract.RefKindCalls {
			calls[fmt.Sprintf("%s@%d", r.Name, r.Line)] = callRecord{r.Name, r.PkgAlias, r.ReceiverType, r.ArgTypes, r.ArgLabels}
		}
	}
	want := map[string]callRecord{
		"log@5":     {"log", "", "Logger", []string{"String", "Int"}, []string{"", "level"}},
		"add@7":     {"add", "", "Builder", []string{"Int"}, []string{""}},
		"assist@9":  {"assist", "", goextract.DeferredTypePrefix + "8:16", nil, nil},
		"add@11":    {"add", "", "Sink", []string{"Double"}, []string{""}},
		"call@12":   {"call", "", "unknown", nil, nil},
		"log@13":    {"log", "", "Logger", []string{"String"}, []string{""}},
		"assist@14": {"assist", "", "helper", nil, nil},
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
