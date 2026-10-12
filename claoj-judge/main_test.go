package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLoadAvg(t *testing.T) {
	// /proc/loadavg: "1min 5min 15min running/total lastpid"
	load, ok := parseLoadAvg("0.24 0.19 0.17 1/234 5678\n", 8)
	assert.True(t, ok)
	assert.InDelta(t, 0.03, load, 1e-9)

	_, ok = parseLoadAvg("", 8)
	assert.False(t, ok)
	_, ok = parseLoadAvg("not-a-number", 8)
	assert.False(t, ok)
	_, ok = parseLoadAvg("-1 0 0", 8)
	assert.False(t, ok)

	// ncpu < 1 clamps to 1 instead of dividing by zero
	load, ok = parseLoadAvg("2.0 0 0", 0)
	assert.True(t, ok)
	assert.InDelta(t, 2.0, load, 1e-9)
}