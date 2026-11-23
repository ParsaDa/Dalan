package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

func activityRouter(t *testing.T, ok bool, sawBody *string) *gin.Engine {
	t.Helper()
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/panel/api")
	api.Use(ActivityLog(&service.ActivityService{}))
	handler := func(c *gin.Context) {
		b, _ := io.ReadAll(c.Request.Body)
		*sawBody = string(b)
		jsonMsg(c, "", map[bool]error{true: nil, false: io.EOF}[ok])
	}
	api.POST("/clients/add", handler)
	api.POST("/clients/del/:email", handler)
	api.POST("/clients/bulkDel", handler)
	api.POST("/nodes/setEnable/:id", handler)
	api.POST("/clients/onlines", handler)
	return r
}

func post(r *gin.Engine, path, body string) {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)
}

func TestActivityLogRecordsSuccessfulChanges(t *testing.T) {
	var saw string
	r := activityRouter(t, true, &saw)
	post(r, "/panel/api/clients/add", `{"email":"sara.k","totalGB":0}`)
	if saw != `{"email":"sara.k","totalGB":0}` {
		t.Fatalf("handler saw body %q, want it untouched", saw)
	}
	post(r, "/panel/api/clients/del/amir", ``)
	post(r, "/panel/api/clients/bulkDel", `{"emails":["a","b","c"]}`)
	post(r, "/panel/api/nodes/setEnable/7", `{"enable":false}`)
	post(r, "/panel/api/clients/onlines", `{}`)

	var rows []model.Activity
	database.GetDB().Order("id").Find(&rows)
	got := make([]string, 0, len(rows))
	for _, a := range rows {
		got = append(got, a.Actor+" "+a.Action+" "+a.Target)
	}
	want := []string{"you client.add sara.k", "you client.delete amir", "you client.bulkDelete 3", "you node.disable #7"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("activity = %v, want %v", got, want)
	}
	if rows[3].NodeId != 7 {
		t.Fatalf("node toggle filed under node %d, want 7", rows[3].NodeId)
	}
}

func TestActivityLogSkipsFailedChanges(t *testing.T) {
	var saw string
	r := activityRouter(t, false, &saw)
	post(r, "/panel/api/clients/add", `{"email":"sara.k"}`)
	var n int64
	database.GetDB().Model(&model.Activity{}).Count(&n)
	if n != 0 {
		t.Fatalf("a failed add was logged (%d rows)", n)
	}
}
