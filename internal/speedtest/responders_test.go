package speedtest

import (
	"net"
	"strings"
	"testing"
	"time"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

func reachable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func TestRespondersFollowTheAddressesTheyAreGiven(t *testing.T) {
	var r Responders
	t.Cleanup(func() { _ = r.Sync(nil) })
	a, b := freeAddr(t), freeAddr(t)

	if err := r.Sync([]string{a, b}); err != nil {
		t.Fatal(err)
	}
	if !reachable(a) || !reachable(b) {
		t.Fatal("both addresses must answer after they are added")
	}
	if err := r.Sync([]string{b}); err != nil {
		t.Fatal(err)
	}
	if reachable(a) {
		t.Fatal("an address that was dropped still answers")
	}
	if !reachable(b) {
		t.Fatal("an address that was kept stopped answering")
	}
	if err := r.Sync(nil); err != nil {
		t.Fatal(err)
	}
	if reachable(b) {
		t.Fatal("nothing should answer once the list is empty")
	}
}

func TestRespondersReportAnAddressTheyCannotTakeAndServeTheRest(t *testing.T) {
	var r Responders
	t.Cleanup(func() { _ = r.Sync(nil) })
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	free := freeAddr(t)

	err = r.Sync([]string{taken.Addr().String(), free})
	if err == nil || !strings.Contains(err.Error(), taken.Addr().String()) {
		t.Fatalf("err = %v, want the address that could not be taken", err)
	}
	if !reachable(free) {
		t.Fatal("one address failing must not stop the others")
	}
}
