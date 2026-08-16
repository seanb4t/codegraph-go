package recovery

import "testing"

// TestAccountRecovery is case (c) of the behavioral corpus: its name
// lexically matches a query like "account recovery" (it's a Test*-named
// function in a weakly-connected cluster), but it has ZERO inbound graph
// edges — nothing in the source graph calls it (only the go test runtime
// invokes it via reflection, which is not a source-visible call edge).
// Once EXPL-03's file-relevance gate lands, this is exactly the
// weakly-connected Test* symbol it must not let top the results ahead of
// recoverAccount (recovery.go), the structurally-connected non-test
// symbol in the same cluster.
func TestAccountRecovery(t *testing.T) {
	if !recoverAccount("synthetic") {
		t.Fatal("expected recovery to succeed")
	}
}
