package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func seedActivity(t *testing.T, s *ActivityService, at int64, action string, node int) {
	t.Helper()
	activityNow = func() time.Time { return time.Unix(at, 0) }
	t.Cleanup(func() { activityNow = time.Now })
	s.Record(model.ActorYou, action, "x", node)
}

func TestActivityListFiltersAndPages(t *testing.T) {
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	s := &ActivityService{}
	seedActivity(t, s, 100, "client.add", 0)
	seedActivity(t, s, 200, "tunnel.update", 2)
	seedActivity(t, s, 300, "client.delete", 2)
	seedActivity(t, s, 400, "clientgroup.add", 0)

	got, err := s.List(ActivityFilter{Category: "client"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 2 || got.Items[0].Action != "client.delete" || got.Items[1].Action != "client.add" {
		t.Fatalf("client filter = %+v, want delete then add, not clientgroup", got)
	}

	node := 2
	got, _ = s.List(ActivityFilter{NodeId: &node, Since: 250})
	if got.Total != 1 || got.Items[0].Action != "client.delete" {
		t.Fatalf("node+since = %+v, want only client.delete", got)
	}

	got, _ = s.List(ActivityFilter{Page: 2, PageSize: 3})
	if got.Total != 4 || len(got.Items) != 1 || got.Items[0].At != 100 {
		t.Fatalf("page 2 = %+v, want the oldest of four", got)
	}
}

func TestActivityPruneKeepsTheRetentionWindow(t *testing.T) {
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	s := &ActivityService{}
	now := time.Unix(100*24*3600, 0)
	seedActivity(t, s, now.Add(-91*24*time.Hour).Unix(), "client.add", 0)
	seedActivity(t, s, now.Add(-89*24*time.Hour).Unix(), "client.update", 0)
	if err := s.Prune(now); err != nil {
		t.Fatal(err)
	}
	got, _ := s.List(ActivityFilter{})
	if got.Total != 1 || got.Items[0].Action != "client.update" {
		t.Fatalf("after prune = %+v, want only the 89-day-old entry", got)
	}
}
