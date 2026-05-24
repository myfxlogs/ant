#!/usr/bin/env bash
# verify-cards-strict.sh — 机器化校验某 milestone 全部声明 ☑ 的卡片
# 用法：bash scripts/verify-cards-strict.sh M10
# 退出码：0 全部通过；非 0 = 有卡片未达 §0.3 3 条硬条件
set -euo pipefail

MS="${1:-M10}"
ROADMAP="docs/plan/ROADMAP.md"
HANDOVER_DIR="docs/handover"
LOG_MIN_LINES=30

if [[ ! -f "$ROADMAP" ]]; then
    echo "FATAL: $ROADMAP not found"; exit 2
fi

echo "==> verify-cards-strict: milestone=$MS"
echo "==> 扫描 $ROADMAP 中所有声明 ☑ 的 $MS 卡片"

# 从 ROADMAP 提取 "| M10.x-y | ☑ ..." 行，抽出 ID
mapfile -t CLAIMED < <(
    grep -oE "^\| ${MS}\.[0-9Z]+-[0-9]+ \| ☑" "$ROADMAP" \
        | awk '{print $2}'
)

if [[ ${#CLAIMED[@]} -eq 0 ]]; then
    echo "INFO: 没有声明 ☑ 的 $MS 卡片"; exit 0
fi

echo "==> 检测到 ${#CLAIMED[@]} 张声明 ☑ 的卡片"

FAIL_IDS=()
PASS_IDS=()

for CARD in "${CLAIMED[@]}"; do
    # CARD 形如 M10.1-1
    LOG="$HANDOVER_DIR/verify-${CARD}.log"
    FAIL_REASON=""

    # C1: commit 存在
    if ! git log --grep="Card: ${CARD}" --oneline 2>/dev/null | grep -q .; then
        FAIL_REASON="C1:no-commit"
    # C2: verify log 行数
    elif [[ ! -f "$LOG" ]]; then
        FAIL_REASON="C2:no-log"
    elif [[ $(wc -l < "$LOG") -lt $LOG_MIN_LINES ]]; then
        FAIL_REASON="C2:log-too-short($(wc -l < "$LOG"))"
    # C3: 真实 PASS 关键字
    elif ! grep -qE '^(ok |--- PASS|PASS$)' "$LOG"; then
        FAIL_REASON="C3:no-PASS-token"
    elif grep -q '\[no test files\]' "$LOG"; then
        FAIL_REASON="C3:contains-[no test files]"
    fi

    if [[ -n "$FAIL_REASON" ]]; then
        FAIL_IDS+=("$CARD:$FAIL_REASON")
        printf "  ✗ %-12s  %s\n" "$CARD" "$FAIL_REASON"
    else
        PASS_IDS+=("$CARD")
        printf "  ✓ %-12s  PASS\n" "$CARD"
    fi
done

echo
echo "==> 结果：PASS=${#PASS_IDS[@]}  FAIL=${#FAIL_IDS[@]}"

if [[ ${#FAIL_IDS[@]} -gt 0 ]]; then
    echo
    echo "==> 失败卡片应回退到 🅒"
    for entry in "${FAIL_IDS[@]}"; do
        echo "    $entry"
    done
    echo
    echo "==> 自动回退命令（dry-run，需 --apply 才真改）："
    if [[ "${2:-}" == "--apply" ]]; then
        for entry in "${FAIL_IDS[@]}"; do
            CARD="${entry%%:*}"
            sed -i "s|^\| ${CARD} \| ☑|\| ${CARD} \| 🅒|" "$ROADMAP"
            echo "    sed: ${CARD} ☑ → 🅒 in $ROADMAP"
        done
        echo "==> ROADMAP 已就地修改；请人工 review + commit"
    else
        for entry in "${FAIL_IDS[@]}"; do
            CARD="${entry%%:*}"
            echo "    sed -i 's|^\| ${CARD} \| ☑|\| ${CARD} \| 🅒|' $ROADMAP"
        done
        echo
        echo "（加 --apply 参数自动执行回退）"
    fi
    exit 1
fi

echo "==> 全部卡片通过 3 条硬条件"
exit 0
