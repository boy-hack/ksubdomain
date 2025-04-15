//go:build windows

package utils

import (
	"fmt"
	"strings"

	// "os/exec" // No longer needed
	// "bytes" // No longer needed
	// "syscall" // No longer needed

	"github.com/StackExchange/wmi" // 需要添加这个依赖
)

// Win32_NetworkAdapterConfiguration WMI class structure (subset)
type win32_NetworkAdapterConfiguration struct {
	Description          string
	IPEnabled            bool
	DNSServerSearchOrder []string
	DNSHostName          string
	DefaultIPGateway     []string // Added to help filter relevant adapters
	DHCPEnabled          bool     // Added to help filter relevant adapters
}

// GetSystemDefaultDNS retrieves the default DNS servers configured on Windows systems using WMI.
func GetSystemDefaultDNS() ([]string, error) {
	var dst []win32_NetworkAdapterConfiguration
	// Query WMI for network adapter configurations that have IP enabled
	// We also filter for adapters that likely connect to the internet (have a gateway or are DHCP enabled)
	// This helps avoid virtual/loopback adapters that might pollute the results.
	query := "SELECT Description, IPEnabled, DNSServerSearchOrder, DNSHostName, DefaultIPGateway, DHCPEnabled FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled = TRUE AND (DHCPEnabled = TRUE OR DefaultIPGateway IS NOT NULL)"
	if err := wmi.Query(query, &dst); err != nil {
		return nil, fmt.Errorf("WMI query failed: %w", err)
	}

	servers := []string{}
	found := false

	for _, nic := range dst {
		// Skip adapters without DNS server entries
		if nic.DNSServerSearchOrder == nil || len(nic.DNSServerSearchOrder) == 0 {
			continue
		}

		// Basic filtering: Skip clearly virtual or loopback-like adapters based on description
		descLower := strings.ToLower(nic.Description)
		if strings.Contains(descLower, "loopback") || strings.Contains(descLower, "virtual") || strings.Contains(descLower, "pseudo") {
			continue
		}

		// Add unique DNS servers found
		for _, server := range nic.DNSServerSearchOrder {
			trimmedServer := strings.TrimSpace(server)
			// Add basic validation if needed (is it a valid IP?)
			if trimmedServer != "" && trimmedServer != "::" && !contains(servers, trimmedServer) {
				servers = append(servers, trimmedServer)
				found = true
			}
		}
		// Often, the first adapter with DNS configured is the primary one.
		// We could potentially stop after finding the first valid one, but aggregating
		// from all relevant adapters might be more robust in complex network setups.
	}

	if !found || len(servers) == 0 {
		// Fallback or specific error? Could try the ipconfig method as a last resort?
		// For now, return an error if WMI doesn't yield results.
		return nil, fmt.Errorf("no suitable network adapter with DNS configuration found via WMI")
	}

	return servers, nil
}

// contains checks if a slice contains a specific string.
// (Keep this helper function as it's still needed)
func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
