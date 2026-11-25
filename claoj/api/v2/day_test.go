package v2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The heatmap keys on YYYY-MM-DD, but the driver may hand back a timestamp
// (MySQL with parseTime) or already-text dates (sqlite).
func TestDayKey_NormalizesDriverFormats(t *testing.T) {
	at := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"date text", "2026-10-24", "2026-10-24"},
		{"rfc3339 text", "2026-10-24T00:00:00Z", "2026-10-24"},
		{"text bytes", []byte("2026-10-24"), "2026-10-24"},
		{"timestamp", at, "2026-10-24"},
		{"nil", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, dayKey(tc.in))
		})
	}
}

func TestDayScan_AcceptsTimestamp(t *testing.T) {
	var d day
	require.NoError(t, d.Scan(time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)))
	require.Equal(t, "2026-10-24", string(d))
}
