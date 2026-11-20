package service

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/pool"
)

func twoEntryTunnels(t *testing.T, svc *TunnelService) {
	t.Helper()
	for i, name := range []string{"a", "b"} {
		if _, err := svc.CreatePortal(tunnel.PortalSpec{
			Name: name, Address: "203.0.113.10", Port: 8443 + i, Transport: "tcpmux",
		}, nil); err != nil {
			t.Fatalf("CreatePortal(%s): %v", name, err)
		}
	}
}

func TestAPoolNeedsTwoOrdinaryEntryTunnelsAndAFreeName(t *testing.T) {
	svc := newTunnelDB(t)
	twoEntryTunnels(t, svc)
	pools := &TunnelPoolService{}

	cases := []struct {
		name    string
		pool    string
		members []string
		want    string
	}{
		{"one member", "main", []string{"a"}, "at least two"},
		{"a repeated member", "main", []string{"a", "a"}, "twice"},
		{"an unknown tunnel", "main", []string{"a", "zz"}, "does not exist"},
		{"a bad name", "Main Pool", []string{"a", "b"}, "lower-case"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pools.Create(tc.pool, tc.members, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Create = %v, want an error containing %q", err, tc.want)
			}
		})
	}

	if _, err := pools.Create("main", []string{"a", "b"}, []string{"in-1"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := pools.Create("main", []string{"a", "b"}, nil); err == nil {
		t.Fatal("a second pool with the same name was accepted")
	}
	var a model.ServerTunnel
	if err := database.GetDB().Where("name = ?", "a").First(&a).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateManagementRider(a.Id, "node-x"); err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Create("other", []string{"a", "node-x"}, nil); err == nil || !strings.Contains(err.Error(), "management") {
		t.Fatalf("Create with a management tunnel = %v, want a refusal naming management", err)
	}
}

func TestADeletedTunnelIsNotKeptByAPoolThatNeedsIt(t *testing.T) {
	svc := newTunnelDB(t)
	twoEntryTunnels(t, svc)
	pools := &TunnelPoolService{}
	if _, err := pools.Create("main", []string{"a", "b"}, nil); err != nil {
		t.Fatal(err)
	}
	var a model.ServerTunnel
	if err := database.GetDB().Where("name = ?", "a").First(&a).Error; err != nil {
		t.Fatal(err)
	}
	err := svc.Delete(a.Id)
	if err == nil || !strings.Contains(err.Error(), "pool main") {
		t.Fatalf("Delete of a pooled tunnel = %v, want a refusal naming pool main", err)
	}
}

func TestAPoolMovesToTheSpareOnlyAfterSeveralBadRoundsAndRemembersIt(t *testing.T) {
	svc := newTunnelDB(t)
	twoEntryTunnels(t, svc)
	pools := &TunnelPoolService{}
	if _, err := pools.Create("main", []string{"a", "b"}, nil); err != nil {
		t.Fatal(err)
	}
	monitors := map[int]*pool.Monitor{}
	aDown := func(t model.ServerTunnel) pool.Reading {
		return pool.Reading{Name: t.Name, Up: t.Name == "b", LatencyMs: 120}
	}
	active := func() string {
		list, err := pools.List()
		if err != nil || len(list) != 1 {
			t.Fatalf("List = %v, %v", list, err)
		}
		return list[0].Active
	}

	if active() != "a" {
		t.Fatalf("a new pool's active = %q, want its first member a", active())
	}
	for round := 1; round <= pool.SwitchAfter; round++ {
		if err := pools.Observe(monitors, aDown); err != nil {
			t.Fatalf("Observe round %d: %v", round, err)
		}
		want := "a"
		if round == pool.SwitchAfter {
			want = "b"
		}
		if got := active(); got != want {
			t.Fatalf("after round %d the active tunnel is %q, want %q", round, got, want)
		}
	}

	// A later round starts from the stored choice, as after a panel restart.
	fresh := map[int]*pool.Monitor{}
	aUp := func(t model.ServerTunnel) pool.Reading { return pool.Reading{Name: t.Name, Up: true, LatencyMs: 100} }
	for range pool.SwitchAfter + 1 {
		if err := pools.Observe(fresh, aUp); err != nil {
			t.Fatal(err)
		}
	}
	if got := active(); got != "b" {
		t.Fatalf("active after equal rounds = %q, want b to stay", got)
	}
}

func TestAPoolReachesTheGeneratedXrayConfig(t *testing.T) {
	svc := newTunnelDB(t)
	twoEntryTunnels(t, svc)
	if _, err := (&TunnelPoolService{}).Create("main", []string{"a", "b"}, []string{"in-1"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := (&XrayService{}).GetXrayConfig()
	if err != nil {
		t.Fatalf("GetXrayConfig: %v", err)
	}
	if !strings.Contains(string(cfg.RouterConfig), `"pool-main"`) || !strings.Contains(string(cfg.RouterConfig), `"tunnel-a"`) {
		t.Fatalf("routing = %s, want a pool-main balancer on tunnel-a", cfg.RouterConfig)
	}
}

func TestTheTunnelListCarriesWhatPassedThroughEachEntryTunnel(t *testing.T) {
	svc := newTunnelDB(t)
	twoEntryTunnels(t, svc)
	if err := database.GetDB().Create([]model.OutboundTraffics{
		{Tag: "tunnel-a", Up: 100, Down: 2000, Total: 2100},
		{Tag: "tunnel-b-dial", Up: 5, Down: 6, Total: 11},
	}).Error; err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListWithTraffic()
	if err != nil {
		t.Fatalf("ListWithTraffic: %v", err)
	}
	got := map[string][2]int64{}
	for _, row := range list {
		got[row.Name] = [2]int64{row.Up, row.Down}
	}
	if got["a"] != [2]int64{100, 2000} {
		t.Fatalf("a = %v, want its own outbound's counters", got["a"])
	}
	if got["b"] != [2]int64{0, 0} {
		t.Fatalf("b = %v, want zero: only the entry outbound counts, not the dialling one", got["b"])
	}
}
