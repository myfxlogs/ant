# strategy/workspace AI 语言与主模型持久化 — 修复记录与遗留架构问题

**日期**: 2026-06-16
**状态**: ✅ 热修复已完成（可编译、相关包测试通过）
**范围**: `strategy/workspace` 页 AI 聊天的两个用户可见缺陷
**性质**: 最小化、低风险热修复；底层设计味道仍在，见"遗留架构问题"

---

## 一、缺陷与根因

### 缺陷 1：AI 回复不跟随用户所选语言（英文界面下出现中文）

AI 聊天走两条后端管线，二者都未把用户语言传给 LLM：

- **策略生成管线**（`StrategyGenerationService`）
  - 系统提示词在 `backend/internal/ai/strategy_prompt.go:57` 硬编码为中文（"你是一位专业的量化交易策略工程师…"）。
  - handler 从不读取 `Accept-Language`。
  - 意图分析的澄清问题在 `backend/internal/ai/clarification.go` 规则 5 强制中文。
- **代码修订/讨论管线**（`CodeAssistService`）
  - 前端已发送 `locale`，但后端 `ai.BuildContext` 完全忽略它。

> 前端在每个 ConnectRPC 请求都注入 `Accept-Language`（`frontend/src/client/transport.ts:88-94`），因此无需改 proto。

### 缺陷 2：选中的主模型刷新后丢失

读写数据源不一致：

- `GetAIPrimary` **优先读 `users` 表**（`users.ai_primary_provider_id/model`，`backend/internal/connect/ai/ai_primary_handler.go:34`）。
- `SetAIPrimary` 只写 `system_ai_configs` 表，**从不写 `users` 表**（`backend/internal/service/systemai/service.go`）。
- 更关键：workspace 选择器包含**网关模型**（平台提供，用户无对应 `system_ai_configs` 行）。保存网关模型时 `UPDATE … WHERE provider_id=…` 匹配 0 行、事务静默提交、什么都没存 → 刷新后回到空。

---

## 二、已实施的修复

| 文件 | 改动 |
|------|------|
| `backend/internal/connect/ai/strategy_gen_handler.go` | 从 `Accept-Language` 解析语言，贯穿生成 / 反馈 / 意图分析三条路径，系统提示词后追加 `LangPrompt(lang)` |
| `backend/internal/connect/ai/lang.go` | 新增 `clarifyLangDirective`，强制澄清问题使用用户语言 |
| `backend/internal/ai/clarification.go` | `Analyze` 增加语言指令参数；移除"必须中文"约束 |
| `backend/internal/ai/prompt_context.go` | `BuildContextInput` 增加 `Locale`；新增 `localeDirective`；discuss 模式追加语言指令 |
| `backend/internal/connect/ai/code_assist_handler.go` | 两处 `BuildContext` 传入 `req.Msg.Locale` |
| `backend/internal/service/systemai/service.go` | `SetAIPrimary` 先写 `users` 表（权威，网关模型同样有效），`system_ai_configs` 降级为尽力而为 |

**验证**: `go build ./...` 通过；`go test ./internal/ai/... ./internal/connect/ai/...` 通过。

---

## 三、遗留架构问题（热修复未根治，建议后续重构）

### 主模型持久化

- **P1 双数据源**：`users.ai_primary_*` 与 `system_ai_configs.primary_for/default_model` 同存"主模型"。最优解是**单一权威源**（`users` 表，纯 provider+model 字符串，天然支持网关模型），废弃或将 `system_ai_configs.primary_for` 降为派生视图。
- **P2 静默失败仍在**：`system_ai_configs.SetAIPrimary` 匹配 0 行不报错的根问题未修，只是被绕过；理想上 0 行应可识别。
- **P3 前端 UX 不闭环**：inline 选择器 `onChange` 仅改本地 state，须再点 **Save**。更优为**选中即保存**（`onChange` 直接调 `savePrimary`），去掉单独 Save 按钮。位置：`frontend/src/pages/strategy/components/workspace/WorkspaceCodePanel.tsx`。
- **P4 可能的双选择器**：`AISettingsModal` 的 primary 选择器与 workspace inline 选择器若并存，需确认读写同一权威源。

### 语言处理

- **P1 两套传参机制并存**：策略生成走 `Accept-Language` header，code_assist 走 `req.Msg.Locale` 字段。前端已全局注入 header，应**统一用 header**，`ReviseCodeRequest.locale` 字段可视为冗余。
- **P2 四个语言函数、两套语言码空间**：`LangFromAccept` / `LangPrompt` / `clarifyLangDirective`（handler 包）+ `localeDirective`（ai 包）。header 归一化成 `zh-tw`，而 `localeDirective` 按前端 `zh-TW` 前缀匹配 → 两套 code space。最优：一个归一化函数 + 一个按场景（散文 / JSON 问题 / 代码注释）输出指令的 builder，集中一处。
- **P3 残留中文硬编码**：`backend/internal/ai/clarification.go:104,112` 的 fallback 问题（JSON 解析失败 / 低置信度）仍写死中文，未跟随语言。
- **P4 中文基底 prompt + 末尾英文指令覆盖**：依赖 LLM 服从 override，实践有效但不够稳健；更优是基底 prompt 语言中性化，或语言指令置于最强位置。

---

## 四、建议的后续重构方向（终态）

1. **主模型单一数据源 + 选中即存**：以 `users` 表为唯一权威；前端改为 `onChange` 即保存。
2. **语言模块收敛**：归一化与指令生成合并为单一共享模块，统一走 `Accept-Language` header；移除 `req.Msg.Locale` 字段依赖；fallback 问题接入语言。
