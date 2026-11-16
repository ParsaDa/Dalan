package tunnel

import (
	"net"
	"testing"
	"time"
)

func listenOnFreePort(t *testing.T) (net.Listener, int) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l, l.Addr().(*net.TCPAddr).Port
}

func TestDialPortsSeparatesAnOpenPortFromAClosedOne(t *testing.T) {
	_, open := listenOnFreePort(t)
	closedListener, closed := listenOnFreePort(t)
	_ = closedListener.Close()

	results, err := DialPorts("127.0.0.1", []int{closed, open}, time.Second)
	if err != nil {
		t.Fatalf("DialPorts: %v", err)
	}
	if len(results) != 2 || results[0].Port != closed || results[1].Port != open {
		t.Fatalf("results = %+v, want one per port in the order asked", results)
	}
	if results[0].Reachable || results[0].Error == "" {
		t.Fatalf("closed port = %+v, want unreachable with a reason", results[0])
	}
	if !results[1].Reachable || results[1].Error != "" {
		t.Fatalf("open port = %+v, want reachable", results[1])
	}
}

func TestDialPortsRefusesWhatCannotBeAPortListOrIsTooLong(t *testing.T) {
	tooMany := make([]int, MaxRouteTestPorts+1)
	for i := range tooMany {
		tooMany[i] = 1000 + i
	}
	cases := map[string][]int{"none": nil, "zero": {0}, "above 65535": {70000}, "too many": tooMany, "repeated": {8443, 8443}}
	for name, ports := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DialPorts("127.0.0.1", ports, time.Second); err == nil {
				t.Fatalf("DialPorts(%v) accepted the list", ports)
			}
		})
	}
	if _, err := DialPorts("", []int{8443}, time.Second); err == nil {
		t.Fatal("an empty host was accepted")
	}
}
