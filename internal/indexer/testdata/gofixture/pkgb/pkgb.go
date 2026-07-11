// Package pkgb exercises a cross-package qualified call into pkga.
package pkgb

import "example.com/gofixture/pkga"

// Run calls pkga.Alpha (cross-package qualified call) and constructs a
// pkga.Widget literal before calling its value-receiver method.
func Run() {
	_ = pkga.Alpha()
	w := pkga.Widget{}
	_ = w.Describe()
}
