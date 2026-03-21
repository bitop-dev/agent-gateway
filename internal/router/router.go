package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bitop-dev/agent-gateway/internal/db"
	"github.com/bitop-dev/agent-gateway/internal/events"
)

// Router dispatches tasks to available workers.
type Router struct {
	DB         *db.DB
	Events     *events.Bus
	Client     *http.Client
	MaxRetries int
}

// NewRouter creates a task router.
func NewRouter(database *db.DB, bus *events.Bus) *Router {
	return &Router{
		DB:         database,
		Events:     bus,
		Client:     &http.Client{Timeout: 10 * time.Minute},
		MaxRetries: 2,
	}
}

// Dispatch sends a task to the best available worker and waits for the result.
// On transient failures (timeouts, connection errors), it retries on a different
// worker up to MaxRetries times.
func (r *Router) Dispatch(ctx context.Context, task *db.Task) (*db.Task, error) {
	var lastErr error
	totalStart := time.Now()

	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("router: retry %d/%d for task %s (previous: %v)", attempt, r.MaxRetries, task.ID, lastErr)
			r.Events.Publish(events.Event{
				Topic:   "agent.task.retry",
				TaskID:  task.ID,
				Profile: task.Profile,
				Message: fmt.Sprintf("retry %d/%d: %v", attempt, r.MaxRetries, lastErr),
			})
			// Brief backoff before retry.
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}

		// Find a worker — prefer one that already has the profile cached.
		worker, err := r.DB.FindWorkerForProfile(ctx, task.Profile)
		if err != nil || worker == nil {
			worker, err = r.DB.FindAnyIdleWorker(ctx)
			if err != nil || worker == nil {
				lastErr = fmt.Errorf("no available workers")
				continue
			}
		}

		// Mark worker as busy.
		r.DB.SetWorkerTask(ctx, worker.URL, task.ID)
		if attempt == 0 {
			r.DB.UpdateTaskStarted(ctx, task.ID, worker.URL)
			r.Events.Publish(events.Event{
				Topic:     events.TopicTaskStarted,
				TaskID:    task.ID,
				Profile:   task.Profile,
				WorkerURL: worker.URL,
			})
		}

		result, err := r.callWorker(ctx, worker.URL, task)
		r.DB.ClearWorkerTask(ctx, worker.URL)

		if err != nil {
			lastErr = err
			// Retry on transient errors, fail fast on permanent ones.
			if isTransientError(err) {
				continue
			}
			// Permanent error — don't retry.
			break
		}

		// Success.
		totalDuration := int(time.Since(totalStart).Milliseconds())
		r.DB.UpdateTaskCompleted(ctx, task.ID, result.Output, 0, totalDuration)

		// Record cost if worker reported token usage.
		if result.InputTokens > 0 || result.OutputTokens > 0 {
			cost := db.EstimateCost(result.Model, result.InputTokens, result.OutputTokens)
			r.DB.RecordCost(ctx, db.CostEntry{
				TaskID:        task.ID,
				Profile:       task.Profile,
				Model:         result.Model,
				InputTokens:   result.InputTokens,
				OutputTokens:  result.OutputTokens,
				TotalTokens:   result.InputTokens + result.OutputTokens,
				EstimatedCost: cost,
			})
		}

		r.Events.Publish(events.Event{
			Topic:     events.TopicTaskCompleted,
			TaskID:    task.ID,
			Profile:   task.Profile,
			WorkerURL: worker.URL,
			Data: map[string]any{
				"durationMs":   totalDuration,
				"attempts":     attempt + 1,
				"model":        result.Model,
				"inputTokens":  result.InputTokens,
				"outputTokens": result.OutputTokens,
			},
		})
		task.Status = "completed"
		task.Output = result.Output
		task.DurationMs = totalDuration
		task.WorkerURL = worker.URL
		return task, nil
	}

	// All retries exhausted.
	totalDuration := int(time.Since(totalStart).Milliseconds())
	errMsg := fmt.Sprintf("%v (after %d retries)", lastErr, r.MaxRetries)
	r.DB.UpdateTaskFailed(ctx, task.ID, errMsg, totalDuration)
	r.Events.Publish(events.Event{
		Topic:     events.TopicTaskFailed,
		TaskID:    task.ID,
		Profile:   task.Profile,
		Message:   errMsg,
		Data:      map[string]any{"durationMs": totalDuration, "attempts": r.MaxRetries + 1},
	})
	task.Status = "failed"
	task.Error = errMsg
	task.DurationMs = totalDuration
	return task, nil
}

// isTransientError returns true for errors that are worth retrying.
func isTransientError(err error) bool {
	msg := err.Error()
	transient := []string{
		"timeout", "Timeout",
		"connection refused",
		"connection reset",
		"i/o timeout",
		"EOF",
		"context deadline exceeded",
		"no available workers",
		"502", "503", "504",
	}
	for _, t := range transient {
		if strings.Contains(msg, t) {
			return true
		}
	}
	return false
}

type workerResponse struct {
	Status       string  `json:"status"`
	Output       string  `json:"output"`
	Error        string  `json:"error"`
	Duration     float64 `json:"duration"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
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
