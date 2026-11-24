package carrier

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

const goos = runtime.GOOS

func geteuid() int { return os.Geteuid() }

const dialTimeout = 10 * time.Second

// Serve runs the relays until ctx ends. It fails at once when an address cannot be listened on, so the
// process exits and the supervisor shows why.
func Serve(ctx context.Context, relays []Relay) error {
	var listeners []net.Listener
	closeAll := func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}
	for _, r := range relays {
		l, err := (&net.ListenConfig{}).Listen(ctx, "tcp", r.Listen)
		if err != nil {
			closeAll()
			return err
		}
		listeners = append(listeners, l)
	}
	var wg sync.WaitGroup
	for i, l := range listeners {
		wg.Add(1)
		go func() {
			defer wg.Done()
			accept(ctx, l, relays[i].Dial)
		}()
	}
	<-ctx.Done()
	closeAll()
	wg.Wait()
	return nil
}

func accept(ctx context.Context, l net.Listener, dial string) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go carry(ctx, conn, dial)
	}
}

func carry(ctx context.Context, in net.Conn, dial string) {
	defer in.Close()
	out, err := (&net.Dialer{Timeout: dialTimeout}).DialContext(ctx, "tcp", dial)
	if err != nil {
		return
	}
	defer out.Close()
	stop := context.AfterFunc(ctx, func() { _ = in.Close(); _ = out.Close() })
	defer stop()
	done := make(chan struct{}, 2)
	pipe := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		if tcp, ok := dst.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		done <- struct{}{}
	}
	go pipe(out, in)
	go pipe(in, out)
	<-done
	<-done
}

// Run makes the interface, serves the relays on it and removes the interface when ctx ends or a relay fails.
func Run(ctx context.Context, r Runner, c Config) error {
	if err := Up(ctx, r, c); err != nil {
		return err
	}
	defer func() { _ = Down(context.WithoutCancel(ctx), r, c) }()
	err := Serve(ctx, c.Relays)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
