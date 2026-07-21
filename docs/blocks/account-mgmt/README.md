# account-mgmt — 账户管理

> MT 账户 CRUD、经纪商搜索、用户认证、钱包、WebAuthn。

## 代码位置

```
backend/internal/connect/user/   ← 账户 CRUD、连接生命周期、MT 凭据验证、auth handler
backend/internal/connect/gateway/← AI Gateway handler
backend/internal/service/        ← 35+ 文件：账户同步、注册、密码、钱包、存款、WebAuthn
```

## 关键设计

- 创建账户时先验证 MT 凭据再入事务——避免存储无效凭据
- WebAuthn 用于提现二次确认
- HD 钱包冷签名（ADR-0026）：在线机零私钥，提现走气隙签名

## 依赖

```
mt-gateway(HealthCheck/AccountInfo) → account-mgmt(凭据验证)
```

## 被依赖

```
account-mgmt → 所有需要用户认证和 MT 账户的模块
```

## 关联文档

- `docs/adr/0026-hd-wallet-deposit-system.md`
- [spec/31-saas-foundation.md](spec/31-saas-foundation.md)
