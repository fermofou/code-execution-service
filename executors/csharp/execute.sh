#!/bin/bash

echo "[DEBUG] Starting C# execution script..."

# Check if CODE_URL is provided
if [ -z "$CODE_URL" ]; then
    echo "STDERR:"
    echo "[ERROR] CODE_URL environment variable not set."
    exit 1
fi

# Create a temporary directory
TEMP_DIR=$(mktemp -d)
CODE_FILE="${TEMP_DIR}/Program.cs"

echo "[DEBUG] Temporary directory created at: $TEMP_DIR"
echo "[DEBUG] Code will be saved to: $CODE_FILE"
echo "[DEBUG] Fetching code from: $CODE_URL"

# Download the code using curl
curl -s "$CODE_URL" > "$CODE_FILE"
CURL_STATUS=$?

# Check if download was successful
if [ $CURL_STATUS -ne 0 ] || [ ! -s "$CODE_FILE" ]; then
    echo "STDERR:"
    echo "[ERROR] Failed to download code from $CODE_URL"
    echo "[DEBUG] CURL exit code: $CURL_STATUS"
    echo "[DEBUG] Code file size: $(stat -c%s "$CODE_FILE" 2>/dev/null)"
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "[DEBUG] Code downloaded successfully."

echo "[DEBUG] Creating new C# project..."
dotnet new console -o "${TEMP_DIR}/csproj" --force > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "STDERR:"
    echo "[ERROR] Failed to initialize .NET project."
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Replace default Program.cs with downloaded code
mv "$CODE_FILE" "${TEMP_DIR}/csproj/Program.cs"
echo "[DEBUG] Program.cs moved into project directory."

echo "[DEBUG] Building project..."
dotnet build "${TEMP_DIR}/csproj" -c Release > /dev/null 2> "${TEMP_DIR}/build_stderr"
BUILD_EXIT=$?

if [ $BUILD_EXIT -ne 0 ]; then
    echo "STDERR:"
    echo "[ERROR] Build failed with exit code $BUILD_EXIT"
    cat "${TEMP_DIR}/build_stderr"
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "[DEBUG] Build succeeded."

echo "[DEBUG] Executing compiled program..."
APP_DLL="${TEMP_DIR}/csproj/bin/Release/net7.0/csproj.dll"

timeout 5s dotnet "$APP_DLL" > "${TEMP_DIR}/stdout" 2> "${TEMP_DIR}/stderr"
EXEC_STATUS=$?

# Check if execution timed out
if [ $EXEC_STATUS -eq 124 ]; then
    echo "STDERR:"
    echo "[ERROR] Execution timed out after 5 seconds."
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "[DEBUG] Execution completed with exit code $EXEC_STATUS"

# Output results
echo "STDOUT:"
cat "${TEMP_DIR}/stdout"
echo "STDERR:"
cat "${TEMP_DIR}/stderr"

# Clean up
echo "[DEBUG] Cleaning up temporary directory..."
rm -rf "$TEMP_DIR"
echo "[DEBUG] Script complete."
