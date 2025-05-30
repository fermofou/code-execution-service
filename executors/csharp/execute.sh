#!/bin/bash
set -e

# Fetch code from provided URL
curl -s "$CODE_URL" -o Program.cs

# Compile with Mono C# compiler (mcs) – lightweight
#echo "[DEBUG] Compiling C# code with mcs..."
mcs Program.cs

# Optional: Pre-run Mono once to warm up JIT (reduces timeout risk)
mono --version > /dev/null

# Execute the compiled binary with a timeout (default 8s, configurable)
#echo "[DEBUG] Starting execution..."
timeout ${TIMEOUT:-8}s mono Program.exe
