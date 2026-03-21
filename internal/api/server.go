package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bitop-dev/agent-gateway/internal/db"
	"github.com/bitop-dev/agent-gateway/internal/events"
	"github.com/bitop-dev/agent-gateway/internal/router"
	"github.com/google/uuid"
)

type Server struct {
	DB          *db.DB
	Router      *router.Router
	Events      *events.Bus
	RegistryURL string
	AdminKey    string
}

func NewServer(database *db.DB, rtr *router.Router, bus *events.Bus, registryURL, adminKey string) *Server {
	return &Server{
		DB:          database,
		Router:      rtr,
		Events:      bus,
		RegistryURL: registryURL,
		AdminKey:    adminKey,
	}
}

func (s *Server) Handler(webFS http.FileSystem) http.Handler {
	mux := http.NewServeMux()

	// Dashboard UI
	if webFS != nil {
		mux.Handle("/", http.FileServer(webFS))
	}

	// Health (no auth)
	mux.HandleFunc("/v1/health", s.handleHealth)

	// Workers (no auth — workers self-register)
	mux.HandleFunc("/v1/workers", s.handleWorkers)

	// Authenticated endpoints
	mux.HandleFunc("/v1/tasks", s.requireAuth(s.handleTasks, "tasks:write", "tasks:read"))
	mux.HandleFunc("/v1/tasks/parallel", s.requireAuth(s.handleParallelTasks, "tasks:write"))
	mux.HandleFunc("/v1/tasks/", s.requireAuth(s.handleTaskByID, "tasks:read"))
	mux.HandleFunc("/v1/agents", s.requireAuth(s.handleAgents, "tasks:read"))

	// Admin endpoints
	mux.HandleFunc("/v1/auth/keys", s.requireAuth(s.handleAPIKeys, "admin"))
	mux.HandleFunc("/v1/schedules", s.requireAuth(s.handleSchedules, "admin"))
	mux.HandleFunc("/v1/webhooks", s.requireAuth(s.handleWebhookCRUD, "admin"))

	// Plugins (proxy to registry)
	mux.HandleFunc("/v1/plugins", s.requireAuth(s.handlePlugins, "tasks:read"))

	// Memory
	mux.HandleFunc("/v1/memory", s.requireAuth(s.handleMemory, "tasks:write", "tasks:read"))

	// Costs
	mux.HandleFunc("/v1/costs", s.requireAuth(s.handleCosts, "tasks:read"))
	mux.HandleFunc("/v1/costs/pricing", s.requireAuth(s.handlePricing, "admin"))

	// Event stream (SSE)
	mux.HandleFunc("/v1/events", s.requireAuth(s.handleEventStream, "tasks:read"))

	// Webhook triggers (auth via per-webhook token, not API key)
	mux.HandleFunc("/v1/webhooks/", s.handleWebhook)

	return logMiddleware(mux)
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func (s *Server) requireAuth(handler http.HandlerFunc, requiredScopes ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no admin key is configured, auth is disabled.
		if s.AdminKey == "" {
			handler(w, r)
			return
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		// Also check query param (needed for SSE EventSource which can't set headers).
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "authorization required")
			return
		}

		// Check admin key first.
		if token == s.AdminKey {
			handler(w, r)
			return
		}

		// Check API keys in database.
		keyHash := db.HashKey(token)
		apiKey, err := s.DB.ValidateAPIKey(r.Context(), keyHash)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		// Check scope — need at least one matching scope for the request method.
		scope := requiredScopes[0]
		if r.Method == http.MethodGet && len(requiredScopes) > 1 {
			scope = requiredScopes[1] // read scope for GET
		}
		if !s.DB.HasScope(apiKey, scope) {
			writeError(w, http.StatusForbidden, "insufficient scope: requires "+scope)
			return
		}

		handler(w, r)
	}
}

