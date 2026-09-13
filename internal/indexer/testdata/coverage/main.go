package main

// Hello is the fixture's one exported symbol — carries at least one node
// so indexed == 1 has something real to count (Phase 10 D-13).
func Hello() string {
	return "hello"
}

func main() {
	println(Hello())
}
