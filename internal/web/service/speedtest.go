package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/speedtest"
)

const (
	speedtestHistoryKey       = "speedtestHistory"
	tunnelSpeedtestHistoryKey = "tunnelSpeedtestHistory"
	speedtestHistoryLimit     = 20
	speedtestRunTimeout       = 2 * time.Minute
)

var (
	// ErrSpeedtestRunning is returned by Start while the same target is already being tested.
	ErrSpeedtestRunning = errors.New("a speed test is already running")
	// ErrSpeedtestBusy is returned by Start while a different target is being tested.
	ErrSpeedtestBusy = errors.New("another speed test is running; wait for it to finish")
)

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

// SpeedtestService runs one speed test at a time and keeps the recent results. Target 0 is this server's own
// connection to speedtest.net; any other target is the id of an entry tunnel, whose speed to its exit server is measured.
type SpeedtestService struct {
	run            func(ctx context.Context, tunnelId int, emit func(speedtest.Update)) (speedtest.Result, error)
	check          func(tunnelId int) error
	settingService SettingService

	mu     sync.Mutex
	active int
	busy   bool
	status map[int]SpeedtestStatus
}

var (
	sharedSpeedtest     *SpeedtestService
	sharedSpeedtestOnce sync.Once
)

// NewSpeedtestService returns the one service every controller shares, so the single run slot is global.
func NewSpeedtestService() *SpeedtestService {
	sharedSpeedtestOnce.Do(func() {
		var tunnels TunnelService
		sharedSpeedtest = &SpeedtestService{
			check: func(id int) error { _, _, err := tunnels.SpeedtestLink(id); return err },
			run: func(ctx context.Context, id int, emit func(speedtest.Update)) (speedtest.Result, error) {
				cfg := speedtest.DefaultConfig()
				if id != 0 {
					dial, addr, err := tunnels.SpeedtestLink(id)
					if err != nil {
						return speedtest.Result{}, err
					}
					cfg.Dial, cfg.LinkAddr = dial, addr
				}
				return speedtest.Run(ctx, cfg, emit)
			},
		}
	})
	return sharedSpeedtest
}

// Start begins a run in the background; poll Status for progress.
func (s *SpeedtestService) Start(tunnelId int) error {
	if s.check != nil && tunnelId != 0 {
		if err := s.check(tunnelId); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		if s.active == tunnelId {
			return ErrSpeedtestRunning
		}
		return ErrSpeedtestBusy
	}
	if s.status == nil {
		s.status = map[int]SpeedtestStatus{}
	}
	s.busy, s.active = true, tunnelId
	first := speedtest.PhaseServers
	if tunnelId != 0 {
		first = speedtest.PhasePing
	}
	s.status[tunnelId] = SpeedtestStatus{Running: true, Phase: first}
	go s.execute(tunnelId)
	return nil
}

func (s *SpeedtestService) execute(tunnelId int) {
	ctx, cancel := context.WithTimeout(context.Background(), speedtestRunTimeout)
	defer cancel()
	res, err := s.run(ctx, tunnelId, func(u speedtest.Update) {
		s.mu.Lock()
		defer s.mu.Unlock()
		st := s.status[tunnelId]
		st.Phase = u.Phase
		if u.Progress > 0 || u.Mbps > 0 {
			st.Progress, st.Mbps = u.Progress, u.Mbps
		}
		if u.PingMs > 0 {
			st.PingMs = u.PingMs
		}
		s.status[tunnelId] = st
	})
	if err == nil {
		if herr := s.remember(tunnelId, res); herr != nil {
			logger.Warning("speedtest: could not save the result to history:", herr)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	done := SpeedtestStatus{}
	if err != nil {
		done.Error = err.Error()
	} else {
		done.Result = &res
	}
	s.status[tunnelId] = done
	s.busy = false
}

// Measure runs one test of an entry tunnel for the caller and returns its result without keeping a history.
// It waits its turn when another test holds the single run slot.
func (s *SpeedtestService) Measure(ctx context.Context, tunnelId int) (speedtest.Result, error) {
	for {
		s.mu.Lock()
		if !s.busy {
			s.busy, s.active = true, tunnelId
			s.mu.Unlock()
			break
		}
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return speedtest.Result{}, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	defer func() {
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
	}()
	runCtx, cancel := context.WithTimeout(ctx, speedtestRunTimeout)
	defer cancel()
	return s.run(runCtx, tunnelId, func(speedtest.Update) {})
}

// Status returns a snapshot of one target that is safe to hold while a run goes on.
func (s *SpeedtestService) Status(tunnelId int) SpeedtestStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status[tunnelId]
	if st.Result != nil {
		r := *st.Result
		st.Result = &r
	}
	return st
}

// History returns the saved results of one target, newest first.
func (s *SpeedtestService) History(tunnelId int) ([]speedtest.Result, error) {
	all, err := s.loadHistory(tunnelId)
	if err != nil {
		return nil, err
	}
	return all[strconv.Itoa(tunnelId)], nil
}

// loadHistory reads the store a target lives in; this server keeps one list, tunnels keep a list per id.
func (s *SpeedtestService) loadHistory(tunnelId int) (map[string][]speedtest.Result, error) {
	key := tunnelSpeedtestHistoryKey
	if tunnelId == 0 {
		key = speedtestHistoryKey
	}
	raw, err := s.settingService.getString(key)
	if err != nil {
		return nil, err
	}
	all := map[string][]speedtest.Result{}
	if raw != "" {
		if tunnelId == 0 {
			var list []speedtest.Result
			if err := json.Unmarshal([]byte(raw), &list); err != nil {
				return nil, err
			}
			all["0"] = list
		} else if err := json.Unmarshal([]byte(raw), &all); err != nil {
			return nil, err
		}
	}
	if all[strconv.Itoa(tunnelId)] == nil {
		all[strconv.Itoa(tunnelId)] = []speedtest.Result{}
	}
	return all, nil
}

func (s *SpeedtestService) remember(tunnelId int, res speedtest.Result) error {
	all, err := s.loadHistory(tunnelId)
	if err != nil {
		return err
	}
	id := strconv.Itoa(tunnelId)
	list := append([]speedtest.Result{res}, all[id]...)
	if len(list) > speedtestHistoryLimit {
		list = list[:speedtestHistoryLimit]
	}
	all[id] = list
	var raw []byte
	key := tunnelSpeedtestHistoryKey
	if tunnelId == 0 {
		key = speedtestHistoryKey
		raw, err = json.Marshal(list)
	} else {
		raw, err = json.Marshal(all)
	}
	if err != nil {
		return err
	}
	return s.settingService.setString(key, string(raw))
}
