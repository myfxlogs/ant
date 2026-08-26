#!/usr/bin/env bash
# 一次性环境初始化 —— 新克隆/新工作副本首次启动时执行
# 启用 pre-commit 门禁（git config core.hooksPath 不会随 push 传播，需本地设置）
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

# 1. 启用 pre-commit hook
git config core.hooksPath scripts/hooks
chmod +x scripts/hooks/pre-commit scripts/check-lines.sh 2>/dev/null || true
echo "✅ pre-commit hook 已启用（core.hooksPath = scripts/hooks）"

# 2. 验证 hook 可执行
if bash scripts/hooks/pre-commit 2>&1; then
  echo "✅ pre-commit hook 自检通过"
else
  echo "⚠️  pre-commit hook 自检未通过（可能有未更新的 STATE.md 或超预算文档）"
fi

echo ""
echo "下一步：读 AGENTS.md → docs/handoff/STATE.md → 按施工表开工"
