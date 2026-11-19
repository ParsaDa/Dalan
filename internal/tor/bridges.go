package tor

import (
	"fmt"
	"net"
	"strings"
)

var bridgeTransports = map[string]bool{"obfs4": true, "snowflake": true, "webtunnel": true, "meek_lite": true}

// ValidateBridges returns the bridge lines in text, trimmed and without blanks, or an error naming the first bad line.
// Each line is written after "Bridge " in the torrc, so a line may never carry a second directive.
func ValidateBridges(text string) ([]string, error) {
	var lines []string
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if err := checkBridgeLine(line); err != nil {
			return nil, fmt.Errorf("bridge line %d: %w", i+1, err)
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func checkBridgeLine(line string) error {
	for _, r := range line {
		if r < 0x20 && r != '\t' {
			return fmt.Errorf("it holds a control character")
		}
	}
	fields := strings.Fields(line)
	if bridgeTransports[fields[0]] {
		if len(fields) < 2 {
			return fmt.Errorf("it names %s but no address", fields[0])
		}
		return nil
	}
	host, _, err := net.SplitHostPort(fields[0])
	if err != nil || net.ParseIP(host) == nil {
		return fmt.Errorf("it starts with neither a transport (obfs4, snowflake, webtunnel, meek_lite) nor an address:port")
	}
	return nil
}
