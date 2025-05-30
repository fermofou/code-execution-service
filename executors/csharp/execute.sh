#!/bin/bash
set -e

curl -s "$CODE_URL" -o Program.cs

# Compile directly with csc (no MSBuild involved)
csc Program.cs

# Run the output
timeout 5s ./Program.exe
