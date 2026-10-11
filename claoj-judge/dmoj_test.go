package main

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startChild runs a Child on a random loopback port.
func startChild(t *testing.T, key string) (*Child, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	c := NewChild(key)
	go c.Serve(ln)
	t.Cleanup(func() { ln.Close() })
	return c, ln.Addr().String()
}

// fakeJudge dials the Child the way tiervnoj does and completes the handshake.
func fakeJudge(t *testing.T, addr, key string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, writePacket(conn, Packet{
		"name": "handshake", "id": "Vịt Toàn Năng", "key": key,
		"problems": []interface{}{}, "executors": map[string]interface{}{},
	}))
	resp, err := readPacket(conn)
	require.NoError(t, err)
	require.Equal(t, "handshake-success", resp.Name())
	return conn
}

func nextEvent(t *testing.T, c *Child) Packet {
	t.Helper()
	select {
	case p := <-c.Events():
		return p
	case <-time.After(2 * time.Second):
		t.Fatal("no event from child")
		return nil
	}
}

func TestFraming_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	in := Packet{"name": "grading-end", "submission-id": float64(7), "note": "Vịt"}
	require.NoError(t, writePacket(&buf, in))
	out, err := readPacket(&buf)
	require.NoError(t, err)
	assert.Equal(t, in, out)
	assert.Equal(t, int64(7), out.SubmissionID())
}

func TestChild_RejectsBadKey(t *testing.T) {
	c, addr := startChild(t, "secret")
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, writePacket(conn, Packet{"name": "handshake", "id": "x", "key": "wrong"}))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = readPacket(conn)
	assert.Error(t, err) // closed without handshake-success
	assert.False(t, c.Ready())
}

func TestChild_ForwardsEventsAndFiltersNoise(t *testing.T) {
	c, addr := startChild(t, "secret")
	conn := fakeJudge(t, addr, "secret")
	assert.True(t, c.Ready())

	require.NoError(t, writePacket(conn, Packet{"name": "ping-response", "when": 1.0}))
	require.NoError(t, writePacket(conn, Packet{"name": "supported-problems", "problems": []interface{}{}}))
	require.NoError(t, writePacket(conn, Packet{"name": "grading-begin", "submission-id": 7}))

	assert.Equal(t, "grading-begin", nextEvent(t, c).Name())
}

func TestChild_SendReachesJudge(t *testing.T) {
	c, addr := startChild(t, "secret")
	conn := fakeJudge(t, addr, "secret")

	require.NoError(t, c.Send(Packet{"name": "terminate-submission"}))
	p, err := readPacket(conn)
	require.NoError(t, err)
	assert.Equal(t, "terminate-submission", p.Name())
}

func TestChild_NewestConnectionWins(t *testing.T) {
	c, addr := startChild(t, "secret")
	old := fakeJudge(t, addr, "secret")
	fresh := fakeJudge(t, addr, "secret") // tiervnoj reconnected after its 300 s timeout
	_ = old

	require.NoError(t, c.Send(Packet{"name": "ping", "when": 1.0}))
	fresh.SetReadDeadline(time.Now().Add(2 * time.Second))
	p, err := readPacket(fresh)
	require.NoError(t, err)
	assert.Equal(t, "ping", p.Name())
}

func TestChild_SendWithoutJudge(t *testing.T) {
	c := NewChild("secret")
	assert.ErrorIs(t, c.Send(Packet{"name": "ping"}), errNoChild)
}
