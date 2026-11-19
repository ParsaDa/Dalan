package outbound

import (
	"fmt"
	"os"
	"strconv"
)

// OpenProxy starts a temporary Xray core whose only way out is the given outbound and returns the loopback
// SOCKS address in front of it. allOutbounds are the other outbounds a dialerProxy chain may name; close stops the core.
func (s *OutboundService) OpenProxy(chosen map[string]any, allOutbounds []any) (addr string, closeProxy func(), err error) {
	tag, _ := chosen["tag"].(string)
	if tag == "" {
		return "", nil, fmt.Errorf("the outbound has no tag")
	}
	ports, release, err := reserveLoopbackPorts(1)
	if err != nil {
		return "", nil, fmt.Errorf("reserve a port: %w", err)
	}
	defer release()
	cfg := buildBatchTestConfig([]*httpBatchItem{{tag: tag, outbound: chosen}}, allOutbounds, ports)
	configPath, err := createTestConfigPath()
	if err != nil {
		return "", nil, fmt.Errorf("create the core's config path: %w", err)
	}
	proc := newBatchProcess(cfg, configPath)
	closeProxy = func() {
		if proc.IsRunning() {
			_ = proc.Stop()
		}
		_ = os.Remove(configPath)
	}
	release()
	if err := proc.Start(); err != nil {
		closeProxy()
		return "", nil, fmt.Errorf("start the temporary core: %w", err)
	}
	if notReady := waitForPortsReady(proc, ports, batchPortsReadyTimeout); notReady != nil {
		closeProxy()
		return "", nil, notReady
	}
	return "127.0.0.1:" + strconv.Itoa(ports[0]), closeProxy, nil
}
