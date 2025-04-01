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
	"github.com/jackc/pgx/v4/pgxpool"
)

var ctx = context.Background()

// Initialize Redis client
var rdb = redis.NewClient(&redis.Options{
	Addr: os.Getenv("REDIS_ADDR"),
})

var db *pgxpool.Pool

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

type Reward struct {
	RewardID        int    `json:"reward_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	InventoryCount  int    `json:"inventory_count"`
	Cost            int    `json:"cost"`
}

type Claim struct {
	UserID   int `json:"user_id"`
	RewardID int `json:"reward_id"`
}

type ClaimResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func connectToDB() {
    var err error
    databaseURL := os.Getenv("DATABASE_URL")
    if databaseURL == "" {
        databaseURL = "imanol.terminator"
    }

    
    db, err = pgxpool.Connect(ctx, databaseURL)
    if err != nil {
        log.Fatalf("Unable to connect to database: %v\n", err)
    }
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

func claimHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var claim Claim
	if err := json.NewDecoder(r.Body).Decode(&claim); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Insert claim into the database
	_, err := db.Exec(ctx, "INSERT INTO Claims (user_id, reward_id) VALUES ($1, $2)", claim.UserID, claim.RewardID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to insert claim: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with success
	response := ClaimResponse{
		Success: true,
		Message: "Claim successfully created",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getRewardsHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query(ctx, "SELECT reward_id, name, description, inventory_count, cost FROM Reward")
    if err != nil {
        http.Error(w, "Failed to retrieve rewards", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var rewards []Reward
    for rows.Next() {
        var reward Reward
        if err := rows.Scan(&reward.RewardID, &reward.Name, &reward.Description, &reward.InventoryCount, &reward.Cost); err != nil {
            http.Error(w, "Failed to scan reward", http.StatusInternalServerError)
            return
        }
        rewards = append(rewards, reward)
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(rewards)
}

// CORS middleware to allow all origins
func handleCORS(w http.ResponseWriter, r *http.Request) {
	// Allow all origins
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// Allow specific methods
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	// Allow specific headers
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// If it's a preflight request, just respond with 200
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
}


func main() {
	// Set default Redis address if not provided
	if os.Getenv("REDIS_ADDR") == "" {
		os.Setenv("REDIS_ADDR", "localhost:6379")
	}

	// Connect to the database
	connectToDB()
	defer db.Close()

	router := mux.NewRouter()

	// Apply CORS handler before every route
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleCORS(w, r) // Handle CORS
			next.ServeHTTP(w, r)
		})
	})

	router.HandleFunc("/execute", executeHandler).Methods("POST")
	router.HandleFunc("/result/{id}", resultHandler).Methods("GET")
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")
	router.HandleFunc("/claim", claimHandler).Methods("POST")
	router.HandleFunc("/rewards", getRewardsHandler).Methods("GET")

	log.Println("API server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
