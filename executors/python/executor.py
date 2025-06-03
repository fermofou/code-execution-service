#!/usr/bin/env python3

import subprocess
import sys
import os
import requests
import tempfile

def run_code(code_file, stdin_input):
    try:
        result = subprocess.run(
            ['python', code_file],
            input=stdin_input,
            capture_output=True,
            text=True,
            timeout=5
        )
        return result.stdout.strip(), result.stderr.strip()
    except subprocess.TimeoutExpired:
        return "", "Execution timed out."
    except Exception as e:
        return "", f"Error: {str(e)}"

if __name__ == "__main__":
    code_url = os.environ.get('CODE_URL')
    dir_txt = os.environ.get('DIRTXT')  # e.g. "/app/testdata"
    input_data = ""

    if not code_url:
        print("STDERR:")
        print("Error: CODE_URL environment variable not set.")
        sys.exit(1)

    # If DIRTXT is set, read /app/testdata/input.txt as stdin
    if dir_txt:
        input_path = os.path.join(dir_txt, 'input.txt')
        try:
            with open(input_path, 'r') as f:
                input_data = f.read()
        except FileNotFoundError:
            # If input.txt is missing, treat as empty stdin
            input_data = ""

    try:
        # Fetch the user code
        response = requests.get(code_url)
        if response.status_code != 200:
            print("STDERR:")
            print(f"Error: Failed to download code. Status code: {response.status_code}")
            sys.exit(1)

        code = response.text

        # Write the code to a temp file
        with tempfile.NamedTemporaryFile(suffix='.py', delete=False) as temp_file:
            temp_file.write(code.encode('utf-8'))
            code_file = temp_file.name

        # Run the user code once, feeding it input_data
        stdout, stderr = run_code(code_file, input_data)

        # Remove the temp file
        try:
            os.unlink(code_file)
        except OSError:
            pass

        # Print results
        print("STDOUT:")
        print(stdout)
        print("STDERR:")
        print(stderr)

    except Exception as e:
        print("STDERR:")
        print(f"Error: {str(e)}")
        sys.exit(1)
