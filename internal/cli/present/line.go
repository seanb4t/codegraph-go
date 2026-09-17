package present

import "io"

// Line writes one styled, sanitized line (D-08): pal.Style(role).Render(
// sanitizeControl(text)) followed by a single trailing newline. text is
// sanitized in full before styling — Line never trusts caller-supplied
// text to already be free of embedded ANSI/OSC escapes or newlines
// (CR-01); an empty text renders to a bare "\n", matching
// fmt.Fprintln(out, "").
func Line(w io.Writer, pal Palette, role Role, text string) error {
	return nil
}

// Lines calls Line once per entry in texts, in order. Zero texts writes
// nothing.
func Lines(w io.Writer, pal Palette, role Role, texts ...string) error {
	return nil
}

// KV writes one "<label> <value>\n" row: pal.Label.Render(label) + " " +
// pal.Value.Render(sanitizeControl(value)) + "\n" (D-08) — label itself is
// never sanitized (callers pass a fixed literal, never user data, exactly
// like present/status.go's writeStatLine convention).
func KV(w io.Writer, pal Palette, label, value string) error {
	return nil
}

// NewLineWriter returns an io.Writer that styles each complete
// "\n"-terminated line written to it in role, sanitizing control runes
// first exactly like Line. A trailing, non-newline-terminated partial
// segment from any given Write is sanitized but passed through UNSTYLED
// and is never buffered across calls (D-08) — used by later plans for
// stderr banners and pass-through writers that control write granularity
// themselves.
func NewLineWriter(w io.Writer, pal Palette, role Role) io.Writer {
	return io.Discard
}
