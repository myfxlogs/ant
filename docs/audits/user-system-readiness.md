# 用户系统上线准备审计

---

## 阻塞上线项（5 个）

### 🔴 1. 注册无密码强度检查

用户可用 1 字符密码注册。Admin 创建用户有最小 8 位，但用户面没有。金融交易平台不能接受。加 `len(password) >= 8`。

### 🔴 2. 无用户自助密码找回

`password_reset_repo.go` 存在但只有 Admin 能调用。用户忘了密码必须联系管理员。必须有 `ForgotPassword`（发邮件）+ `ResetPassword`（验 token + 设新密码）。

### 🔴 3. 无 Token 撤销和 Refresh Token 轮转

Logout 只清客户端 Cookie。Refresh token 一旦泄露，7 天内有效，无法服务端撤销。需加 refresh token 轮转（每次 refresh 发新 token、旧 token 失效）。

### 🔴 4. 限流仅在内存

`sync.Map` 存储，多实例部署时各自独立计数。未来横向扩展需换 Redis。当前单实例可运行。

### 🔴 5. Docker 部署 Cookie 无 Secure

`handlers_user.go` 硬编码 `insecure=true`。生产环境必须开启 `Secure` flag。

---

## 可上线但需尽快补齐

| 项 | 问题 | 方案 |
|----|------|------|
| 邮箱验证 | SMTP 未配置时不发邮件 | 上线前配 SMTP + 开 `REQUIRE_EMAIL_VERIFICATION` |
| 注册错误消息 | 返回"email already registered" | 改为通用消息，不泄露邮箱是否注册 |
| 重置密码邮件限流 | resend 端点无限流 | 加 rate limit |

---

## 合格的

- ✅ 密码哈希 Argon2id（64MB/3 iterations），向后兼容 bcrypt
- ✅ 登录错误统一"invalid credentials"，不泄露用户存在
- ✅ JWT 15min + Refresh 7d，HttpOnly + SameSite=Strict
- ✅ 用户删除有审计追踪，30 天软删→硬删
- ✅ 验证 token SHA-256 哈希存储，单次使用，24h 过期
- ✅ 验证邮件重发不泄露邮箱存在性
