package server

import (
	"testing"
)

func TestListAddresses(t *testing.T) {
	addresses := listAddresses()

	// Should return at least one address (loopback)
	if len(addresses) == 0 {
		t.Error("listAddresses() returned empty slice, expected at least one address")
	}

	// Should contain loopback address
	hasLoopback := false
	for _, addr := range addresses {
		if addr == "127.0.0.1" || addr == "::1" {
			hasLoopback = true
			break
		}
	}

	if !hasLoopback {
		t.Error("listAddresses() should include loopback address (127.0.0.1 or ::1)")
	}
}

func TestListAddressesReturnsValidIPs(t *testing.T) {
	addresses := listAddresses()

	for _, addr := range addresses {
		// Each address should be a valid IP string (not empty)
		if addr == "" {
			t.Error("listAddresses() returned empty string address")
		}
	}
}
