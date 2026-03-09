// Package protocol implements the judge communication protocol.
package protocol

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

// Packet represents a protocol packet.
type Packet map[string]interface{}

// PacketManager handles network communication.
type PacketManager struct {
	conn       net.Conn
	host       string
	port       int
	judgeName  string
	judgeKey   string
	secure     bool
	sizePack   binary.ByteOrder
	closed     bool
}

// NewPacketManager creates a new packet manager.
func NewPacketManager(cfg *Config) (*PacketManager, error) {
	pm := &PacketManager{
		host:      cfg.ServerHost,
		port:      cfg.ServerPort,
		judgeName: cfg.JudgeName,
		judgeKey:  cfg.JudgeKey,
		sizePack:  binary.BigEndian,
	}

	if err := pm.connect(); err != nil {
		return nil, err
	}

	return pm, nil
}

// connect establishes connection to the server.
func (pm *PacketManager) connect() error {
	addr := fmt.Sprintf("%s:%d", pm.host, pm.port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}

	pm.conn = conn
	pm.conn.SetDeadline(time.Now().Add(300 * time.Second))
	pm.conn.(*net.TCPConn).SetKeepAlive(true)

	return nil
}

// Run starts the packet reading loop.
func (pm *PacketManager) Run(judge JudgeHandler) error {
	for {
		packet, err := pm.ReadPacket()
		if err != nil {
			return err
		}

		if err := judge.HandlePacket(packet); err != nil {
			return err
		}
	}
}

// ReadPacket reads a single packet from the connection.
func (pm *PacketManager) ReadPacket() (Packet, error) {
	// Read size prefix
	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(pm.conn, sizeBuf); err != nil {
		return nil, err
	}

	size := pm.sizePack.Uint32(sizeBuf)

	// Read compressed data
	data := make([]byte, size)
	if _, err := io.ReadFull(pm.conn, data); err != nil {
		return nil, err
	}

	// Decompress
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}

	// Parse JSON
	var packet Packet
	if err := json.Unmarshal(buf.Bytes(), &packet); err != nil {
		return nil, err
	}

	return packet, nil
}

// WritePacket writes a packet to the connection.
func (pm *PacketManager) WritePacket(packet Packet) error {
	// Marshal to JSON
	data, err := json.Marshal(packet)
	if err != nil {
		return err
	}

	// Compress
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	// Write size prefix + data
	size := uint32(buf.Len())
	sizeBuf := make([]byte, 4)
	pm.sizePack.PutUint32(sizeBuf, size)

	if _, err := pm.conn.Write(sizeBuf); err != nil {
		return err
	}

	if _, err := pm.conn.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

// Handshake performs the initial handshake.
func (pm *PacketManager) Handshake(problems map[string]float64, executors map[string][]string) error {
	packet := Packet{
		"name":     "handshake",
		"id":       pm.judgeName,
		"key":      pm.judgeKey,
		"problems": problems,
		"executors": executors,
	}

	if err := pm.WritePacket(packet); err != nil {
		return err
	}

	// Read response
	response, err := pm.ReadPacket()
	if err != nil {
		return err
	}

	if response["name"] != "handshake-success" {
		return fmt.Errorf("handshake failed: %v", response)
	}

	return nil
}

// SendSupportedProblems sends the list of supported problems.
func (pm *PacketManager) SendSupportedProblems(problems map[string]float64) {
	// Convert to expected format
	problemList := make([][]interface{}, 0, len(problems))
	for id, mtime := range problems {
		problemList = append(problemList, []interface{}{id, mtime})
	}

	pm.WritePacket(Packet{
		"name":     "supported-problems",
		"problems": problemList,
	})
}

// SendGradingBegin sends grading begin notification.
func (pm *PacketManager) SendGradingBegin(submissionID uint) {
	pm.WritePacket(Packet{
		"name":          "grading-begin",
		"submission-id": submissionID,
		"pretested":     false,
	})
}

// SendGradingEnd sends grading end notification.
func (pm *PacketManager) SendGradingEnd(submissionID uint, result *GratingResult) {
	pm.WritePacket(Packet{
		"name":          "grading-end",
		"submission-id": submissionID,
	})
}

// SendTestCaseStatus sends a test case result.
func (pm *PacketManager) SendTestCaseStatus(submissionID uint, position int, result TestCaseResult) {
	pm.WritePacket(Packet{
		"name":          "test-case-status",
		"submission-id": submissionID,
		"cases": []map[string]interface{}{
			{
				"position": position,
				"status":   statusToNum(result.Status),
				"time":     result.Time,
				"memory":   result.Memory,
				"points":   result.Points,
				"total-points": result.TotalPoints,
				"feedback": result.Feedback,
			},
		},
	})
}

// SendCompileError sends a compile error notification.
func (pm *PacketManager) SendCompileError(submissionID uint, log string) {
	pm.WritePacket(Packet{
		"name":          "compile-error",
		"submission-id": submissionID,
		"log":           log,
	})
}

// SendInternalError sends an internal error notification.
func (pm *PacketManager) SendInternalError(submissionID uint, message string) {
	pm.WritePacket(Packet{
		"name":          "internal-error",
		"submission-id": submissionID,
		"message":       message,
	})
}

// SendSubmissionAborted sends submission aborted notification.
func (pm *PacketManager) SendSubmissionAborted(submissionID uint) {
	pm.WritePacket(Packet{
		"name":          "submission-terminated",
		"submission-id": submissionID,
	})
}

// SendPingResponse sends a ping response.
func (pm *PacketManager) SendPingResponse(when float64, load float64) {
	pm.WritePacket(Packet{
		"name": "ping-response",
		"when": when,
		"time": float64(time.Now().Unix()),
		"load": load,
	})
}

// Close closes the connection.
func (pm *PacketManager) Close() {
	if pm.conn != nil && !pm.closed {
		pm.conn.Close()
		pm.closed = true
	}
}

// statusToNum converts status string to protocol number.
func statusToNum(status string) int {
	// Protocol status codes:
	// 1 = WA, 2 = RTE, 4 = TLE, 8 = MLE, 16 = OLE, 32 = SC, 64 = IR
	switch status {
	case "AC":
		return 0
	case "WA":
		return 1
	case "RTE":
		return 2
	case "TLE":
		return 4
	case "MLE":
		return 8
	case "OLE":
		return 16
	case "SC":
		return 32
	case "IR":
		return 64
	default:
		return 1 // Default to WA
	}
}

// JudgeHandler is the interface for handling packets.
type JudgeHandler interface {
	HandlePacket(packet Packet) error
}

// GratingResult contains grading results.
type GratingResult struct {
	SubmissionID    uint
	Status          string
	Points          float64
	TotalPoints     float64
	Time            float64
	Memory          float64
	TestCaseResults []TestCaseResult
}

// TestCaseResult contains a single test case result.
type TestCaseResult struct {
	Position    int
	Status      string
	Time        float64
	Memory      float64
	Points      float64
	TotalPoints float64
	Feedback    string
}
