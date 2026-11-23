package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type API struct {
	db *sql.DB
}

func main() {
	port := env("PORT", "8080")
	db, err := openDatabase()
	if err != nil {
		log.Fatalf("database startup failed: %v", err)
	}
	defer db.Close()

	api := &API{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("/", api.home)
	mux.HandleFunc("/healthz", api.healthz)
	mux.HandleFunc("/readyz", api.readyz)
	mux.HandleFunc("/api/tasks", api.tasks)
	mux.HandleFunc("/api/tasks/", api.taskByID)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           logging(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Println("shutting down HTTP server")
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("Go Task API listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func openDatabase() (*sql.DB, error) {
	host := env("DB_HOST", "postgres")
	port := env("DB_PORT", "5432")
	name := env("DB_NAME", "tasks")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	if user == "" || password == "" {
		return nil, errors.New("DB_USER and DB_PASSWORD must be configured")
	}

	// URL-escape credentials so special characters in passwords are safe.
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		url.QueryEscape(user), url.QueryEscape(password), net.JoinHostPort(host, port), url.PathEscape(name))

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		lastErr = db.PingContext(ctx)
		cancel()
		if lastErr == nil {
			break
		}
		log.Printf("waiting for PostgreSQL (attempt %d/30): %v", attempt, lastErr)
		time.Sleep(2 * time.Second)
	}
	if lastErr != nil {
		db.Close()
		return nil, fmt.Errorf("could not connect to PostgreSQL: %w", lastErr)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id BIGSERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create tasks table: %w", err)
	}
	return db, nil
}

func (a *API) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "golang-openshift-task-api",
		"status":  "running",
	})
}

func (a *API) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (a *API) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) tasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.QueryContext(r.Context(),
			`SELECT id, title, completed, created_at FROM tasks ORDER BY id`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not fetch tasks")
			return
		}
		defer rows.Close()
		result := make([]Task, 0)
		for rows.Next() {
			var task Task
			if err := rows.Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt); err != nil {
				writeError(w, http.StatusInternalServerError, "could not read tasks")
				return
			}
			result = append(result, task)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read tasks")
			return
		}
		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var input struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		input.Title = strings.TrimSpace(input.Title)
		if input.Title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		var task Task
		err := a.db.QueryRowContext(r.Context(),
			`INSERT INTO tasks (title) VALUES ($1)
			 RETURNING id, title, completed, created_at`, input.Title).
			Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not create task")
			return
		}
		writeJSON(w, http.StatusCreated, task)

	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) taskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	idText := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid task ID")
		return
	}
	var input struct {
		Completed *bool `json:"completed"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil || input.Completed == nil {
		writeError(w, http.StatusBadRequest, "completed (boolean) is required")
		return
	}
	var task Task
	err = a.db.QueryRowContext(r.Context(),
		`UPDATE tasks SET completed = $1 WHERE id = $2
		 RETURNING id, title, completed, created_at`, *input.Completed, id).
		Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
