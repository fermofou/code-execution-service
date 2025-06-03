#!/usr/bin/env python3
import os, sys, subprocess, tempfile, requests

def run_code(code_file, stdin_input):
    result = subprocess.run(
        ["python3", code_file],
        input=stdin_input,
        capture_output=True,
        text=True,
        timeout=5,
    )
    return result.stdout.strip(), result.stderr.strip(), result.returncode

if __name__ == "__main__":
    code_url = os.environ.get("CODE_URL")
    dir_txt = os.environ.get("DIRTXT", "/code")  # default to /code
    input_data = ""
    if not code_url:
        print("Error: CODE_URL not set", file=sys.stderr)
        sys.exit(1)

    # Read from /code/input.txt if it exists
    input_path = os.path.join(dir_txt, "input.txt")
    if os.path.exists(input_path):
        with open(input_path, "r") as f:
            input_data = f.read()

    # Fetch user‐submitted code
    r = requests.get(code_url)
    if r.status_code != 200:
        print(f"Failed to download code: {r.status_code}", file=sys.stderr)
        sys.exit(1)

    code = r.text
    with tempfile.NamedTemporaryFile(suffix=".py", delete=False) as tmp:
        tmp.write(code.encode("utf-8"))
        code_file = tmp.name

    stdout, stderr, retcode = run_code(code_file, input_data)
    try:
        os.unlink(code_file)
    except:
        pass

    if retcode == 0 and not stderr:
        print(stdout)
    else:
        # Print both stdout and stderr on error
        print((stdout + "\n" + stderr).strip())
        sys.exit(retcode or 1)
