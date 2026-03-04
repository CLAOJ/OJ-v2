package events

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/CLAOJ/claoj-go/cache"
	"github.com/redis/go-redis/v9"
)

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	// Registered clients map: map[channel]map[*Client]bool
	clients map[string]map[*Client]bool

	// Inbound messages from Redis
	broadcast chan *redis.Message

	// Register requests from the clients.
	// Structure: struct{ client *Client; channels []string }
	register chan SubscriptionRequest

	// Unregister requests from clients.
	unregister chan *Client

	mu sync.RWMutex
}

type SubscriptionRequest struct {
	client   *Client
	channels []string
}

// GlobalHub is the singleton hub instance
var GlobalHub *Hub

func InitHub() {
	GlobalHub = &Hub{
		broadcast:  make(chan *redis.Message, 256),
		register:   make(chan SubscriptionRequest),
		unregister: make(chan *Client),
		clients:    make(map[string]map[*Client]bool),
	}

	go GlobalHub.run()
	go GlobalHub.subscribeRedis()
}

func (h *Hub) run() {
	for {
		select {
		case req := <-h.register:
			h.mu.Lock()
			for _, ch := range req.channels {
				if h.clients[ch] == nil {
					h.clients[ch] = make(map[*Client]bool)
				}
				h.clients[ch][req.client] = true
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			for ch, clMap := range h.clients {
				if _, ok := clMap[client]; ok {
					delete(clMap, client)
					close(client.send)
					if len(clMap) == 0 {
						delete(h.clients, ch)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// message.Channel e.g., "submissions", "sub_<secret>"
			h.mu.RLock()
			clients := h.clients[message.Channel]

			// We must construct the payload format expected by the frontend.
			// The original node app wraps it like: [channel, payload_json]
			var rawPayload interface{}
			json.Unmarshal([]byte(message.Payload), &rawPayload)

			outboundMsg, err := json.Marshal([]interface{}{message.Channel, rawPayload})
			if err != nil {
				h.mu.RUnlock()
				continue
			}

			for client := range clients {
				select {
				case client.send <- outboundMsg:
				default:
					// Cannot send, client is stuck or dead
					close(client.send)
					delete(h.clients[message.Channel], client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) subscribeRedis() {
	log.Println("events: subscribing to Redis pub/sub channels")
	// The original Node daemon allows clients to subscribe to specific channels.
	// We'll use PSUBSCRIBE to listen to all channels that match our interested patterns.
	pubsub := cache.Client.PSubscribe(context.Background(), "submissions", "sub_*", "contest_*")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		msg, ok := <-ch
		if !ok {
			// Backoff and reconnect if Redis drops
			time.Sleep(5 * time.Second)
			pubsub = cache.Client.PSubscribe(context.Background(), "submissions", "sub_*", "contest_*")
			ch = pubsub.Channel()
			continue
		}

		h.broadcast <- msg
	}
}
