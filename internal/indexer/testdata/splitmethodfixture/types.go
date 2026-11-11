package splitmethod

// Greeter is satisfied by Base through its method in this file.
type Greeter interface {
	Hello() string
}

type Base struct{}

func (Base) Hello() string { return "hello" }

// Derived gets its Run and Check methods from run.go.
type Derived struct {
	Base
	inner holder
}

type holder struct {
	value unknown
}
