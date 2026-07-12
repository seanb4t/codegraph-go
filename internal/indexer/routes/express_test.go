package routes

import "testing"

const expressFixture = "testdata/express_fixture.ts"

// TestExpress_DetectsRoutesWithAnyReceiver proves the Express detector
// emits a route per app.get/app.post call, with the SAME any-identifier-
// receiver precision gate as Gin (Pattern 4).
func TestExpress_DetectsRoutesWithAnyReceiver(t *testing.T) {
	root, src := parseFixture(t, "typescript", expressFixture)

	resolver := newStubResolver().
		withName("getUserHandler", "function:aaa").
		withName("createUserHandler", "function:bbb")

	got := walkExpressRoutes(root, src, resolver)

	get, ok := findRoute(got, "GET", "/users/:id")
	if !ok {
		t.Fatalf("walkExpressRoutes() missing GET /users/:id, got %+v", got)
	}
	if get.HandlerID != "function:aaa" {
		t.Errorf("GET route HandlerID = %q, want function:aaa", get.HandlerID)
	}

	post, ok := findRoute(got, "POST", "/users")
	if !ok {
		t.Fatalf("walkExpressRoutes() missing POST /users, got %+v", got)
	}
	if post.HandlerID != "function:bbb" {
		t.Errorf("POST route HandlerID = %q, want function:bbb", post.HandlerID)
	}
}

// TestExpress_SignatureOptIn proves the Express detector's Signature only
// fires when "express" appears as a package.json dependency key (D-09).
func TestExpress_SignatureOptIn(t *testing.T) {
	var det Detector
	for _, d := range Registered() {
		if d.ID == "express-route" && d.Language == "typescript" {
			det = d
		}
	}
	if det.ID == "" {
		t.Fatal("express-route (typescript) detector not registered")
	}
	if !det.Signature(`{"dependencies":{"express":"^4.18.0"}}`) {
		t.Error("Signature() = false for a package.json mentioning express, want true")
	}
	if det.Signature(`{"dependencies":{"koa":"^2.0.0"}}`) {
		t.Error("Signature() = true with no express dependency, want false (D-09 opt-in)")
	}
}

const nestjsFixture = "testdata/nestjs_fixture.ts"

// TestNest_DetectsMethodDecorators proves the NestJS detector pairs each
// class_body's preceding "decorator" sibling with the method_definition
// that immediately follows it, resolving the METHOD's own StartPosition
// (excluding the decorator) as the handler lookup key.
func TestNest_DetectsMethodDecorators(t *testing.T) {
	root, src := parseFixture(t, "typescript", nestjsFixture)

	resolver := newStubResolver().
		withLine(6, "method:aaa"). // getUser's own line, decorator excluded
		withLine(11, "method:bbb") // createUser's own line

	got := walkNestJSRoutes(root, src, resolver)

	get, ok := findRoute(got, "GET", ":id")
	if !ok {
		t.Fatalf("walkNestJSRoutes() missing GET :id, got %+v", got)
	}
	if get.HandlerID != "method:aaa" {
		t.Errorf("GET route HandlerID = %q, want method:aaa", get.HandlerID)
	}

	post, ok := findRoute(got, "POST", "")
	if !ok {
		t.Fatalf("walkNestJSRoutes() missing bare POST route, got %+v", got)
	}
	if post.HandlerID != "method:bbb" {
		t.Errorf("POST route HandlerID = %q, want method:bbb", post.HandlerID)
	}
}

// TestNest_SignatureOptIn proves the NestJS detector's Signature only
// fires when @nestjs appears in the manifest (D-09).
func TestNest_SignatureOptIn(t *testing.T) {
	var det Detector
	for _, d := range Registered() {
		if d.ID == "nestjs-route" && d.Language == "typescript" {
			det = d
		}
	}
	if det.ID == "" {
		t.Fatal("nestjs-route (typescript) detector not registered")
	}
	if !det.Signature(`{"dependencies":{"@nestjs/common":"^10.0.0"}}`) {
		t.Error("Signature() = false for a package.json mentioning @nestjs, want true")
	}
	if det.Signature(`{"dependencies":{"express":"^4.18.0"}}`) {
		t.Error("Signature() = true with no @nestjs dependency, want false (D-09 opt-in)")
	}
}
