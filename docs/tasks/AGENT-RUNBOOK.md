# AGENT-RUNBOOK — ant 执行入口

> 最后更新：2026-05-23（M0.2） · 读完全文约 10 分钟

## 阅读顺序

新 Agent 进仓库按以下顺序建立认知（约 30-45 分钟）：

| 顺序 | 文档 | 时间 | 目标 |
|------|------|------|------|
| 1 | `AGENT.md` | 15min | 理解硬性规则、三域结构、复杂度上限、防偷懒 7 条 |
| 2 | `docs/plan/ROADMAP.md` | 10min | 理解 M0.0-M6 里程碑、退出标准、依赖关系 |
| 3 | `docs/adr/README.md` | 5min | 了解 12 篇 ADR 决策主题与实施里程碑 |
| 4 | 本文 | 10min | 理解执行循环、验收标准、卡住升级路径 |

## 执行循环

### 日常工作流

```
1. 打开 ROADMAP.md → 找到当前里程碑（第一个非 ✅ 的）
2. 读取里程碑下第一张未完成的卡片
3. 对照 AGENT.md §防偷懒约束逐条执行
4. 完成后产出 commit + verify.log
5. 卡片标 ☑，更新 ROADMAP 状态
6. 重复 2-5 直到里程碑全部 ☑
7. 里程碑退出标准全部满足 → 标 ✅ → 进入下一里程碑
```

### 核心约束

- **不跳步**：上一里程碑未拿到 ✅ 不开下一
- **每卡片三齐全**：代码 + 测试 + 实测 stdout
- **禁止 mock**：验收必须真实容器/真实 PG/Redis/broker
- **卡片状态机**：commit body 必须含 `Verify: docs/handover/RS-final-verify.log:<行号>`

## 卡片执行检查清单

每张卡片开始前：

- [ ] 理解 ROADMAP 中该卡片的验收口径
- [ ] 确认关联 ADR 状态为 Accepted
- [ ] 确认关联 DESIGN-REVIEW finding（如有）

每张卡片完成后：

- [ ] commit conventional format + body 含 Verify 行
- [ ] `docs/handover/RS-final-verify.log` 追加 ≥20 行验收日志
- [ ] 防偷懒 7 条自检全过
- [ ] ROADMAP 卡片状态更新

## 卡住时怎么升级

### 阻塞分类

| 阻塞类型 | 识别特征 | 处理方式 |
|----------|----------|----------|
| **依赖缺失** | 需要的服务/账户/凭证不存在 | 标记卡片 🅒 + `BLOCKED: <原因>`，汇报给用户 |
| **ADR 未拍板** | 卡片涉及的决策在 Proposed 状态 | 停下汇报，附推荐方向 |
| **技术不可行** | 尝试 ≥2 种方案均失败 | 标记 🅒 + 详细记录尝试历史 |
| **工程量超预期** | 单卡片 > 2 天 | 汇报拆卡建议 |

### 升级路径

```
Agent 自身受阻 → 检查 AGENT.md 是否有遗漏约束
             → 检查对应 ADR 是否已 Accepted
             → 检查 DESIGN-REVIEW 相关 finding
             → 回退上一步、尝试替代方案
             → 记录详细尝试历史 → 汇报给用户
```

## 自检脚本

每轮提交前跑（来自 AGENT.md §防偷懒约束第 7 条）：

```bash
# 防 mock 检查
grep -rE '\b(mock|stub|[Ff]ake[A-Z])' backend/internal/ backend/cmd/ --include='*.go' \
  | grep -vE '_test\.go|/testutil/|fixtures/' && echo "FAIL" || echo "PASS"

# 测试净增减
# git diff --numstat origin/main...HEAD -- '*_test.go' 'test_*.py' | awk '{add+=$1;del+=$2} END{print add,del}'
```

## 关键文档清单

| 文档 | 路径 | 频率 |
|------|------|------|
| 硬性规则 | `AGENT.md` | 每次会话 |
| 路线图 | `docs/plan/ROADMAP.md` | 每卡片前 |
| ADR 索引 | `docs/adr/README.md` | 每里程碑 |
| 设计审查 | `docs/audit/DESIGN-REVIEW-2026-05.md` | 卡片关联到 finding 时 |
| 验收日志 | `docs/handover/RS-final-verify.log` | 每卡片后 |
| AlfQ 蓝本 | `docs/AlfQ功能迁移计划.md` | M1-M6 参考 |

## 项目宪法速查

- **协议**：Connect RPC + SSE。禁 REST 新接口、禁 WebSocket
- **数据**：PG 18 + Redis 8
- **代码生成**：buf (proto) / sqlc (SQL)
- **错误**：`internal/errs/` 包，禁裸字符串
- **日志**：zap 结构化 JSON，必带 `trace_id` `user_id`
- **安全**：用户 Python 代码仅在 strategy-service 沙箱执行
- **部署**：单机 docker-compose，5 容器，ant-* 前缀
- **复杂度**：Go ≤300 行/文件、TS ≤250 行/文件、函数 ≤50 行
- **依赖**：新增依赖走 ADR 立项，禁 AGPL
