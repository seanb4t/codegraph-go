package routes

import "testing"

const ginFixture = "testdata/gin_fixture.go"

// TestGin_DetectsRoutesWithGroupVariable proves the Gin detector emits a
// route per GET/POST call on a group VARIABLE (not a fixed "router"/"r"
// receiver name — Pattern 4's precision gate is verb + string-literal
// path + handler argument, not the receiver's identifier).
func TestGin_DetectsRoutesWithGroupVariable(t *testing.T) {
	root, src := parseFixture(t, "go", ginFixture)

	resolver := newStubResolver().
		withName("getUserHandler", "function:aaa").
		withName("createUserHandler", "function:bbb")

	got := walkGinRoutes(root, src, resolver)

	get, ok := findRoute(got, "GET", "/users/:id")
	if !ok {
		t.Fatalf("walkGinRoutes() missing GET /users/:id, got %+v", got)
	}
	if get.HandlerID != "function:aaa" {
		t.Errorf("GET route HandlerID = %q, want function:aaa", get.HandlerID)
	}

	post, ok := findRoute(got, "POST", "/users")
	if !ok {
		t.Fatalf("walkGinRoutes() missing POST /users, got %+v", got)
	}
	if post.HandlerID != "function:bbb" {
		t.Errorf("POST route HandlerID = %q, want function:bbb", post.HandlerID)
	}
}

// TestGin_UnresolvedHandlerSkipsRoute proves a route whose handler cannot
// be resolved is never emitted (D-06a: no dangling edge).
func TestGin_UnresolvedHandlerSkipsRoute(t *testing.T) {
	root, src := parseFixture(t, "go", ginFixture)
	got := walkGinRoutes(root, src, newStubResolver())
	if len(got) != 0 {
		t.Fatalf("walkGinRoutes() with no resolvable handlers = %+v, want empty", got)
	}
}

// TestGin_SignatureOptIn proves the Gin detector's Signature only fires
// when the go.mod manifest text actually mentions gin-gonic/gin (D-09).
func TestGin_SignatureOptIn(t *testing.T) {
	var det Detector
	for _, d := range Registered() {
		if d.ID == "gin-route" {
			det = d
		}
	}
	if det.ID == "" {
		t.Fatal("gin-route detector not registered")
	}
	if !det.Signature("require github.com/gin-gonic/gin v1.9.1") {
		t.Error("Signature() = false for a go.mod mentioning gin-gonic/gin, want true")
	}
	if det.Signature("module example.com/foo\n\ngo 1.26\n") {
		t.Error("Signature() = true for a go.mod with no gin dependency, want false (D-09 opt-in)")
	}
}
