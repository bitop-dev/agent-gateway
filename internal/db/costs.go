package db

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CostEntry struct {
	TaskID        string    `json:"taskId"`
	Profile       string    `json:"profile"`
	Model         string    `json:"model"`
	InputTokens   int       `json:"inputTokens"`
	OutputTokens  int       `json:"outputTokens"`
	TotalTokens   int       `json:"totalTokens"`
	EstimatedCost float64   `json:"estimatedCost"` // USD
	CreatedAt     time.Time `json:"createdAt"`
}

type CostSummary struct {
	Profile       string  `json:"profile"`
	TotalTasks    int     `json:"totalTasks"`
	TotalTokens   int     `json:"totalTokens"`
	InputTokens   int     `json:"inputTokens"`
	OutputTokens  int     `json:"outputTokens"`
	TotalCost     float64 `json:"totalCost"`
	AvgTokens     int     `json:"avgTokensPerTask"`
}

// ModelPricing holds per-million-token pricing.
// Input and Output are USD per 1M tokens — the industry standard unit.
// Sourced from https://models.dev (community-maintained, open-source).
type ModelPricing struct {
	Input  float64 `json:"inputPerMillion"`  // USD per 1M input tokens
	Output float64 `json:"outputPerMillion"` // USD per 1M output tokens
}

// Pricing is the live pricing table. Populated from models.dev at startup
// and overridable by the admin via POST /v1/costs/pricing.
var Pricing = map[string]ModelPricing{}

// SyncPricingFromModelsDev fetches the latest pricing from https://models.dev/api.json
// and populates the pricing table. Called at gateway startup.
func SyncPricingFromModelsDev() (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://models.dev/api.json")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("models.dev returned %d", resp.StatusCode)
	}

	var providers map[string]struct {
		Models map[string]struct {
			ID   string `json:"id"`
			Cost struct {
				Input  float64 `json:"input"`
				Output float64 `json:"output"`
			} `json:"cost"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return 0, fmt.Errorf("decode: %w", err)
	}

	count := 0
	for _, provider := range providers {
		for _, model := range provider.Models {
			if model.Cost.Input > 0 || model.Cost.Output > 0 {
				Pricing[model.ID] = ModelPricing{
					Input:  model.Cost.Input,
					Output: model.Cost.Output,
				}
				count++
			}
		}
	}
	return count, nil
}

// EstimateCost calculates USD cost from token counts and per-million-token pricing.
//
//	cost = (inputTokens / 1,000,000) × inputPricePerMillion
//	     + (outputTokens / 1,000,000) × outputPricePerMillion
func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := Pricing[model]
	if !ok {
		return 0.0
	}
	return (float64(inputTokens)/1_000_000)*pricing.Input +
		(float64(outputTokens)/1_000_000)*pricing.Output
}

// SetPricing allows the admin to override pricing for a model at runtime.
func SetPricing(model string, inputPerMillion, outputPerMillion float64) {
	Pricing[model] = ModelPricing{Input: inputPerMillion, Output: outputPerMillion}
}

func (d *DB) RecordCost(ctx context.Context, e CostEntry) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO cost_tracking (task_id, profile, model, input_tokens, output_tokens, total_tokens, estimated_cost, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.TaskID, e.Profile, e.Model, e.InputTokens, e.OutputTokens, e.TotalTokens, e.EstimatedCost, time.Now())
	return err
}

func (d *DB) GetCostSummary(ctx context.Context, since time.Time) ([]CostSummary, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT profile, COUNT(*) as tasks,
		        COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0),
		        COALESCE(SUM(total_tokens),0), COALESCE(SUM(estimated_cost),0)
		 FROM cost_tracking WHERE created_at >= $1
		 GROUP BY profile ORDER BY SUM(estimated_cost) DESC`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var summaries []CostSummary
	for rows.Next() {
		var s CostSummary
		if err := rows.Scan(&s.Profile, &s.TotalTasks, &s.InputTokens, &s.OutputTokens,
			&s.TotalTokens, &s.TotalCost); err != nil {
			return nil, err
		}
		if s.TotalTasks > 0 {
			s.AvgTokens = s.TotalTokens / s.TotalTasks
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}
