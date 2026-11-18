package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type API struct {
	mu     sync.RWMutex
	tasks  []Task
	nextID int
}

func newAPI() *API {
	return &API{tasks: []Task{}, nextID: 1}
}

func (a *API) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", a.health)
	mux.HandleFunc("/api/tasks", a.tasksHandler)
	mux.HandleFunc("/api/tasks/", a.taskHandler)
	mux.HandleFunc("/", a.home)
	return logging(mux)
}

func (a *API) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Welcome to the OpenShift Go Demo API",
		"health":  "/healthz",
		"tasks":   "/api/tasks",
	})
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		result := append([]Task(nil), a.tasks...)
		a.mu.RUnlock()
		writeJSON(w, http.StatusOK, result)

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

		a.mu.Lock()
		task := Task{
			ID:        a.nextID,
			Title:     input.Title,
			Completed: false,
			CreatedAt: time.Now().UTC(),
		}
		a.nextID++
		a.tasks = append(a.tasks, task)
		a.mu.Unlock()

		writeJSON(w, http.StatusCreated, task)

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) taskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idText := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	id, err := strconv.Atoi(idText)
	if err != nil || id < 1 {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Completed *bool `json:"completed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Completed == nil {
		http.Error(w, `provide {"completed": true|false}`, http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.tasks {
		if a.tasks[i].ID == id {
			a.tasks[i].Completed = *input.Completed
			writeJSON(w, http.StatusOK, a.tasks[i])
			return
		}
	}
	http.Error(w, "task not found", http.StatusNotFound)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newAPI().routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Go demo API listening on port %s", port)
	log.Fatal(server.ListenAndServe())
}
