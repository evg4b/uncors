#!/bin/bash

# Test script for Lua handler examples
# Make sure UNCORS is running first: uncors

echo "🧪 Testing Lua Script Handler Examples"
echo "======================================"
echo ""

BASE_URL="http://localhost:3000"

# Color codes
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local extra_args=$4

    echo -e "${BLUE}Testing: ${name}${NC}"
    echo "Endpoint: ${method} ${endpoint}"

    if [ "$method" == "GET" ]; then
        curl -s -w "\nStatus: %{http_code}\n" ${extra_args} "${BASE_URL}${endpoint}"
    else
        curl -s -w "\nStatus: %{http_code}\n" -X ${method} ${extra_args} "${BASE_URL}${endpoint}"
    fi

    echo ""
    echo "---"
    echo ""
}

# Test 1: Hello World
test_endpoint "Hello World" "GET" "/api/hello"

# Test 2: Echo Service
test_endpoint "Echo Service" "POST" "/api/echo" \
    '-H "Content-Type: application/json" -d "{\"name\": \"Alice\", \"age\": 30}"'

# Test 3: Calculator - Random
test_endpoint "Calculator (Random)" "GET" "/api/calculate?operation=random&min=1&max=100"

# Test 4: Calculator - Square Root
test_endpoint "Calculator (Square Root)" "GET" "/api/calculate?operation=sqrt&value=16"

# Test 5: Calculator - Power
test_endpoint "Calculator (Power)" "GET" "/api/calculate?operation=power&base=2&exponent=8"

# Test 6: User API
test_endpoint "User API" "GET" "/api/users/123"

# Test 7: Protected endpoint without auth
test_endpoint "Protected (No Auth)" "GET" "/api/protected"

# Test 8: Protected endpoint with auth
test_endpoint "Protected (With Auth)" "GET" "/api/protected" \
    '-H "Authorization: Bearer secret-token"'

# Test 9: Complex file-based script
test_endpoint "Complex Script" "GET" "/api/complex"

# Test 10: Data processor
test_endpoint "Data Processor" "POST" "/api/process" \
    '-H "Content-Type: text/plain" -d "Hello World from Lua Script Handler"'

# Test 11: Health check
test_endpoint "Health Check" "GET" "/health"

echo -e "${GREEN}✓ All tests completed!${NC}"
echo ""
echo "You can also test with:"
echo "  - Browser: open http://localhost:3000/api/hello"
echo "  - Postman/Insomnia: Import the endpoints"
echo "  - curl: curl -v http://localhost:3000/api/hello"
