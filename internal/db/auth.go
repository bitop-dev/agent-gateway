package db

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"
)

type APIKey struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsed  *time.Time `json:"lastUsed,omitempty"`
	Revoked   bool      `json:"revoked"`
}

func HashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h)
}

func (d *DB) CreateAPIKey(ctx context.Context, id, name, keyHash string, scopes []string) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO api_keys (id, name, key_hash, scopes, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, name, keyHash, scopes, time.Now())
	return err
}

func (d *DB) ValidateAPIKey(ctx context.Context, keyHash string) (*APIKey, error) {
	var k APIKey
	err := d.Pool.QueryRow(ctx,
		`UPDATE api_keys SET last_used=now()
		 WHERE key_hash=$1 AND revoked=false
		 RETURNING id, name, scopes, created_at, last_used, revoked`,
		keyHash).Scan(&k.ID, &k.Name, &k.Scopes, &k.CreatedAt, &k.LastUsed, &k.Revoked)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (d *DB) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, scopes, created_at, last_used, revoked
		 FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.Scopes, &k.CreatedAt, &k.LastUsed, &k.Revoked); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (d *DB) RevokeAPIKey(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE api_keys SET revoked=true WHERE id=$1`, id)
	return err
}

func (d *DB) HasScope(key *APIKey, scope string) bool {
	for _, s := range key.Scopes {
		if s == scope || s == "admin" {
			return true
		}
	}
	return false
}
