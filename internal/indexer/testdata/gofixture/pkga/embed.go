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

// ID is a non-struct/non-interface type declaration, exercising the
// type_alias node kind (D-06) — distinct from Base/Derived's struct kind
// and Reader/ReadWriter's interface kind.
type ID = int
