package csharpextract

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
// type and argument types for a call through a field, a property, a
// parameter, a typed local, a `var` local from `new`, a literal, and an
// unknown name, and that a lowercase type and an uppercase local resolve
// by their declarations, never by their case (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `namespace Logging;

public class App
{
    private ILogger logger;
    public Sink Output { get; set; }
    private helper h;

    public void Run(string s, int n)
    {
        logger.Write(s, n);
        Output.Add("x");
        Sink sink = new Sink();
        sink.Add(1L);
        var b = new Builder();
        b.Add(2.5);
        h.Assist();
        Sink Upper = new Sink();
        Upper.Add(true);
        unknown.Call();
        this.logger.Write(s);
        var r = Factory.Make();
        r.Run();
        helper.Assist();
    }
}
`
	p := newTestParser(t)
	res, err := Extract(p, "Logging", "App.cs", []byte(src))
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
		"Write@11":  {"Write", "", "ILogger", []string{"string", "int"}},
		"Add@12":    {"Add", "", "Sink", []string{"string"}},
		"Add@14":    {"Add", "", "Sink", []string{"long"}},
		"Add@16":    {"Add", "", "Builder", []string{"double"}},
		"Assist@17": {"Assist", "", "helper", nil},
		"Add@19":    {"Add", "", "Sink", []string{"bool"}},
		"Call@20":   {"Call", "", "unknown", nil},
		"Write@21":  {"Write", "", "ILogger", []string{"string"}},
		"Make@22":   {"Make", "", "Factory", nil},
		"Run@23":    {"Run", "", goextract.DeferredTypePrefix + "22:16", nil},
		"Assist@24": {"Assist", "", "helper", nil},
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
