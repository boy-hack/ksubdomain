package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/sdk"
)

func main() {
	// Advanced configuration
	scanner := sdk.NewScanner(&sdk.Config{
		Bandwidth:      "10m",
		Retry:          5,
		Timeout:        10,
		Resolvers:      []string{"8.8.8.8", "1.1.1.1"},
		Predict:        true,
		WildcardFilter: "advanced",
		Silent:         false,
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start scanning
	fmt.Println("🚀 Starting advanced scan...")
	start := time.Now()

	results, err := scanner.EnumWithContext(ctx, "example.com")
	if err != nil {
		if err == context.DeadlineExceeded {
			fmt.Println("⏰ Scan timeout, showing partial results")
		} else {
			log.Fatal(err)
		}
	}

	elapsed := time.Since(start)

	// Statistics
	typeCount := make(map[string]int)
	for _, r := range results {
		typeCount[r.Type]++
	}

	// Print results
	fmt.Printf("\n📊 Scan Results:\n")
	fmt.Printf("   Total: %d subdomains\n", len(results))
	fmt.Printf("   Time: %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("   Speed: %.0f domains/s\n\n", float64(len(results))/elapsed.Seconds())

	fmt.Println("📋 Record Types:")
	for recType, count := range typeCount {
		fmt.Printf("   %s: %d\n", recType, count)
	}

	fmt.Println("\n✅ Discovered Subdomains:")
	for i, result := range results {
		if i >= 10 {
			fmt.Printf("   ... and %d more\n", len(results)-10)
			break
		}
		fmt.Printf("   %-30s [%s] %v\n", result.Domain, result.Type, result.Records)
	}
}
