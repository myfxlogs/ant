# ADR-0030 · 策略部署 UX 两步法（Configure → Confirm）

- **状态**：Accepted
- **日期**：2026-08-12
- **决策者**：人类负责人 × Windsurf（双角色 agent：审计+施工）
- **涉及功能块**：`strategy-gallery`（DeployScheduleModal）、`strategy-live`（LiveStrategyPage Schedules tab）
- **关联**：ADR-0029（购买→实盘执行链路）、ADR-0027（策略画廊重设计）

> 本 ADR 回答：**用户从策略画廊点 Deploy 后，到策略实盘运行之间，前端 UX 应该是怎样的？**

---

## 1. 背景

### 1.1 现状缺陷

`DeployScheduleModal`（StrategyCard Deploy 按钮）创建调度后：
- **原实现**：创建 `is_active=false` 的调度 → 显示成功 toast → 关闭弹窗。调度从未启用，用户以为已部署但 Live Monitor 看不到。
- **中间修复（已回退）**：创建后自动 `toggle(id, true)` → 跳过用户确认直接启动。
- **问题**：把"配置"和"启动"混在一个动作里，用户没有最终确认机会。

### 1.2 第性原则分析

策略部署涉及真金白银风险。交易系统中"配置"和"启动"是两个不同动作：

- **Deploy（配置）** = 填写参数（账号、品种、时间周期、调度类型）→ 创建调度记录
- **Enable（启动）** = 确认配置无误后，真正开始运行

**类比**：下单流程 — 先填订单参数，然后有"确认提交"的最终步骤。不会填完参数就自动提交。

### 1.3 约束

- ADR-0029 已定义后端执行链路：`ToggleSchedule(active=true)` → `StartSchedule` → `launchEventSession` → `RunLiveStrategy`。本 ADR 不改后端，只改前端 UX。
- entitlement/quota/bound-account 闸门在 `launchEventSession` 中执行（ADR-0029 决策 3/5）。前端不应绕过这些闸门。
- `CreateSchedule` 后端 handler 创建 `is_active=false` 记录，只调 `engine.Notify()`（不启动）。这是正确行为。

---

## 2. 架构决策

### 决策 1：两步法 — Configure → Confirm

Deploy 流程拆为两步：

1. **Configure（配置）**：`DeployScheduleModal` 弹窗 → 用户选参数 → 创建调度（`is_active=false`）→ 关闭弹窗 → 跳转到 `/strategy/live` Schedules tab
2. **Confirm（确认启动）**：Schedules tab 展示新创建的调度 → 用户审查配置 → 手动点 Enable → `ToggleSchedule(active=true)` → 后端启动会话

**理由**：
- **安全边界**：用户刚填完参数，跳转到监控页可看到完整配置，确认无误后再启用 — 交易系统标准实践
- **操作可见性**：用户被引导到正确页面，知道在哪里管理调度，不会"部署后找不到"
- **错误前置**：entitlement/quota 不满足时在 Enable 才报错，用户在正确页面看到错误并可处理
- **不增加多余摩擦**：用户只需多点一次 Enable，这是合理的安全确认

### 决策 2：不采用的备选

| 备选 | 否决理由 |
|------|---------|
| 创建后自动 toggle 启用 | 跳过用户确认，交易系统不可接受 |
| 创建后弹二次确认框 | 多一个 modal 层，不如跳转到真实页面有用（用户看不到调度列表上下文） |
| 创建后留在当前页 + toast | 用户不知道去哪管理，操作可见性差 |

### 决策 3：跳转时高亮新调度

跳转到 Schedules tab 时，新创建的调度行高亮闪烁 2 秒，引导用户注意力。通过 URL query param `?scheduleId=xxx` 传递，Schedules tab 读取后高亮。

**理由**：用户跳转后第一眼需要找到刚创建的调度，高亮比手动滚动查找更友好。

---

## 3. 实现

### 3.1 DeployScheduleModal

- 创建调度后不 toggle
- `onCreated` 回调改为 `navigate('/strategy/live?tab=schedules&scheduleId=<id>')`
- 成功 toast 保留（"调度已创建"）

### 3.2 LiveStrategyPage

- 读取 URL query `tab` 和 `scheduleId`
- `tab=schedules` → 默认选中 Schedules tab
- `scheduleId=xxx` → 传给 `LiveSchedulesTab` 用于高亮

### 3.3 LiveSchedulesTab

- 接收 `highlightScheduleId` prop
- 对应行高亮 2 秒后消失

---

## 4. 与其他 ADR 的关系

- **ADR-0029**：后端执行链路不变，本 ADR 只改前端 UX 触发方式
- **ADR-0027**：策略画廊的 Deploy 按钮入口不变，只改弹窗后的行为

---

## 5. 验收标准

- [ ] Deploy 弹窗创建调度后跳转到 `/strategy/live?tab=schedules&scheduleId=xxx`
- [ ] Schedules tab 默认选中
- [ ] 新调度行高亮 2 秒
- [ ] 调度 `is_active=false`，需用户手动 Enable
- [ ] Enable 后后端正常启动会话（ADR-0029 链路）
- [ ] entitlement denied 时 Schedules tab 显示 `last_error` 信息
