package present

import (
	"io"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// Line writes one styled, sanitized line (D-08): pal.Style(role).Render(
// sanitizeControl(text)) followed by a single trailing newline. text is
// sanitized in full before styling — Line never trusts caller-supplied
// text to already be free of embedded ANSI/OSC escapes or newlines
// (CR-01); an empty text renders to a bare "\n", matching
// fmt.Fprintln(out, "").
func Line(w io.Writer, pal Palette, role Role, text string) error {
	_, err := io.WriteString(w, pal.Style(role).Render(sanitizeControl(text))+"\n")
	return err
}

// Lines calls Line once per entry in texts, in order. Zero texts writes
// nothing.
func Lines(w io.Writer, pal Palette, role Role, texts ...string) error {
	for _, t := range texts {
		if err := Line(w, pal, role, t); err != nil {
			return err
		}
	}
	return nil
}

// KV writes one "<label> <value>\n" row: pal.Label.Render(label) + " " +
// pal.Value.Render(sanitizeControl(value)) + "\n" (D-08) — label itself is
// never sanitized (callers pass a fixed literal, never user data, exactly
// like present/status.go's writeStatLine convention).
func KV(w io.Writer, pal Palette, label, value string) error {
	_, err := io.WriteString(w, pal.Label.Render(label)+" "+pal.Value.Render(sanitizeControl(value))+"\n")
	return err
}

// lineWriter is NewLineWriter's implementation.
type lineWriter struct {
	w     io.Writer
	style lipgloss.Style
}

// NewLineWriter returns an io.Writer that styles each complete
// "\n"-terminated line written to it in role, sanitizing control runes
// first exactly like Line. A trailing, non-newline-terminated partial
// segment from any given Write is sanitized but passed through UNSTYLED
// and is never buffered across calls (D-08) — used by later plans for
// stderr banners and pass-through writers that control write granularity
// themselves.
func NewLineWriter(w io.Writer, pal Palette, role Role) io.Writer {
	return &lineWriter{w: w, style: pal.Style(role)}
}

// Write implements io.Writer: p is split on "\n"; every complete segment
// (all but the last) is sanitized, styled, and written with a trailing
// "\n"; the final segment — the partial tail, possibly empty — is
// sanitized and written verbatim, unstyled, with no buffering across
// calls (D-08). An empty p writes nothing and returns (0, nil).
//
// On a partial failure (an inner io.WriteString for one segment errors
// after earlier segments already wrote successfully), n reflects the
// number of bytes of p actually consumed by the segments written so far
// — never 0 — per io.Writer's general contract that n should account for
// what was really written even when n < len(p) (WR-02, 04-REVIEW.md). A
// caller that retries a short write from byte 0 would otherwise duplicate
// the already-emitted lines.
func (lw *lineWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	segs := strings.Split(string(p), "\n")
	last := len(segs) - 1
	n := 0
	for i, seg := range segs {
		clean := sanitizeControl(seg)
		if i < last {
			if _, err := io.WriteString(lw.w, lw.style.Render(clean)+"\n"); err != nil {
				return n, err
			}
			n += len(seg) + 1 // the segment plus the "\n" separator consumed from p
			continue
		}
		if clean == "" {
			n += len(seg)
			continue
		}
		if _, err := io.WriteString(lw.w, clean); err != nil {
			return n, err
		}
		n += len(seg)
	}

	return n, nil
}
