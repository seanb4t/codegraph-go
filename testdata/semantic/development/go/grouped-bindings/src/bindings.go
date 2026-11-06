package grouped

import "strings"

// héllo: the comment puts non-ASCII bytes before the declarations.
var (
	First, Second = 1, 2
	Third         = strings.ToUpper("é") + name()
)

const (
	Alpha = iota
	Beta
)

var Single, Other int

func name() string { return "x" }

type Point struct {
	X, Y int
	Tag  string
}

func (p Point) Label() string { return p.Tag + name() }
