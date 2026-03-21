package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bitop-dev/agent-gateway/internal/api"
	"github.com/bitop-dev/agent-gateway/internal/db"
	"github.com/bitop-dev/agent-gateway/internal/events"
	"github.com/bitop-dev/agent-gateway/internal/router"
	"github.com/bitop-dev/agent-gateway/internal/scheduler"
)

//go:embed migrations.sql
var migrationSQL string

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dsn := flag.String("dsn", "", "PostgreSQL connection string")
	natsURL := flag.String("nats", "", "NATS URL (optional, e.g. nats://localhost:4222)")
	registryURL := flag.String("registry", "", "agent-registry URL")
	adminKey := flag.String("admin-key", "", "admin API key")
	flag.Parse()

	if *dsn == "" {
		*dsn = os.Getenv("DATABASE_URL")
	}
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "error: --dsn or DATABASE_URL is required")
		os.Exit(1)
	}
	if *registryURL == "" {
		*registryURL = os.Getenv("REGISTRY_URL")
	}
	if *adminKey == "" {
		*adminKey = os.Getenv("ADMIN_KEY")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database.
	log.Printf("connecting to database...")
	database, err := db.Connect(ctx, *dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	// Run migrations.
	log.Printf("running migrations...")
	if err := database.Migrate(ctx, migrationSQL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Mark stale workers.
	stale, _ := database.MarkStaleWorkers(ctx, 15*time.Minute)
	if stale > 0 {
		log.Printf("marked %d stale workers", stale)
	}

	// Connect to NATS (optional).
	natsAddr := *natsURL
	if natsAddr == "" {
		natsAddr = os.Getenv("NATS_URL")
	}
	bus, err := events.Connect(natsAddr)
	if err != nil {
		log.Printf("NATS connection failed (events will be logged only): %v", err)
		bus, _ = events.Connect("")
	}
	defer bus.Close()

	// Build server.
	rtr := router.NewRouter(database, bus)
	srv := api.NewServer(database, rtr, bus, *registryURL, *adminKey)

	// Start scheduler.
	sched := &scheduler.Scheduler{DB: database, Router: rtr}
	go sched.Run(ctx)

	// Start stale worker cleanup goroutine.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, _ := database.MarkStaleWorkers(ctx, 15*time.Minute)
				if n > 0 {
					log.Printf("marked %d stale workers", n)
				}
			}
		}
	}()

	httpServer := &http.Server{Addr: *addr, Handler: srv.Handler()}

	// Graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Printf("shutting down...")
		cancel()
		httpServer.Close()
	}()

	log.Printf("agent-gateway started on %s", *addr)
	log.Printf("  POST /v1/tasks       — submit a task")
	log.Printf("  GET  /v1/tasks       — list tasks")
	log.Printf("  GET  /v1/tasks/{id}  — get task details")
	log.Printf("  POST /v1/workers     — register worker")
	log.Printf("  GET  /v1/workers     — list workers")
	log.Printf("  GET  /v1/agents      — discover agents")
	log.Printf("  GET  /v1/health      — health check")
	if *registryURL != "" {
		log.Printf("  registry: %s", *registryURL)
	}

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
