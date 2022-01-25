package main

import (
	"ksubdomain/core/gologger"
	"ksubdomain/runner"
)

func main() {
	options := runner.ParseOptions()

	r, err := runner.New(options)
	if err != nil {
		gologger.Fatalf("%s\n", err.Error())
		return
	}
	r.RunEnumeration()
	r.Close()
}
