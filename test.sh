#!/bin/bash

# Exit on error
set -e

API_URL="http://localhost:8080"

echo "Testing LeetCode Clone Code Execution Service..."

# Test Python code execution
echo "Testing Python code execution..."
PYTHON_RESPONSE=$(curl -s -X POST "$API_URL/execute" \
  -H "Content-Type: application/json" \
  -d '{"language":"python","code":"print(\"Hello from Python!\")"}')

PYTHON_JOB_ID=$(echo $PYTHON_RESPONSE | sed 's/.*"job_id": "\([^"]*\)".*/\1/')
echo "Python job ID: $PYTHON_JOB_ID"

# Test JavaScript code execution
echo "Testing JavaScript code execution..."
JS_RESPONSE=$(curl -s -X POST "$API_URL/execute" \
  -H "Content-Type: application/json" \
  -d '{"language":"javascript","code":"console.log(\"Hello from JavaScript!\")"}')

JS_JOB_ID=$(echo $JS_RESPONSE | sed 's/.*"job_id": "\([^"]*\)".*/\1/')
echo "JavaScript job ID: $JS_JOB_ID"

# Test C++ code execution
echo "Testing C++ code execution..."
CPP_RESPONSE=$(curl -s -X POST "$API_URL/execute" \
  -H "Content-Type: application/json" \
  -d '{"language":"cpp","code":"#include <iostream>\nint main() {\n  std::cout << \"Hello from C++!\" << std::endl;\n  return 0;\n}"}')

CPP_JOB_ID=$(echo $CPP_RESPONSE | sed 's/.*"job_id": "\([^"]*\)".*/\1/')
echo "C++ job ID: $CPP_JOB_ID"

# Wait for jobs to complete
echo "Waiting for jobs to complete..."
sleep 10

# Check Python result
echo "Checking Python result..."
curl -s "$API_URL/result/$PYTHON_JOB_ID"
echo ""

# Check JavaScript result
echo "Checking JavaScript result..."
curl -s "$API_URL/result/$JS_JOB_ID"
echo ""

# Check C++ result
echo "Checking C++ result..."
curl -s "$API_URL/result/$CPP_JOB_ID"
echo ""

echo "Test complete!" 