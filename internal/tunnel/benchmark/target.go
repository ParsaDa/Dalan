package benchmark

import (
	"context"
	"errors"
	"net"
	"strings"
)

// Spec is the tunnel to make for one candidate; the caller turns it into its own create request.
type Spec struct {
	Name    string
	Address string
	Port    int
	Candidate
}

// SpecFor describes the temporary tunnel for a candidate. A reverse tunnel is dialled by the node, so it
// needs this server's address; a direct one is dialled from here and takes the node's own address.
func SpecFor(name string, c Candidate, address string, port int) Spec {
	if c.Mode == "direct" {
		address = ""
	}
	return Spec{Name: name, Address: address, Port: port, Candidate: c}
}

// CheckAddress refuses a run that has reverse methods but no address for the node to dial.
func CheckAddress(address string, candidates []Candidate) error {
	if strings.TrimSpace(address) != "" {
		return nil
	}
	for _, c := range candidates {
		if c.Mode == "reverse" {
			return errors.New("reverse methods need the address of this server, which the node dials")
		}
	}
	return nil
}

// FreePort returns a port nothing listens on at this moment.
func FreePort() (int, error) {
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
