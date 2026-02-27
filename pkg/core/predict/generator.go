package predict

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"
)

//go:embed data/regular.cfg
var cfg string

//go:embed data/regular.dict
var dict string

// DomainGenerator is used to generate predicted domain names
type DomainGenerator struct {
	categories map[string][]string // stores block categories and their values
	patterns   []string            // domain name combination patterns
	subdomain  string              // subdomain part
	domain     string              // root domain part
	output     chan string         // output channel
	count      int                 // count of generated domain names
	mu         sync.Mutex          // mutex to protect count and output
}

// NewDomainGenerator creates a new domain name generator
func NewDomainGenerator(output chan string) (*DomainGenerator, error) {
	// Create generator instance
	dg := &DomainGenerator{
		categories: make(map[string][]string),
		output:     output,
	}

	// Load category dictionary
	if err := dg.loadDictionary(); err != nil {
		return nil, fmt.Errorf("failed to load dictionary file: %v", err)
	}

	// Load config patterns
	if err := dg.loadPatterns(); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	return dg, nil
}

// loadDictionary loads category information from the dictionary file
func (dg *DomainGenerator) loadDictionary() error {
	scanner := bufio.NewScanner(strings.NewReader(dict))
	var currentCategory string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check if it's a category identifier [category]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentCategory = line[1 : len(line)-1]
			dg.categories[currentCategory] = []string{}
		} else if currentCategory != "" {
			// If there's a current category, add the value
			dg.categories[currentCategory] = append(dg.categories[currentCategory], line)
		}
	}

	return scanner.Err()
}

// loadPatterns loads domain generation patterns from the config file
func (dg *DomainGenerator) loadPatterns() error {
	scanner := bufio.NewScanner(strings.NewReader(cfg))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			dg.patterns = append(dg.patterns, line)
		}
	}

	return scanner.Err()
}

// SetBaseDomain sets the base domain
func (dg *DomainGenerator) SetBaseDomain(domain string) {
	// Split subdomain and root domain
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		// Only root domain (example.com)
		dg.subdomain = ""
		dg.domain = domain
	} else {
		// Has subdomain (sub.example.com)
		dg.subdomain = parts[0]
		dg.domain = strings.Join(parts[1:], ".")
	}
}

// GenerateDomains generates predicted domain names and outputs them in real time
func (dg *DomainGenerator) GenerateDomains() int {
	dg.mu.Lock()
	dg.count = 0
	dg.mu.Unlock()

	// If no subdomain is set, return immediately
	if dg.subdomain == "" && dg.domain == "" {
		return 0
	}

	// Iterate over all patterns
	for _, pattern := range dg.patterns {
		// Recursively process tag replacements in each pattern
		dg.processPattern(pattern, map[string]string{
			"subdomain": dg.subdomain,
			"domain":    dg.domain,
		})
	}

	dg.mu.Lock()
	result := dg.count
	dg.mu.Unlock()
	return result
}

// processPattern recursively processes tag replacements in a pattern
func (dg *DomainGenerator) processPattern(pattern string, replacements map[string]string) {
	// Find the first tag
	startIdx := strings.Index(pattern, "{")
	if startIdx == -1 {
		// No more tags, output the final result
		if pattern != "" && dg.output != nil {
			dg.mu.Lock()
			dg.output <- pattern
			dg.count++
			dg.mu.Unlock()
		}
		return
	}

	endIdx := strings.Index(pattern, "}")
	if endIdx == -1 || endIdx < startIdx {
		// Malformed tag, return directly
		return
	}

	// Extract tag name
	tagName := pattern[startIdx+1 : endIdx]

	// Check if a replacement value already exists
	if value, exists := replacements[tagName]; exists {
		// Already has a replacement value, substitute and continue processing
		newPattern := pattern[:startIdx] + value + pattern[endIdx+1:]
		dg.processPattern(newPattern, replacements)
		return
	}

	// Get replacement values from category
	values, exists := dg.categories[tagName]
	if !exists || len(values) == 0 {
		// No replacement value found, skip this tag
		newPattern := pattern[:startIdx] + pattern[endIdx+1:]
		dg.processPattern(newPattern, replacements)
		return
	}

	// Recursively process each possible replacement value
	for _, value := range values {
		// Create a new replacement map
		newReplacements := make(map[string]string)
		for k, v := range replacements {
			newReplacements[k] = v
		}
		newReplacements[tagName] = value

		// Replace current tag and continue processing
		newPattern := pattern[:startIdx] + value + pattern[endIdx+1:]
		dg.processPattern(newPattern, newReplacements)
	}
}

// PredictDomains predicts possible domain name variants for a given domain and outputs them directly
func PredictDomains(domain string, output chan string) (int, error) {
	// Check if output channel is nil
	if output == nil {
		return 0, fmt.Errorf("output channel cannot be nil")
	}

	// Create domain generator
	generator, err := NewDomainGenerator(output)
	if err != nil {
		return 0, err
	}

	// Set base domain
	generator.SetBaseDomain(domain)

	// Generate predicted domains and return the count
	return generator.GenerateDomains(), nil
}
