FROM python:3.9-slim

WORKDIR /app

# Install requests library for HTTP requests
RUN pip install requests

# Copy the executor script
COPY executor.py /app/

# Make it executable
RUN chmod +x /app/executor.py

# Keep the container alive indefinitely, waiting for docker exec
ENTRYPOINT ["sh", "-c", "while true; do sleep 300; done"]
