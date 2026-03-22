#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'
PASS=0
FAIL=0

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
API_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
cd "$API_DIR"

API_BASE_URL=${API_BASE_URL:-http://localhost:8080}
ENGINE_BASE_URL=${ENGINE_BASE_URL:-http://localhost:8081}
WEEK2_VERIFY_MODE=${WEEK2_VERIFY_MODE:-auto}
export GOCACHE=${GOCACHE:-/tmp/ttt-go-build}
export GOTOOLCHAIN=${GOTOOLCHAIN:-go1.25.8+auto}

mkdir -p "$GOCACHE"

pass() {
    echo -e "${GREEN}PASS${NC} $1"
    PASS=$((PASS + 1))
}

fail() {
    echo -e "${RED}FAIL${NC} $1"
    if [ -n "${2:-}" ]; then
        echo "$2"
    fi
    FAIL=$((FAIL + 1))
}

check_cmd() {
    local label="$1"
    local cmd="$2"
    local output

    output=$(eval "$cmd" 2>&1) && {
        pass "$label"
        return
    }

    fail "$label" "$output"
}

check_contains() {
    local label="$1"
    local value="$2"
    local pattern="$3"

    if printf '%s' "$value" | grep -q "$pattern"; then
        pass "$label"
        return
    fi

    fail "$label" "$value"
}

json_get() {
    local path="$1"

    python3 - "$path" <<'PY'
import json
import sys

path = sys.argv[1].split(".")
value = json.load(sys.stdin)
for part in path:
    value = value[part]
print(value)
PY
}

live_ready() {
    curl -sf "$API_BASE_URL/health" >/dev/null 2>&1 &&
        curl -sf "$ENGINE_BASE_URL/internal/v1/pool/status" >/dev/null 2>&1
}

run_live_verification() {
    local run_id seller_a_phone seller_b_phone
    local register_a_resp register_b_resp login_a_resp login_b_resp
    local seller_a_token seller_b_token
    local add_a_resp add_b_resp account_a_id account_b_id
    local list_resp detail_resp update_resp usage_resp revoke_resp earnings_resp forbidden_resp pool_resp

    echo -e "\n${YELLOW}=== Week 2 Verify (live HTTP mode) ===${NC}\n"

    run_id=$(date +%s)
    seller_a_phone=$(printf "139%08d" $((run_id % 100000000)))
    seller_b_phone=$(printf "138%08d" $(((run_id + 1) % 100000000)))

    pool_resp=$(curl -s "$ENGINE_BASE_URL/internal/v1/pool/status")
    check_contains "Engine pool status is reachable" "$pool_resp" '"code":0'

    register_a_resp=$(curl -s -X POST "$API_BASE_URL/api/v1/seller/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$seller_a_phone\",\"code\":\"123456\"}")
    register_b_resp=$(curl -s -X POST "$API_BASE_URL/api/v1/seller/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$seller_b_phone\",\"code\":\"123456\"}")

    check_contains "Seller A register returns token" "$register_a_resp" '"token"'
    check_contains "Seller B register returns token" "$register_b_resp" '"token"'

    login_a_resp=$(curl -s -X POST "$API_BASE_URL/api/v1/seller/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$seller_a_phone\",\"code\":\"123456\"}")
    login_b_resp=$(curl -s -X POST "$API_BASE_URL/api/v1/seller/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$seller_b_phone\",\"code\":\"123456\"}")

    seller_a_token=$(printf '%s' "$login_a_resp" | json_get data.token)
    seller_b_token=$(printf '%s' "$login_b_resp" | json_get data.token)
    check_contains "Seller A login returns token" "$login_a_resp" '"token"'
    check_contains "Seller B login returns token" "$login_b_resp" '"token"'

    add_a_resp=$(curl -s -X POST "$API_BASE_URL/api/v1/seller/accounts" \
        -H "Authorization: Bearer $seller_a_token" \
        -H "Content-Type: application/json" \
        -d '{"vendor":"anthropic","api_key":"sk-ant-test","authorized_credits_usd":100,"expected_rate":0.75,"expire_at":"2026-09-01T00:00:00Z"}')
    add_b_resp=$(curl -s -X POST "$API_BASE_URL/api/v1/seller/accounts" \
        -H "Authorization: Bearer $seller_b_token" \
        -H "Content-Type: application/json" \
        -d '{"vendor":"anthropic","api_key":"sk-ant-test","authorized_credits_usd":80,"expected_rate":0.75,"expire_at":"2026-09-01T00:00:00Z"}')

    check_contains "Seller A add account succeeds" "$add_a_resp" '"status":"pending_verify"'
    check_contains "Seller B add account succeeds" "$add_b_resp" '"status":"pending_verify"'

    account_a_id=$(printf '%s' "$add_a_resp" | json_get data.account_id)
    account_b_id=$(printf '%s' "$add_b_resp" | json_get data.account_id)

    list_resp=$(curl -s -H "Authorization: Bearer $seller_a_token" "$API_BASE_URL/api/v1/seller/accounts")
    detail_resp=$(curl -s -H "Authorization: Bearer $seller_a_token" "$API_BASE_URL/api/v1/seller/accounts/$account_a_id")
    update_resp=$(curl -s -X PATCH "$API_BASE_URL/api/v1/seller/accounts/$account_a_id/authorization" \
        -H "Authorization: Bearer $seller_a_token" \
        -H "Content-Type: application/json" \
        -d '{"authorized_credits_usd":150}')
    usage_resp=$(curl -s -H "Authorization: Bearer $seller_a_token" "$API_BASE_URL/api/v1/seller/accounts/$account_a_id/usage")
    revoke_resp=$(curl -s -X DELETE -H "Authorization: Bearer $seller_a_token" "$API_BASE_URL/api/v1/seller/accounts/$account_a_id/authorization")
    earnings_resp=$(curl -s -H "Authorization: Bearer $seller_a_token" "$API_BASE_URL/api/v1/seller/earnings")
    forbidden_resp=$(curl -s -H "Authorization: Bearer $seller_a_token" "$API_BASE_URL/api/v1/seller/accounts/$account_b_id")

    check_contains "Seller accounts list succeeds" "$list_resp" '"code":0'
    check_contains "Seller account detail succeeds" "$detail_resp" "\"id\":\"$account_a_id\""
    check_contains "Update authorization succeeds" "$update_resp" '"authorized_credits_usd":150'
    check_contains "Account usage succeeds" "$usage_resp" '"health_score"'
    check_contains "Revoke authorization succeeds" "$revoke_resp" '"revoked_amount_usd"'
    check_contains "Seller earnings succeeds" "$earnings_resp" '"pending_usd"'
    check_contains "Other seller account returns forbidden" "$forbidden_resp" '"code":1003'
}

run_local_verification() {
    echo -e "\n${YELLOW}=== Week 2 Verify (local test mode) ===${NC}\n"
    echo "live api/engine not reachable; falling back to build + httptest verification"

    check_cmd "Go build" "go build ./..."
    check_cmd "Go test all packages" "go test ./..."
    check_cmd "Seller Week 2 flow test" "go test ./internal/seller -run TestSellerWeek2Flow -v"
    check_cmd "Seller forbidden ownership test" "go test ./internal/seller -run TestSellerAccountForbiddenForOtherSeller -v"
}

if [ "$WEEK2_VERIFY_MODE" = "live" ]; then
    run_live_verification
elif [ "$WEEK2_VERIFY_MODE" = "test" ]; then
    run_local_verification
elif live_ready; then
    run_live_verification
else
    run_local_verification
fi

echo -e "\n${YELLOW}Result: passed $PASS, failed $FAIL${NC}"
[ $FAIL -eq 0 ] && echo -e "${GREEN}Week 2 verification passed!${NC}" || exit 1
