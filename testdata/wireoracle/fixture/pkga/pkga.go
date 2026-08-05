// FROZEN: this tree is checked in solely for the wire oracle
// (testdata/wireoracle/fixture) and must not be edited once transcripts
// under testdata/wireoracle/transcripts/ are frozen (D-08).

// Package pkga exports two functions where one calls the other, plus a
// cross-package call into pkgb — the shape the wire oracle's
// handshake-explore scenario queries against.
package pkga

import "wireoraclefixture/pkgb"

// Alpha is an exported function that calls Beta (intra-package) and
// pkgb.Helper (cross-package). This is the symbol name the
// handshake-explore scenario's codegraph_explore query argument names.
func Alpha() string {
	return Beta() + pkgb.Helper()
}

// Beta is an exported function called by Alpha.
func Beta() string {
	return "beta"
}
