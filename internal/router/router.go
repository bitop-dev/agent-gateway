package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bitop-dev/agent-gateway/internal/db"
	"github.com/bitop-dev/agent-gateway/internal/events"
)

// Router dispatches tasks to available workers.
type Router struct {
	DB     *db.DB
	Events *events.Bus
	Client *http.Client
}

// NewRouter creates a task router.
func NewRouter(database *db.DB, bus *events.Bus) *Router {
	return &Router{
		DB:     database,
		Events: bus,
		Client: &http.Client{Timeout: 10 * time.Minute},
	}
}

// Dispatch sends a task to the best available worker and waits for the result.
func (r *Router) Dispatch(ctx context.Context, task *db.Task) (*db.Task, error) {
	// Find a worker — prefer one that already has the profile cached.
	worker, err := r.DB.FindWorkerForProfile(ctx, task.Profile)
	if err != nil || worker == nil {
		// Fall back to any idle worker (on-demand install will handle the profile).
		worker, err = r.DB.FindAnyIdleWorker(ctx)
		if err != nil || worker == nil {
			return nil, fmt.Errorf("no available workers")
		}
	}

	// Mark worker as busy.
	r.DB.SetWorkerTask(ctx, worker.URL, task.ID)
	r.DB.UpdateTaskStarted(ctx, task.ID, worker.URL)

	r.Events.Publish(events.Event{
		Topic:     events.TopicTaskStarted,
		TaskID:    task.ID,
		Profile:   task.Profile,
		WorkerURL: worker.URL,
	})

	start := time.Now()

	// Dispatch to worker.
	result, err := r.callWorker(ctx, worker.URL, task)

	durationMs := int(time.Since(start).Milliseconds())

	// Clear worker task.
	r.DB.ClearWorkerTask(ctx, worker.URL)

	if err != nil {
		r.DB.UpdateTaskFailed(ctx, task.ID, err.Error(), durationMs)
		r.Events.Publish(events.Event{
			Topic:     events.TopicTaskFailed,
			TaskID:    task.ID,
			Profile:   task.Profile,
			WorkerURL: worker.URL,
			Message:   err.Error(),
			Data:      map[string]any{"durationMs": durationMs},
		})
		task.Status = "failed"
		task.Error = err.Error()
		task.DurationMs = durationMs
		return task, nil
	}

	r.DB.UpdateTaskCompleted(ctx, task.ID, result.Output, 0, durationMs)
	r.Events.Publish(events.Event{
		Topic:     events.TopicTaskCompleted,
		TaskID:    task.ID,
		Profile:   task.Profile,
		WorkerURL: worker.URL,
		Data:      map[string]any{"durationMs": durationMs},
	})
	task.Status = "completed"
	task.Output = result.Output
	task.DurationMs = durationMs
	task.WorkerURL = worker.URL
	return task, nil
}

type workerResponse struct {
	Status   string `json:"status"`
	Output   string `json:"output"`
	Error    string `json:"error"`
	Duration float64 `json:"duration"`
}

func (r *Router) callWorker(ctx context.Context, workerURL string, task *db.Task) (*workerResponse, error) {
	body := map[string]any{
		"profile": task.Profile,
		"task":    task.Task,
	}
	if len(task.Context) > 0 {
		body["context"] = task.Context
	}
	data, _ := json.Marshal(body)

	url := strings.TrimRight(workerURL, "/") + "/v1/task"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("worker %s: %w", workerURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("worker %s: read response: %w", workerURL, err)
	}

	var result workerResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("worker %s: decode response: %w", workerURL, err)
	}
	if result.Status == "failed" || result.Error != "" {
		return nil, fmt.Errorf("worker %s: %s", workerURL, result.Error)
	}
	return &result, nil
}
