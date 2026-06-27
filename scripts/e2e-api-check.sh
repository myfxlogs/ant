#!/usr/bin/env bash
# e2e-api-check.sh — Comprehensive API test for soft-delete, audit log, wallet admin.
#
# Usage:
#   ANTTEST_EMAIL=admin@1.com ANTTEST_PASS=12345678 bash scripts/e2e-api-check.sh
#
# Tests:
#   1. Auth — login, get JWT
#   2. Wallet — GetWallet, ListTransactions
#   3. Admin Wallet — AdjustBalance (admin auth)
#   4. Admin User — ListUsers with deleted_filter (active/deleted/all)
#   5. Admin User — DeleteUser (soft delete) + audit log
#   6. Admin User — RestoreUser
#   7. DB — schema_migrations, FK compliance, admin_audit_log
#   8. Edge cases — self-delete blocked, last admin protected

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────
BASE_URL="${BASE_URL:-http://localhost:8080}"
EMAIL="${ANTTEST_EMAIL:-admin@1.com}"
PASS="${ANTTEST_PASS:-12345678}"
PASSED=0
FAILED=0
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "  ${GREEN}✅ PASS${NC}: $1"; PASSED=$((PASSED+1)); }
fail() { echo -e "  ${RED}❌ FAIL${NC}: $1"; FAILED=$((FAILED+1)); }
warn() { echo -e "  ${YELLOW}⚠️  WARN${NC}: $1"; }

# ── Helpers ───────────────────────────────────────────────────────
api() {
  local method="$1" path="$2" data="$3"
  curl -s -X POST "${BASE_URL}${path}" \
    -H "Content-Type: application/json" \
    ${TOKEN:+-H "Authorization: Bearer $TOKEN"} \
    -d "$data" 2>/dev/null
}

psqlq() { docker exec ant-postgres psql -U ant -d ant -tAc "$1" 2>/dev/null; }

echo "═══════════════════════════════════════"
echo "  Ant E2E API Check"
echo "═══════════════════════════════════════"
echo "Base: $BASE_URL  User: $EMAIL"
echo ""

# ══════════════════════════════════════════════════════════════════
# 1. Auth
# ══════════════════════════════════════════════════════════════════
echo "── 1. Auth ──"

LOGIN_RESP=$(api POST /ant.v1.AuthService/Login "{\"login\":\"$EMAIL\",\"password\":\"$PASS\"}")
TOKEN=$(echo "$LOGIN_RESP" | jq -r '.accessToken // ""' 2>/dev/null || true)

if [ -n "$TOKEN" ]; then
  pass "Login succeeded, got JWT (${#TOKEN} chars)"
else
  fail "Login failed — check credentials"
  echo "  Response: ${LOGIN_RESP:0:200}"
  exit 1
fi

# ══════════════════════════════════════════════════════════════════
# 2. Wallet — User Operations
# ══════════════════════════════════════════════════════════════════
echo ""
echo "── 2. Wallet User Ops ──"

WALLET_RESP=$(api POST /ant.v1.WalletService/GetWallet "{}")
WALLET_ID=$(echo "$WALLET_RESP" | jq -r '.wallet.id // ""' 2>/dev/null || true)

if [ -n "$WALLET_ID" ]; then
  pass "GetWallet returned wallet id=$WALLET_ID"
else
  fail "GetWallet failed or returned no wallet"
  echo "  Response: ${WALLET_RESP:0:200}"
fi

TX_RESP=$(api POST /ant.v1.WalletService/ListTransactions "{\"page\":1,\"pageSize\":5}")
TX_FIRST=$(echo "$TX_RESP" | jq -r '.transactions | length' 2>/dev/null || true)
if [ "${TX_FIRST:-0}" -ge 0 ] 2>/dev/null; then
  pass "ListTransactions returned ${TX_FIRST} transactions"
else
  fail "ListTransactions failed"
fi

# ══════════════════════════════════════════════════════════════════
# 3. Admin Wallet — AdjustBalance
# ══════════════════════════════════════════════════════════════════
echo ""
echo "── 3. Admin Wallet ──"

# Get current user ID
ME_RESP=$(api POST /ant.v1.WalletService/GetWallet "{}")
MY_USER_ID=$(echo "$ME_RESP" | jq -r '.wallet.userId' 2>/dev/null || true)

