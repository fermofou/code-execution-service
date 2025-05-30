#!/bin/bash

# Function to log debug messages to stderr (won't interfere with stdout)
debug_log() {
    echo "[DEBUG] $1" >&2
}

# Function to clean up and exit
cleanup_and_exit() {
    local exit_code=$1
    debug_log "Cleaning up temporary directory: $TEMP_DIR"
    rm -rf "$TEMP_DIR" 2>/dev/null
    exit $exit_code
}

debug_log "Starting C# execution script..."

# Check if CODE_URL is provided
if [ -z "$CODE_URL" ]; then
    echo "[ERROR] CODE_URL environment variable not set." >&2
    exit 1
fi

# Create a temporary directory
TEMP_DIR=$(mktemp -d)
if [ $? -ne 0 ]; then
    echo "[ERROR] Failed to create temporary directory." >&2
    exit 1
fi

CODE_FILE="${TEMP_DIR}/Program.cs"

debug_log "Temporary directory created at: $TEMP_DIR"
debug_log "Code will be saved to: $CODE_FILE"
debug_log "Fetching code from: $CODE_URL"

# Download the code using curl
curl -s "$CODE_URL" > "$CODE_FILE"
CURL_STATUS=$?

# Check if download was successful
if [ $CURL_STATUS -ne 0 ] || [ ! -s "$CODE_FILE" ]; then
    echo "[ERROR] Failed to download code from $CODE_URL" >&2
    debug_log "CURL exit code: $CURL_STATUS"
    debug_log "Code file size: $(stat -c%s "$CODE_FILE" 2>/dev/null || echo "unknown")"
    cleanup_and_exit 1
fi

debug_log "Code downloaded successfully ($(stat -c%s "$CODE_FILE") bytes)"

# Create new C# project
PROJECT_DIR="${TEMP_DIR}/csproj"
debug_log "Creating new C# project at: $PROJECT_DIR"

dotnet new console -o "$PROJECT_DIR" --force > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "[ERROR] Failed to initialize .NET project." >&2
    cleanup_and_exit 1
fi

# Replace default Program.cs with downloaded code
mv "$CODE_FILE" "$PROJECT_DIR/Program.cs"
if [ $? -ne 0 ]; then
    echo "[ERROR] Failed to move code file to project directory." >&2
    cleanup_and_exit 1
fi

debug_log "Program.cs moved into project directory"

# Build the project
debug_log "Building project..."
#BUILD_OUTPUT=$(dotnet build "$PROJECT_DIR" -c Release --verbosity quiet 2>&1)
debug_log "===== Build output ====="
dotnet build "$PROJECT_DIR" -c Release --verbosity minimal
BUILD_EXIT=$?

if [ $BUILD_EXIT -ne 0 ]; then
    echo "[ERROR] Build failed:" >&2
    echo "$BUILD_OUTPUT" >&2
    cleanup_and_exit 1
fi

debug_log "Build succeeded"

# Find the built DLL (handle different .NET versions)
APP_DLL=$(find "$PROJECT_DIR/bin/Release" -name "*.dll" | grep -v 'ref' | head -1)
#APP_DLL=$(find "$PROJECT_DIR/bin/Release" -name "csproj.dll" | head -1)
if [ -z "$APP_DLL" ] || [ ! -f "$APP_DLL" ]; then
    echo "[ERROR] Could not find compiled DLL." >&2
    debug_log "Searched in: $PROJECT_DIR/bin/Release"
    debug_log "Contents: $(ls -la "$PROJECT_DIR/bin/Release" 2>/dev/null || echo "directory not found")"
    cleanup_and_exit 1
fi

debug_log "Found compiled DLL: $APP_DLL"
debug_log "Executing compiled program..."

# Execute with a reasonable timeout (10 seconds)
timeout 10s dotnet "$APP_DLL" 2>&1
EXEC_STATUS=$?

# Handle timeout
if [ $EXEC_STATUS -eq 124 ]; then
    echo "[ERROR] Execution timed out after 15 seconds." >&2
    cleanup_and_exit 1
fi

debug_log "Execution completed with exit code $EXEC_STATUS"

# Clean up
cleanup_and_exit $EXEC_STATUS