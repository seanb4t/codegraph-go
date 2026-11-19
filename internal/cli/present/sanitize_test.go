package present

import "testing"

func TestSanitizeControl(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"clean ascii unchanged", "main.go", "main.go"},
		{"clean unicode unchanged", "café/münchen.txt", "café/münchen.txt"},
		{"empty", "", ""},
		{"strips ESC (ANSI SGR injection)", "a\x1b[31mred\x1b[0m", "a[31mred[0m"},
		{"strips OSC introducer, keeps printable payload", "file\x1b]0;pwned\x07.go", "file]0;pwned.go"},
		{"strips embedded newline", "line1\nline2", "line1line2"},
		{"strips CR, keeps tab (CR-02: a bare tab cannot start an escape sequence)", "a\tb\rc", "a\tbc"},
		{"keeps leading tab indentation", "\treturn helper()", "\treturn helper()"},
		{"strips DEL", "a\x7fb\u009bc", "abc"},
		{"escapes the bidi override", "gpj.\u202Eexe", "gpj.<U+202E>exe"},
		{"escapes the bidi isolates", "a\u2066b\u2067c\u2068d\u2069e", "a<U+2066>b<U+2067>c<U+2068>d<U+2069>e"},
		{"escapes zero-width and direction marks", "a\u200Bb\u200Fc\u200Ed", "a<U+200B>b<U+200F>c<U+200E>d"},
		{"escapes line and paragraph separators", "a\u2028b\u2029c", "a<U+2028>b<U+2029>c"},
		{"keeps ZWJ in an emoji sequence", "\U0001F468\u200D\U0001F469", "\U0001F468\u200D\U0001F469"},
		{"keeps ZWNJ", "a\u200Cb", "a\u200Cb"},
		{"keeps tag characters", "\U0001F3F4\U000E0067\U000E007F", "\U0001F3F4\U000E0067\U000E007F"},
		{"keeps no-break space and em space", "a\u00A0b\u2003c", "a\u00A0b\u2003c"},
		{"keeps a combining mark", "e\u0301", "e\u0301"},
		{"replaces an invalid byte, so a raw 0x9B cannot reach the writer", "a\x9b8mb", "a\uFFFD8mb"},
		{"drops a control rune and escapes a format rune in one string", "\x1b\u202E", "<U+202E>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeControl(tt.in); got != tt.want {
				t.Errorf("sanitizeControl(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestSanitizeControl_CleanStringIdentity guards the fast path: a string with
// no control characters must be returned unchanged so normal filenames are
// never mutated (protects the RenderFiles output for ordinary trees).
func TestSanitizeControl_CleanStringIdentity(t *testing.T) {
	const clean = "internal/cli/present/files.go"
	if got := sanitizeControl(clean); got != clean {
		t.Fatalf("clean string was altered: got %q, want %q", got, clean)
	}
}

func TestEscapeRune(t *testing.T) {
	for _, r := range []rune{0x200B, 0x200E, 0x200F, 0x202A, 0x202E, 0x2060, 0x2066, 0x2069, 0x00AD, 0xFEFF, 0x2028, 0x2029, 0x061C} {
		if !escapeRune(r) {
			t.Errorf("escapeRune(%U) = false, want true", r)
		}
	}
	for _, r := range []rune{'a', ' ', '/', 'é', '世', 0x00A0, 0x2003, 0xE000, 0x0301, 0x200C, 0x200D, 0xE0041, 0x1F600} {
		if escapeRune(r) {
			t.Errorf("escapeRune(%U) = true, want false", r)
		}
	}
}

func TestDropRune(t *testing.T) {
	for _, r := range []rune{0x00, 0x09, 0x0A, 0x0D, 0x1b, 0x7f, 0x85, 0x9b} {
		if !dropRune(r) {
			t.Errorf("dropRune(%U) = false, want true", r)
		}
	}
	for _, r := range []rune{'a', ' ', 0x00A0, 0x202E} {
		if dropRune(r) {
			t.Errorf("dropRune(%U) = true, want false", r)
		}
	}
}

func TestSanitizeDisplay(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"clean path unchanged", "/repo/café/世界/main.go", "/repo/café/世界/main.go"},
		{"path with a space unchanged", "/a b", "/a b"},
		{"drops a tab, unlike sanitizeControl", "/a\tb", "/ab"},
		{"drops ESC and newline", "/a\x1b[31m/b\nc", "/a[31m/bc"},
		{"escapes the bidi override", "/repo/gpj.\u202Eexe", "/repo/gpj.<U+202E>exe"},
		{"shows a zero-width space, so two paths differ", "/w/pro\u200Bject", "/w/pro<U+200B>ject"},
		{"keeps ZWJ", "/a\u200Db", "/a\u200Db"},
		{"replaces an invalid byte", "/a\x9b8m", "/a\uFFFD8m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeDisplay(tt.in); got != tt.want {
				t.Errorf("SanitizeDisplay(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
