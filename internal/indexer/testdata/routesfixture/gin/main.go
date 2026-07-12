package main

func getUserHandler() {}

func createUserHandler() {}

type Router struct{}

func (r *Router) Group(path string) *Router  { return r }
func (r *Router) GET(path string, h func())  {}
func (r *Router) POST(path string, h func()) {}

func setupRouter() *Router { return &Router{} }

func main() {
	r := setupRouter()
	v1 := r.Group("/v1")
	v1.GET("/users/:id", getUserHandler)
	v1.POST("/users", createUserHandler)
}
