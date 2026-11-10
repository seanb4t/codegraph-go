package splitmethod

// Run calls Hello, which Derived has only through the Base it embeds.
func (d Derived) Run() string {
	return d.Hello()
}

// Check calls Hello through a chain of fields whose type no file declares.
// The call resolves only in the conformance retry: the retry finds that
// Derived owns Check, through the contains edge that crosses the two
// files, and then finds Hello on the Base that Derived embeds.
func (d Derived) Check() string {
	return d.inner.value.Hello()
}
