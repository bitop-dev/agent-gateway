package scheduler

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bitop-dev/agent-gateway/internal/db"
	"github.com/bitop-dev/agent-gateway/internal/router"
	"github.com/google/uuid"
)

// Scheduler checks for due schedules and dispatches tasks.
type Scheduler struct {
	DB     *db.DB
	Router *router.Router
}

// Run checks every 30 seconds for due schedules and creates tasks.
func (s *Scheduler) Run(ctx context.Context) {
	log.Printf("scheduler started (checking every 30s)")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	due, err := s.DB.GetDueSchedules(ctx)
	if err != nil {
		log.Printf("scheduler: get due schedules: %v", err)
		return
	}
	for _, sched := range due {
		task := db.Task{
			ID:        "task-" + uuid.New().String()[:8],
			Profile:   sched.Profile,
			Task:      sched.Task,
			Context:   sched.Context,
			Status:    "queued",
			Priority:  "normal",
			CreatedAt: time.Now(),
		}
		if err := s.DB.CreateTask(ctx, task); err != nil {
			log.Printf("scheduler: create task for %s: %v", sched.Name, err)
			continue
		}
		log.Printf("scheduler: triggered %s → task %s (profile=%s)", sched.Name, task.ID, task.Profile)

		// Dispatch async.
		go func(t db.Task) {
			dispatchCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			s.Router.Dispatch(dispatchCtx, &t)
		}(task)

		// Calculate next run.
		now := time.Now()
		nextRun := nextCronTime(sched.CronExpr, sched.Timezone, now)
		s.DB.UpdateScheduleRun(ctx, sched.ID, now, nextRun)
	}
}

// nextCronTime calculates the next run time from a simple cron expression.
// Supports: "minute hour day-of-month month day-of-week" (standard 5-field)
// with * for any. Not a full cron parser — handles simple cases.
func nextCronTime(expr, tz string, after time.Time) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	t := after.In(loc)

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		// Default: 1 hour from now.
		return after.Add(1 * time.Hour)
	}

	minute := parseField(fields[0], 0)
	hour := parseField(fields[1], t.Hour())

	// Find the next occurrence.
	next := time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, loc)
	if !next.After(after) {
		next = next.Add(24 * time.Hour)
	}
	return next.UTC()
}

func parseField(field string, fallback int) int {
	if field == "*" {
		return fallback
	}
	n, err := strconv.Atoi(field)
	if err != nil {
		return fallback
	}
	return n
}
