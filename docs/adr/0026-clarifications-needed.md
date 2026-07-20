# ADR-0026 设计审计 — 待 Opus 澄清问题

> 审计日期：2026-07-19
> 审计范围：ADR 文档内在设计质量（非代码对比）
> 目标：逐条获得 Opus 的确认/修正/拒绝后再进入 Phase A 实现
>
> **✅ 已全部定案（2026-07-19）**：Q1–Q12 权威结论见 `0026-hd-wallet-deposit-system.md` **§11**。
> 摘要：Q1=A(xpub 按需派生) · Q2=USB导出+R12治理(否决裸TOFU) · Q3=持久化已签bundle+三腿状态+长过期 ·
> Q4/Q5=接受并记录于§8.2 · Q6=enum · Q7=oneof · Q8=WithdrawalAuth(nonce) · Q9=已修正§2.8 ·
> Q10=SOP框架入§11 · Q11=DR要点入§11+独立runbook · Q12=阈值刻意不对称。下方原始问题仅存档。

---

## 🔴 阻塞实现的问题（需决策后才可动工）

### Q1. 地址派生：地址池预导入 vs xpub 按需派生，选哪个？

**问题**：§2.3 描述了两套方案并存（"在线机用 xpub 公开派生，或由冷签名机离线预派生列表导入 DB"），§10.1 的代码示例按需派生，§5.4 的 `hdgen` 工具又是为预导入设计的。

两套方案的取舍：

| 维度 | A. xpub 按需派生 | B. 冷机预导入地址池 |
|------|-----------------|-------------------|
| 运维负担 | 无（永远按需） | 需监控池余量、定期补充 |
| 代码复杂度 | 低（`INSERT` + `MAX(index)+1`） | 中（`FOR UPDATE SKIP LOCKED` pool claim） |
| DB schema | 不需要 `AVAILABLE` 状态 | 需要 status 字段 |
| 安全性 | 等同（都无私钥在线机） | 等同 |

**需要你确认**：选 A 还是 B？如果选 A，是否删除 `hdgen` 导出地址表的功能（仅保留种子/xpub 生成）？

---

### Q2. WebAuthn 公钥如何分发到气隙冷签名机？

**问题**：§9.4 要求 `coldsign` 用**自持公钥**验证用户 WebAuthn 断言（"不信在线 DB 的公钥"）。但 `coldsign` 是气隙机，唯一 I/O 是 USB proto 文件。用户注册 passkey 后，公钥存在在线 DB，怎么同步到冷签名机？

**需要你确认**：

- A. 定期从在线 DB 导出一份 `UserCredentialList`（proto）→ USB → 冷签名机导入。冷签名机对每个公钥做 TOFU（首次导入时操作员核对）。频率？每次提现前？
- B. 在线机在 `UnsignedTx` 中附带用户公钥 + 凭证 ID，冷签名机对公钥做 TOFU 存储，首次遇到未知公钥时在屏幕上展示 fingerprint 供操作员人工核对后存入本地库。
- C. 其他方案（请描述）。

附带问题：冷签名机存储的公钥库有上限吗？丢了怎么办（所有用户的提现都签不了直到重建）？

---

### Q3. 归集 3 笔交易的原子性缺口怎么补？

**问题**：归集 = delegate → transfer → undelegate。冷签名模型下，`SignedSweepBundle` 包含全部 3N 笔已签名交易，由在线机按序广播。如果 delegate 广播成功后在线机重启（丢失内存中的 bundle 引用），transfer 不会自动继续——在线机没有私钥重新签名。

可能的 stuck 状态：

| 中断点 | 后果 |
|--------|------|
| delegate 成功，transfer 未广播 | energy 锁定在分地址（14 天解押），资金未归集 |
| transfer 成功，undelegate 未广播 | energy 锁定在分地址，资金已归集 |

**需要你确认**：

- `sweep_logs` 是否拆为 3 条记录（delegate / transfer / undelegate）分别追踪状态？
- 是否需要「孤儿 energy 回收」流程（冷签名机签一批 undelegate-only 交易收回卡死的 energy）？
- 还是接受 14 天自然解押作为兜底（不需要额外机制）？

---

## 🟡 安全模型补充

### Q4. BIP32 非硬化派生的级联风险是否接受？

**问题**：`m/44'/195'/0'/0/index` 中 `index` 是非硬化派生。BIP32 数学特性：**xpub + 任意一个子私钥 → 可计算所有兄弟私钥**。

