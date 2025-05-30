# Use the official Mono image for running C# code
FROM mono:latest

# Set the working directory inside the container
WORKDIR /app

# Install curl for downloading code
RUN apt-get update && apt-get install -y curl

# Copy the executor script
COPY execute.sh /app/

# Make the executor script executable
RUN chmod +x /app/execute.sh

# Run the executor when the container starts
ENTRYPOINT ["/app/execute.sh"]
