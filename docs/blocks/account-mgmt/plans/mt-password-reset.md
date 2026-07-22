# MT 凭据验证密码重置 · 施工清单

> 目标：用户可通过验证已绑定的 MT4/MT5 账户凭据来重置密码，无需邮件。
> 状态：✅ 代码已完成，待 Gate 0 手动验证。
>
> ⚠️ **前置修复**：broker_host 惰性重新发现。见 §0。

## §0 · 前置修复：broker_host 惰性重新发现

**问题**：`mt_accounts.broker_host` 是绑定时的静态快照。broker 更换服务器后 host 失效，连接失败且无法自我修复。

**修复**：在连接失败时自动触发 broker 重新搜索：

```
连接失败（非认证错误）
  → broker search（broker_company + platform → 返回当前在线服务器列表）
  → 逐个尝试连接
  → 成功 → UPDATE mt_accounts SET broker_host = new_host → 继续业务
  → 失败 → 返回"broker 服务器不可达"
```

- [x] **0a** 统一在 `ConnectAccount`（手动连接）、自动重连路径（`ReconnectGateway`）、`VerifyPassword`（MT 验证）三个入口处，连接失败时按错误类型分流：

  | 错误类型 | 行为 |
  |---------|------|
  | 认证失败（wrong password / investor only） | `ErrAuth` → 立即终止，不重试 |
  | 连接拒绝 / DNS 解析失败（host 错误） | `ErrHost` → 触发 broker 重新搜索 → 逐个试 → 成功更新 host |
  | 超时 / EOF / broker 返回服务不可用 | `ErrTransient` → **不触发重搜索**，走现有退避重连（host 对的，broker 挂了） |

  实现：`mdgateway/broker_rediscovery.go` — `ClassifyConnError()` 三档分类 + `HostRediscoverer.MaybeRediscover()`
  接入点：`runner_gateway.go:startGatewayForAccount`、`manager.go:ReconnectGateway`、`mt_tester.go:VerifyPassword`

- [x] **0b** 重搜索成功后 `UPDATE mt_accounts SET broker_host = $1 WHERE id = $2`
  - `HostRediscoverer` 和 `mt_tester.maybeRediscover` 均在成功后更新 DB
- [x] **0c** 不改 `Connect()` 的正常路径——仅在连接失败时触发。不引入定时器
- [ ] **Gate 0**：`go build ./...` ✅ 通过 + 手动测试（修改 DB 中的 broker_host 为错误值 → 触发连接 → 验证自动恢复）

---

## 已有基础设施（无需重复建设）

| 组件 | 位置 | 状态 |
|------|------|------|
| `PasswordResetRepo` | `password_reset_repo.go` | ✅ CreateResetToken/ValidateResetToken/ConsumeResetToken |
| `ForgotPassword`/`ResetPassword` RPC | `auth.proto:19-20` | ✅ 已定义+handler实现 |
| `MTConnectionTester` 接口 | `account_handler.go:25` | ✅ 含 `VerifyPassword(ctx,platform,brokerHost,login,password)` |
| `mtConnectionTester` 实现 | `mt_tester.go` | ✅ MT4/MT5 gateway Connect→Disconnect |
| `mt_accounts` 表 | `001_init.up.sql:25` | ✅ login,mt_type,broker_host,broker_server,user_id |
| `AuthServer` | `auth_handler.go:22` | ✅ 含 passwordResetRepo,emailNotifier |
| `ForgotPassword.tsx` | 前端 | ✅ 三 Tab（邮箱/MT验证/管理员），含 broker search 流程 |
| `ResetPassword.tsx` | 前端 | ✅ 新密码输入页，从 `?token=xxx` 读取 |
| `VerifyMTIdentity` RPC | `auth.proto:21` | ✅ 已定义 + handler 实现 |
| `broker_rediscovery.go` | `mdgateway/` | ✅ 三档错误分类 + HostRediscoverer |

---

## 模块 M1 · Proto 扩展

- [x] **M1a** 在 `proto/ant/v1/auth.proto` 新增 `VerifyMTIdentity` RPC

  ```protobuf
  message VerifyMTIdentityRequest {
    string email = 1;        // 用户邮箱（定位用户）
    string mt_login = 2;     // MT 账户号
    string mt_password = 3;  // MT 账户密码（只读 or 交易密码）
  }

  message VerifyMTIdentityResponse {
    bool verified = 1;
    string reset_token = 2;    // 验证通过后直接返回 reset token
    string message = 3;
  }

  // AuthService:
  // rpc VerifyMTIdentity(VerifyMTIdentityRequest) returns (VerifyMTIdentityResponse);
  ```

  `buf generate` 已完成。

  **设计**：要求 `email` + `mt_login` + `mt_password` 三个字段。`email` 用于定位用户（防枚举 + 解决跨 broker 同 login 冲突），`mt_login` 定位具体 MT 账户，`broker_host` 和 `platform` 由后端从 `mt_accounts` 表自动查取。用户无需知道 broker 服务器地址或平台类型。

---

## 模块 M2 · 后端 Handler

