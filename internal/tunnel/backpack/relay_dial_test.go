package backpack

import (
	"context"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// socksRelay is a one-connection SOCKS5 server with password auth that records who logged in.
func socksRelay(t *testing.T) (port int, login chan [2]string) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	login = make(chan [2]string, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 262)
		io.ReadFull(c, buf[:2])
		io.ReadFull(c, buf[:int(buf[1])])
		c.Write([]byte{5, 2})
		io.ReadFull(c, buf[:2])
		ulen := int(buf[1])
		io.ReadFull(c, buf[:ulen+1])
		user := string(buf[:ulen])
		plen := int(buf[ulen])
		io.ReadFull(c, buf[:plen])
		login <- [2]string{user, string(buf[:plen])}
		c.Write([]byte{1, 0})
		io.ReadFull(c, buf[:4])
		var host string
		switch buf[3] {
		case 1:
			io.ReadFull(c, buf[:4])
			host = net.IP(buf[:4]).String()
		case 3:
			io.ReadFull(c, buf[:1])
			n := int(buf[0])
			io.ReadFull(c, buf[:n])
			host = string(buf[:n])
		}
		io.ReadFull(c, buf[:2])
		dest := net.JoinHostPort(host, strconv.Itoa(int(buf[0])<<8|int(buf[1])))
		up, err := net.Dial("tcp", dest)
		if err != nil {
			c.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
			return
		}
		defer up.Close()
		c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
		go io.Copy(up, c)
		io.Copy(c, up)
	}()
	return l.Addr().(*net.TCPAddr).Port, login
}

func TestRelayDialReachesTheFarEndThroughTheRelayWithTheTunnelsAccount(t *testing.T) {
	port, login := socksRelay(t)
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { target.Close() })
	go func() {
		c, err := target.Accept()
		if err == nil {
			io.Copy(c, c)
			c.Close()
		}
	}()
	tun := model.ServerTunnel{UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", RelayPort: port}

	conn, err := RelayDial(tun)(context.Background(), "tcp", target.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	echo := make([]byte, 4)
	if _, err := io.ReadFull(conn, echo); err != nil || string(echo) != "ping" {
		t.Fatalf("echo = %q, %v; want ping", echo, err)
	}
	user, pass := RelayCredentials(tun)
	if got := <-login; got != [2]string{user, pass} {
		t.Fatalf("relay saw login %v, want the tunnel's own account %v", got, [2]string{user, pass})
	}
}

func TestThroughputPortNeverLandsOnAnotherTunnelsRelayPortOrPastTheLastPort(t *testing.T) {
	for relay := RelayPortMin; relay <= RelayPortMax; relay++ {
		p := ThroughputPort(relay)
		if p >= RelayPortMin && p <= RelayPortMax {
			t.Fatalf("relay %d: speed test port %d is inside the relay range", relay, p)
		}
		if p > 65535 {
			t.Fatalf("relay %d: speed test port %d is not a port", relay, p)
		}
	}
}
