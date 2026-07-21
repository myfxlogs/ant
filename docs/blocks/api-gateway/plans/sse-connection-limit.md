# api-gateway SSE 连接限制 · 施工清单

> 来源：`docs/audits/foundation-api-gateway.md` 黄标

- [ ] **S1** 在 SSE handler 层加入 per-user 并发连接数限制（建议每人最多 5 个流）
- [ ] **S2** 超出限制时返回 `ResourceExhausted` gRPC 状态码，前端显示"连接数已满，请关闭其他页面后重试"
- [ ] **S3** Admin 面板显示当前 SSE 连接总数 + TOP 用户
- [ ] **验收**：同一用户打开第 6 个 SSE 流 → 返回错误 → 关闭一个 → 新流可建立
