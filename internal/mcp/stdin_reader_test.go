package mcp

import (
	"bytes"
	"io"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

// readThrough feeds input through a stdinLingerReader and returns what it
// passed on to go-sdk and what it answered itself.
func readThrough(t *testing.T, input io.Reader) (passed, replies string) {
	t.Helper()
	var pending atomic.Int64
	var out bytes.Buffer
	r := newStdinLingerReader(io.NopCloser(input), &pending, &out)
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll through stdinLingerReader: %v", err)
	}
	return string(got), out.String()
}

// The valid lines in these tests are notifications. A call would leave
// pending above zero with nothing to answer it, and the EOF read would
// then wait the whole stdinLingerGrace.
func TestStdinReaderAnswersLinesGoSDKWouldEndTheSessionOn(t *testing.T) {
	first := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	second := `{"jsonrpc":"2.0","method":"notifications/roots/list_changed"}` + "\n"
	notification := `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{}}` + "\n"
	batch := `[{"jsonrpc":"2.0","id":3,"method":"ping"}]` + "\n"
	parseError := string(parseErrorReply)
	invalidRequest := string(invalidRequestReply)

	cases := []struct {
		name        string
		input       string
		wantPassed  string
		wantReplies string
	}{
		{"not-json", first + "{oops not json\n" + second, first + second, parseError},
		{"unterminated-last-line", first + "{oops", first, parseError},
		{"scalar", first + "5\n" + second, first + second, invalidRequest},
		{"object-without-version", first + `{"foo":1}` + "\n" + second, first + second, invalidRequest},
		{"empty-batch", first + "[ ]\n" + second, first + second, invalidRequest},
		{"batch-passes", batch + second, batch + second, ""},
		{"blank-line-passes", first + "\n" + second, first + "\n" + second, ""},
		{"notification-passes", notification + second, notification + second, ""},
		{"two-bad-lines", "x\n5\n" + second, second, parseError + invalidRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			passed, replies := readThrough(t, strings.NewReader(tc.input))
			if passed != tc.wantPassed {
				t.Errorf("passed to go-sdk = %q, want %q", passed, tc.wantPassed)
			}
			if replies != tc.wantReplies {
				t.Errorf("replies = %q, want %q", replies, tc.wantReplies)
			}
		})
	}
}

// repeatReader yields n copies of b, then EOF.
type repeatReader struct {
	b byte
	n int
}

func (r *repeatReader) Read(p []byte) (int, error) {
	if r.n == 0 {
		return 0, io.EOF
	}
	n := min(len(p), r.n)
	for i := range n {
		p[i] = r.b
	}
	r.n -= n
	return n, nil
}

// The reader keeps at most maxInboundLine bytes of a line, so what it
// allocates does not grow with the line. append grows the buffer by about
// 1.25 times a step, which allocates about five times maxInboundLine in
// total.
func TestStdinReaderSkipsALineOverTheLimitInBoundedMemory(t *testing.T) {
	next := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	const lineLen = 16 * maxInboundLine
	input := io.MultiReader(&repeatReader{b: 'a', n: lineLen}, strings.NewReader("\n"+next))

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	passed, replies := readThrough(t, input)
	runtime.ReadMemStats(&after)

	if passed != next {
		t.Errorf("passed to go-sdk = %q, want only the line after the long one", passed)
	}
	if replies != string(parseErrorReply) {
		t.Errorf("replies = %q, want one parse error", replies)
	}
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > lineLen/2 {
		t.Errorf("reading a %d-byte line allocated %d bytes, want at most %d", lineLen, alloc, lineLen/2)
	}
}