威胁场景：冷签名机遭到攻击（物理入侵/内部威胁/恶意软件），泄露**任意一个**分地址私钥。攻击者结合在线机上的 xpub（公开），可计算出**全部**分地址私钥。即"泄漏 1 个密钥 = 泄漏全部"。

**需要你确认**：

- A. 接受此风险。缓解：冷签名机物理安全 + Shamir 分片保管人隔离 + 审计。在 §8.2 威胁模型中显式记录此特性。
- B. 改为硬化派生 `m/44'/195'/0'/0'/index'`，代价是在线机不能用 xpub 公开派生地址，需要改为冷签名机预派生地址列表。
- C. 其他方案。

---

### Q5. 死人开关的告警通道自身可被阻断，风险是否记录？

**问题**：§9.2 死人开关的告警走出站（Telegram/邮件）。持续 root 攻击者如果同时切断在线机网络连接，出站告警也发不出去——死人开关不会响。

这并非设计缺陷（单机下不可能自举告警），但应该显式记录。对应的缓解：校验器同时**独立查链**——如果链上冷钱包出现未授权的资金移动（在线机没理由发起这种交易），校验器即报警。

**需要你确认**：是否在 §9.2 或 §8.2 增加这个剩余风险的记录？

---

## 🟢 技术细节修正

### Q6. proto `UnsignedTx.kind` 是否改为 enum？

**问题**：§8.3：

```proto
string kind = 1;  // "delegate" | "transfer" | "undelegate"
```

**建议**：改用 proto `enum`，提供编译期类型检查：

```proto
enum TxKind {
  TX_KIND_UNSPECIFIED = 0;
  TX_KIND_DELEGATE = 1;
  TX_KIND_TRANSFER = 2;
  TX_KIND_UNDELEGATE = 3;
}
```

**需要你确认**：改还是不改？

---

### Q7. `UnsignedTx` 字段语义是否按 kind 拆分？

**问题**：当前 `UnsignedTx` 所有字段对三种 kind 平铺：
- `amount` 对 delegate/undelegate 无意义
- `energy` 仅对 delegate 有意义
- `derivation_index` 对 delegate/undelegate 是能量账户（但能量账户不按 derivation_index 派生）

**建议**：用 `oneof` 区分：

```proto
message UnsignedTx {
  oneof tx {
    DelegateTx delegate = 1;
    TransferTx transfer = 2;
    UndelegateTx undelegate = 3;
  }
}
```

**需要你确认**：改还是不改？

---

### Q8. WebAuthn nonce 是否加入 proto？

**问题**：§10.7: `challenge = sha256(amount|dest|nonce|user_id)`，但 §8.3 `UnsignedTx` 没有 `nonce` 字段。冷签名机需重建 challenge 来验证 WebAuthn 断言，缺少 nonce 则无法独立重建。

**需要你确认**：nonce 是否需要加入 `UnsignedTx` 或在提现 bundle 中携带？

---

### Q9. 文档残余的 v1 概念是否修正？

**问题**：§2.8 链上对账公式仍引用 "热钱包 USDT 余额"。v2 冷签名模型中不存在热钱包。

**需要你确认**：修正为仅 Σ(分地址) + 冷钱包？

---

## 📋 文档补全需求

### Q10. MANUAL_REVIEW 的 SOP 框架？

当 deposits 进入 MANUAL_REVIEW 状态后，缺少：
- 管理员通过什么界面处理？（需要什么 RPC？）
- 验证真伪的标准操作是什么？
- 确认后如何补入账 / 确认伪后如何处理？

**需要你确认**：是否需要在 ADR 中定义 SOP 框架，或留到实现文档？

---

### Q11. 灾难恢复流程？

缺失场景：
- 冷签名机硬件故障 → 恢复签名能力的步骤
- 一枚 Shamir 分片丢失 → 2-of-3 降级处理
- USB 介质丢失（含未广播的已签名 bundle）→ 已签名交易的泄露风险

**需要你确认**：是否需要在 ADR 中覆盖，还是作为独立运维文档？

---

### Q12. 链上对账正负阈值不对称的理由？

§2.8: `差异 > $10 或 < -$1 → 立即告警`。为什么正负阈值不对称（多 $10 vs 少 $1）？

**需要你确认**：是不对称有意为之（少钱更紧急）还是笔误？

---

## 优先级总结

