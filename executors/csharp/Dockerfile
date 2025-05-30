# Use a lightweight base image with Mono
FROM debian:bookworm-slim

# Set working directory
WORKDIR /app

# Install dependencies and Mono from official repo
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl \
    gnupg \
    ca-certificates \
    apt-transport-https && \
    curl -sSL https://download.mono-project.com/repo/xamarin.gpg -o /etc/apt/trusted.gpg.d/mono.gpg && \
    echo "deb https://download.mono-project.com/repo/debian stable-bookworm main" > /etc/apt/sources.list.d/mono-official-stable.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    mono-mcs \
    mono-runtime && \
    apt-get clean && rm -rf /var/lib/apt/lists/*


# Copy the executor script
COPY execute.sh /app/

# Make it executable
RUN chmod +x /app/execute.sh

# Run the executor when the container starts
ENTRYPOINT ["/app/execute.sh"]
