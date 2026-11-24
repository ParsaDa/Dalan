package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/speedtest"
)

func newSpeedtestService(t *testing.T, run func(context.Context, int, func(speedtest.Update)) (speedtest.Result, error)) *SpeedtestService {
	t.Helper()
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	return &SpeedtestService{run: run}
}

func waitIdle(t *testing.T, s *SpeedtestService, id ...int) SpeedtestStatus {
	target := 0
	if len(id) > 0 {
		target = id[0]
	}
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if st := s.Status(target); !st.Running {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the speed test never finished")
	return SpeedtestStatus{}
}

func resultWith(down float64) speedtest.Result {
	return speedtest.Result{Server: speedtest.ServerInfo{ID: "1", Sponsor: "ISP"}, PingMs: 12, DownMbps: down, UpMbps: 5, At: 1}
}

func TestStartRefusesASecondRunWhileOneIsRunning(t *testing.T) {
	release := make(chan struct{})
	s := newSpeedtestService(t, func(ctx context.Context, _ int, _ func(speedtest.Update)) (speedtest.Result, error) {
		<-release
		return resultWith(100), nil
	})

	if err := s.Start(0); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(0); !errors.Is(err, ErrSpeedtestRunning) {
		t.Fatalf("second Start err = %v, want ErrSpeedtestRunning", err)
	}
	close(release)
	waitIdle(t, s)
	if err := s.Start(0); err != nil {
		t.Fatalf("Start after the first run finished: %v", err)
	}
	waitIdle(t, s)
}

func TestStatusFollowsTheRunAndKeepsTheFinishedResult(t *testing.T) {
	step := make(chan struct{})
	s := newSpeedtestService(t, func(ctx context.Context, _ int, emit func(speedtest.Update)) (speedtest.Result, error) {
		emit(speedtest.Update{Phase: speedtest.PhaseDownload, Progress: 0.5, Mbps: 77})
		<-step
		return resultWith(90), nil
	})
	if err := s.Start(0); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var live SpeedtestStatus
	for time.Now().Before(deadline) {
		if live = s.Status(0); live.Mbps == 77 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !live.Running || live.Phase != speedtest.PhaseDownload || live.Progress != 0.5 || live.Mbps != 77 {
		t.Fatalf("live status = %+v", live)
	}
	close(step)

	done := waitIdle(t, s)
	if done.Result == nil || done.Result.DownMbps != 90 || done.Error != "" {
		t.Fatalf("finished status = %+v, want the result and no error", done)
	}
}

func TestAFailedRunReportsItsErrorAndLeavesTheHistoryAlone(t *testing.T) {
	s := newSpeedtestService(t, func(ctx context.Context, _ int, _ func(speedtest.Update)) (speedtest.Result, error) {
		return speedtest.Result{}, errors.New("could not get the server list from speedtest.net: status 503")
	})
	if err := s.Start(0); err != nil {
		t.Fatal(err)
	}
	st := waitIdle(t, s)
	if st.Error != "could not get the server list from speedtest.net: status 503" || st.Result != nil {
		t.Fatalf("status = %+v, want the error and no result", st)
	}
	if h, err := s.History(0); err != nil || len(h) != 0 {
		t.Fatalf("history = %v, %v; want empty", h, err)
	}
}

func TestHistoryKeepsTheNewestTwentyAndSurvivesANewService(t *testing.T) {
	n := 0
	s := newSpeedtestService(t, func(ctx context.Context, _ int, _ func(speedtest.Update)) (speedtest.Result, error) {
		n++
		return resultWith(float64(n)), nil
	})
	for range speedtestHistoryLimit + 3 {
		if err := s.Start(0); err != nil {
			t.Fatal(err)
		}
		waitIdle(t, s)
	}

	h, err := (&SpeedtestService{}).History(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != speedtestHistoryLimit {
		t.Fatalf("history length = %d, want %d", len(h), speedtestHistoryLimit)
	}
	if h[0].DownMbps != float64(speedtestHistoryLimit+3) || h[len(h)-1].DownMbps != 4 {
		t.Fatalf("history runs %v..%v, want newest first from %d down to 4", h[0].DownMbps, h[len(h)-1].DownMbps, speedtestHistoryLimit+3)
	}
}

func TestTunnelRunsKeepTheirOwnStatusAndHistoryAndShareOneSlot(t *testing.T) {
	release := make(chan struct{})
	s := newSpeedtestService(t, func(ctx context.Context, id int, _ func(speedtest.Update)) (speedtest.Result, error) {
		<-release
		return resultWith(float64(id)), nil
	})

	if err := s.Start(7); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(7); !errors.Is(err, ErrSpeedtestRunning) {
		t.Fatalf("same tunnel again: err = %v, want ErrSpeedtestRunning", err)
	}
	if err := s.Start(8); !errors.Is(err, ErrSpeedtestBusy) {
		t.Fatalf("another tunnel meanwhile: err = %v, want ErrSpeedtestBusy", err)
	}
	if s.Status(8).Running || s.Status(0).Running {
		t.Fatal("a run of tunnel 7 showed up as running for another target")
	}
	close(release)
	waitIdle(t, s, 7)

	if h, err := s.History(7); err != nil || len(h) != 1 || h[0].DownMbps != 7 {
		t.Fatalf("tunnel 7 history = %v, %v; want its one run", h, err)
	}
	if h, err := s.History(8); err != nil || len(h) != 0 {
		t.Fatalf("tunnel 8 history = %v, %v; want empty", h, err)
	}
	if h, err := s.History(0); err != nil || len(h) != 0 {
		t.Fatalf("this server's history = %v, %v; want untouched", h, err)
	}
}
