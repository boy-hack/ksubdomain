// Package errors defines sentinel error variables for ksubdomain.
//
// SDK users can test errors with errors.Is:
//
//	_, err := sdk.NewScanner(cfg).Enum("example.com")
//	if errors.Is(err, kserrors.ErrPermissionDenied) {
//	    log.Fatal("run with sudo")
//	}
package errors

import "errors"

// Device / pcap errors
var (
	// ErrPermissionDenied is returned when the process lacks privileges to
	// open the network interface (e.g., missing CAP_NET_RAW or not root).
	ErrPermissionDenied = errors.New("permission denied: run with sudo or grant CAP_NET_RAW")

	// ErrDeviceNotFound is returned when the specified network interface does
	// not exist on the current host.
	ErrDeviceNotFound = errors.New("network device not found")

	// ErrDeviceNotActive is returned when the network interface exists but is
	// not up/active.
	ErrDeviceNotActive = errors.New("network device not active")

	// ErrPcapInit is returned when libpcap/npcap fails to initialise for a
	// reason other than the above (catch-all).
	ErrPcapInit = errors.New("pcap initialisation failed")
)

// Runner / domain channel errors
var (
	// ErrDomainChanNil is returned when the domain input channel is nil or
	// uninitialized.
	ErrDomainChanNil = errors.New("domain channel is nil")
)