func (s *Server) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys, err := s.DB.ListAPIKeys(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"keys": keys, "count": len(keys)})

	case http.MethodPost:
		var req struct {
			Name   string   `json:"name"`
			Scopes []string `json:"scopes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		if len(req.Scopes) == 0 {
			req.Scopes = []string{"tasks:write", "tasks:read"}
		}

		keyID := "key-" + uuid.New().String()[:8]
		rawKey := "ag-" + uuid.New().String()
		keyHash := db.HashKey(rawKey)

		if err := s.DB.CreateAPIKey(r.Context(), keyID, req.Name, keyHash, req.Scopes); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Return the raw key ONCE — it can never be retrieved again.
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":     keyID,
			"name":   req.Name,
			"key":    rawKey,
			"scopes": req.Scopes,
			"note":   "Save this key — it cannot be retrieved again",
		})

	case http.MethodDelete:
		keyID := r.URL.Query().Get("id")
		if keyID == "" {
			writeError(w, http.StatusBadRequest, "id parameter required")
			return
		}
		s.DB.RevokeAPIKey(r.Context(), keyID)
		writeJSON(w, http.StatusOK, map[string]any{"revoked": true, "id": keyID})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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

	s.Events.Publish(events.Event{
		Topic:   events.TopicTaskSubmitted,
		TaskID:  task.ID,
		Profile: task.Profile,
		Message: task.Task,
	})

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

// handleParallelTasks accepts multiple tasks and dispatches them to different workers concurrently.
func (s *Server) handleParallelTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Tasks []struct {
			Profile string         `json:"profile"`
			Task    string         `json:"task"`
			Context map[string]any `json:"context,omitempty"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.Tasks) == 0 {
		writeError(w, http.StatusBadRequest, "tasks array is required")
		return
	}

	// Create all tasks.
	var tasks []*db.Task
	for _, t := range req.Tasks {
		task := &db.Task{
			ID:        "task-" + uuid.New().String()[:8],
			Profile:   t.Profile,
			Task:      t.Task,
			Context:   t.Context,
			Status:    "queued",
			Priority:  "normal",
			CreatedAt: time.Now(),
		}
		s.DB.CreateTask(r.Context(), *task)
		tasks = append(tasks, task)
	}

	// Dispatch all concurrently to different workers.
	type result struct {
		idx  int
		task *db.Task
		err  error
	}
	results := make(chan result, len(tasks))
	for i, task := range tasks {
		go func(idx int, t *db.Task) {
			dispatched, err := s.Router.Dispatch(r.Context(), t)
			results <- result{idx, dispatched, err}
		}(i, task)
	}

	// Collect results.
	output := make([]*db.Task, len(tasks))
	for range tasks {
		r := <-results
		if r.err != nil {
			tasks[r.idx].Status = "failed"
			tasks[r.idx].Error = r.err.Error()
			output[r.idx] = tasks[r.idx]
		} else {
			output[r.idx] = r.task
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"tasks": output, "count": len(output)})
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
		s.Events.Publish(events.Event{
			Topic:     events.TopicWorkerJoined,
			WorkerURL: req.URL,
			Message:   fmt.Sprintf("worker registered with %d profiles", len(req.Profiles)),
		})
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
		Accepts      string   `json:"accepts,omitempty"`
		Returns      string   `json:"returns,omitempty"`
		Extends      string   `json:"extends,omitempty"`
		Mode         string   `json:"mode,omitempty"`
		Model        string   `json:"model,omitempty"`
		Provider     string   `json:"provider,omitempty"`
		Tools        []string `json:"tools,omitempty"`
		Source       string   `json:"source"`
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
					Name         string   `json:"name"`
					Description  string   `json:"description"`
					Capabilities []string `json:"capabilities"`
					Accepts      string   `json:"accepts"`
					Returns      string   `json:"returns"`
					Extends      string   `json:"extends"`
					Mode         string   `json:"mode"`
					Model        string   `json:"model"`
					Provider     string   `json:"provider"`
					Tools        []string `json:"tools"`
				} `json:"profiles"`
			}
			json.NewDecoder(resp.Body).Decode(&index)
			resp.Body.Close()
			for _, p := range index.Profiles {
				if seen[p.Name] {
					continue
				}
				seen[p.Name] = true
				agents = append(agents, agentInfo{
					Name: p.Name, Description: p.Description,
					Capabilities: p.Capabilities, Accepts: p.Accepts,
					Returns: p.Returns, Extends: p.Extends,
					Mode: p.Mode, Model: p.Model,
					Provider: p.Provider, Tools: p.Tools,
					Source: "registry",
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"agents": agents, "count": len(agents)})
}

