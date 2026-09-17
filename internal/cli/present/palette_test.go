package present

import (
	"reflect"
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
)

// allRoles is the fixed enumeration order every palette test walks —
// mirrors Role's own iota declaration.
var allRoles = []Role{RoleHeader, RoleLabel, RoleValue, RolePath, RoleCount, RoleWarning, RoleError}

// boldRoles/faintRoles are D-05's structural-chrome assignments: Header,
// Warning and Error are bold; Label is faint; Value/Path/Count carry
// neither.
var boldRoles = map[Role]bool{RoleHeader: true, RoleWarning: true, RoleError: true}
var faintRoles = map[Role]bool{RoleLabel: true}

// TestNewPaletteRolesFollowDark pins D-05/D-06: NewPalette(dark) selects
// each role's dark/light hex via lipgloss.LightDark, Header/Warning/Error
// are bold, Label is faint, and the seven light hexes (and, separately,
// the seven dark hexes) are pairwise distinct. This reads our own data
// back through lipgloss's own Style getters — never a render/ANSI-byte
// test (D-00: lipgloss's own encoding is never re-tested).
func TestNewPaletteRolesFollowDark(t *testing.T) {
	dark := NewPalette(true)
	light := NewPalette(false)

	for _, r := range allRoles {
		hp, ok := paletteHex[r]
		if !ok {
			t.Fatalf("paletteHex has no entry for role %v", r)
		}

		wantDark := lipgloss.Color(hp.dark)
		if got := dark.Style(r).GetForeground(); !reflect.DeepEqual(got, wantDark) {
			t.Errorf("NewPalette(true).Style(%v).GetForeground() = %#v, want %#v", r, got, wantDark)
		}

		wantLight := lipgloss.Color(hp.light)
		if got := light.Style(r).GetForeground(); !reflect.DeepEqual(got, wantLight) {
			t.Errorf("NewPalette(false).Style(%v).GetForeground() = %#v, want %#v", r, got, wantLight)
		}

		if wantBold := boldRoles[r]; dark.Style(r).GetBold() != wantBold {
			t.Errorf("Style(%v).GetBold() = %v, want %v", r, dark.Style(r).GetBold(), wantBold)
		}
		if wantFaint := faintRoles[r]; dark.Style(r).GetFaint() != wantFaint {
			t.Errorf("Style(%v).GetFaint() = %v, want %v", r, dark.Style(r).GetFaint(), wantFaint)
		}
	}

	lightHexes := make(map[string]Role, len(allRoles))
	darkHexes := make(map[string]Role, len(allRoles))
	for _, r := range allRoles {
		hp := paletteHex[r]
		if prior, seen := lightHexes[hp.light]; seen {
			t.Errorf("light hex %q used by both role %v and role %v — the seven light hexes must be pairwise distinct", hp.light, prior, r)
		}
		lightHexes[hp.light] = r
		if prior, seen := darkHexes[hp.dark]; seen {
			t.Errorf("dark hex %q used by both role %v and role %v — the seven dark hexes must be pairwise distinct", hp.dark, prior, r)
		}
		darkHexes[hp.dark] = r
	}

	// Positive control (rule 84d1gfpywd): NewPalette(true).Header.Render
	// must actually emit ANSI, and stripANSI must strip it back to the
	// bare text — proves the shared ansistrip_test.go helper works before
	// any other test in this package relies on it.
	rendered := dark.Header.Render("x")
	if !strings.Contains(rendered, "\x1b[") {
		t.Fatalf("NewPalette(true).Header.Render(%q) = %q, expected an ANSI escape sequence", "x", rendered)
	}
	if stripped := stripANSI(rendered); stripped != "x" {
		t.Errorf("stripANSI(%q) = %q, want %q", rendered, stripped, "x")
	}
}
