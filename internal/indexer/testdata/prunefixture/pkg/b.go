package pkg

// UseFoo calls Foo (declared in pkg/a.go) — a cross-file call edge that
// must regenerate cleanly, or vanish with no dangling remnant, across
// every 04-04 file operation applied to pkg/a.go.
func UseFoo() int {
	return Foo()
}
