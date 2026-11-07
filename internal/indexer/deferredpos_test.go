package indexer

import (
	"strconv"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestParseDeferredPositionRoundTrips parses each position in the form
// that goextract.DeferredType writes back to its line and column.
func TestParseDeferredPositionRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		line := rapid.Int32Min(0).Draw(t, "line")
		col := rapid.Int32Min(0).Draw(t, "col")
		typ := goextract.DeferredTypePrefix + strconv.Itoa(int(line)) + ":" + strconv.Itoa(int(col))
		gotLine, gotCol, ok := parseDeferredPosition(typ)
		if !ok || gotLine != line || gotCol != col {
			t.Fatalf("parseDeferredPosition(%q) = %d, %d, %v; want %d, %d, true", typ, gotLine, gotCol, ok, line, col)
		}
	})
}

// TestParseDeferredPositionRejectsOtherText refuses text that
// goextract.DeferredType never writes, so such a type finds no call.
func TestParseDeferredPositionRejectsOtherText(t *testing.T) {
	p := goextract.DeferredTypePrefix
	for _, typ := range []string{
		"12:3",             // no prefix
		p,                  // no position
		p + "12",           // no colon
		p + "12:",          // no column
		p + ":3",           // no line
		p + "+12:3",        // a sign
		p + "12:-3",        // a negative column
		p + "012:3",        // a leading zero
		p + "12:03",        // a leading zero in the column
		p + "1-2",          // another separator
		p + "2147483648:3", // above the 32-bit range
		p + "12:3:4",       // a third part
	} {
		if line, col, ok := parseDeferredPosition(typ); ok {
			t.Errorf("parseDeferredPosition(%q) = %d, %d, true; want false", typ, line, col)
		}
	}
}
