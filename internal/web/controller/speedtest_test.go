package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/speedtest"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type fakeSpeedtest struct {
	startErr error
	started  []int
	status   service.SpeedtestStatus
	history  []speedtest.Result
}

func (f *fakeSpeedtest) Start(id int) error {
	f.started = append(f.started, id)
	return f.startErr
}
func (f *fakeSpeedtest) Status(int) service.SpeedtestStatus      { return f.status }
func (f *fakeSpeedtest) History(int) ([]speedtest.Result, error) { return f.history, nil }

func speedtestRequest(t *testing.T, f *fakeSpeedtest, method, path string) map[string]any {
	t.Helper()
	a := &ServerController{speedtestService: f}
	engine := gin.New()
	engine.POST("/speedtest/start", a.speedtestStart)
	engine.GET("/speedtest/status", a.speedtestStatus)
	engine.GET("/speedtest/history", a.speedtestHistory)
	tunnels := &TunnelController{speedtest: f}
	engine.POST("/tunnels/speedtest/start/:id", tunnels.speedtestStart)
	engine.GET("/tunnels/speedtest/status/:id", tunnels.speedtestStatus)
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

func TestTunnelSpeedtestStartsTheTunnelInThePathAndReportsABusySlot(t *testing.T) {
	f := &fakeSpeedtest{}
	if env := speedtestRequest(t, f, http.MethodPost, "/tunnels/speedtest/start/7"); env["success"] != true {
		t.Fatalf("envelope = %v, want success", env)
	}
	if len(f.started) != 1 || f.started[0] != 7 {
		t.Fatalf("started = %v, want [7]", f.started)
	}

	busy := &fakeSpeedtest{startErr: service.ErrSpeedtestBusy}
	env := speedtestRequest(t, busy, http.MethodPost, "/tunnels/speedtest/start/7")
	if env["success"] != false || !strings.Contains(env["msg"].(string), "another speed test is running") {
		t.Fatalf("envelope = %v, want a failure naming the running test", env)
	}
}

func TestTunnelSpeedtestRefusesAnIdThatIsNotATunnel(t *testing.T) {
	f := &fakeSpeedtest{}
	env := speedtestRequest(t, f, http.MethodGet, "/tunnels/speedtest/status/0")
	if env["success"] != false || len(f.started) != 0 {
		t.Fatalf("envelope = %v, want a failure for id 0 (that is this server's own test)", env)
	}
}
