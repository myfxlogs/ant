#!/usr/bin/env bash
# verify-adversarial.sh — 对抗证明自动化校验（审计方/施工方共用）。
#
# 解决"对抗证明无效"复发模式：施工方写的对抗测试常"测拷贝/字面量/错路径"，
# 删掉修复行测试仍绿。本脚本自动做审计方一直在做的"删行→必红"检查。
#
# 原理：对生产代码施加一个突变（删行/改条件），跑指定测试，
#   测试 FAIL = ✅ RED（对抗证明有效，真守住了修复）；
#   测试 PASS = ❌ GREEN（对抗证明无效，测试没测到真代码——要求补强）。
# 然后自动还原文件（用备份，不依赖 git，对未提交的工作树也安全）。
#
# 用法：
#   verify-adversarial.sh <test-pattern> <pkg> <file> <sed-mutation>
#
# 示例（验 LIVE-PRICE-1：删 onTick 调用，测试应变 RED）：
#   verify-adversarial.sh TestOnTickCallbackFired ./internal/mdgateway/ \
#     internal/mdgateway/manager_tick.go '/m\.onTick(t)/d'
#
# 示例（验 LIVE-PRICE-3：禁用 GetError 检查，测试应变 RED——实际仍 GREEN=无效）：
#   verify-adversarial.sh TestSubscribeMany_ResponseErrorDetected \
#     ./internal/mdgateway/adapter/mt4/ internal/mdgateway/adapter/mt4/quotes.go \
#     's/e != nil && e\.GetCode() != 0/e != nil \&\& false/'
#
# 退出码：0=对抗证明有效(RED)；1=无效(GREEN，需补强)；2=参数/运行错误。
set -uo pipefail

if [ "$#" -ne 4 ]; then
  echo "用法: $0 <test-pattern> <pkg> <file> <sed-mutation>" >&2
  echo "  test-pattern: go test -run 的模式（如 TestOnTickCallbackFired）" >&2
  echo "  pkg:          go 包路径（如 ./internal/mdgateway/）" >&2
  echo "  file:         要施加突变的源文件（相对 backend/ 或绝对）" >&2
  echo "  sed-mutation: sed 脚本（如 '/m\\.tonTick(t)/d' 删行，或 's/cond/false/' 改条件）" >&2
  exit 2
fi

TEST="$1"; PKG="$2"; FILE="$3"; MUTATION="$4"
BAK="${FILE}.adversarial.bak"

[ -f "$FILE" ] || { echo "❌ 文件不存在: $FILE" >&2; exit 2; }

cp "$FILE" "$BAK" || { echo "❌ 备份失败" >&2; exit 2; }
trap 'cp "$BAK" "$FILE" 2>/dev/null; rm -f "$BAK"' EXIT

echo "▶ 施加突变: sed -i \"$MUTATION\" $FILE"
if ! sed -i.bak2 "$MUTATION" "$FILE"; then
  rm -f "${FILE}.bak2"
  echo "❌ sed 突变失败（检查 sed 脚本语法）" >&2; exit 2
fi
rm -f "${FILE}.bak2"

# 确认突变真的改了文件（diff 非空），否则突变无效
if diff -q "$BAK" "$FILE" >/dev/null 2>&1; then
  echo "⚠️  突变未改动文件（sed 模式没匹配到）——检查模式是否正确。仍跑测试。" >&2
else
  echo "✓ 文件已改动（$(diff "$BAK" "$FILE" | grep -c '^[<>]') 行变化）"
fi

echo "▶ 跑测试: go test -run '$TEST' $PKG （突变状态下）"
echo "------------------------------------------------------------"
# 定位 go.mod 目录（从 FILE 所在路径向上找），go test 必须在模块根跑
FILE_ABS="$(cd "$(dirname "$FILE")" && pwd)/$(basename "$FILE")"
GOROOT_DIR="$(dirname "$FILE_ABS")"
while [ "$GOROOT_DIR" != "/" ] && [ ! -f "$GOROOT_DIR/go.mod" ]; do
  GOROOT_DIR="$(dirname "$GOROOT_DIR")"
done
if [ ! -f "$GOROOT_DIR/go.mod" ]; then
  echo "❌ 找不到 go.mod（从 $FILE 向上）" >&2; exit 2
fi
echo "（模块根: $GOROOT_DIR）"
GO_TEST_OUT=$(cd "$GOROOT_DIR" && go test -run "$TEST" -timeout 60s "$PKG" 2>&1)
RC=$?
echo "$GO_TEST_OUT" | tail -20
echo "------------------------------------------------------------"

# go test 退出码 0=PASS, 1=FAIL(编译失败也可能是非0), 其它=错误
if echo "$GO_TEST_OUT" | grep -qE "^(ok|--- PASS)"; then
  echo ""
  echo "❌ GREEN（测试在突变下仍 PASS）= 对抗证明【无效】"
  echo "   删/改修复行后测试没红 → 它没测到真代码路径。要求补强测试。"
  exit 1
elif echo "$GO_TEST_OUT" | grep -qE "^(FAIL|--- FAIL|panic)"; then
  echo ""
  echo "✅ RED（测试在突变下 FAIL）= 对抗证明【有效】"
  echo "   删/改修复行后测试红了 → 它真守住了修复。"
  exit 0
else
  echo ""
  echo "⚠️  无法判定（非 PASS 也非标准 FAIL，可能是编译错误/无测试）。检查输出。" >&2
  exit 2
fi
