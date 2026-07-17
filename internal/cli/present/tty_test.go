package present

import "testing"

// TestChoosePresentation covers the full (isTTY × NO_COLOR) truth matrix
// (D-04/D-05): the pretty branch renders only when isTTY is true AND
// NO_COLOR is unset/empty. This is testable without a real terminal or
// fd precisely because ChoosePresentation is pure (Pitfall 3).
func TestChoosePresentation(t *testing.T) {
	tests := []struct {
		name    string
		isTTY   bool
		noColor string
		want    bool
	}{
		{"tty, no NO_COLOR -> pretty", true, "", true},
		{"non-tty, no NO_COLOR -> plain", false, "", false},
		{"tty, NO_COLOR set -> plain", true, "1", false},
		{"non-tty, NO_COLOR set -> plain", false, "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChoosePresentation(tt.isTTY, tt.noColor); got != tt.want {
				t.Errorf("ChoosePresentation(%v, %q) = %v, want %v", tt.isTTY, tt.noColor, got, tt.want)
			}
		})
	}
}
