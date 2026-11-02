package phpextract

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
// type and argument types for a call through a typed property, a
// promoted constructor property, a parameter, a local from `new`, a
// literal, a named argument, a static call, `self::` and an unknown name,
// and that a lowercase class and an uppercase local resolve by their
// declarations, never by their case (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `<?php
namespace App;

class App {
    private Sink $sink;

    public function __construct(private Logger $logger) {}

    public function run(string $s, int $n): void {
        $this->logger->log($s, $n);
        $this->sink->add("x");
        $b = new Builder();
        $b->add(1, flag: true);
        $h = helper();
        $h->assist();
        $Upper = new Sink();
        $Upper->add(2.5);
        $unknown->call();
        Factory::make($s);
        self::helper();
        helper::assist(...$parts);
    }
}
`
	p := newTestParser(t)
	res, err := Extract(p, "App", "App.php", []byte(src))
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
		"log@10":    {"log", "", "Logger", []string{"string", "int"}, nil},
		"add@11":    {"add", "", "Sink", []string{"string"}, nil},
		"add@13":    {"add", "", "Builder", []string{"int", "bool"}, []string{"", "flag"}},
		"assist@15": {"assist", "", goextract.DeferredTypePrefix + "14:13", nil, nil},
		"add@17":    {"add", "", "Sink", []string{"float"}, nil},
		"call@18":   {"call", "<local:$unknown>", "", nil, nil},
		"make@19":   {"make", "", "Factory", []string{"string"}, nil},
		"helper@20": {"helper", "", "App", nil, nil},
		"assist@21": {"assist", "", "helper", []string{""}, nil},
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
