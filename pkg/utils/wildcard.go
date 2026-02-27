package utils

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"sort"
	"strings"
)

type Pair struct {
	Key   string
	Value int
}
type PairList []Pair

func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value > p[j].Value }

// A function to turn a map into a PairList, then sort and return it.
func sortMapByValue(m map[string]int) PairList {
	p := make(PairList, len(m))
	i := 0
	for k, v := range m {
		p[i] = Pair{k, v}
		i++
	}
	sort.Sort(p)
	return p
}

// WildFilterOutputResult filters wildcard DNS results
func WildFilterOutputResult(outputType string, results []result.Result) []result.Result {
	if outputType == "none" {
		return results
	} else if outputType == "basic" {
		return FilterWildCard(results)
	} else if outputType == "advanced" {
		return FilterWildCardAdvanced(results)
	}
	return nil
}

// FilterWildCard filters wildcard DNS results based on Result type data.
// Input: []result.Result, Output: filtered []result.Result.
// Analyzes overall results and applies threshold-based filtering on repeated IP addresses.
func FilterWildCard(results []result.Result) []result.Result {
	if len(results) == 0 {
		return results
	}

	gologger.Debugf("Processing wildcard filter, total %d records...\n", len(results))

	// Count IP occurrence frequency
	ipFrequency := make(map[string]int)
	// Record IP-to-domain mappings
	ipToDomains := make(map[string][]string)
	// Total domain count
	totalDomains := len(results)

	// First pass: count IP frequency
	for _, res := range results {
		for _, answer := range res.Answers {
			// Skip non-IP records (CNAME, etc.)
			if !strings.HasPrefix(answer, "CNAME ") && !strings.HasPrefix(answer, "NS ") &&
				!strings.HasPrefix(answer, "TXT ") && !strings.HasPrefix(answer, "PTR ") {
				ipFrequency[answer]++
				ipToDomains[answer] = append(ipToDomains[answer], res.Subdomain)
			}
		}
	}

	// Sort IPs by occurrence frequency
	sortedIPs := sortMapByValue(ipFrequency)

	// Determine suspicious wildcard IP list.
	// Uses two criteria:
	// 1. IP resolution exceeds a certain percentage of total domains (dynamic threshold)
	// 2. Number of subdomains resolved by this IP exceeds a specific threshold
	suspiciousIPs := make(map[string]bool)

	for _, pair := range sortedIPs {
		ip := pair.Key
		count := pair.Value

		// Calculate the percentage of total domains this IP resolves
		percentage := float64(count) / float64(totalDomains) * 100

		// Dynamic threshold: adjust based on total domain count.
		// Higher threshold for fewer domains, lower threshold for more domains.
		var threshold float64
		if totalDomains < 100 {
			threshold = 30 // If total domains < 100, threshold is 30%
		} else if totalDomains < 1000 {
			threshold = 20 // If total domains 100-1000, threshold is 20%
		} else {
			threshold = 10 // If total domains > 1000, threshold is 10%
		}

		// Absolute count threshold
		absoluteThreshold := 70

		// Mark as suspicious if threshold is exceeded
		if percentage > threshold || count > absoluteThreshold {
			gologger.Debugf("Suspicious wildcard IP detected: %s (resolved %d domains, %.2f%%)\n",
				ip, count, percentage)
			suspiciousIPs[ip] = true
		}
	}

	// Second pass: filter results
	var filteredResults []result.Result

	for _, res := range results {
		// Check if all IPs for this domain are suspicious.
		// Keep the record if there is at least one non-suspicious IP.
		validRecord := false
		var filteredAnswers []string

		for _, answer := range res.Answers {
			// Keep all non-IP records (e.g., CNAME)
			if strings.HasPrefix(answer, "CNAME ") || strings.HasPrefix(answer, "NS ") ||
				strings.HasPrefix(answer, "TXT ") || strings.HasPrefix(answer, "PTR ") {
				validRecord = true
				filteredAnswers = append(filteredAnswers, answer)
			} else if !suspiciousIPs[answer] {
				// Keep IPs not in the suspicious list
				validRecord = true
				filteredAnswers = append(filteredAnswers, answer)
			}
		}

		if validRecord && len(filteredAnswers) > 0 {
			filteredRes := result.Result{
				Subdomain: res.Subdomain,
				Answers:   filteredAnswers,
			}
			filteredResults = append(filteredResults, filteredRes)
		}
	}

	gologger.Infof("Wildcard filtering completed, filtered %d valid records from %d total\n",
		len(filteredResults), totalDomains)

	return filteredResults
}

