package backpack

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestRenderConfigWritesTheListeningEndForAPortal(t *testing.T) {
	got, err := RenderConfig(model.ServerTunnel{
		Name: "de1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10",
		Transport: "wssmux", FallbackTransports: []string{"quic", "tcpmux"},
		ForwardPorts: []string{"443=127.0.0.1:2096", "8080"}, AcceptUDP: true,
	})
	if err != nil {
		t.Fatalf("RenderConfig: %v", err)
	}
	want := `[server]
bind_addr = "0.0.0.0:8443"
transport = "wssmux"
token = "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"
ports = ["443=127.0.0.1:2096", "8080"]
accept_udp = true
fallback_transports = ["quic", "tcpmux"]
`
	if got != want {
		t.Fatalf("config =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderConfigWritesTheDialingEndForABridge(t *testing.T) {
	got, err := RenderConfig(model.ServerTunnel{
		Name: "de1", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineBackpack,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", Transport: "tcp",
	})
	if err != nil {
		t.Fatalf("RenderConfig: %v", err)
	}
	want := `[client]
remote_addr = "203.0.113.10:8443"
transport = "tcp"
token = "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"
`
	if got != want {
		t.Fatalf("config =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderConfigBracketsIPv6AndRefusesWhatWouldBreakTheFile(t *testing.T) {
	base := model.ServerTunnel{
		Name: "v6", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineBackpack,
		Address: "2001:db8::1", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", Transport: "tcp",
	}
	got, err := RenderConfig(base)
	if err != nil || !strings.Contains(got, `remote_addr = "[2001:db8::1]:8443"`) {
		t.Fatalf("IPv6 config = %q, %v; want a bracketed remote_addr", got, err)
	}

	cases := []struct {
		name    string
		mutate  func(*model.ServerTunnel)
		wantErr string
	}{
		{"unknown transport", func(t *model.ServerTunnel) { t.Transport = "carrier-pigeon" }, "transport"},
		{"unknown fallback", func(t *model.ServerTunnel) { t.FallbackTransports = []string{"tcp", "nope"} }, "transport"},
		{"quote in a forwarded port", func(t *model.ServerTunnel) {
			t.Role = model.TunnelRolePortal
			t.ForwardPorts = []string{`443"]` + "\n" + `token = "x`}
		}, "forwarded port"},
		{"portal without forwarded ports", func(t *model.ServerTunnel) { t.Role = model.TunnelRolePortal }, "forwarded port"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tun := base
			c.mutate(&tun)
			if _, err := RenderConfig(tun); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("RenderConfig error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
}

func TestReadSnapshotReportsConnectionAndTraffic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/de1.metrics.json",
		`{"name":"de1","transport":"tcp","role":"server","taken":"2026-11-15T05:00:00Z","bytes_in":1500,"bytes_out":900,"peer":"198.51.100.7:51234","connected":true}`)
	snap, err := ReadSnapshot(dir, "de1")
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if !snap.Connected || snap.Peer != "198.51.100.7:51234" || snap.BytesIn != 1500 || snap.BytesOut != 900 {
		t.Fatalf("snapshot = %+v, want connected to 198.51.100.7:51234 with 1500 in / 900 out", snap)
	}

	// An older engine writes no "connected" key; a known peer is then the only evidence there is.
	writeFile(t, dir+"/old.metrics.json", `{"name":"old","peer":"198.51.100.7:1","bytes_in":1}`)
	if snap, _ := ReadSnapshot(dir, "old"); !snap.Connected {
		t.Fatalf("snapshot without a connected key but with a peer = %+v, want connected", snap)
	}
	writeFile(t, dir+"/down.metrics.json", `{"name":"down","connected":false,"peer":"198.51.100.7:1"}`)
	if snap, _ := ReadSnapshot(dir, "down"); snap.Connected {
		t.Fatalf("snapshot saying connected=false = %+v, want not connected", snap)
	}
	if _, err := ReadSnapshot(dir, "missing"); err == nil {
		t.Fatal("ReadSnapshot of a tunnel that never wrote one returned no error")
	}
}

func TestRenderConfigOpensTheRelayPortOnLoopbackOnlyAndNeedsNoOtherPort(t *testing.T) {
	portal := model.ServerTunnel{
		Name: "de1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10",
		Transport: "tcpmux", RelayPort: 63500,
	}
	got, err := RenderConfig(portal)
	if err != nil {
		t.Fatalf("RenderConfig with a relay and no forwarded port: %v", err)
	}
	if !strings.Contains(got, `ports = ["127.0.0.1:63500=127.0.0.1:63500"]`) {
		t.Fatalf("config =\n%s\nwant the relay forwarded on 127.0.0.1 only", got)
	}

	portal.ForwardPorts = []string{"443=127.0.0.1:2096"}
	got, _ = RenderConfig(portal)
	if !strings.Contains(got, `ports = ["443=127.0.0.1:2096", "127.0.0.1:63500=127.0.0.1:63500"]`) {
		t.Fatalf("config =\n%s\nwant the operator's ports first and the relay last", got)
	}

	portal.Role = model.TunnelRoleBridge
	got, _ = RenderConfig(portal)
	if strings.Contains(got, "63500") {
		t.Fatalf("bridge config mentions the relay port; only the entry end forwards it:\n%s", got)
	}
}

func TestRelayCredentialsFollowTheTunnelTokenAndAreNotTheToken(t *testing.T) {
	a := model.ServerTunnel{UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"}
	b := model.ServerTunnel{UUID: "11111111-5c1b-4f0e-9a51-2f6d1c7b9e10"}
	userA, passA := RelayCredentials(a)
	userA2, passA2 := RelayCredentials(a)
	_, passB := RelayCredentials(b)
	if userA == "" || passA == "" || userA != userA2 || passA != passA2 {
		t.Fatalf("credentials for one token differ between calls: %q/%q vs %q/%q", userA, passA, userA2, passA2)
	}
	if passA == passB || passA == a.UUID {
		t.Fatalf("password %q must differ per tunnel and from the tunnel token", passA)
	}
}
