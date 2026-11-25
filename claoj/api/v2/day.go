package v2

import (
	"fmt"
	"time"
)

// day is a DATE() column value. MySQL with parseTime hands back a time.Time,
// sqlite a plain string; both scan down to the YYYY-MM-DD key the activity
// heatmap buckets on.
type day string

func (d *day) Scan(v any) error {
	*d = day(dayKey(v))
	return nil
}

// dayKey reduces a date value to its 10-char YYYY-MM-DD day. It accepts
// "2026-10-24", "2026-10-24T00:00:00Z" and time.Time.
func dayKey(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case time.Time:
		return t.Format("2006-01-02")
	case []byte:
		return dayKey(string(t))
	}
	s := fmt.Sprint(v)
	if len(s) > 10 {
		s = s[:10]
	}
	return s
}
