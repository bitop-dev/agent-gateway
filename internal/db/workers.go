package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Worker struct {
	URL            string    `json:"url"`
	Profiles       []string  `json:"profiles"`
	Capabilities   []string  `json:"capabilities"`
	Status         string    `json:"status"`
	CurrentTask    string    `json:"currentTask,omitempty"`
	TasksCompleted int       `json:"tasksCompleted"`
	RegisteredAt   time.Time `json:"registeredAt"`
	LastHeartbeat  time.Time `json:"lastHeartbeat"`
}

func (d *DB) UpsertWorker(ctx context.Context, w Worker) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO workers (url, profiles, capabilities, status, registered_at, last_heartbeat)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (url) DO UPDATE SET
		   profiles=EXCLUDED.profiles,
		   capabilities=EXCLUDED.capabilities,
		   status='active',
		   last_heartbeat=EXCLUDED.last_heartbeat`,
		w.URL, w.Profiles, w.Capabilities, "active", w.RegisteredAt, w.LastHeartbeat)
	return err
}

func (d *DB) SetWorkerTask(ctx context.Context, url, taskID string) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE workers SET current_task=$2 WHERE url=$1`, url, taskID)
	return err
}

func (d *DB) ClearWorkerTask(ctx context.Context, url string) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE workers SET current_task=NULL, tasks_completed=tasks_completed+1 WHERE url=$1`, url)
	return err
}

func (d *DB) RemoveWorker(ctx context.Context, url string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM workers WHERE url=$1`, url)
	return err
}

func (d *DB) ListWorkers(ctx context.Context) ([]Worker, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT url, profiles, capabilities, status, COALESCE(current_task,''),
		        tasks_completed, registered_at, last_heartbeat
		 FROM workers WHERE status='active' ORDER BY url`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectWorkers(rows)
}

func (d *DB) FindWorkerForProfile(ctx context.Context, profile string) (*Worker, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT url, profiles, capabilities, status, COALESCE(current_task,''),
		        tasks_completed, registered_at, last_heartbeat
		 FROM workers
		 WHERE status='active' AND current_task IS NULL AND $1 = ANY(profiles)
		 ORDER BY tasks_completed ASC
		 LIMIT 1`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	workers, err := collectWorkers(rows)
	if err != nil || len(workers) == 0 {
		return nil, err
	}
	return &workers[0], nil
}

func (d *DB) FindAnyIdleWorker(ctx context.Context) (*Worker, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT url, profiles, capabilities, status, COALESCE(current_task,''),
		        tasks_completed, registered_at, last_heartbeat
		 FROM workers
		 WHERE status='active' AND current_task IS NULL
		 ORDER BY tasks_completed ASC
		 LIMIT 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	workers, err := collectWorkers(rows)
	if err != nil || len(workers) == 0 {
		return nil, err
	}
	return &workers[0], nil
}

func (d *DB) MarkStaleWorkers(ctx context.Context, timeout time.Duration) (int, error) {
	result, err := d.Pool.Exec(ctx,
		`UPDATE workers SET status='stale' WHERE status='active' AND last_heartbeat < $1`,
		time.Now().Add(-timeout))
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

func collectWorkers(rows pgx.Rows) ([]Worker, error) {
	var workers []Worker
	for rows.Next() {
		var w Worker
		if err := rows.Scan(&w.URL, &w.Profiles, &w.Capabilities, &w.Status,
			&w.CurrentTask, &w.TasksCompleted, &w.RegisteredAt, &w.LastHeartbeat); err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}
	return workers, nil
}
