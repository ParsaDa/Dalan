package tunnel

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// DefaultProbeURL answers with the caller's address and country, which shows where a tunnel exits.
const DefaultProbeURL = "https://www.cloudflare.com/cdn-cgi/trace"

// Status is what one request sent through a tunnel revealed about it.
type Status struct {
	Connected bool   `json:"connected" example:"true"`
	LatencyMs int64  `json:"latencyMs" example:"182"`
	ExitIP    string `json:"exitIp" example:"203.0.113.77"`
	ExitLoc   string `json:"exitLoc" example:"DE"`
	Error     string `json:"error" example:""`
	// LanesUp, TargetLanes and LossPct come from a Rakhsh process; other engines leave them zero.
	LanesUp     int     `json:"lanesUp" example:"3"`
	TargetLanes int     `json:"targetLanes" example:"4"`
	LossPct     float64 `json:"lossPct" example:"0.4"`
	// History is the latency of each probe in the last hour, oldest first; 0 is a failed probe.
	History []int64 `json:"history,omitempty" example:"[40,0,38]"`
}

// ProbeVia sends one GET through the SOCKS port that is routed into a tunnel; a failure is a status, not an error.
func ProbeVia(socksAddr, url string, timeout time.Duration) Status {
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, &net.Dialer{Timeout: timeout})
	if err != nil {
		return Status{Error: err.Error()}
	}
	client := &http.Client{Timeout: timeout, Transport: &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Status{Error: err.Error()}
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return Status{Error: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	latency := time.Since(start).Milliseconds()
	if latency < 1 {
		latency = 1
	}
	ip, loc := parseTrace(string(body))
	return Status{Connected: true, LatencyMs: latency, ExitIP: ip, ExitLoc: loc}
}

func parseTrace(body string) (ip, loc string) {
	for _, line := range strings.Split(body, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "ip":
			ip = value
		case "loc":
			loc = value
		}
	}
	return ip, loc
}
