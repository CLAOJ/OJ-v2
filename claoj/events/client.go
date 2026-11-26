package events

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// In production, tighten this to allow only the frontend origin
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	// Who is connected, for per-message filtering in the hub.
	profileID uint // 0 = anonymous
	seeAll    bool // judge.view_all_submission
	// allowed reports whether this viewer may subscribe to a channel.
	allowed func(channel string) bool
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("events: read_error: %v", err)
			}
			break
		}

		// {"type":"subscribe"|"unsubscribe","channel":"sub_<secret>"}
		var req struct{ Type, Channel string }
		if json.Unmarshal(message, &req) != nil || req.Channel == "" {
			continue
		}
		switch {
		case req.Type == "unsubscribe":
			c.hub.subscribe <- subRequest{client: c, channel: req.Channel, on: false}
		case req.Type == "subscribe" && c.allowed(req.Channel):
			c.hub.subscribe <- subRequest{client: c, channel: req.Channel, on: true}
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWS handles websocket requests from the peer.
func ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("events: Failed to upgrade WS: %v", err)
		return
	}

	profileID, _ := auth.CurrentProfileID(c)
	client := &Client{
		hub: GlobalHub, conn: conn, send: make(chan []byte, 256),
		profileID: profileID,
		seeAll:    auth.HasPerm(c, "judge.view_all_submission"),
		allowed:   func(channel string) bool { return canSubscribe(c, channel) },
	}

	go client.writePump()
	// readPump stays on the handler goroutine: allowed() reads this request's
	// gin context, which gin recycles as soon as ServeWS returns.
	client.readPump()
}

// canSubscribe is the per-account gate for a channel.
//   - submissions: anyone; the hub filters each message per viewer
//   - sub_<secret>: anyone holding the secret, which only the submission page
//     (itself permission-checked) hands out
//   - contest_<key>: viewers who may see that contest and its scoreboard
func canSubscribe(c *gin.Context, channel string) bool {
	switch {
	case channel == "submissions", strings.HasPrefix(channel, "sub_"):
		return true
	case strings.HasPrefix(channel, "contest_"):
		var ct models.Contest
		if db.DB.Where("`key` = ?", strings.TrimPrefix(channel, "contest_")).First(&ct).Error != nil {
			return false
		}
		return auth.CanViewContest(c, &ct) && auth.CanSeeContestScoreboard(c, &ct)
	}
	return false
}
