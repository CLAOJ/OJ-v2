package main

import (
	"bytes"
	"compress/zlib"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const maxPacket = 64 << 20

// Packet is one DMOJ judge protocol message (JSON object with a "name").
type Packet map[string]interface{}

func (p Packet) Name() string { s, _ := p["name"].(string); return s }

// SubmissionID reads "submission-id" (a float64 after JSON decoding).
func (p Packet) SubmissionID() int64 {
	switch v := p["submission-id"].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

// readPacket reads one frame: 4-byte big-endian length + zlib(JSON).
func readPacket(r io.Reader) (Packet, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	if size > maxPacket {
		return nil, fmt.Errorf("packet too large: %d bytes", size)
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	zr, err := zlib.NewReader(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var p Packet
	if err := json.NewDecoder(zr).Decode(&p); err != nil {
		return nil, err
	}
	return p, nil
}

// writePacket writes one frame in a single Write so frames never interleave.
func writePacket(w io.Writer, p Packet) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write(raw); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	frame := make([]byte, 4+z.Len())
	binary.BigEndian.PutUint32(frame, uint32(z.Len()))
	copy(frame[4:], z.Bytes())
	_, err = w.Write(frame)
	return err
}

var errNoChild = errors.New("tiervnoj is not connected")

// Child is the agent's loopback DMOJ bridge: the local tiervnoj process
// dials it exactly as it would dial the site's bridged. tiervnoj reconnects
// on its own (e.g. after its 300 s read timeout), so the newest
// authenticated connection always wins.
type Child struct {
	key    string
	events chan Packet

	mu   sync.Mutex // guards conn and serialises writes
	conn net.Conn
}

func NewChild(key string) *Child {
	return &Child{key: key, events: make(chan Packet, 1024)}
}

// Serve accepts tiervnoj connections until ln is closed.
func (c *Child) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go c.handle(conn)
	}
}

func (c *Child) handle(conn net.Conn) {
	defer conn.Close()

	hello, err := readPacket(conn)
	if err != nil || hello.Name() != "handshake" {
		log.Printf("child: bad handshake: %v", err)
		return
	}
	key, _ := hello["key"].(string)
	if subtle.ConstantTimeCompare([]byte(key), []byte(c.key)) != 1 {
		log.Printf("child: handshake rejected: wrong key")
		return
	}

	c.mu.Lock()
	err = writePacket(conn, Packet{"name": "handshake-success"})
	if err == nil {
		c.conn = conn
	}
	c.mu.Unlock()
	if err != nil {
		return
	}
	log.Printf("child: tiervnoj connected")

	for {
		p, err := readPacket(conn)
		if err != nil {
			c.mu.Lock()
			if c.conn == conn {
				c.conn = nil
			}
			c.mu.Unlock()
			log.Printf("child: connection closed: %v", err)
			return
		}
		switch p.Name() {
		case "ping-response", "supported-problems", "current-submission-id", "handshake":
			// bookkeeping the backend never needed
		default:
			c.events <- p
		}
	}
}

// Send writes a packet to the connected tiervnoj.
func (c *Child) Send(p Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return errNoChild
	}
	return writePacket(c.conn, p)
}

func (c *Child) Events() <-chan Packet { return c.events }

func (c *Child) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// KeepAlive pings tiervnoj so its 300 s socket timeout never fires while
// the queue is idle.
func (c *Child) KeepAlive(every time.Duration) {
	for range time.Tick(every) {
		_ = c.Send(Packet{"name": "ping", "when": float64(time.Now().UnixNano()) / 1e9})
	}
}
