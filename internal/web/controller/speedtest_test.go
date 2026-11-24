package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/speedtest"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type fakeSpeedtest struct {
	startErr error
	status   service.SpeedtestStatus
	history  []speedtest.Result
}

func (f *fakeSpeedtest) Start() error                         { return f.startErr }
func (f *fakeSpeedtest) Status() service.SpeedtestStatus      { return f.status }
func (f *fakeSpeedtest) History() ([]speedtest.Result, error) { return f.history, nil }

func speedtestRequest(t *testing.T, f *fakeSpeedtest, method, path string) map[string]any {
	t.Helper()
	a := &ServerController{speedtestService: f}
	engine := gin.New()
	engine.POST("/speedtest/start", a.speedtestStart)
	engine.GET("/speedtest/status", a.speedtestStatus)
	engine.GET("/speedtest/history", a.speedtestHistory)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	return env
}

func TestStartingWhileARunIsActiveStillSucceedsSoASecondTabCanAttach(t *testing.T) {
	env := speedtestRequest(t, &fakeSpeedtest{startErr: service.ErrSpeedtestRunning}, http.MethodPost, "/speedtest/start")
	if env["success"] != true {
		t.Fatalf("envelope = %v, want success", env)
	}
}

func TestStatusAndHistoryUseTheKeysTheDashboardReads(t *testing.T) {
	f := &fakeSpeedtest{
		status:  service.SpeedtestStatus{Running: true, Phase: speedtest.PhaseUpload, Progress: 0.25, Mbps: 41.5},
		history: []speedtest.Result{{DownMbps: 90, UpMbps: 30, PingMs: 8}},
	}

	obj, _ := speedtestRequest(t, f, http.MethodGet, "/speedtest/status")["obj"].(map[string]any)
	if obj["running"] != true || obj["phase"] != "upload" || obj["progress"] != 0.25 || obj["mbps"] != 41.5 {
		t.Fatalf("status obj = %v", obj)
	}
	list, _ := speedtestRequest(t, f, http.MethodGet, "/speedtest/history")["obj"].([]any)
	first, _ := list[0].(map[string]any)
	if len(list) != 1 || first["downMbps"] != 90.0 || first["upMbps"] != 30.0 || first["pingMs"] != 8.0 {
		t.Fatalf("history obj = %v", list)
	}
}
