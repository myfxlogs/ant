"""意图 IR — 从 MQL 源码提取的策略意图。

当前阶段: Python 优先（识别器预翻译 expr 字段），ast_node 字段为未来 Go 目标保留。
这些数据结构是积木识别器（Layer 2）的输出，也是代码生成器（Layer 3）的输入。
"""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict, List, Optional, Set


# ── Parameter types ────────────────────────────────────────────────────


class ParamType(Enum):
    INT = "int"
    DOUBLE = "double"
    STRING = "string"
    BOOL = "bool"
    ENUM = "enum"


class ParamGroup(Enum):
    ENTRY = "入场参数"
    EXIT = "出场参数"
    SIZING = "手数参数"
    RISK = "风控参数"
    SYSTEM = "系统参数"


@dataclass
class ParamRange:
    """参数数值约束。"""
    min: Optional[float] = None
    max: Optional[float] = None
    step: Optional[float] = None


@dataclass
class ParamSpec:
    """单个策略参数规格 — 足够驱动 UI 面板渲染。"""
    name: str                               # MQL 原始名称
    label: str = ""                         # 中文显示名
    param_type: ParamType = ParamType.DOUBLE
    default: Any = None
    range: Optional[ParamRange] = None
    group: ParamGroup = ParamGroup.SYSTEM
    description: str = ""                   # 参数用途说明


# ── Strategy metadata ──────────────────────────────────────────────────


class MQLVersion(Enum):
    MQL4 = "mql4"
    MQL5 = "mql5"


@dataclass
class StrategyMeta:
    """策略元信息。"""
    name: str = ""
    mql_version: MQLVersion = MQLVersion.MQL4
    description: str = ""


# ── Execution model ────────────────────────────────────────────────────


class ExecutionKind(Enum):
    ON_TICK = "on_tick"             # 每 tick 执行
    ON_BAR = "on_bar"               # 每 bar close 执行（SDK 惯用）
    ON_INIT_GRID = "on_init_grid"   # 初始化时一次性放单


@dataclass
class ExecutionModel:
    """策略的执行模型。"""
    kind: ExecutionKind = ExecutionKind.ON_TICK
    timeframe_filter: Optional[str] = None   # 如 "H1"，仅 on_bar 模式使用
    require_account_check: bool = False      # 是否需要 AccountMode 检查
    require_on_trade: bool = False            # 是否需要 on_trade 回调


# ── State variables ────────────────────────────────────────────────────


@dataclass
class StateVar:
    """策略状态变量（对应 MQL 全局变量）。"""
    name: str
    var_type: str = "double"         # MQL 类型名
    initial_value: str = "0"         # Python 表达式
    description: str = ""


# ── Entry / Exit rules ─────────────────────────────────────────────────


class OrderAction(Enum):
    MARKET_BUY = "market_buy"
    MARKET_SELL = "market_sell"
    BUY_LIMIT = "buy_limit"
    SELL_LIMIT = "sell_limit"
    BUY_STOP = "buy_stop"
    SELL_STOP = "sell_stop"


class ExitTrigger(Enum):
    REVERSE_SIGNAL = "reverse_signal"    # 反向信号触发
    MAGIC_CLOSE = "magic_close"         # magic 隔离平仓
    MAGIC_DELETE = "magic_delete"       # magic 删挂单
    CLOSE_ALL = "close_all"             # 全部平仓（on_deinit）
    SL_TP = "sl_tp"                     # 止损止盈触发
    TIMER = "timer"                     # 定时器触发


class CloseAction(Enum):
    POSITION_CLOSE = "position_close"
    ORDER_DELETE = "order_delete"


@dataclass
class Condition:
    """条件表达式 — MQL AST 节点 + Python 表达式缓存。

    ``ast_node`` 是规范表示（MQL AST Expression），供非 Python 生成器使用。
    ``expr`` 是 Python 表达式字符串（识别器预翻译，Python 生成器直接消费）。

    当前阶段 Python 生成器优先使用 ``expr``。Go/其他生成器使用
    ``ast_node`` + 对应的 ExpressionGen。
    """
    ast_node: Any = None                # MQL AST Expression（规范表示）
    expr: str = ""                      # Python 表达式（识别器预翻译）
    comment: str = ""                   # 人类可读说明


@dataclass
class OrderParams:
    """订单参数 — str 字段是 Python 缓存，_ast 字段是规范 AST。

    Python 生成器使用 volume/price/sl/tp 字符串缓存。
    Go/其他生成器使用 volume_ast/price_ast/sl_ast/tp_ast。
    """
    symbol: str = "self.ctx.symbol"
    order_type: OrderAction = OrderAction.MARKET_BUY
    volume: str = ""                    # Python: 手数表达式
    volume_ast: Any = None              # AST: 手数（规范表示）
    price: str = ""                     # Python: 价格表达式
    price_ast: Any = None               # AST: 价格
    sl: str = ""                        # Python: 止损
    sl_ast: Any = None                  # AST: 止损
    tp: str = ""                        # Python: 止盈
    tp_ast: Any = None                  # AST: 止盈
    deviation: str = "3"
    magic: str = ""
    comment: str = ""


