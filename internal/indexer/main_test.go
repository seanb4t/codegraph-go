package indexer

import (
	"flag"
	"testing"

	"go.uber.org/goleak"
)

// TestMain lowers the rapid check count of this package's property tests
// to twenty on a plain run, since each case of TestSyncEqualsFreshIndex
// runs real syncs on disk and the default of one hundred costs about
// ninety seconds on every push (design D7). A -rapid.checks flag on the command line wins: task
// test:properties and the weekly fuzz workflow pass the deep count.
// After every test, goleak fails the run if a goroutine started by the
// package is still alive.
func TestMain(m *testing.M) {
	flag.Parse()
	given := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "rapid.checks" {
			given = true
		}
	})
	if !given {
		if err := flag.Set("rapid.checks", "20"); err != nil {
			panic(err)
		}
	}
	goleak.VerifyTestMain(m)
}
