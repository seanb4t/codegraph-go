package mcp

import (
	"context"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listResourceURIs connects to s via newTestSession and returns the sorted
// set of advertised resource URIs — listToolNames' sibling.
func listResourceURIs(t *testing.T, s *mcp.Server) []string {
	t.Helper()

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	result, err := session.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}

	uris := make([]string, len(result.Resources))
	for i, r := range result.Resources {
		uris[i] = r.URI
	}
	sort.Strings(uris)
	return uris
}

// TestResourcesListAdvertisesRegisteredURIs proves RSRC-01: a session's
// ListResources sees codegraph://tools/explore, mimeType text/markdown.
func TestResourcesListAdvertisesRegisteredURIs(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	companions, _ := ResolveCompanions("", false)
	s := BuildServer(true, companions, dir, dir)

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	result, err := session.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}

	var found *mcp.Resource
	for _, r := range result.Resources {
		if r.URI == "codegraph://tools/explore" {
			found = r
			break
		}
	}
	if found == nil {
		uris := listResourceURIs(t, s)
		t.Fatalf("resources/list did not advertise codegraph://tools/explore, got %v", uris)
	}
	if found.MIMEType != "text/markdown" {
		t.Fatalf("codegraph://tools/explore MIMEType = %q, want text/markdown", found.MIMEType)
	}
}

// TestResourcesReadReturnsMarkdown proves RSRC-02: for every URI
// ListResources advertised, ReadResource returns exactly one Contents
// element with non-empty Text and MIMEType text/markdown. Driven off the
// LIST result, never a re-typed URI literal, so this grows for free as
// later plans add resources.
func TestResourcesReadReturnsMarkdown(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	companions, _ := ResolveCompanions("", false)
	s := BuildServer(true, companions, dir, dir)

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	listRes, err := session.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(listRes.Resources) == 0 {
		t.Fatalf("ListResources returned no resources")
	}

	for _, r := range listRes.Resources {
		readRes, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: r.URI})
		if err != nil {
			t.Fatalf("ReadResource(%q): %v", r.URI, err)
		}
		if len(readRes.Contents) != 1 {
			t.Fatalf("ReadResource(%q) returned %d Contents, want 1", r.URI, len(readRes.Contents))
		}
		content := readRes.Contents[0]
		if content.Text == "" {
			t.Fatalf("ReadResource(%q) returned empty Text", r.URI)
		}
		if content.MIMEType != "text/markdown" {
			t.Fatalf("ReadResource(%q) MIMEType = %q, want text/markdown", r.URI, content.MIMEType)
		}
	}
}

// TestResourcesRegisterWithoutIndex proves RSRC-03 as a behavior, not
// inferred from source position: BuildServer(false, nil, dir, dir) — the
// zero-tools path (MCP-03) — still returns the identical URI set from
// ListResources, and ReadResource still returns non-empty text/markdown.
func TestResourcesRegisterWithoutIndex(t *testing.T) {
	dir := copyFixture(t)

	withIndexDir := copyFixture(t)
	indexFixture(t, withIndexDir)
	companions, _ := ResolveCompanions("", false)
	withIndex := BuildServer(true, companions, withIndexDir, withIndexDir)
	want := listResourceURIs(t, withIndex)

	s := BuildServer(false, nil, dir, dir)
	got := listResourceURIs(t, s)

	if len(got) != len(want) {
		t.Fatalf("resources/list without an index = %v, want the same URI set as with an index %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("resources/list without an index = %v, want %v", got, want)
		}
	}

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	readRes, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "codegraph://tools/explore"})
	if err != nil {
		t.Fatalf("ReadResource(codegraph://tools/explore) without an index: %v", err)
	}
	if len(readRes.Contents) != 1 || readRes.Contents[0].Text == "" {
		t.Fatalf("ReadResource(codegraph://tools/explore) without an index returned no/empty content")
	}
	if readRes.Contents[0].MIMEType != "text/markdown" {
		t.Fatalf("ReadResource(codegraph://tools/explore) without an index MIMEType = %q, want text/markdown", readRes.Contents[0].MIMEType)
	}
}

// TestResourcesReadIsNotVacuous proves the read assertions above actually
// discriminate: without this, a handler that returned empty-but-successful
// for every URI would pass the three tests above forever.
func TestResourcesReadIsNotVacuous(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	companions, _ := ResolveCompanions("", false)
	s := BuildServer(true, companions, dir, dir)

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	_, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "codegraph://tools/does-not-exist"})
	if err == nil {
		t.Fatalf("ReadResource(codegraph://tools/does-not-exist) returned no error, want ResourceNotFoundError")
	}
}
