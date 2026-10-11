package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var errUnauthorized = errors.New("backend rejected judge credentials")

// Forwarder POSTs judge packets to the backend's /internal/judge/event.
// Callers send one packet at a time and wait, which keeps per-submission
// order even when the backend Service load-balances across replicas.
type Forwarder struct {
	URL     string        // e.g. http://backend:8081/internal/judge/event
	Name    string        // judge_judge.name
	Key     string        // judge_judge.auth_key
	Client  *http.Client  //
	MaxWait time.Duration // give up after this long; the asynq task is then retried
	Backoff time.Duration // first retry delay; doubles up to 10 s
}

// Post delivers p, retrying network errors and 5xx. 401/403 return
// errUnauthorized; 400 (malformed packet) is logged and dropped.
func (f *Forwarder) Post(p Packet) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(f.MaxWait)
	wait := f.Backoff
	for {
		status, err := f.postOnce(body)
		switch {
		case err == nil && status < 300:
			return nil
		case status == http.StatusUnauthorized || status == http.StatusForbidden:
			return errUnauthorized
		case status == http.StatusBadRequest:
			log.Printf("forward: backend rejected %q as malformed; dropping it", p.Name())
			return nil
		}
		if time.Now().Add(wait).After(deadline) {
			return fmt.Errorf("forward %q: giving up: status %d, err %v", p.Name(), status, err)
		}
		time.Sleep(wait)
		if wait *= 2; wait > 10*time.Second {
			wait = 10 * time.Second
		}
	}
}

func (f *Forwarder) postOnce(body []byte) (int, error) {
	req, err := http.NewRequest(http.MethodPost, f.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(f.Name, f.Key)
	resp, err := f.Client.Do(req)
	if err != nil {
		return 0, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode, nil
}
