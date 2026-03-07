package bridge

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	v2 "github.com/CLAOJ/claoj-go/api/v2"
	"github.com/CLAOJ/claoj-go/contest_format"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
)

// Manager keeps track of all connected judges.
type Manager struct {
	sync.RWMutex
	judges map[string]*Handler
}

func NewManager() *Manager {
	return &Manager{
		judges: make(map[string]*Handler),
	}
}

func (m *Manager) Add(name string, h *Handler) {
	m.Lock()
	defer m.Unlock()
	m.judges[name] = h
}

func (m *Manager) Remove(name string) {
	m.Lock()
	defer m.Unlock()
	delete(m.judges, name)
}

// Handler manages a single judge connection's state machine.
type Handler struct {
	conn       *Connection
	manager    *Manager
	name       string
	authKey    string
	problems   map[string]bool
	executors  map[string]interface{}
	working    bool
	workingSub uint
	load       float64
}

func NewHandler(conn *Connection, manager *Manager) *Handler {
	return &Handler{
		conn:      conn,
		manager:   manager,
		problems:  make(map[string]bool),
		executors: make(map[string]interface{}),
	}
}

// loop eternally reads packets from the judge and dispatches them.
func (h *Handler) loop() {
	// Start a goroutine to send periodic pings to keep the connection alive
	// and monitor judge health. Send ping every 45 seconds (under 60s timeout).
	go func() {
		ticker := time.NewTicker(45 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if h.name == "" {
				continue // Not authenticated yet
			}
			if err := h.conn.WritePacket(Packet{"name": "ping", "when": time.Now().Unix()}); err != nil {
				log.Printf("bridge [%s]: ping error: %v", h.conn.RemoteAddr(), err)
				return
			}
		}
	}()

	for {
		_ = h.conn.SetDeadline(time.Now().Add(60 * time.Second))
		pkt, err := h.conn.ReadPacket()
		if err != nil {
			log.Printf("bridge [%s]: read error: %v", h.conn.RemoteAddr(), err)
			break
		}

		if err := h.handlePacket(pkt); err != nil {
			log.Printf("bridge [%s]: handle packet %q error: %v", h.conn.RemoteAddr(), pkt.Name(), err)
			// Disconnect on protocol errors
			break
		}
	}

	h.cleanup()
}

func (h *Handler) handlePacket(pkt Packet) error {
	name := pkt.Name()

	// Before handshake, we only accept 'handshake'
	if h.name == "" && name != "handshake" {
		return fmt.Errorf("expected handshake, got %q", name)
	}

	switch name {
	case "handshake":
		return h.onHandshake(pkt)
	case "supported-problems":
		return h.onSupportedProblems(pkt)
	case "ping-response":
		return h.onPingResponse(pkt)
	case "grading-begin":
		return h.onGradingBegin(pkt)
	case "grading-end":
		return h.onGradingEnd(pkt)
	case "test-case-status":
		return h.onTestCase(pkt)
	case "batch-begin":
		return nil // ignoring batch for now
	case "batch-end":
		return nil // ignoring batch for now
	case "compile-error", "compile-message":
		return h.onCompileError(pkt)
	case "internal-error", "submission-terminated":
		return h.onInternalError(pkt)
	case "submission-acknowledged":
		return h.onSubmissionAcknowledged(pkt)
	default:
		log.Printf("bridge [%s]: unhandled packet %q", h.conn.RemoteAddr(), name)
		return nil
	}
}

// onHandshake verifies the auth key
func (h *Handler) onHandshake(pkt Packet) error {
	judgeID, ok := pkt["id"].(string)
	if !ok {
		return errors.New("invalid handshake: id must be a string")
	}
	key, ok := pkt["key"].(string)
	if !ok {
		return errors.New("invalid handshake: key must be a string")
	}

	var judge models.Judge
	if err := db.DB.Where("name = ?", judgeID).First(&judge).Error; err != nil {
		return fmt.Errorf("judge %q not found", judgeID)
	}

	if subtle.ConstantTimeCompare([]byte(judge.AuthKey), []byte(key)) != 1 {
		return fmt.Errorf("invalid auth key for %q", judgeID)
	}

	if judge.IsBlocked {
		return fmt.Errorf("judge %q is blocked", judgeID)
	}

	// Authenticated
	h.name = judgeID
	h.authKey = key
	h.manager.Add(judgeID, h)

	if problems, ok := pkt["problems"].(map[string]interface{}); ok {
		for code := range problems {
			h.problems[code] = true
		}
	}
	if executors, ok := pkt["executors"].(map[string]interface{}); ok {
		h.executors = executors
	}

	log.Printf("bridge [%s]: authenticated as %s", h.conn.RemoteAddr(), h.name)

	// Update DB to online
	var judgeAddr string
	if addr, ok := h.conn.RemoteAddr().(*net.TCPAddr); ok {
		judgeAddr = addr.IP.String()
	}
	db.DB.Model(&judge).Updates(map[string]interface{}{
		"online":  true,
		"last_ip": judgeAddr,
	})

	// Send handshake-success response
	if err := h.conn.WritePacket(Packet{"name": "handshake-success"}); err != nil {
		return fmt.Errorf("failed to send handshake-success: %w", err)
	}

	return nil
}

