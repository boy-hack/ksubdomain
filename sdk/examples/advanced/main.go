package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
	// Advanced configuration.
	// Note: Timeout field no longer exists — the scanner uses a dynamic
	// RTT-based timeout with a hardcoded upper bound of 10 s.
	scanner := sdk.NewScanner(&sdk.Config{
		Bandwidth:      "10m",
		Retry:          5,
		Resolvers:      []string{"8.8.8.8", "1.1.1.1"},
		Predict:        true,
		WildcardFilter: "advanced",
		Silent:         false,
	})

	// Context with a hard wall-clock limit for the whole scan.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Println("🚀 Starting advanced scan...")
	start := time.Now()

	results, err := scanner.EnumWithContext(ctx, "example.com")
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			fmt.Println("⏰ Scan reached 2-minute wall-clock limit; showing partial results")
		case errors.Is(err, sdk.ErrPermissionDenied):
			log.Fatal("permission denied — run with sudo or grant CAP_NET_RAW")
		default:
			log.Fatal(err)
		}
	}

	elapsed := time.Since(start)

	// Aggregate stats
	typeCount := make(map[string]int)
	for _, r := range results {
		typeCount[r.Type]++
	}

	fmt.Printf("\n📊 Scan Results:\n")
	fmt.Printf("   Total:  %d subdomains\n", len(results))
	fmt.Printf("   Time:   %v\n", elapsed.Round(time.Millisecond))
	if elapsed.Seconds() > 0 {
		fmt.Printf("   Speed:  %.0f domains/s\n", float64(len(results))/elapsed.Seconds())
	}

	fmt.Println("\n📋 Record Types:")
	for recType, count := range typeCount {
		fmt.Printf("   %s: %d\n", recType, count)
	}

	fmt.Println("\n✅ Discovered Subdomains (first 10):")
	for i, r := range results {
		if i >= 10 {
			fmt.Printf("   ... and %d more\n", len(results)-10)
			break
		}
		fmt.Printf("   %-40s [%-5s] %v\n", r.Domain, r.Type, r.Records)
	}
}
