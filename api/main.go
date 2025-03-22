package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var ctx = context.Background()

// Initialize Redis client
var rdb = redis.NewClient(&redis.Options{
	Addr: os.Getenv("REDIS_ADDR"),
})

type CodeRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type Job struct {
	ID        string    `json:"id"`
	Language  string    `json:"language"`
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

type JobResult struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	Output    string    `json:"output"`
	Error     string    `json:"error"`
	ExecTime  int64     `json:"exec_time_ms"`
	Timestamp time.Time `json:"timestamp"`
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Language == "" || req.Code == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate language
	if req.Language != "python" && req.Language != "javascript" && req.Language != "cpp" {
		http.Error(w, "Unsupported language. Supported languages: python, javascript, cpp", http.StatusBadRequest)
		return
	}

	// Create job with unique ID
	job := Job{
		ID:        uuid.NewString(),
		Language:  req.Language,
		Code:      req.Code,
		Timestamp: time.Now(),
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		http.Error(w, "Error processing job", http.StatusInternalServerError)
		return
	}

	// Push job to Redis queue
	if err := rdb.LPush(ctx, "code_jobs", jobData).Err(); err != nil {
		http.Error(w, "Failed to enqueue job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"job_id": "%s"}`, job.ID)
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Get result from Redis
	resultData, err := rdb.Get(ctx, "result:"+jobID).Result()
	if err != nil {
		if err == redis.Nil {
			// Check if job exists but hasn't been processed yet
			_, err := rdb.LRange(ctx, "code_jobs", 0, -1).Result()
			if err != nil {
				http.Error(w, "Error checking job status", http.StatusInternalServerError)
				return
			}

			// Job is still in queue or being processed
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"job_id": "%s", "status": "pending"}`, jobID)
			return
		}
		http.Error(w, "Error retrieving job result", http.StatusInternalServerError)
		return
	}

	// Return the result
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(resultData))
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "ok"}`)
}

func main() {
	// Set default Redis address if not provided
	if os.Getenv("REDIS_ADDR") == "" {
		os.Setenv("REDIS_ADDR", "localhost:6379")
	}

	router := mux.NewRouter()
	router.HandleFunc("/execute", executeHandler).Methods("POST")
	router.HandleFunc("/result/{id}", resultHandler).Methods("GET")
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")

	log.Println("API server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