func (h *Handler) cleanup() {
	if h.name != "" {
		h.manager.Remove(h.name)
		db.DB.Model(&models.Judge{}).Where("name = ?", h.name).Update("online", false)
		log.Printf("bridge [%s]: disconnected", h.name)
	}
}

func (h *Handler) onSupportedProblems(pkt Packet) error {
	log.Printf("bridge [%s]: updated problems list", h.name)
	return nil
}

func (h *Handler) onPingResponse(pkt Packet) error {
	// Update judge load metric if provided
	if load, ok := pkt["load"].(float64); ok {
		h.load = load
	}
	// Reset connection deadline on ping response
	_ = h.conn.SetDeadline(time.Now().Add(60 * time.Second))
	return nil
}

func (h *Handler) onSubmissionAcknowledged(pkt Packet) error {
	subIDVal, ok := pkt["submission-id"].(float64)
	if !ok {
		return errors.New("invalid submission-id: must be a number")
	}
	subID := uint(subIDVal)
	db.DB.Model(&models.Submission{}).Where("id = ?", subID).Update("status", "P")
	PostSubmissionState(subID, "processing", nil)
	return nil
}

func (h *Handler) onGradingBegin(pkt Packet) error {
	subIDVal, ok := pkt["submission-id"].(float64)
	if !ok {
		return errors.New("invalid submission-id: must be a number")
	}
	subID := uint(subIDVal)
	log.Printf("bridge [%s]: grading begin for sub %d", h.name, subID)

	db.DB.Model(&models.Submission{}).Where("id = ?", subID).Updates(map[string]interface{}{
		"status":           "G",
		"current_testcase": 1,
		"batch":            false,
		"judged_date":      time.Now(),
	})
	db.DB.Where("submission_id = ?", subID).Delete(&models.SubmissionTestCase{})

	PostSubmissionState(subID, "grading-begin", nil)
	return nil
}

func (h *Handler) onTestCase(pkt Packet) error {
	subIDVal, ok := pkt["submission-id"].(float64)
	if !ok {
		return errors.New("invalid submission-id: must be a number")
	}
	subID := uint(subIDVal)

	casesRaw, ok := pkt["cases"].([]interface{})
	if !ok {
		return errors.New("invalid cases: must be an array")
	}

	var maxPos int
	var bulk []models.SubmissionTestCase

	for _, cInterface := range casesRaw {
		c, ok := cInterface.(map[string]interface{})
		if !ok {
			continue // Skip invalid case entries
		}

		posVal, ok := c["position"].(float64)
		if !ok {
			continue
		}
		pos := int(posVal)
		if pos > maxPos {
			maxPos = pos
		}

		statusNumVal, ok := c["status"].(float64)
		if !ok {
			continue
		}
		statusNum := int(statusNumVal)

		statusCode := "AC"
		if statusNum&4 != 0 {
			statusCode = "TLE"
		} else if statusNum&8 != 0 {
			statusCode = "MLE"
		} else if statusNum&64 != 0 {
			statusCode = "OLE"
		} else if statusNum&2 != 0 {
			statusCode = "RTE"
		} else if statusNum&16 != 0 {
			statusCode = "IR"
		} else if statusNum&1 != 0 {
			statusCode = "WA"
		} else if statusNum&32 != 0 {
			statusCode = "SC"
		}

		// Safe type assertions with defaults
		var timeLimit, memLimit, points, totalPts float64
		var feedback, extFeedback, output string

		if v, ok := c["time"].(float64); ok {
			timeLimit = v
		}
		if v, ok := c["memory"].(float64); ok {
			memLimit = v
		}
		if v, ok := c["points"].(float64); ok {
			points = v
		}
		if v, ok := c["total-points"].(float64); ok {
			totalPts = v
		}
		if v, ok := c["feedback"].(string); ok {
			feedback = v
		}
		if v, ok := c["extended-feedback"].(string); ok {
			extFeedback = v
		}
		if v, ok := c["output"].(string); ok {
			output = v
		}

		bulk = append(bulk, models.SubmissionTestCase{
			SubmissionID:     subID,
			Case:             pos,
			Status:           statusCode,
			Time:             &timeLimit,
			Memory:           &memLimit,
			Points:           &points,
			Total:            &totalPts,
			Feedback:         feedback,
			ExtendedFeedback: extFeedback,
			Output:           output,
		})
	}

	db.DB.Create(&bulk)
	db.DB.Model(&models.Submission{}).Where("id = ?", subID).Update("current_testcase", maxPos+1)

	PostSubmissionState(subID, "test-case", map[string]interface{}{"id": maxPos})
	return nil
}

