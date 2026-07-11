// Package pkga exercises the constructs the extractor/resolver must handle:
// exported/unexported functions, intra-package calls, value/pointer
// receiver methods, constants, and package-level variables.
package pkga

// Version is the fixture package's exported constant.
const Version = "1"

// Registry is an exported package-level variable.
var Registry = map[string]int{}

// Alpha is an exported function that calls the unexported helper.
func Alpha() int {
	return helper()
}

// helper is an unexported function called intra-package by Alpha.
func helper() int {
	return 1
}

// Widget is an exported struct with both a value-receiver and a
// pointer-receiver method.
type Widget struct {
	Name string
}

// Describe is a value-receiver method.
func (w Widget) Describe() string {
	return w.Name
}

// Rename is a pointer-receiver method.
func (w *Widget) Rename(n string) {
	w.Name = n
}
