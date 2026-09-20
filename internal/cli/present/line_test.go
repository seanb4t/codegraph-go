package present

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// TestLineSanitizesAndStyles pins D-08's Line contract: the FULL text is
// sanitized (every control rune dropped, including an embedded ANSI/OSC
// escape byte and any literal newline) before styling, a single trailing
// "\n" is always appended, and the real style wrapping actually emits an
// ESC byte (the positive control — a purely negative "no garbage bytes"
// assertion would pass vacuously against a renderer that emits nothing at
// all, rule 84d1gfpywd).
func TestLineSanitizesAndStyles(t *testing.T) {
	pal := NewPalette(true)

	t.Run("sanitizes controls and styles", func(t *testing.T) {
		var b bytes.Buffer
		text := "a\x1b[31mb\nc"
		if err := Line(&b, pal, RoleWarning, text); err != nil {
			t.Fatalf("Line: unexpected error: %v", err)
		}
		out := b.String()

		want := sanitizeControl(text) + "\n"
		if got := stripANSI(out); got != want {
			t.Errorf("stripANSI(out) = %q, want %q", got, want)
		}
		if !strings.Contains(out, "\x1b[") {
			t.Errorf("PositiveControl: styled output %q does not contain an ESC byte", out)
		}
	})

	t.Run("empty text writes a bare newline", func(t *testing.T) {
		var b bytes.Buffer
		if err := Line(&b, pal, RoleValue, ""); err != nil {
			t.Fatalf("Line: unexpected error: %v", err)
		}
		// lipgloss.Style.Render("") still wraps the empty content in its
		// own prefix/reset ESC codes — stripped, it strips back to a bare
		// "\n", matching fmt.Fprintln(out, "")'s plain-path byte shape.
		if got := stripANSI(b.String()); got != "\n" {
			t.Errorf("stripANSI(Line(..., \"\")) = %q, want %q (matching fmt.Fprintln(out, \"\"))", got, "\n")
		}
	})
}

// TestLinesAndKV pins Lines' per-text delegation (including the zero-texts
// no-op) and KV's "label value" composition, where only value passes
// through sanitizeControl (label is a fixed literal, never user data).
func TestLinesAndKV(t *testing.T) {
	pal := NewPalette(true)

	t.Run("Lines with zero texts writes nothing", func(t *testing.T) {
		var b bytes.Buffer
		if err := Lines(&b, pal, RoleValue); err != nil {
			t.Fatalf("Lines: unexpected error: %v", err)
		}
		if got := b.String(); got != "" {
			t.Errorf("Lines(zero texts) wrote %q, want empty", got)
		}
	})

	t.Run("Lines writes one styled line per text", func(t *testing.T) {
		var b bytes.Buffer
		if err := Lines(&b, pal, RoleValue, "x", "y"); err != nil {
			t.Fatalf("Lines: unexpected error: %v", err)
		}
		want := "x\ny\n"
		if got := stripANSI(b.String()); got != want {
			t.Errorf("stripANSI(out) = %q, want %q", got, want)
		}
	})

	t.Run("KV composes label and sanitized value", func(t *testing.T) {
		var b bytes.Buffer
		if err := KV(&b, pal, "Files:", "4"); err != nil {
			t.Fatalf("KV: unexpected error: %v", err)
		}
		want := "Files: 4\n"
		if got := stripANSI(b.String()); got != want {
			t.Errorf("stripANSI(out) = %q, want %q", got, want)
		}
	})

	t.Run("KV sanitizes only the value", func(t *testing.T) {
		var b bytes.Buffer
		if err := KV(&b, pal, "Label:", "a\x1bb"); err != nil {
			t.Fatalf("KV: unexpected error: %v", err)
		}
		want := "Label: ab\n"
		if got := stripANSI(b.String()); got != want {
			t.Errorf("stripANSI(out) = %q, want %q", got, want)
		}
	})
}

