#!/bin/bash

# Check if CODE_URL is provided
if [ -z "$CODE_URL" ]; then
    echo "STDERR:"
    echo "Error: CODE_URL environment variable not set."
    exit 1
fi

# Create a temporary directory
TEMP_DIR=$(mktemp -d)
CODE_FILE="${TEMP_DIR}/Program.cs"

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

echo "Creating new C# project..."

# Initialize new .NET console project
dotnet new console -o "${TEMP_DIR}/csproj" --force > /dev/null 2>&1

# Replace default Program.cs with downloaded code
mv "$CODE_FILE" "${TEMP_DIR}/csproj/Program.cs"

echo "Building project..."

# Build project (not run!)
dotnet build "${TEMP_DIR}/csproj" -c Release > /dev/null 2> "${TEMP_DIR}/build_stderr"
if [ $? -ne 0 ]; then
    echo "STDERR:"
    cat "${TEMP_DIR}/build_stderr"
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "Executing compiled program..."

# Run the compiled .dll directly (C++-like behavior)
APP_DLL="${TEMP_DIR}/csproj/bin/Release/net7.0/csproj.dll"
timeout 5s dotnet "$APP_DLL" > "${TEMP_DIR}/stdout" 2> "${TEMP_DIR}/stderr"

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
