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
		{"a dropped rune does not unlock a ZWJ", "pro\u0085\u200Dject", "pro<U+200D>ject"},
		{"escapes a selector before a non-keycap after a digit", "v1\uFE0F.go", "v1<U+FE0F>.go"},
		{"escapes a selector and a joiner after a digit", "v1\uFE0F\u200D.go", "v1<U+FE0F><U+200D>.go"},
		{"keeps a keycap sequence for a digit", "1\uFE0F\u20E3", "1\uFE0F\u20E3"},
		{"keeps a keycap sequence for a hash", "#\uFE0F\u20E3", "#\uFE0F\u20E3"},
		{"keeps a keycap sequence for a star", "*\uFE0F\u20E3", "*\uFE0F\u20E3"},
		{"a keycap does not unlock a ZWJ", "1\uFE0F\u20E3\u200D", "1\uFE0F\u20E3<U+200D>"},
		{"a mark on an ASCII base does not unlock a selector and a joiner", "a\u0307\uFE0F\u200D", "a\u0307<U+FE0F><U+200D>"},
		{"keeps a Devanagari conjunct", "क\u094D\u200Dष", "क\u094D\u200Dष"},
		{"keeps a Sinhala conjunct", "ක\u0DCA\u200Dෂ", "ක\u0DCA\u200Dෂ"},
		{"keeps a combining mark on a letter", "a\u0307", "a\u0307"},
		{"escapes a presentation selector after ASCII", "main\uFE0F.go", "main<U+FE0F>.go"},
		{"escapes a presentation selector at the start", "\uFE0Fmain.go", "<U+FE0F>main.go"},
		{"escapes a selector then a ZWJ after ASCII", "pro\uFE0F\u200Dject", "pro<U+FE0F><U+200D>ject"},
		{"escapes a run of selector and joiners after ASCII", "a\uFE0F\u200D\u200Cb", "a<U+FE0F><U+200D><U+200C>b"},
		{"escapes a second joiner in a row", "\U0001F468\u200D\u200D\U0001F469", "\U0001F468\u200D<U+200D>\U0001F469"},
		{"escapes a second selector in a row", "❤\uFE0F\uFE0F", "❤\uFE0F<U+FE0F>"},
		{"escapes a selector after a joiner", "\U0001F468\u200D\uFE0F", "\U0001F468\u200D<U+FE0F>"},
		{"keeps a keycap sequence", "1\uFE0F\u20E3", "1\uFE0F\u20E3"},
		{"keeps an emoji ZWJ sequence with selectors", "\U0001F441\uFE0F\u200D\U0001F5E8\uFE0F", "\U0001F441\uFE0F\u200D\U0001F5E8\uFE0F"},
		{"keeps a ZWJ emoji after a dropped control rune", "\x1b\U0001F468\u200D\U0001F469", "\U0001F468\u200D\U0001F469"},
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
		{"sanitizeControl keycap", sanitizeControl, "1" + string(rune(0xFE0F)) + string(rune(0x20E3))},
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
		name       string
		r          rune
		prev, base rune
		next       rune
		want       bool
	}{
		{"format rune", 0x202E, 'a', 'a', 0, true},
		{"line separator", 0x2028, 'a', 'a', 0, true},
		{"variation selector", 0xE0100, 'a', 'a', 0, true},
		{"other default-ignorable rune", 0x034F, 'a', 'a', 0, true},
		{"tag character", 0xE0041, 'a', 'a', 0, true},
		{"ZWJ after ASCII", 0x200D, 'a', 'a', 0, true},
		{"ZWJ at the start", 0x200D, 0, 0, 0, true},
		{"ZWNJ after ASCII", 0x200C, 'a', 'a', 0, true},
		{"ZWJ after a non-ASCII rune", 0x200D, 0x1F468, 0x1F468, 0, false},
		{"ZWNJ after a non-ASCII rune", 0x200C, 0x06CC, 0x06CC, 0, false},
		{"ZWJ after a selector", 0x200D, 0xFE0F, 0x2764, 0, false},
		{"ZWJ after a joiner", 0x200D, 0x200C, 0x06CC, 0, true},
		{"ZWJ after the last rune of an escape", 0x200D, '>', '>', 0, true},
		{"ZWJ after a mark on an ASCII base", 0x200D, 0x0307, 'a', 0, true},
		{"ZWJ after a virama on a non-ASCII base", 0x200D, 0x094D, 0x0915, 0, false},
		{"ZWJ after a keycap mark", 0x200D, 0x20E3, '1', 0, true},
		{"text presentation selector after a non-ASCII rune", 0xFE0E, 0x2764, 0x2764, 0, false},
		{"emoji presentation selector after ASCII", 0xFE0F, 'a', 'a', 0, true},
		{"emoji presentation selector at the start", 0xFE0F, 0, 0, 0, true},
		{"emoji presentation selector after a non-ASCII rune", 0xFE0F, 0x2764, 0x2764, 0, false},
		{"emoji presentation selector after a selector", 0xFE0F, 0xFE0F, 0x2764, 0, true},
		{"emoji presentation selector after a joiner", 0xFE0F, 0x200D, 0x1F468, 0, true},
		{"keycap selector before the keycap mark", 0xFE0F, '1', '1', 0x20E3, false},
		{"keycap selector after # and *", 0xFE0F, '#', '#', 0x20E3, false},
		{"keycap selector without the keycap mark", 0xFE0F, '1', '1', '.', true},
		{"keycap selector at the end", 0xFE0F, '1', '1', 0, true},
		{"text presentation selector is not a keycap selector", 0xFE0E, '1', '1', 0x20E3, true},
		{"letter", 'a', 'b', 'b', 0, false},
		{"no-break space", 0x00A0, 'a', 'a', 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := runeState{prev: tt.prev, base: tt.base}
			if got := escapeRune(tt.r, st, tt.next); got != tt.want {
				t.Errorf("escapeRune(%U, prev %U, base %U, next %U) = %v, want %v", tt.r, tt.prev, tt.base, tt.next, got, tt.want)
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
		{"an escaped rune does not unlock a ZWJ", "é\u200B\u200Db", "é<U+200B><U+200D>b"},
		{"keeps a ZWJ emoji after an escaped control rune", "/a\tb\U0001F468\u200D\U0001F469", "/a<U+0009>b\U0001F468\u200D\U0001F469"},
		{"escapes a presentation selector after ASCII", "/w/main\uFE0F.go", "/w/main<U+FE0F>.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeDisplay(tt.in); got != tt.want {
				t.Errorf("SanitizeDisplay(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
