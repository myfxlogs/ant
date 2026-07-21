# 地基审计 · account-mgmt

## 1. 核心设计替代方案

**当前**：MT 账户 CRUD + 用户认证（login/register/email verify）+ WebAuthn + 钱包 + HD 冷签名提现。创建账户时先验证 MT 凭据再入事务。

**替代方案**：OAuth/第三方登录。

不选。量化交易平台的用户需要绑定 MT 账户——OAuth 不能替代 MT 凭据验证。email+password+email verify 是合适的方案。未来可加 OAuth 作为补充，不是替代。

**结论**：✅ 架构最优。凭据先验证后存储的设计防止无效凭据入库。HD 冷签名（ADR-0026）是在线机零私钥的最优解。

## 2. 上下游契约

**上游**：mt-gateway 提供 `HealthCheck`/`AccountInfo` → account-mgmt 验证凭据。

**下游**：所有需要用户认证和 MT 账户的模块（13 个模块都要用）。

**隐患 A**：`users` 表关联了 13 个模块。用户删除时（GDPR/账户注销）的级联删除是否正确？所有 FK 是否都是 `ON DELETE CASCADE` 或 `SET NULL`？**需要验证 migration 150 的覆盖范围。**

**隐患 B**：MT 密码明文存储策略。当前 decrypt → 使用 → 不重新加密存储。如果 vault key 轮转，已存的加密密码需要重新加密（re-encrypt pipeline）。

## 3. 已知架构债务

| 债务 | 严重度 | 方案 |
|------|--------|------|
| MT 密码明文存储流程 | 🟡 | 确认 re-encrypt pipeline 在 vault key 轮转时是否覆盖所有已存密码 |
| 用户删除级联覆盖 | 🟡 | 验证 migration 150 的 FK 约束是否覆盖所有关联表 |
| MT 账户不能自助改密码 | 🟡 | ChangePassword 是 broker 端操作，不在平台做。确认用户是否可以通过 broker 官网自助管理 |

## 4. 总评

架构最优。HD 冷签名是安全模型的最优解（在线机零私钥）。三个黄标是操作层面的验证问题——不代表设计缺陷。
