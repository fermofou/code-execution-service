#!/bin/bash

# LeetCode Clone Code Execution Service Setup Script
# This script automates the setup process for the code execution service

set -e

echo "Setting up LeetCode Clone Code Execution Service..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker not found. Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
    echo "Docker installed successfully."
else
    echo "Docker is already installed."
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "Docker Compose not found. Installing Docker Compose..."
    curl -L "https://github.com/docker/compose/releases/download/v2.20.3/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    echo "Docker Compose installed successfully."
else
    echo "Docker Compose is already installed."
fi

# Build and start services
echo "Building and starting services..."
docker-compose build
docker-compose up -d

# Wait for services to start
echo "Waiting for services to start..."
sleep 10

# Check if services are running
echo "Checking service status..."
docker-compose ps

# Display API endpoint information
echo ""
echo "Setup completed successfully!"
echo ""
echo "API Endpoints:"
echo "- Submit code: POST http://localhost:8080/execute"
echo "- Get result: GET http://localhost:8080/result/{job_id}"
echo "- Health check: GET http://localhost:8080/health"
echo ""
echo "Run './test.sh' to verify the code execution service."
echo "" 