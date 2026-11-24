package speedtest

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// LinkServerID marks a result measured on the link to a tunnel's exit server, not against speedtest.net.
const LinkServerID = "link"

const (
	linkPing     byte = 'p'
	linkDownload byte = 'd'
	linkUpload   byte = 'u'

	linkMaxSession = 40 * time.Second
	linkMaxConns   = 64
	linkPingWait   = 10 * time.Second
)

// ServeLink answers speed tests on ln until ctx ends: one command byte, then echoes (ping), sends (download)
// or discards (upload) until the client hangs up or the session limit passes.
func ServeLink(ctx context.Context, ln net.Listener) {
	stop := context.AfterFunc(ctx, func() { _ = ln.Close() })
	defer stop()
	slots := make(chan struct{}, linkMaxConns)
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		select {
		case slots <- struct{}{}:
		default:
			_ = conn.Close()
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			serveLinkConn(ctx, conn)
		}()
	}
}

func serveLinkConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	_ = conn.SetDeadline(time.Now().Add(linkMaxSession))
	var cmd [1]byte
	if _, err := io.ReadFull(conn, cmd[:]); err != nil {
		return
	}
	switch cmd[0] {
	case linkPing:
		var b [1]byte
		for {
			if _, err := io.ReadFull(conn, b[:]); err != nil {
				return
			}
			if _, err := conn.Write(b[:]); err != nil {
				return
			}
		}
	case linkDownload:
		chunk := randomChunk()
		for {
			if _, err := conn.Write(chunk); err != nil {
				return
			}
		}
	case linkUpload:
		_, _ = io.Copy(io.Discard, conn)
	}
}

// dialLink opens a session with the exit end's responder; release closes it when ctx ends or the caller is done.
func dialLink(ctx context.Context, cfg Config, cmd byte) (net.Conn, func(), error) {
	dial := cfg.Dial
	if dial == nil {
		dial = (&net.Dialer{Timeout: probeTimeout}).DialContext
	}
	conn, err := dial(ctx, "tcp", cfg.LinkAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("the exit server does not answer the speed test (it needs a Dalan version with this test): %w", err)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	release := func() { stop(); _ = conn.Close() }
	if _, err := conn.Write([]byte{cmd}); err != nil {
		release()
		return nil, nil, fmt.Errorf("the exit server does not answer the speed test: %w", err)
	}
	return conn, release, nil
}

func runLink(ctx context.Context, cfg Config, emit func(Update)) (Result, error) {
	res := Result{Server: ServerInfo{ID: LinkServerID, Name: "tunnel link", Host: cfg.LinkAddr}}
	emit(Update{Phase: PhasePing})
	samples, err := linkPings(ctx, cfg, func(ms float64) { emit(Update{Phase: PhasePing, PingMs: ms}) })
	if err != nil {
		return Result{}, err
	}
	res.PingMs = minOf(samples)
	res.JitterMs = jitter(samples)

	emit(Update{Phase: PhaseDownload})
	if res.DownMbps, err = measure(ctx, cfg, PhaseDownload, linkWorker(cfg, linkDownload), emit); err != nil {
		return Result{}, err
	}
	emit(Update{Phase: PhaseUpload})
	if res.UpMbps, err = measure(ctx, cfg, PhaseUpload, linkWorker(cfg, linkUpload), emit); err != nil {
		return Result{}, err
	}
	res.At = time.Now().UnixMilli()
	return res, nil
}

func linkPings(ctx context.Context, cfg Config, each func(float64)) ([]float64, error) {
	conn, release, err := dialLink(ctx, cfg, linkPing)
	if err != nil {
		return nil, err
	}
	defer release()
	samples := make([]float64, 0, cfg.PingSamples)
	var b [1]byte
	for len(samples) < cfg.PingSamples {
		_ = conn.SetDeadline(time.Now().Add(linkPingWait))
		start := time.Now()
		if _, err := conn.Write(b[:]); err != nil {
			return nil, linkErr(ctx, err)
		}
		if _, err := io.ReadFull(conn, b[:]); err != nil {
			return nil, linkErr(ctx, err)
		}
		ms := float64(time.Since(start).Microseconds()) / 1000
		samples = append(samples, ms)
		each(ms)
	}
	return samples, nil
}

func linkErr(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("the exit server does not answer the speed test: %w", err)
}

// linkWorker moves bytes over sessions with the responder: reads for a download, writes for an upload.
func linkWorker(cfg Config, cmd byte) worker {
	chunk := randomChunk()
	return func(ctx context.Context, moved *atomic.Int64) {
		buf := make([]byte, 64<<10)
		for ctx.Err() == nil {
			conn, release, err := dialLink(ctx, cfg, cmd)
			if err != nil {
				pause(ctx)
				continue
			}
			for {
				var n int
				if cmd == linkDownload {
					n, err = conn.Read(buf)
				} else {
					n, err = conn.Write(chunk)
				}
				moved.Add(int64(n))
				if err != nil {
					break
				}
			}
			release()
		}
	}
}
