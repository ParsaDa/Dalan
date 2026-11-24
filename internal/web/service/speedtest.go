package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/speedtest"
)

const (
	speedtestHistoryKey   = "speedtestHistory"
	speedtestHistoryLimit = 20
	speedtestRunTimeout   = 2 * time.Minute
)

// ErrSpeedtestRunning is returned by Start while another run is in flight.
var ErrSpeedtestRunning = errors.New("a speed test is already running")

type speedtestRunner func(ctx context.Context, emit func(speedtest.Update)) (speedtest.Result, error)

// SpeedtestStatus is what the dashboard polls: live progress while Running, then the Result or Error.
type SpeedtestStatus struct {
	Running  bool              `json:"running"`
	Phase    speedtest.Phase   `json:"phase"`
	Progress float64           `json:"progress"`
	Mbps     float64           `json:"mbps"`
	PingMs   float64           `json:"pingMs"`
	Result   *speedtest.Result `json:"result"`
	Error    string            `json:"error"`
}

// SpeedtestService runs one speed test at a time against this server and keeps the recent results.
type SpeedtestService struct {
	run            speedtestRunner
	settingService SettingService

	mu     sync.Mutex
	status SpeedtestStatus
}

// NewSpeedtestService returns the shared service wired to the real speedtest.net runner.
func NewSpeedtestService() *SpeedtestService {
	return &SpeedtestService{run: func(ctx context.Context, emit func(speedtest.Update)) (speedtest.Result, error) {
		return speedtest.Run(ctx, speedtest.DefaultConfig(), emit)
	}}
}

// Start begins a run in the background; poll Status for progress.
func (s *SpeedtestService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Running {
		return ErrSpeedtestRunning
	}
	s.status = SpeedtestStatus{Running: true, Phase: speedtest.PhaseServers}
	go s.execute()
	return nil
}

func (s *SpeedtestService) execute() {
	ctx, cancel := context.WithTimeout(context.Background(), speedtestRunTimeout)
	defer cancel()
	res, err := s.run(ctx, func(u speedtest.Update) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.status.Phase = u.Phase
		if u.Progress > 0 || u.Mbps > 0 {
			s.status.Progress, s.status.Mbps = u.Progress, u.Mbps
		}
		if u.PingMs > 0 {
			s.status.PingMs = u.PingMs
		}
	})
	if err == nil {
		if herr := s.remember(res); herr != nil {
			logger.Warning("speedtest: could not save the result to history:", herr)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = SpeedtestStatus{}
	if err != nil {
		s.status.Error = err.Error()
		return
	}
	s.status.Result = &res
}

// Status returns a snapshot that is safe to hold while the run goes on.
func (s *SpeedtestService) Status() SpeedtestStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	if st.Result != nil {
		r := *st.Result
		st.Result = &r
	}
	return st
}

// History returns saved results, newest first.
func (s *SpeedtestService) History() ([]speedtest.Result, error) {
	raw, err := s.settingService.getString(speedtestHistoryKey)
	if err != nil {
		return nil, err
	}
	history := []speedtest.Result{}
	if raw == "" {
		return history, nil
	}
	if err := json.Unmarshal([]byte(raw), &history); err != nil {
		return nil, err
	}
	return history, nil
}

func (s *SpeedtestService) remember(res speedtest.Result) error {
	history, err := s.History()
	if err != nil {
		return err
	}
	history = append([]speedtest.Result{res}, history...)
	if len(history) > speedtestHistoryLimit {
		history = history[:speedtestHistoryLimit]
	}
	raw, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return s.settingService.setString(speedtestHistoryKey, string(raw))
}
