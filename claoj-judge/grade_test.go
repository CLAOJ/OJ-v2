package main

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recorder collects posted packets.
type recorder struct {
	mu  sync.Mutex
	got []Packet
	err error
}

func (r *recorder) post(p Packet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, p)
	return r.err
}

func (r *recorder) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, p := range r.got {
		out = append(out, p.Name())
	}
	return out
}

func newAgent(t *testing.T) (*Agent, *recorder, net.Conn) {
	t.Helper()
	child, addr := startChild(t, "secret")
	judge := fakeJudge(t, addr, "secret")
	rec := &recorder{}
	return &Agent{child: child, post: rec.post, abortWait: 200 * time.Millisecond}, rec, judge
}

var request = Packet{"name": "submission-request", "submission-id": float64(7), "problem-id": "aplusb"}

// expectPacket reads the next packet the agent sent tiervnoj.
func expectPacket(t *testing.T, judge net.Conn, name string) Packet {
	t.Helper()
	judge.SetReadDeadline(time.Now().Add(2 * time.Second))
	p, err := readPacket(judge)
	require.NoError(t, err)
	require.Equal(t, name, p.Name())
	return p
}

func TestGrade_HappyPathForwardsInOrderAndSkipsStale(t *testing.T) {
	a, rec, judge := newAgent(t)
	go func() {
		expectPacket(t, judge, "submission-request")
		for _, p := range []Packet{
			{"name": "submission-acknowledged", "submission-id": 7},
			{"name": "grading-begin", "submission-id": 7},
			{"name": "test-case-status", "submission-id": 6}, // stale: another submission
			{"name": "test-case-status", "submission-id": 7, "cases": []interface{}{}},
			{"name": "grading-end", "submission-id": 7},
		} {
			writePacket(judge, p)
		}
	}()

	require.NoError(t, a.grade(context.Background(), request, false))
	assert.Equal(t, []string{"submission-acknowledged", "grading-begin", "test-case-status", "grading-end"}, rec.names())
}

func TestGrade_CompileErrorIsTerminal(t *testing.T) {
	a, rec, judge := newAgent(t)
	go func() {
		expectPacket(t, judge, "submission-request")
		writePacket(judge, Packet{"name": "compile-error", "submission-id": 7, "log": "boom"})
	}()

	require.NoError(t, a.grade(context.Background(), request, false))
	assert.Equal(t, []string{"compile-error"}, rec.names())
}

func TestGrade_AbortSendsTerminateAndForwardsTermination(t *testing.T) {
	a, rec, judge := newAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		expectPacket(t, judge, "submission-request")
		writePacket(judge, Packet{"name": "grading-begin", "submission-id": 7})
		time.Sleep(50 * time.Millisecond)
		cancel() // asynq CancelProcessing
		expectPacket(t, judge, "terminate-submission")
		writePacket(judge, Packet{"name": "submission-terminated", "submission-id": 7})
	}()

	require.NoError(t, a.grade(ctx, request, false))
	assert.Equal(t, []string{"grading-begin", "submission-terminated"}, rec.names())
}

func TestGrade_AbortWithoutAnswerRecordsTermination(t *testing.T) {
	a, rec, judge := newAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		expectPacket(t, judge, "submission-request")
		cancel()
		expectPacket(t, judge, "terminate-submission") // ...and never answer
	}()

	require.NoError(t, a.grade(ctx, request, false))
	assert.Equal(t, []string{"submission-terminated"}, rec.names())
}

func TestGrade_ShutdownLeavesTaskForRequeue(t *testing.T) {
	a, rec, judge := newAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		expectPacket(t, judge, "submission-request")
		a.stopping.Store(true)
		cancel()
	}()

	assert.ErrorIs(t, a.grade(ctx, request, false), context.Canceled)
	assert.Empty(t, rec.names())

	judge.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := readPacket(judge)
	assert.Error(t, err) // no terminate-submission was sent
}

func TestGrade_GiveUpReportsInternalErrorWithoutGrading(t *testing.T) {
	a, rec, judge := newAgent(t)

	require.NoError(t, a.grade(context.Background(), request, true))
	assert.Equal(t, []string{"internal-error"}, rec.names())
	assert.Equal(t, int64(7), rec.got[0].SubmissionID())

	judge.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := readPacket(judge)
	assert.Error(t, err) // tiervnoj never saw the submission
}

func TestGrade_PostErrorFailsTheTask(t *testing.T) {
	a, rec, judge := newAgent(t)
	rec.err = errors.New("backend down")
	go func() {
		expectPacket(t, judge, "submission-request")
		writePacket(judge, Packet{"name": "grading-begin", "submission-id": 7})
	}()

	assert.Error(t, a.grade(context.Background(), request, false))
}

func TestGrade_NoChildFailsTheTask(t *testing.T) {
	a := &Agent{child: NewChild("secret"), post: (&recorder{}).post}
	assert.ErrorIs(t, a.grade(context.Background(), request, false), errNoChild)
}
