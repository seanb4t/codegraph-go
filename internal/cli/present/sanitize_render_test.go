package present

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// hostileText holds a bidi override, an OSC sequence with its BEL
// terminator and a raw 0x9B byte (invalid UTF-8, a C1 CSI in some
// terminals).
const hostileText = "gpj.\u202Eexe\x1b]0;x\x07\x9b8m"

// requireSanitizedRaw checks the raw styled output, with no ANSI stripping.
// The output must be valid UTF-8 and hold no U+202E, no ESC "]" and no BEL.
// It must also hold the "<U+202E>" escape, so an empty output cannot pass.
func requireSanitizedRaw(t *testing.T, raw string) {
	t.Helper()
	if !utf8.ValidString(raw) {
		t.Errorf("output holds invalid UTF-8, such as a raw 0x9B byte: %q", raw)
	}
	for _, bad := range []string{"\u202E", "\x1b]", "\x07"} {
		if strings.Contains(raw, bad) {
			t.Errorf("output holds %q: %q", bad, raw)
		}
	}
	if !strings.Contains(raw, "<U+202E>") {
		t.Errorf("output lacks the <U+202E> escape: %q", raw)
	}
}

func TestRenderNodeSanitizesRepositoryText(t *testing.T) {
	n := &schema.Node{Id: "h", Name: hostileText, Kind: "func", FilePath: hostileText, StartLine: 1, EndLine: 1, Signature: hostileText}
	source := []byte(hostileText + "\n")

	t.Run("single definition", func(t *testing.T) {
		d := query.NodeDetail{Mode: query.NodeDetailModeSingleDef, Definition: &query.DefinitionDetail{Node: n, Source: source}}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		requireSanitizedRaw(t, buf.String())
	})

	t.Run("file source", func(t *testing.T) {
		d := query.NodeDetail{Mode: query.NodeDetailModeFile, File: &query.FileDetail{Path: hostileText, Source: source}}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		requireSanitizedRaw(t, buf.String())
	})
}

func TestRenderExploreSanitizesRepositoryText(t *testing.T) {
	n := &schema.Node{Id: "h", Name: hostileText, Kind: "func", FilePath: hostileText, StartLine: 1, EndLine: 1, Signature: hostileText}
	r := query.ExploreResult{
		Query:       hostileText,
		Groups:      []query.ExploreFileGroup{{Path: hostileText, Symbols: []*schema.Node{n}}},
		SymbolCount: 1,
		Blasts:      []query.ExploreBlast{{Symbol: n, CallerCount: 1, TestFiles: []string{hostileText}}},
		Sources:     map[string][]byte{hostileText: []byte(hostileText + "\n")},
	}

	var buf bytes.Buffer
	if err := RenderExplore(r, NewPalette(true), &buf); err != nil {
		t.Fatalf("RenderExplore: %v", err)
	}
	requireSanitizedRaw(t, buf.String())
}
