package events

import "testing"

// Subscribe, deliver, unsubscribe, and unregister a client subscribed to two
// channels (the old hub closed its send channel twice and panicked).
func TestHubRouting(t *testing.T) {
	h := &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan outbound),
		subscribe:  make(chan subRequest),
		unregister: make(chan *Client),
	}
	go h.run()
	c := &Client{send: make(chan []byte, 4)}

	h.subscribe <- subRequest{c, "sub_a", true}
	h.subscribe <- subRequest{c, "contest_x", true}
	h.broadcast <- outbound{channel: "sub_a", payload: []byte("1"), public: true}
	h.broadcast <- outbound{channel: "other", payload: []byte("2"), public: true}
	h.subscribe <- subRequest{c, "sub_a", false}
	h.broadcast <- outbound{channel: "sub_a", payload: []byte("3"), public: true}
	h.unregister <- c

	var got []string
	for m := range c.send {
		got = append(got, string(m))
	}
	if len(got) != 1 || got[0] != "1" {
		t.Fatalf("got %v, want [1]", got)
	}
}

// A private submission event reaches its submitter and view-all holders only.
func TestOutboundVisibility(t *testing.T) {
	private := outbound{owner: 7}
	cases := []struct {
		c    Client
		want bool
	}{
		{Client{}, false},             // anonymous
		{Client{profileID: 8}, false}, // another user
		{Client{profileID: 7}, true},  // the submitter
		{Client{seeAll: true}, true},  // view_all_submission
	}
	for i, tc := range cases {
		if got := private.visibleTo(&tc.c); got != tc.want {
			t.Errorf("case %d: got %v, want %v", i, got, tc.want)
		}
	}
	if !(outbound{public: true}).visibleTo(&Client{}) {
		t.Error("public event hidden from anonymous viewer")
	}
}
