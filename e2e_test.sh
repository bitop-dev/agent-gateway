#!/usr/bin/env bash
# E2E tests for agent-gateway running in k8s
# Usage: GATEWAY=http://127.0.0.1:8080 ADMIN_KEY=admin-secret-k8s bash e2e_test.sh

set -euo pipefail

GW="${GATEWAY:-http://127.0.0.1:8080}"
KEY="${ADMIN_KEY:-admin-secret-k8s}"

PASS=0
FAIL=0
TESTS=()

pass() { echo "  ✅ $1"; PASS=$((PASS + 1)); TESTS+=("PASS: $1"); }
fail() { echo "  ❌ $1: $2"; FAIL=$((FAIL + 1)); TESTS+=("FAIL: $1: $2"); }

assert_status() {
  local desc="$1" method="$2" path="$3" expected="$4"
  shift 4
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$GW$path" "$@" 2>/dev/null)
  if [ "$status" = "$expected" ]; then
    pass "$desc (HTTP $status)"
  else
    fail "$desc" "expected $expected, got $status"
  fi
}

assert_json() {
  local desc="$1" expr="$2"
  shift 2
  local body
  body=$(curl -s "$@" 2>/dev/null || echo '{}')
  if echo "$body" | python3 -c "import json,sys; d=json.load(sys.stdin); assert $expr, f'assertion failed: {d}'" 2>/dev/null; then
    pass "$desc"
  else
    fail "$desc" "json assertion [$expr] on: $(echo "$body" | head -c 200)"
  fi
}

echo ""
echo "═══════════════════════════════════════════"
echo "  Agent Gateway E2E Tests"
echo "  Target: $GW"
echo "═══════════════════════════════════════════"

# ─────────────────────────────────────────────
echo ""
echo "▸ Health"
# ─────────────────────────────────────────────

assert_status "GET /v1/health returns 200" GET "/v1/health" 200

assert_json "health response has ok=true" \
  "d.get('ok') == True" \
  "$GW/v1/health"

assert_json "health shows workers > 0" \
  "d.get('workers', 0) > 0" \
  "$GW/v1/health"

# ─────────────────────────────────────────────
echo ""
echo "▸ Auth"
# ─────────────────────────────────────────────

assert_status "unauthenticated GET /v1/tasks returns 401" \
  GET "/v1/tasks" 401

assert_status "bad key returns 401" \
  GET "/v1/tasks" 401 \
  -H "Authorization: Bearer wrong-key"

assert_status "admin key returns 200" \
  GET "/v1/tasks" 200 \
  -H "Authorization: Bearer $KEY"

# query param auth (SSE workaround)
assert_status "query param token auth works" \
  GET "/v1/tasks?token=$KEY" 200

# ─────────────────────────────────────────────
echo ""
echo "▸ API Keys CRUD"
# ─────────────────────────────────────────────

KEY_RESP=$(curl -s -X POST "$GW/v1/auth/keys" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"e2e-test-key","scopes":["tasks:read"]}')
TEST_KEY=$(echo "$KEY_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('key',''))" 2>/dev/null)

