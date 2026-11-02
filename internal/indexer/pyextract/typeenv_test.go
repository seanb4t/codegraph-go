package pyextract

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
// type and argument types for a call through an annotated attribute, a
// parameter, an annotated local, a local from a constructor call, a
// literal, a keyword argument, a splat, an imported module and an unknown
// name, and that a lowercase class and an uppercase local resolve by
// their declarations, never by their case (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `import os
from pkg import Logger

class App:
    sink: Sink = None

    def __init__(self):
        self.logger: Logger = Logger()

    def run(self, s: str, n: int):
        self.logger.log(s, n)
        self.sink.add("x")
        b: Builder = make()
        b.add(1)
        h = helper()
        h.assist()
        Upper = Sink()
        Upper.add(2.5, flag=True)
        unknown.call()
        os.getcwd()
        Logger.create(s)
        r = Factory.make()
        r.run()
        helper.assist(*parts)
`
	p := newTestParser(t)
	res, err := Extract(p, "app", "app.py", []byte(src))
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
		"log@11":    {"log", "", "Logger", []string{"str", "int"}, nil},
		"add@12":    {"add", "", "Sink", []string{"str"}, nil},
		"add@14":    {"add", "", "Builder", []string{"int"}, nil},
		"assist@16": {"assist", "", goextract.DeferredTypePrefix + "15:12", nil, nil},
		"add@18":    {"add", "", goextract.DeferredTypePrefix + "17:16", []string{"float", "bool"}, []string{"", "flag"}},
		"call@19":   {"call", "", "unknown", nil, nil},
		"getcwd@20": {"getcwd", "os", "os", nil, nil},
		"create@21": {"create", "Logger", "Logger", []string{"str"}, nil},
		"make@22":   {"make", "", "Factory", nil, nil},
		"run@23":    {"run", "", goextract.DeferredTypePrefix + "22:12", nil, nil},
		"assist@24": {"assist", "", "helper", []string{""}, nil},
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