func (h *Handler) onGradingEnd(pkt Packet) error {
	subIDVal, ok := pkt["submission-id"].(float64)
	if !ok {
		return errors.New("invalid submission-id: must be a number")
	}
	subID := uint(subIDVal)
	log.Printf("bridge [%s]: grading end for sub %d", h.name, subID)

	var cases []models.SubmissionTestCase
	db.DB.Where("submission_id = ?", subID).Find(&cases)

	var maxMem float64
	var sumTime float64
	var pts, total float64
	var worstStatus = "AC"

	statusOrder := map[string]int{"AC": 0, "SC": 1, "WA": 2, "MLE": 3, "TLE": 4, "IR": 5, "RTE": 6, "OLE": 7}

	for _, c := range cases {
		if c.Time != nil {
			sumTime += *c.Time
		}
		if c.Memory != nil && *c.Memory > maxMem {
			maxMem = *c.Memory
		}
		if c.Points != nil {
			pts += *c.Points
		}
		if c.Total != nil {
			total += *c.Total
		}

		if statusOrder[c.Status] > statusOrder[worstStatus] {
			worstStatus = c.Status
		}
	}

	var sub models.Submission
	db.DB.Preload("Problem").Where("id = ?", subID).First(&sub)

	finalPts := 0.0
	if total > 0 {
		finalPts = (pts / total) * sub.Problem.Points
	}
	if !sub.Problem.Partial && finalPts != sub.Problem.Points {
		finalPts = 0
	}

	db.DB.Model(&sub).Updates(map[string]interface{}{
		"status":      "D",
		"result":      worstStatus,
		"time":        sumTime,
		"memory":      maxMem,
		"points":      finalPts,
		"case_points": pts,
		"case_total":  total,
	})

	PostSubmissionState(subID, "grading-end", map[string]interface{}{
		"time": sumTime, "memory": maxMem, "points": pts, "total": total, "result": worstStatus,
	})
	PostGlobalSubmissionUpdate(subID, "grading-end", true)

	// Create notification for submission result
	go func() {
		_, _ = v2.CreateSubmissionResultNotification(sub.UserID, int(subID), sub.Problem.Code, sub.Problem.Name, worstStatus)
	}()

	// ---------------------------------------------------------
	// NEW: Update Contest Participation scores if applicable
	// ---------------------------------------------------------
	var cSub models.ContestSubmission
	if err := db.DB.Preload("Participation.Contest").Where("submission_id = ?", subID).First(&cSub).Error; err == nil {
		// 1. Update the ContestSubmission score
		cSub.Points = finalPts
		db.DB.Save(&cSub)

		// 2. Recompute the whole Participation using the appropriate ContestFormat
		cf := contest_format.GetFormat(cSub.Participation.Contest.FormatName, &cSub.Participation.Contest, cSub.Participation.Contest.FormatConfig)
		if err := cf.UpdateParticipation(&cSub.Participation); err != nil {
			log.Printf("bridge [%s]: failed to update contest participation: %v", h.name, err)
		}
	}

	return nil
}

func (h *Handler) onCompileError(pkt Packet) error {
	subIDVal, ok := pkt["submission-id"].(float64)
	if !ok {
		return errors.New("invalid submission-id: must be a number")
	}
	subID := uint(subIDVal)

	// Safe type assertion with default
	var logRaw string
	if v, ok := pkt["log"].(string); ok {
		logRaw = v
	}

	db.DB.Model(&models.Submission{}).Where("id = ?", subID).Updates(map[string]interface{}{
		"status": "CE", "result": "CE", "error": logRaw,
	})
	PostSubmissionState(subID, "compile-error", map[string]interface{}{"log": logRaw})
	PostGlobalSubmissionUpdate(subID, "compile-error", true)
	return nil
}

func (h *Handler) onInternalError(pkt Packet) error {
	subIDVal, ok := pkt["submission-id"].(float64)
	if !ok {
		return errors.New("invalid submission-id: must be a number")
	}
	subID := uint(subIDVal)

	// Safe type assertion with default
	var msg string
	if v, ok := pkt["message"].(string); ok {
		msg = v
	}

	db.DB.Model(&models.Submission{}).Where("id = ?", subID).Updates(map[string]interface{}{
		"status": "IE", "result": "IE", "error": msg,
	})
	PostSubmissionState(subID, "internal-error", nil)
	PostGlobalSubmissionUpdate(subID, "internal-error", true)
	return nil
}

// Abort sends an abort request to the judge for a specific submission
func (h *Handler) Abort(subID uint) error {
	pkt := Packet{
		"name":          "submission-abort",
		"submission-id": subID,
	}
	return h.conn.WritePacket(pkt)
}
