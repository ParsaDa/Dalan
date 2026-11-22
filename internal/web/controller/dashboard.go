package controller

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// DashboardController serves what the dashboard shows beyond live status:
// open alerts, the activity log and this month's traffic against the plan.
type DashboardController struct {
	alertService    service.AlertService
	activityService service.ActivityService
	trafficService  service.TrafficPeriodService
}

func NewDashboardController(api *gin.RouterGroup) *DashboardController {
	a := &DashboardController{}
	api.GET("/alerts/list", a.alerts)
	api.GET("/activity/list", a.activity)
	api.GET("/server/monthlyTraffic", a.monthlyTraffic)
	return a
}

func (a *DashboardController) alerts(c *gin.Context) {
	rows, err := a.alertService.ListOpen()
	jsonObj(c, rows, err)
}

func (a *DashboardController) activity(c *gin.Context) {
	f := service.ActivityFilter{Category: c.Query("category"), Actor: c.Query("actor")}
	f.Page, _ = strconv.Atoi(c.Query("page"))
	f.PageSize, _ = strconv.Atoi(c.Query("pageSize"))
	if raw := c.Query("nodeId"); raw != "" {
		if id, err := strconv.Atoi(raw); err == nil {
			f.NodeId = &id
		}
	}
	if days, err := strconv.Atoi(c.Query("days")); err == nil && days > 0 {
		f.Since = time.Now().Add(-time.Duration(days) * 24 * time.Hour).Unix()
	}
	page, err := a.activityService.List(f)
	jsonObj(c, page, err)
}

func (a *DashboardController) monthlyTraffic(c *gin.Context) {
	usage, err := a.trafficService.Usage(time.Now())
	jsonObj(c, usage, err)
}
