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
REPO_ROOT=$(cd "$API_DIR/.." && pwd)
WEB_DIR="$REPO_ROOT/web"
WEB_BASE_URL=${WEB_BASE_URL:-http://127.0.0.1:3000}
WEB_HOST=${WEB_HOST:-127.0.0.1}
WEB_PORT=${WEB_PORT:-3000}
export GOCACHE=${GOCACHE:-/tmp/ttt-go-build}
export GOTOOLCHAIN=${GOTOOLCHAIN:-go1.25.8+auto}
DEV_PID=""
DEV_STARTED=0

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

cleanup() {
    if [ "$DEV_STARTED" -eq 1 ] && [ -n "$DEV_PID" ]; then
        kill "$DEV_PID" >/dev/null 2>&1 || true
        wait "$DEV_PID" >/dev/null 2>&1 || true
    fi
}

trap cleanup EXIT

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

check_url_contains() {
    local label="$1"
    local path="$2"
    local pattern="$3"
    local output

    output=$(curl -s "${WEB_BASE_URL}${path}" 2>&1) || {
        fail "$label" "$output"
        return
    }

    if printf '%s' "$output" | grep -q "$pattern"; then
        pass "$label"
        return
    fi

    fail "$label" "$output"
}

ensure_web_ready() {
    if curl -sf "${WEB_BASE_URL}/seller/login" >/dev/null 2>&1; then
        return
    fi

    note "web server not detected, starting npm run start on ${WEB_HOST}:${WEB_PORT}"
    (
        cd "$WEB_DIR" &&
        npm run start -- --hostname "$WEB_HOST" --port "$WEB_PORT"
    ) >/tmp/week5-next.log 2>&1 &
    DEV_PID=$!
    DEV_STARTED=1

    for _ in $(seq 1 30); do
        if curl -sf "${WEB_BASE_URL}/seller/login" >/dev/null 2>&1; then
            note "web dev server is ready"
            return
        fi
        sleep 1
    done

    fail "前端登录页可访问" "$(cat /tmp/week5-next.log 2>/dev/null || true)"
}

check_cmd "结算 Service 编译" "cd \"$API_DIR\" && go build ./internal/accounting/..."
check_cmd "轮询器编译" "cd \"$API_DIR\" && go build ./internal/poller/..."
check_cmd "前端构建" "cd \"$WEB_DIR\" && npm run build"

ensure_web_ready

if [ "$FAIL" -eq 0 ]; then
    check_url_contains "前端登录页可访问" "/seller/login" "卖家登录"
    check_url_contains "前端注册页可访问" "/seller/register" "卖家注册"
fi

echo -e "\n${YELLOW}Result: passed $PASS, failed $FAIL${NC}"
[ $FAIL -eq 0 ] && echo -e "${GREEN}Week 5 verification passed!${NC}" || exit 1
