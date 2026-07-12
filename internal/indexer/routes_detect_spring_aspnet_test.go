package indexer

import "testing"

// TestSpring_RouteDetectionEndToEnd proves a full Run() over the Spring
// fixture commits route nodes for both a direct-verb annotation
// (@GetMapping) and the generic @RequestMapping, each edged to its own
// annotated method as the handler.
func TestSpring_RouteDetectionEndToEnd(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/spring")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 2 {
		t.Fatalf("route node count = %d, want 2; got %+v", len(routeNodes), routeNodes)
	}

	getEdge := findRouteEdge(t, r, routeNodes, "GET", "/users/{id}")
	if getEdge == nil {
		t.Fatal("no route->handler edge found for GET /users/{id}")
	}
	handler, err := r.GetNode(getEdge.Target)
	if err != nil {
		t.Fatalf("GetNode(handler): %v", err)
	}
	if handler.Name != "getUser" {
		t.Errorf("resolved handler Name = %q, want getUser", handler.Name)
	}

	postEdge := findRouteEdge(t, r, routeNodes, "POST", "/users")
	if postEdge == nil {
		t.Fatal("no route->handler edge found for POST /users (@RequestMapping)")
	}
	if postEdge.Metadata["synthesizedBy"] != "spring-route" {
		t.Errorf("POST route edge synthesizedBy = %q, want spring-route", postEdge.Metadata["synthesizedBy"])
	}
}

// TestAspNet_RouteDetectionEndToEnd proves a full Run() over the ASP.NET
// fixture commits route nodes for its Http<Verb> attributes.
func TestAspNet_RouteDetectionEndToEnd(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/aspnet")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 2 {
		t.Fatalf("route node count = %d, want 2; got %+v", len(routeNodes), routeNodes)
	}

	getEdge := findRouteEdge(t, r, routeNodes, "GET", "{id}")
	if getEdge == nil {
		t.Fatal("no route->handler edge found for GET {id}")
	}
	if getEdge.Metadata["synthesizedBy"] != "aspnet-route" {
		t.Errorf("GET route edge synthesizedBy = %q, want aspnet-route", getEdge.Metadata["synthesizedBy"])
	}
	handler, err := r.GetNode(getEdge.Target)
	if err != nil {
		t.Fatalf("GetNode(handler): %v", err)
	}
	if handler.Name != "GetUser" {
		t.Errorf("resolved handler Name = %q, want GetUser", handler.Name)
	}
}
