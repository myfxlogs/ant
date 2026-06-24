#!/usr/bin/env bash
# gen_capability_map.sh — 从代码库自动重生成 docs/CAPABILITIES.md 的"自动清单"部分。
#
# 目的：让执行者（DeepSeek/Cascade）动工前能查到"什么已实现"，避免重复造轮子。
# 设计：脚本只重写哨兵注释以下的"自动生成"部分；哨兵以上的「人工策展表 + 别名索引」保留。
# 用法：在仓库任意位置运行 `bash scripts/gen_capability_map.sh`（每个 Task / Phase 开工前跑）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/docs/CAPABILITIES.md"
SENTINEL="<!-- AUTOGEN-BELOW: 由 scripts/gen_capability_map.sh 重生成，勿手工编辑以下内容 -->"

# 重复造轮子风险最高的域。按需扩展。
GO_DIRS=(
  backend/internal/mthub
  backend/internal/risk
  backend/internal/risksvc
  backend/internal/paper
  backend/internal/service
  backend/internal/connect/strategy
)
PY_DIRS=(
  strategy-service/app/sdk
  strategy-service/app/engine
)

emit() { printf '%s\n' "$1"; }

gen() {
  emit "$SENTINEL"
  emit ""
  emit "_最后生成：$(date -u '+%Y-%m-%d %H:%M UTC')。运行 \`bash scripts/gen_capability_map.sh\` 刷新。_"
  emit ""

  emit "## 公开 ConnectRPC（前端/跨服务可直接调用）"
  emit ""
  emit '```'
  grep -rEn "rpc [A-Za-z0-9_]+ ?\(" "$ROOT/proto/ant/v1" 2>/dev/null \
    | sed -E "s#$ROOT/##" | sort || true
  emit '```'
  emit ""

  emit "## MT 网关原始 RPC（mtapi 层，经 executor/adapter 暴露）"
  emit ""
  emit '```'
  grep -rEn "rpc [A-Za-z0-9_]+ ?\(" \
    "$ROOT/reference/grpc/mt4.proto" "$ROOT/reference/grpc/mt5.proto" 2>/dev/null \
    | sed -E "s#$ROOT/##" | sort || true
  emit '```'
  emit ""

  emit "## Go 服务方法（已实现的后端能力）"
  emit ""
  emit '```'
  for d in "${GO_DIRS[@]}"; do
    [ -d "$ROOT/$d" ] && grep -rEn "func \([a-z]+ \*[A-Za-z0-9_]+\) [A-Z][A-Za-z0-9_]*\(" \
      "$ROOT/$d" --include='*.go' 2>/dev/null \
      | grep -v '_test.go' | sed -E "s#$ROOT/##" || true
  done | sort
  emit '```'
  emit ""

  emit "## Python SDK / 引擎（已实现的类与函数）"
  emit ""
  emit '```'
  for d in "${PY_DIRS[@]}"; do
    [ -d "$ROOT/$d" ] && grep -rEn "^[[:space:]]*(def|class) [A-Za-z_]" \
      "$ROOT/$d" --include='*.py' 2>/dev/null \
      | grep -v '__pycache__' | sed -E "s#$ROOT/##" || true
  done | sort
  emit '```'
  emit ""

  emit "## Handler 注册（已接进生产路由 = 非 shelf-ware）"
  emit ""
  emit "> 在此列表 = 真正可被调用；只在某 *_test.go 出现而不在此 = 货架闲置（shelf-ware）。"
  emit ""
  emit '```'
  grep -rEn "mux.Handle\(antv1c.New[A-Za-z0-9_]+Handler" \
    "$ROOT/backend/cmd" --include='*.go' 2>/dev/null \
    | sed -E "s#$ROOT/##" | sort || true
  emit '```'
}

# 保留哨兵以上的人工策展内容；重写哨兵及以下。
if [ -f "$OUT" ] && grep -qF "$SENTINEL" "$OUT"; then
  line="$(grep -nF "$SENTINEL" "$OUT" | head -1 | cut -d: -f1)"
  head -n "$((line - 1))" "$OUT" > "$OUT.tmp"
else
  [ -f "$OUT" ] && cat "$OUT" > "$OUT.tmp" || : > "$OUT.tmp"
fi
gen >> "$OUT.tmp"
mv "$OUT.tmp" "$OUT"
echo "Wrote $OUT"
