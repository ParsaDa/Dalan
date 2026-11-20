package job

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/eventbus"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/util/sys"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// AlertsJob keeps the dashboard's alerts in step with this server, its nodes and its
// clients; it samples CPU from the previous run so it never blocks a cron stop.
type AlertsJob struct {
	mu           sync.Mutex
	alerts       service.AlertService
	activity     service.ActivityService
	nodes        service.NodeService
	xray         service.XrayService
	traffic      service.TrafficPeriodService
	readTimes    func(percpu bool) ([]cpu.TimesStat, error)
	last         cpu.TimesStat
	hasLast      bool
	lastPruneDay int
}

func NewAlertsJob() *AlertsJob {
	return &AlertsJob{readTimes: cpu.Times}
}

func (j *AlertsJob) Run() {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	snap := service.AlertSnapshot{HaveXray: true, XrayRunning: j.xray.IsXrayRunning()}
	if times, err := j.readTimes(false); err == nil && len(times) > 0 {
		if j.hasLast {
			snap.CPUPercent, snap.HaveCPU = sys.CPUWorkPercent(j.last, times[0])
		}
		j.last, j.hasLast = times[0], true
	}
	if vm, err := mem.VirtualMemory(); err == nil && vm != nil {
		snap.MemPercent, snap.HaveMem = vm.UsedPercent, true
	}
	if nodes, err := j.nodes.GetAll(); err == nil {
		snap.Nodes = append([]*model.Node{}, nodes...)
	}
	if n, err := j.alerts.CountExpiringClients(now); err == nil {
		snap.ExpiringClients, snap.HaveClients = n, true
	}
	if err := j.alerts.Apply(snap, now.Unix()); err != nil {
		logger.Warning("alerts: reconcile failed:", err)
	}
	if bytes, err := service.PhysicalNetBytes(); err == nil {
		if _, err := j.traffic.Sample(bytes, now); err != nil {
			logger.Warning("alerts: traffic sample failed:", err)
		}
	}
	if day := now.YearDay(); day != j.lastPruneDay {
		j.lastPruneDay = day
		if err := j.activity.Prune(now); err != nil {
			logger.Warning("activity: prune failed:", err)
		}
	}
}

// HandleEvent re-evaluates at once on node and Xray transitions, and logs them as
// system activity, so the dashboard does not wait for the next tick.
func (j *AlertsJob) HandleEvent(e eventbus.Event) {
	action := map[eventbus.EventType]string{
		eventbus.EventNodeDown:  "node.offline",
		eventbus.EventNodeUp:    "node.online",
		eventbus.EventXrayCrash: "xray.crash",
	}[e.Type]
	if action == "" {
		return
	}
	nodeId := 0
	if d, ok := e.Data.(*eventbus.NodeHealthData); ok {
		nodeId = d.NodeId
	}
	j.activity.Record(model.ActorSystem, action, e.Source, nodeId)
	j.Run()
}
