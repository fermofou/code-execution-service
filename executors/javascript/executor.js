#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const http = require('http');
const https = require('https');
const os = require('os');

function runCode(codeFile) {
    try {
        // Run the submitted code with a timeout of 5 seconds
        const output = execSync(`node ${codeFile}`, { 
            timeout: 5000,
            encoding: 'utf-8'
        });
        return { stdout: output, stderr: '' };
    } catch (error) {
        if (error.code === 'ETIMEDOUT') {
            return { stdout: '', stderr: 'Execution timed out.' };
        }
        return { stdout: '', stderr: `Error: ${error.message}` };
    }
}

// Function to download code via HTTP
function downloadCode(url) {
    return new Promise((resolve, reject) => {
        const client = url.startsWith('https') ? https : http;
        
        client.get(url, (res) => {
            if (res.statusCode !== 200) {
                reject(new Error(`Failed to download code. Status code: ${res.statusCode}`));
                return;
            }
            
            let data = '';
            res.on('data', (chunk) => {
                data += chunk;
            });
            
            res.on('end', () => {
                resolve(data);
            });
        }).on('error', (err) => {
            reject(err);
        });
    });
}

// Main execution
async function main() {
    try {
        // Get code URL from environment variable
        const codeUrl = process.env.CODE_URL;
        if (!codeUrl) {
            console.log('STDERR:');
            console.log('Error: CODE_URL environment variable not set.');
            process.exit(1);
        }
        
        console.log(`Fetching code from: ${codeUrl}`);
        
        // Download the code
        const code = await downloadCode(codeUrl);
        
        // Create a temporary file
        const tempFilePath = path.join(os.tmpdir(), `code-${Date.now()}.js`);
        fs.writeFileSync(tempFilePath, code);
        
        console.log(`Executing file: ${tempFilePath}`);
        const { stdout, stderr } = runCode(tempFilePath);
        
        // Clean up
        try {
            fs.unlinkSync(tempFilePath);
        } catch (error) {
            // Ignore cleanup errors
        }
        
        console.log('STDOUT:');
        console.log(stdout);
        console.log('STDERR:');
        console.log(stderr);
        
    } catch (error) {
        console.log('STDERR:');
        console.log(`Error: ${error.message}`);
        process.exit(1);
    }
}

main();
