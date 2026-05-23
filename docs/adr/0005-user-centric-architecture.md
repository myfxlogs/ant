# ADR-0005 用户中心架构（非多租户）

## Status
Accepted (2026-05-23)

## Context
ant 是 to-C 散户产品；AlfQ 是 to-B 多租户。AlfQ 的 `tenant_id` + RLS policy 体系若直接迁移会大幅增加复杂度且与产品定位不符。

## Options
- **A. 用户中心**（已选）：一个用户 = 一个隔离单元；所有表外键到 `users.id`；不引入 tenant 概念
- B. 多租户 + RLS：B2B 标准方案，但散户产品过度设计
- C. 混合：长期可能需要（如机构客户）；但 M5 前不引入

## Decision
**A**。所有业务表通过 `user_id` 隔离；权限模型为「用户拥有自己的资源」+「平台管理员可见全局」。AlfQ 迁移时删除所有 `tenant_id` / RLS `app.tenant_id` 调用。

## Consequences
- **+** 数据模型简单；查询不需要每次绑定 tenant
- **+** 与策略市场（M5）的"多个用户互相订阅"模型自然契合
- **−** 后期接入机构客户需要 ADR superseding（届时再说）
- **−** 数据隔离依赖应用层 user_id 过滤；缺 RLS 兜底（→ M0.3 加 SQL review checklist）

## Related
- AGENT.md ADR 清单
- AlfQ 迁移计划 §1.2「不要搬的部分」

## History
- 2026-05-23 Accepted
