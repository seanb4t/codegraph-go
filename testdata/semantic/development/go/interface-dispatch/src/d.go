package dispatch

type Runner interface{ Run() int }

type A struct{}

func (A) Run() int { return 1 }

func Use(r Runner) int { return r.Run() }
