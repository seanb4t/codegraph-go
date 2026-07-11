// skip_linux.go is excluded from a non-linux build context by its GOOS
// filename suffix; discovery tests use it to prove go/build.MatchFile
// filtering is honored.
package main
