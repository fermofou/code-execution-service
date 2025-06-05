# Use a lightweight base image with Mono
FROM debian:bullseye-slim

# Set working directory
WORKDIR /app

# Install curl and mono compiler/runtime
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl \
    mono-mcs \
    mono-runtime && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy the executor script
COPY execute.sh /app/

# Make it executable
RUN chmod +x /app/execute.sh

# Run the executor when the container starts
#ENTRYPOINT ["/app/execute.sh"]
# Keep the container alive indefinitely for docker exec (same as Python)
ENTRYPOINT ["sh", "-c", "while true; do sleep 300; done"]
