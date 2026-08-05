// FROZEN: this tree is checked in solely for the wire oracle
// (testdata/wireoracle/fixture) and must not be edited once transcripts
// under testdata/wireoracle/transcripts/ are frozen (D-08).

// Package pkgb exports one function pkga.Alpha calls (cross-package call).
package pkgb

// Helper is an exported function that pkga.Alpha calls.
func Helper() string {
	return "helper"
}
