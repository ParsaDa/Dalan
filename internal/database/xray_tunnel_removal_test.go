package database

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestMigratingRemovesXrayTunnelsRidersAndPoolMembersAndFlagsTheirNodes(t *testing.T) {
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "x-ui.db")
	t.Setenv("XUI_DB_FOLDER", dbDir)
	t.Setenv("XUI_DB_TYPE", "sqlite")
	if err := InitDB(dbPath); err != nil {
		t.Fatal(err)
	}

	seed := GetDB()
	for _, statement := range []string{
		"INSERT INTO server_tunnels (id, name, engine, parent_id) VALUES (1, 'old-portal', 'xray', 0)",
		"INSERT INTO server_tunnels (id, name, engine, parent_id) VALUES (2, 'node-old', 'xray', 1)",
		"INSERT INTO server_tunnels (id, name, engine, parent_id) VALUES (3, 'blank-engine', '', 0)",
		"INSERT INTO server_tunnels (id, name, engine, parent_id) VALUES (4, 'bp1', 'backpack', 0)",
		"INSERT INTO server_tunnels (id, name, engine, parent_id) VALUES (5, 'node-bp1', 'backpack', 4)",
		"INSERT INTO server_tunnels (id, name, engine, parent_id) VALUES (6, 'rk1', 'rakhsh', 0)",
		`INSERT INTO tunnel_pools (id, name, members, route_inbounds, active) VALUES (1, 'main', '["old-portal","bp1","rk1"]', '[]', 'old-portal')`,
		"INSERT INTO node_enrolments (id, name, expires_at, used_at, tunnel_id, node_id, data_tunnel_id) VALUES (1, 'old-node', 0, 5, 2, 7, 1)",
		"INSERT INTO node_enrolments (id, name, expires_at, used_at, tunnel_id, node_id, data_tunnel_id) VALUES (2, 'new-node', 0, 5, 5, 8, 4)",
		"INSERT INTO node_enrolments (id, name, expires_at, used_at, tunnel_id, node_id, data_tunnel_id) VALUES (3, 'unused-old', 0, 0, 2, 0, 0)",
	} {
		if err := seed.Exec(statement).Error; err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	if err := CloseDB(); err != nil {
		t.Fatal(err)
	}

	snapshot := func() (names []string, pool model.TunnelPool, flagged map[string]bool) {
		if err := InitDB(dbPath); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = CloseDB() })
		if err := GetDB().Model(&model.ServerTunnel{}).Order("id").Pluck("name", &names).Error; err != nil {
			t.Fatal(err)
		}
		if err := GetDB().First(&pool).Error; err != nil {
			t.Fatal(err)
		}
		var enrolments []model.NodeEnrolment
		if err := GetDB().Find(&enrolments).Error; err != nil {
			t.Fatal(err)
		}
		flagged = map[string]bool{}
		for _, e := range enrolments {
			flagged[e.Name] = e.NeedsReenrol
		}
		return names, pool, flagged
	}

	names, pool, flagged := snapshot()
	if want := []string{"bp1", "node-bp1", "rk1"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("surviving tunnels = %v, want %v", names, want)
	}
	if want := []string{"bp1", "rk1"}; !reflect.DeepEqual(pool.Members, want) || pool.Active != "" {
		t.Fatalf("pool = members %v active %q, want members %v and no active member", pool.Members, pool.Active, want)
	}
	if want := map[string]bool{"old-node": true, "new-node": false, "unused-old": false}; !reflect.DeepEqual(flagged, want) {
		t.Fatalf("needs re-enrol = %v, want %v", flagged, want)
	}

	if err := CloseDB(); err != nil {
		t.Fatal(err)
	}
	againNames, againPool, againFlagged := snapshot()
	if !reflect.DeepEqual(againNames, names) || !reflect.DeepEqual(againPool, pool) || !reflect.DeepEqual(againFlagged, flagged) {
		t.Fatalf("a second start changed things: %v %+v %v", againNames, againPool, againFlagged)
	}
}