@dataclass
class MagicFilter:
    """magic 号码过滤条件。"""
    kind: str = "exact"                 # exact | range | all
    magic_value: str = ""               # 精确值表达式
    magic_min: str = ""                 # 范围下界
    magic_max: str = ""                 # 范围上界


@dataclass
class EntryRule:
    """单条入场规则。"""
    conditions: List[Condition] = field(default_factory=list)
    action: OrderAction = OrderAction.MARKET_BUY
    order_params: OrderParams = field(default_factory=OrderParams)
    source_location: str = ""           # 源码位置 "on_tick:L28-L34"


@dataclass
class ExitRule:
    """单条出场规则。"""
    trigger: ExitTrigger = ExitTrigger.MAGIC_CLOSE
    conditions: List[Condition] = field(default_factory=list)
    action: CloseAction = CloseAction.POSITION_CLOSE
    target: MagicFilter = field(default_factory=MagicFilter)
    source_location: str = ""


# ── Sizing / Risk ──────────────────────────────────────────────────────


class SizingKind(Enum):
    FIXED = "fixed"
    MARTINGALE = "martingale"
    PERCENT_BALANCE = "percent_balance"


@dataclass
class SizingRule:
    """手数规则。"""
    kind: SizingKind = SizingKind.FIXED
    expression: str = "0.10"            # 手数表达式


@dataclass
class IndicatorSpec:
    """策略需要的指标规格 — 代码生成器据此 emit 指标调用。

    SDK 方法名和参数名与目标语言无关（Python/Go 共用相同的方法名约定）。
    """
    sdk_method: str                     # "ema", "rsi", "i_custom", "atr"
    params: Dict[str, Any] = field(default_factory=dict)  # {"period": 14, "shift": 1}
    result_var: str = ""                # 结果赋值给哪个变量
    comment: str = ""


@dataclass
class RiskCheck:
    """风控检查规则。"""
    kind: str = "margin_check"          # margin_check | max_positions
    condition: str = ""                 # 触发条件表达式
    action: str = "close_all"           # close_all | block
    trigger: str = "on_timer"           # on_timer | on_bar | on_tick


@dataclass
class TimerRule:
    """定时器规则。"""
    interval_seconds: int = 300
    on_timer_actions: List[str] = field(default_factory=list)


# ── Blind spots ────────────────────────────────────────────────────────


class BlindSpotCategory(Enum):
    UNRECOGNIZED_ENTRY = "未识别的入场逻辑"
    UNRECOGNIZED_EXIT = "未识别的出场逻辑"
    UNRECOGNIZED_SIZING = "未识别的手数计算"
    UNRECOGNIZED_RISK = "未识别的风控逻辑"
    CUSTOM_LOGIC = "自定义逻辑"
    UNSUPPORTED_API = "不支持的API调用"


class Severity(Enum):
    CRITICAL = "致命"
    WARNING = "警告"
    INFO = "信息"


class HandlingStrategy(Enum):
    LLM_BEST_EFFORT = "LLM兜底翻译（已自动处理，需验证）"
    STUB_PLACEHOLDER = "占位保留（不影响编译，需人工补充）"
    SKIP_WITH_COMMENT = "标注跳过（不影响主逻辑）"
    HARD_BLOCK = "阻断（不允许上线）"


@dataclass
class BlindSpotFingerprint:
    """盲区指纹 — 基于 AST 结构特征，非源码文本。"""
    pattern_signature: str              # AST 模式哈希
    mql_functions: Set[str] = field(default_factory=set)
    context: str = ""                   # "entry_logic" / "exit_logic" / "sizing"


@dataclass
class BlindSpot:
    """单个盲区记录。"""
    id: str = ""                        # 指纹 hash
    location: str = ""                  # 源码位置
    category: BlindSpotCategory = BlindSpotCategory.CUSTOM_LOGIC
    severity: Severity = Severity.WARNING
    description: str = ""
    handling: HandlingStrategy = HandlingStrategy.STUB_PLACEHOLDER
    user_action_required: bool = False


# ── Top-level intent ───────────────────────────────────────────────────


@dataclass
class StrategyIntent:
    """从 MQL 源码提取的完整策略意图。Python 优先，ast_node 保留多语言扩展点。"""
    meta: StrategyMeta = field(default_factory=StrategyMeta)
    params: List[ParamSpec] = field(default_factory=list)
    state: List[StateVar] = field(default_factory=list)
    entry: List[EntryRule] = field(default_factory=list)
    exit: List[ExitRule] = field(default_factory=list)
    indicators: List[IndicatorSpec] = field(default_factory=list)
    sizing: Optional[SizingRule] = None
    risk: List[RiskCheck] = field(default_factory=list)
    execution: ExecutionModel = field(default_factory=ExecutionModel)
    timer: Optional[TimerRule] = None
    blind_spots: List[BlindSpot] = field(default_factory=list)
    # 用户可见的统计
    coverage_score: float = 0.0         # 0.0 – 1.0
    total_blocks: int = 0               # 总积木数
    recognized_blocks: int = 0           # 成功识别数