// ── Plugins ───────────────────────────────────────────────────────────────────

func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	type pluginInfo struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Description  string   `json:"description,omitempty"`
		Category     string   `json:"category,omitempty"`
		Runtime      string   `json:"runtime,omitempty"`
		Tools        []string `json:"tools,omitempty"`
		Dependencies []string `json:"dependencies,omitempty"`
		Source       string   `json:"source"`
	}

	var plugins []pluginInfo

	if s.RegistryURL != "" {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(strings.TrimRight(s.RegistryURL, "/") + "/v1/index.json")
		if err == nil && resp.StatusCode == http.StatusOK {
			var index struct {
				Packages []struct {
					Name          string   `json:"name"`
					LatestVersion string   `json:"latestVersion"`
					Description   string   `json:"description"`
					Category      string   `json:"category"`
					Runtime       string   `json:"runtime"`
					Tools         []string `json:"tools"`
					Dependencies  []string `json:"dependencies"`
				} `json:"packages"`
			}
			json.NewDecoder(resp.Body).Decode(&index)
			resp.Body.Close()
			for _, p := range index.Packages {
				plugins = append(plugins, pluginInfo{
					Name:         p.Name,
					Version:      p.LatestVersion,
					Description:  p.Description,
					Category:     p.Category,
					Runtime:      p.Runtime,
					Tools:        p.Tools,
					Dependencies: p.Dependencies,
					Source:       "registry",
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"plugins": plugins, "count": len(plugins)})
}

// ── Costs ─────────────────────────────────────────────────────────────────────

func (s *Server) handleCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sinceStr := r.URL.Query().Get("since")
	since := time.Now().AddDate(0, 0, -30) // default 30 days
	if sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t
		}
	}
	summaries, err := s.DB.GetCostSummary(r.Context(), since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var totalCost float64
	var totalTokens int
	for _, s := range summaries {
		totalCost += s.TotalCost
		totalTokens += s.TotalTokens
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"since":       since.Format("2006-01-02"),
		"profiles":    summaries,
		"totalCost":   totalCost,
		"totalTokens": totalTokens,
	})
}

// ── Pricing ───────────────────────────────────────────────────────────────────

