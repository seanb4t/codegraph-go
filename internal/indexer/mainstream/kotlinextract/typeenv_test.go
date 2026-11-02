package kotlinextract

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
// type and argument types for a call through a constructor property, a
// body property, a parameter, a typed local, a local from a constructor
// call, a literal, a named argument and an unknown name, and that a
// lowercase class and an uppercase local resolve by their declarations,
// never by their case (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `class App(private val logger: Logger) {
    val sink: Sink = Sink()

    fun run(s: String, n: Int) {
        logger.log(s, n)
        sink.add("x")
        val b: Builder = make()
        b.add(1L)
        val h = helper()
        h.assist()
        val Upper: Sink = Sink()
        Upper.add(2.5, flag = true)
        unknown.call()
        this.logger.log(s)
        helper.assist()
    }
}
`
	p := newTestParser(t)
	res, err := Extract(p, "app", "App.kt", []byte(src))
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
		"log@5":     {"log", "", "Logger", []string{"String", "Int"}, nil},
		"add@6":     {"add", "", "Sink", []string{"String"}, nil},
		"add@8":     {"add", "", "Builder", []string{"Long"}, nil},
		"assist@10": {"assist", "", goextract.DeferredTypePrefix + "9:16", nil, nil},
		"add@12":    {"add", "", "Sink", []string{"Double", "Boolean"}, []string{"", "flag"}},
		"call@13":   {"call", "", "unknown", nil, nil},
		"log@14":    {"log", "", "Logger", []string{"String"}, nil},
		"assist@15": {"assist", "", "helper", nil, nil},
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
