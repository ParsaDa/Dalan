package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// MaxRouteTestPorts bounds how many ports one route test may try.
const MaxRouteTestPorts = 20

// PortResult is what dialling one TCP port showed.
type PortResult struct {
	Port      int    `json:"port" example:"8443"`
	Reachable bool   `json:"reachable" example:"true"`
	LatencyMs int64  `json:"latencyMs" example:"96"`
	Error     string `json:"error" example:""`
}

// ValidatePorts checks a route test's port list.
func ValidatePorts(ports []int) error {
	if len(ports) == 0 || len(ports) > MaxRouteTestPorts {
		return fmt.Errorf("a route test takes between 1 and %d ports", MaxRouteTestPorts)
	}
	seen := map[int]bool{}
	for _, p := range ports {
		if p < 1 || p > 65535 {
			return fmt.Errorf("port %d is not between 1 and 65535", p)
		}
		if seen[p] {
			return fmt.Errorf("port %d is listed twice", p)
		}
		seen[p] = true
	}
	return nil
}

// DialPorts opens a TCP connection to each port of host in parallel and reports how each went.
func DialPorts(ctx context.Context, host string, ports []int, timeout time.Duration) ([]PortResult, error) {
	if host == "" {
		return nil, errors.New("a route test needs the address to dial")
	}
	if err := ValidatePorts(ports); err != nil {
		return nil, err
	}
	results := make([]PortResult, len(ports))
	var wg sync.WaitGroup
	for i, port := range ports {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = PortResult{Port: port}
			start := time.Now()
			dialer := net.Dialer{Timeout: timeout}
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
			if err != nil {
				results[i].Error = err.Error()
				return
			}
			_ = conn.Close()
			results[i].Reachable = true
			results[i].LatencyMs = time.Since(start).Milliseconds()
		}()
	}
	wg.Wait()
	return results, nil
}
