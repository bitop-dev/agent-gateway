package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Task struct {
	ID           string         `json:"id"`
	Profile      string         `json:"profile"`
	Task         string         `json:"task"`
	Context      map[string]any `json:"context,omitempty"`
	Status       string         `json:"status"`
	Priority     string         `json:"priority"`
	WorkerURL    string         `json:"workerUrl,omitempty"`
	Output       string         `json:"output,omitempty"`
	Error        string         `json:"error,omitempty"`
	Model        string         `json:"model,omitempty"`
	InputTokens  int            `json:"inputTokens"`
	OutputTokens int            `json:"outputTokens"`
	ToolCalls    int            `json:"toolCalls"`
	DurationMs   int            `json:"durationMs"`
	CallbackURL  string         `json:"callbackUrl,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	StartedAt    *time.Time     `json:"startedAt,omitempty"`
	CompletedAt  *time.Time     `json:"completedAt,omitempty"`
}

func (d *DB) CreateTask(ctx context.Context, t Task) error {
	ctxJSON, _ := json.Marshal(t.Context)
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO tasks (id, profile, task, context, status, priority, callback_url, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Profile, t.Task, ctxJSON, t.Status, t.Priority, t.CallbackURL, t.CreatedAt)
	return err
}

func (d *DB) UpdateTaskStarted(ctx context.Context, id, workerURL string) error {
	now := time.Now()
	_, err := d.Pool.Exec(ctx,
		`UPDATE tasks SET status='running', worker_url=$2, started_at=$3 WHERE id=$1`,
		id, workerURL, now)
	return err
}

func (d *DB) UpdateTaskCompleted(ctx context.Context, id, output, model string, toolCalls, durationMs, inputTokens, outputTokens int) error {
	now := time.Now()
	_, err := d.Pool.Exec(ctx,
		`UPDATE tasks SET status='completed', output=$2, tool_calls=$3, duration_ms=$4, completed_at=$5,
		 model=$6, input_tokens=$7, output_tokens=$8 WHERE id=$1`,
		id, output, toolCalls, durationMs, now, model, inputTokens, outputTokens)
	return err
}

func (d *DB) UpdateTaskFailed(ctx context.Context, id, errMsg string, durationMs int) error {
	now := time.Now()
	_, err := d.Pool.Exec(ctx,
		`UPDATE tasks SET status='failed', error=$2, duration_ms=$3, completed_at=$4 WHERE id=$1`,
		id, errMsg, durationMs, now)
	return err
}

func (d *DB) GetTask(ctx context.Context, id string) (*Task, error) {
	var t Task
	var ctxJSON []byte
	err := d.Pool.QueryRow(ctx,
		`SELECT id, profile, task, context, status, priority, COALESCE(worker_url,''),
		        COALESCE(output,''), COALESCE(error,''), COALESCE(model,''),
		        COALESCE(input_tokens,0), COALESCE(output_tokens,0),
		        tool_calls, COALESCE(duration_ms,0),
		        COALESCE(callback_url,''), created_at, started_at, completed_at
		 FROM tasks WHERE id=$1`, id).
		Scan(&t.ID, &t.Profile, &t.Task, &ctxJSON, &t.Status, &t.Priority,
			&t.WorkerURL, &t.Output, &t.Error, &t.Model,
			&t.InputTokens, &t.OutputTokens,
			&t.ToolCalls, &t.DurationMs,
			&t.CallbackURL, &t.CreatedAt, &t.StartedAt, &t.CompletedAt)
	if err != nil {
		return nil, err
	}
	if len(ctxJSON) > 0 {
		json.Unmarshal(ctxJSON, &t.Context)
	}
	return &t, nil
}

func (d *DB) ListTasks(ctx context.Context, status string, limit int) ([]Task, error) {
	query := `SELECT id, profile, task, status, priority, COALESCE(worker_url,''),
	                 COALESCE(output,''), COALESCE(error,''), COALESCE(model,''),
	                 COALESCE(input_tokens,0), COALESCE(output_tokens,0),
	                 tool_calls, COALESCE(duration_ms,0),
	                 created_at, started_at, completed_at
	          FROM tasks`
	args := []any{}
	if status != "" {
		query += " WHERE status=$1"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT " + fmt.Sprintf("%d", limit)

	rows, err := d.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Profile, &t.Task, &t.Status, &t.Priority,
			&t.WorkerURL, &t.Output, &t.Error, &t.Model,
			&t.InputTokens, &t.OutputTokens,
			&t.ToolCalls, &t.DurationMs,
			&t.CreatedAt, &t.StartedAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
