FROM node:14-slim

WORKDIR /executor

# Copy your executor script into the container
COPY executor.js /executor/executor.js

# Set execute permissions for the executor script
RUN chmod +x /executor/executor.js

# Create the /code directory where the mounted volume will be
RUN mkdir -p /code && chmod 777 /code

# Execute the script; the script should accept a file path to execute the submitted code
CMD ["node", "executor.js"]