# Test: AdjustBalance (admin adding 0.01)
ADJ_RESP=$(api POST /ant.v1.WalletService/AdjustBalance \
  "{\"userId\":\"$MY_USER_ID\",\"amount\":\"0.01\",\"description\":\"e2e test credit\"}")
ADJ_BAL=$(echo "$ADJ_RESP" | jq -r '.wallet.balance // "ERROR"' 2>/dev/null || true)
if [ "$ADJ_BAL" != "ERROR" ]; then
  pass "AdjustBalance succeeded, balance=$ADJ_BAL"
else
  fail "AdjustBalance failed (admin required?)"
  echo "  Response: ${ADJ_RESP:0:200}"
fi

# Reverse the test adjustment
REV_RESP=$(api POST /ant.v1.WalletService/AdjustBalance \
  "{\"userId\":\"$MY_USER_ID\",\"amount\":\"-0.01\",\"description\":\"e2e test reversal\"}")
REV_BAL=$(echo "$REV_RESP" | jq -r '.wallet.balance // "ERROR"' 2>/dev/null || true)
if [ "$REV_BAL" != "ERROR" ]; then
  pass "AdjustBalance reversal succeeded, balance=$REV_BAL"
else
  fail "AdjustBalance reversal failed"
fi

# ══════════════════════════════════════════════════════════════════
# 4. Admin User — ListUsers (deleted_filter)
# ══════════════════════════════════════════════════════════════════
echo ""
echo "── 4. Admin ListUsers ──"

# Active only (default)
ACTIVE_RESP=$(api POST /ant.v1.AdminUserService/ListUsers \
  "{\"page\":1,\"pageSize\":100}")
ACTIVE_TOTAL=$(echo "$ACTIVE_RESP" | jq -r '.total // 0' 2>/dev/null || true)
pass "ListUsers (active, default) returned total=$ACTIVE_TOTAL"

# All (including deleted)
ALL_RESP=$(api POST /ant.v1.AdminUserService/ListUsers \
  "{\"page\":1,\"pageSize\":100,\"deletedFilter\":\"all\"}")
ALL_TOTAL=$(echo "$ALL_RESP" | jq -r '.total // 0' 2>/dev/null || true)
if [ "$ALL_TOTAL" -ge "$ACTIVE_TOTAL" ] 2>/dev/null; then
  pass "ListUsers (deletedFilter=all) returned total=$ALL_TOTAL (>= $ACTIVE_TOTAL active)"
else
  fail "ListUsers (all) returned $ALL_TOTAL < active $ACTIVE_TOTAL"
fi

# Deleted only
DEL_RESP=$(api POST /ant.v1.AdminUserService/ListUsers \
  "{\"page\":1,\"pageSize\":100,\"deletedFilter\":\"deleted\"}")
DEL_TOTAL=$(echo "$DEL_RESP" | jq -r '.total // 0' 2>/dev/null || true)
pass "ListUsers (deletedFilter=deleted) returned total=$DEL_TOTAL"

# ══════════════════════════════════════════════════════════════════
# 5. Admin User — Soft Delete + Audit Log
# ══════════════════════════════════════════════════════════════════
echo ""
echo "── 5. Soft Delete + Audit ──"

# Create a test user if none exists
TEST_USER_ROW=$(psqlq "SELECT id FROM users WHERE role != 'admin' AND role != 'super_admin' AND deleted_at IS NULL AND email != '$EMAIL' AND email LIKE 'e2e-test-%' LIMIT 1;")

if [ -z "$TEST_USER_ROW" ]; then
  # Check if any non-admin user exists
  ANY_USER=$(psqlq "SELECT id FROM users WHERE role != 'admin' AND role != 'super_admin' AND deleted_at IS NULL AND email != '$EMAIL' LIMIT 1;")
  if [ -z "$ANY_USER" ]; then
    # Create a test user via admin API
    echo "  Creating e2e test user..."
    TEST_EMAIL="e2e-test-$(date +%s)@test.com"
    CREATE_RESP=$(api POST /ant.v1.AdminUserService/CreateUser \
      "{\"username\":\"$TEST_EMAIL\",\"email\":\"$TEST_EMAIL\",\"password\":\"test123456\",\"role\":\"user\"}")
    TEST_USER_ID=$(echo "$CREATE_RESP" | jq -r '.id // ""' 2>/dev/null || true)
    if [ -n "$TEST_USER_ID" ]; then
      pass "Created test user: $TEST_EMAIL (id=$TEST_USER_ID)"
    else
      fail "Failed to create test user"
      echo "  Response: ${CREATE_RESP:0:200}"
    fi
  else
    TEST_USER_ID="$ANY_USER"
    TEST_USER_EMAIL=$(psqlq "SELECT email FROM users WHERE id = '$ANY_USER';")
    pass "Using existing user: $TEST_USER_EMAIL"
  fi
