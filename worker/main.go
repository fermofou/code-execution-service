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
	TestCases int 		`json:"test_cases"`
	totalTestCases int `json:"total_test_cases"`
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
	// Map from language to executor path inside the container
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
	}

	startTime := time.Now()

	// Store code for HTTP server (so executors can do an HTTP GET)
	codeID := uuid.New().String()
	codeStore[codeID] = job.Code
	defer delete(codeStore, codeID)

	// Determine worker‐host and port (for CODE_URL)
	workerHost := os.Getenv("WORKER_HOST")
	if workerHost == "" {
		workerHost = "worker"
	}
	workerPort := os.Getenv("WORKER_PORT")
	if workerPort == "" {
		workerPort = "8081"
	}

	// Pick the executor image
	var containerImage string
	switch job.Language {
	case "python":
		containerImage = "python-executor:latest"
	case "javascript":
		containerImage = "javascript-executor:latest"
	case "cpp":
		containerImage = "cpp-executor:latest"
	case "csharp":
		containerImage = "csharp-executor:latest"
	default:
		return JobResult{
			JobID:     job.ID,
			Status:    "error",
			Error:     fmt.Sprintf("Unsupported language: %s", job.Language),
			Timestamp: time.Now(),
		}
	}

	// “validate” == we have multiple Inputs/Outputs (a submission with test cases)
	validate := len(job.Inputs) > 0 && len(job.Outputs) > 0

	if validate {
    // 1) Create the per-job folder under /code
    tmpDir := filepath.Join("/code", "codeexec-"+job.ID)
    if err := os.MkdirAll(tmpDir, 0755); err != nil {
        return JobResult{
            JobID:     job.ID,
            Status:    "error",
            Error:     fmt.Sprintf("Failed to create tmpDir: %v", err),
            Timestamp: time.Now(),
        }
    }
    defer os.RemoveAll(tmpDir) // cleanup at the end

    // 2) Start one detached executor container (it sees /code as the shared volume)
    containerID := fmt.Sprintf("code-exec-%s", job.ID)
    _ = exec.Command("docker", "rm", "-f", containerID).Run() // best‐effort cleanup

    dockerRunArgs := []string{
        "run", "-d",
        "--name", containerID,
        "--network=code-execution-service_default",
        "--memory=100m", "--cpus=0.5", "--pids-limit=50",
        "-e", fmt.Sprintf("CODE_URL=http://%s:%s/code?id=%s", workerHost, workerPort, codeID),
        "-e", fmt.Sprintf("CODE_LANGUAGE=%s", job.Language),
        "-v", "shared-code:/code",
        containerImage,
    }
    if err := exec.Command("docker", dockerRunArgs...).Run(); err != nil {
        return JobResult{
            JobID:     job.ID,
            Status:    "error",
            Error:     fmt.Sprintf("Failed to start executor container: %v", err),
            Timestamp: time.Now(),
        }
    }
    defer exec.Command("docker", "rm", "-f", containerID).Run()

    // 3) For each test, write input.txt into /code/codeexec-<ID>, then `docker exec` with DIRTXT pointed there
    for i, input := range job.Inputs {
		// We don't actually need to write to input.txt since we're piping via stdin
		// But keeping it for compatibility if your executor needs it
		inputPath := filepath.Join(tmpDir, "input.txt")
		if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
			return JobResult{
				JobID:     job.ID,
				Status:    "error",
				Error:     fmt.Sprintf("Failed to write input.txt: %v", err),
				Timestamp: time.Now(),
			}
		}

		execCmd := exec.Command(
			"docker", "exec", "-i", // Add -i flag for interactive stdin
			"-e", fmt.Sprintf("CODE_URL=http://%s:%s/code?id=%s", workerHost, workerPort, codeID),
			"-e", fmt.Sprintf("CODE_LANGUAGE=%s", job.Language),
			"-e", fmt.Sprintf("DIRTXT=/code/codeexec-%s", job.ID),
			containerID,
			execPath,
		)
		
		// Provide input via stdin
		execCmd.Stdin = strings.NewReader(input)
		
		outputBytes, err := execCmd.CombinedOutput()
		actual := strings.TrimSpace(string(outputBytes))
		expected := strings.TrimSpace(job.Outputs[i])

		if err != nil || actual != expected {
			exec.Command("docker", "rm", "-f", containerID).Run()
			return JobResult{
				JobID:     job.ID,
				Status:    "fail",
				Output:    fmt.Sprintf("Test #%d failed\nInput: %q\nExpected: %q\nGot: %q", i+1, input, expected, actual),
				ExecTime:  time.Since(startTime).Milliseconds(),
				Timestamp: time.Now(),
				TestCases: i + 1,
				totalTestCases: len(job.Inputs),
			}
		}
	}

    	// All tests passed
    	return JobResult{
			JobID:     job.ID,
			Status:    "accept",
			Output:    "All tests passed.",
			ExecTime:  time.Since(startTime).Milliseconds(),
			Timestamp: time.Now(),
			TestCases: len(job.Inputs),
			totalTestCases: len(job.Inputs),
    	}
	}


	
	dockerArgs := []string{
		"run", "--rm",
		"--entrypoint", execPath,
		"--network=code-execution-service_default",
		"--memory=100m", "--cpus=0.5", "--pids-limit=50",
		"-e", fmt.Sprintf("CODE_URL=http://%s:%s/code?id=%s", workerHost, workerPort, codeID),
		"-e", fmt.Sprintf("CODE_LANGUAGE=%s", job.Language),
		"-v", "shared-code:/code",
		"-e", "DIRTXT=/code",
		containerImage,
	}

	outputBytes, err := exec.Command("docker", dockerArgs...).CombinedOutput()
	execTime := time.Since(startTime).Milliseconds()
	if err != nil {
		if execTime >= 5000 {
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
			Error:     fmt.Sprintf("Execution error: %v\nOutput: %s", err, string(outputBytes)),
			ExecTime:  execTime,
			Timestamp: time.Now(),
		}
	}

	return JobResult{
		JobID:     job.ID,
		Status:    "success",
		Output:    string(outputBytes),
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
		} else {
			log.Printf("Result stored in Redis for job %s (status: %s)", job.ID, jobResult.Status)
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
