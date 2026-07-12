package indexer

import "testing"

// TestDjango_RouteDetectionEndToEnd proves a full Run() over the Django
// fixture resolves urlpatterns' path()/re_path() handler references
// ACROSS FILES (views.py's get_user/admin_view from urls.py's own
// urlpatterns list) via the global, cross-file handler index.
func TestDjango_RouteDetectionEndToEnd(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/django")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 2 {
		t.Fatalf("route node count = %d, want 2; got %+v", len(routeNodes), routeNodes)
	}

	edge := findRouteEdge(t, r, routeNodes, "ANY", "users/<int:id>")
	if edge == nil {
		t.Fatal("no route->handler edge found for users/<int:id>")
	}
	handler, err := r.GetNode(edge.Target)
	if err != nil {
		t.Fatalf("GetNode(handler): %v", err)
	}
	if handler.Name != "get_user" || handler.FilePath != "views.py" {
		t.Errorf("resolved handler = %s@%s, want get_user@views.py (cross-file resolution)", handler.Name, handler.FilePath)
	}
}

// TestFlask_RouteDetectionEndToEnd proves a full Run() over the Flask
// fixture commits route nodes for both `.route(methods=[...])` and a
// direct-verb decorator.
func TestFlask_RouteDetectionEndToEnd(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/flask")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 2 {
		t.Fatalf("route node count = %d, want 2; got %+v", len(routeNodes), routeNodes)
	}
	if e := findRouteEdge(t, r, routeNodes, "GET", "/users/<id>"); e == nil {
		t.Error("no route->handler edge found for GET /users/<id>")
	}
	if e := findRouteEdge(t, r, routeNodes, "POST", "/users"); e == nil {
		t.Error("no route->handler edge found for POST /users")
	}
}

// TestFastAPI_RouteDetectionEndToEnd proves a full Run() over the FastAPI
// fixture commits a route node for its direct-verb decorator.
func TestFastAPI_RouteDetectionEndToEnd(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/fastapi")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 1 {
		t.Fatalf("route node count = %d, want 1; got %+v", len(routeNodes), routeNodes)
	}
	if e := findRouteEdge(t, r, routeNodes, "GET", "/items/{id}"); e == nil {
		t.Error("no route->handler edge found for GET /items/{id}")
	}
}

// TestExpress_RouteDetectionEndToEnd proves a full Run() over the Express
// fixture commits route nodes for its app.get/app.post calls.
func TestExpress_RouteDetectionEndToEnd(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/express")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 2 {
		t.Fatalf("route node count = %d, want 2; got %+v", len(routeNodes), routeNodes)
	}
	if e := findRouteEdge(t, r, routeNodes, "GET", "/users/:id"); e == nil {
		t.Error("no route->handler edge found for GET /users/:id")
	}
	if e := findRouteEdge(t, r, routeNodes, "POST", "/users"); e == nil {
		t.Error("no route->handler edge found for POST /users")
	}
}

// TestNest_RouteDetectionEndToEnd proves a full Run() over the NestJS
// fixture commits route nodes for its @Get/@Post method decorators.
func TestNest_RouteDetectionEndToEnd(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/nestjs")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 2 {
		t.Fatalf("route node count = %d, want 2; got %+v", len(routeNodes), routeNodes)
	}
	getEdge := findRouteEdge(t, r, routeNodes, "GET", ":id")
	if getEdge == nil {
		t.Fatal("no route->handler edge found for GET :id")
	}
	if getEdge.Metadata["synthesizedBy"] != "nestjs-route" {
		t.Errorf("GET route edge synthesizedBy = %q, want nestjs-route", getEdge.Metadata["synthesizedBy"])
	}
	handler, err := r.GetNode(getEdge.Target)
	if err != nil {
		t.Fatalf("GetNode(handler): %v", err)
	}
	if handler.Name != "getUser" {
		t.Errorf("resolved handler Name = %q, want getUser", handler.Name)
	}
}
