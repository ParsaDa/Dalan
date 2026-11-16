package job

import (
	"sync"

	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/pool"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// TunnelPoolJob measures every pool's members and moves the traffic to a spare when one stays better.
type TunnelPoolJob struct {
	pools    service.TunnelPoolService
	monitors map[int]*pool.Monitor
	running  sync.Mutex
}

func NewTunnelPoolJob() *TunnelPoolJob {
	return &TunnelPoolJob{monitors: map[int]*pool.Monitor{}}
}

// Run skips a round while the previous one is still measuring.
func (j *TunnelPoolJob) Run() {
	if !j.running.TryLock() {
		return
	}
	defer j.running.Unlock()
	if err := j.pools.Observe(j.monitors, j.pools.Probe); err != nil {
		logger.Warning("tunnel pools:", err)
	}
}
