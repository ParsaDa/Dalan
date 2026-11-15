package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/enrol"
)

func newEnrolmentEngine(t *testing.T) *gin.Engine {
	t.Helper()
	newHostTestDB(t)
	engine := gin.New()
	base := engine.Group("/secret/")
	NewEnrolmentController(base.Group("/panel/api/enrolments"), base)
	NewNodeController(base.Group("/panel/api/nodes"))
	return engine
}

func postRaw(engine *gin.Engine, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestANodeEnrolsThroughThePublicRouteAndDeletingItClosesItsTunnel(t *testing.T) {
	engine := newEnrolmentEngine(t)

	added := doHostReq(t, engine, http.MethodPost, "/secret/panel/api/enrolments/add", map[string]any{
		"name": "de1", "panelUrl": "http://203.0.113.10:2053/secret/", "address": "203.0.113.10", "port": 8443,
		"security": "reality", "serverName": "www.example.com",
	})
	if !added.Success {
		t.Fatalf("add failed: %s", added.Msg)
	}
	var created struct {
		Id      int    `json:"id"`
		Link    string `json:"link"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(added.Obj, &created); err != nil || created.Id == 0 || !strings.HasPrefix(created.Link, "dalan://enrol/") {
		t.Fatalf("add returned %s (%v), want an enrolment with its link", added.Obj, err)
	}
	for _, secret := range []string{"tokenHash", "privateKey", "PrivateKey", "TokenHash"} {
		if strings.Contains(string(added.Obj), secret) {
			t.Fatalf("the API exposed %s: %s", secret, added.Obj)
		}
	}

	link, err := enrol.ParseLink(created.Link)
	if err != nil {
		t.Fatalf("ParseLink: %v", err)
	}
	if link.URL != "http://203.0.113.10:2053/secret/" {
		t.Fatalf("link points at %q, want the panel URL the operator gave", link.URL)
	}
	body, openReply, err := link.Seal(enrol.NodeInfo{Token: link.Token, Port: 2053, BasePath: "/", Scheme: "http", APIToken: "tok"})
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if w := postRaw(engine, "/secret/"+enrol.RedeemPath, []byte(`{"id":"nope","key":"x","box":"y"}`)); w.Code != http.StatusGone {
		t.Fatalf("garbage enrolment: status %d body %q, want 410", w.Code, w.Body.String())
	}
	if w := postRaw(engine, "/secret/"+enrol.RedeemPath, bytes.Repeat([]byte("a"), 1<<20)); w.Code == http.StatusOK {
		t.Fatalf("a 1 MiB enrolment body was accepted")
	}
	// No session and no API token: the sealed body is the only credential.
	w := postRaw(engine, "/secret/"+enrol.RedeemPath, body)
	if w.Code != http.StatusOK {
		t.Fatalf("enrolment: status %d body %q", w.Code, w.Body.String())
	}
	if grant, err := openReply(w.Body.Bytes()); err != nil || !strings.HasPrefix(grant.Tunnel, "dalan://tunnel/") {
		t.Fatalf("reply = %+v, %v; want a tunnel link", grant, err)
	}
	if w := postRaw(engine, "/secret/"+enrol.RedeemPath, body); w.Code != http.StatusGone {
		t.Fatalf("second use of the link: status %d, want 410", w.Code)
	}

	var node model.Node
	if err := database.GetDB().Where("name = ?", "de1").First(&node).Error; err != nil {
		t.Fatalf("node row: %v", err)
	}
	listed := doHostReq(t, engine, http.MethodGet, "/secret/panel/api/enrolments/list", nil)
	var rows []model.NodeEnrolment
	if err := json.Unmarshal(listed.Obj, &rows); err != nil || len(rows) != 1 || rows[0].NodeId != node.Id {
		t.Fatalf("list = %s (%v), want the one enrolment tied to node %d", listed.Obj, err, node.Id)
	}
	if refused := doHostReq(t, engine, http.MethodPost, "/secret/panel/api/enrolments/del/1", nil); refused.Success {
		t.Fatalf("an enrolment that belongs to a node was deleted on its own")
	}

	if del := doHostReq(t, engine, http.MethodPost, "/secret/panel/api/nodes/del/"+strconv.Itoa(node.Id), nil); !del.Success {
		t.Fatalf("delete node: %s", del.Msg)
	}
	var tunnels int64
	if err := database.GetDB().Model(&model.ServerTunnel{}).Count(&tunnels).Error; err != nil || tunnels != 0 {
		t.Fatalf("%d tunnel(s) left after the node was deleted (err %v)", tunnels, err)
	}
}

func TestAPendingEnrolmentCanBeCancelled(t *testing.T) {
	engine := newEnrolmentEngine(t)
	added := doHostReq(t, engine, http.MethodPost, "/secret/panel/api/enrolments/add", map[string]any{
		"name": "de1", "panelUrl": "http://203.0.113.10:2053/", "address": "203.0.113.10", "port": 8443, "security": "none",
	})
	if !added.Success {
		t.Fatalf("add failed: %s", added.Msg)
	}
	if del := doHostReq(t, engine, http.MethodPost, "/secret/panel/api/enrolments/del/1", nil); !del.Success {
		t.Fatalf("cancel: %s", del.Msg)
	}
	listed := doHostReq(t, engine, http.MethodGet, "/secret/panel/api/enrolments/list", nil)
	if string(listed.Obj) != "[]" {
		t.Fatalf("list after cancel = %s, want []", listed.Obj)
	}
}
