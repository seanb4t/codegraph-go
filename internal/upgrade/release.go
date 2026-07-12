// Package upgrade implements `codegraph upgrade` (CLI-02, D-11..D-14):
// resolve the target release version from GitHub Releases, download and
// verify its sigstore-go signature bundle IN-PROCESS (D-12 — never shell
// out to a cosign CLI), and only on successful verification atomically
// replace the running binary (D-13). This is the only intentional network
// path in the whole binary (D-15/telemetry) and the project's first
// cryptographic-verification code (T-06-06-01/T-06-06-02).
package upgrade

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

// tagFromLocation extracts a release tag (e.g. "v1.2.3") from a GitHub
// Releases redirect's Location header of the form
// ".../releases/tag/v1.2.3" (RESEARCH "GitHub Releases latest resolution
// without hitting the rate-limited API").
var tagFromLocation = regexp.MustCompile(`/releases/tag/([^/?#]+)`)

// httpDoer is the injectable HTTP seam: release_test.go substitutes a stub
// so resolveLatestVersion/resolveLatestVersionViaAPI never touch a real
// network under `go test` (RESEARCH Environment Availability — this is a
// runtime dependency of the shipped binary, not a build/test dependency).
// *http.Client satisfies this interface directly.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// newLatestRedirectClient returns an *http.Client configured to capture
// (not follow) the redirect GitHub's unauthenticated
// .../releases/latest issues, per the Don't Hand-Roll redirect trick: the
// redirect endpoint has no rate limit, unlike the JSON API used only as a
// fallback.
func newLatestRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// resolveLatestVersion resolves repoSlug's ("owner/repo") latest release
// tag via the unauthenticated GitHub Releases redirect trick, falling back
// to the (rate-limited) REST API only if the redirect's Location header is
// missing/blank or the request itself fails. get is injected so tests run
// fully offline.
func resolveLatestVersion(repoSlug string, get httpDoer) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://github.com/"+repoSlug+"/releases/latest", nil)
	if err != nil {
		return "", fmt.Errorf("upgrade: build releases/latest request: %w", err)
	}

	resp, err := get.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if loc := resp.Header.Get("Location"); loc != "" {
			if m := tagFromLocation.FindStringSubmatch(loc); m != nil {
				return m[1], nil
			}
		}
	}

	// Fall back to the rate-limited API only if the redirect trick failed.
	return resolveLatestVersionViaAPI(repoSlug, get)
}

// resolveLatestVersionViaAPI is the fallback path: GET
// https://api.github.com/repos/<repoSlug>/releases/latest and parse
// {"tag_name": "..."}. Unlike the redirect trick this endpoint IS
// rate-limited (60 req/h/IP unauthenticated) — it is only reached when the
// redirect trick's Location header can't be parsed.
func resolveLatestVersionViaAPI(repoSlug string, get httpDoer) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+repoSlug+"/releases/latest", nil)
	if err != nil {
		return "", fmt.Errorf("upgrade: build releases API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := get.Do(req)
	if err != nil {
		return "", fmt.Errorf("upgrade: could not resolve the latest version from GitHub. Check your network, or pin a version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upgrade: could not resolve the latest version from GitHub (API returned %s). Check your network, or pin a version", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("upgrade: decode releases API response: %w", err)
	}
	if payload.TagName == "" {
		return "", errors.New("upgrade: could not resolve the latest version from GitHub. Check your network, or pin a version")
	}

	return payload.TagName, nil
}
