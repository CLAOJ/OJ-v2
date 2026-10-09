package bridge

// Packet is one DMOJ judge protocol message: a JSON object with a "name"
// field. Judge agents POST them to /internal/judge/event unchanged. JSON
// numbers decode as float64.
type Packet map[string]interface{}

// Name returns the "name" field of the packet (the event type).
func (p Packet) Name() string {
	if n, ok := p["name"].(string); ok {
		return n
	}
	return ""
}

// SubmissionID returns the packet's "submission-id".
func (p Packet) SubmissionID() (uint, bool) {
	v, ok := p["submission-id"].(float64)
	if !ok || v <= 0 {
		return 0, false
	}
	return uint(v), true
}
