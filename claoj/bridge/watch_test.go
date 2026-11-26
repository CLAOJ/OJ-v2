package bridge

import "testing"

func TestWatcherStep(t *testing.T) {
	ac := "AC"
	w := &watcher{last: 10, seen: map[uint]string{}}
	var got []uint
	pub := func(r watchRow) { got = append(got, r.ID) }

	w.step([]watchRow{{ID: 11, Status: "QU"}}, pub)                    // new: published, tracked
	w.step([]watchRow{{ID: 11, Status: "QU"}}, pub)                    // unchanged: skipped
	w.step([]watchRow{{ID: 11, Status: "G", CurrentTestcase: 2}}, pub) // progress: published
	w.step([]watchRow{{ID: 11, Status: "D", Result: &ac}}, pub)        // done: published, dropped
	w.step([]watchRow{{ID: 12, Status: "D", Result: &ac}}, pub)        // new and already done

	if len(got) != 4 || got[0] != 11 || got[3] != 12 {
		t.Fatalf("published %v, want [11 11 11 12]", got)
	}
	if len(w.seen) != 0 || w.last != 12 {
		t.Fatalf("seen=%v last=%d, want empty and 12", w.seen, w.last)
	}
}
