package main

import (
	"ksubdomain/core/gologger"
	options2 "ksubdomain/core/options"
	"ksubdomain/runner"
)

func main() {
	options := options2.ParseOptions()

	r, err := runner.New(options)
	if err != nil {
		gologger.Fatalf("%s\n", err.Error())
		return
	}
	r.RunEnumeration()
	r.Close()
}
