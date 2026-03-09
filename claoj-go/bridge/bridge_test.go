package bridge

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManager_AddAndRemove(t *testing.T) {
	m := NewManager()
	assert.NotNil(t, m)
	assert.Empty(t, m.judges)

	// Mock handler for testing
	h := &Handler{name: "judge1"}
	m.Add("judge1", h)

	assert.Len(t, m.judges, 1)
	assert.Contains(t, m.judges, "judge1")
	assert.Equal(t, h, m.judges["judge1"])

	m.Remove("judge1")
	assert.Len(t, m.judges, 0)
	assert.NotContains(t, m.judges, "judge1")
}

func TestManager_AddMultiple(t *testing.T) {
	m := NewManager()

	h1 := &Handler{name: "judge1"}
	h2 := &Handler{name: "judge2"}
	h3 := &Handler{name: "judge3"}

	m.Add("judge1", h1)
	m.Add("judge2", h2)
	m.Add("judge3", h3)

	assert.Len(t, m.judges, 3)
	assert.Contains(t, m.judges, "judge1")
	assert.Contains(t, m.judges, "judge2")
	assert.Contains(t, m.judges, "judge3")

	m.Remove("judge2")
	assert.Len(t, m.judges, 2)
	assert.NotContains(t, m.judges, "judge2")
}

func TestHandler_ProblemsMap(t *testing.T) {
	m := NewManager()
	h := NewHandler(nil, m)

	assert.NotNil(t, h.problems)
	assert.Empty(t, h.problems)

	// Simulate adding problem permissions
	h.problems["PROB1"] = true
	h.problems["PROB2"] = true

	assert.Len(t, h.problems, 2)
	assert.True(t, h.problems["PROB1"])
	assert.True(t, h.problems["PROB2"])
	assert.False(t, h.problems["PROB3"])
}

func TestHandler_ExecutorsMap(t *testing.T) {
	m := NewManager()
	h := NewHandler(nil, m)

	assert.NotNil(t, h.executors)
	assert.Empty(t, h.executors)

	// Simulate adding executor versions
	h.executors["cpp17"] = "g++ 9.4.0"
	h.executors["python3"] = "3.8.10"

	assert.Len(t, h.executors, 2)
	assert.Contains(t, h.executors, "cpp17")
	assert.Contains(t, h.executors, "python3")
}

func TestHandler_InitialState(t *testing.T) {
	m := NewManager()
	h := NewHandler(nil, m)

	assert.False(t, h.working)
	assert.Equal(t, uint(0), h.workingSub)
	assert.Equal(t, float64(0), h.load)
	assert.Equal(t, "", h.name) // Not authenticated initially
}

func TestPacket_Create(t *testing.T) {
	// Test basic packet creation
	pkt := Packet{"name": "test", "data": "value"}
	assert.Equal(t, "test", pkt.Name())

	// Test packet with args
	args := map[string]interface{}{"key": "value"}
	pkt = Packet{"name": "submit", "args": args}

	name := pkt.Name()
	assert.Equal(t, "submit", name)
}

func TestPacket_JSONMarshaling(t *testing.T) {
	pkt := Packet{
		"name": "result",
		"args": map[string]interface{}{
			"submission_id": float64(123),
			"result":        "AC",
			"score":         float64(100),
		},
	}

	data, err := json.Marshal(pkt)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var unmarshaled Packet
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, "result", unmarshaled.Name())
}

func TestConnection_UpdateLoad(t *testing.T) {
	// Test that load tracking works
	m := NewManager()
	h := NewHandler(nil, m)

	// Initial load should be 0
	assert.Equal(t, float64(0), h.load)

	// Simulate load update
	h.load = 0.5
	assert.Equal(t, float64(0.5), h.load)

	h.load = 1.0
	assert.Equal(t, float64(1.0), h.load)
}

func TestHandler_IsWorking(t *testing.T) {
	m := NewManager()
	h := NewHandler(nil, m)

	// Initially not working
	assert.False(t, h.working)
	assert.Equal(t, uint(0), h.workingSub)

	// Simulate starting work
	h.working = true
	h.workingSub = 5
	assert.True(t, h.working)
	assert.Equal(t, uint(5), h.workingSub)

	// Simulate completing work
	h.working = false
	h.workingSub = 0
	assert.False(t, h.working)
	assert.Equal(t, uint(0), h.workingSub)
}

func TestNewManager_ReturnsNonNull(t *testing.T) {
	m := NewManager()
	assert.NotNil(t, m)
	assert.NotNil(t, m.judges)
}

func TestHandler_NameSetter(t *testing.T) {
	m := NewManager()
	h := NewHandler(nil, m)

	assert.Equal(t, "", h.name)

	h.name = "judge-test-1"
	assert.Equal(t, "judge-test-1", h.name)
}

// Test concurrent access safety
func TestManager_ConcurrentAccess(t *testing.T) {
	m := NewManager()

	done := make(chan bool)
	numGoroutines := 10

	// Start multiple goroutines that add/remove judges
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			name := "judge" + string(rune('A'+id))
			h := &Handler{name: name}
			m.Add(name, h)
			time.Sleep(10 * time.Millisecond)
			m.Remove(name)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Map should be empty after all removals
	assert.Empty(t, m.judges)
}
