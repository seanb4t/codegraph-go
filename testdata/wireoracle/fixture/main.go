// FROZEN: this tree is checked in solely for the wire oracle
// (testdata/wireoracle/fixture) and must not be edited once transcripts
// under testdata/wireoracle/transcripts/ are frozen (D-08).

// Command wireoraclefixture is the wire oracle's dedicated, purpose-built
// discovery fixture — calling into both packages below.
package main

import (
	"fmt"

	"wireoraclefixture/pkga"
	"wireoraclefixture/pkgb"
)

func main() {
	fmt.Println(pkga.Alpha())
	fmt.Println(pkgb.Helper())
}
