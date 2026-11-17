// Package tor runs one Tor process per exit country and exposes each as a loopback SOCKS outbound for Xray.
package tor

import "strings"

const (
	// MaxExits caps the enabled exits, since every one is a Tor process holding tens of megabytes.
	MaxExits = 10
	// PortBase and PortLimit bound the loopback SOCKS ports exits use, clear of the panel-egress, probe and relay ports.
	PortBase  = 62700
	PortLimit = 62789
)

// NormalizeCountry lower-cases a two-letter country code and reports whether it is one.
func NormalizeCountry(code string) (string, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	if len(code) != 2 {
		return "", false
	}
	for _, r := range code {
		if r < 'a' || r > 'z' {
			return "", false
		}
	}
	return code, true
}

// OutboundTag is the Xray outbound an exit appears as; rules and balancers may target it.
func OutboundTag(country string) string { return "tor-" + country }
