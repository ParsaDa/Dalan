package speedtest

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func startResponder(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { ServeLink(ctx, ln); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return ln.Addr().String()
}

func linkConfig(addr string, dial func(context.Context, string, string) (net.Conn, error)) Config {
	return Config{
		LinkAddr: addr, Dial: dial, PingSamples: 4, Phase: 400 * time.Millisecond, Connections: 2,
	}
}

func TestRunMeasuresTheLinkToTheExitEndInsteadOfSpeedtestNet(t *testing.T) {
	addr := startResponder(t)
	var mu sync.Mutex
	var dialed []string
	dial := func(ctx context.Context, network, a string) (net.Conn, error) {
		mu.Lock()
		dialed = append(dialed, a)
		mu.Unlock()
		return (&net.Dialer{}).DialContext(ctx, network, a)
	}
	res, err := Run(context.Background(), linkConfig(addr, dial), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.PingMs <= 0 || res.DownMbps <= 0 || res.UpMbps <= 0 {
		t.Fatalf("ping, download and upload must all be measured, got %+v", res)
	}
	if res.Server.ID != LinkServerID {
		t.Fatalf("server id = %q, want %q", res.Server.ID, LinkServerID)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(dialed) == 0 {
		t.Fatal("the run never dialed through the dialer it was given")
	}
	for _, a := range dialed {
		if a != addr {
			t.Fatalf("dialed %q, want only the exit end's responder %q", a, addr)
		}
	}
}

func TestRunSaysTheExitEndDoesNotAnswerTheSpeedTest(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	dial := func(ctx context.Context, network, a string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, a)
	}
	_, err = Run(context.Background(), linkConfig(addr, dial), nil)
	if err == nil || !strings.Contains(err.Error(), "exit server does not answer the speed test") {
		t.Fatalf("err = %v, want the exit server to be named as not answering", err)
	}
}

func TestResponderStopsWhenItsContextEnds(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { ServeLink(ctx, ln); close(done) }()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte{linkDownload}); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the responder kept running after its context ended")
	}
	if _, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second); err == nil {
		t.Fatal("the listener still accepts after the responder stopped")
	}
}
