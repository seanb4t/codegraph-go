package goextract

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type callRecord struct {
	Name, PkgAlias, ReceiverType string
	ArgTypes                     []string
	Spread                       bool
}

// TestCallsRecordReceiverAndArgumentTypes checks the recorded receiver
// type and argument types for a call through a struct field, the
// receiver, a parameter, a typed local, a short declaration from a
// composite literal, a literal, a spread argument, and an unknown name,
// and that a lowercase type and an uppercase local resolve by their
// declarations, never by their case (design.md D17, D18).
func TestCallsRecordReceiverAndArgumentTypes(t *testing.T) {
	src := `package app

import "example.com/app/util"

type parser struct{}

func (parser) Run() {}

type Server struct {
	log Logger
}

func (s *Server) Serve(name string, n int) {
	s.log.Write(name, n)
	s.Close()
	var p parser
	p.Run()
	q := &parser{}
	q.Run()
	Upper := parser{}
	Upper.Run()
	unknown.Call()
	util.Parse("x", 1.5)
	r := util.Open()
	r.Close()
	parts := []string{"a"}
	util.Join(parts...)
}
`
	p := newTestParser(t)
	res, err := Extract(p, "example.com/app", "server.go", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	calls := make(map[string]callRecord)
	for _, r := range res.Unresolved {
		if r.Kind == RefKindCalls {
			calls[fmt.Sprintf("%s@%d", r.Name, r.Line)] = callRecord{r.Name, r.PkgAlias, r.ReceiverType, r.ArgTypes, r.HasSpread}
		}
	}
	want := map[string]callRecord{
		"Write@14": {"Write", "", "Logger", []string{"string", "int"}, false},
		"Close@15": {"Close", "", "Server", nil, false},
		"Run@17":   {"Run", "", "parser", nil, false},
		"Run@19":   {"Run", "", "parser", nil, false},
		"Run@21":   {"Run", "", "parser", nil, false},
		"Call@22":  {"Call", "", "unknown", nil, false},
		"Parse@23": {"Parse", "util", "util", []string{"string", "float64"}, false},
		"Open@24":  {"Open", "util", "util", nil, false},
		"Close@25": {"Close", "", DeferredTypePrefix + "24:6", nil, false},
		"Join@27":  {"Join", "util", "util", []string{"[]string"}, true},
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

// TestShadowedAliasResolvesByPosition checks that a local that shadows an
// import alias hides it only from its declaration to the end of its
// block, and that a struct field is never a bare name: `paths.Escape()`
// names the import even when the receiver has a field `paths`.
func TestShadowedAliasResolvesByPosition(t *testing.T) {
	src := `package p

import (
	"example.com/m/markup"
	"example.com/m/paths"
)

type scope struct {
	paths string
}

func (s *scope) render() {
	markup.Resolve("x")
	if true {
		markup := s.name()
		markup.Resolve("y")
	}
	markup.Resolve("z")
	paths.Escape(s.paths)
}
`
	p := newTestParser(t)
	res, err := Extract(p, "example.com/m/p", "p/scope.go", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	got := map[int32]UnresolvedRef{}
	for _, r := range res.Unresolved {
		if r.Kind == RefKindCalls && (r.Name == "Resolve" || r.Name == "Escape") {
			got[r.Line] = r
		}
	}
	for line, want := range map[int32]string{13: "markup", 18: "markup", 19: "paths"} {
		r, ok := got[line]
		if !ok {
			t.Errorf("line %d: no call recorded", line)
			continue
		}
		if r.PkgAlias != want || r.ReceiverType != want {
			t.Errorf("line %d: alias %q type %q, want the import %q", line, r.PkgAlias, r.ReceiverType, want)
		}
	}
	if r, ok := got[16]; !ok || r.PkgAlias != "" || !strings.HasPrefix(r.ReceiverType, DeferredTypePrefix) {
		t.Errorf("line 16: alias %q type %q, want the deferred type of the local markup", r.PkgAlias, r.ReceiverType)
	}
}
