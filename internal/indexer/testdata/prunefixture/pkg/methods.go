package pkg

// Describe is a value-receiver method on Widget, whose type declaration
// lives in pkg/types.go.
func (w Widget) Describe() string {
	return w.Name
}
