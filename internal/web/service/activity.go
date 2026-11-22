package service

import (
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
)

const activityKeep = 90 * 24 * time.Hour

// ActivityFilter narrows the activity log. Category matches the action prefix ("client").
type ActivityFilter struct {
	Page     int
	PageSize int
	Category string
	Actor    string
	NodeId   *int
	Since    int64
}

type ActivityPage struct {
	Items []model.Activity `json:"items"`
	Total int64            `json:"total" example:"42"`
}

type ActivityService struct{}

var activityNow = time.Now

// Record logs one change; a failure is logged, never returned, so it cannot undo the change.
func (s *ActivityService) Record(actor, action, target string, nodeId int) {
	row := model.Activity{At: activityNow().Unix(), Actor: actor, Action: action, Target: target, NodeId: nodeId}
	if err := database.GetDB().Create(&row).Error; err != nil {
		logger.Warning("activity: record", action, "failed:", err)
	}
}

func (s *ActivityService) List(f ActivityFilter) (ActivityPage, error) {
	if f.PageSize <= 0 || f.PageSize > 200 {
		f.PageSize = 50
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	q := database.GetDB().Model(&model.Activity{})
	if f.Category != "" {
		q = q.Where("action LIKE ?", f.Category+".%")
	}
	if f.Actor != "" {
		q = q.Where("actor = ?", f.Actor)
	}
	if f.NodeId != nil {
		q = q.Where("node_id = ?", *f.NodeId)
	}
	if f.Since > 0 {
		q = q.Where("at >= ?", f.Since)
	}
	var page ActivityPage
	if err := q.Count(&page.Total).Error; err != nil {
		return page, err
	}
	page.Items = []model.Activity{}
	err := q.Order("at desc, id desc").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&page.Items).Error
	return page, err
}

// Prune drops entries older than the retention window.
func (s *ActivityService) Prune(now time.Time) error {
	return database.GetDB().Where("at < ?", now.Add(-activityKeep).Unix()).Delete(&model.Activity{}).Error
}
