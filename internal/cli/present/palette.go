package present

import lipgloss "charm.land/lipgloss/v2"

// Role identifies one of the seven semantic hues a Palette carries (D-05):
// section headers, field labels, plain data values, filesystem paths,
// numeric counts, warnings, and errors.
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

// paletteHex pairs each Role with a light/dark truecolor hex pair (D-06).
// Chosen for legibility on Solarized Light (or macOS light Terminal) and
// on a dark theme — readability itself is a human UAT item (D-06/CLI-04),
// harvested at end of phase, not verified here. Truecolor hex values are
// downsampled to the terminal's actual profile at the RunE boundary
// (colorprofile.Writer, D-10) — present never performs that downsampling
// itself.
var paletteHex = map[Role]struct{ light, dark string }{
	RoleHeader:  {light: "#005f87", dark: "#5fd7ff"},
	RoleLabel:   {light: "#586e75", dark: "#93a1a1"},
	RoleValue:   {light: "#073642", dark: "#eee8d5"},
	RolePath:    {light: "#5f5faf", dark: "#af87ff"},
	RoleCount:   {light: "#875f00", dark: "#ffd75f"},
	RoleWarning: {light: "#af5f00", dark: "#ffaf00"},
	RoleError:   {light: "#af0000", dark: "#ff5f5f"},
}

// Palette holds exactly seven lipgloss.Style values, one per semantic
// role (D-05). Every Render* takes a Palette as an explicit parameter;
// there is no mutable package-level style and no setter — the palette is
// built once per RunE call from the resolver's single Dark answer
// (NewPalette).
type Palette struct {
	Header, Label, Value, Path, Count, Warning, Error lipgloss.Style
}

// NewPalette builds a Palette from ONE lipgloss.LightDark(dark) closure
// over paletteHex — Header/Warning/Error are bold, Label is faint, and
// Value/Path/Count carry no additional decoration beyond their hue (D-05).
func NewPalette(dark bool) Palette {
	ld := lipgloss.LightDark(dark)
	styleFor := func(r Role) lipgloss.Style {
		hp := paletteHex[r]
		return lipgloss.NewStyle().Foreground(ld(lipgloss.Color(hp.light), lipgloss.Color(hp.dark)))
	}

	return Palette{
		Header:  styleFor(RoleHeader).Bold(true),
		Label:   styleFor(RoleLabel).Faint(true),
		Value:   styleFor(RoleValue),
		Path:    styleFor(RolePath),
		Count:   styleFor(RoleCount),
		Warning: styleFor(RoleWarning).Bold(true),
		Error:   styleFor(RoleError).Bold(true),
	}
}

// Style returns p's lipgloss.Style for role r.
func (p Palette) Style(r Role) lipgloss.Style {
	switch r {
	case RoleHeader:
		return p.Header
	case RoleLabel:
		return p.Label
	case RoleValue:
		return p.Value
	case RolePath:
		return p.Path
	case RoleCount:
		return p.Count
	case RoleWarning:
		return p.Warning
	case RoleError:
		return p.Error
	default:
		return lipgloss.NewStyle()
	}
}
