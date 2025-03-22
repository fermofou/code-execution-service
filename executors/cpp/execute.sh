#!/bin/bash

# Check if CODE_URL is provided
if [ -z "$CODE_URL" ]; then
    echo "STDERR:"
    echo "Error: CODE_URL environment variable not set."
    exit 1
fi

# Create a temporary directory
TEMP_DIR=$(mktemp -d)
CODE_FILE="${TEMP_DIR}/code.cpp"

echo "Fetching code from: $CODE_URL"

# Download the code using curl
curl -s "$CODE_URL" > "$CODE_FILE"

# Check if download was successful
if [ $? -ne 0 ] || [ ! -s "$CODE_FILE" ]; then
    echo "STDERR:"
    echo "Error: Failed to download code from $CODE_URL"
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "Compiling file: $CODE_FILE"

# Compile the code
g++ -std=c++17 -o "${TEMP_DIR}/program" "$CODE_FILE"

# Check if compilation was successful
if [ $? -ne 0 ]; then
    echo "STDERR:"
    echo "Compilation error."
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "Executing compiled program"

# Run the program with timeout
timeout 5s "${TEMP_DIR}/program" > "${TEMP_DIR}/stdout" 2> "${TEMP_DIR}/stderr"

# Check if execution timed out
if [ $? -eq 124 ]; then
    echo "STDERR:"
    echo "Execution timed out."
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Output results
echo "STDOUT:"
cat "${TEMP_DIR}/stdout"
echo "STDERR:"
cat "${TEMP_DIR}/stderr"

# Clean up
rm -rf "$TEMP_DIR"
