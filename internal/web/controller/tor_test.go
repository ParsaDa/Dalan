package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTorEngine(t *testing.T) *gin.Engine {
	t.Helper()
	newHostTestDB(t)
	engine := gin.New()
	NewTorController(engine.Group("/panel/api/tor"))
	return engine
}

type torRow struct {
	Id      int    `json:"id"`
	Country string `json:"country"`
	Port    int    `json:"port"`
	Enable  bool   `json:"enable"`
	Tag     string `json:"tag"`
	Status  struct {
		State string `json:"state"`
	} `json:"status"`
}

func listTor(t *testing.T, engine *gin.Engine) []torRow {
	t.Helper()
	env := doHostReq(t, engine, http.MethodGet, "/panel/api/tor/list", nil)
	if !env.Success {
		t.Fatalf("list failed: %s", env.Msg)
	}
	var rows []torRow
	if err := json.Unmarshal(env.Obj, &rows); err != nil {
		t.Fatalf("decode list: %v (%s)", err, env.Obj)
	}
	return rows
}

func postTorForm(t *testing.T, engine *gin.Engine, path string, form url.Values) hostEnvelope {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var env hostEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("POST %s: %v (%s)", path, err, w.Body.String())
	}
	return env
}

func TestTorRoutesAddListToggleAndDelete(t *testing.T) {
	engine := newTorEngine(t)
	if empty := doHostReq(t, engine, http.MethodGet, "/panel/api/tor/list", nil); string(empty.Obj) != "[]" {
		t.Fatalf("a fresh panel lists %s, want an empty array the page can map over", empty.Obj)
	}

	added := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/add", map[string]any{"country": "DE"})
	if !added.Success {
		t.Fatalf("add failed: %s", added.Msg)
	}
	if bad := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/add", map[string]any{"country": "xx1"}); bad.Success {
		t.Fatal("a country of three characters was accepted")
	}
	if dup := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/add", map[string]any{"country": "de"}); dup.Success {
		t.Fatal("the same country was added twice")
	}

	rows := listTor(t, engine)
	if len(rows) != 1 || rows[0].Country != "de" || rows[0].Tag != "tor-de" || !rows[0].Enable || rows[0].Port < 62700 || rows[0].Port > 62789 {
		t.Fatalf("list = %+v, want one enabled de exit tagged tor-de on a port in range", rows)
	}
	if rows[0].Status.State != "not_installed" {
		t.Fatalf("status = %q, want not_installed before the bundle is installed", rows[0].Status.State)
	}

	id := rows[0].Id
	off := postTorForm(t, engine, "/panel/api/tor/setEnable/"+strconv.Itoa(id), url.Values{"enable": {"false"}})
	if !off.Success {
		t.Fatalf("setEnable (form) failed: %s", off.Msg)
	}
	if rows := listTor(t, engine); rows[0].Enable {
		t.Fatal("the exit is still enabled after setEnable=false")
	}
	if on := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/setEnable/"+strconv.Itoa(id), map[string]any{"enable": true}); !on.Success {
		t.Fatalf("setEnable (json) failed: %s", on.Msg)
	}
	if rows := listTor(t, engine); !rows[0].Enable {
		t.Fatal("the exit is still disabled after setEnable=true")
	}

	if del := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/del/"+strconv.Itoa(id), nil); !del.Success {
		t.Fatalf("del failed: %s", del.Msg)
	}
	if rows := listTor(t, engine); len(rows) != 0 {
		t.Fatalf("the exit survived its deletion: %+v", rows)
	}
	if del := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/del/x", nil); del.Success {
		t.Fatal("a non-numeric id was accepted")
	}
}

