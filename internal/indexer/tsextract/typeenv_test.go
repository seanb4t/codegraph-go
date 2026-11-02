package tsextract

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

type callRecord struct {
	Name, PkgAlias, ReceiverType string
	ArgTypes                     []string
	Spread                       bool
}

// TestCallsRecordReceiverAndArgumentTypes checks the recorded receiver
// type and argument types for a call through a class field, a
// constructor parameter property, a parameter, a typed local, a const
// from `new`, a literal, a spread argument, a named import and an unknown
// name, and that a lowercase class and an uppercase local resolve by
// their declarations, never by their case (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `import { Logger } from "./logger";
import * as util from "./util";

class App {
  private sink: Sink;

  constructor(private logger: Logger) {}

  run(s: string, n: number) {
    this.logger.log(s, n);
    this.sink.add("x");
    const b = new Builder();
    b.add(1);
    let h: helper = make();
    h.assist();
    const Upper = new Sink();
    Upper.add(true);
    unknown.call();
    Logger.create(s);
    util.parse(s);
    const r = util.open();
    r.close();
    util.join(...parts);
    helper.assist();
  }
}
`
	p := newTestParser(t, "ts")
	res, err := Extract(p, "app", "app.ts", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	calls := make(map[string]callRecord)
	for _, r := range res.Unresolved {
		if r.Kind == goextract.RefKindCalls {
			calls[fmt.Sprintf("%s@%d", r.Name, r.Line)] = callRecord{r.Name, r.PkgAlias, r.ReceiverType, r.ArgTypes, r.HasSpread}
		}
	}
	want := map[string]callRecord{
		"log@10":    {"log", "", "Logger", []string{"string", "number"}, false},
		"add@11":    {"add", "", "Sink", []string{"string"}, false},
		"add@13":    {"add", "", "Builder", []string{"number"}, false},
		"assist@15": {"assist", "", "helper", nil, false},
		"add@17":    {"add", "", "Sink", []string{"boolean"}, false},
		"call@18":   {"call", "", "unknown", nil, false},
		"create@19": {"create", "Logger", "Logger", []string{"string"}, false},
		"parse@20":  {"parse", "util", "util", []string{"string"}, false},
		"open@21":   {"open", "util", "util", nil, false},
		"close@22":  {"close", "", goextract.DeferredTypePrefix + "21:14", nil, false},
		"join@23":   {"join", "util", "util", []string{""}, true},
		"assist@24": {"assist", "", "helper", nil, false},
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
