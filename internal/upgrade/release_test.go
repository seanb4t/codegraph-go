package upgrade

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// stubDoer is the release_test.go network double for httpDoer — every test
// in this file must be fully offline (RESEARCH Environment Availability:
// resolveLatestVersion is a runtime dep, not a build/test dep).
type stubDoer struct {
	do func(req *http.Request) (*http.Response, error)
}

func (s stubDoer) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

// TestResolveLatestVersion_RedirectTrick asserts the primary path: an
// unauthenticated .../releases/latest redirect Location header is parsed
// into a bare tag, with no network call ever made (the stub is the only
// transport).
func TestResolveLatestVersion_RedirectTrick(t *testing.T) {
	doer := stubDoer{do: func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://github.com/seanb4t/codegraph-go/releases/latest" {
			t.Fatalf("unexpected request URL: %s", req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://github.com/seanb4t/codegraph-go/releases/tag/v1.2.3"}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}}

	got, err := resolveLatestVersion("seanb4t/codegraph-go", doer)
	if err != nil {
		t.Fatalf("resolveLatestVersion: %v", err)
	}
	if got != "v1.2.3" {
		t.Errorf("resolveLatestVersion = %q, want v1.2.3", got)
	}
}

// TestResolveLatestVersion_APIFallbackOnBlankLocation exercises the
// fallback: a blank/missing Location header (redirect trick failed) falls
// through to the rate-limited API resolver, whose stubbed response still
// never touches a real network.
func TestResolveLatestVersion_APIFallbackOnBlankLocation(t *testing.T) {
	doer := stubDoer{do: func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "github.com":
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		case "api.github.com":
			if got := req.Header.Get("Accept"); got != "application/vnd.github+json" {
				t.Errorf("API fallback Accept header = %q, want application/vnd.github+json", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v9.9.9"}`)),
			}, nil
		default:
			t.Fatalf("unexpected request host: %s", req.URL.Host)
			return nil, nil
		}
	}}

	got, err := resolveLatestVersion("seanb4t/codegraph-go", doer)
	if err != nil {
		t.Fatalf("resolveLatestVersion: %v", err)
	}
	if got != "v9.9.9" {
		t.Errorf("resolveLatestVersion = %q, want v9.9.9", got)
	}
}

// TestResolveLatestVersionViaAPI_EmptyTagIsError guards against silently
// resolving to an empty version string if GitHub's API response shape ever
// changes.
func TestResolveLatestVersionViaAPI_EmptyTagIsError(t *testing.T) {
	doer := stubDoer{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	}}

	if _, err := resolveLatestVersionViaAPI("seanb4t/codegraph-go", doer); err == nil {
		t.Fatal("resolveLatestVersionViaAPI: expected error on empty tag_name, got nil")
	}
}
