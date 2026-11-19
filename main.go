package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type API struct {
	db *sql.DB
}

func newAPI(db *sql.DB) *API {
	return &API{db: db}
}

// Initialize the database table.
func initializeDatabase(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id BIGSERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		completed BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`

	_, err := db.Exec(query)
	return err
}

// Register API routes.
func (a *API) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", a.health)
	mux.HandleFunc("/readyz", a.ready)
	mux.HandleFunc("/api/tasks", a.tasksHandler)
	mux.HandleFunc("/api/tasks/", a.taskHandler)
	mux.HandleFunc("/", a.home)

	return logging(mux)
}

// Home endpoint.
func (a *API) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Welcome to the OpenShift Go PostgreSQL API",
		"health":  "/healthz",
		"ready":   "/readyz",
		"tasks":   "/api/tasks",
	})
}

// Application health check.
func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Database readiness check.
func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := a.db.PingContext(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ready",
		"database": "connected",
	})
}

// Create and retrieve tasks.
func (a *API) tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		rows, err := a.db.QueryContext(r.Context(), `
			SELECT id, title, completed, created_at
			FROM tasks
			ORDER BY id
		`)
		if err != nil {
			http.Error(w, "failed to retrieve tasks", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		tasks := []Task{}

		for rows.Next() {
			var task Task

			if err := rows.Scan(
				&task.ID,
				&task.Title,
				&task.Completed,
				&task.CreatedAt,
			); err != nil {
				http.Error(w, "failed to read tasks", http.StatusInternalServerError)
				return
			}

			tasks = append(tasks, task)
		}

		if err := rows.Err(); err != nil {
			http.Error(w, "failed to read tasks", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, tasks)

	case http.MethodPost:
		var input struct {
			Title string `json:"title"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		input.Title = strings.TrimSpace(input.Title)

		if input.Title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		var task Task

		err := a.db.QueryRowContext(r.Context(), `
			INSERT INTO tasks (title)
			VALUES ($1)
			RETURNING id, title, completed, created_at
		`, input.Title).Scan(
			&task.ID,
			&task.Title,
			&task.Completed,
			&task.CreatedAt,
		)

		if err != nil {
			http.Error(w, "failed to create task", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, task)

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Update an existing task.
func (a *API) taskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idText := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	id, err := strconv.Atoi(idText)

	if err != nil || id < 1 || strings.Contains(idText, "/") {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Completed *bool `json:"completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Completed == nil {
		http.Error(w, `provide {"completed": true} or {"completed": false}`, http.StatusBadRequest)
		return
	}

	var task Task

	err = a.db.QueryRowContext(r.Context(), `
		UPDATE tasks
		SET completed = $1
		WHERE id = $2
		RETURNING id, title, completed, created_at
	`, *input.Completed, id).Scan(
		&task.ID,
		&task.Title,
		&task.Completed,
		&task.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "failed to update task", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// JSON response helper.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// Request logging middleware.
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		log.Printf("%s %s %s",
			r.Method,
			r.URL.Path,
			time.Since(start),
		)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal("failed to configure database:", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify database connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("failed to connect to PostgreSQL:", err)
	}

	log.Println("Connected to PostgreSQL successfully")

	// Create the table if it does not exist.
	if err := initializeDatabase(db); err != nil {
		log.Fatal("failed to initialize database:", err)
	}

	log.Println("Database initialized successfully")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newAPI(db).routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Go API listening on port %s", port)

	log.Fatal(server.ListenAndServe())
}
