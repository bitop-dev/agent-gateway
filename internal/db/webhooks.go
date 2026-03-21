package db

import (
	"context"
	"encoding/json"
	"time"
)

type Webhook struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Path            string         `json:"path"`
	Profile         string         `json:"profile"`
	TaskTemplate    string         `json:"taskTemplate"`
	ContextTemplate map[string]any `json:"contextTemplate,omitempty"`
	AuthToken       string         `json:"authToken,omitempty"`
	Enabled         bool           `json:"enabled"`
	CreatedAt       time.Time      `json:"createdAt"`
}

func (d *DB) CreateWebhook(ctx context.Context, w Webhook) error {
	ctxJSON, _ := json.Marshal(w.ContextTemplate)
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO webhooks (id, name, path, profile, task_template, context_template, auth_token, enabled, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		w.ID, w.Name, w.Path, w.Profile, w.TaskTemplate, ctxJSON, w.AuthToken, w.Enabled, time.Now())
	return err
}

func (d *DB) GetWebhookByPath(ctx context.Context, path string) (*Webhook, error) {
	var w Webhook
	var ctxJSON []byte
	err := d.Pool.QueryRow(ctx,
		`SELECT id, name, path, profile, task_template, context_template, COALESCE(auth_token,''), enabled, created_at
		 FROM webhooks WHERE path=$1 AND enabled=true`, path).
		Scan(&w.ID, &w.Name, &w.Path, &w.Profile, &w.TaskTemplate, &ctxJSON, &w.AuthToken, &w.Enabled, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	if len(ctxJSON) > 0 {
		json.Unmarshal(ctxJSON, &w.ContextTemplate)
	}
	return &w, nil
}

func (d *DB) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, path, profile, task_template, COALESCE(auth_token,''), enabled, created_at
		 FROM webhooks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var webhooks []Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.Name, &w.Path, &w.Profile, &w.TaskTemplate, &w.AuthToken, &w.Enabled, &w.CreatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, nil
}

func (d *DB) DeleteWebhook(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM webhooks WHERE id=$1`, id)
	return err
}
