账号管理模块
- Migration 187-189：状态机、审计日志、软删除
- 6 条管线全部修复 + 端口解析 + 多 host 逐试
- 账号级/策略级分析分离，恢复 Share + AI Report
- GatewayRemover wired，出入金纳入统计，SSE 触发分析刷新
- 14 个 commits

策略模块
- 审计完成，删 2 个死代码，路由/管线确认正常

 commit and push it, then deploy