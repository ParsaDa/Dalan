package speedtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
)

// Responders keeps one speed test responder listening on each address it is given.
type Responders struct {
	mu      sync.Mutex
	running map[string]*responder
}

type responder struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// Sync starts responders for new addresses and stops those no longer listed. An address that cannot be taken
// is reported once all the others are in place.
func (r *Responders) Sync(addrs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running == nil {
		r.running = map[string]*responder{}
	}
	want := make(map[string]bool, len(addrs))
	for _, a := range addrs {
		want[a] = true
	}
	for a, resp := range r.running {
		if !want[a] {
			resp.cancel()
			<-resp.done
			delete(r.running, a)
		}
	}
	var errs []error
	for _, a := range addrs {
		if _, ok := r.running[a]; ok {
			continue
		}
		ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", a)
		if err != nil {
			errs = append(errs, fmt.Errorf("speed test responder on %s: %w", a, err))
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { ServeLink(ctx, ln); close(done) }()
		r.running[a] = &responder{cancel: cancel, done: done}
	}
	return errors.Join(errs...)
}
