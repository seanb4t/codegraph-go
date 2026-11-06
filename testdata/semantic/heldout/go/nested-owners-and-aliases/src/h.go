package held

type Reader interface {
	Read() int
}

type ID = int

type left struct{ Value int }

func (l left) value() int { return l.Value }

type right struct{ Value int }

func (r right) value() int { return r.Value + helper() }

func helper() int { return 1 }

const limit, Max = 1, 2
