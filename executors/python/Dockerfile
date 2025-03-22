FROM python:3.9-slim

WORKDIR /app

# Install requests library for HTTP requests
RUN pip install requests

# Copy the executor script
COPY executor.py /app/

# Make it executable
RUN chmod +x /app/executor.py

# Run the executor when the container starts
ENTRYPOINT ["/app/executor.py"]