if [ -n "$TEST_KEY" ] && [ "$TEST_KEY" != "None" ]; then
  pass "create API key"

  assert_status "scoped key can read tasks" \
    GET "/v1/tasks" 200 \
    -H "Authorization: Bearer $TEST_KEY"

  assert_status "scoped key cannot write tasks (403)" \
    POST "/v1/tasks" 403 \
    -H "Authorization: Bearer $TEST_KEY" \
    -H "Content-Type: application/json" \
    -d '{"profile":"researcher","task":"test"}'

  assert_json "list API keys includes e2e-test-key" \
    "any(k.get('name')=='e2e-test-key' for k in d.get('keys',[]))" \
    -H "Authorization: Bearer $KEY" \
    "$GW/v1/auth/keys"

  KEY_ID=$(echo "$KEY_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
  if [ -n "$KEY_ID" ] && [ "$KEY_ID" != "None" ]; then
    assert_status "delete API key" DELETE "/v1/auth/keys?id=$KEY_ID" 200 \
      -H "Authorization: Bearer $KEY"
  fi
else
  fail "create API key" "no key in response: $KEY_RESP"
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Workers"
# ─────────────────────────────────────────────

assert_status "GET /v1/workers returns 200" GET "/v1/workers" 200

assert_json "workers registered (>= 1)" \
  "len(d.get('workers',[])) >= 1" \
  "$GW/v1/workers"

# ─────────────────────────────────────────────
echo ""
echo "▸ Agents"
# ─────────────────────────────────────────────

assert_status "GET /v1/agents returns 200" \
  GET "/v1/agents" 200 \
  -H "Authorization: Bearer $KEY"

assert_json "agents list is non-empty" \
  "len(d.get('agents',[])) > 0" \
  -H "Authorization: Bearer $KEY" \
  "$GW/v1/agents"

# ─────────────────────────────────────────────
echo ""
echo "▸ Task Submission (sync)"
# ─────────────────────────────────────────────

TASK_RESP=$(curl -s -X POST "$GW/v1/tasks" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"profile":"researcher","task":"Say hello in one word."}')
TASK_ID=$(echo "$TASK_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('id',d.get('taskId','')))" 2>/dev/null)
TASK_STATUS=$(echo "$TASK_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status',''))" 2>/dev/null)

if [ -n "$TASK_ID" ] && [ "$TASK_ID" != "None" ] && [ "$TASK_ID" != "" ]; then
  pass "sync task submitted (id=$TASK_ID)"

  if [ "$TASK_STATUS" = "completed" ]; then
    pass "sync task completed"
  else
    fail "sync task completed" "status=$TASK_STATUS"
  fi

  HAS_RESULT=$(echo "$TASK_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print('yes' if d.get('output') or d.get('result') else 'no')" 2>/dev/null)
  if [ "$HAS_RESULT" = "yes" ]; then
    pass "sync task has output"
  else
    fail "sync task has output" "empty"
  fi
else
  fail "sync task submitted" "$(echo "$TASK_RESP" | head -c 200)"
  TASK_ID=""
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Task Detail"
# ─────────────────────────────────────────────

if [ -n "$TASK_ID" ]; then
  assert_status "GET /v1/tasks/$TASK_ID returns 200" \
    GET "/v1/tasks/$TASK_ID" 200 \
    -H "Authorization: Bearer $KEY"

  assert_json "task detail has correct id" \
    "d.get('id') == '$TASK_ID'" \
    -H "Authorization: Bearer $KEY" \
    "$GW/v1/tasks/$TASK_ID"

  assert_json "task detail has profile=researcher" \
    "d.get('profile') == 'researcher'" \
    -H "Authorization: Bearer $KEY" \
    "$GW/v1/tasks/$TASK_ID"
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Task List"
# ─────────────────────────────────────────────

assert_json "task list returns array" \
  "isinstance(d.get('tasks'), list)" \
  -H "Authorization: Bearer $KEY" \
  "$GW/v1/tasks?limit=10"

assert_json "task list respects limit" \
  "len(d.get('tasks',[])) <= 10" \
  -H "Authorization: Bearer $KEY" \
  "$GW/v1/tasks?limit=10"

# ─────────────────────────────────────────────
echo ""
echo "▸ Async Task"
# ─────────────────────────────────────────────

ASYNC_RESP=$(curl -s -X POST "$GW/v1/tasks" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"profile":"researcher","task":"Say goodbye in one word.","async":true}')
ASYNC_ID=$(echo "$ASYNC_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('id',d.get('taskId','')))" 2>/dev/null)
ASYNC_STATUS=$(echo "$ASYNC_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status',''))" 2>/dev/null)

if [ -n "$ASYNC_ID" ] && [ "$ASYNC_ID" != "None" ] && [ "$ASYNC_ID" != "" ]; then
  pass "async task submitted (id=$ASYNC_ID)"

  if [ "$ASYNC_STATUS" = "queued" ] || [ "$ASYNC_STATUS" = "running" ]; then
    pass "async task returns non-blocking status ($ASYNC_STATUS)"
  else
    fail "async task returns non-blocking" "got $ASYNC_STATUS"
  fi

  PSTATUS=""
  for i in $(seq 1 20); do
    sleep 3
    PSTATUS=$(curl -s "$GW/v1/tasks/$ASYNC_ID" -H "Authorization: Bearer $KEY" | \
      python3 -c "import json,sys; print(json.load(sys.stdin).get('status',''))" 2>/dev/null)
    if [ "$PSTATUS" = "completed" ] || [ "$PSTATUS" = "failed" ]; then break; fi
  done

  if [ "$PSTATUS" = "completed" ]; then
    pass "async task completed after polling"
  else
    fail "async task completed" "final status=$PSTATUS"
  fi
else
  fail "async task submitted" "$(echo "$ASYNC_RESP" | head -c 200)"
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Parallel Tasks"
# ─────────────────────────────────────────────

PAR_RESP=$(curl -s -X POST "$GW/v1/tasks/parallel" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"tasks":[{"profile":"researcher","task":"Say yes."},{"profile":"researcher","task":"Say no."}]}')

PAR_OK=$(echo "$PAR_RESP" | python3 -c "
import json,sys; d=json.load(sys.stdin)
results = d.get('results', d.get('tasks',[]))
print('yes' if len(results)==2 else 'no')" 2>/dev/null)
if [ "$PAR_OK" = "yes" ]; then
  pass "parallel tasks returned 2 results"
else
  fail "parallel tasks" "$(echo "$PAR_RESP" | head -c 200)"
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Webhooks CRUD"
# ─────────────────────────────────────────────

WH_RESP=$(curl -s -X POST "$GW/v1/webhooks" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"e2e-test","path":"e2e-test","profile":"researcher","taskTemplate":"E2E test: {{msg}}","enabled":true}')
WH_ID=$(echo "$WH_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$WH_ID" ] && [ "$WH_ID" != "None" ]; then
  pass "create webhook"

  assert_json "list webhooks includes e2e-test" \
    "any(w.get('name')=='e2e-test' for w in d.get('webhooks',[]))" \
    -H "Authorization: Bearer $KEY" \
    "$GW/v1/webhooks"

  TRIG_RESP=$(curl -s -X POST "$GW/v1/webhooks/e2e-test" \
    -H "Content-Type: application/json" \
    -d '{"msg":"hello from e2e"}')
  TRIG_ID=$(echo "$TRIG_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('taskId',d.get('id','')))" 2>/dev/null)
  if [ -n "$TRIG_ID" ] && [ "$TRIG_ID" != "None" ] && [ "$TRIG_ID" != "" ]; then
    pass "trigger webhook creates task ($TRIG_ID)"
  else
    fail "trigger webhook" "$(echo "$TRIG_RESP" | head -c 200)"
  fi

  assert_status "delete webhook" DELETE "/v1/webhooks?id=$WH_ID" 200 \
    -H "Authorization: Bearer $KEY"
else
  fail "create webhook" "$(echo "$WH_RESP" | head -c 200)"
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Schedules CRUD"
# ─────────────────────────────────────────────

SCHED_RESP=$(curl -s -X POST "$GW/v1/schedules" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"e2e-sched","cron":"0 0 31 2 *","timezone":"UTC","profile":"researcher","task":"never runs","enabled":false}')
SCHED_ID=$(echo "$SCHED_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$SCHED_ID" ] && [ "$SCHED_ID" != "None" ]; then
  pass "create schedule"

  assert_json "list schedules includes e2e-sched" \
    "any(s.get('name')=='e2e-sched' for s in d.get('schedules',[]))" \
    -H "Authorization: Bearer $KEY" \
    "$GW/v1/schedules"

  assert_status "delete schedule" DELETE "/v1/schedules?id=$SCHED_ID" 200 \
    -H "Authorization: Bearer $KEY"
else
  fail "create schedule" "$(echo "$SCHED_RESP" | head -c 200)"
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Memory CRUD"
# ─────────────────────────────────────────────

# Write
assert_status "write memory" POST "/v1/memory?profile=e2e-test" 200 \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"key":"test_key","value":"test_value"}'

# Read all — gateway returns "entries" not "memories"
assert_json "read memory returns test_key" \
  "any(m.get('key')=='test_key' and m.get('value')=='test_value' for m in d.get('entries',d.get('memories',[])))" \
  -H "Authorization: Bearer $KEY" \
  "$GW/v1/memory?profile=e2e-test"

# Read specific key — gateway returns single object with key/value
SINGLE_MEM=$(curl -s -H "Authorization: Bearer $KEY" "$GW/v1/memory?profile=e2e-test&key=test_key")
SINGLE_OK=$(echo "$SINGLE_MEM" | python3 -c "
import json,sys; d=json.load(sys.stdin)
# might be object {key,value} or {entries:[...]}
if d.get('key') == 'test_key': print('yes')
elif any(m.get('key')=='test_key' for m in d.get('entries',d.get('memories',[]))): print('yes')
else: print('no')" 2>/dev/null)
if [ "$SINGLE_OK" = "yes" ]; then
  pass "read specific memory key"
else
  fail "read specific memory key" "$(echo "$SINGLE_MEM" | head -c 200)"
fi

# Overwrite
curl -s -X POST "$GW/v1/memory?profile=e2e-test" \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"key":"test_key","value":"updated_value"}' > /dev/null

UPD_MEM=$(curl -s -H "Authorization: Bearer $KEY" "$GW/v1/memory?profile=e2e-test&key=test_key")
UPD_OK=$(echo "$UPD_MEM" | python3 -c "
import json,sys; d=json.load(sys.stdin)
if d.get('value') == 'updated_value': print('yes')
elif any(m.get('value')=='updated_value' for m in d.get('entries',d.get('memories',[]))): print('yes')
else: print('no')" 2>/dev/null)
if [ "$UPD_OK" = "yes" ]; then
  pass "overwrite memory updates value"
else
  fail "overwrite memory" "$(echo "$UPD_MEM" | head -c 200)"
fi

# Delete specific key
assert_status "delete specific memory key" DELETE "/v1/memory?profile=e2e-test&key=test_key" 200 \
  -H "Authorization: Bearer $KEY"

# Verify deleted
DEL_CHECK=$(curl -s -H "Authorization: Bearer $KEY" "$GW/v1/memory?profile=e2e-test")
DEL_OK=$(echo "$DEL_CHECK" | python3 -c "
import json,sys; d=json.load(sys.stdin)
entries = d.get('entries', d.get('memories', []))
if isinstance(entries, list): print('yes' if not any(m.get('key')=='test_key' for m in entries) else 'no')
else: print('yes' if not d.get('key') else 'no')" 2>/dev/null)
if [ "$DEL_OK" = "yes" ]; then
  pass "memory key deleted"
else
  fail "memory key deleted" "$(echo "$DEL_CHECK" | head -c 200)"
fi

# Bulk: write two, delete all
curl -s -X POST "$GW/v1/memory?profile=e2e-test" \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"key":"a","value":"1"}' > /dev/null
curl -s -X POST "$GW/v1/memory?profile=e2e-test" \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"key":"b","value":"2"}' > /dev/null
curl -s -X DELETE "$GW/v1/memory?profile=e2e-test" \
  -H "Authorization: Bearer $KEY" > /dev/null

CLEAR_CHECK=$(curl -s -H "Authorization: Bearer $KEY" "$GW/v1/memory?profile=e2e-test")
CLEAR_OK=$(echo "$CLEAR_CHECK" | python3 -c "
import json,sys; d=json.load(sys.stdin)
entries = d.get('entries', d.get('memories', []))
print('yes' if not entries or (isinstance(entries, list) and len(entries)==0) else 'no')" 2>/dev/null)
if [ "$CLEAR_OK" = "yes" ]; then
  pass "delete all memory for profile"
else
  fail "delete all memory" "$(echo "$CLEAR_CHECK" | head -c 200)"
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Costs"
# ─────────────────────────────────────────────

assert_status "GET /v1/costs returns 200" \
  GET "/v1/costs" 200 \
  -H "Authorization: Bearer $KEY"

# Gateway returns "profiles" not "costs"
assert_json "costs response has expected shape" \
  "'totalCost' in d or 'costs' in d or 'profiles' in d" \
  -H "Authorization: Bearer $KEY" \
  "$GW/v1/costs"

assert_json "costs with since param works" \
  "'totalCost' in d or 'costs' in d or 'profiles' in d" \
  -H "Authorization: Bearer $KEY" \
  "$GW/v1/costs?since=2026-01-01"

assert_status "GET /v1/costs/pricing returns 200" \
  GET "/v1/costs/pricing" 200 \
  -H "Authorization: Bearer $KEY"

assert_json "pricing has models" \
  "len(d.get('pricing',d.get('models',[]))) > 0" \
  -H "Authorization: Bearer $KEY" \
  "$GW/v1/costs/pricing"

# ─────────────────────────────────────────────
echo ""
echo "▸ SSE Events"
# ─────────────────────────────────────────────

# SSE connects with query param auth (timeout = success for streaming)
SSE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" --max-time 2 \
  -H "Accept: text/event-stream" \
  "$GW/v1/events?token=$KEY" 2>/dev/null || true)
# 000 = timeout (connection stayed open = success for SSE), 200 also fine
if [ "$SSE_STATUS" = "200" ] || [ "$SSE_STATUS" = "000" ] || [ -z "$SSE_STATUS" ]; then
  pass "SSE events endpoint connects"
else
  fail "SSE events endpoint" "status $SSE_STATUS"
fi

assert_status "SSE without auth returns 401" \
  GET "/v1/events" 401

# ─────────────────────────────────────────────
echo ""
echo "▸ Dashboard Static Assets"
# ─────────────────────────────────────────────

assert_status "GET / serves dashboard" GET "/" 200

DASH_HTML=$(curl -s "$GW/")
if echo "$DASH_HTML" | grep -q "Agent Platform"; then
  pass "dashboard HTML contains 'Agent Platform'"
else
  fail "dashboard HTML content" "missing 'Agent Platform'"
fi

if echo "$DASH_HTML" | grep -q "assets/index-"; then
  pass "dashboard references JS bundle"
else
  fail "dashboard JS reference" "missing"
fi

JS_FILE=$(echo "$DASH_HTML" | grep -o 'assets/index-[^"]*\.js' | head -1)
if [ -n "$JS_FILE" ]; then
  assert_status "JS bundle loads" GET "/$JS_FILE" 200
fi

CSS_FILE=$(echo "$DASH_HTML" | grep -o 'assets/index-[^"]*\.css' | head -1)
if [ -n "$CSS_FILE" ]; then
  assert_status "CSS bundle loads" GET "/$CSS_FILE" 200
fi

# ─────────────────────────────────────────────
echo ""
echo "▸ Error Handling"
# ─────────────────────────────────────────────

assert_status "nonexistent task returns 404" \
  GET "/v1/tasks/nonexistent-id-000" 404 \
  -H "Authorization: Bearer $KEY"

assert_status "missing profile returns 400" \
  POST "/v1/tasks" 400 \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"task":"no profile"}'

assert_status "empty body returns 400" \
  POST "/v1/tasks" 400 \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{}'

# ─────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════"
echo "  Results: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════════════"

if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "Failed tests:"
  for t in "${TESTS[@]}"; do
    if [[ "$t" == FAIL* ]]; then echo "  $t"; fi
  done
  exit 1
fi

echo ""
echo "All tests passed! 🎉"
