package rubyextract

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
// type and argument types for a call through a local from `Foo.new`, an
// instance variable, a parameter, a literal, a keyword argument, a splat,
// a constant and an unknown name. Ruby writes no types, so a parameter's
// type stays unknown (design.md D16, D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `class App
  def run(s, n)
    @logger = Logger.new
    @logger.log(s, n)
    b = Builder.new
    b.add("x", 1, flag: true)
    h = helper()
    h.assist
    unknown.call
    Factory.make(s)
    self.close
    helper.assist(*parts)
  end
end
`
	p := newTestParser(t)
	res, err := Extract(p, "app", "app.rb", []byte(src))
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
		"log@4":     {"log", "", "Logger", []string{"", ""}, nil},
		"add@6":     {"add", "", "Builder", []string{"String", "Integer", "Boolean"}, []string{"", "", "flag"}},
		"assist@8":  {"assist", "", goextract.DeferredTypePrefix + "7:8", nil, nil},
		"call@9":    {"call", "", "unknown", nil, nil},
		"make@10":   {"make", "", "Factory", []string{""}, nil},
		"close@11":  {"close", "", "App", nil, nil},
		"assist@12": {"assist", "", "helper", []string{""}, nil},
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
