#!/bin/bash

# LeetCode Clone Code Execution Service Status Script
# This script checks the status of all services

set -e

echo "Checking service status..."
echo "=========================="
echo ""

# Check Docker service
echo "Docker Service:"
if systemctl is-active docker &> /dev/null; then
    echo "✅ Docker is running"
else
    echo "❌ Docker is not running"
    echo "   Try starting it with: systemctl start docker"
fi
echo ""

# Check if Docker Compose services are running
echo "Docker Compose Services:"
if docker-compose ps &> /dev/null; then
    # Get service status
    SERVICES=$(docker-compose ps --services)
    
    for SERVICE in $SERVICES; do
        STATUS=$(docker-compose ps $SERVICE | grep $SERVICE)
        if [[ $STATUS == *"Up"* ]]; then
            echo "✅ $SERVICE is running"
        else
            echo "❌ $SERVICE is not running"
        fi
    done
else
    echo "❌ Docker Compose services are not running"
    echo "   Try starting them with: docker-compose up -d"
fi
echo ""

# Check API health
echo "API Health Check:"
if curl -s http://localhost:8080/health &> /dev/null; then
    HEALTH=$(curl -s http://localhost:8080/health)
    if [[ $HEALTH == *"ok"* ]]; then
        echo "✅ API is healthy"
    else
        echo "⚠️ API returned unexpected response: $HEALTH"
    fi
else
    echo "❌ API health check failed"
    echo "   API is not responding at http://localhost:8080/health"
fi
echo ""

# Check Redis
echo "Redis Status:"
if docker-compose exec -T redis redis-cli ping &> /dev/null; then
    PING=$(docker-compose exec -T redis redis-cli ping)
    if [[ $PING == "PONG" ]]; then
        echo "✅ Redis is responding"
    else
        echo "⚠️ Redis returned unexpected response: $PING"
    fi
else
    echo "❌ Redis is not responding"
fi
echo ""

# Check executor images
echo "Executor Images:"
for LANG in python javascript cpp csharp; do
    if docker images | grep -q "${LANG}-executor"; then
        echo "✅ ${LANG}-executor image is available"
    else
        echo "❌ ${LANG}-executor image is missing"
    fi
done
echo ""

echo "============================"
echo "Status check completed."
echo ""
echo "For detailed logs, run:"
echo "  docker-compose logs api    # API service logs"
echo "  docker-compose logs worker # Worker service logs"
echo "" 