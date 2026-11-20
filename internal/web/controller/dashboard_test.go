package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func dashboardRouter(t *testing.T) *gin.Engine {
	t.Helper()
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewDashboardController(r.Group("/panel/api"))
	return r
}

func getJSON(t *testing.T, r *gin.Engine, path string, out any) {
	t.Helper()
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, path, nil))
	if err := json.Unmarshal(resp.Body.Bytes(), out); err != nil {
		t.Fatalf("%s: %v (%s)", path, err, resp.Body.String())
	}
}

func TestAlertsListReturnsOnlyOpenAlertsMostSeriousFirst(t *testing.T) {
	r := dashboardRouter(t)
	db := database.GetDB()
	db.Create(&model.Alert{Kind: model.AlertClientsExpiring, Subject: "local", Value: 14, StartedAt: 10})
	db.Create(&model.Alert{Kind: model.AlertNodeHighLoad, Subject: "node:3", NodeId: 3, Value: 91, StartedAt: 20})
	db.Create(&model.Alert{Kind: model.AlertNodeOffline, Subject: "node:4", NodeId: 4, StartedAt: 30})
	db.Create(&model.Alert{Kind: model.AlertNodeOffline, Subject: "node:5", NodeId: 5, StartedAt: 5, ResolvedAt: 9})

	var body struct {
		Success bool          `json:"success"`
		Obj     []model.Alert `json:"obj"`
	}
	getJSON(t, r, "/panel/api/alerts/list", &body)
	if !body.Success || len(body.Obj) != 3 {
		t.Fatalf("got %+v, want the three open alerts", body)
	}
	if body.Obj[0].Kind != model.AlertNodeOffline || body.Obj[1].Kind != model.AlertNodeHighLoad || body.Obj[2].Kind != model.AlertClientsExpiring {
		t.Fatalf("order = %s, %s, %s; want offline, load, expiring", body.Obj[0].Kind, body.Obj[1].Kind, body.Obj[2].Kind)
	}
}

func TestActivityListAppliesQueryFilters(t *testing.T) {
	r := dashboardRouter(t)
	db := database.GetDB()
	db.Create(&model.Activity{At: 100, Actor: "you", Action: "client.add", Target: "a"})
	db.Create(&model.Activity{At: 200, Actor: "you", Action: "tunnel.update", Target: "b", NodeId: 2})
	db.Create(&model.Activity{At: 300, Actor: "system", Action: "client.delete", Target: "c", NodeId: 2})

	var body struct {
		Obj struct {
			Items []model.Activity `json:"items"`
			Total int              `json:"total"`
		} `json:"obj"`
	}
	getJSON(t, r, "/panel/api/activity/list?category=client&nodeId=2", &body)
	if body.Obj.Total != 1 || body.Obj.Items[0].Target != "c" {
		t.Fatalf("got %+v, want only c", body.Obj)
	}
	getJSON(t, r, "/panel/api/activity/list?pageSize=2&page=1", &body)
	if body.Obj.Total != 3 || len(body.Obj.Items) != 2 || body.Obj.Items[0].Target != "c" {
		t.Fatalf("got %+v, want newest two of three", body.Obj)
	}
}
