package routes

import "testing"

const djangoFixture = "testdata/django_fixture.py"

// TestDjango_DetectsUrlpatternsEntries proves the Django detector walks
// urlpatterns' path()/re_path() entries, resolving a cross-file handler
// reference (views.get_user) via ResolveByName's own by-name lookup —
// this test's stub simulates the caller's global (cross-file) fallback by
// keying byName directly, since routes package itself has no notion of
// "same file vs. cross file" (that split lives in the caller's
// HandlerResolver implementation, internal/indexer/routes_detect.go).
func TestDjango_DetectsUrlpatternsEntries(t *testing.T) {
	root, src := parseFixture(t, "python", djangoFixture)

	resolver := newStubResolver().
		withName("get_user", "function:aaa").
		withName("admin_view", "function:bbb")

	got := walkDjangoRoutes(root, src, resolver)

	pathRoute, ok := findRoute(got, "ANY", "users/<int:id>")
	if !ok {
		t.Fatalf("walkDjangoRoutes() missing path() entry, got %+v", got)
	}
	if pathRoute.HandlerID != "function:aaa" {
		t.Errorf("path() route HandlerID = %q, want function:aaa", pathRoute.HandlerID)
	}

	reRoute, ok := findRoute(got, "ANY", "^admin/$")
	if !ok {
		t.Fatalf("walkDjangoRoutes() missing re_path() entry, got %+v", got)
	}
	if reRoute.HandlerID != "function:bbb" {
		t.Errorf("re_path() route HandlerID = %q, want function:bbb", reRoute.HandlerID)
	}
}

// TestDjango_SignatureOptIn proves the Django detector's Signature only
// fires when django appears in the manifest (D-09).
func TestDjango_SignatureOptIn(t *testing.T) {
	var det Detector
	for _, d := range Registered() {
		if d.ID == "django-route" {
			det = d
		}
	}
	if det.ID == "" {
		t.Fatal("django-route detector not registered")
	}
	if !det.Signature(`dependencies = ["django"]`) {
		t.Error("Signature() = false for a manifest mentioning django, want true")
	}
	if det.Signature(`dependencies = ["flask"]`) {
		t.Error("Signature() = true with no django dependency, want false (D-09 opt-in)")
	}
}

const flaskFixture = "testdata/flask_fixture.py"

// TestFlask_DetectsRouteAndDirectVerbDecorators proves the shared Flask/
// FastAPI walker detects `.route(..., methods=[...])` (deriving its verb
// from the methods list) and a direct-verb decorator (`.post(...)`),
// resolving each decorated function itself as the handler.
func TestFlask_DetectsRouteAndDirectVerbDecorators(t *testing.T) {
	root, src := parseFixture(t, "python", flaskFixture)

	resolver := newStubResolver().
		withLine(7, "function:aaa"). // def get_user's own line (decorator excluded)
		withLine(12, "function:bbb") // def create_user's own line

	got := walkFlaskFastapiRoutes(root, src, resolver)

	get, ok := findRoute(got, "GET", "/users/<id>")
	if !ok {
		t.Fatalf("walkFlaskFastapiRoutes() missing GET /users/<id>, got %+v", got)
	}
	if get.HandlerID != "function:aaa" {
		t.Errorf("GET route HandlerID = %q, want function:aaa", get.HandlerID)
	}

	post, ok := findRoute(got, "POST", "/users")
	if !ok {
		t.Fatalf("walkFlaskFastapiRoutes() missing POST /users, got %+v", got)
	}
	if post.HandlerID != "function:bbb" {
		t.Errorf("POST route HandlerID = %q, want function:bbb", post.HandlerID)
	}
}

// TestFlask_SignatureOptIn proves the Flask detector's Signature only
// fires when flask appears in the manifest (D-09).
func TestFlask_SignatureOptIn(t *testing.T) {
	var det Detector
	for _, d := range Registered() {
		if d.ID == "flask-route" {
			det = d
		}
	}
	if det.ID == "" {
		t.Fatal("flask-route detector not registered")
	}
	if !det.Signature(`dependencies = ["flask"]`) {
		t.Error("Signature() = false for a manifest mentioning flask, want true")
	}
	if det.Signature(`dependencies = ["fastapi"]`) {
		t.Error("Signature() = true with no flask dependency, want false (D-09 opt-in)")
	}
}

const fastapiFixture = "testdata/fastapi_fixture.py"

// TestFastAPI_DetectsDirectVerbDecorator proves the shared Flask/FastAPI
// walker also covers a FastAPI-style `.get(...)` decorator.
func TestFastAPI_DetectsDirectVerbDecorator(t *testing.T) {
	root, src := parseFixture(t, "python", fastapiFixture)

	resolver := newStubResolver().withLine(7, "function:aaa")

	got := walkFlaskFastapiRoutes(root, src, resolver)

	get, ok := findRoute(got, "GET", "/items/{id}")
	if !ok {
		t.Fatalf("walkFlaskFastapiRoutes() missing GET /items/{id}, got %+v", got)
	}
	if get.HandlerID != "function:aaa" {
		t.Errorf("GET route HandlerID = %q, want function:aaa", get.HandlerID)
	}
}

// TestFastAPI_SignatureOptIn proves the FastAPI detector's Signature only
// fires when fastapi appears in the manifest (D-09).
func TestFastAPI_SignatureOptIn(t *testing.T) {
	var det Detector
	for _, d := range Registered() {
		if d.ID == "fastapi-route" {
			det = d
		}
	}
	if det.ID == "" {
		t.Fatal("fastapi-route detector not registered")
	}
	if !det.Signature(`dependencies = ["fastapi"]`) {
		t.Error("Signature() = false for a manifest mentioning fastapi, want true")
	}
	if det.Signature(`dependencies = ["django"]`) {
		t.Error("Signature() = true with no fastapi dependency, want false (D-09 opt-in)")
	}
}
