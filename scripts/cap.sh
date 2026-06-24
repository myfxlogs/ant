#!/usr/bin/env bash
# cap.sh <keyword> — 查能力清单（Reuse Preflight 唯一许可的查询方式）。
#
# 只返回命中行，token 有上界。**禁止整篇 Read docs/CAPABILITIES.md。**
# 用法：bash scripts/cap.sh modify        # 查"改单"相关已实现能力
#       bash scripts/cap.sh ModifyOrder   # 按符号名查
#       bash scripts/cap.sh close          # 按动词/别名查
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAP="$ROOT/docs/CAPABILITIES.md"

kw="${1:-}"
if [ -z "$kw" ]; then
  echo "usage: bash scripts/cap.sh <动词/别名/符号>"
  echo "  例：bash scripts/cap.sh modify | close | OrderSend | equity | gate"
  exit 2
fi

# 清单不存在或可能过时时先刷新（廉价：纯 grep 生成）。
if [ ! -f "$MAP" ]; then
  bash "$ROOT/scripts/gen_capability_map.sh" >/dev/null
fi

echo "== CAPABILITIES.md 命中（关键词：$kw）=="
if ! grep -in -- "$kw" "$MAP"; then
  echo "（清单无命中）"
fi

echo ""
echo "== 代码层命中（权威真相，关键词：$kw）=="
# 直接 grep 高风险域源码，作为清单的兜底校验（防清单滞后）。
code_hits="$(grep -rEin -- "$kw" \
  "$ROOT/backend/internal/mthub" \
  "$ROOT/backend/internal/risk" \
  "$ROOT/strategy-service/app/sdk" \
  "$ROOT/strategy-service/app/engine" \
  --include='*.go' --include='*.py' 2>/dev/null \
  | grep -v '_test.go' \
  | grep -iE "func |def |class |rpc " \
  | sed -E "s#^$ROOT/##" \
  | head -n 40 || true)"
if [ -n "$code_hits" ]; then
  printf '%s\n' "$code_hits"
else
  echo "（代码层无命中 → 大概率确为 NEW）"
fi
