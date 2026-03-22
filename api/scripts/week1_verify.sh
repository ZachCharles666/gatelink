#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'
PASS=0
FAIL=0
export GOCACHE=${GOCACHE:-/tmp/ttt-go-build}
export GOTOOLCHAIN=${GOTOOLCHAIN:-go1.25.8+auto}
API_PID=""

mkdir -p "$GOCACHE"

cleanup() {
    if [ -n "$API_PID" ]; then
        kill "$API_PID" 2>/dev/null || true
        wait "$API_PID" 2>/dev/null || true
    fi
}

trap cleanup EXIT

check() {
    if eval "$2" | grep -q "$3"; then
        echo -e "${GREEN}PASS${NC} $1"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} $1"
        FAIL=$((FAIL + 1))
    fi
}

echo -e "\n${YELLOW}=== Week 1 Verify ===${NC}\n"

go run ./cmd/api &
API_PID=$!
sleep 3

check "API health" \
    "curl -s http://localhost:8080/health" '"code":0'

check "Seller register returns token" \
    "curl -s -X POST http://localhost:8080/api/v1/seller/auth/register -H 'Content-Type: application/json' -d '{\"phone\":\"13900000001\",\"code\":\"123456\"}'" '"token"'

BUYER_RESP=$(curl -s -X POST http://localhost:8080/api/v1/buyer/auth/register \
    -H 'Content-Type: application/json' \
    -d '{"email":"test_w1@example.com","password":"pass123"}')

check "Buyer register returns api_key" "echo '$BUYER_RESP'" '"api_key"'

API_KEY=$(echo "$BUYER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['api_key'])")
check "api_key length is 64" "echo -n $API_KEY | wc -c | tr -d ' '" "64"

check "Seller endpoint returns 401 without token" \
    "curl -s http://localhost:8080/api/v1/seller/accounts" '"code":1002'

echo -e "\n${YELLOW}Result: passed $PASS, failed $FAIL${NC}"
[ $FAIL -eq 0 ] && echo -e "${GREEN}Week 1 verification passed!${NC}" || exit 1
