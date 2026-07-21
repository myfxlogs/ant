# mql-compiler — MQL/Python 编译器

> MQL4/MQL5/Python → tree-sitter → IR → Bytecode → VM。

## 代码位置

```
backend/tools/mql2go/           ← 55+ Go 文件
  interp/                       ← IR 定义、分析器
  compile.go / compile_py.go   ← 编译入口
  vm.go / vm_execute.go        ← 字节码 VM（操作数栈、全局变量、调用深度 256、最大 tick 10M）
  builtins.go                   ← 277+ 内置函数注册
  ast_coverage.go               ← 盲区追踪
```

## 关键设计

- 双编译前端：MQL（tree-sitter CGo） + Python（AST 解析），共用一个 IR 和 VM
- 确定性管道——同一份源码两次编译得到相同字节码
- 盲区追踪：标记未支持的 MQL 操作，Agent 用盲区桥接（agent-engine）翻译到 Python 子集

## 依赖

```
MQL/Python 源码 → mql-compiler → Bytecode
```

## 被依赖

```
mql-compiler → strategy-runtime(VM 执行策略)
mql-compiler → agent-engine(编译验证 + 盲区桥接)
mql-compiler → backtest-engine(回测执行)
```
