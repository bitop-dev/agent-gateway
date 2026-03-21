package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bitop-dev/agent-gateway/internal/db"
	"github.com/bitop-dev/agent-gateway/internal/router"
	"github.com/google/uuid"
)

type Server struct {
	DB          *db.DB
	Router      *router.Router
	RegistryURL string
	AdminKey    string
}

func NewServer(database *db.DB, rtr *router.Router, registryURL, adminKey string) *Server {
	return &Server{
		DB:          database,
		Router:      rtr,
		RegistryURL: registryURL,
		AdminKey:    adminKey,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/v1/health", s.handleHealth)

	// Tasks
	mux.HandleFunc("/v1/tasks", s.handleTasks)     // POST + GET
	mux.HandleFunc("/v1/tasks/", s.handleTaskByID)  // GET /v1/tasks/{id}

	// Workers
	mux.HandleFunc("/v1/workers", s.handleWorkers)  // POST + GET + DELETE

	// Agents (discovery proxy)
	mux.HandleFunc("/v1/agents", s.handleAgents)

	return logMiddleware(mux)
}

// ── Health ────────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	workers, _ := s.DB.ListWorkers(r.Context())
	tasks, _ := s.DB.ListTasks(r.Context(), "", 1)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"workers": len(workers),
		"tasks":   len(tasks),
	})
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.submitTask(w, r)
	case http.MethodGet:
		s.listTasks(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) submitTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Profile     string         `json:"profile"`
		Task        string         `json:"task"`
		Context     map[string]any `json:"context,omitempty"`
		Priority    string         `json:"priority,omitempty"`
		CallbackURL string         `json:"callback,omitempty"`
		Async       bool           `json:"async,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Profile) == "" {
		writeError(w, http.StatusBadRequest, "profile is required")
		return
	}
	if strings.TrimSpace(req.Task) == "" {
		writeError(w, http.StatusBadRequest, "task is required")
		return
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}

	task := db.Task{
		ID:          "task-" + uuid.New().String()[:8],
		Profile:     req.Profile,
		Task:        req.Task,
		Context:     req.Context,
		Status:      "queued",
		Priority:    req.Priority,
		CallbackURL: req.CallbackURL,
		CreatedAt:   time.Now(),
	}

	if err := s.DB.CreateTask(r.Context(), task); err != nil {
		writeError(w, http.StatusInternalServerError, "create task: "+err.Error())
		return
	}

	if req.Async {
		// Return immediately — task will be dispatched in the background.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			s.Router.Dispatch(ctx, &task)
		}()
		writeJSON(w, http.StatusAccepted, task)
		return
	}

	// Synchronous — wait for result.
	result, err := s.Router.Dispatch(r.Context(), &task)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
		limit = n
	}

	tasks, err := s.DB.ListTasks(r.Context(), status, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks, "count": len(tasks)})
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}
	task, err := s.DB.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// ── Workers ───────────────────────────────────────────────────────────────────

func (s *Server) handleWorkers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			URL          string   `json:"url"`
			Profiles     []string `json:"profiles"`
			Capabilities []string `json:"capabilities"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.URL == "" {
			writeError(w, http.StatusBadRequest, "url is required")
			return
		}
		now := time.Now()
		err := s.DB.UpsertWorker(r.Context(), db.Worker{
			URL:           req.URL,
			Profiles:      req.Profiles,
			Capabilities:  req.Capabilities,
			Status:        "active",
			RegisteredAt:  now,
			LastHeartbeat: now,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"registered": true, "url": req.URL})

	case http.MethodGet:
		workers, err := s.DB.ListWorkers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"workers": workers, "count": len(workers)})

	case http.MethodDelete:
		url := r.URL.Query().Get("url")
		if url == "" {
			writeError(w, http.StatusBadRequest, "url parameter required")
			return
		}
		s.DB.RemoveWorker(r.Context(), url)
		writeJSON(w, http.StatusOK, map[string]any{"deregistered": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── Agents ────────────────────────────────────────────────────────────────────

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	// Aggregate from registered workers + registry profile index.
	type agentInfo struct {
		Name         string   `json:"name"`
		Description  string   `json:"description,omitempty"`
		Capabilities []string `json:"capabilities,omitempty"`
		Source       string   `json:"source"` // "worker" or "registry"
	}

	seen := make(map[string]bool)
	var agents []agentInfo

	// From workers.
	workers, _ := s.DB.ListWorkers(r.Context())
	for _, w := range workers {
		for _, p := range w.Profiles {
			if seen[p] {
				continue
			}
			seen[p] = true
			agents = append(agents, agentInfo{Name: p, Source: "worker"})
		}
	}

	// From registry profile index.
	if s.RegistryURL != "" {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(strings.TrimRight(s.RegistryURL, "/") + "/v1/profiles/index.json")
		if err == nil && resp.StatusCode == http.StatusOK {
			var index struct {
				Profiles []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				} `json:"profiles"`
			}
			json.NewDecoder(resp.Body).Decode(&index)
			resp.Body.Close()
			for _, p := range index.Profiles {
				if seen[p.Name] {
					continue
				}
				seen[p.Name] = true
				agents = append(agents, agentInfo{Name: p.Name, Description: p.Description, Source: "registry"})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"agents": agents, "count": len(agents)})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
