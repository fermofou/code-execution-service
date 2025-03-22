#!/usr/bin/env python3

import requests
import json
import time

API_URL = "http://localhost:8080"

def test_complex_python():
    print("Testing Complex Python Code Execution...")
    code = """
def fibonacci(n):
    if n <= 0:
        return 0
    elif n == 1:
        return 1
    else:
        return fibonacci(n-1) + fibonacci(n-2)

# Calculate the 10th Fibonacci number
result = fibonacci(10)
print(f"The 10th Fibonacci number is: {result}")

# Test a list comprehension
squares = [x**2 for x in range(10)]
print(f"Squares of numbers 0-9: {squares}")

# Test exception handling
try:
    result = 10 / 0
except ZeroDivisionError:
    print("Caught a zero division error!")
"""
    
    response = requests.post(
        f"{API_URL}/execute",
        json={
            "language": "python",
            "code": code
        }
    )
    
    if response.status_code != 200:
        print(f"Error: {response.status_code}")
        print(response.text)
        return
    
    job_id = response.json().get("job_id")
    print(f"Python job ID: {job_id}")
    
    # Wait for job to complete
    time.sleep(10)
    
    # Get result
    result_response = requests.get(f"{API_URL}/result/{job_id}")
    
    if result_response.status_code != 200:
        print(f"Error getting result: {result_response.status_code}")
        print(result_response.text)
        return
    
    print("Result:")
    print(json.dumps(result_response.json(), indent=2))

def test_complex_javascript():
    print("\nTesting Complex JavaScript Code Execution...")
    code = """
// Recursive function to calculate factorial
function factorial(n) {
    if (n <= 1) return 1;
    return n * factorial(n-1);
}

// Calculate factorial of 5
const fact5 = factorial(5);
console.log(`Factorial of 5 is: ${fact5}`);

// Test array methods
const numbers = [1, 2, 3, 4, 5];
const doubled = numbers.map(n => n * 2);
console.log(`Doubled numbers: ${doubled}`);

// Test error handling
try {
    const result = null.property;
} catch (error) {
    console.log(`Caught an error: ${error.message}`);
}
"""
    
    response = requests.post(
        f"{API_URL}/execute",
        json={
            "language": "javascript",
            "code": code
        }
    )
    
    if response.status_code != 200:
        print(f"Error: {response.status_code}")
        print(response.text)
        return
    
    job_id = response.json().get("job_id")
    print(f"JavaScript job ID: {job_id}")
    
    # Wait for job to complete
    time.sleep(10)
    
    # Get result
    result_response = requests.get(f"{API_URL}/result/{job_id}")
    
    if result_response.status_code != 200:
        print(f"Error getting result: {result_response.status_code}")
        print(result_response.text)
        return
    
    print("Result:")
    print(json.dumps(result_response.json(), indent=2))

def test_complex_cpp():
    print("\nTesting Complex C++ Code Execution...")
    code = """
#include <iostream>
#include <vector>
#include <algorithm>
#include <stdexcept>

// Recursive function to calculate factorial
int factorial(int n) {
    if (n <= 1) return 1;
    return n * factorial(n-1);
}

int main() {
    // Calculate factorial of 5
    int fact5 = factorial(5);
    std::cout << "Factorial of 5 is: " << fact5 << std::endl;
    
    // Test vector operations
    std::vector<int> numbers = {1, 2, 3, 4, 5};
    std::vector<int> doubled;
    
    std::transform(numbers.begin(), numbers.end(), 
                  std::back_inserter(doubled),
                  [](int n) { return n * 2; });
    
    std::cout << "Doubled numbers: ";
    for (int n : doubled) {
        std::cout << n << " ";
    }
    std::cout << std::endl;
    
    // Test exception handling
    try {
        throw std::runtime_error("This is a test exception");
    } catch (const std::exception& e) {
        std::cout << "Caught an exception: " << e.what() << std::endl;
    }
    
    return 0;
}
"""
    
    response = requests.post(
        f"{API_URL}/execute",
        json={
            "language": "cpp",
            "code": code
        }
    )
    
    if response.status_code != 200:
        print(f"Error: {response.status_code}")
        print(response.text)
        return
    
    job_id = response.json().get("job_id")
    print(f"C++ job ID: {job_id}")
    
    # Wait for job to complete
    time.sleep(10)
    
    # Get result
    result_response = requests.get(f"{API_URL}/result/{job_id}")
    
    if result_response.status_code != 200:
        print(f"Error getting result: {result_response.status_code}")
        print(result_response.text)
        return
    
    print("Result:")
    print(json.dumps(result_response.json(), indent=2))

if __name__ == "__main__":
    test_complex_python()
    test_complex_javascript()
    test_complex_cpp() 