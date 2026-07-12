# AlphaForge — 项目总路线图

> **最后更新**：2026-07-11
> **状态**：M11 完成，M12 进行中

---

## 项目愿景

**"让普通人都能用上量化交易系统"** — 用户用自然语言描述交易想法，AI 自动生成策略代码，在回测/仿真/实盘环境中运行。

核心价值链：

```
行情数据 → 因子 → 信号 → 回测验证 → 仿真交易 → 风控 → 下单 → 持仓 → 复盘
                                                ↑
                                         AI 策略生成（自然语言 → 策略代码）
```

---

## 已完成里程碑

### M7-M10：基础设施 + 金融语义

- ✅ MT4/MT5 gRPC 适配层（mtapi.io 双层代理）
- ✅ 行情网关（tick dedup、CircuitBreaker、SpillWriter、ClickHouse 存储）
- ✅ 策略执行引擎（MQL → AST → Bytecode VM，328 个内置函数）
- ✅ 回测引擎（SimBroker，统一回测/实盘代码路径）
- ✅ 风控引擎（RiskGate，6 条预检规则 + Capability 模型）
- ✅ 订单状态机 + 持仓管理
- ✅ 仿真交易（Paper Trading）
- ✅ 多用户基础设施（user_id + account_id + broker 三维隔离）
- ✅ ConnectRPC + SSE 实时推送架构

### M11：MQL→VM 单一执行管线

- ✅ 清除旧解释器执行路径
- ✅ 清除 Go 代码生成路径
- ✅ VM 内置函数完善（328 + 24 MQL5 指标）
- ✅ 前端修复

### M12：Agent-Native 策略平台

- ✅ AI 策略生成 Agent（双编译前端：Python Agent 层 + Go VM 执行层）
- ✅ Agent 循环（观察 → 思考 → 行动 → 迭代）
- ✅ 多语言 Agent 提示词（en, zh-cn, zh-tw, ja, vi）
- ✅ Plan Mode 交互（先出方案 → 用户确认 → 生成代码）
- ✅ 策略工作区（Workspace + 代码编辑器 + 回测面板）
- ✅ 策略模板库 + 调度执行
- ✅ 实盘策略管理页（LiveStrategyPage + 信号 SSE 流）
- ✅ 策略市场（Marketplace）
- ✅ 自动交易设置
- ✅ 三层记忆系统（全局领域知识 + 用户手写 + Agent 自动写）
- ✅ 模型降级链 + Fail-closed 安全

### 品牌与增长基础设施

- ✅ AlphaForge 品牌改名（前后端全量）
- ✅ SEO 基础设施（prerendered HTML、sitemap、robots、og-image）
- ✅ Umami 自托管分析
- ✅ nginx 资产缓存优化

---

## 进行中 / 近期

### P0：上线准备 ✅ 已完成

| 任务 | 状态 | 说明 |
|------|------|------|
| Google Search Console | ✅ | 域名级验证通过，sitemap 已提交 |
| Cloudflare 缓存策略 | ✅ | index.html DYNAMIC，不被 CDN 缓存 |
| 域名切换 | ✅ | alfq.org 通过 Cloudflare 正常访问 |
| Umami 验证 | ✅ | /umami/script.js 正常返回 JS |
| OG image | ✅ | /og-image.svg 返回 image/svg+xml |
| robots.txt | ✅ | 正常返回 |
| sitemap.xml | ✅ | 正常返回，5 个 URL |
| SEO meta 标签 | ✅ | og/twitter/keywords 全部正确 |

### P1：用户体验完善

| 任务 | 优先级 | 说明 |
|------|--------|------|
| 策略分享页优化 | 高 | SharePerformancePage 性能图表渲染 + 移动端适配 |
| 前端 i18n 完整性审计 | 中 | 检查所有页面是否有未翻译的硬编码字符串 |
| 账户连接向导 UX | 中 | MT4/MT5 账户绑定的引导流程优化 |
| 错误提示友好化 | 中 | ConnectRPC 错误码 → 用户可读的多语言提示 |

