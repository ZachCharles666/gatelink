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
WEEK3_VERIFY_MODE=${WEEK3_VERIFY_MODE:-auto}
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

note() {
    echo -e "${YELLOW}NOTE${NC} $1"
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

    python3 -c '
import json
import sys

path = sys.argv[1].split(".")
value = json.load(sys.stdin)
for part in path:
    value = value[part]
print(value)
' "$path"
}

live_ready() {
    curl -sf "$API_BASE_URL/health" >/dev/null 2>&1 &&
        curl -sf "$ENGINE_BASE_URL/health" >/dev/null 2>&1
}

run_live_verification() {
    local run_id register_resp buyer_token buyer_api_key
    local balance_resp topup_resp topup_records_resp usage_resp models_resp unauthorized_resp proxy_resp
    local engine_dispatch_resp

    echo -e "\n${YELLOW}=== Week 3 Verify (live HTTP mode) ===${NC}\n"

    run_id=$(date +%s)
    register_resp=$(curl -s -X POST "$API_BASE_URL/api/v1/buyer/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"week3_${run_id}@test.com\",\"password\":\"pass123\"}")

    check_contains "Buyer register returns token" "$register_resp" '"token"'
    check_contains "Buyer register returns api_key" "$register_resp" '"api_key"'

    buyer_token=$(printf '%s' "$register_resp" | json_get data.token)
    buyer_api_key=$(printf '%s' "$register_resp" | json_get data.api_key)

    balance_resp=$(curl -s -H "Authorization: Bearer $buyer_token" "$API_BASE_URL/api/v1/buyer/balance")
    topup_resp=$(curl -s -X POST "$API_BASE_URL/api/v1/buyer/topup" \
        -H "Authorization: Bearer $buyer_token" \
        -H "Content-Type: application/json" \
        -d "{\"amount_usd\":100,\"tx_hash\":\"0xtest${run_id}\",\"network\":\"TRC20\"}")
    topup_records_resp=$(curl -s -H "Authorization: Bearer $buyer_token" "$API_BASE_URL/api/v1/buyer/topup/records")
    usage_resp=$(curl -s -H "Authorization: Bearer $buyer_token" "$API_BASE_URL/api/v1/buyer/usage")
    models_resp=$(curl -s -H "Authorization: Bearer $buyer_api_key" "$API_BASE_URL/v1/models")
    unauthorized_resp=$(curl -s -X POST "$API_BASE_URL/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -d '{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}]}')
    proxy_resp=$(curl -s -X POST "$API_BASE_URL/v1/chat/completions" \
        -H "Authorization: Bearer $buyer_api_key" \
        -H "Content-Type: application/json" \
        -d '{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}]}')
    engine_dispatch_resp=$(curl -s -X POST "$ENGINE_BASE_URL/internal/v1/dispatch" \
        -H "Content-Type: application/json" \
        -d '{"buyer_id":"week3-live-check","vendor":"anthropic","model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}],"stream":false}')

    check_contains "Buyer balance endpoint works" "$balance_resp" '"balance_usd"'
    check_contains "Buyer topup request works" "$topup_resp" '"topup_id"'
    check_contains "Buyer topup records endpoint works" "$topup_records_resp" '"records"'
    check_contains "Buyer usage endpoint works" "$usage_resp" '"total"'
    check_contains "Proxy models endpoint works" "$models_resp" '"object":"list"'
    check_contains "Proxy missing api_key returns 401" "$unauthorized_resp" '"code":1002'

    if printf '%s' "$proxy_resp" | grep -q '"code":1005'; then
        fail "Proxy live request reaches upstream" "$proxy_resp"
        note "Live blocker: buyer starts with zero balance and Week 3 has no approved topup path yet."
        if printf '%s' "$engine_dispatch_resp" | grep -q '"code":4001'; then
            note "Live blocker: direct engine dispatch currently returns no available account in pool."
        fi
        return
    fi

    if printf '%s' "$proxy_resp" | grep -q 'service_unavailable'; then
        fail "Proxy live request reaches upstream" "$proxy_resp"
        note "Live blocker: engine is running but no available account exists in the pool."
        return
    fi

    if printf '%s' "$proxy_resp" | grep -q '"id":"chatcmpl'; then
        pass "Proxy live request returns completion payload"
        return
    fi

    fail "Proxy live request returns recognized result" "$proxy_resp"
}

run_local_verification() {
    echo -e "\n${YELLOW}=== Week 3 Verify (local test mode) ===${NC}\n"
    echo "live api/engine not reachable; falling back to build + httptest verification"

    check_cmd "Go build" "go build ./..."
    check_cmd "Go test all packages" "go test ./..."
    check_cmd "Buyer Week 3 Day 1 flow test" "go test ./internal/buyer -run 'TestBuyerWeek3Day1Flow|TestBuyerDuplicateTopupRejected' -v"
    check_cmd "API Week 3 route integration tests" "go test ./internal/api -run 'TestWeek3Day4RouteFlowSuccess|TestWeek3Day4RouteFlowAuditBlocked|TestWeek3Day4RouteFlowNoAvailableAccount|TestSetupRoutesMountsProxyEndpointsWithAPIKeyAuth' -v"
}

if [ "$WEEK3_VERIFY_MODE" = "live" ]; then
    run_live_verification
elif [ "$WEEK3_VERIFY_MODE" = "test" ]; then
    run_local_verification
elif live_ready; then
    run_live_verification
else
    run_local_verification
fi

echo -e "\n${YELLOW}Result: passed $PASS, failed $FAIL${NC}"
[ $FAIL -eq 0 ] && echo -e "${GREEN}Week 3 verification passed!${NC}" || exit 1