// FilterWildCardAdvanced provides a more advanced wildcard detection algorithm.
// Uses multiple heuristics and feature detection to identify wildcard DNS.
func FilterWildCardAdvanced(results []result.Result) []result.Result {
	if len(results) == 0 {
		return results
	}

	gologger.Debugf("Advanced wildcard detection started, total %d records...\n", len(results))

	// Count IP occurrence frequency
	ipFrequency := make(map[string]int)
	// Count the variety of subdomain prefixes resolved by each IP
	ipPrefixVariety := make(map[string]map[string]bool)
	// Count the variety of TLDs resolved by each IP
	ipTLDVariety := make(map[string]map[string]bool)
	// Record IP-to-domain mappings
	ipToDomains := make(map[string][]string)
	// Record CNAME information
	cnameRecords := make(map[string][]string)

	totalDomains := len(results)

	// First round: collect statistics
	for _, res := range results {
		subdomain := res.Subdomain
		parts := strings.Split(subdomain, ".")

		// Extract TLD and prefix
		prefix := ""
		tld := ""
		if len(parts) > 1 {
			prefix = parts[0]
			tld = strings.Join(parts[1:], ".")
		} else {
			prefix = subdomain
			tld = subdomain
		}

		for _, answer := range res.Answers {
			if strings.HasPrefix(answer, "CNAME ") {
				// Extract CNAME target
				cnameParts := strings.SplitN(answer, " ", 2)
				if len(cnameParts) == 2 {
					cnameTarget := cnameParts[1]
					cnameRecords[subdomain] = append(cnameRecords[subdomain], cnameTarget)
				}
				continue
			}

			// Process IP records only
			if !strings.HasPrefix(answer, "NS ") &&
				!strings.HasPrefix(answer, "TXT ") &&
				!strings.HasPrefix(answer, "PTR ") {
				// Count IP frequency
				ipFrequency[answer]++

				// Initialize prefix and TLD sets for this IP
				if ipPrefixVariety[answer] == nil {
					ipPrefixVariety[answer] = make(map[string]bool)
				}
				if ipTLDVariety[answer] == nil {
					ipTLDVariety[answer] = make(map[string]bool)
				}

				// Record which prefixes and TLDs this IP resolves
				ipPrefixVariety[answer][prefix] = true
				ipTLDVariety[answer][tld] = true

				// Record IP-to-domain mapping
				ipToDomains[answer] = append(ipToDomains[answer], subdomain)
			}
		}
	}

	// Sort IPs by frequency
	sortedIPs := sortMapByValue(ipFrequency)

	// Identify suspicious IPs
	suspiciousIPs := make(map[string]float64) // IP -> suspicion score (0-100)

	for _, pair := range sortedIPs {
		ip := pair.Key
		count := pair.Value

		// Initial suspicion score
		suspiciousScore := 0.0

		// Factor 1: IP frequency percentage
		freqPercentage := float64(count) / float64(totalDomains) * 100

		// Factor 2: prefix variety
		prefixVariety := len(ipPrefixVariety[ip])
		prefixVarietyRatio := float64(prefixVariety) / float64(count) * 100

		// Factor 3: TLD variety
		tldVariety := len(ipTLDVariety[ip])

		// Calculate suspicion score
		// 1. Frequency factor
		if freqPercentage > 30 {
			suspiciousScore += 40
		} else if freqPercentage > 10 {
			suspiciousScore += 20
		} else if freqPercentage > 5 {
			suspiciousScore += 10
		}

		// 2. Prefix variety factor
		// If an IP resolves many different subdomain prefixes, it may be CDN or wildcard DNS
		if prefixVarietyRatio > 90 && prefixVariety > 10 {
			suspiciousScore += 30
		} else if prefixVarietyRatio > 70 && prefixVariety > 5 {
			suspiciousScore += 20
		}

		// 3. Absolute count factor
		if count > 100 {
			suspiciousScore += 20
		} else if count > 50 {
			suspiciousScore += 10
		} else if count > 20 {
			suspiciousScore += 5
		}

		// 4. TLD variety factor - if an IP resolves multiple different TLDs, it's more likely legitimate
		if tldVariety > 3 {
			suspiciousScore -= 20
		} else if tldVariety > 1 {
			suspiciousScore -= 10
		}

		// Mark as suspicious only if the suspicion score exceeds the threshold
		if suspiciousScore >= 35 {
			gologger.Debugf("Suspicious IP: %s (resolved domains: %d, percentage: %.2f%%, prefix variety: %d/%d, suspicion score: %.2f)\n",
				ip, count, freqPercentage, prefixVariety, count, suspiciousScore)
			suspiciousIPs[ip] = suspiciousScore
		}
	}

	// Second round: filter results
	var filteredResults []result.Result

	// CNAME clustering analysis: detect multiple CNAME records pointing to the same target
	cnameTargetCount := make(map[string]int)
	for _, targets := range cnameRecords {
		for _, target := range targets {
			cnameTargetCount[target]++
		}
	}

	// Identify suspicious CNAME targets
	suspiciousCnames := make(map[string]bool)
	for cname, count := range cnameTargetCount {
		if count > 5 && float64(count)/float64(totalDomains)*100 > 10 {
			gologger.Debugf("Suspicious CNAME target: %s (pointed to %d times)\n", cname, count)
			suspiciousCnames[cname] = true
		}
	}

	// Filter results
	for _, res := range results {
		// Check if it contains a suspicious CNAME
		hasSuspiciousCname := false
		if targets, ok := cnameRecords[res.Subdomain]; ok {
			for _, target := range targets {
				if suspiciousCnames[target] {
					hasSuspiciousCname = true
					break
				}
			}
		}

		validRecord := !hasSuspiciousCname
		var filteredAnswers []string

		// Process all answers
		for _, answer := range res.Answers {
			isIP := !strings.HasPrefix(answer, "CNAME ") &&
				!strings.HasPrefix(answer, "NS ") &&
				!strings.HasPrefix(answer, "TXT ") &&
				!strings.HasPrefix(answer, "PTR ")

			// Keep all non-IP records but exclude suspicious CNAMEs
			if !isIP {
				if strings.HasPrefix(answer, "CNAME ") {
					cnameParts := strings.SplitN(answer, " ", 2)
					if len(cnameParts) == 2 && suspiciousCnames[cnameParts[1]] {
						continue // Skip suspicious CNAME
					}
				}
				validRecord = true
				filteredAnswers = append(filteredAnswers, answer)
			} else {
				// For IP records, filter based on suspicion score
				suspiciousScore, isSuspicious := suspiciousIPs[answer]

				// Keep if not in suspicious IP list or suspicion score is low
				if !isSuspicious || suspiciousScore < 50 {
					validRecord = true
					filteredAnswers = append(filteredAnswers, answer)
				}
			}
		}

		// Add only valid records
		if validRecord && len(filteredAnswers) > 0 {
			filteredRes := result.Result{
				Subdomain: res.Subdomain,
				Answers:   filteredAnswers,
			}
			filteredResults = append(filteredResults, filteredRes)
		}
	}

	gologger.Infof("Advanced wildcard filtering completed, filtered %d valid records from %d total\n",
		len(filteredResults), totalDomains)

	return filteredResults
}
