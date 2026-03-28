#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-${BASE_URL:-http://127.0.0.1:6789}}"
USERNAME="${VPS_MONITOR_USERNAME:-admin}"
PASSWORD="${VPS_MONITOR_PASSWORD:-password}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

PASS_COUNT=0
WARN_COUNT=0

say() {
  printf '\n== %s ==\n' "$1"
}

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  printf 'PASS: %s\n' "$1"
}

warn() {
  WARN_COUNT=$((WARN_COUNT + 1))
  printf 'WARN: %s\n' "$1"
}

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

http_json() {
  local method="$1"
  local path="$2"
  local output="$3"
  local body="${4:-}"
  shift 4 || true

  local args=(
    -sS
    -o "$output"
    -w '%{http_code}'
    -X "$method"
    "$BASE_URL$path"
  )

  if [[ -n "${TOKEN:-}" ]]; then
    args+=(-H "Authorization: Bearer $TOKEN")
  fi

  if [[ -n "$body" ]]; then
    args+=(-H 'Content-Type: application/json' -d "$body")
  fi

  curl "${args[@]}"
}

expect_status() {
  local actual="$1"
  local expected="$2"
  local label="$3"
  if [[ "$actual" != "$expected" ]]; then
    fail "$label returned HTTP $actual (expected $expected)"
  fi
}

extract_json_string() {
  local key="$1"
  local file="$2"
  sed -n "s/.*\"$key\":\"\\([^\"]*\\)\".*/\\1/p" "$file" | head -n1
}

say "Authentication"
LOGIN_BODY='{"username":"'"$USERNAME"'","password":"'"$PASSWORD"'"}'
status="$(http_json POST /api/v1/auth/login "$tmpdir/login.json" "$LOGIN_BODY")"
expect_status "$status" 200 "auth/login"
TOKEN="$(extract_json_string token "$tmpdir/login.json")"
[[ -n "$TOKEN" ]] || fail "auth/login did not return a token"
pass "auth/login"

status="$(http_json GET /api/v1/auth/me "$tmpdir/me.json")"
expect_status "$status" 200 "auth/me"
grep -q "\"username\":\"$USERNAME\"" "$tmpdir/me.json" || fail "auth/me response missing username"
pass "auth/me"

say "Core mobile endpoints"
status="$(http_json GET /api/v1/system/stats "$tmpdir/system-stats.json")"
expect_status "$status" 200 "system/stats"
grep -q '"hostInfo"' "$tmpdir/system-stats.json" || fail "system/stats response missing hostInfo"
pass "system/stats"

status="$(http_json GET /api/v1/alerts "$tmpdir/alerts.json")"
expect_status "$status" 200 "alerts"
grep -q '"alerts"' "$tmpdir/alerts.json" || fail "alerts response missing alerts"
pass "alerts"

status="$(http_json GET /api/v1/alerts/config "$tmpdir/alerts-config.json")"
expect_status "$status" 200 "alerts/config"
grep -q '"config"' "$tmpdir/alerts-config.json" || fail "alerts/config response missing config"
pass "alerts/config"

status="$(http_json POST /api/v1/devices/register "$tmpdir/device-register.json" '{"token":"smoke-test-token","platform":"android"}')"
expect_status "$status" 202 "devices/register"
grep -q '"message":"Device registration accepted"' "$tmpdir/device-register.json" || fail "devices/register did not acknowledge registration"
pass "devices/register"

say "Docker-backed endpoints"
status="$(http_json GET /api/v1/containers "$tmpdir/containers.json")"
if [[ "$status" == "200" ]]; then
  grep -q '"containers"' "$tmpdir/containers.json" || fail "containers response missing containers"
  pass "containers"
else
  if grep -q 'Cannot connect to the Docker daemon' "$tmpdir/containers.json"; then
    warn "containers unavailable because Docker daemon is not reachable from the server environment"
  else
    fail "containers returned HTTP $status: $(cat "$tmpdir/containers.json")"
  fi
fi

status="$(http_json GET /api/v1/images "$tmpdir/images.json")"
if [[ "$status" == "200" ]]; then
  grep -q '"images"' "$tmpdir/images.json" || fail "images response missing images"
  pass "images"
else
  if grep -q 'Cannot connect to the Docker daemon' "$tmpdir/images.json"; then
    warn "images unavailable because Docker daemon is not reachable from the server environment"
  else
    fail "images returned HTTP $status: $(cat "$tmpdir/images.json")"
  fi
fi

status="$(http_json GET /api/v1/networks "$tmpdir/networks.json")"
if [[ "$status" == "200" ]]; then
  grep -q '"networks"' "$tmpdir/networks.json" || fail "networks response missing networks"
  pass "networks"
else
  if grep -q 'Cannot connect to the Docker daemon' "$tmpdir/networks.json"; then
    warn "networks unavailable because Docker daemon is not reachable from the server environment"
  else
    fail "networks returned HTTP $status: $(cat "$tmpdir/networks.json")"
  fi
fi

say "Summary"
printf 'PASS=%s WARN=%s\n' "$PASS_COUNT" "$WARN_COUNT"
