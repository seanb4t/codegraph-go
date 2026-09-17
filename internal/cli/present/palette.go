package present

import lipgloss "charm.land/lipgloss/v2"

// Role identifies one of the seven semantic hues a Palette carries (D-05).
//
// PLACEHOLDER (RED phase, 04-03 Task 2): real values land in GREEN.
type Role int

const (
	RoleHeader Role = iota
	RoleLabel
	RoleValue
	RolePath
	RoleCount
	RoleWarning
	RoleError
)

// paletteHex is a PLACEHOLDER: empty during RED, so every role lookup
// fails and TestNewPaletteRolesFollowDark reports an assertion failure,
// not a compile error.
var paletteHex = map[Role]struct{ light, dark string }{}

// Palette is a PLACEHOLDER: the real seven-field struct lands in GREEN.
type Palette struct {
	Header, Label, Value, Path, Count, Warning, Error lipgloss.Style
}

// NewPalette is a PLACEHOLDER: always returns the zero Palette during RED.
func NewPalette(dark bool) Palette {
	return Palette{}
}

// Style is a PLACEHOLDER: always returns the zero Style during RED.
func (p Palette) Style(r Role) lipgloss.Style {
	return lipgloss.Style{}
}
