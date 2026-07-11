package pkga

// Base is embedded by Derived to exercise struct embedding.
type Base struct {
	ID int
}

// Derived embeds Base and adds its own field.
type Derived struct {
	Base
	Extra string
}

// Reader is embedded by ReadWriter to exercise interface embedding.
type Reader interface {
	Read() int
}

// ReadWriter embeds Reader.
type ReadWriter interface {
	Reader
	Write(int)
}
