package output

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"github.com/logrusorgru/aurora"
)

// BeautifiedOutput 美化输出器
// 提供彩色、对齐、emoji 等美化功能
type BeautifiedOutput struct {
	windowsWidth int
	silent       bool
	onlyDomain   bool
	useColor     bool
	results      []result.Result
	mu           sync.Mutex
	startTime    time.Time
	typeCount    map[string]int
}

// NewBeautifiedOutput creates a beautified output writer
func NewBeautifiedOutput(silent bool, useColor bool, onlyDomain ...bool) (*BeautifiedOutput, error) {
	b := &BeautifiedOutput{
		windowsWidth: 120, // Default width
		silent:       silent,
		useColor:     useColor,
		results:      make([]result.Result, 0),
		startTime:    time.Now(),
		typeCount:    make(map[string]int),
	}

	if len(onlyDomain) > 0 {
		b.onlyDomain = onlyDomain[0]
	}

	return b, nil
}

// WriteDomainResult writes a single domain result with beautification
func (b *BeautifiedOutput) WriteDomainResult(r result.Result) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.results = append(b.results, r)

	recordType, displayRecords := parseAnswers(r.Answers)

	// Count by type
	b.typeCount[recordType]++

	// Format output
	var output string

	if b.onlyDomain {
		output = r.Subdomain
	} else {
		// Build formatted output
		domain := r.Subdomain
		recordsStr := strings.Join(displayRecords, ", ")

		if b.useColor {
			// Colorized output
			au := aurora.NewAurora(true)

			switch recordType {
			case "A", "AAAA":
				// Green for A/AAAA records
				output = fmt.Sprintf("%s %s %s",
					au.Green("✓").String(),
					au.Cyan(domain).String(),
					au.White(recordsStr).String())
			case "CNAME":
				// Blue for CNAME
				output = fmt.Sprintf("%s %s %s %s",
					au.Green("✓").String(),
					au.Cyan(domain).String(),
					au.Blue("[CNAME]").String(),
					au.White(recordsStr).String())
			case "NS":
				// Yellow for NS
				output = fmt.Sprintf("%s %s %s %s",
					au.Green("✓").String(),
					au.Cyan(domain).String(),
					au.Yellow("[NS]").String(),
					au.White(recordsStr).String())
			default:
				output = fmt.Sprintf("%s %s %s",
					au.Green("✓").String(),
					au.Cyan(domain).String(),
					au.White(recordsStr).String())
			}
		} else {
			// Plain output
			if recordType == "A" || recordType == "AAAA" {
				output = fmt.Sprintf("%-40s => %s", domain, recordsStr)
			} else {
				output = fmt.Sprintf("%-40s [%s] %s", domain, recordType, recordsStr)
			}
		}
	}

	if !b.silent {
		gologger.Silentf("%s\n", output)
	}

	return nil
}

// Close prints summary and closes the output
func (b *BeautifiedOutput) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.results) == 0 {
		return nil
	}

	elapsed := time.Since(b.startTime)

	// Print summary
	if b.useColor {
		au := aurora.NewAurora(true)
		fmt.Println()
		fmt.Println(au.Green("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━").String())
		fmt.Printf("%s Scan Summary\n", au.Bold("📊"))
		fmt.Println(au.Green("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━").String())
		fmt.Printf("  %s %s\n", au.Bold("Total Found:"), au.Green(len(b.results)))
		fmt.Printf("  %s %s\n", au.Bold("Time Elapsed:"), au.Cyan(elapsed.Round(time.Millisecond)))
		fmt.Printf("  %s %.0f domains/s\n", au.Bold("Speed:"), float64(len(b.results))/elapsed.Seconds())
		
		if len(b.typeCount) > 0 {
			fmt.Printf("\n  %s\n", au.Bold("Record Types:"))
			for recType, count := range b.typeCount {
				percentage := float64(count) / float64(len(b.results)) * 100
				fmt.Printf("    %s: %d (%.1f%%)\n", recType, count, percentage)
			}
		}
		fmt.Println(au.Green("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━").String())
	} else {
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("📊 Scan Summary")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("  Total Found:  %d\n", len(b.results))
		fmt.Printf("  Time Elapsed: %v\n", elapsed.Round(time.Millisecond))
		fmt.Printf("  Speed:        %.0f domains/s\n", float64(len(b.results))/elapsed.Seconds())
		
		if len(b.typeCount) > 0 {
			fmt.Println("\n  Record Types:")
			for recType, count := range b.typeCount {
				percentage := float64(count) / float64(len(b.results)) * 100
				fmt.Printf("    %s: %d (%.1f%%)\n", recType, count, percentage)
			}
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}

	return nil
}
