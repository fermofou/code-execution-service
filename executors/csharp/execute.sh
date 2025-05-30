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

echo "Restoring and building project..."

# Restore and build
cd "${TEMP_DIR}/csproj"
dotnet build -c Release > /dev/null 2> "${TEMP_DIR}/build_stderr"

if [ $? -ne 0 ]; then
    echo "STDERR:"
    cat "${TEMP_DIR}/build_stderr"
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "Running program..."

# Execute the program with timeout
timeout 5s dotnet run -c Release > "${TEMP_DIR}/stdout" 2> "${TEMP_DIR}/stderr"

if [ $? -eq 124 ]; then
    echo "STDERR:"
    echo "Execution timed out."
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "STDOUT:"
cat "${TEMP_DIR}/stdout"
echo "STDERR:"
cat "${TEMP_DIR}/stderr"

# Clean up
rm -rf "$TEMP_DIR"
