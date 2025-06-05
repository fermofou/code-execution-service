#!/bin/bash
set -e

# Fetch code
curl -s "$CODE_URL" -o Program.cs

# Compile
mcs Program.cs

# Warm up JIT
mono --version > /dev/null

# Logic:
if [ -n "$SINGLE" ]; then
    # SINGLE is set → run without input
    timeout ${TIMEOUT:-8}s mono Program.exe
else
    # SINGLE is not set → run with stdin input
    timeout ${TIMEOUT:-8}s mono Program.exe < /dev/stdin
fi
