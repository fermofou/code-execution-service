#!/bin/bash
set -e

# Fetch code from provided URL
curl -s "$CODE_URL" -o Program.cs

# Compile with Mono C# compiler (mcs) – lightweight
#echo "[DEBUG] Compiling C# code with mcs..."
mcs Program.cs

# Optional: Pre-run Mono once to warm up JIT (reduces timeout risk)
mono --version > /dev/null

# Check if this is a single run or test run
if [ -n "$SINGLE" ]; then
    # Single run: check if stdin has data available
    if [ -t 0 ]; then
        # No input available, run without input
        echo "[DEBUG] Running single execution without input..."
        timeout ${TIMEOUT:-8}s mono Program.exe
    else
        # Input available, pipe it
        echo "[DEBUG] Running single execution with piped input..."
        timeout ${TIMEOUT:-8}s mono Program.exe < /dev/stdin
    fi
else
    # Test run: always read without stdin (piped via docker exec -i)
    echo "[DEBUG] Running test execution with piped input..."
    timeout ${TIMEOUT:-8}s mono Program.exe < /dev
fi


