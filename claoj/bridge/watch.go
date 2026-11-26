package bridge

import (
	"fmt"
	"log"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
)

// WatchSubmissions publishes websocket events for submission changes this
// backend did not make itself: while v1 shares the database, its judges grade
// v1 submissions through v1's bridge, which never reaches /internal/judge/event.
// Every interval it reads new rows plus the rows it saw still queued/grading,
// so each poll is a primary-key lookup. A v2-graded submission also gets the
// bridge's own events; the duplicates are harmless (pages just refetch).
// ponytail: one poller per backend process; with several replicas, elect one.
func WatchSubmissions(interval time.Duration) {
	w := newWatcher()
	for range time.Tick(interval) {
		ids := make([]uint, 0, len(w.seen))
		for id := range w.seen {
			ids = append(ids, id)
		}
		var rows []watchRow
		q := db.DB.Model(&models.Submission{}).Select("id, status, result, current_testcase").Where("id > ?", w.last)
		if len(ids) > 0 {
			q = q.Or("id IN ?", ids)
		}
		if err := q.Order("id").Limit(500).Scan(&rows).Error; err != nil {
			log.Printf("bridge: watch submissions: %v", err)
			continue
		}
		w.step(rows, publishChange)
	}
}

type watchRow struct {
	ID              uint
	Status          string
	Result          *string
	CurrentTestcase int
}

type watcher struct {
	last uint            // highest id seen
	seen map[uint]string // in-flight id -> last published state
}

func newWatcher() *watcher {
	w := &watcher{seen: map[uint]string{}}
	db.DB.Model(&models.Submission{}).Select("COALESCE(MAX(id), 0)").Scan(&w.last)
	var inflight []watchRow
	db.DB.Model(&models.Submission{}).Select("id, status, result, current_testcase").
		Where("status IN ?", []string{"QU", "P", "G"}).Scan(&inflight)
	for _, r := range inflight {
		w.seen[r.ID] = r.state()
	}
	return w
}

func (r watchRow) state() string {
	res := ""
	if r.Result != nil {
		res = *r.Result
	}
	return fmt.Sprintf("%s/%s/%d", r.Status, res, r.CurrentTestcase)
}

func (r watchRow) done() bool { return r.Status != "QU" && r.Status != "P" && r.Status != "G" }

// step publishes every row whose state changed and tracks the in-flight ones.
func (w *watcher) step(rows []watchRow, publish func(watchRow)) {
	for _, r := range rows {
		if r.ID > w.last {
			w.last = r.ID
		}
		if prev, ok := w.seen[r.ID]; ok && prev == r.state() {
			continue
		}
		publish(r)
		if r.done() {
			delete(w.seen, r.ID)
		} else {
			w.seen[r.ID] = r.state()
		}
	}
}

func publishChange(r watchRow) {
	PostSubmissionState(r.ID, "update", map[string]interface{}{"status": r.Status})
	PostGlobalSubmissionUpdate(r.ID, r.Status, r.done())
}