| 优先级 | 编号 | 阻塞 Phase |
|--------|------|-----------|
| 🔴 立即 | Q1（派生方式） | A |
| 🔴 立即 | Q2（公钥分发） | E |
| 🟡 尽快 | Q3（归集原子性） | C |
| 🟡 尽快 | Q4（BIP32 风险） | A |
| 🟢 可后续 | Q5（死人开关残余） | D |
| 🟢 可后续 | Q6 Q7 Q8（proto 精确度） | A/C/E |
| 🟢 可后续 | Q9（v1 残余） | 文档 |
| 📋 补充 | Q10 Q11 Q12 | 文档 |

---

# 第二轮审计（2026-07-19，澄清后）

> Opus 已通过 §11 对 Q1–Q12 逐条定案。本轮审计范围：**§11 决议是否完整传播回 §2–§10 正文**，以及定案后是否引入新缺口。
> 结论：12 条决议本身质量高、逻辑自洽。但 **6 处一致性断裂 + 3 个新缺口** 需要修正。
>
> **✅ 已修正（2026-07-19）**：🔴 A/B/E/G + 🟡 C/D/F/I 已全部传播回 `0026-hd-wallet-deposit-system.md`：
> A/C=§2.3 改按需派生(去地址池/AVAILABLE)+§2.2/§5.x/§6.3/§7 清理 · B=§2.3 sweep_logs 三腿 · I=§2.3 新增 sweep_bundles ·
> G=§2.2 能量账户路径 m/44'/195'/0'/1/0 · E=§5.1/§5.4 统一 hdkeychain · D=§8.4/§10.2 新 proto 术语 · F=§10.1 CI 断言在线/离线分离。
> 🟢 H（§9.4 credential_id 查找步骤）未改（仅影响理解，不阻塞实现）。

## 🔴 一致性断裂（§11 决议 vs §2–§10 正文）

### A. §2.3 仍在使用地址池模型，与 Q1 按需派生冲突

Q1 定案："不搞预导入地址池、不需要 AVAILABLE 状态机"。但 §2.3 仍保留：

- DDL 有 `status VARCHAR(16) NOT NULL DEFAULT 'AVAILABLE'` — 按需派生下地址创建即 ASSIGNED，不存在 AVAILABLE
- 分配 SQL 是 `FOR UPDATE SKIP LOCKED` 池 claim — 按需派生应为：查已有 → 无则 `INSERT ... derivation_index = next_index, status = 'ASSIGNED'`
- 文本 "或由冷签名机离线预派生列表导入 DB" — Q1 已否决预导入，应删除
- `address_pool_min_threshold` 配置项 — 无池则无需此配置

**改法**：重写 §2.3 DDL（`status` 改为 `ASSIGNED / RETIRED`，去掉 `AVAILABLE`）、重写分配 SQL、删除双方案表述。

---

### B. §2.3 sweep_logs 表结构未反映 Q3 三腿拆分

Q3 定案："sweep_logs 拆为 3 腿（delegate/transfer/undelegate）各自追踪广播状态"。但 §2.3 DDL 仍是单 `tx_hash` + 单 `status`。

一个归集操作 = 3 笔链上交易（3 个 tx_hash，各自独立状态），当前 schema 只能追踪一笔。

**改法**：拆为 3 条记录（每条有 `leg_type` + 自己的 `tx_hash` + 自己的 `status`），加 `batch_id` 关联同批次。

---

### C. 多处残留预导入地址池引用

Q1 定案删除地址池后，以下位置仍含池假设：

| 位置 | 残留 | 应改为 |
|------|------|--------|
| §5.5 Phase 1 | `hdgen(离线: 种子/xpub/地址表)` | 删"地址表"，仅保留种子/xpub/fingerprint |
| §7 步骤 5 | "地址池补充：通过 hdgen 离线导出「地址 + index」列表导入 DB" | 删除或改为"可选：导出一份地址表供一次性交叉核对"（按 Q1） |
| §6.3 | "地址池并发 claim 测试" | 改为"并发地址分配测试"（基于 SEQUENCE/计数器） |
| §6.3 | "地址池耗尽告警测试" | 删除 |
| §5.6 | `address_pool_min_threshold` 配置项 | 删除 |

---

### D. §8.4 和 §10.2 使用旧 flat proto 术语

Q6/Q7 已将 proto 升级为 `TxKind` enum + `oneof`，但：

