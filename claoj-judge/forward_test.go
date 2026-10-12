package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newForwarder(url string) *Forwarder {
	return &Forwarder{URL: url, Name: "Vịt Toàn Năng", Key: "secret", Client: http.DefaultClient, MaxWait: time.Second, Backoff: time.Millisecond}
}

func TestForwarder_SendsBasicAuthAndBody(t *testing.T) {
	var got Packet
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "Vịt Toàn Năng", user)
		assert.Equal(t, "secret", pass)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	require.NoError(t, newForwarder(srv.URL).Post(Packet{"name": "grading-end", "submission-id": float64(7)}))
	assert.Equal(t, "grading-end", got.Name())
}

func TestForwarder_RetriesServerErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	require.NoError(t, newForwarder(srv.URL).Post(Packet{"name": "grading-begin"}))
	assert.Equal(t, int32(3), calls.Load())
}

func TestForwarder_UnauthorizedIsFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	assert.ErrorIs(t, newForwarder(srv.URL).Post(Packet{"name": "grading-begin"}), errUnauthorized)
}

func TestForwarder_BadRequestIsDroppedNotRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	assert.NoError(t, newForwarder(srv.URL).Post(Packet{"name": "grading-begin"}))
	assert.Equal(t, int32(1), calls.Load())
}

func TestForwarder_GivesUpAfterMaxWait(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	f := newForwarder(srv.URL)
	f.MaxWait = 30 * time.Millisecond
	start := time.Now()
	assert.Error(t, f.Post(Packet{"name": "grading-begin"}))
	assert.Less(t, time.Since(start), time.Second)
}

func TestForwarder_RecordsRTT(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	f := newForwarder(srv.URL)
	assert.Zero(t, f.RTT())
	require.NoError(t, f.Post(Packet{"name": "judge-status", "ping": float64(0)}))
	assert.Positive(t, f.RTT())
}
