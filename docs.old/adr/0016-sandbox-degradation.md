# ADR-0016: 沙箱降级

**Status**: Accepted  
**Date**: 2026-05-23  
**Deciders**: 架构决策  
**Supersedes**: ADR-0004（Python 沙箱进程模型）  
**关联**: `strategy-service/app/engine/sandbox.py` · ADR-0013 · ADR-0015

## 背景

ADR-0004 确立了 Python 沙箱的进程模型（RestrictedPython + AST 白名单 + fork-per-execution）。但随着 M7 引入 DSL + ONNX 替代方案，沙箱在**生产路径**的风险已超过收益：

1. **安全攻面持续暴露**：RestrictedPython 历史上多次被绕过（CVE-2017-1000158 等），蚁穴式防御无法彻底杜绝
2. **GIL 阻塞**：多策略并发时互相阻塞，fork-per-execution 资源开销大
3. **漂移风险**：沙箱的 `indicators.py` 算子与 DSL 引擎算子不是同一实现
4. **已被替代**：DSL 覆盖 80%+ 策略场景，ONNX 覆盖 ML 场景

## 决策

**沙箱保留，但降级为研究专用。生产路径完全切断 Python 代码执行。**

### 具体措施

1. **`sandbox.py` 加 `production_mode` 全局锁**
   - `set_production_mode(True)` → 所有 `StrategyRunner.__init__` 抛出 `SandboxBlockedError`
   - 一经设置不可回退（进程生命周期内）
   - strategy-service 启动时根据环境变量 `PRODUCTION_MODE=true` 调用

2. **路由分裂**
   - `/research/*` → 沙箱可用（notebook、本地回测、AI 实验）
   - `/production/*` → 仅接受 DSL 表达式或 model_id 引用

3. **AI 生成策略输出切换**
   - `ai_strategy_validator.go` 输出从 Python code → DSL 字符串 + 可选的 ONNX URI
   - 不再生成任何 Python 代码

4. **老策略迁移工具**
   - `migrate_legacy_strategy.py` 自动转换可迁移的 Python 策略到 DSL
   - 不可自动转换的标记为 `unconvertible`，提示用户手动重写

### 保留原因

沙箱在研究环境仍有价值：
- Jupyter notebook 交互式回测
- DSL/ONNX 模型开发前的快速原型
- 用户本地环境的策略调试

但**不在生产资金路径上运行任何用户 Python 代码**。

## 后果

### 正面

- 生产安全攻面降为零（无 Python 执行路径）
- 策略执行性能提升（Go DSL 引擎 vs Python 沙箱）
- 策略热更新（DSL 表达式 / ONNX 模型可动态加载）
- 回测/实盘一致（同一套算子实现）

### 负面

- 图灵完备策略不可用于生产（需重写为 DSL/ONNX）
- 部分老用户策略需迁移
- 研究环境和生产环境的策略表达方式不同（学习成本）

### 风险缓解

- `migrate_legacy_strategy.py` 降低迁移成本（目标 50% 自动转换）
- DSL 算子持续扩展，覆盖更多策略场景
- 研究环境保持沙箱可用，降低学习摩擦

## 验证

- `grep -r "StrategyRunner\|sandbox\|exec(" backend/internal/connect/ backend/internal/service/` — 生产路径零结果
- `make test` 通过（包括 sandbox production_mode 测试）
- 实盘下单链路 `grep` 无 sandbox 调用

## 回退方案

如果 DSL/ONNX 无法覆盖关键策略需求，可临时降级：
- `set_production_mode(False)` 重新开启沙箱（需重启 + 人工审批）
- 同时在对应 issue 中记录未覆盖的策略类型，推动 DSL 扩展

## 备选方案

| 方案 | 优点 | 缺点 | 决定 |
|------|------|------|------|
| **沙箱降级（选中）** | 生产零风险、已有替代方案 | 需迁移老策略 | ✅ |
| 删除沙箱 | 彻底干净 | 失去研究环境、太激进 | ❌ |
| 保持沙箱 | 兼容现有策略 | 安全风险、性能差 | ❌ |
| Docker 隔离沙箱 | 更强隔离 | 延迟高、运维复杂 | ❌ |