func TestTorBridgesAreStoredAndRefusedWhenInvalid(t *testing.T) {
	engine := newTorEngine(t)
	const line = "obfs4 192.0.2.4:443 0123456789ABCDEF0123456789ABCDEF01234567 cert=abc iat-mode=0"

	if saved := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/bridges", map[string]any{"bridges": line}); !saved.Success {
		t.Fatalf("saving a valid bridge failed: %s", saved.Msg)
	}
	got := doHostReq(t, engine, http.MethodGet, "/panel/api/tor/bridges", nil)
	var out struct {
		Bridges string `json:"bridges"`
	}
	if err := json.Unmarshal(got.Obj, &out); err != nil || out.Bridges != line {
		t.Fatalf("bridges = %s (%v), want the stored line back", got.Obj, err)
	}

	bad := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/bridges", map[string]any{"bridges": line + "\nExitPolicy accept *:*"})
	if bad.Success || !strings.Contains(bad.Msg, "bridge line 2") {
		t.Fatalf("a line carrying a torrc directive: success=%v msg=%q, want a refusal naming bridge line 2", bad.Success, bad.Msg)
	}
	again := doHostReq(t, engine, http.MethodGet, "/panel/api/tor/bridges", nil)
	if err := json.Unmarshal(again.Obj, &out); err != nil || out.Bridges != line {
		t.Fatalf("a refused save changed the stored bridges to %s", again.Obj)
	}
}

func TestTorEngineAndTestAnswerBeforeTheBundleIsInstalled(t *testing.T) {
	engine := newTorEngine(t)
	info := doHostReq(t, engine, http.MethodGet, "/panel/api/tor/engine", nil)
	var engineInfo struct {
		Engine    string `json:"engine"`
		Installed bool   `json:"installed"`
		Pinned    string `json:"pinned"`
	}
	if err := json.Unmarshal(info.Obj, &engineInfo); err != nil || engineInfo.Engine != "tor" || engineInfo.Installed || engineInfo.Pinned != "16.0.1" {
		t.Fatalf("engine = %s (%v), want tor, not installed, pinned 16.0.1", info.Obj, err)
	}

	if test := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/test", map[string]any{"country": "de"}); test.Success || !strings.Contains(test.Msg, "not installed") {
		t.Fatalf("test before install: success=%v msg=%q, want a refusal that says Tor is not installed", test.Success, test.Msg)
	}
	if test := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/test", map[string]any{"country": "d1"}); test.Success {
		t.Fatal("a bad country was tested")
	}
	if rollback := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/engine/rollback", nil); rollback.Success {
		t.Fatal("a rollback with nothing to roll back to reported success")
	}
}

func TestXraySettingListsOnlyEnabledTorExitsForRouting(t *testing.T) {
	newHostTestDB(t)
	engine := gin.New()
	NewTorController(engine.Group("/panel/api/tor"))
	NewXraySettingController(engine.Group("/panel/api"))
	for _, country := range []string{"de", "nl"} {
		if added := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/add", map[string]any{"country": country}); !added.Success {
			t.Fatalf("add %s: %s", country, added.Msg)
		}
	}
	nl := listTor(t, engine)[1]
	if off := doHostReq(t, engine, http.MethodPost, "/panel/api/tor/setEnable/"+strconv.Itoa(nl.Id), map[string]any{"enable": false}); !off.Success {
		t.Fatalf("disable nl: %s", off.Msg)
	}

	env := doHostReq(t, engine, http.MethodPost, "/panel/api/xray/", nil)
	var payload string
	if err := json.Unmarshal(env.Obj, &payload); err != nil {
		t.Fatalf("the xray setting reply is not a string: %v (%s)", err, env.Obj)
	}
	var reply struct {
		TorOutbounds []struct {
			Tag     string `json:"tag"`
			Country string `json:"country"`
		} `json:"torOutbounds"`
	}
	if err := json.Unmarshal([]byte(payload), &reply); err != nil {
		t.Fatal(err)
	}
	if len(reply.TorOutbounds) != 1 || reply.TorOutbounds[0].Tag != "tor-de" || reply.TorOutbounds[0].Country != "de" {
		t.Fatalf("torOutbounds = %+v, want only the enabled de exit tagged tor-de", reply.TorOutbounds)
	}
}
