FROM gcc:latest

WORKDIR /app

# Install curl for downloading code
RUN apt-get update && apt-get install -y curl

# Copy the executor script
COPY execute.sh /app/

# Make it executable
RUN chmod +x /app/execute.sh

# Create the /code directory where the mounted volume will be
RUN mkdir -p /code && chmod 777 /code

# Run the script when the container starts
# Keep the container alive indefinitely for docker exec (same as Python)
ENTRYPOINT ["sh", "-c", "while true; do sleep 300; done"]