func (s *Server) handlePricing(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"pricing": db.Pricing,
			"unit":    "USD per million tokens",
		})

	case http.MethodPost:
		var req struct {
			Model           string  `json:"model"`
			InputPerMillion  float64 `json:"inputPerMillion"`
			OutputPerMillion float64 `json:"outputPerMillion"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Model == "" {
			writeError(w, http.StatusBadRequest, "model is required")
			return
		}
		db.SetPricing(req.Model, req.InputPerMillion, req.OutputPerMillion)
		writeJSON(w, http.StatusOK, map[string]any{
			"model":            req.Model,
			"inputPerMillion":  req.InputPerMillion,
			"outputPerMillion": req.OutputPerMillion,
			"unit":             "USD per million tokens",
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── Memory ────────────────────────────────────────────────────────────────────

func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	profile := r.URL.Query().Get("profile")
	if profile == "" {
		writeError(w, http.StatusBadRequest, "profile parameter required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		key := r.URL.Query().Get("key")
		if key != "" {
			value, err := s.DB.Recall(r.Context(), profile, key)
			if err != nil {
				writeError(w, http.StatusNotFound, "key not found")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"profile": profile, "key": key, "value": value})
		} else {
			entries, err := s.DB.RecallAll(r.Context(), profile)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"profile": profile, "entries": entries, "count": len(entries)})
		}

	case http.MethodPost:
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Key == "" || req.Value == "" {
			writeError(w, http.StatusBadRequest, "key and value required")
			return
		}
		if err := s.DB.Remember(r.Context(), profile, req.Key, req.Value); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stored": true, "profile": profile, "key": req.Key})

	case http.MethodDelete:
		key := r.URL.Query().Get("key")
		if key != "" {
			s.DB.Forget(r.Context(), profile, key)
		} else {
			s.DB.ForgetAll(r.Context(), profile)
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "profile": profile})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── Event stream (SSE) ────────────────────────────────────────────────────────

func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Subscribe to all agent events.
	eventCh := make(chan events.Event, 64)
	s.Events.Subscribe("agent.>", func(e events.Event) {
		select {
		case eventCh <- e:
		default: // drop if buffer full
		}
	})

	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-eventCh:
			data, _ := json.Marshal(e)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Topic, data)
			flusher.Flush()
		}
	}
}

// ── Schedules ─────────────────────────────────────────────────────────────────

func (s *Server) handleSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		schedules, err := s.DB.ListSchedules(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"schedules": schedules, "count": len(schedules)})

	case http.MethodPost:
		var req struct {
			Name     string         `json:"name"`
			Cron     string         `json:"cron"`
			Timezone string         `json:"timezone"`
			Profile  string         `json:"profile"`
			Task     string         `json:"task"`
			Context  map[string]any `json:"context"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" || req.Cron == "" || req.Profile == "" || req.Task == "" {
			writeError(w, http.StatusBadRequest, "name, cron, profile, and task are required")
			return
		}
		if req.Timezone == "" {
			req.Timezone = "UTC"
		}
		// Calculate first run time.
		now := time.Now()
		nextRun := now.Add(1 * time.Minute) // placeholder — scheduler will recalculate

		sched := db.Schedule{
			ID:       "sched-" + uuid.New().String()[:8],
			Name:     req.Name,
			CronExpr: req.Cron,
			Timezone: req.Timezone,
			Profile:  req.Profile,
			Task:     req.Task,
			Context:  req.Context,
			Enabled:  true,
			NextRun:  &nextRun,
		}
		if err := s.DB.CreateSchedule(r.Context(), sched); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, sched)

	case http.MethodPut:
		var req struct {
			ID       string         `json:"id"`
			Name     string         `json:"name"`
			Cron     string         `json:"cron"`
			Timezone string         `json:"timezone"`
			Profile  string         `json:"profile"`
			Task     string         `json:"task"`
			Context  map[string]any `json:"context"`
			Enabled  *bool          `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.ID == "" {
			writeError(w, http.StatusBadRequest, "id is required")
			return
		}
		if req.Name == "" || req.Cron == "" || req.Profile == "" || req.Task == "" {
			writeError(w, http.StatusBadRequest, "name, cron, profile, and task are required")
			return
		}
		if req.Timezone == "" {
			req.Timezone = "UTC"
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		now := time.Now()
		nextRun := now.Add(1 * time.Minute)
		sched := db.Schedule{
			ID:       req.ID,
			Name:     req.Name,
			CronExpr: req.Cron,
			Timezone: req.Timezone,
			Profile:  req.Profile,
			Task:     req.Task,
			Context:  req.Context,
			Enabled:  enabled,
			NextRun:  &nextRun,
		}
		if err := s.DB.UpdateSchedule(r.Context(), sched); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, sched)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id parameter required")
			return
		}
		s.DB.DeleteSchedule(r.Context(), id)
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── Webhook management ────────────────────────────────────────────────────────

func (s *Server) handleWebhookCRUD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		webhooks, err := s.DB.ListWebhooks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"webhooks": webhooks, "count": len(webhooks)})

	case http.MethodPost:
		var req struct {
			Name            string         `json:"name"`
			Path            string         `json:"path"`
			Profile         string         `json:"profile"`
			TaskTemplate    string         `json:"taskTemplate"`
			ContextTemplate map[string]any `json:"contextTemplate"`
			AuthToken       string         `json:"authToken"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" || req.Path == "" || req.Profile == "" || req.TaskTemplate == "" {
			writeError(w, http.StatusBadRequest, "name, path, profile, and taskTemplate are required")
			return
		}
		wh := db.Webhook{
			ID:              "wh-" + uuid.New().String()[:8],
			Name:            req.Name,
			Path:            req.Path,
			Profile:         req.Profile,
			TaskTemplate:    req.TaskTemplate,
			ContextTemplate: req.ContextTemplate,
			AuthToken:       req.AuthToken,
			Enabled:         true,
		}
		if err := s.DB.CreateWebhook(r.Context(), wh); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, wh)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id parameter required")
			return
		}
		s.DB.DeleteWebhook(r.Context(), id)
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── Webhook triggers ──────────────────────────────────────────────────────────

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract the webhook path: /v1/webhooks/grafana-alerts → grafana-alerts
	path := strings.TrimPrefix(r.URL.Path, "/v1/webhooks/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "webhook path required")
		return
	}

	webhook, err := s.DB.GetWebhookByPath(r.Context(), path)
	if err != nil {
		writeError(w, http.StatusNotFound, "webhook not found: "+path)
		return
	}

	// Check webhook-specific auth token if configured.
	if webhook.AuthToken != "" {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token != webhook.AuthToken {
			writeError(w, http.StatusUnauthorized, "invalid webhook token")
			return
		}
	}

	// Parse the incoming payload.
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	// Expand the task template with the payload.
	taskText := expandTemplate(webhook.TaskTemplate, payload)

	// Build context from template.
	taskContext := make(map[string]any)
	for k, v := range webhook.ContextTemplate {
		if s, ok := v.(string); ok {
			taskContext[k] = expandTemplate(s, payload)
		} else {
			taskContext[k] = v
		}
	}
	// Also include the raw payload in context.
	taskContext["_webhook"] = path
	taskContext["_payload"] = payload

	// Create and dispatch the task.
	task := db.Task{
		ID:        "task-" + uuid.New().String()[:8],
		Profile:   webhook.Profile,
		Task:      taskText,
		Context:   taskContext,
		Status:    "queued",
		Priority:  "normal",
		CreatedAt: time.Now(),
	}
	s.DB.CreateTask(r.Context(), task)

	// Dispatch async — webhooks shouldn't block.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.Router.Dispatch(ctx, &task)
	}()

	s.Events.Publish(events.Event{
		Topic:   events.TopicWebhookFired,
		TaskID:  task.ID,
		Profile: task.Profile,
		Message: fmt.Sprintf("webhook %s triggered", path),
		Data:    map[string]any{"webhook": path, "payload": payload},
	})
	log.Printf("webhook %s → task %s (profile=%s)", path, task.ID, task.Profile)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"taskId":  task.ID,
		"profile": task.Profile,
		"status":  "queued",
	})
}

// expandTemplate replaces {{key}} and {{nested.key}} with values from the payload.
func expandTemplate(tmpl string, payload map[string]any) string {
	result := tmpl
	for k, v := range payload {
		placeholder := "{{" + k + "}}"
		switch val := v.(type) {
		case string:
			result = strings.ReplaceAll(result, placeholder, val)
		case map[string]any:
			// Handle nested: {{labels.team}} etc.
			for nk, nv := range val {
				nested := "{{" + k + "." + nk + "}}"
				if s, ok := nv.(string); ok {
					result = strings.ReplaceAll(result, nested, s)
				}
			}
		default:
			result = strings.ReplaceAll(result, placeholder, fmt.Sprint(v))
		}
	}
	return result
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
