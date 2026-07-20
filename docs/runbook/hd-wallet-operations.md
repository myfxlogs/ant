# HD 钱包充值系统 · 运维手册

> 权威依据：`docs/adr/0026-hd-wallet-deposit-system.md` §12 运维风险缓解。
> 本手册为 operator 日常操作和应急响应的标准流程。

## 日常运维

### 日检（每 24h，建议 UTC 00:00 后，在对账完成后执行）

- [ ] **对账报告**：检查 §2.8 阶段 2 链上对账日志输出，确认 `diff = 0`（或 < 告警阈值）
- [ ] **MANUAL_REVIEW 积压**：管理端 `ListManualReviewDeposits`，处理过去 24h 未解决的条目
- [ ] **地址审计**：`coldsign` 导出最近 10 个 index 地址，与 DB `user_deposit_addresses` 比对（§12.1），确认一致
- [ ] **sweep_logs 异常**：查询 `sweep_logs WHERE status IN ('SWEEPING','MANUAL_REVIEW') AND updated_at < NOW() - INTERVAL '1h'`，确认无卡死
- [ ] **energy 价格**：记录 TRON Energy 当前均价，环比 > 30% 通知 administrator（§12.2）

### 周检（每 7d）

- [ ] **sweep_bundles 过期**：查询 `sweep_bundles WHERE status='BROADCASTING' AND built_at_ms + 24h < NOW()`（超过 24h 的 bundle 已过期，归集需重新冷签），手工处理
- [ ] **能量账户 TRX 余额**：查询能量账户链上 TRX 余额 + 已质押量，确认资源充足
- [ ] **xpub 全量校验**：`coldsign` 导出全部已派生地址与 DB 逐条比对（§2.4 地址审计），确认无篡改
- [ ] **校验器存活**：确认 `solvency-check` 过去一周有持续日志输出，无超过 6h 的空窗

---

## 归集操作 SOP

### 正常归集

1. 打开管理端归集看板（`ListUnsweptAddresses` 按链上余额降序）
2. 选择待归集地址 → 构建 `UnsignedSweepBundle` → 导出到 USB
3. USB → 气隙冷签名机 → `coldsign` 加载 → 逐笔核对（金额/源/目的地）→ 签名 → 输出 `SignedSweepBundle` → USB 回传
4. 在线机导入 `SignedSweepBundle` → 按序广播（delegate → transfer → undelegate）
5. 确认三腿全部 `DONE`（管理端查看 `sweep_logs` 状态）

### 归集卡死恢复

**症状**：`sweep_logs` 某腿停在 `SWEEPING` 超过 5 分钟。

**处理**：
1. 查链上该 `tx_hash` 最终状态（TronGrid `GetTransactionInfoByID`）
2. SUCCESS → 手动标记 `DONE`，继续下一腿
3. FAILED → 标记 `FAILED`，分析失败原因（energy 不足/合约拒绝），修复后从卡死的腿重新冷签
4. tx_hash 不存在（未广播）→ 检查 `sweep_bundles` 是否有对应 bundle：
   - 有 bundle → 读回续播（重广播不需私钥）
   - 无 bundle → 查该分地址链上出账历史，确认无成功转账后 → 标记 `FAILED` → 重新走冷签归集

### 孤儿 Energy 回收

**场景**：delegate 成功但 transfer 超过 24h 未执行（bundle 过期，已签 transaction 失效）。

**处理**：
1. 冷签名机：单独签一批 `UndelegateTx`（仅 energy_account → 分地址）→ USB → 在线机广播
2. 或等待 14 天自然解押（Stake 2.0 自动到期回收）

---

## 应急响应

### 冷签名机硬件故障

1. 定位 2 位 Shamir 分片保管人 → 当面或通过安全信道合并分片 → 在新气隙机上用 `hdgen` 从种子重建
2. `deposit_xpub` 不变（无需重派地址，无需迁移 DB），恢复签名能力
3. 测试：签一笔测试交易并验证 `expected_txid` 与链上回执一致
4. 若因原分片合并暴露，重新生成 Shamir 分片（3-of-2）并轮换

### 一枚 Shamir 分片丢失

1. 仍可正常操作（2-of-3，任意 2 枚可用）
2. 尽快在气隙机上重新生成 Shamir 分片（3 枚新分片），旧分片全部作废
3. 新分片分发给保管人

### xpub 运行时被篡改

1. 立即停止 `GetDepositAddress`（管理端开关或重启时跳过该 handler）
2. 从配置文件恢复 `deposit_xpub` 正确值
3. 执行地址审计：`coldsign` 导出全部地址，与 DB 逐条比对
4. 比对中发现不一致的行 → 该地址已被劫持 → 检查是否有充值 → 联系对应用户 → 归集该地址（如有资金）→ 退役该地址
5. 排查入侵范围（如何获得 root 的？改了多久？）

### MANUAL_REVIEW 批量出现（> 10 条/24h）

1. 检查 TronGrid 和 TronScan API 连通性（可能一侧故障导致多源验证全部失败）
2. 若 API 正常 → 逐笔人工在 TronScan 浏览器上核对 `tx_hash`，按 §11 Q10 SOP 处理
3. 若同一 `tx_hash` 反复出现 → 检查 `deposits` 表唯一约束是否正常

### 偿付能力校验器告警

1. `liability > custody` → 先查是否对账正常（排除 DB 统计误差）→ 再查链上冷钱包/分地址余额 → 若差额 > 阈值持续存在 → 可能存在账本篡改 → 从链重建内部账本（`wallet_transactions` 哈希链回溯定位断层点）
2. `entry_hash` 链断裂 → 确认篡改事件 → 从离机镜像定位篡改时间点 → 从链上重建该时间点之后的真实流水
3. 死人开关触发（校验器超过 N 小时无输出）→ 检查校验器是否在运行 → 检查在线机是否可用 → 若在线机被切断 → 独立查链判断冷钱包是否有异常资金移动

---

## 配置参考

| 配置项 | 值 | 说明 |
|--------|-----|------|
| 归集建议频率 | 每天 1-2 次 | 按余额阈值触发，不按时间 |
| 地址审计频率 | 每 24h | 日检时执行 |
| xpub 配置文件路径 | `config.yaml` 或环境变量 `DEPOSIT_XPUB` | 非 `system_config`（§12.1） |
| 校验器 cron 间隔 | 每 2h | 建议 `0 */2 * * *` |
| Energy 价格告警环比阈值 | +30% | 触发补质押评估 |
| MANUAL_REVIEW 积压告警 | > 0 条且 > 24h | 每 6h 对账附带检查 |
| Sweep bundle 过期时间 | 24h | 超过需重新冷签 |
