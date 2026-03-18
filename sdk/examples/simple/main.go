package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
	// Create scanner with default configuration.
	// Timeout is managed automatically (dynamic RTT-based, upper bound 10 s).
	scanner := sdk.NewScanner(sdk.DefaultConfig)

	fmt.Println("Scanning example.com...")
	results, err := scanner.Enum("example.com")
	if err != nil {
		switch {
		case errors.Is(err, sdk.ErrPermissionDenied):
			log.Fatal("permission denied — run with sudo or grant CAP_NET_RAW")
		case errors.Is(err, sdk.ErrDeviceNotFound):
			log.Fatal("network device not found — check your interface name")
		default:
			log.Fatal(err)
		}
	}

	fmt.Printf("\nFound %d subdomains:\n\n", len(results))
	for _, r := range results {
		fmt.Printf("%-40s [%-5s] %s\n", r.Domain, r.Type, strings.Join(r.Records, ", "))
	}
}
