package db

import (
	"context"
	"encoding/json"
	"time"
)

type Schedule struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	CronExpr  string         `json:"cron"`
	Timezone  string         `json:"timezone"`
	Profile   string         `json:"profile"`
	Task      string         `json:"task"`
	Context   map[string]any `json:"context,omitempty"`
	Enabled   bool           `json:"enabled"`
	LastRun   *time.Time     `json:"lastRun,omitempty"`
	NextRun   *time.Time     `json:"nextRun,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

func (d *DB) CreateSchedule(ctx context.Context, s Schedule) error {
	ctxJSON, _ := json.Marshal(s.Context)
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO schedules (id, name, cron_expr, timezone, profile, task, context, enabled, next_run, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		s.ID, s.Name, s.CronExpr, s.Timezone, s.Profile, s.Task, ctxJSON, s.Enabled, s.NextRun, time.Now())
	return err
}

func (d *DB) ListSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, cron_expr, timezone, profile, task, context, enabled, last_run, next_run, created_at
		 FROM schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		var ctxJSON []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.CronExpr, &s.Timezone, &s.Profile, &s.Task,
			&ctxJSON, &s.Enabled, &s.LastRun, &s.NextRun, &s.CreatedAt); err != nil {
			return nil, err
		}
		if len(ctxJSON) > 0 {
			json.Unmarshal(ctxJSON, &s.Context)
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (d *DB) GetDueSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, cron_expr, timezone, profile, task, context, enabled, last_run, next_run, created_at
		 FROM schedules WHERE enabled=true AND next_run <= $1`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		var ctxJSON []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.CronExpr, &s.Timezone, &s.Profile, &s.Task,
			&ctxJSON, &s.Enabled, &s.LastRun, &s.NextRun, &s.CreatedAt); err != nil {
			return nil, err
		}
		if len(ctxJSON) > 0 {
			json.Unmarshal(ctxJSON, &s.Context)
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (d *DB) UpdateScheduleRun(ctx context.Context, id string, lastRun, nextRun time.Time) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE schedules SET last_run=$2, next_run=$3 WHERE id=$1`,
		id, lastRun, nextRun)
	return err
}

func (d *DB) UpdateSchedule(ctx context.Context, s Schedule) error {
	ctxJSON, _ := json.Marshal(s.Context)
	_, err := d.Pool.Exec(ctx,
		`UPDATE schedules SET name=$2, cron_expr=$3, timezone=$4, profile=$5, task=$6,
		 context=$7, enabled=$8, next_run=$9 WHERE id=$1`,
		s.ID, s.Name, s.CronExpr, s.Timezone, s.Profile, s.Task, ctxJSON, s.Enabled, s.NextRun)
	return err
}

func (d *DB) DeleteSchedule(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM schedules WHERE id=$1`, id)
	return err
}
