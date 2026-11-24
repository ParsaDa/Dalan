package benchmark

import (
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestSpecForAReverseMethodDialsThisServersAddress(t *testing.T) {
	c := Candidate{Engine: "gost", Transport: "grpc", Mode: "reverse"}
	got := SpecFor("bench-1", c, "203.0.113.10", 41000)
	want := Spec{Name: "bench-1", Address: "203.0.113.10", Port: 41000, Candidate: c}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("spec = %+v, want %+v", got, want)
	}
}

func TestSpecForADirectMethodLeavesTheAddressToTheNode(t *testing.T) {
	got := SpecFor("bench-2", Candidate{Engine: "singbox", Transport: "wss", Mode: "direct"}, "203.0.113.10", 41001)
	if got.Address != "" || got.Mode != "direct" || got.Port != 41001 {
		t.Fatalf("spec = %+v, want no address (the node's own is used) and mode direct", got)
	}
}

func TestCheckNeedsThisServersAddressOnlyForReverseMethods(t *testing.T) {
	reverse := Candidate{Engine: "gost", Transport: "tls", Mode: "reverse"}
	direct := Candidate{Engine: "gost", Transport: "tls", Mode: "direct"}
	for name, tc := range map[string]struct {
		address    string
		candidates []Candidate
		wantErr    string
	}{
		"reverse without an address": {"", []Candidate{reverse}, "address of this server"},
		"blank address":              {"  ", []Candidate{direct, reverse}, "address of this server"},
		"direct needs none":          {"", []Candidate{direct}, ""},
		"reverse with an address":    {"203.0.113.10", []Candidate{reverse, direct}, ""},
	} {
		err := CheckAddress(tc.address, tc.candidates)
		switch {
		case tc.wantErr == "" && err != nil:
			t.Errorf("%s: err = %v", name, err)
		case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
			t.Errorf("%s: err = %v, want it to mention %q", name, err, tc.wantErr)
		}
	}
}

func TestFreePortCanBeListenedOnRightAway(t *testing.T) {
	port, err := FreePort()
	if err != nil {
		t.Fatal(err)
	}
	if port < 1024 || port > 65535 {
		t.Fatalf("port = %d, want an unprivileged port", port)
	}
	l, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("the port FreePort returned is not free: %v", err)
	}
	l.Close()
}