- [x] **M2a** 在 `backend/internal/connect/user/auth_mt_identity.go` 实现 `VerifyMTIdentity` 方法

  **逻辑**（两步查询）：
  1. 校验必填字段（`email`, `mt_login`, `mt_password`）
  2. 查 `users` 表：`SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL LIMIT 1`（用 email 定位用户）
  3. 查 `mt_accounts` 表：`SELECT broker_host, mt_type FROM mt_accounts WHERE user_id = $1 AND login = $2 AND deleted_at IS NULL LIMIT 1`（用 user_id + login 定位 MT 账户，自动获取 `broker_host` 和 `platform`）
  4. 调用 `mtTester.VerifyPassword(ctx, platform, brokerHost, mtLogin, mtPassword)` 验证 MT 凭据（含 §0 broker host rediscovery）
  5. 成功 → 调用 `s.passwordResetRepo.CreateResetToken(ctx, userID)` 生成 token
  6. 返回 `verified=true` + `reset_token`
  7. 失败 → 返回 `verified=false` + "MT credential verification failed"（防枚举：不区分用户不存在/MT账户不存在/密码错误）

  **依赖注入**：AuthServer 新增 `pg *pgxpool.Pool` 和 `mtTester MTConnectionTester` 字段，通过 `WithMTIdentityVerification(pg, tester)` 链式注入。

- [x] **M2b** 在 `handlers.go` 接线

  `mtTester` 在 `handlers.go` 中提前创建（共享给 AuthServer 和 AccountServer），`searcher` 也提前创建并传入 `NewMTConnectionTester`。

- [x] **M2c** Rate limit

  `interceptor/ratelimit.go` 已注册 `/ant.v1.AuthService/VerifyMTIdentity`（10 req/min per IP）。
  `interceptor/auth.go` 已将 `VerifyMTIdentity` 加入 auth-free 白名单。

---

## 模块 M3 · 前端

- [x] **M3a** 改造 `/forgot-password` 页面（`frontend/src/pages/auth/ForgotPassword.tsx`）

  三选一 Tab（📧 邮箱 / 🔐 MT验证 / 👤 管理员）。MT 验证 Tab 三个字段：
  ```
  ┌─────────────────────────────────────────────┐
  │  忘记密码                                     │
  │                                              │
  │  [📧 邮箱] [🔐 MT验证] [👤 管理员]            │
  │                                              │
  │  ── MT 验证 ──                                │
  │  邮箱:       [___________]                    │
  │  MT 账户号:  [___________]                    │
  │  MT 密码:    [___________]                    │
  │                                              │
  │  服务器和平台将自动检测                        │
  │                                              │
  │  [🚀 验证并重置密码]                           │
  └─────────────────────────────────────────────┘
  ```

  - 用户输入邮箱 + MT 账户号 + 密码，后端自动从 DB 查取 `broker_host` 和 `platform`
  - 验证成功 → 跳转 `/reset-password?token=xxx`
  - 验证失败 → 提示"MT credential verification failed"

- [x] **M3b** 新建 `/reset-password` 页面（`frontend/src/pages/auth/ResetPassword.tsx`）

  输入新密码 + 确认密码 → 调用 `ResetPassword` RPC → 成功跳转登录页。
  从 URL query param `?token=xxx` 读取 token。

- [x] **M3c** 路由注册

  `AppRoutes.tsx` 已添加 `/reset-password` 路由。

- [x] **M3d** i18n

  翻译 key 已添加（`auth.forgotPassword.*`, `auth.resetPassword.*`）。

---

## 模块 M4 · 测试

- [x] **M4a** 单元测试：`auth_mt_identity_test.go` — `TestVerifyMTIdentity_MissingFields/NotConfigured/InvalidPlatform`
- [ ] **M4b** E2E 测试（可选）：Playwright MT 验证流程 → 重置密码 → 登录

---

## Gate

```bash
buf generate          # ✅ 已完成
go build ./...        # ✅ 通过
go test ./internal/connect/user/...  # ✅ 通过
cd frontend && npm run build          # 待验证
```

**关键验收**：

1. 绑定 MT 账户的用户 → MT 验证通过 → 收到 reset token → 设新密码 → 登录成功
2. 未绑定 MT 账户的用户 → MT 验证失败 → 提示"未找到匹配的 MT 账户"
3. 错误 MT 密码 → 验证失败 → 不泄露是否账户存在
4. Rate limit 生效 → 10 次/min 后被拒绝
5. 邮箱 Tab 仍然可用（不破坏现有流程）

---

## 依赖关系

```text
M1a (proto) → buf generate → M2a (handler) → M2b (wiring) → M2c (rate limit)
                                                           ↓
M3a (ForgotPassword 改造) → M3b (ResetPassword 页面) → M3c (路由) → M3d (i18n)
                                                           ↓
                                                         M4a (测试)
```

## 风险与注意事项

- **防枚举**：无论用户是否存在、MT 账户是否绑定、密码是否正确，都返回通用消息 "MT credential verification failed"，不泄露信息。email + mt_login 双重定位，攻击者需同时知道两者才能探测。
- **跨 broker 同 login 冲突**：MT 账号是 broker 分配的，非全局唯一。通过 email 先定位 user_id，再用 user_id + login 查 mt_accounts，避免 `LIMIT 1` 取错用户。
- **Token 安全**：`PasswordResetRepo` 已使用 SHA-256 hash 存储 token，不存明文
- **MT 密码不存储**：`VerifyPassword` 只做 Connect → Disconnect，不持久化密码
- **broker_host 来源**：后端从 `mt_accounts` 表自动查取，用户无需输入。连接失败时由 §0 broker host rediscovery 自动恢复。
- **§0 broker host rediscovery**：`VerifyPassword` 连接失败时，如错误类型为 `ErrHost`（DNS/refused），自动触发 broker 重新搜索并更新 DB 中的 `broker_host`
- **并发限制**：MT gateway 连接是网络操作，rate limit 10 req/min 防滥用
