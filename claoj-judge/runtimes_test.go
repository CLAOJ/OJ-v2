package main

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJudgeExec dials the Child exactly as tiervnoj does, with an explicit
// executor map; executors == nil omits the field entirely.
func fakeJudgeExec(t *testing.T, addr, key string, executors interface{}) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	hello := Packet{
		"name": "handshake", "id": "Vịt Toàn Năng", "key": key,
		"problems": []interface{}{},
	}
	if executors != nil {
		hello["executors"] = executors
	}
	require.NoError(t, writePacket(conn, hello))
	resp, err := readPacket(conn)
	require.NoError(t, err)
	require.Equal(t, "handshake-success", resp.Name())
	return conn
}

// nextReport waits for one judge-runtimes report from the handshake hook.
func nextReport(t *testing.T, reports <-chan Packet) Packet {
	t.Helper()
	select {
	case p := <-reports:
		return p
	case <-time.After(2 * time.Second):
		t.Fatal("no judge-runtimes report")
		return nil
	}
}

func executorsFixture() map[string]interface{} {
	// float64 components: this is what the JSON round-trip through
	// writePacket/readPacket produces, and the report must pass the map
	// through unchanged.
	return map[string]interface{}{
		"C":   []interface{}{[]interface{}{"GCC 12", []interface{}{12.0, 2.0, 0.0}}},
		"PY3": []interface{}{[]interface{}{"Python 3", []interface{}{3.0, 10.0, 12.0}}},
	}
}

func TestChild_ReportsRuntimesOnHandshake(t *testing.T) {
	c, addr := startChild(t, "secret")
	reports := make(chan Packet, 4)
	started := make(chan struct{})
	gate := make(chan struct{})
	c.onRuntimes = func(p Packet) {
		reports <- p
		close(started)
		<-gate // hold the report open so the test can watch Ready()
	}

	executors := executorsFixture()
	_ = fakeJudgeExec(t, addr, "secret", executors) // handshake-success is already out

	<-started
	// while the report is still in flight the child must not look ready:
	// main() waits on Ready() before it pulls work
	ready := make(chan bool, 1)
	go func() { ready <- c.Ready() }()
	select {
	case <-ready:
		t.Fatal("child became ready before its runtime report completed")
	case <-time.After(100 * time.Millisecond):
	}
	close(gate)

	p := nextReport(t, reports)
	assert.Equal(t, "judge-runtimes", p.Name())
	assert.Equal(t, executors, p["executors"], "the handshake's map must pass through unchanged")
	select {
	case v := <-ready:
		assert.True(t, v, "the child becomes ready once its report is done")
	case <-time.After(2 * time.Second):
		t.Fatal("child never became ready")
	}
}

func TestChild_ReportsRuntimesAgainOnReconnect(t *testing.T) {
	c, addr := startChild(t, "secret")
	reports := make(chan Packet, 4)
	c.onRuntimes = func(p Packet) { reports <- p }

	executors := executorsFixture()
	_ = fakeJudgeExec(t, addr, "secret", executors)
	require.Equal(t, "judge-runtimes", nextReport(t, reports).Name())

	// tiervnoj reconnects (e.g. after its 300 s read timeout): the backend
	// must get the list again because it deletes rows on every report
	_ = fakeJudgeExec(t, addr, "secret", executors)
	require.Equal(t, "judge-runtimes", nextReport(t, reports).Name())
}

func TestChild_NoReportWithoutExecutors(t *testing.T) {
	c, addr := startChild(t, "secret")
	reports := make(chan Packet, 4)
	c.onRuntimes = func(p Packet) { reports <- p }

	// a handshake with no executor map must not wipe the backend's rows
	_ = fakeJudgeExec(t, addr, "secret", nil)
	select {
	case p := <-reports:
		t.Fatalf("unexpected report: %v", p)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestChild_NoReportOnRejectedHandshake(t *testing.T) {
	c, addr := startChild(t, "secret")
	reports := make(chan Packet, 4)
	c.onRuntimes = func(p Packet) { reports <- p }

	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, writePacket(conn, Packet{
		"name": "handshake", "id": "x", "key": "wrong",
		"executors": executorsFixture(),
	}))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = readPacket(conn)
	require.Error(t, err) // closed without handshake-success

	select {
	case p := <-reports:
		t.Fatalf("unauthenticated handshake reported: %v", p)
	case <-time.After(200 * time.Millisecond):
	}
}
