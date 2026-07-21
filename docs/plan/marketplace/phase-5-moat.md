> ⚠️ 已迁移至 docs/blocks/strategy-marketplace/plans/phase-5-moat.md。此文件保留为兼容旧引用。

# Phase 5 · AI 供给侧持续 + 增值 · 落地排期清单

> 权威依据：`docs/roadmaps/strategy-marketplace.md` Phase 5 (v4)
> 前置条件：Phase 1-4 完成，Phase 4.4（版本管理）完成，用户量达临界质量
> 优先级：🟢 P3
>
> **v4 变更**：
> - 新增 5.1 AI 策略迭代更新（对抗 R-A Alpha 衰减的系统性手段）
> - 删除跟单 UI（按产品边界：不做跟单服务）
> - 删除白标/API（按产品边界：不和 broker 绑定、不做白标）

## 0. 目标与边界

- **达成即最优**：策略不会因为 alpha 衰减而变成废品——AI 持续监控、自动优化、迭代更新。这才是区别于 MQL5 Market（卖死的 EA）的核心能力。
- **时机判断**：月活跃买方 >1000、上架策略 >200、有足够实盘数据训练迭代模型时启动。

---

## 依赖图

```
5.1 (AI 迭代更新) ──── 依赖 Phase 4.4 (版本管理) + Phase 1.1 (实盘数据)
5.2 (策略捆绑包)   ──── 独立
5.3 (阶梯费率)     ──── 独立
```

---

## 模块 5.1 · AI 策略迭代更新

**Why**: Alpha 衰减是策略市场的结构性风险。MQL5 Market 的 EA 是死的——买完就是那个版本，随时间贬值。我们能做到"策略自我进化"——这是供给侧对提供者和买方的核心卖点。

**架构**：

```
实盘监控（已有 Phase 1.1 数据）
      │
      ▼
衰减检测: 夏普连续下滑 N 周 / 回撤突破阈值 / 收益率趋势逆转
      │
      ▼
AI 优化: agent-engine 拿原始策略 + 近期行情 → 生成优化版本
      │
      ▼
验证: walk-forward/purged CV（复用 1.2 门槛逻辑）
      │
      ▼
提供者 Dashboard: "策略 EURUSD_H1_MA 检测到衰减，AI 已生成 v1.3.0 优化版，回测夏普 1.8→2.3"
      │
      ▼
提供者一键发布 → 订阅者收到通知 → 可升级到新版本
```

- [ ] **5.1a 衰减检测规则**

  **文件**：`backend/internal/marketplace/decay_detector.go` (new)

  **配置**（`system_config` 表）：
  ```sql
  INSERT INTO system_config (key, value, value_type, description, enabled) VALUES
    ('marketplace.decay.sharpe_decline_threshold', '0.3', 'decimal', '夏普下滑超过此值触发衰减告警', true),
    ('marketplace.decay.drawdown_breach_pct',   '0.25', 'decimal', '实盘回撤突破此值触发告警', true),
    ('marketplace.decay.lookback_weeks',        '4',    'int',    '衰减检测回溯周数', true),
    ('marketplace.decay.min_live_days',         '30',   'int',    '最少实盘天数才启动检测', true);
  ```

  **逻辑**：定时（惰性触发——提供者查看 Dashboard 时执行）扫描所有有实盘数据的已上架策略，计算滚动窗口夏普/回撤趋势，标记 `decay_status`（healthy / warning / critical）。

- [ ] **5.1b AI 优化生成**

  **文件**：`backend/internal/marketplace/strategy_optimizer.go` (new)

  **逻辑**：
  1. 衰减策略加入优化队列（`strategy_optimization_tasks` 表）
  2. 调用 agent-engine：输入原始策略源码 + 近期行情特征 + 衰减指标 → 生成优化版本
  3. mql-compiler 编译 → backtest-engine 回测 → walk-forward 验证
  4. 优化结果写入 `marketplace_strategy_versions`（复用 Phase 4.4 版本管理）

  **数据模型**：
  ```sql
  CREATE TABLE strategy_optimization_tasks (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id),
      decay_metrics JSONB NOT NULL,          -- 触发时的衰减指标快照
      optimized_code TEXT,                    -- AI 生成的优化源码
      backtest_snapshot BYTEA,                -- 优化版本的回测
      improvement_score NUMERIC(5,2),         -- 综合改善分（0-100）
      status VARCHAR(20) DEFAULT 'pending',   -- pending / optimizing / completed / rejected
      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```

- [ ] **5.1c 提供者通知 + 一键发布**

  **文件**：`backend/internal/connect/marketplace/marketplace_handler_optimization.go` (new)

  **RPC**：
  - `ListOptimizationSuggestions(strategy_id)` → 返回 AI 优化建议列表
  - `PreviewOptimization(task_id)` → 返回优化版本的回测对比（原版 vs 优化版）
  - `PublishOptimization(task_id, changelog)` → 发布为新版本（复用 `UpdateStrategyVersion`）

  **前端**：Author Dashboard 新增"优化建议"Tab——黄色 warning 标记 + "AI 已生成优化版" + 回测对比表格 + "发布"按钮。

- [ ] **5.1d 订阅者升级通知**

  订阅者收到通知（复用 Phase 3.4 通知系统）："你订阅的策略 EURUSD_H1_MA 发布了 v1.3.0 优化版"。策略详情页"版本历史"区域可查看新旧版本回测对比，自主选择升级。

- **Gate 5.1**：`go build ./...` + `buf generate` + 衰减检测 + 优化生成 + 版本发布完整链路测试。

---

## 模块 5.2 · 策略捆绑包

> 内容与 v1 相同，略。详见原 Phase 5 文档（5.1 策略捆绑包部分）。

---

## 模块 5.3 · 阶梯费率

> 内容与 v1 相同，略。详见原 Phase 5 文档（5.3 阶梯费率部分）。

---

## Phase 5 完成检验

```bash
go build ./...
buf generate
go test ./internal/marketplace/...
cd frontend && npm run build
bash scripts/gen_capability_map.sh
```

**关键验收场景**：
1. 策略实盘夏普连续下滑 4 周 → Dashboard 出现衰减告警
2. AI 自动生成优化版本 → 回测对比原版改善 → 提供者一键发布
3. 订阅者收到新版本通知 → 查看回测对比 → 升级
4. 捆绑包购买 → N 个策略同时订阅
5. 提供者月度收入达标 → 等级自动升级 → 新购买使用更低平台费率
