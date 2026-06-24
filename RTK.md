@CLAUDE.md

# Reuse Preflight (避免重复造轮子 — 强制)

动工任何**新 file/function** 前必须执行复用核对：

1. 用 `bash scripts/cap.sh <动词/别名/符号>` 查能力（自动刷新 + 只返回命中行 + 代码层兜底）。**禁止整篇 Read `docs/CAPABILITIES.md`**（浪费 token）。
2. 按**动词 + 别名**（见该文件「动词/别名索引」）多换几个词查，确认是否已有现成能力。
3. 在 PR 描述里逐条给结论，二选一：`REUSE: <symbol> @ <file:line>`（复用现成）或 `NEW: 无现成能力（已搜：<关键词>）`（确认真空白）。
4. 发现能力状态/注释过时（注释说"不存在"但其实已实现）→ 同步修正注释与 `docs/CAPABILITIES.md`。

**缺少 `REUSE:`/`NEW:` 引用 = 该任务判失败。** 既防重复造轮子，也防误判 shelf-ware（分层状态：gateway-rpc / executor / mthub-method / connect-rpc / wired-live）。
