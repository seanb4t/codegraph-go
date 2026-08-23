package mcp

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"
)

// TestPendingWriterDrainsEarlyWithoutFix is FIX-01's committed reproduction
// (ROADMAP success criterion 5: the fix must be "proven against a
// reproduction that showed the loss"). It demonstrates a PREMATURE DRAIN —
// the pending count reaching zero while a response is still owed, so
// waitForDrain stops waiting and the reader propagates EOF early — NOT a
// proven abandoned response. Whether a particular in-flight response is
// then actually lost depends on the surrounding go-sdk session and is not
// what this primitive-level reproduction observes; a full-stdio test
// confirming an abandoned response would be stronger evidence and is out
// of scope here.
//
// Reproduction: one client call is accepted (pending=1, mirroring
// stdinLingerReader.Read's increment for an accepted call whose response
// has not yet been written). A server-initiated notification — never
// counted on the increment side, since stdinLingerReader only increments
// for lines looksLikeJSONRPCCall classifies as calls — is written through
// pendingWriter BEFORE the call's own response is written. Before the fix,
// pendingWriter.Write unconditionally decrements on every Write, so this
// single notification alone drains pending to zero and waitForDrain
// returns immediately even though the call's response has not been
// written. After the fix, the notification does not decrement (it carries
// a "method" and no "id", so looksLikeJSONRPCResponse classifies it as not
// a response), pending stays at 1, and waitForDrain keeps blocking until
// the real response write lands.
func TestPendingWriterDrainsEarlyWithoutFix(t *testing.T) {
	var pending atomic.Int64
	pending.Store(1) // one accepted client call, response not yet written

	var buf bytes.Buffer
	pw := &pendingWriter{w: &buf, pending: &pending}

	notification := []byte(`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}` + "\n")
	if _, err := pw.Write(notification); err != nil {
		t.Fatalf("Write notification: %v", err)
	}

	// The reproduction: waitForDrain must still be blocked here, because
	// the accepted call's response has not been written yet. If it
	// unblocks after only the notification was written, that is the
	// premature drain — the pending count reached zero while a response
	// was still owed.
	s := &stdinLingerReader{pending: &pending}
	drained := make(chan struct{})
	go func() {
		s.waitForDrain()
		close(drained)
	}()

	select {
	case <-drained:
		t.Fatalf("waitForDrain returned after a notification write alone (pending=%d) — premature drain: the accepted call's response was never written, so drain should still be blocking", pending.Load())
	case <-time.After(20 * time.Millisecond):
		// Expected on the fixed code: still blocked, because the
		// notification write did not decrement pending.
	}

	// Now the accepted call's actual response lands.
	response := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}` + "\n")
	if _, err := pw.Write(response); err != nil {
		t.Fatalf("Write response: %v", err)
	}

	select {
	case <-drained:
		// Expected: drain unblocks once the real response is written.
	case <-time.After(stdinLingerGrace + time.Second):
		t.Fatalf("waitForDrain did not unblock after the accepted call's response was written")
	}
}
