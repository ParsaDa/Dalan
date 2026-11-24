package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/benchmark"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type fakeBenchmark struct {
	started   []service.BenchmarkRequest
	startErr  error
	cancelled int
	status    benchmark.RunStatus
}

func (f *fakeBenchmark) Start(req service.BenchmarkRequest) error {
	f.started = append(f.started, req)
	return f.startErr
}
func (f *fakeBenchmark) Cancel()                     { f.cancelled++ }
func (f *fakeBenchmark) Status() benchmark.RunStatus { return f.status }

func benchmarkRequest(t *testing.T, f *fakeBenchmark, method, path, body string) map[string]any {
	t.Helper()
	a := &TunnelController{benchmark: f}
	engine := gin.New()
	engine.POST("/benchmark/start", a.benchmarkStart)
	engine.POST("/benchmark/cancel", a.benchmarkCancel)
	engine.GET("/benchmark/status", a.benchmarkStatus)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	return env
}

func TestBenchmarkStartPassesTheRequestOnAndReportsARefusal(t *testing.T) {
	f := &fakeBenchmark{}
	body := `{"nodeId":3,"address":"203.0.113.10","objective":"balanced","candidates":[{"engine":"gost","transport":"tls","mode":"reverse"}]}`
	if env := benchmarkRequest(t, f, http.MethodPost, "/benchmark/start", body); env["success"] != true {
		t.Fatalf("envelope = %v, want success", env)
	}
	if len(f.started) != 1 || f.started[0].NodeId != 3 || f.started[0].Objective != benchmark.Balanced ||
		len(f.started[0].Candidates) != 1 || f.started[0].Candidates[0].Transport != "tls" {
		t.Fatalf("started = %+v", f.started)
	}

	f.startErr = benchmark.ErrRunning
	env := benchmarkRequest(t, f, http.MethodPost, "/benchmark/start", body)
	if msg, _ := env["msg"].(string); env["success"] != false || !strings.Contains(msg, "already running") {
		t.Fatalf("envelope = %v, want a failure that says a benchmark is already running", env)
	}
}

func TestBenchmarkStatusUsesTheKeysThePageReads(t *testing.T) {
	f := &fakeBenchmark{status: benchmark.RunStatus{
		Running: true, Objective: benchmark.Speed, Total: 4, Done: 1, Phase: benchmark.PhaseMeasure,
		Results: []benchmark.MethodResult{{
			Candidate:   benchmark.Candidate{Engine: "gost", Transport: "tls", Mode: "reverse"},
			Measurement: benchmark.Measurement{DownMbps: 90}, OK: true, Score: 63,
		}},
	}}
	obj, _ := benchmarkRequest(t, f, http.MethodGet, "/benchmark/status", "")["obj"].(map[string]any)
	results, _ := obj["results"].([]any)
	first, _ := results[0].(map[string]any)
	if obj["running"] != true || obj["total"] != 4.0 || obj["done"] != 1.0 || obj["phase"] != "measure" ||
		first["engine"] != "gost" || first["downMbps"] != 90.0 || first["ok"] != true || first["score"] != 63.0 {
		t.Fatalf("status obj = %v", obj)
	}
}

func TestBenchmarkCancelAsksTheRunToStop(t *testing.T) {
	f := &fakeBenchmark{}
	if env := benchmarkRequest(t, f, http.MethodPost, "/benchmark/cancel", ""); env["success"] != true || f.cancelled != 1 {
		t.Fatalf("envelope = %v, cancelled = %d", env, f.cancelled)
	}
}

func TestAddRefusesANameKeptForBenchmarkTunnels(t *testing.T) {
	a := &TunnelController{}
	engine := gin.New()
	engine.POST("/add", a.add)
	req := httptest.NewRequest(http.MethodPost, "/add", strings.NewReader(`{"name":"dalan-bench-4","address":"203.0.113.10","port":8443}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	if msg, _ := env["msg"].(string); env["success"] != false || !strings.Contains(msg, "kept for benchmark tunnels") {
		t.Fatalf("envelope = %v, want the reserved name refused before anything is created", env)
	}
}
