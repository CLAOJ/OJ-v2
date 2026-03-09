package bridge

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

const MaxPacketSize = 64 * 1024 * 1024 // 64MB

// Packet represents a parsed JSON message from the judge
type Packet map[string]interface{}

// Name returns the "name" field of the packet (the event type)
func (p Packet) Name() string {
	if n, ok := p["name"].(string); ok {
		return n
	}
	return ""
}

// Connection represents a wrapped TCP connection that handles the
// Zlib compressed + length-prefixed protocol used by DMOJ/CLAOJ judges.
type Connection struct {
	conn net.Conn
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{conn: conn}
}

// ReadPacket blocks until a full packet is read and parsed.
// Protocol: 4 bytes (BigEndian uint32 length) followed by `length` bytes of Zlib-compressed JSON.
func (c *Connection) ReadPacket() (Packet, error) {
	// Read 4-byte length header
	var length uint32
	if err := binary.Read(c.conn, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	if length > MaxPacketSize {
		return nil, fmt.Errorf("packet too large: %d bytes", length)
	}

	// Read compressed payload
	compressed := make([]byte, length)
	if _, err := io.ReadFull(c.conn, compressed); err != nil {
		return nil, err
	}

	// Decompress zlib
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("zlib open error: %w", err)
	}
	defer r.Close()

	payload, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("zlib read error: %w", err)
	}

	// Parse JSON
	var pkt Packet
	if err := json.Unmarshal(payload, &pkt); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}

	return pkt, nil
}

// WritePacket JSON-encodes, zlib-compresses, and sends the packet.
func (c *Connection) WritePacket(pkt Packet) error {
	payload, err := json.Marshal(pkt)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(payload); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	compressed := buf.Bytes()
	length := uint32(len(compressed))

	// Write 4-byte header
	if err := binary.Write(c.conn, binary.BigEndian, length); err != nil {
		return err
	}

	// Write compressed payload
	if _, err := c.conn.Write(compressed); err != nil {
		return err
	}

	return nil
}

// SetDeadline sets the read/write deadline for the underlying connection
func (c *Connection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// Close closes the underlying connection
func (c *Connection) Close() error {
	return c.conn.Close()
}

// RemoteAddr returns the remote address
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
