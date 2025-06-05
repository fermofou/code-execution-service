#!/usr/bin/env node

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const http = require("http");
const https = require("https");
const os = require("os");

function runCode(codeFile, input = null) {
  try {
    const options = {
      timeout: 5000,
      encoding: "utf-8",
      input: input || undefined,
    };
    const output = execSync(`node ${codeFile}`, options);
    return { stdout: output, stderr: "" };
  } catch (error) {
    if (error.killed || error.signal === "SIGTERM") {
      return { stdout: "", stderr: "Execution timed out." };
    }
    return { stdout: "", stderr: `Error: ${error.message}` };
  }
}

function downloadCode(url) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith("https") ? https : http;

    client
      .get(url, (res) => {
        if (res.statusCode !== 200) {
          reject(
            new Error(`Failed to download code. Status code: ${res.statusCode}`)
          );
          return;
        }

        let data = "";
        res.on("data", (chunk) => (data += chunk));
        res.on("end", () => resolve(data));
      })
      .on("error", reject);
  });
}

function readStdin() {
  return new Promise((resolve, reject) => {
    let input = "";
    process.stdin.setEncoding("utf-8");
    process.stdin.on("data", (chunk) => (input += chunk));
    process.stdin.on("end", () => resolve(input));
    process.stdin.on("error", reject);
  });
}

async function main() {
  try {
    const codeUrl = process.env.CODE_URL;
    const singleMode = process.env.SINGLE;

    if (!codeUrl) {
      //console.log("STDERR:");
      console.log("Error: CODE_URL environment variable not set.");
      process.exit(1);
    }

    const code = await downloadCode(codeUrl);
    const tempFilePath = path.join(os.tmpdir(), `code-${Date.now()}.js`);
    fs.writeFileSync(tempFilePath, code);

    let input = null;
    if (!singleMode) {
      // Wait for piped stdin if SINGLE is not set
      input = await readStdin();
    }

    const { stdout, stderr } = runCode(tempFilePath, input);

    try {
      fs.unlinkSync(tempFilePath);
    } catch {}

    //console.log("STDOUT:");
    console.log(stdout);
    //console.log("STDERR:");
    //console.log(stderr);
  } catch (error) {
    console.log("STDERR:");
    console.log(`Error: ${error.message}`);
    process.exit(1);
  }
}

main();
