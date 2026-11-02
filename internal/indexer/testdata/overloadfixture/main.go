package main

import "example.com/overloadfixture/util"

type parser struct{}

func (parser) Run() {}

func main() {
	util.Parse("x")
	var p parser
	p.Run()
}
