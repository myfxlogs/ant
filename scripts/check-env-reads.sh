#!/usr/bin/env bash
# check-env-reads.sh —— ENV-TO-PG-1 防回潮门禁（ADR-0031 / 派工单
# docs/plan/2026-10-env-to-pg-consolidation.md 机制要求③）。
# 原则：业务配置唯一真相=PG（system_config/platform_secrets）；env 只留「连库之前的引导件」。
# 「.env 里能读到=合法」不是理由，读本表才是。
#
# 规则：backend/{internal,cmd,tools} 的非 _test.go 中，字面 os.Getenv("KEY")/os.LookupEnv("KEY")
# 只允许出现在 scripts/env-allowlist.txt 登记的白名单文件（bootstrap 清单）里；
# 其余文件出现即拒（新键/新读取点必须走 seed-once + overlay，或过评审改本表）。
# 变量式读取（os.Getenv(key)）不可静态判键，白名单文件外一律视为违规面——
# 现有白名单文件即唯一合法收口（config.getenv*/bootstrap envValue/sentry/master_provider/otel）。
# 自身排除（本脚本含模式字面量）。CI 用法：bash scripts/check-env-reads.sh [额外文件...]。
set -u
cd "$(git rev-parse --show-toplevel 2>/dev/null || echo .)"

ALLOWLIST="scripts/env-allowlist.txt"
[ -f "$ALLOWLIST" ] || { echo "FAIL: $ALLOWLIST 缺失"; exit 1; }

fails=0

# Go 消费面：internal/cmd/tools 非 _test.go，逐文件判定
while IFS= read -r f; do
  # 白名单文件（bootstrap 清单）整文件放行
  grep -qxF "$f" "$ALLOWLIST" && continue
  while IFS= read -r ln; do
    lnno=$(echo "$ln" | cut -d: -f1)
    echo "FAIL: $f:$lnno env 直读（业务配置唯一真相=PG；引导件读取点须登记 $ALLOWLIST）"
    fails=$((fails+1))
  done < <(grep -nE 'os\.Getenv\(|os\.LookupEnv\(' "$f" 2>/dev/null)
done < <(find backend/internal backend/cmd backend/tools -name '*.go' 2>/dev/null | grep -v _test)

# T7 锚支持：额外扫描路径（临时文件——新增读取点即时判定）
for extra in "$@"; do
  [ -f "$extra" ] || continue
  grep -qxF "$extra" "$ALLOWLIST" && continue
  while IFS= read -r ln; do
    echo "FAIL: $extra:$(echo "$ln" | cut -d: -f1) env 直读（业务配置唯一真相=PG；须登记 $ALLOWLIST）"
    fails=$((fails+1))
  done < <(grep -nE 'os\.Getenv\(|os\.LookupEnv\(' "$extra" 2>/dev/null)
done

if [ $fails -gt 0 ]; then
  echo "check-env-reads: $fails 处违规（env 只留引导件；业务旋钮→system_config，秘密件→platform_secrets）"
  exit 1
fi
echo "check-env-reads: OK（白名单 $(grep -cE '^backend/' "$ALLOWLIST") 文件；引导件外零 env 直读）"
