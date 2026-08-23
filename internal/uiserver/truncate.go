package uiserver

import (
	"bytes"

	"github.com/seanb4t/codegraph-go/internal/textutil"
)

// sourceLineCap is the PRIMARY bound (D-11) on a verbatim source blob
// leaving this process: the line count a source view renders before
// truncation kicks in. It matches how a file is actually read and
// rendered — a client thinks in lines, not bytes, when deciding whether a
// source view is "too long". Chosen conservatively large enough that the
// vast majority of real source files are not visibly truncated (the
// repository's own corpora — see corpora/observations.json — carry no
// per-file line-count distribution, only aggregate node/edge counts, so
// this value is sized against common "large file" editor conventions
// rather than a measured percentile: 4096 lines comfortably exceeds all
// but the largest generated or vendored files). Referenced from its own
// declaration, truncateSource's call site, and truncate_test.go — nowhere
// else, per this package's discipline of never repeating a limit as a
// bare literal (mirroring internal/mcp/session_line.go's
// clientFieldMaxBytes).
const sourceLineCap = 4096

// sourceByteCap is the SECONDARY bound (D-11) underneath sourceLineCap: a
// pathological minified single-line file (the canonical example this
// phase's threat model names, T-01-05) blows any byte budget at ONE line,
// so the byte cap is not optional even though the line cap is primary.
// 262144 bytes (256 KiB) matches the scale editors and code hosts
// commonly use as a "this file is unusually large" threshold. Referenced
// from its own declaration, truncateSource's call site, and
// truncate_test.go — nowhere else.
const sourceByteCap = 262144

// transportSendMaxBytes is the transport backstop (D-13) on outgoing
// response size, set on the Connect handler in server.go's Listen.
// connect-go defaults to UNLIMITED on both send and receive, so RPC-05 is
// satisfied by no default — this constant and transportReadMaxBytes are
// what supplies it.
//
// The ordering to transportSendMaxBytes > sourceByteCap is load-bearing
// (D-13): a backstop at or below the application cap would silently
// convert CORRECT truncation into a resource_exhausted error the user
// sees for no reason, which is exactly what RPC-05's two-layer design
// exists to prevent (TestTransportBackstopSitsAboveApplicationCap asserts
// this inequality directly so a later edit cannot invert it silently).
//
// The headroom is sized above more than a single blob: GetNodeDetail's
// multi-definition mode can return up to uiMultiDefCap gathered
// candidates in one response, each carrying its own SourceBlob up to
// sourceByteCap — an aggregate worst case on the order of
// uiMultiDefCap*sourceByteCap (roughly 5 MiB at today's values), plus the
// surrounding message's own overhead (node metadata, calls/called-by
// lists). 16 MiB leaves comfortable headroom above that aggregate, not
// merely above one blob's cap.
const transportSendMaxBytes = 16 * 1024 * 1024

// transportReadMaxBytes is D-13's transport backstop on incoming request
// size. Every request message this service accepts (GetNodeDetailRequest,
// ExploreRequest, and the rest) is a handful of short strings and small
// integers — nowhere near sourceByteCap's scale — so 1 MiB is a generous
// bound with no relationship to the response-side caps above; it exists
// only because connect-go's own default is unlimited.
const transportReadMaxBytes = 1024 * 1024

// countLines implements RPC-05's locked line-counting rule (D-11, the
// checkpoint) in exactly one place, so the total and the returned count
// can never disagree about what "how many lines is this" means: the
// number of newline characters in b, plus one when b is non-empty and
// does not end in a newline.
//
// Worked examples, matching the checkpoint's four cases exactly:
//
//	countLines([]byte(""))     == 0  // empty input is zero lines
//	countLines([]byte("\n"))   == 1  // a lone newline is one line
//	countLines([]byte("a\nb\n")) == 2  // two newline-terminated lines
//	countLines([]byte("a\nb"))   == 2  // an unterminated final line counts
func countLines(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	n := bytes.Count(b, []byte{'\n'})
	if b[len(b)-1] != '\n' {
		n++
	}
	return n
}

// firstNLines returns the prefix of b containing exactly n lines, cutting
// immediately after the n-th newline byte. It is truncateSource's line-cap
// step: callers only invoke it when countLines(b) > n, which guarantees at
// least n newline bytes exist in b, so the returned prefix always ends in
// a newline and always satisfies countLines(result) == n.
func firstNLines(b []byte, n int) []byte {
	count := 0
	for i, c := range b {
		if c == '\n' {
			count++
			if count == n {
				return b[:i+1]
			}
		}
	}
	return b
}

// truncatedSource is truncateSource's result: the (possibly truncated)
// content plus the six values SourceBlob's wire shape carries (D-11, D-12,
// the checkpoint) — both totals AND both returned counts, in both units,
// so a client can render "showing first N of M lines" without
// recomputing anything and without guessing which cap fired.
type truncatedSource struct {
	Content       []byte
	Truncated     bool
	TotalLines    int
	TotalBytes    int
	ReturnedLines int
	ReturnedBytes int
}

// truncateSource is RPC-05's single bounding step for every verbatim
// source blob leaving this process (D-11, D-12): it counts the TRUE
// totals first, over the full input and before any cutting, because the
// totals are the client's "of M" and computing them from the truncated
// output would report the wrong number. It then applies sourceLineCap (the
// primary limit), and applies sourceByteCap to the LINE-capped result (the
// secondary limit, because a pathological minified single-line file blows
// the byte budget at one line regardless of the line cap), cutting on a
// UTF-8 rune boundary via textutil.TruncateOnRuneBoundary so a byte cut
// never splits a rune. The UTF-8 guarantee is scoped, never universal: an
// input that was valid UTF-8 stays valid; an input that was not is neither
// repaired nor rejected (D-11's checkpoint, mirrored on SourceBlob.content's
// doc comment in ui.proto).
//
// Truncation is never signalled by an error (D-12): the caller always gets
// usable content, whether or not either cap fired.
func truncateSource(b []byte) truncatedSource {
	totalLines := countLines(b)
	totalBytes := len(b)

	content := b
	truncated := false

	if totalLines > sourceLineCap {
		content = firstNLines(content, sourceLineCap)
		truncated = true
	}

	if len(content) > sourceByteCap {
		cut := textutil.TruncateOnRuneBoundary(string(content), sourceByteCap)
		content = []byte(cut)
		truncated = true
	}

	return truncatedSource{
		Content:       content,
		Truncated:     truncated,
		TotalLines:    totalLines,
		TotalBytes:    totalBytes,
		ReturnedLines: countLines(content),
		ReturnedBytes: len(content),
	}
}
