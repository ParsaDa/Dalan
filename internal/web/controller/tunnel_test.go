package controller

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func newTunnelEngine(t *testing.T) *gin.Engine {
	t.Helper()
	newHostTestDB(t)
	engine := gin.New()
	NewTunnelController(engine.Group("/panel/api/tunnels"))
	return engine
}

func listTunnels(t *testing.T, engine *gin.Engine) []model.ServerTunnel {
	t.Helper()
	env := doHostReq(t, engine, http.MethodGet, "/panel/api/tunnels/list", nil)
	var tunnels []model.ServerTunnel
	if err := json.Unmarshal(env.Obj, &tunnels); err != nil {
		t.Fatalf("decode list: %v (%s)", err, env.Obj)
	}
	return tunnels
}

// One panel plays both ends here only to drive every route; a real pair lives on two servers.
func TestTunnelRoutesCreateLinkImportToggleAndDelete(t *testing.T) {
	engine := newTunnelEngine(t)

	added := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/add", map[string]any{
		"name": "ir1", "address": "203.0.113.10", "port": 8443,
		"security": "reality", "serverName": "www.example.com", "routeInbounds": []string{"inbound-443"},
	})
	if !added.Success {
		t.Fatalf("add failed: %s", added.Msg)
	}
	var portal model.ServerTunnel
	if err := json.Unmarshal(added.Obj, &portal); err != nil || portal.Id == 0 {
		t.Fatalf("add returned %s (%v), want the stored portal", added.Obj, err)
	}
	if strings.Contains(string(added.Obj), "rivateKey") {
		t.Fatalf("the API exposed the REALITY private key: %s", added.Obj)
	}

	linkEnv := doHostReq(t, engine, http.MethodGet, "/panel/api/tunnels/link/1", nil)
	var link string
	if err := json.Unmarshal(linkEnv.Obj, &link); err != nil || !strings.HasPrefix(link, "dalan://tunnel/") {
		t.Fatalf("link = %s (%v), want a dalan://tunnel/ link", linkEnv.Obj, err)
	}

	clash := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/import", map[string]any{"link": link})
	if clash.Success || !strings.Contains(clash.Msg, "ir1") {
		t.Fatalf("importing a link whose name is taken: success=%v msg=%q, want a refusal naming ir1", clash.Success, clash.Msg)
	}
	bad := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/import", map[string]any{"link": "hello"})
	if bad.Success || !strings.Contains(bad.Msg, "not a Dalan tunnel link") {
		t.Fatalf("importing garbage: success=%v msg=%q", bad.Success, bad.Msg)
	}

	statusEnv := doHostReq(t, engine, http.MethodGet, "/panel/api/tunnels/status/1", nil)
	var status struct {
		Connected bool   `json:"connected"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(statusEnv.Obj, &status); err != nil || !statusEnv.Success || status.Connected || status.Error == "" {
		t.Fatalf("status with no Xray running = %s (success=%v, %v), want a not-connected status with its reason", statusEnv.Obj, statusEnv.Success, err)
	}

	off := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/setEnable/1", map[string]any{"enable": false})
	if !off.Success {
		t.Fatalf("setEnable failed: %s", off.Msg)
	}
	if got := listTunnels(t, engine); len(got) != 1 || got[0].Enable || got[0].RouteInbounds[0] != "inbound-443" {
		t.Fatalf("list = %+v, want one disabled tunnel routing inbound-443", got)
	}

	if del := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/del/1", nil); !del.Success {
		t.Fatalf("del failed: %s", del.Msg)
	}
	if got := listTunnels(t, engine); len(got) != 0 {
		t.Fatalf("list after delete = %+v, want none", got)
	}
	if gone := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/del/1", nil); gone.Success {
		t.Fatal("deleting a missing tunnel reported success")
	}
}

func TestTunnelRoutesCreateABackpackTunnelAndReportTheEngine(t *testing.T) {
	engine := newTunnelEngine(t)

	added := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/add", map[string]any{
		"name": "bp1", "address": "203.0.113.10", "port": 9443, "engine": "backpack",
		"transport": "wssmux", "fallbackTransports": []string{"quic"},
		"forwardPorts": []string{"443=127.0.0.1:2096"}, "acceptUdp": true,
	})
	var portal model.ServerTunnel
	if err := json.Unmarshal(added.Obj, &portal); err != nil || !added.Success {
		t.Fatalf("add failed: %s (%v)", added.Msg, err)
	}
	if portal.Engine != "backpack" || portal.Transport != "wssmux" || !portal.AcceptUDP ||
		len(portal.ForwardPorts) != 1 || len(portal.FallbackTransports) != 1 {
		t.Fatalf("stored tunnel = %+v, want the BackPack settings from the request", portal)
	}

	routed := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/add", map[string]any{
		"name": "bp2", "address": "203.0.113.10", "port": 9444, "engine": "backpack", "transport": "wssmux",
	})
	var routable model.ServerTunnel
	if err := json.Unmarshal(routed.Obj, &routable); err != nil || !routed.Success {
		t.Fatalf("a BackPack tunnel with no forwarded port: success=%v msg=%q (%v)", routed.Success, routed.Msg, err)
	}
	if routable.RelayPort < 63400 || routable.RelayPort > 64399 {
		t.Fatalf("relay port = %d, want one in 63400-64399 so the tunnel can be routed to", routable.RelayPort)
	}

	info := doHostReq(t, engine, http.MethodGet, "/panel/api/tunnels/engine", nil)
	var got struct {
		Installed bool   `json:"installed"`
		Version   string `json:"version"`
		Pinned    string `json:"pinned"`
	}
	if err := json.Unmarshal(info.Obj, &got); err != nil || got.Installed || got.Version != "" || got.Pinned != "v1.9.0" {
		t.Fatalf("engine info = %s (%v), want not installed, offering v1.9.0", info.Obj, err)
	}
}

func TestAddingATunnelForAnUnknownNodeCreatesNothing(t *testing.T) {
	engine := newTunnelEngine(t)

	refused := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/add", map[string]any{
		"name": "tr9", "address": "203.0.113.10", "port": 9555, "security": "none", "nodeId": 999,
	})
	if refused.Success || !strings.Contains(refused.Msg, "node 999") {
		t.Fatalf("add for an unknown node: success=%v msg=%q, want a refusal naming node 999", refused.Success, refused.Msg)
	}
	list := doHostReq(t, engine, http.MethodGet, "/panel/api/tunnels/list", nil)
	var tunnels []model.ServerTunnel
	if err := json.Unmarshal(list.Obj, &tunnels); err != nil || len(tunnels) != 0 {
		t.Fatalf("tunnels after the refused add = %s (%v), want none", list.Obj, err)
	}
}

func TestPoolRoutesCreateListAndDeleteAPool(t *testing.T) {
	engine := newTunnelEngine(t)
	for i, name := range []string{"a", "b"} {
		added := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/add", map[string]any{
			"name": name, "address": "203.0.113.10", "port": 8443 + i, "security": "none",
		})
		if !added.Success {
			t.Fatalf("add %s: %s", name, added.Msg)
		}
	}

	refused := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/pools/add", map[string]any{
		"name": "main", "members": []string{"a"},
	})
	if refused.Success || !strings.Contains(refused.Msg, "at least two") {
		t.Fatalf("a pool of one: success=%v msg=%q, want a refusal naming the minimum", refused.Success, refused.Msg)
	}

	created := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/pools/add", map[string]any{
		"name": "main", "members": []string{"b", "a"}, "routeInbounds": []string{"in-1"},
	})
	if !created.Success {
		t.Fatalf("pools/add: %s", created.Msg)
	}
	var made model.TunnelPool
	if err := json.Unmarshal(created.Obj, &made); err != nil || made.Id == 0 || made.Active != "b" {
		t.Fatalf("pools/add returned %s (%v), want the stored pool with b first", created.Obj, err)
	}

	listed := doHostReq(t, engine, http.MethodGet, "/panel/api/tunnels/pools/list", nil)
	var pools []model.TunnelPool
	if err := json.Unmarshal(listed.Obj, &pools); err != nil || len(pools) != 1 || pools[0].Name != "main" {
		t.Fatalf("pools/list = %s (%v), want the one pool", listed.Obj, err)
	}

	if deleted := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/pools/del/"+strconv.Itoa(made.Id), nil); !deleted.Success {
		t.Fatalf("pools/del: %s", deleted.Msg)
	}
	listed = doHostReq(t, engine, http.MethodGet, "/panel/api/tunnels/pools/list", nil)
	if strings.Contains(string(listed.Obj), "main") {
		t.Fatalf("the pool is still listed after deleting it: %s", listed.Obj)
	}
}

func TestDialRouteReportsWhichPortsThisServerCanReach(t *testing.T) {
	engine := newTunnelEngine(t)
	open, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer open.Close()
	port := open.Addr().(*net.TCPAddr).Port

	env := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/dial", map[string]any{
		"host": "127.0.0.1", "ports": []int{port},
	})
	var results []struct {
		Port      int  `json:"port"`
		Reachable bool `json:"reachable"`
	}
	if err := json.Unmarshal(env.Obj, &results); err != nil || len(results) != 1 || results[0].Port != port || !results[0].Reachable {
		t.Fatalf("dial = %s (%v), want the open port reachable", env.Obj, err)
	}

	refused := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/dial", map[string]any{"host": "127.0.0.1", "ports": []int{}})
	if refused.Success {
		t.Fatal("a dial request without ports was accepted")
	}
	unknown := doHostReq(t, engine, http.MethodPost, "/panel/api/tunnels/routetest", map[string]any{
		"nodeId": 99, "address": "203.0.113.10", "ports": []int{8443},
	})
	if unknown.Success {
		t.Fatal("a route test through a node that does not exist succeeded")
	}
}
