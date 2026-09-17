package present

import (
	"bytes"
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
