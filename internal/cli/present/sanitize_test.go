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
		{"strips CRLF line ending", "x := 1\r", "x := 1"},
		{"escapes the bidi override", "gpj.\u202Eexe", "gpj.<U+202E>exe"},
		{"escapes the bidi isolates", "a\u2066b\u2067c\u2068d\u2069e", "a<U+2066>b<U+2067>c<U+2068>d<U+2069>e"},
		{"escapes zero-width and direction marks", "a\u200Bb\u200Fc\u200Ed", "a<U+200B>b<U+200F>c<U+200E>d"},
		{"escapes line and paragraph separators", "a\u2028b\u2029c", "a<U+2028>b<U+2029>c"},
		{"escapes a tag run", "\U0001F3F4\U000E0067\U000E007F", "\U0001F3F4<U+E0067><U+E007F>"},
		{"escapes a variation selector", "a\uFE00b", "a<U+FE00>b"},
		{"escapes a default-ignorable rune", "a\u034Fb", "a<U+034F>b"},
		{"escapes ZWJ after an ASCII rune", "pro\u200Dject", "pro<U+200D>ject"},
		{"escapes ZWJ at the start", "\u200Da", "<U+200D>a"},
		{"keeps ZWJ in an emoji sequence", "\U0001F468\u200D\U0001F469", "\U0001F468\u200D\U0001F469"},
		{"keeps ZWJ after an emoji presentation selector", "❤\uFE0F\u200D\U0001F525", "❤\uFE0F\u200D\U0001F525"},
		{"keeps ZWNJ after a non-ASCII letter", "می\u200Cخ", "می\u200Cخ"},
		{"keeps the emoji presentation selector", "❤\uFE0F", "❤\uFE0F"},
		{"keeps the text presentation selector", "❤\uFE0E", "❤\uFE0E"},
		{"keeps no-break space and em space", "a\u00A0b\u2003c", "a\u00A0b\u2003c"},
		{"keeps a private-use rune", "a\uE000b", "a\uE000b"},
		{"keeps a combining mark", "e\u0301", "e\u0301"},
		{"replaces an invalid byte, so a raw 0x9B cannot reach the writer", "a\x9b8mb", "a�8mb"},
		{"replaces a run of invalid bytes with one U+FFFD", "a\x9b\x9bb", "a�b"},
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

// TestSanitizeCleanStringDoesNotAllocate guards the fast path: a string with
// nothing to change comes back as is, with no allocation.
func TestSanitizeCleanStringDoesNotAllocate(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		in   string
	}{
		{"sanitizeControl", sanitizeControl, "internal/cli/present/files.go\té\U0001F468\u200D\U0001F469"},
		{"SanitizeDisplay", SanitizeDisplay, "/repo/café/main.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if allocs := testing.AllocsPerRun(100, func() { got = tt.fn(tt.in) }); allocs != 0 {
				t.Errorf("%s allocated %v times for a clean string, want 0", tt.name, allocs)
			}
			if got != tt.in {
				t.Errorf("%s(%q) = %q, want the string unchanged", tt.name, tt.in, got)
			}
		})
	}
}

// TestEscapeRune covers the exemptions escapeRune makes and one sample of
// each class it escapes.
func TestEscapeRune(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		prev rune
		want bool
	}{
		{"format rune", 0x202E, 'a', true},
		{"line separator", 0x2028, 'a', true},
		{"variation selector", 0xE0100, 'a', true},
		{"other default-ignorable rune", 0x034F, 'a', true},
		{"tag character", 0xE0041, 'a', true},
		{"ZWJ after ASCII", 0x200D, 'a', true},
		{"ZWJ at the start", 0x200D, 0, true},
		{"ZWNJ after ASCII", 0x200C, 'a', true},
		{"ZWJ after a non-ASCII rune", 0x200D, 0x1F468, false},
		{"ZWNJ after a non-ASCII rune", 0x200C, 0x06CC, false},
		{"text presentation selector", 0xFE0E, 'a', false},
		{"emoji presentation selector", 0xFE0F, 'a', false},
		{"letter", 'a', 'b', false},
		{"no-break space", 0x00A0, 'a', false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeRune(tt.r, tt.prev); got != tt.want {
				t.Errorf("escapeRune(%U, %U) = %v, want %v", tt.r, tt.prev, got, tt.want)
			}
		})
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
		{"escapes a tab, unlike sanitizeControl", "/a\tb", "/a<U+0009>b"},
		{"escapes ESC and newline", "/a\x1b[31m/b\nc", "/a<U+001B>[31m/b<U+000A>c"},
		{"escapes an interior control rune, so two paths differ", "src/ma\x01in.go", "src/ma<U+0001>in.go"},
		{"escapes a C1 control rune", "/a\u009Bb", "/a<U+009B>b"},
		{"escapes the bidi override", "/repo/gpj.\u202Eexe", "/repo/gpj.<U+202E>exe"},
		{"shows a zero-width space, so two paths differ", "/w/pro\u200Bject", "/w/pro<U+200B>ject"},
		{"shows a ZWJ after an ASCII rune", "/w/pro\u200Dject", "/w/pro<U+200D>ject"},
		{"keeps ZWJ in an emoji sequence", "/a\U0001F468\u200D\U0001F469", "/a\U0001F468\u200D\U0001F469"},
		{"replaces an invalid byte", "/a\x9b8m", "/a�8m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeDisplay(tt.in); got != tt.want {
				t.Errorf("SanitizeDisplay(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
