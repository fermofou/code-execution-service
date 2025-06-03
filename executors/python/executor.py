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
    is_single_run = os.environ.get("SINGLE") is not None
    input_data = ""
    
    if not code_url:
        print("Error: CODE_URL not set", file=sys.stderr)
        sys.exit(1)

    if is_single_run:
        # Single run: read from stdin if available
        if not sys.stdin.isatty():
            input_data = sys.stdin.read()
    else:
        # Test run: always read from stdin (piped via docker exec -i)
        input_data = sys.stdin.read()

    # Fetch user-submitted code
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
        print((stdout + "\n" + stderr).strip())
        sys.exit(retcode or 1)