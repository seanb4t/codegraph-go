package util

func Helper() int { return 1 }

type Thing struct{}

func (Thing) Run() int { return Helper() }