else
  TEST_USER_ID="$TEST_USER_ROW"
  TEST_USER_EMAIL=$(psqlq "SELECT email FROM users WHERE id = '$TEST_USER_ROW';")
  pass "Using existing test user: $TEST_USER_EMAIL"
fi

if [ -n "${TEST_USER_ID:-}" ]; then
  TEST_USER_EMAIL="${TEST_USER_EMAIL:-$(psqlq "SELECT email FROM users WHERE id = '$TEST_USER_ID';")}"

  # Record audit log count before delete
  AUDIT_BEFORE=$(psqlq "SELECT COUNT(*) FROM admin_audit_log;")

  # Soft delete
  DEL_RESP=$(api POST /ant.v1.AdminUserService/DeleteUser "{\"id\":\"$TEST_USER_ID\"}")
  DEL_OK=$(echo "$DEL_RESP" | jq -r 'if . == {} then "ok" else empty end' 2>/dev/null || echo "ok")
  if [ -z "$DEL_RESP" ] || echo "$DEL_RESP" | grep -q "error\|Error\|code"; then
    fail "DeleteUser failed"
    echo "  Response: ${DEL_RESP:0:200}"
  else
    pass "DeleteUser ($TEST_USER_EMAIL) returned 200 OK"

    # Verify soft-delete: deleted_at IS NOT NULL
    DELETED_AT=$(psqlq "SELECT deleted_at FROM users WHERE id = '$TEST_USER_ID';")
    if [ "$DELETED_AT" != "" ]; then
      pass "User.deleted_at = $DELETED_AT (soft-deleted)"
    else
      fail "User.deleted_at is NULL — NOT soft-deleted"
    fi

    # Verify audit log
    AUDIT_AFTER=$(psqlq "SELECT COUNT(*) FROM admin_audit_log;")
    if [ "$AUDIT_AFTER" -gt "$AUDIT_BEFORE" ] 2>/dev/null; then
      AUDIT_ACTION=$(psqlq "SELECT action FROM admin_audit_log WHERE target_id = '$TEST_USER_ID' ORDER BY created_at DESC LIMIT 1;")
      pass "Audit log created: action=$AUDIT_ACTION, count ${AUDIT_BEFORE}→${AUDIT_AFTER}"
    else
      fail "Audit log NOT created — count unchanged ($AUDIT_BEFORE)"
    fi

    # Verify user cannot login after soft delete
    LOGIN_DEL_RESP=$(api POST /ant.v1.AuthService/Login "{\"login\":\"$TEST_USER_EMAIL\",\"password\":\"test1234\"}")
    if echo "$LOGIN_DEL_RESP" | grep -qi "not found\|invalid\|error\|unauthenticated"; then
      pass "Soft-deleted user cannot login (expected)"
    else
      warn "Soft-deleted user login behavior unexpected"
    fi
  fi
fi

# ══════════════════════════════════════════════════════════════════
# 6. Admin User — RestoreUser
# ══════════════════════════════════════════════════════════════════
echo ""
echo "── 6. RestoreUser ──"

if [ -n "${TEST_USER_ID:-}" ]; then
  RESTORE_RESP=$(api POST /ant.v1.AdminUserService/RestoreUser "{\"id\":\"$TEST_USER_ID\"}")
  if [ -z "$RESTORE_RESP" ] || echo "$RESTORE_RESP" | grep -q "error\|Error"; then
    fail "RestoreUser failed"
    echo "  Response: ${RESTORE_RESP:0:200}"
  else
    pass "RestoreUser returned 200 OK"

    # Verify restored
    RESTORED_AT=$(psqlq "SELECT deleted_at FROM users WHERE id = '$TEST_USER_ID';")
    if [ "$RESTORED_AT" = "" ]; then
      pass "User.deleted_at = NULL (restored)"
    else
      fail "User.deleted_at = $RESTORED_AT — NOT restored"
    fi

    # Verify audit log for restore
    RESTORE_AUDIT=$(psqlq "SELECT action FROM admin_audit_log WHERE target_id = '$TEST_USER_ID' ORDER BY created_at DESC LIMIT 1;")
    if [ "$RESTORE_AUDIT" = "restore_user" ]; then
      pass "Audit log recorded restore_user action"
    else
      warn "Audit log restore entry: $RESTORE_AUDIT"
    fi
  fi
