package routes

import "testing"

const springFixture = "testdata/spring_fixture.java"

// TestSpring_DetectsDirectAndRequestMappingAnnotations proves both a
// direct-verb annotation (@GetMapping) and the generic @RequestMapping
// (with an explicit method element) are detected, resolving the
// annotated method declaration itself as the handler.
func TestSpring_DetectsDirectAndRequestMappingAnnotations(t *testing.T) {
	root, src := parseFixture(t, "java", springFixture)

	resolver := newStubResolver().
		withLine(7, "method:aaa"). // @GetMapping line -> method_declaration's own StartLine
		withLine(12, "method:bbb") // @RequestMapping line

	got := walkSpringRoutes(root, src, resolver)

	get, ok := findRoute(got, "GET", "/users/{id}")
	if !ok {
		t.Fatalf("walkSpringRoutes() missing GET /users/{id}, got %+v", got)
	}
	if get.HandlerID != "method:aaa" {
		t.Errorf("GET route HandlerID = %q, want method:aaa", get.HandlerID)
	}

	post, ok := findRoute(got, "POST", "/users")
	if !ok {
		t.Fatalf("walkSpringRoutes() missing POST /users, got %+v", got)
	}
	if post.HandlerID != "method:bbb" {
		t.Errorf("POST route HandlerID = %q, want method:bbb", post.HandlerID)
	}
}

// TestSpring_SignatureOptIn proves the Spring detector's Signature only
// fires when org.springframework appears in the manifest (D-09).
func TestSpring_SignatureOptIn(t *testing.T) {
	var det Detector
	for _, d := range Registered() {
		if d.ID == "spring-route" {
			det = d
		}
	}
	if det.ID == "" {
		t.Fatal("spring-route detector not registered")
	}
	if !det.Signature("<groupId>org.springframework.boot</groupId>") {
		t.Error("Signature() = false for a pom.xml mentioning org.springframework, want true")
	}
	if det.Signature("<groupId>com.example</groupId>") {
		t.Error("Signature() = true with no Spring dependency, want false (D-09 opt-in)")
	}
}