### P2：策略能力扩展

| 任务 | 优先级 | 说明 |
|------|--------|------|
| Python 策略 SDK 完善 | 高 | Agent 生成的 Python 策略需要更完整的 SDK 支持 |
| 策略回测增强 | 中 | 参数优化、蒙特卡洛模拟、walk-forward 分析 |
| 多品种策略支持 | 中 | 当前策略单品种，扩展到多品种组合 |
| 策略版本管理 | 低 | Git-like 版本历史 + diff + 回滚 |

### P3：平台商业化

| 任务 | 优先级 | 说明 |
|------|--------|------|
| 用户计费体系 | 高 | AI token 用量计费 + 策略运行时长计费 |
| 订阅计划 | 高 | Free / Pro / Enterprise 分层 |
| 策略市场交易 | 中 | 策略创作者经济（付费策略 + 分成） |
| 用户注册流程完善 | 中 | 邮箱验证 + 欢迎引导 + 首策略创建向导 |
| 管理后台完善 | 中 | 用户管理 + 系统监控 + 运营数据看板 |

### P4：技术债务 & 可靠性

| 任务 | 优先级 | 说明 |
|------|--------|------|
| Go module path 重命名 | 低 | `anttrader/` → `alphaforge/`（影响大，需全量 import 重写） |
| Proto i18n 品牌同步 | 中 | proto 源文件中的 AntTrader → AlphaForge，重新 buf generate |
| Docker 容器名统一 | 低 | `ant-backend` → `alphaforge-backend` 等 |
| E2E 测试覆盖 | 中 | Playwright 回归测试（登录 + 策略生成 + 回测 + 实盘） |
| 监控告警 | 中 | Prometheus + AlertManager + 运营通知 |

---

## 技术架构概览

```
┌──────────────────────────────────────────────────────────────────┐
│  Frontend (React + Ant Design + ConnectRPC + SSE)                │
│  ┌──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┐      │
│  │ Auth │ Dash │ AI   │ Str  │ Live │ Mkt  │ Trade │ Admin│      │
│  └──────┴──────┴──────┴──────┴──────┴──────┴──────┴──────┘      │
└──────────────────────────┬───────────────────────────────────────┘
                           │ ConnectRPC + SSE
┌──────────────────────────┴───────────────────────────────────────┐
│  Backend (Go)                                                     │
│  ┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┐         │
│  │ AI  │ Str │ Mkt │Paper│ OMS │Risk │ Not │ Sys │User │         │
│  └─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┘         │
│  ┌──────────────────────────────────────────────────┐            │
│  │  MT4/MT5 Adapter (gRPC) │ MD Gateway │ MQL VM    │            │
│  └──────────────────────────────────────────────────┘            │
└───────┬──────────────┬──────────────┬──────────────┬─────────────┘
        │              │              │              │
   PostgreSQL     ClickHouse       Redis         NATS JS
   (业务数据)      (行情时序)      (缓存)        (事件流)
```

---

## 关键 ADR 索引

| ADR | 标题 | 状态 |
|-----|------|------|
| 0003 | 直连 mtapi 不封装 | Accepted |
| 0012 | 统一回测/实盘代码路径 | Accepted |
| 0022 | MQL 盲区架构 | Accepted |
| 0023 | AST 解释器 + MQL 源码为唯一真实来源 | Accepted |
| 0024 | Agent-Native 策略平台 | Accepted |
| 0025 | Agent UX 与自我进化 | Accepted |

P0 上线准备：Google Search Console、Cloudflare 缓存、域名切换（你手动）
P1 用户体验：分享页优化、i18n 审计、账户绑定 UX、错误提示
P2 策略能力：Python SDK 完善、回测增强、多品种、版本管理
P3 商业化：计费体系、订阅计划、策略市场交易、注册流程、管理后台
P4 技术债务：module path 重命名、proto 同步、E2E 测试、监控告警