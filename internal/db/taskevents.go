package db

import (
	"sync"
	"time"
)

// TaskEvent represents a real-time event from a running task.
type TaskEvent struct {
	Type    string `json:"type"`    // tool_requested, tool_started, tool_finished, sub_agent_spawned, etc.
	Tool    string `json:"tool"`    // tool name (e.g. "web/search", "agent/spawn")
	Message string `json:"message"` // human-readable description
	Time    string `json:"time"`
}

// TaskEventBuffer stores in-memory events for running tasks.
// Events are kept until the task completes, then cleaned up.
type TaskEventBuffer struct {
	mu     sync.RWMutex
	events map[string][]TaskEvent // taskID → events
}

func NewTaskEventBuffer() *TaskEventBuffer {
	return &TaskEventBuffer{events: make(map[string][]TaskEvent)}
}

func (b *TaskEventBuffer) Push(taskID string, event TaskEvent) {
	if event.Time == "" {
		event.Time = time.Now().UTC().Format(time.RFC3339)
	}
	b.mu.Lock()
	b.events[taskID] = append(b.events[taskID], event)
	// Cap at 100 events per task
	if len(b.events[taskID]) > 100 {
		b.events[taskID] = b.events[taskID][len(b.events[taskID])-100:]
	}
	b.mu.Unlock()
}

func (b *TaskEventBuffer) Get(taskID string) []TaskEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	events := b.events[taskID]
	if events == nil {
		return []TaskEvent{}
	}
	out := make([]TaskEvent, len(events))
	copy(out, events)
	return out
}

func (b *TaskEventBuffer) Clear(taskID string) {
	b.mu.Lock()
	delete(b.events, taskID)
	b.mu.Unlock()
}

// Cleanup removes events older than maxAge
func (b *TaskEventBuffer) Cleanup(maxAge time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for taskID, events := range b.events {
		if len(events) > 0 {
			last, _ := time.Parse(time.RFC3339, events[len(events)-1].Time)
			if last.Before(cutoff) {
				delete(b.events, taskID)
			}
		}
	}
}
