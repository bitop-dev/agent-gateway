package events

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// Event topics published by the gateway.
const (
	TopicTaskSubmitted  = "agent.task.submitted"
	TopicTaskStarted    = "agent.task.started"
	TopicTaskCompleted  = "agent.task.completed"
	TopicTaskFailed     = "agent.task.failed"
	TopicWorkerJoined   = "agent.worker.joined"
	TopicWorkerLost     = "agent.worker.lost"
	TopicScheduleFired  = "agent.schedule.fired"
	TopicWebhookFired   = "agent.webhook.fired"
)

// Event is the standard envelope for all published events.
type Event struct {
	Topic     string         `json:"topic"`
	TaskID    string         `json:"taskId,omitempty"`
	Profile   string         `json:"profile,omitempty"`
	WorkerURL string         `json:"workerUrl,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// Bus wraps a NATS connection for publishing events.
// If NATS is not configured, events are logged but not published.
type Bus struct {
	conn *nats.Conn
}

// Connect creates a Bus connected to NATS. If natsURL is empty, returns
// a no-op bus that only logs events.
func Connect(natsURL string) (*Bus, error) {
	if natsURL == "" {
		log.Printf("events: NATS not configured — events will be logged only")
		return &Bus{}, nil
	}
	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Printf("events: NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Printf("events: NATS reconnected")
		}),
	)
	if err != nil {
		return nil, err
	}
	log.Printf("events: connected to NATS at %s", natsURL)
	return &Bus{conn: nc}, nil
}

// Publish sends an event to NATS and logs it.
func (b *Bus) Publish(e Event) {
	e.Timestamp = time.Now()
	data, _ := json.Marshal(e)

	log.Printf("event: %s %s", e.Topic, string(data))

	if b.conn != nil {
		if err := b.conn.Publish(e.Topic, data); err != nil {
			log.Printf("events: publish %s failed: %v", e.Topic, err)
		}
	}
}

// Subscribe registers a handler for a topic pattern (supports NATS wildcards).
func (b *Bus) Subscribe(topic string, handler func(Event)) error {
	if b.conn == nil {
		return nil // no-op when NATS not configured
	}
	_, err := b.conn.Subscribe(topic, func(msg *nats.Msg) {
		var e Event
		if err := json.Unmarshal(msg.Data, &e); err != nil {
			return
		}
		handler(e)
	})
	return err
}

// Close shuts down the NATS connection.
func (b *Bus) Close() {
	if b.conn != nil {
		b.conn.Drain()
	}
}
