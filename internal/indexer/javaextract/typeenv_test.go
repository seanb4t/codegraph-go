package javaextract

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// callRecord is what a test reads back from one recorded call.
type callRecord struct {
	Name, PkgAlias, ReceiverType string
	ArgTypes                     []string
}

func recordedCalls(t *testing.T, src string) map[string]callRecord {
	t.Helper()
	p := newTestParser(t)
	res, err := Extract(p, "com.example", "App.java", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	out := make(map[string]callRecord)
	for _, r := range res.Unresolved {
		if r.Kind != goextract.RefKindCalls {
			continue
		}
		key := r.Name
		if r.Line > 0 {
			key = r.Name + "@" + itoa(int(r.Line))
		}
		out[key] = callRecord{r.Name, r.PkgAlias, r.ReceiverType, r.ArgTypes}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// TestCallsRecordReceiverAndArgumentTypes checks the recorded receiver
// type and argument types for a call through a field, a parameter, a
// typed local, a `var` local from `new`, a literal, and an unknown name,
// and that a lowercase type and an uppercase local resolve by their
// declarations, never by their case (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `package com.example;

import java.util.List;

public class App {
    private Logger logger;
    private helper h;

    public void run(String s, int n) {
        logger.log(s, n);
        Sink sink = new Sink();
        sink.add("x");
        var b = new Builder();
        b.add(1L);
        h.assist();
        Sink Upper = new Sink();
        Upper.add(2.5);
        unknown.call();
        List.of(s);
        this.logger.log(s);
        Factory.make();
        var r = Factory.make();
        r.run();
        helper.assist();
    }
}
`
	calls := recordedCalls(t, src)
	want := map[string]callRecord{
		"log@10":    {"log", "", "Logger", []string{"String", "int"}},
		"add@12":    {"add", "", "Sink", []string{"String"}},
		"add@14":    {"add", "", "Builder", []string{"long"}},
		"assist@15": {"assist", "", "helper", nil},
		"add@17":    {"add", "", "Sink", []string{"double"}},
		"call@18":   {"call", "", "unknown", nil},
		"of@19":     {"of", "List", "List", []string{"String"}},
		"log@20":    {"log", "", "Logger", []string{"String"}},
		"make@21":   {"make", "", "Factory", nil},
		"make@22":   {"make", "", "Factory", nil},
		"run@23":    {"run", "", goextract.DeferredTypePrefix + "22:16", nil},
		"assist@24": {"assist", "", "helper", nil},
	}
	for key, w := range want {
		got, ok := calls[key]
		if !ok {
			t.Errorf("no recorded call %s; recorded %v", key, keysOf(calls))
			continue
		}
		if diff := cmp.Diff(w, got); diff != "" {
			t.Errorf("%s (-want +got):\n%s", key, diff)
		}
	}
}

func keysOf(m map[string]callRecord) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
