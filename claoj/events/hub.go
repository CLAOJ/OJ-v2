package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/CLAOJ/claoj/cache"
)

// Hub fans Redis pub/sub messages out to the websocket clients subscribed to
// each channel. Only run() touches clients, so no lock is needed.
type Hub struct {
	clients    map[string]map[*Client]bool
	broadcast  chan outbound
	subscribe  chan subRequest
	unregister chan *Client
}

type subRequest struct {
	client  *Client
	channel string
	on      bool // false = unsubscribe
}

type outbound struct {
	channel string
	payload []byte
	// Who may receive it: everyone when public, else the submitter
	// (owner, a judge_profile id) and view_all_submission holders.
	public bool
	owner  uint
}

func (m outbound) visibleTo(c *Client) bool {
	return m.public || c.seeAll || (m.owner != 0 && m.owner == c.profileID)
}

// GlobalHub is the singleton hub instance
var GlobalHub *Hub

func InitHub() {
	GlobalHub = &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan outbound, 256),
		subscribe:  make(chan subRequest),
		unregister: make(chan *Client),
	}
	go GlobalHub.run()
	go GlobalHub.subscribeRedis()
}

func (h *Hub) run() {
	for {
		select {
		case req := <-h.subscribe:
			if req.on {
				if h.clients[req.channel] == nil {
					h.clients[req.channel] = make(map[*Client]bool)
				}
				h.clients[req.channel][req.client] = true
			} else {
				h.remove(req.channel, req.client)
			}

		case client := <-h.unregister:
			for ch := range h.clients {
				h.remove(ch, client)
			}
			close(client.send)

		case msg := <-h.broadcast:
			for client := range h.clients[msg.channel] {
				if !msg.visibleTo(client) {
					continue
				}
				select {
				case client.send <- msg.payload:
				default: // client too slow: drop this message for it
				}
			}
		}
	}
}

func (h *Hub) remove(channel string, client *Client) {
	delete(h.clients[channel], client)
	if len(h.clients[channel]) == 0 {
		delete(h.clients, channel)
	}
}

func (h *Hub) subscribeRedis() {
	for {
		pubsub := cache.Client.PSubscribe(context.Background(), "submissions", "sub_*", "contest_*")
		log.Println("events: subscribed to Redis pub/sub")
		for msg := range pubsub.Channel() {
			// Same shape the frontend's useWebSocket expects.
			payload, err := json.Marshal(map[string]interface{}{
				"type":    "message",
				"channel": msg.Channel,
				"data":    json.RawMessage(msg.Payload),
			})
			if err != nil {
				continue
			}
			out := outbound{channel: msg.Channel, payload: payload, public: true}
			if msg.Channel == "submissions" {
				// The bridge marks each global event with its audience; an
				// unmarked or unreadable one reaches privileged viewers only.
				var meta struct {
					User   uint `json:"user"`
					Public bool `json:"public"`
				}
				_ = json.Unmarshal([]byte(msg.Payload), &meta)
				out.public, out.owner = meta.Public, meta.User
			}
			h.broadcast <- out
		}
		pubsub.Close()
		time.Sleep(5 * time.Second) // Redis dropped; reconnect
	}
}
