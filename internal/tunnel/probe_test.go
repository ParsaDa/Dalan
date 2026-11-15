package tunnel

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseTraceReadsExitAddressAndCountry(t *testing.T) {
	ip, loc := parseTrace("fl=123\nh=www.cloudflare.com\nip=203.0.113.77\nts=1.5\nloc=DE\ntls=TLSv1.3\n")
	if ip != "203.0.113.77" || loc != "DE" {
		t.Fatalf("parseTrace = %q, %q; want 203.0.113.77, DE", ip, loc)
	}
	if ip, loc := parseTrace("<html>blocked</html>"); ip != "" || loc != "" {
		t.Fatalf("parseTrace of a non-trace body = %q, %q; want both empty", ip, loc)
	}
}

func TestProbeReportsADeadTunnelInsteadOfFailing(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := l.Addr().String()
	l.Close()

	status := ProbeVia(addr, "http://203.0.113.1/", 2*time.Second)
	if status.Connected || status.LatencyMs != 0 || !strings.Contains(status.Error, "refused") {
		t.Fatalf("status = %+v, want not connected with a connection-refused error", status)
	}
}
