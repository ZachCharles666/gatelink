#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

API_BASE_URL=${API_BASE_URL:-http://localhost:8080}
BUYER_API_KEY=${BUYER_API_KEY:-}
BUYER_TOKEN=${BUYER_TOKEN:-}
REPORT_PATH=${REPORT_PATH:-/tmp/load_test_report.txt}
LOAD_TEST_MODE=${LOAD_TEST_MODE:-auto}
HEY_BIN=${HEY_BIN:-$(command -v hey || true)}
AB_BIN=${AB_BIN:-$(command -v ab || true)}

pass() {
    echo -e "${GREEN}PASS${NC} $1"
}

note() {
    echo -e "${YELLOW}NOTE${NC} $1"
}

fail() {
    echo -e "${RED}FAIL${NC} $1"
}

write_plan_report() {
    cat >"$REPORT_PATH" <<EOF
=== Dev-B 压测计划 $(date) ===
mode: plan
api_base_url: $API_BASE_URL

Prerequisites:
- install hey: go install github.com/rakyll/hey@latest
- or use built-in ab if available
- ensure API is reachable at $API_BASE_URL
- optionally set BUYER_API_KEY for /v1/models
- optionally set BUYER_TOKEN for /api/v1/buyer/balance
- otherwise the script will auto-register a temp buyer when API is reachable

Commands:
hey -n 1000 -c 50 $API_BASE_URL/health
hey -n 500 -c 20 -H "Authorization: Bearer \$BUYER_API_KEY" $API_BASE_URL/v1/models
hey -n 200 -c 10 -H "Authorization: Bearer \$BUYER_TOKEN" $API_BASE_URL/api/v1/buyer/balance
EOF
    cat "$REPORT_PATH"
    echo "报告已保存至 $REPORT_PATH"
}

run_case() {
    local label="$1"
    shift
    {
        echo "--- $label ---"
        "$@"
        echo
    } >>"$REPORT_PATH" 2>&1
}

extract_json_field() {
    local json="$1"
    local path="$2"
    JSON_INPUT="$json" python3 -c 'import json, os, sys
path = sys.argv[1].split(".")
data = json.loads(os.environ["JSON_INPUT"])
for part in path:
    data = data[part]
print(data)' "$path"
}

ensure_buyer_credentials() {
    if [ -n "$BUYER_API_KEY" ] && [ -n "$BUYER_TOKEN" ]; then
        return
    fi

    local buyer_resp
    buyer_resp=$(curl -s -X POST "${API_BASE_URL}/api/v1/buyer/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"load_$(date +%s)@test.com\",\"password\":\"pass123\"}")

    if ! printf '%s' "$buyer_resp" | grep -q '"api_key"'; then
        note "无法自动生成压测 buyer 凭证，将只运行无需鉴权的压测项"
        return
    fi

    BUYER_TOKEN=${BUYER_TOKEN:-$(extract_json_field "$buyer_resp" "data.token" 2>/dev/null || true)}
    BUYER_API_KEY=${BUYER_API_KEY:-$(extract_json_field "$buyer_resp" "data.api_key" 2>/dev/null || true)}
}

run_hey_case() {
    local label="$1"
    shift
    run_case "$label" "$HEY_BIN" "$@"
}

run_ab_case() {
    local label="$1"
    shift
    run_case "$label" "$AB_BIN" -q -l "$@"
}

if [ "$LOAD_TEST_MODE" = "plan" ]; then
    note "LOAD_TEST_MODE=plan，输出压测计划，不实际发起请求"
    write_plan_report
    exit 0
fi

if [ -z "$HEY_BIN" ] && [ -z "$AB_BIN" ]; then
    note "未检测到 hey 或 ab，切换到计划模式"
    write_plan_report
    exit 0
fi

if ! curl -sf "$API_BASE_URL/health" >/dev/null 2>&1; then
    note "API 不可达，切换到计划模式"
    write_plan_report
    exit 0
fi

ensure_buyer_credentials

echo "=== Dev-B 压测报告 $(date) ===" >"$REPORT_PATH"
echo "mode: run" >>"$REPORT_PATH"
echo "api_base_url: $API_BASE_URL" >>"$REPORT_PATH"
if [ -n "$HEY_BIN" ]; then
    echo "tool: hey" >>"$REPORT_PATH"
else
    echo "tool: ab" >>"$REPORT_PATH"
fi
echo >>"$REPORT_PATH"

if [ -n "$HEY_BIN" ]; then
    run_hey_case "/health" -n 1000 -c 50 "$API_BASE_URL/health"
else
    run_ab_case "/health" -n 1000 -c 50 "$API_BASE_URL/health"
fi

if [ -n "$BUYER_API_KEY" ]; then
    if [ -n "$HEY_BIN" ]; then
        run_hey_case "/v1/models" -n 500 -c 20 -H "Authorization: Bearer $BUYER_API_KEY" "$API_BASE_URL/v1/models"
    else
        run_ab_case "/v1/models" -n 500 -c 20 -H "Authorization: Bearer $BUYER_API_KEY" "$API_BASE_URL/v1/models"
    fi
else
    echo "--- /v1/models ---" >>"$REPORT_PATH"
    echo "SKIPPED: BUYER_API_KEY is not set" >>"$REPORT_PATH"
    echo >>"$REPORT_PATH"
fi

if [ -n "$BUYER_TOKEN" ]; then
    if [ -n "$HEY_BIN" ]; then
        run_hey_case "/api/v1/buyer/balance" -n 200 -c 10 -H "Authorization: Bearer $BUYER_TOKEN" "$API_BASE_URL/api/v1/buyer/balance"
    else
        run_ab_case "/api/v1/buyer/balance" -n 200 -c 10 -H "Authorization: Bearer $BUYER_TOKEN" "$API_BASE_URL/api/v1/buyer/balance"
    fi
else
    echo "--- /api/v1/buyer/balance ---" >>"$REPORT_PATH"
    echo "SKIPPED: BUYER_TOKEN is not set" >>"$REPORT_PATH"
    echo >>"$REPORT_PATH"
fi

pass "压测完成"
cat "$REPORT_PATH"
echo "报告已保存至 $REPORT_PATH"
