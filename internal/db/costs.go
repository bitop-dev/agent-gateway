package db

import (
	"context"
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

// ModelPricing holds per-million-token pricing set by the admin.
// Input and Output are USD per 1M tokens — the industry standard unit.
// Example: gpt-4o charges $2.50/1M input tokens and $10.00/1M output tokens.
type ModelPricing struct {
	Input  float64 `json:"inputPerMillion"`  // USD per 1M input tokens
	Output float64 `json:"outputPerMillion"` // USD per 1M output tokens
}

// DefaultPricing is the built-in pricing table. Admins can override via
// the gateway API or config. All prices are USD per million tokens.
var DefaultPricing = map[string]ModelPricing{
	// OpenAI
	"gpt-4o":              {Input: 2.50, Output: 10.00},
	"gpt-4o-mini":         {Input: 0.15, Output: 0.60},
	"gpt-4-turbo":         {Input: 10.00, Output: 30.00},
	"gpt-3.5-turbo":       {Input: 0.50, Output: 1.50},
	// Self-hosted (free)
	"gpt-oss-120b":                {Input: 0, Output: 0},
	"gpt-oss-20b":                 {Input: 0, Output: 0},
	"nemotron-3-super-120b-a12b":  {Input: 0, Output: 0},
	"nemotron-3-nano-30b-a3b":     {Input: 0, Output: 0},
	"llama-3.1-70b-instruct":      {Input: 0, Output: 0},
	// Anthropic
	"claude-3.7-sonnet":   {Input: 3.00, Output: 15.00},
	"claude-3.5-sonnet":   {Input: 3.00, Output: 15.00},
	"claude-4.0-sonnet":   {Input: 3.00, Output: 15.00},
	"claude-4.5-sonnet":   {Input: 3.00, Output: 15.00},
	// Google
	"gemini-2.5-pro":      {Input: 1.25, Output: 10.00},
	"gemini-2.5-flash":    {Input: 0.15, Output: 0.60},
}

// EstimateCost calculates USD cost from token counts and per-million-token pricing.
//
//	cost = (inputTokens / 1,000,000) × inputPricePerMillion
//	     + (outputTokens / 1,000,000) × outputPricePerMillion
func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := DefaultPricing[model]
	if !ok {
		return 0.0
	}
	return (float64(inputTokens)/1_000_000)*pricing.Input +
		(float64(outputTokens)/1_000_000)*pricing.Output
}

// SetPricing allows the admin to override pricing for a model at runtime.
func SetPricing(model string, inputPerMillion, outputPerMillion float64) {
	DefaultPricing[model] = ModelPricing{Input: inputPerMillion, Output: outputPerMillion}
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
