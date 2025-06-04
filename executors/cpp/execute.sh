#!/bin/bash

# Check if CODE_URL is provided
if [ -z "$CODE_URL" ]; then
    echo "Error: CODE_URL environment variable not set." >&2
    exit 1
fi

# Create a temporary directory
TEMP_DIR=$(mktemp -d)
CODE_FILE="${TEMP_DIR}/code.cpp"

# Download the code using curl
curl -s "$CODE_URL" > "$CODE_FILE"

# Check if download was successful
if [ $? -ne 0 ] || [ ! -s "$CODE_FILE" ]; then
    echo "Error: Failed to download code from $CODE_URL" >&2
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Compile the code
g++ -std=c++17 -o "${TEMP_DIR}/program" "$CODE_FILE" 2>"${TEMP_DIR}/compile_error"

# Check if compilation was successful
if [ $? -ne 0 ]; then
    echo "Compilation error:" >&2
    cat "${TEMP_DIR}/compile_error" >&2
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Check if this is a single run or test run
if [ -n "$SINGLE" ]; then
    # Single run: check if stdin has data available
    if [ -t 0 ]; then
        # No input available, run without input
        timeout 5s "${TEMP_DIR}/program"
    else
        # Input available, pipe it
        timeout 5s "${TEMP_DIR}/program"
    fi
else
    # Test run: always read from stdin (piped via docker exec -i)
    timeout 5s "${TEMP_DIR}/program"
fi

# Capture the exit code
EXIT_CODE=$?

# Check if execution timed out
if [ $EXIT_CODE -eq 124 ]; then
    echo "Execution timed out." >&2
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Clean up
rm -rf "$TEMP_DIR"

# Exit with the same code as the program
exit $EXIT_CODE