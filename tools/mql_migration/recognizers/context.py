"""共享分析上下文 — 识别器之间通过 Context 协作。

所有识别器读写同一个 AnalysisContext 实例，共享发现：
  - 执行模型（其他识别器据此选择扫描策略）
  - 指标调用列表（生成器据此生成指标代码）
  - 发现的变量（生成器据此去重和引用）
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Set

from tools.mql_migration.intent_ir import ExecutionKind, ExecutionModel


@dataclass
class IndicatorRef:
    """指标调用引用 — 识别器发现后存入 Context。"""
    name: str                           # SDK 方法名: "ema", "rsi", "i_custom"
    params: Dict[str, Any] = field(default_factory=dict)  # SDK 参数
    result_var: str = ""                # 结果赋值给哪个变量
    source_expr: Any = None             # MQL AST 节点（供 ExpressionGen 翻译）
    comment: str = ""


@dataclass
class AnalysisContext:
    """识别器共享的分析上下文。

    识别器通过此对象通信——一个识别器的发现可供其他识别器使用。
    """

    # 执行模型（最先被 exec_model 识别器写入）
    execution_model: Optional[ExecutionModel] = None

    # 指标引用 —— 识别器发现的指标调用集合
    indicators: List[IndicatorRef] = field(default_factory=list)

    # 已知的局部变量 → 识别器名映射（去重）
    local_vars: Dict[str, str] = field(default_factory=dict)

    # 已知的全局变量 → 识别器名映射
    global_vars: Dict[str, str] = field(default_factory=dict)

    # 函数级局部变量（keyed by function name）
    func_local_vars: Dict[str, Set[str]] = field(default_factory=dict)

    # 识别器运行统计
    recognizer_results: Dict[str, int] = field(default_factory=dict)

    def add_indicator(self, ref: IndicatorRef) -> None:
        """注册一个发现的指标引用（自动去重）。"""
        key = f"{ref.name}:{ref.result_var}"
        for existing in self.indicators:
            if f"{existing.name}:{existing.result_var}" == key:
                return  # 已存在
        self.indicators.append(ref)

    def add_local_var(self, name: str, func_name: str) -> None:
        """注册函数内的局部变量。"""
        if func_name not in self.func_local_vars:
            self.func_local_vars[func_name] = set()
        self.func_local_vars[func_name].add(name)

    def get_local_vars(self, func_name: str) -> Set[str]:
        """获取指定函数的局部变量。"""
        return self.func_local_vars.get(func_name, set())

    def record_recognizer(self, name: str, count: int) -> None:
        """记录识别器运行结果。"""
        self.recognizer_results[name] = count

    @property
    def execution_kind(self) -> ExecutionKind:
        """获取执行模型类型（安全访问）。"""
        if self.execution_model:
            return self.execution_model.kind
        return ExecutionKind.ON_TICK
