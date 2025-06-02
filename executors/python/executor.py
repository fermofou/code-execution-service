#!/usr/bin/env python3

import subprocess
import sys
import os
import requests
import tempfile

def run_code(code_file):
    try:
        # Run the submitted code with a timeout of 5 seconds
        result = subprocess.run(
            ['python', code_file],
            capture_output=True,
            text=True,
            timeout=5
        )
        return result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return "", "Execution timed out."
    except Exception as e:
        return "", f"Error: {str(e)}"

if __name__ == "__main__":
    # Get code via HTTP request
    code_url = os.environ.get('CODE_URL')
    if not code_url:
        print("STDERR:")
        print("Error: CODE_URL environment variable not set.")
        sys.exit(1)
    
    try:
        print(f"Fetching code from: {code_url}")
        response = requests.get(code_url)
        if response.status_code != 200:
            print("STDERR:")
            print(f"Error: Failed to download code. Status code: {response.status_code}")
            sys.exit(1)
            
        code = response.text
        
        # Create a temporary file for the code
        with tempfile.NamedTemporaryFile(suffix='.py', delete=False) as temp_file:
            temp_file.write(code.encode('utf-8'))
            code_file = temp_file.name
        
        print(f"Executing file: {code_file}")
        stdout, stderr = run_code(code_file)
        
        # Clean up the temporary file
        try:
            os.unlink(code_file)
        except:
            pass
            
        print("STDOUT:")
        print(stdout)
        print("STDERR:")
        print(stderr)
        
    except Exception as e:
        print("STDERR:")
        print(f"Error: {str(e)}")
        sys.exit(1)