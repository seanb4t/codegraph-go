package pkg

// CallDescribe calls Widget.Describe — resolving a method whose receiver
// type is declared in a third file (pkg/types.go), neither this file nor
// pkg/methods.go.
func CallDescribe(w Widget) string {
	return w.Describe()
}
