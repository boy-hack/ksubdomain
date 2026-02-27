package main

import (
	"fmt"
	"log"

	"github.com/boy-hack/ksubdomain/v2/pkg/sdk"
)

func main() {
	// Create scanner with default configuration
	scanner := sdk.NewScanner(sdk.DefaultConfig)

	// Enumerate subdomains
	fmt.Println("Scanning example.com...")
	results, err := scanner.Enum("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Print results
	fmt.Printf("\nFound %d subdomains:\n\n", len(results))
	for _, result := range results {
		fmt.Printf("%-30s => %s\n", result.Domain, result.Records[0])
	}
}
