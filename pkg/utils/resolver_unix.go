//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const resolvConfPath = "/etc/resolv.conf"

// GetSystemDefaultDNS retrieves the default DNS servers configured on Unix-like systems.
// It parses the /etc/resolv.conf file.
func GetSystemDefaultDNS() ([]string, error) {
	file, err := os.Open(resolvConfPath)
	if err != nil {
		// Consider ENOENT specifically? Maybe not necessary for this function.
		return nil, fmt.Errorf("failed to open %s: %w", resolvConfPath, err)
	}
	defer file.Close()

	servers := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Look for nameserver lines
		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// Basic validation could be added here (e.g., is it a valid IP?)
				servers = append(servers, fields[1])
			}
		}
		// Potentially handle 'search' and 'options' lines if needed in the future
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", resolvConfPath, err)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no nameservers found in %s", resolvConfPath)
	}

	return servers, nil
}
