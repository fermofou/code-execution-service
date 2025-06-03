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
        return result.stdout.strip(), result.stderr.strip(), result.returncode
    except subprocess.TimeoutExpired:
        return "", "Execution timed out.", 1
    except Exception as e:
        return "", str(e), 1

if __name__ == "__main__":
    code_url = os.environ.get('CODE_URL')
    dir_txt = os.environ.get('DIRTXT')  # /app/testdata

    if not code_url:
        print("Missing CODE_URL", file=sys.stderr)
        sys.exit(1)

    # Optional: input.txt path
    input_data = ""
    if dir_txt:
        input_path = os.path.join(dir_txt, 'input.txt')
        if os.path.exists(input_path):
            with open(input_path, 'r') as f:
                input_data = f.read()

    try:
        # Download the code
        response = requests.get(code_url)
        if response.status_code != 200:
            print(f"Failed to download code. Status code: {response.status_code}", file=sys.stderr)
            sys.exit(1)

        code = response.text

        # Save to temp file
        with tempfile.NamedTemporaryFile(suffix='.py', delete=False) as tmp:
            tmp.write(code.encode('utf-8'))
            code_file = tmp.name

        # Run the code with input
        stdout, stderr, returncode = run_code(code_file, input_data)

        # Clean up
        try:
            os.unlink(code_file)
        except OSError:
            pass

        # Print only result for Go worker comparison
        if returncode == 0 and not stderr:
            print(stdout)
        else:
            # Send both stdout and stderr in case of error
            print(f"{stdout}\n{stderr}".strip())
            sys.exit(returncode or 1)

    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        sys.exit(1)
