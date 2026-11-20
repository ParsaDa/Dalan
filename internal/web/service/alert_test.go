package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func kindsOf(cs []AlertCondition) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Kind+"@"+c.Subject)
	}
	return out
}

func TestDesiredAlertsHysteresis(t *testing.T) {
	cpuKey := alertKey(model.AlertCPUHigh, alertLocal)
	tests := []struct {
		name string
		cpu  float64
		open map[string]bool
		want []string
	}{
		{"below the open line stays quiet", 82, nil, []string{}},
		{"crossing the open line opens", 86, nil, []string{"cpu.high@local"}},
		{"an open alert holds between the lines", 82, map[string]bool{cpuKey: true}, []string{"cpu.high@local"}},
		{"an open alert clears below the clear line", 79, map[string]bool{cpuKey: true}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := DesiredAlerts(AlertSnapshot{CPUPercent: tt.cpu, HaveCPU: true}, tt.open)
			if g := kindsOf(got); len(g) != len(tt.want) || (len(g) > 0 && g[0] != tt.want[0]) {
				t.Fatalf("got %v, want %v", g, tt.want)
			}
		})
	}
}

func TestDesiredAlertsNodes(t *testing.T) {
	nodes := []*model.Node{
		{Id: 1, Enable: true, Status: "offline"},
		{Id: 2, Enable: true, Status: "online", CpuPct: 91},
		{Id: 3, Enable: true, Status: "online", CpuPct: 20, MemPct: 30},
		{Id: 4, Enable: false, Status: "offline"},
	}
	got, kinds := DesiredAlerts(AlertSnapshot{Nodes: nodes}, nil)
	want := []string{"node.offline@node:1", "node.load@node:2"}
	g := kindsOf(got)
	if len(g) != 2 || g[0] != want[0] || g[1] != want[1] {
		t.Fatalf("got %v, want %v", g, want)
	}
	if got[1].NodeId != 2 || got[1].Value != 91 {
		t.Fatalf("load alert carries %+v, want node 2 at 91", got[1])
	}
	if len(kinds) != 2 {
		t.Fatalf("covered kinds %v, want node.offline and node.load", kinds)
	}
}

func TestReconcileOpensKeepsAndResolves(t *testing.T) {
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	s := &AlertService{}
	scope := []string{model.AlertNodeOffline}
	down1 := AlertCondition{Kind: model.AlertNodeOffline, Subject: "node:1", NodeId: 1}
	down2 := AlertCondition{Kind: model.AlertNodeOffline, Subject: "node:2", NodeId: 2}
	other := model.Alert{Kind: model.AlertCPUHigh, Subject: alertLocal, StartedAt: 50}
	if err := database.GetDB().Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	if err := s.Reconcile(scope, []AlertCondition{down1, down2}, 100); err != nil {
		t.Fatal(err)
	}
	if err := s.Reconcile(scope, []AlertCondition{down1}, 200); err != nil {
		t.Fatal(err)
	}

	var rows []model.Alert
	database.GetDB().Order("id").Find(&rows)
	if len(rows) != 3 {
		t.Fatalf("rows = %+v, want the cpu alert plus one per node", rows)
	}
	byKey := map[string]model.Alert{}
	for _, r := range rows {
		byKey[r.Subject+r.Kind] = r
	}
	n1, n2, cpu := byKey["node:1node.offline"], byKey["node:2node.offline"], byKey["localcpu.high"]
	if n1.StartedAt != 100 || n1.UpdatedAt != 200 || n1.ResolvedAt != 0 {
		t.Fatalf("held alert = %+v, want started 100, updated 200, open", n1)
	}
	if n2.ResolvedAt != 200 {
		t.Fatalf("cleared alert = %+v, want resolved at 200", n2)
	}
	if cpu.ResolvedAt != 0 {
		t.Fatalf("out-of-scope alert resolved: %+v", cpu)
	}
}

func TestCountExpiringClients(t *testing.T) {
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	now := time.UnixMilli(1_763_600_000_000)
	day := int64(24 * time.Hour / time.Millisecond)
	for _, c := range []model.ClientRecord{
		{Email: "soon", Enable: true, ExpiryTime: now.UnixMilli() + 2*day},
		{Email: "edge", Enable: true, ExpiryTime: now.UnixMilli() + 7*day},
		{Email: "later", Enable: true, ExpiryTime: now.UnixMilli() + 8*day},
		{Email: "gone", Enable: true, ExpiryTime: now.UnixMilli() - day},
		{Email: "off", Enable: false, ExpiryTime: now.UnixMilli() + day},
		{Email: "never", Enable: true, ExpiryTime: 0},
		{Email: "after-first-use", Enable: true, ExpiryTime: -3 * day},
	} {
		if err := database.GetDB().Create(&c).Error; err != nil {
			t.Fatalf("seed %s: %v", c.Email, err)
		}
	}
	// enable has a true default, so a false in Create is dropped; switch it off explicitly.
	database.GetDB().Model(&model.ClientRecord{}).Where("email = ?", "off").Update("enable", false)
	got, err := (&AlertService{}).CountExpiringClients(now)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("expiring = %d, want 2 (soon, edge)", got)
	}
}
