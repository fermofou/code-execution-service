FROM node:14-slim

WORKDIR /executor

# Copy your executor script into the container
COPY executor.js /executor/executor.js

# Set execute permissions for the executor script
RUN chmod +x /executor/executor.js

# Create the /code directory where the mounted volume will be
RUN mkdir -p /code && chmod 777 /code

# Keep the container alive for test execution via docker exec
ENTRYPOINT ["sh", "-c", "while true; do sleep 300; done"]

