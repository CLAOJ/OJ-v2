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
	h.broadcast <- outbound{"sub_a", []byte("1")}
	h.broadcast <- outbound{"other", []byte("2")}
	h.subscribe <- subRequest{c, "sub_a", false}
	h.broadcast <- outbound{"sub_a", []byte("3")}
	h.unregister <- c

	var got []string
	for m := range c.send {
		got = append(got, string(m))
	}
	if len(got) != 1 || got[0] != "1" {
		t.Fatalf("got %v, want [1]", got)
	}
}
