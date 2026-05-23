# Proto 拆分指南

## 1. 拆分目标

- `proto/*.proto`、Go 生成文件、TS 生成文件均应低于 800 行。
- 领域边界清晰，避免跨领域循环依赖。
- 每次拆分一组 proto，生成后立即验证引用与行数。

## 2. 拆分顺序

1. 优先拆 message 过大的领域文件。
2. 再拆 service 过多的文件。
3. 与策略生命周期、AI 流程相关的 proto 优先处理。
4. 外部 MT4/MT5 协议本体不在项目自有 proto 拆分范围内。

## 3. 文件组织

- 实体基础字段独立文件，例如 `backtest_run.proto`。
- 启动、查询、控制等用例拆成独立文件。
- service proto 只保留 RPC 定义和必要 import。
- 共享字段放入小型 common proto，避免形成超大 common。

## 4. 生成与检查

每次拆分后执行：

```bash
make proto
python3 scripts/check-file-lines.py
go test ./... # backend 目录
npm run build # frontend 目录
```

## 5. 兼容要求

- 不随意改 RPC 名称、字段编号和字段语义。
- 删除字段前必须确认前后端调用已迁移。
- 新字段优先追加，不复用已删除字段编号。
