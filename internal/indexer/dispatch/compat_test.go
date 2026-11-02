package dispatch

import "testing"

// TestCompatible_CArithmetic checks that C and C++ pass any arithmetic
// or bool argument to any arithmetic or bool parameter, narrowing
// included, while a pointer still needs its own type.
func TestCompatible_CArithmetic(t *testing.T) {
	for _, c := range []struct {
		lang, arg, param string
		want             bool
	}{
		{"c", "double", "int", true},
		{"c", "int", "bool", true},
		{"cpp", "bool", "double", true},
		{"cpp", "uint32_t", "long", true},
		{"c", "char*", "int", false},
		{"c", "int", "char*", false},
		{"java", "double", "int", false},
	} {
		if got := Compatible(c.lang, c.arg, c.param); got != c.want {
			t.Errorf("Compatible(%s, %s -> %s) = %v, want %v", c.lang, c.arg, c.param, got, c.want)
		}
	}
}

// TestIsTypeVariable checks that only a single capital letter, with an
// optional digit, reads as a type parameter.
func TestIsTypeVariable(t *testing.T) {
	for name, want := range map[string]bool{
		"T": true, "K": true, "T1": true, "V2": true,
		"TK": false, "URL": false, "ABC": false, "T12": false, "t": false, "": false,
	} {
		if got := isTypeVariable(name); got != want {
			t.Errorf("isTypeVariable(%q) = %v, want %v", name, got, want)
		}
	}
}
