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
	"github.com/jackc/pgx/v4"

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

//para leaderboard
type User struct {
	Name   string `json:"name"`
	Points int    `json:"points"`
	Level  int    `json:"level"`

}

//para pagina navbar tienda
type UserData struct {
	Name   string `json:"name"`
	Points int    `json:"points"`
	Level  int    `json:"level"`
	Admin  bool   `json:"admin"`
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
	RewardID       int    `json:"reward_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	InventoryCount int    `json:"inventory_count"`
	Cost           int    `json:"cost"`
}

type Claim struct {
	UserID   string `json:"user_id"`
	RewardID int    `json:"reward_id"`
}

type ClaimResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Define problem structure
type ProblemSmall struct {
	ProblemID  int    `json:"problem_id"`
	Title      string `json:"title"`
	Difficulty int    `json:"difficulty"`
	Solved     *bool  `json:"solved"` // pointer to allow NULL
}

// parte
type Problem struct {
	ProblemID   int      `json:"problem_id"`
	Title       string   `json:"title"`
	Difficulty  int      `json:"difficulty"`
	Solved      *bool    `json:"solved"` // pointer to allow NULL
	TimeLimit   int      `json:"timelimit"`
	Tests       string   `json:"tests"`
	MemoryLimit int      `json:"memorylimit"`
	Question    string   `json:"question"`
	Inputs      []string `json:"inputs"`
	Outputs     []string `json:"outputs"`
}

type UploadProblemFormat struct {
	Title       string `json:"title"`
	Difficulty  int    `json:"difficulty"`
	TimeLimit   int    `json:"timelimit"`
	SampleTests string `json:"sampletests"`
	MemoryLimit int    `json:"memorylimit"`
	Question    string `json:"question"`
	// Tags        []string `json:"tags"`
}

func connectToDB() {
	var err error
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://avnadmin:AVNS_lOhjYg-hwx2CdWSGKk_@postgres-moran-tec-c540.j.aivencloud.com:13026/defaultdb?sslmode=require"
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

	// Define a struct that includes userId
	type ExecuteRequest struct {
		Language  string              `json:"language"`
		Code      string              `json:"code"`
		TestCases []map[string]string `json:"testCases"`
		UserId    string              `json:"userId"`
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request payload: %v", err), http.StatusBadRequest)
		return
	}

	if req.Language == "" || req.Code == "" {
		http.Error(w, "Missing required fields (language, code)", http.StatusBadRequest)
		return
	}

	// Log received user ID if available
	if req.UserId != "" {
		fmt.Printf("Received execution request from user: %s\n", req.UserId)
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

	// Handle Clerk IDs by extracting a consistent hash value
	var userIDForDB string
	if len(claim.UserID) > 20 && claim.UserID[:5] == "user_" {
		// This appears to be a Clerk ID, use a hash of it for DB queries
		userIDForDB = claim.UserID[5:10] // Take a portion to create a simpler ID
		fmt.Printf("Converting Clerk user ID to: %s for claim\n", userIDForDB)
	} else {
		userIDForDB = claim.UserID
	}

	// Use the simplified user ID for database operations
	_, err := db.Exec(ctx, "INSERT INTO Claims (user_id, reward_id) VALUES ($1, $2)", userIDForDB, claim.RewardID)
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

//conectar clerk con id
func getDataUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clerkID := vars["clerk_id"]
	name := vars["name"]
	email := vars["email"]

	if clerkID == "" {
		http.Error(w, "clerk_id parameter is required", http.StatusBadRequest)
		return
	}

	var userData UserData
	err := db.QueryRow(context.Background(),
		`SELECT name, points, level, is_admin FROM "User" WHERE clerk_id = $1`,
		clerkID,
	).Scan(&userData.Name, &userData.Points, &userData.Level, &userData.Admin)

	if err == pgx.ErrNoRows {
		_, insertErr := db.Exec(context.Background(),
			`INSERT INTO "User" (clerk_id, name, mail, points, level, is_admin)
			 VALUES ($1, $2, $3, 0, 1, false)`,
			clerkID, name, email,
		)
		if insertErr != nil {
			http.Error(w, fmt.Sprintf("Insert error: %v", insertErr), http.StatusInternalServerError)
			return
		}

		userData = UserData{
			Name:   name,
			Points: 0,
			Level:  1,
			Admin:  false,
		}
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userData)
}



func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(ctx, `SELECT name, points, level FROM "User" WHERE is_admin = false ORDER BY points DESC LIMIT 10`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch leaderboard: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.Name, &u.Points, &u.Level); err != nil {
			http.Error(w, "Failed to scan user", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func getAllProblems(w http.ResponseWriter, r *http.Request) {
	// Set headers
	w.Header().Set("Content-Type", "application/json")

	// Extract user ID from query parameters
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		// Log the error but use a fallback default user ID instead of failing
		fmt.Printf("Warning: Missing userId parameter in getAllProblems. Request URL: %s. Using fallback userId.\n", r.URL.String())
		userID = "default_fallback_id" // Replace with your actual fallback clerk ID if needed
	}
	fmt.Printf("Fetching all problems for userID: %s\n", userID)

	query := `
		WITH user_submissions AS (
			SELECT s.*
			FROM submission s
			JOIN "User" u ON u.user_id = s.user_id
			WHERE u.clerk_id = $1
		)
		SELECT 
			p.problem_id,
			p.title,
			p.difficulty,
			CASE 
				WHEN MAX(CASE WHEN us.correct = true THEN 1 ELSE 0 END) = 1 THEN true
				WHEN COUNT(us.submission_id) > 0 THEN false
				ELSE NULL
			END AS solved
		FROM 
			problem p
		LEFT JOIN 
			user_submissions us ON p.problem_id = us.problem_id
		GROUP BY 
			p.problem_id, p.title, p.difficulty
		ORDER BY 
			p.problem_id;
	`

	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve problems: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var problems []ProblemSmall
	for rows.Next() {
		var p ProblemSmall
		if err := rows.Scan(&p.ProblemID, &p.Title, &p.Difficulty, &p.Solved); err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan problem: %v", err), http.StatusInternalServerError)
			return
		}
		problems = append(problems, p)
	}

	// Check for errors after iterating through rows
	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating through problems: %v", err), http.StatusInternalServerError)
		return
	}

	// Encode and return the problems as JSON
	if err := json.NewEncoder(w).Encode(problems); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode problems: %v", err), http.StatusInternalServerError)
		return
	}
}



// parte
func getChallengeId(w http.ResponseWriter, r *http.Request) {
	// Set headers
	w.Header().Set("Content-Type", "application/json")

	// Only extract probID from query parameters
	probID := r.URL.Query().Get("probID")
	if probID == "" {
		// Log the error but use a fallback default problem ID
		fmt.Printf("Warning: Missing probID parameter. Request URL: %s. Using fallback probID.\n", r.URL.String())
		probID = "1" // Fallback problem ID
	}

	fmt.Printf("Fetching challenge with probID: %s\n", probID)

	// Simple query to get problem details without any user-specific data
	rows, err := db.Query(ctx, `   
   SELECT 
    p.problem_id,
    p.title,
    p.difficulty,
    p.question,
    p.inputs,
    p.outputs,
    p.timelimit,
    p.memorylimit,
    p.tests,
    NULL AS solved
FROM 
    problem p
WHERE 
    p.problem_id = $1;
`, probID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve problems: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Create a Problem struct
	var problem Problem
	if rows.Next() {
		err := rows.Scan(&problem.ProblemID, &problem.Title, &problem.Difficulty, &problem.Question, &problem.Inputs, &problem.Outputs, &problem.TimeLimit, &problem.MemoryLimit, &problem.Tests, &problem.Solved)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan problem: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		// No problem found
		http.Error(w, fmt.Sprintf("Problem with ID %s not found", probID), http.StatusNotFound)
		return
	}

	// Check for errors after iterating through rows
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating through problems: %v", err), http.StatusInternalServerError)
		return
	}

	// Encode and return the problem as JSON
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode problem: %v", err), http.StatusInternalServerError)
		return
	}
}

func uploadProblemStatement(w http.ResponseWriter, r *http.Request) {
	var problem UploadProblemFormat
	if err := json.NewDecoder(r.Body).Decode(&problem); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	rows, err := db.Query(ctx,
		`INSERT INTO problem (title, difficulty, timelimit, memorylimit, question, answer, inputs, outputs, tests)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING problem_id`,
		problem.Title, problem.Difficulty, problem.TimeLimit, problem.MemoryLimit, problem.Question, " ", []string{}, []string{}, problem.SampleTests)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to insert problem: %v", err), http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	var problemID int

	if rows.Next() {
		if err := rows.Scan(&problemID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan problem ID: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating through problems: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Status    string `json:"status"`
		ProblemID int    `json:"problem_id"`
	}{
		Status:    "success",
		ProblemID: problemID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("Failed to encode response: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
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
	router.HandleFunc("/leaderboard", leaderboardHandler).Methods("GET")
	router.HandleFunc("/problems", getAllProblems).Methods("GET")
	router.HandleFunc("/challenge", getChallengeId).Methods("GET")
	router.HandleFunc("/user/{clerk_id}/{name}/{email}", getDataUser).Methods("GET")
	router.HandleFunc("/admin/uploadProblemStatement", uploadProblemStatement).Methods("POST", "OPTIONS")


	log.Println("API server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
