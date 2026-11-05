package bounds

// 😀 é: four-byte and two-byte runes precede every declaration on their lines.
/* 😀 */ func Größe() int { return 1 }

const note = "func NotAFunction() {}" // func AlsoNot() {}

type Ä struct{}

func (Ä) Same() int { return Größe() }

type B struct{}

func (B) Same() int { return 2 }
