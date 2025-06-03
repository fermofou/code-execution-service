package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
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

type LeaderboardUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Points   int    `json:"points"`
	Level    int    `json:"level"`
	ImageURL string `json:"image_url"`
}

// para pagina navbar tienda
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
	//TestCases string    `json:"testcases"`
	Inputs    []string            `json:"inputs"`
	Outputs   []string            `json:"outputs"`

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

// este es de compras
type Claim struct {
	UserID   string `json:"userID"`   // Exact match for your JSON
	RewardID int    `json:"rewardID"` // Exact match for your JSON
}

// para admins
type ClaimAdmins struct {
	ClaimID  int       `json:"claim_id"`
	Mail     string    `json:"mail"`
	Date     time.Time `json:"timestamp"`
	Name     string    `json:"name"`
	RewardID int       `json:"reward_id"`
}

// para mis compras
type ClaimUser struct {
	ClaimID  int       `json:"claim_id"`
	Date     time.Time `json:"timestamp"`
	Name     string    `json:"name"`
	RewardID int       `json:"reward_id"`
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

type EditProblemFormat struct {
	ProblemID   int    `json:"problem_id"`
	Title       string `json:"title"`
	Difficulty  int    `json:"difficulty"`
	TimeLimit   int    `json:"timelimit"`
	SampleTests string `json:"sampletests"`
	MemoryLimit int    `json:"memorylimit"`
	Question    string `json:"question"`
}

type TestCaseFiles struct {
	In  string `json:"in"`
	Out string `json:"out"`
}

type Badge struct {
	BadgeID     int       `json:"badge_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Requirement string    `json:"requirement"`
	ImageURL    string    `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type UpdateBadgeRequest struct {
	Badges []int `json:"medals"` // List of badge IDs to update
}

// User management types
type UpdateUserRequest struct {
	Name   string `json:"name"`
	Level  int    `json:"level"`
	Points int    `json:"points"`
}

func connectToDB() {
	//var err error
	err := godotenv.Load() // Load .env file
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = ""
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

	// struct that includes possible userId and prob id
	type ExecuteRequest struct {
		Language  string              `json:"language"`
		Code      string              `json:"code"`
		UserId    string              `json:"userId"`
		ProblemID string 			  `json:"probId"`
		Inputs  []string
		Outputs []string

	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request payload: %v", err), http.StatusBadRequest)
		return
	}
	
	//find testcases
		if req.UserId != "" && req.ProblemID != "" {
		rows, err := db.Query(ctx, `
			SELECT t.tin, t.tout
			FROM testcases t
			JOIN problem p ON p.problem_id = t.problem_id
			WHERE p.problem_id = $1;
		`, req.ProblemID)

		if err != nil {
			log.Printf("Warning: failed to fetch testcases for problem %s: %v", req.ProblemID, err)
			// Continue without testcases
		} else {
			defer rows.Close()

			for rows.Next() {
				var input, output string
				if err := rows.Scan(&input, &output); err != nil {
					log.Printf("Warning: error reading testcase row: %v", err)
					continue // Skip this testcase
				}
				req.Inputs = append(req.Inputs, input)
				req.Outputs = append(req.Outputs, output)
			}

			if err := rows.Err(); err != nil {
				log.Printf("Warning: error iterating over testcases: %v", err)
			}
		}
	}

	// Log received user ID if available, si es submission
	if req.UserId != "" {
		fmt.Printf("Received execution request from user: %s\n", req.UserId)
	}

	// Validate language
	if req.Language != "python" && req.Language != "javascript" && req.Language != "cpp" && req.Language != "csharp" {
		http.Error(w, "Unsupported language. Supported languages: python, javascript, cpp, c#", http.StatusBadRequest)
		return
	}

	// Create job with unique ID
	job := Job{
		ID:        uuid.NewString(),
		Language:  req.Language,
		Code:      req.Code,
		Timestamp: time.Now(),
		Inputs:    req.Inputs,
		Outputs:   req.Outputs,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		http.Error(w, "Error processing job", http.StatusInternalServerError)
		return
	}

	// Push job to Redis queue
	if err := rdb.LPush(ctx, "code_jobs",  jobData).Err(); err != nil {
		http.Error(w, "Failed to enqueue job", http.StatusInternalServerError)
		return
	}

	if err := rdb.LPush(ctx, "code_jobs", jobData).Err(); err != nil {
	http.Error(w, "Failed to enqueue job", http.StatusInternalServerError)
	return
	}

	if err := rdb.HSet(ctx, "job:"+job.ID, "status", "pending").Err(); err != nil {
		log.Printf(" Failed to set job status for %s: %v", job.ID, err)
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

	// Read the full request body for logging
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	log.Printf("Raw request body: %s", string(bodyBytes))

	var claim Claim
	if err := json.NewDecoder(r.Body).Decode(&claim); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Processing claim for userID: %s, rewardID: %d", claim.UserID, claim.RewardID)

	// Start a transaction to ensure all operations are consistent
	tx, err := db.Begin(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start transaction: %v", err), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx) // Will be a no-op if transaction is committed

	// 1. Verify reward exists
	var rewardExists bool
	var rewardCost int
	var inventoryCount int
	err = tx.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM reward WHERE reward_id = $1), "+
			"(SELECT cost FROM reward WHERE reward_id = $1), "+
			"(SELECT inventory_count FROM reward WHERE reward_id = $1)",
		claim.RewardID).Scan(&rewardExists, &rewardCost, &inventoryCount)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error checking reward: %v", err), http.StatusInternalServerError)
		return
	}

	if !rewardExists {
		http.Error(w, fmt.Sprintf("Reward with ID %d does not exist", claim.RewardID), http.StatusBadRequest)
		return
	}

	log.Printf("Reward #%d exists with cost: %d and inventory: %d", claim.RewardID, rewardCost, inventoryCount)

	// 2. Verify user exists and has enough points
	var userExists bool
	var userPoints int
	err = tx.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM \"User\" WHERE user_id = $1), "+
			"(SELECT points FROM \"User\" WHERE user_id = $1)",
		claim.UserID).Scan(&userExists, &userPoints)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error checking user: %v", err), http.StatusInternalServerError)
		return
	}

	if !userExists {
		http.Error(w, fmt.Sprintf("User with ID %s does not exist", claim.UserID), http.StatusBadRequest)
		return
	}

	log.Printf("User %s exists with %d points", claim.UserID, userPoints)

	// 3. Check if user has enough points
	if userPoints < rewardCost {
		http.Error(w, "Not enough points to claim this reward", http.StatusBadRequest)
		return
	}

	// 4. Check if reward has inventory
	if inventoryCount <= 0 {
		http.Error(w, "This reward is out of stock", http.StatusBadRequest)
		return
	}

	// 6. Now insert the claim record
	_, err = tx.Exec(ctx,
		"INSERT INTO claims (user_id, reward_id) VALUES ($1, $2)",
		claim.UserID, claim.RewardID)
	if err != nil {
		log.Printf("ERROR inserting claim: %v", err)
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			http.Error(w, fmt.Sprintf("Foreign key violation. Details: %v", err), http.StatusBadRequest)
		} else {
			http.Error(w, fmt.Sprintf("Failed to insert claim: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to commit transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with success
	response := ClaimResponse{
		Success: true,
		Message: "Reward claimed successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
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

// conectar clerk con id
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
		`SELECT name, points, level, is_admin FROM "User" WHERE user_id = $1`,
		clerkID,
	).Scan(&userData.Name, &userData.Points, &userData.Level, &userData.Admin)

	if err == pgx.ErrNoRows {
		_, insertErr := db.Exec(context.Background(),
			`INSERT INTO "User" (user_id, name, mail, points, level, is_admin)
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
	rows, err := db.Query(ctx, `SELECT user_id, name, points, level FROM "User" WHERE is_admin = false ORDER BY points DESC LIMIT 10`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch leaderboard: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []LeaderboardUser

	for rows.Next() {
		var u LeaderboardUser
		if err := rows.Scan(&u.ID, &u.Name, &u.Points, &u.Level); err != nil {
			http.Error(w, "Failed to scan user", http.StatusInternalServerError)
			return
		}

		// Fetch image from Clerk
		clerkUserURL := fmt.Sprintf("https://api.clerk.com/v1/users/%s", u.ID)
		req, err := http.NewRequest("GET", clerkUserURL, nil)
		if err != nil {
			http.Error(w, "Failed to create Clerk request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", "Bearer "+os.Getenv("CLERK_SECRET_KEY"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "Failed to fetch user from Clerk", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			u.ImageURL = "" // fallback
		} else {
			var clerkData struct {
				ImageURL string `json:"image_url"`
			}
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &clerkData)
			u.ImageURL = clerkData.ImageURL
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
	// fmt.Printf("Fetching all problems for userID: %s\n", userID)

	query := `
		WITH user_submissions AS (
			SELECT s.*
			FROM submission s
			
			WHERE s.user_id = $1
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

	// fmt.Printf("Fetching challenge with probID: %s\n", probID)

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

func editProblemStatement(w http.ResponseWriter, r *http.Request) {
	var problem EditProblemFormat
	if err := json.NewDecoder(r.Body).Decode(&problem); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(ctx,
		`UPDATE problem SET title = $1, difficulty = $2, timelimit = $3, memorylimit = $4, question = $5, inputs = $6, outputs = $7, tests = $8 WHERE problem_id = $9 RETURNING problem_id`,
		problem.Title, problem.Difficulty, problem.TimeLimit, problem.MemoryLimit, problem.Question, []string{}, []string{}, problem.SampleTests, problem.ProblemID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update problem: %v", err), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func deleteProblem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	problemID := r.URL.Query().Get("problemId")

	if problemID == "" {
		return
	}

	_, err := db.Exec(ctx, `DELETE FROM problem WHERE problem_id = $1`, problemID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete problem: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Status string `json:"status"`
	}{
		Status: "success",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func uploadTestCases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	respondWithError := func(status int, message string) {
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
	}

	problemID := r.URL.Query().Get("problemId")
	if problemID == "" {
		respondWithError(http.StatusBadRequest, "Missing problemId")
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		fmt.Println("Error parsing form:", err)
		respondWithError(http.StatusBadRequest, "No se pudo parsear el formulario")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error retrieving the file:", err)
		respondWithError(http.StatusBadRequest, "Error al obtener el archivo")
		return
	}
	defer file.Close()

	fmt.Println("Uploaded File:", handler.Filename)

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		fmt.Println("Error copying file to buffer:", err)
		respondWithError(http.StatusInternalServerError, "Error leyendo el archivo")
		return
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		fmt.Println("Error opening zip:", err)
		respondWithError(http.StatusInternalServerError, "Error abriendo el archivo zip")
		return
	}
	testCases := make(map[string]*TestCaseFiles)
	validNameRegex := regexp.MustCompile(`^\d+\.(in|out)$`)
	for _, zipFile := range reader.File {
		_, fileName := extractFileName(zipFile.Name)

		// Solo procesar archivos que matcheen el patrón
		if !validNameRegex.MatchString(fileName) {
			continue
		}

		parts := strings.Split(fileName, ".")
		if len(parts) != 2 {
			continue
		}
		number := parts[0]
		ext := parts[1]

		zippedFile, err := zipFile.Open()
		if err != nil {
			fmt.Println("Error opening file inside zip:", err)
			continue
		}

		content, err := io.ReadAll(zippedFile)
		zippedFile.Close()
		if err != nil {
			fmt.Println("Error reading file inside zip:", err)
			continue
		}

		// Inicializar si no existe
		if _, ok := testCases[number]; !ok {
			testCases[number] = &TestCaseFiles{}
		}

		// Guardar el contenido según sea .in o .out
		if ext == "in" {
			testCases[number].In = string(content)
		} else if ext == "out" {
			testCases[number].Out = string(content)
		}
	}

	for _, files := range testCases {
		if files.In == "" || files.Out == "" {
			continue
		}

		
		_, err := db.Exec(ctx, `INSERT INTO testcases (problem_id, tin, tout) VALUES ($1, $2, $3)`, problemID, files.In, files.Out)
		if err != nil {
			fmt.Println("Error inserting test case into database:", err)
			respondWithError(http.StatusInternalServerError, "Error inserting test case into database")
			return
		}
	}

	// Si todo fue exitoso
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Test cases procesados exitosamente"})
}

func extractFileName(path string) (string, string) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "", ""
	}
	return strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1]
}

func getBadgesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(ctx, `SELECT badge_id, name, description, requirement, image_url, created_at FROM badge`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve badges: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var badges []Badge
	for rows.Next() {
		var b Badge
		if err := rows.Scan(&b.BadgeID, &b.Name, &b.Description, &b.Requirement, &b.ImageURL, &b.CreatedAt); err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan badge: %v", err), http.StatusInternalServerError)
			return
		}
		badges = append(badges, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(badges)
}

func createBadgeHandler(w http.ResponseWriter, r *http.Request) {
	var badge Badge
	if err := json.NewDecoder(r.Body).Decode(&badge); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	var badgeID int
	err := db.QueryRow(ctx, `
		INSERT INTO badge (name, description, requirement, image_url, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING badge_id`,
		badge.Name, badge.Description, badge.Requirement, badge.ImageURL,
	).Scan(&badgeID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to insert badge: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "created",
		"badge_id": badgeID,
	})
}

func updateBadgeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var badge Badge
	if err := json.NewDecoder(r.Body).Decode(&badge); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(ctx, `
		UPDATE badge
		SET name = $1, description = $2, requirement = $3, image_url = $4
		WHERE badge_id = $5`,
		badge.Name, badge.Description, badge.Requirement, badge.ImageURL, id,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update badge: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func deleteBadgeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := db.Exec(ctx, `DELETE FROM badge WHERE badge_id = $1`, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete badge: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(ctx, `SELECT user_id, name, mail, points, level, is_admin FROM "User" ORDER BY points DESC`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Mail    string `json:"mail"`
		Points  int    `json:"points"`
		Level   int    `json:"level"`
		IsAdmin bool   `json:"is_admin"`
	}

	for rows.Next() {
		var u struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Mail    string `json:"mail"`
			Points  int    `json:"points"`
			Level   int    `json:"level"`
			IsAdmin bool   `json:"is_admin"`
		}
		if err := rows.Scan(&u.ID, &u.Name, &u.Mail, &u.Points, &u.Level, &u.IsAdmin); err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan user: %v", err), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.Level < 1 {
		http.Error(w, "Level must be at least 1", http.StatusBadRequest)
		return
	}
	if req.Points < 0 {
		http.Error(w, "Points cannot be negative", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(ctx,
		`UPDATE "User" 
		 SET name = $1, level = $2, points = $3 
		 WHERE user_id = $4`,
		req.Name, req.Level, req.Points, userID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func getUserBadgesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	rows, err := db.Query(ctx, `
		SELECT b.badge_id, b.name, b.description, b.requirement, b.image_url, ub.awarded_at
		FROM badge b
		JOIN user_badge ub ON b.badge_id = ub.badge_id
		WHERE ub.user_id = $1
		ORDER BY ub.awarded_at DESC`,
		userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve user badges: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var badges []Badge
	for rows.Next() {
		var b Badge
		if err := rows.Scan(&b.BadgeID, &b.Name, &b.Description, &b.Requirement, &b.ImageURL, &b.CreatedAt); err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan badge: %v", err), http.StatusInternalServerError)
			return
		}
		badges = append(badges, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(badges)
}

// admin ver todas las compras
func getAllClaimsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query(ctx, `
		SELECT c.claim_id, u.mail, c.date, r.name, r.reward_id
		FROM claims c
		LEFT JOIN reward r ON r.reward_id = c.reward_id
		LEFT JOIN "User" u ON u.user_id = c.user_id
		ORDER BY c.date DESC
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve claims: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var claims []ClaimAdmins

	for rows.Next() {
		var c ClaimAdmins
		err := rows.Scan(&c.ClaimID, &c.Mail, &c.Date, &c.Name, &c.RewardID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan claim: %v", err), http.StatusInternalServerError)
			return
		}
		claims = append(claims, c)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

func getUserClaimsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "Missing userId parameter", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(ctx, `
		SELECT c.claim_id, c.date, r.name, r.reward_id
		FROM claims c
		LEFT JOIN reward r ON r.reward_id = c.reward_id
		WHERE c.user_id = $1
		ORDER BY c.date DESC
	`, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve user claims: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var claims []ClaimUser
	for rows.Next() {
		var c ClaimUser
		err := rows.Scan(&c.ClaimID, &c.Date, &c.Name, &c.RewardID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan user claim: %v", err), http.StatusInternalServerError)
			return
		}
		claims = append(claims, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

func updateUserBadgesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]
	// get the badge IDs from the request body
	var badgeIDs []int
	if err := json.NewDecoder(r.Body).Decode(&badgeIDs); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// hacer un query para eliminar los badges que ya no estan en la lista poniendo uno por uno
	_, err := db.Exec(ctx, `DELETE FROM user_badge WHERE user_id = $1 AND badge_id NOT IN (SELECT unnest($2::int[]))`, userID, badgeIDs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user badges: %v", err), http.StatusInternalServerError)
		return
	}

	// insertar las badges que no están en la lista INSERT INTO "user_badge" (user_id, badge_id) VALUES ('user_2xScE26jLQSnkf5GZjSVsmJVP75', 1)
	for _, badgeID := range badgeIDs {
		_, err := db.Exec(ctx, `INSERT INTO user_badge (user_id, badge_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, badgeID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to insert user badge: %v", err), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "badges updated"})
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

func getAdminStats(w http.ResponseWriter, r *http.Request) {
	type RewardStat struct {
		Name  string `json:"reward"`
		Count int    `json:"timesRedeemed"`
	}

	type LangStat struct {
		Language    string `json:"language"`
		Submissions int    `json:"submissions"`
	}

	type NewUser struct {
		Name string `json:"name"`
		Date string `json:"date"`
	}

	type BadgeWeek struct {
		Week   string `json:"week"`
		Badges int    `json:"badges"`
	}

	stats := make(map[string]interface{})

	query := `
		SELECT 
			(SELECT COUNT(*) FROM "User") AS total_users,
			(SELECT COUNT(*) FROM badge) AS total_badges,
			(SELECT AVG(level) FROM "User") AS average_level,
			(SELECT COUNT(*) FROM problem) AS total_problems,
			(SELECT COUNT(*) FROM submission WHERE date >= NOW() - INTERVAL '10 minutes') AS active_sessions
	`
	var totalUsers, totalBadges, totalProblems, activeSessions int
	var averageLevel float64

	err := db.QueryRow(ctx, query).Scan(&totalUsers, &totalBadges, &averageLevel, &totalProblems, &activeSessions)
	if err != nil {
		http.Error(w, "Error retrieving stats", http.StatusInternalServerError)
		return
	}

	stats["totalUsers"] = totalUsers
	stats["totalBadges"] = totalBadges
	stats["averageLevel"] = int(averageLevel)
	stats["activeSessions"] = activeSessions
	stats["totalProblems"] = totalProblems

	rewardsQuery := `
		SELECT r.name, COUNT(*) AS count
		FROM claims c
		JOIN reward r ON r.reward_id = c.reward_id
		GROUP BY r.name
		ORDER BY count DESC
		LIMIT 5;
	`
	var rewards []RewardStat
	rows, _ := db.Query(ctx, rewardsQuery)
	for rows.Next() {
		var r RewardStat
		rows.Scan(&r.Name, &r.Count)
		rewards = append(rewards, r)
	}
	stats["rewardsRedeemed"] = rewards

	languagesQuery := `
		SELECT language, COUNT(*) as submissions
		FROM submission
		GROUP BY language
		ORDER BY submissions DESC
		LIMIT 5;
	`
	var langs []LangStat
	rows, _ = db.Query(ctx, languagesQuery)
	for rows.Next() {
		var l LangStat
		rows.Scan(&l.Language, &l.Submissions)
		langs = append(langs, l)
	}
	stats["popularLanguages"] = langs

	usersQuery := `
		SELECT name, date_trunc('day', CURRENT_DATE - (CURRENT_DATE - created)::interval) as created
		FROM "User"
		WHERE created >= CURRENT_DATE - INTERVAL '7 days'
		ORDER BY created DESC
		LIMIT 5;
	`
	var newUsers []NewUser
	rows, _ = db.Query(ctx, usersQuery)
	for rows.Next() {
		var u NewUser
		var created time.Time
		rows.Scan(&u.Name, &created)
		u.Date = created.Format("Jan 2")
		newUsers = append(newUsers, u)
	}
	stats["newUsers"] = newUsers

	badgesQuery := `
		SELECT TO_CHAR(awarded_at, 'YYYY-"W"IW') as week, COUNT(*) 
		FROM user_badge
		GROUP BY week
		ORDER BY week DESC
		LIMIT 5;
	`
	var growth []BadgeWeek
	rows, _ = db.Query(ctx, badgesQuery)
	for rows.Next() {
		var b BadgeWeek
		rows.Scan(&b.Week, &b.Badges)
		growth = append(growth, b)
	}
	stats["userGrowthData"] = growth

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
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
	router.HandleFunc("/admin/editProblemStatement", editProblemStatement).Methods("POST", "OPTIONS")
	router.HandleFunc("/admin/deleteProblem", deleteProblem).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/admin/uploadTestcases", uploadTestCases).Methods("POST", "OPTIONS")
	router.HandleFunc("/badges", getBadgesHandler).Methods("GET")
	router.HandleFunc("/badges", createBadgeHandler).Methods("POST")
	router.HandleFunc("/badges/{id}", updateBadgeHandler).Methods("PUT")
	router.HandleFunc("/badges/{id}", deleteBadgeHandler).Methods("DELETE")
	router.HandleFunc("/admin/users", getAllUsersHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/admin/updateUser/{user_id}", updateUserHandler).Methods("PUT", "OPTIONS")
	router.HandleFunc("/admin/user/{id}/updateBadges", updateUserBadgesHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/admin/user/{id}/badges", getUserBadgesHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/admin/claims", getAllClaimsHandler).Methods("GET")
	router.HandleFunc("/myRewards", getUserClaimsHandler).Methods("GET")
	router.HandleFunc("/admin/stats", getAdminStats).Methods("GET")

	log.Println("API server running on port 8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", router))
}
