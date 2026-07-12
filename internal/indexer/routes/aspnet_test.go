package routes

import "testing"

const aspnetFixture = "testdata/aspnet_fixture.cs"

// TestAspNet_DetectsHttpVerbAttributes proves the ASP.NET detector emits
// a route per Http<Verb> attribute, resolving the attributed method
// declaration itself as the handler — including a bare, argument-less
// attribute ([HttpPost]) whose Path defaults to "".
func TestAspNet_DetectsHttpVerbAttributes(t *testing.T) {
	root, src := parseFixture(t, "csharp", aspnetFixture)

	resolver := newStubResolver().
		withLine(7, "method:aaa").
		withLine(12, "method:bbb")

	got := walkAspNetRoutes(root, src, resolver)

	get, ok := findRoute(got, "GET", "{id}")
	if !ok {
		t.Fatalf("walkAspNetRoutes() missing GET {id}, got %+v", got)
	}
	if get.HandlerID != "method:aaa" {
		t.Errorf("GET route HandlerID = %q, want method:aaa", get.HandlerID)
	}

	post, ok := findRoute(got, "POST", "")
	if !ok {
		t.Fatalf("walkAspNetRoutes() missing bare POST route, got %+v", got)
	}
	if post.HandlerID != "method:bbb" {
		t.Errorf("POST route HandlerID = %q, want method:bbb", post.HandlerID)
	}
}

// TestAspNet_SignatureOptIn proves the ASP.NET detector's Signature only
// fires when Microsoft.AspNetCore appears in the manifest (D-09).
func TestAspNet_SignatureOptIn(t *testing.T) {
	var det Detector
	for _, d := range Registered() {
		if d.ID == "aspnet-route" {
			det = d
		}
	}
	if det.ID == "" {
		t.Fatal("aspnet-route detector not registered")
	}
	if !det.Signature(`<PackageReference Include="Microsoft.AspNetCore.App" />`) {
		t.Error("Signature() = false for a csproj mentioning Microsoft.AspNetCore, want true")
	}
	if det.Signature(`<PackageReference Include="Newtonsoft.Json" />`) {
		t.Error("Signature() = true with no ASP.NET dependency, want false (D-09 opt-in)")
	}
}
