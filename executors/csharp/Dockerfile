FROM mcr.microsoft.com/dotnet/sdk:7.0

WORKDIR /app

# Install dependencies
RUN apt-get update && apt-get install -y curl

# Copy the executor script
COPY execute.sh /app/execute.sh

# Make it executable
RUN chmod +x /app/execute.sh

# Create shared code mount dir
RUN mkdir -p /code && chmod 777 /code

CMD ["/app/execute.sh"]