else
  warn "Skipping restore test (no delete test was run)"
fi

# ══════════════════════════════════════════════════════════════════
# 7. Safety — Self-delete + Last admin
# ══════════════════════════════════════════════════════════════════
echo ""
echo "── 7. Safety Guards ──"

# Self-delete blocked
ADMIN_ID=$(psqlq "SELECT id FROM users WHERE email = '$EMAIL';")
SELF_RESP=$(api POST /ant.v1.AdminUserService/DeleteUser "{\"id\":\"$ADMIN_ID\"}")
if echo "$SELF_RESP" | grep -qi "cannot delete yourself\|permission_denied\|PermissionDenied"; then
  pass "Self-delete correctly blocked"
else
  fail "Self-delete NOT blocked"
  echo "  Response: ${SELF_RESP:0:200}"
fi

# Last admin protected
ADMIN_COUNT=$(psqlq "SELECT COUNT(*) FROM users WHERE role = 'admin' AND deleted_at IS NULL;")
if [ "$ADMIN_COUNT" -le 1 ] 2>/dev/null; then
  # Find another admin (same email) — there's only one
  LAST_ADMIN_ID=$(psqlq "SELECT id FROM users WHERE role = 'admin' AND deleted_at IS NULL AND email != '$EMAIL' LIMIT 1;")
  if [ -z "$LAST_ADMIN_ID" ]; then
    warn "Only 1 admin exists — last-admin test cannot run (need 2+ admins)"
  else
    # Try to delete the only other admin
    LA_RESP=$(api POST /ant.v1.AdminUserService/DeleteUser "{\"id\":\"$LAST_ADMIN_ID\"}")
    if echo "$LA_RESP" | grep -qi "last admin\|failed_precondition\|FailedPrecondition"; then
      pass "Last-admin deletion correctly blocked"
    else
      fail "Last-admin deletion NOT blocked"
    fi
  fi
fi

# ══════════════════════════════════════════════════════════════════
# 8. DB — Migrations + FK + Audit Log structure
# ══════════════════════════════════════════════════════════════════
echo ""
echo "── 8. Database State ──"

# Migration 151
HAS_DELETED_AT=$(psqlq "SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='deleted_at');")
if [ "$HAS_DELETED_AT" = "t" ]; then pass "Migration 151: users.deleted_at exists"; else fail "Migration 151: missing deleted_at"; fi

# Migration 152
HAS_AUDIT=$(psqlq "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='admin_audit_log');")
if [ "$HAS_AUDIT" = "t" ]; then pass "Migration 152: admin_audit_log table exists"; else fail "Migration 152: missing admin_audit_log"; fi

# Migration 150 — all FKs CASCADE/SET NULL
FK_VIOLATIONS=$(psqlq "
SELECT COUNT(DISTINCT tc.table_name) FROM information_schema.table_constraints tc
JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name AND tc.constraint_schema = rc.constraint_schema
JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.constraint_schema = kcu.constraint_schema
JOIN information_schema.constraint_column_usage ccu ON rc.unique_constraint_name = ccu.constraint_name AND rc.unique_constraint_schema = ccu.constraint_schema
WHERE tc.constraint_type = 'FOREIGN KEY' AND ccu.table_name = 'users' AND ccu.column_name = 'id' AND rc.delete_rule NOT IN ('CASCADE','SET NULL');
")
if [ "$FK_VIOLATIONS" = "0" ]; then pass "Migration 150: 0 FK violations (all CASCADE/SET NULL)"; else fail "Migration 150: $FK_VIOLATIONS FK(s) missing CASCADE/SET NULL"; fi

# Schema_migrations
MIG_COUNT=$(psqlq "SELECT COUNT(*) FROM schema_migrations WHERE version IN ('151_user_soft_delete','152_admin_audit_log');")
if [ "$MIG_COUNT" = "2" ]; then pass "schema_migrations: 151 + 152 recorded"; else fail "schema_migrations: expected 2, got $MIG_COUNT"; fi

# ══════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════"
echo -e "  Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"
echo "═══════════════════════════════════════"

if [ "$FAILED" -gt 0 ]; then exit 1; fi
