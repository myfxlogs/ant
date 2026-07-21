# backtest-engine 真实手续费 · 施工清单

> 来源：`docs/audits/foundation-backtest-engine.md` 黄标
> 依赖：mt-gateway `SymbolParams` RPC 封装（P1）完成后才可实施

- [ ] **C1** 从 mt-gateway `SymbolParams` 获取品种的真实手续费配置（替代当前固定费率）
- [ ] **C2** SimBroker 使用真实手续费计算每笔交易成本
- [ ] **C3** 文档声明：回测手续费 = broker 当前费率（非历史费率），可能与实际成交有微小差异
- [ ] **验收**：同一策略在 SimBroker 和 MT Strategy Tester 的手续费误差 <5%
