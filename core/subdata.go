package core

import (
	_ "embed"
	"strings"
)

//go:embed data/subnext.txt
var subnext string

//go:embed data/subdomain.txt
var subdomain string

func GetDefaultSubdomainData() []string {
	return strings.Split(subdomain, "\n")
}

func GetDefaultSubNextData() []string {
	return strings.Split(subnext, "\n")
}
