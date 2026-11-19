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
		{"strips the bidi override", "gpj.\u202Eexe", "gpj.exe"},
		{"strips the bidi isolates", "a\u2066b\u2067c\u2068d\u2069e", "abcde"},
		{"strips zero-width and direction marks", "a\u200Bb\u200Fc\u200Ed", "abcd"},
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

func TestUnsafeDisplayRune(t *testing.T) {
	for _, r := range []rune{0x00, 0x1b, 0x7f, 0x9b, 0x200B, 0x200F, 0x202A, 0x202E, 0x2066, 0x2069, 0xFEFF} {
		if !UnsafeDisplayRune(r) {
			t.Errorf("UnsafeDisplayRune(%U) = false, want true", r)
		}
	}
	for _, r := range []rune{'a', ' ', '/', 'é', '世', 0x1F600} {
		if UnsafeDisplayRune(r) {
			t.Errorf("UnsafeDisplayRune(%U) = true, want false", r)
		}
	}
}
