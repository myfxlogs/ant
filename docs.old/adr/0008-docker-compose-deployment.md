# ADR-0008 单机 docker-compose 部署形态

## Status
Accepted (2026-05-23)

## Context
ant 服务规模（5 容器）+ 用户量（散户产品早期 < 10 万 DAU 预期）+ 团队规模（小）。是否引入 K8s / Helm / 多副本？

## Options
- **A. 单机 docker-compose**（已选）：5 容器（frontend / backend / strategy-service / postgres / redis）单机部署，宿主仅暴露 frontend 端口
- B. K8s + Helm：扩缩容方便；但运维复杂度激增
- C. Nomad / 其他编排：过度设计

## Decision
**A**。`docker-compose.yml` 是部署唯一源；不引入 K8s/Helm/ArgoCD/Service Mesh/HPA/多副本。容器命名 / 网络 / 卷统一 `ant-*` 前缀，与 anttrader 生产环境隔离。

## Consequences
- **+** 运维简单；新人 30 分钟上手
- **+** 与 AGENT.md 部署形态一致
- **−** 单点故障（M5 上线前评估异地备份方案）
- **−** 性能上限受单机资源约束（触发器：MAU > 5 万 / API P95 > 500ms 时立 ADR 评估扩展）

## Related
- AGENT.md 部署形态
- DESIGN-REVIEW MR-07（healthcheck 缺口）

## History
- 2026-05-23 Accepted