- **§8.4**：`UnsignedTx(kind=transfer, from=冷钱包/热汇集, to=用户地址)` — `kind=transfer` 是旧 flat 字段，应改为 `UnsignedTx.transfer{...}`。"热汇集" 是 v1 概念。
- **§10.2** coldsign 步骤 2："每条 tx: 按 derivation_index 派生私钥" — `DelegateTx` / `UndelegateTx` 没有 `derivation_index`，它们用 `energy_account`。

**改法**：§8.4 和 §10.2 改用新 proto 术语。`DelegateTx`/`UndelegateTx` 签名时按能量账户路径派生私钥（见下方 E）。

---

### E. §5.1 技术栈在线派生库与 §10.1 代码示例不一致

§5.1 写 `github.com/fbsobreira/gotron-sdk/pkg/keys`，§10.1 代码用 `github.com/btcsuite/btcd/btcutil/hdkeychain`。两者不同。

**改法**：统一到一个库。建议用 `btcutil/hdkeychain`（通用 BIP32，无 TRON 依赖，更轻量），gotron-sdk 仅用于气隙机签名。

---

### F. §10.1 Phase A CI 断言路径需调整

§10.1 CI grep 路径写 `internal/{service,repository,chain,connect,sweep}`，但：
- Q3 后新增 `internal/sweep/` 已覆盖
- `internal/hdwallet/` 也应纳入检查（在线侧，不应有私钥代码）
- 气隙机代码 `cmd/hdgen/` `cmd/coldsign/` 应有私钥但必须只在离线二进制中

**改法**：区分在线 CI 断言（`internal/` + `cmd/server/`）vs 离线工具（`cmd/hdgen/` `cmd/coldsign/` 可豁免）。

---

## 🟡 新发现的设计缺口

### G. 能量账户的派生路径未指定

冷签名机需从种子派生以下私钥签名：

| 场景 | 路径 | ADR 指定？ |
|------|------|-----------|
| 分地址 transfer | `m/44'/195'/0'/0/index` | ✅ §2.2 |
| 能量账户 delegate/undelegate | `m/44'/195'/0'/???` | ❌ 缺失 |

`DelegateTx` 有 `energy_account` 地址字段，但 coldsign 需要**私钥**签名 — 需知道从种子的哪个路径派生。

**改法**：在 §2.2 增加能量账户派生路径（如 `m/44'/195'/0'/1/0` 或 `m/44'/195'/1'/0/0`），或 `DelegateTx` 中增加 `derivation_path` 字段。

---

### H. §9.4 提现验证中 credential 查找步骤缺失

Q8 的 `WithdrawalAuth.credential_id` 提供了查找键，但 §9.4 步骤 3 只写"用冷签器自持的用户凭证公钥验证"，未写出 `credential_id → 查本地库 → 取公钥 → 验 assertion` 这一关键步骤。

**改法**：在 §9.4 步骤 3 补充 credential_id 查找流程。

---

### I. Q3 已签 bundle 持久化缺少对应 DB 表

Q3："SignedSweepBundle 持久化入 DB"。但 ADR 没有定义存储表。需要 `sweep_bundles` 表存序列化 proto bytes + 批次元数据。

**改法**：在 §2.3 增加 `sweep_bundles` 表定义，或在 §10.4 明确存储方式。

---

## ✅ 确认正确的传播

| 决议 | 传播位置 | 状态 |
|------|---------|------|
| Q4 BIP32 级联风险 | §8.2 残余风险第 1 条 | ✅ 完整 |
| Q5 死人开关阻断 | §8.2 残余风险第 2 条 | ✅ 完整 |
| Q6/Q7/Q8 proto 升级 | §8.3 完整重写 | ✅ 完整 |
| Q9 v1 热钱包 | §2.8 已修正 | ✅ 完整 |
| Q10 MANUAL_REVIEW SOP | §11 框架 | ✅ 完备 |
| Q11 灾难恢复要点 | §11 | ✅ 完备 |
| Q12 阈值不对称理由 | §2.8 注释 | ✅ 完备 |
| R4 冷地址硬白名单 | §2.7 + §8.1 | ✅ 自洽 |

## 优先级

| 优先级 | 编号 | 说明 |
|--------|------|------|
| 🔴 必须修正 | A, B | §2.3 是实现时直接引用的核心章节，当前文本会误导实现 |
| 🔴 必须修正 | E, G | 能量账户派生路径缺失 → Phase C 无法实现签名 |
| 🟡 尽快修正 | C, D, F, I | 细碎不一致，逐个清理即可 |
| 🟢 建议补充 | H | 不影响实现但影响理解 |
