package uiserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// testPort and testOtherPort are the ports the guard's rejection matrix is
// built around — testOtherPort exists solely to prove the port is part of
// a Host's identity (bullet: "the port is part of the identity").
const (
	testPort      = "54321"
	testOtherPort = "9999"
)

// newGuardedHandler is the test-only seam this suite drives through: it
// wraps next in the real originHostGuard bound to port. (RED history:
// before originHostGuard existed, this seam was a bare passthrough and
// the "should reject" rows below failed — see the commit that introduced
// this file for the recorded 14-failing-subtest RED count, ROADMAP
// success criterion 2's required evidence. This commit turns them green.)
func newGuardedHandler(port string, next http.Handler) http.Handler {
	return originHostGuard(port, next)
}

type guardCase struct {
	name           string
	host           string
	origin         string
	method         string
	wantStatus     int
	wantNextCalled bool
}

func TestOriginHostGuard(t *testing.T) {
	// allowedHosts(p) / allowedOrigins(p) must be EXACTLY the three
	// admitted literals for this port — asserted by set equality with an
	// explicit length check, so an added or dropped entry fails even
	// though no rejection-matrix row would notice a spuriously-admitted
	// fourth entry.
	wantHosts := map[string]struct{}{
		"127.0.0.1:" + testPort: {},
		"localhost:" + testPort: {},
		"[::1]:" + testPort:     {},
	}
	gotHosts := allowedHosts(testPort)
	if len(gotHosts) != 3 {
		t.Fatalf("allowedHosts(%q) has %d entries, want 3: %v", testPort, len(gotHosts), gotHosts)
	}
	for h := range wantHosts {
		if _, ok := gotHosts[h]; !ok {
			t.Fatalf("allowedHosts(%q) missing %q", testPort, h)
		}
	}
	for h := range gotHosts {
		if _, ok := wantHosts[h]; !ok {
			t.Fatalf("allowedHosts(%q) has unexpected entry %q", testPort, h)
		}
	}

	wantOrigins := map[string]struct{}{
		"http://127.0.0.1:" + testPort: {},
		"http://localhost:" + testPort: {},
		"http://[::1]:" + testPort:     {},
	}
	gotOrigins := allowedOrigins(testPort)
	if len(gotOrigins) != 3 {
		t.Fatalf("allowedOrigins(%q) has %d entries, want 3: %v", testPort, len(gotOrigins), gotOrigins)
	}
	for o := range wantOrigins {
		if _, ok := gotOrigins[o]; !ok {
			t.Fatalf("allowedOrigins(%q) missing %q", testPort, o)
		}
	}
	for o := range gotOrigins {
		if _, ok := wantOrigins[o]; !ok {
			t.Fatalf("allowedOrigins(%q) has unexpected entry %q", testPort, o)
		}
	}

	cases := []guardCase{
		// Admitted Host, no Origin (bullets 1-3).
		{name: "admitted host 127.0.0.1", host: "127.0.0.1:" + testPort, method: http.MethodGet, wantStatus: http.StatusOK, wantNextCalled: true},
		{name: "admitted host localhost", host: "localhost:" + testPort, method: http.MethodGet, wantStatus: http.StatusOK, wantNextCalled: true},
		{name: "admitted host [::1]", host: "[::1]:" + testPort, method: http.MethodGet, wantStatus: http.StatusOK, wantNextCalled: true},
		// Rejected Host (bullets 4-9).
		{name: "rejected host evil.com", host: "evil.com", method: http.MethodGet, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "rejected host prefix-defeat 127.0.0.1.evil.com", host: "127.0.0.1.evil.com:" + testPort, method: http.MethodGet, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "rejected host suffix-defeat evil.com:port", host: "evil.com:" + testPort, method: http.MethodGet, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "rejected host substring-defeat notlocalhost", host: "notlocalhost:" + testPort, method: http.MethodGet, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "rejected host wrong port", host: "127.0.0.1:" + testOtherPort, method: http.MethodGet, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "rejected host empty", host: "", method: http.MethodGet, wantStatus: http.StatusForbidden, wantNextCalled: false},
		// Admitted PAIR: Host+Origin same spelling, POST (bullets 10-12).
		{name: "admitted pair 127.0.0.1", host: "127.0.0.1:" + testPort, origin: "http://127.0.0.1:" + testPort, method: http.MethodPost, wantStatus: http.StatusOK, wantNextCalled: true},
		{name: "admitted pair localhost", host: "localhost:" + testPort, origin: "http://localhost:" + testPort, method: http.MethodPost, wantStatus: http.StatusOK, wantNextCalled: true},
		{name: "admitted pair [::1]", host: "[::1]:" + testPort, origin: "http://[::1]:" + testPort, method: http.MethodPost, wantStatus: http.StatusOK, wantNextCalled: true},
		// Rejected Origin, admitted Host (bullets 13-14).
		{name: "rejected origin evil.com", host: "127.0.0.1:" + testPort, origin: "http://evil.com", method: http.MethodPost, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "rejected origin suffix-defeat", host: "127.0.0.1:" + testPort, origin: "http://127.0.0.1:" + testPort + ".evil.com", method: http.MethodPost, wantStatus: http.StatusForbidden, wantNextCalled: false},
		// Rejected CROSS-SPELLING pairs (bullets 15-18): both members are
		// individually admitted literals, but the PAIR is not, because
		// the three spellings are three different names, not three
		// renderings of one.
		{name: "cross-spelling host=127.0.0.1 origin=localhost", host: "127.0.0.1:" + testPort, origin: "http://localhost:" + testPort, method: http.MethodPost, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "cross-spelling host=localhost origin=127.0.0.1", host: "localhost:" + testPort, origin: "http://127.0.0.1:" + testPort, method: http.MethodPost, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "cross-spelling host=[::1] origin=localhost", host: "[::1]:" + testPort, origin: "http://localhost:" + testPort, method: http.MethodPost, wantStatus: http.StatusForbidden, wantNextCalled: false},
		{name: "cross-spelling host=localhost origin=[::1]", host: "localhost:" + testPort, origin: "http://[::1]:" + testPort, method: http.MethodPost, wantStatus: http.StatusForbidden, wantNextCalled: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			handler := newGuardedHandler(testPort, next)

			req := httptest.NewRequest(tc.method, "http://example.com/", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("host=%q origin=%q: status = %d, want %d", tc.host, tc.origin, rec.Code, tc.wantStatus)
			}
			if nextCalled != tc.wantNextCalled {
				t.Fatalf("host=%q origin=%q: nextCalled = %v, want %v (status alone does not prove whether the downstream handler ran)", tc.host, tc.origin, nextCalled, tc.wantNextCalled)
			}
		})
	}

	// Bullet 20: concurrency — N goroutines issuing a mixed
	// admitted/foreign workload must each receive the verdict their own
	// request earns; the guard is stateless (built once from read-only
	// maps), so no cross-request contamination should be observable
	// under -race.
	t.Run("concurrency: mixed admitted/foreign workload is verdict-correct per request", func(t *testing.T) {
		const n = 64
		handler := newGuardedHandler(testPort, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		var wg sync.WaitGroup
		errs := make(chan string, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				admitted := i%2 == 0
				host := "evil.com"
				if admitted {
					host = "127.0.0.1:" + testPort
				}
				req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
				req.Host = host
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				wantStatus := http.StatusForbidden
				if admitted {
					wantStatus = http.StatusOK
				}
				if rec.Code != wantStatus {
					errs <- fmt.Sprintf("goroutine %d: host=%q got status %d, want %d", i, host, rec.Code, wantStatus)
				}
			}(i)
		}
		wg.Wait()
		close(errs)
		for e := range errs {
			t.Error(e)
		}
	})
}

// TestOriginHostGuardAdmitsOriginlessGET is behavior bullet 19: a GET
// carrying no Origin header at all must still be admitted when its Host
// is one of the three allowed literals — over-rejection (treating
// Origin's absence as suspicious) is the failure mode 01-RESEARCH.md's
// Pitfall 5 predicts, since browsers do not reliably send Origin on
// same-origin GET navigation.
func TestOriginHostGuardAdmitsOriginlessGET(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	handler := newGuardedHandler(testPort, next)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Host = "127.0.0.1:" + testPort
	// Deliberately no Origin header.

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET with admitted Host and no Origin header: status = %d, want 200 (over-rejection is the predicted failure mode)", rec.Code)
	}
	if !nextCalled {
		t.Fatal("GET with admitted Host and no Origin header: next handler did not run")
	}
}
