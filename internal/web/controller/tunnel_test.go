package controller

import (
	"encoding/json"
	"net/http"
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