// TestLineWriterStylesEachLine pins NewLineWriter's split-on-"\n"
// contract: every complete line is sanitized and styled, the trailing
// partial tail from a Write is sanitized but written through unstyled and
// immediately (never buffered across Write calls so a later Write can
// complete it), and Write always returns (len(p), nil) on success. The
// two-Write fixture is the partial-tail probe the plan's acceptance
// criteria names explicitly.
func TestLineWriterStylesEachLine(t *testing.T) {
	pal := NewPalette(true)

	t.Run("two complete lines then a partial tail", func(t *testing.T) {
		var b bytes.Buffer
		w := NewLineWriter(&b, pal, RoleWarning)

		n, err := w.Write([]byte("one\ntwo\n"))
		if err != nil {
			t.Fatalf("first Write: unexpected error: %v", err)
		}
		if n != len("one\ntwo\n") {
			t.Errorf("first Write returned n=%d, want %d", n, len("one\ntwo\n"))
		}

		n, err = w.Write([]byte("par"))
		if err != nil {
			t.Fatalf("second Write: unexpected error: %v", err)
		}
		if n != len("par") {
			t.Errorf("second Write returned n=%d, want %d", n, len("par"))
		}

		out := b.String()
		want := "one\ntwo\npar"
		if got := stripANSI(out); got != want {
			t.Errorf("stripANSI(out) = %q, want %q", got, want)
		}
		if !strings.HasSuffix(out, "par") {
			t.Errorf("partial tail %q was not written unstyled/verbatim at the end of %q", "par", out)
		}
		if got := strings.Count(out, "\x1b["); got < 2 {
			t.Errorf("expected at least 2 styled (ESC-containing) lines, got %d in %q", got, out)
		}
	})

	t.Run("empty Write writes nothing and returns 0, nil", func(t *testing.T) {
		var b bytes.Buffer
		w := NewLineWriter(&b, pal, RoleValue)

		n, err := w.Write([]byte{})
		if err != nil {
			t.Fatalf("Write(empty): unexpected error: %v", err)
		}
		if n != 0 {
			t.Errorf("Write(empty) returned n=%d, want 0", n)
		}
		if got := b.String(); got != "" {
			t.Errorf("Write(empty) wrote %q, want empty", got)
		}
	})
}

// failAfterWriter is an io.Writer stub that lets the first `after` Write
// calls through to the wrapped writer, then fails every call after that —
// simulating an underlying pipe (e.g. stderr) breaking partway through a
// multi-line Write. It deliberately does NOT implement io.StringWriter, so
// io.WriteString(w, s) always routes through Write([]byte(s)) exactly once
// per call, matching lineWriter.Write's own io.WriteString usage.
type failAfterWriter struct {
	w     io.Writer
	after int
	calls int
}

func (f *failAfterWriter) Write(p []byte) (int, error) {
	f.calls++
	if f.calls > f.after {
		return 0, errors.New("boom: pipe broke")
	}
	return f.w.Write(p)
}

// TestLineWriterPartialFailureReturnsBytesWritten is WR-02's positive
// assertion (04-REVIEW.md): when an inner io.WriteString fails partway
// through a multi-line Write (after earlier complete lines already wrote
// successfully), Write must return the number of bytes of p actually
// consumed by those earlier lines — never 0 — per io.Writer's general
// contract (n should reflect real progress even when n < len(p) and err
// != nil). Before the fix, this always returned (0, err), which could
// make a caller that retries a short write from byte 0 duplicate the
// already-emitted "one\n" line.
func TestLineWriterPartialFailureReturnsBytesWritten(t *testing.T) {
	pal := NewPalette(true)
	var b bytes.Buffer
	// Allow exactly one successful io.WriteString (the styled "one\n"
	// line) through, then fail every subsequent write.
	fw := &failAfterWriter{w: &b, after: 1}
	w := NewLineWriter(fw, pal, RoleWarning)

	p := []byte("one\ntwo\nthree\n")
	n, err := w.Write(p)
	if err == nil {
		t.Fatalf("Write: expected an error from the second line's failed write, got nil (wrote %q)", b.String())
	}

	wantN := len("one\n") // bytes of p consumed by the one successfully-written segment
	if n != wantN {
		t.Errorf("Write returned n=%d after a partial failure, want %d (bytes of p actually consumed before the error)", n, wantN)
	}
	if n == 0 {
		t.Fatalf("Write returned n=0 on a partial failure — violates io.Writer's contract that n reflect bytes actually written")
	}
	if got := stripANSI(b.String()); got != "one\n" {
		t.Errorf("underlying writer received %q, want exactly the one successfully-written line %q", got, "one\n")
	}
}
