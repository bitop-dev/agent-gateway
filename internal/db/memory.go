package db

import (
	"context"
	"time"
)

type MemoryEntry struct {
	ID        int       `json:"id"`
	Profile   string    `json:"profile"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (d *DB) Remember(ctx context.Context, profile, key, value string) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO agent_memory (profile, key, value, created_at, updated_at)
		 VALUES ($1, $2, $3, now(), now())
		 ON CONFLICT (profile, key) DO UPDATE SET value=EXCLUDED.value, updated_at=now()`,
		profile, key, value)
	return err
}

func (d *DB) Recall(ctx context.Context, profile, key string) (string, error) {
	var value string
	err := d.Pool.QueryRow(ctx,
		`SELECT value FROM agent_memory WHERE profile=$1 AND key=$2`, profile, key).Scan(&value)
	return value, err
}

func (d *DB) RecallAll(ctx context.Context, profile string) ([]MemoryEntry, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, profile, key, value, created_at, updated_at
		 FROM agent_memory WHERE profile=$1 ORDER BY updated_at DESC`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		if err := rows.Scan(&e.ID, &e.Profile, &e.Key, &e.Value, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (d *DB) Forget(ctx context.Context, profile, key string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM agent_memory WHERE profile=$1 AND key=$2`, profile, key)
	return err
}

func (d *DB) ForgetAll(ctx context.Context, profile string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM agent_memory WHERE profile=$1`, profile)
	return err
}
