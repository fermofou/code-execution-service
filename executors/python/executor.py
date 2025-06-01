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
    inputs = os.environ.get('CODE_INPUTS', '')
    expected_outputs = os.environ.get('CODE_OUTPUTS', '')

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

        with tempfile.NamedTemporaryFile(suffix='.py', delete=False) as temp_file:
            temp_file.write(code.encode('utf-8'))
            code_file = temp_file.name

        # Prepare stdin input (multiple lines)
        stdin_input = inputs.replace('|', '\n') if inputs else ""

        # Run the code
        stdout, stderr = run_code(code_file, stdin_input)

        # Clean up temp file
        try:
            os.unlink(code_file)
        except:
            pass

        print("STDOUT:")
        print(stdout)
        print("STDERR:")
        print(stderr)

        # Optional output validation
        if expected_outputs:
            expected_list = expected_outputs.strip().split('|')
            actual_list = stdout.strip().splitlines()

            if actual_list == expected_list:
                match = "true"
                print("OUTPUT_MATCH:")
                print(match)
            else:
                match = "false"
                print("OUTPUT_MATCH:")
                print(match)
                print("Expected output:")
                for line in expected_list:
                    print(line)
                print("Your output:")
                for line in actual_list:
                    print(line)

            
    except Exception as e:
        print("STDERR:")
        print(f"Error: {str(e)}")
        sys.exit(1)