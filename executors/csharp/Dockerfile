FROM mcr.microsoft.com/dotnet/sdk:7.0

WORKDIR /executor

# Install dependencies
RUN apt-get update && apt-get install -y curl

# Copy the executor script
COPY executor.sh /executor/executor.sh

# Make it executable
RUN chmod +x /executor/executor.sh

# Create shared code mount dir
RUN mkdir -p /code && chmod 777 /code

CMD ["/executor/executor.sh"]
