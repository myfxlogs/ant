"""MQL → Native Strategy 迁移引擎。

积木式架构：
  Layer 1: ExpressionGen — 原子表达式翻译（复用 ast_transpiler）
  Layer 2: BlockRecognizer — AST 模式匹配 → 意图 IR
  Layer 3: CodeGenerator — 意图 IR → 原生 SDK 代码
"""
