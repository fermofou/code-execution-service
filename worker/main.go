package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

var ctx = context.Background()

// Initialize Redis client
var rdb = redis.NewClient(&redis.Options{
	Addr: os.Getenv("REDIS_ADDR"),
})

// Map to store code by ID
var codeStore = make(map[string]string)


// Job represents a code execution job
type Job struct {
	ID        string    `json:"id"`
	Language  string    `json:"language"`
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
	Inputs    []string  `json:"inputs"`
	Outputs   []string  `json:"outputs"`
}

// JobResult represents the result of a code execution
type JobResult struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	Output    string    `json:"output"`
	Error     string    `json:"error"`
	ExecTime  int64     `json:"exec_time_ms"`
	Timestamp time.Time `json:"timestamp"`
}

// HTTP handler for serving code files
func codeHandler(w http.ResponseWriter, r *http.Request) {
	codeID := r.URL.Query().Get("id")
	if codeID == "" {
		http.Error(w, "Code ID is required", http.StatusBadRequest)
		return
	}

	code, exists := codeStore[codeID]
	if !exists {
		http.Error(w, "Code not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, code)
}


// executeCode executes the code in a Docker container
func executeCode(job Job) JobResult {
	execPaths := map[string]string{
		"python":    "/app/executor.py",
		"javascript": "/executor/executor.js",
		"cpp":       "/app/execute.sh",
		"csharp":    "/app/execute.sh",
	}
	execPath, ok := execPaths[job.Language]
	if !ok {
		return JobResult{
			JobID:     job.ID,
			Status:    "error",
			Error:     fmt.Sprintf("Unsupported language: %s", job.Language),
			Timestamp: time.Now(),
	}

	startTime := time.Now()

	// Store code for HTTP server
	codeID := uuid.New().String()
	codeStore[codeID] = job.Code

	defer delete(codeStore, codeID) // Clean up after execution

	// Get worker hostname and port
	workerHost := os.Getenv("WORKER_HOST")
	if workerHost == "" {
		workerHost = "worker" // Default to service name in docker-compose
	}

	workerPort := os.Getenv("WORKER_PORT")
	if workerPort == "" {
		workerPort = "8081" // Default HTTP port
	}

		// Determine container name based on language
	var containerName string
	switch job.Language {
	case "python":
		containerName = "python-executor:latest"
	case "javascript":
		containerName = "javascript-executor:latest"
	case "cpp":
		containerName = "cpp-executor:latest"
	case "csharp":
		containerName = "csharp-executor:latest"

	default:
		return JobResult{
			JobID:     job.ID,
			Status:    "error",
			Error:     fmt.Sprintf("Unsupported language: %s", job.Language),
			Timestamp: time.Now(),
		}
	}

	

	// Run the container with the code ID as argument
	containerID := fmt.Sprintf("code-exec-%s", job.ID)
	validate := len(job.Inputs) > 0 && len(job.Outputs) > 0

	dockerArgs := []string{
		"run", "-d", "--name", containerID,
		"--network=code-execution-service_default",
		"--memory=100m", "--cpus=0.5", "--pids-limit=50",
		"-e", fmt.Sprintf("CODE_URL=http://%s:%s/code?id=%s", workerHost, workerPort, codeID),
		"-e", fmt.Sprintf("CODE_LANGUAGE=%s", job.Language),
	}

	var tmpDir string
	if validate {
		tmpDir = filepath.Join("/tmp", "codeexec-"+job.ID)
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/app/testdata", tmpDir), "-e", "DIRTXT=/app/testdata")
	}
	dockerArgs = append(dockerArgs, containerName)
	if err := exec.Command("docker", dockerArgs...).Run(); err != nil {
		return JobResult{
			JobID:     job.ID,
			Status:    "error",
			Error:     fmt.Sprintf("Failed to start Docker container: %v", err),
			Timestamp: time.Now(),
		}
	}
	defer exec.Command("docker", "rm", "-f", containerID).Run()

	if validate {
		if len(job.Inputs) != len(job.Outputs) {
			return JobResult{
				JobID:     job.ID,
				Status:    "error",
				Error:     "Mismatched input/output count",
				Timestamp: time.Now(),
			}
		}
		for i, input := range job.Inputs {
			inputPath := filepath.Join(tmpDir, "input.txt")
			if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
				return JobResult{JobID: job.ID, Status: "error", Error: err.Error(), Timestamp: time.Now()}
			}
			execCmd := exec.Command("docker", "exec", containerID, execPath)
			outputBytes, err := execCmd.CombinedOutput()
			actual := strings.TrimSpace(string(outputBytes))
			expected := strings.TrimSpace(job.Outputs[i])
			if err != nil || actual != expected {
				// Kill container and return on first failure
				exec.Command("docker", "rm", "-f", containerID).Run()
				return JobResult{
					JobID:     job.ID,
					Status:    "fail",
					Output:    fmt.Sprintf("Test #%d failed\nInput: %q\nExpected: %q\nGot: %q", i+1, input, expected, actual),
					ExecTime:  time.Since(startTime).Milliseconds(),
					Timestamp: time.Now(),
				}
			}
		}

		return JobResult{
		JobID:     job.ID,
		Status:    "pass",
		Output:    "All tests passed.",
		ExecTime:  time.Since(startTime).Milliseconds(),
		Timestamp: time.Now(),
		}	
	}

	execCmd := exec.Command("docker", "logs", containerID)
	output, err := execCmd.CombinedOutput()
	execTime := time.Since(startTime).Milliseconds()
	if err != nil {
		if execTime >= 5000 { // 5 seconds
			return JobResult{
				JobID:     job.ID,
				Status:    "timeout",
				Error:     "Code execution timed out",
				ExecTime:  execTime,
				Timestamp: time.Now(),
			}
		}

		return JobResult{
			JobID:     job.ID,
			Status:    "error",
			Error:     fmt.Sprintf("Execution error: %v\nOutput: %s", err, string(output)),
			ExecTime:  execTime,
			Timestamp: time.Now(),
		}
	}

	return JobResult{
		JobID:     job.ID,
		Status:    "success",
		Output:    string(output),
		ExecTime:  execTime,
		Timestamp: time.Now(),
	}
}

func processJobs() {
	for {
		// Pop job from Redis queue with timeout
		result, err := rdb.BRPop(ctx, 5*time.Second, "code_jobs").Result()
		if err != nil {
			if err == redis.Nil {
				// No jobs available, continue polling
				continue
			}
			log.Printf("Error fetching job from Redis: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Parse job data
		var job Job
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			log.Printf("Error parsing job data: %v", err)
			continue
		}

		log.Printf("Processing job %s (Language: %s)", job.ID, job.Language)

		// Execute code
		jobResult := executeCode(job)

		// Store result in Redis
		resultData, err := json.Marshal(jobResult)
		if err != nil {
			log.Printf("Error marshaling result data: %v", err)
			continue
		}

		// Store result with expiration (24 hours)
		if err := rdb.Set(ctx, "result:"+job.ID, resultData, 24*time.Hour).Err(); err != nil {
			log.Printf("Error storing result in Redis: %v", err)
		}
	}
}

func main() {
	// Set default Redis address if not provided
	if os.Getenv("REDIS_ADDR") == "" {
		os.Setenv("REDIS_ADDR", "localhost:6379")
	}

	log.Println("Starting code execution worker...")

	// Start HTTP server for code serving
	http.HandleFunc("/code", codeHandler)
	
	port := os.Getenv("WORKER_PORT")
	if port == "" {
		port = "8081"
	}
	go func() {
		log.Printf("Starting HTTP server on port %s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start multiple worker goroutines to handle concurrent jobs
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		go processJobs()
	}

	// Keep the main goroutine alive
	select {}
}
