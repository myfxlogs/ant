# Factor DSL 规范

**Status**: Accepted  
**Date**: 2026-05-23  
**关联 ADR**: ADR-0013  
**依赖**: Go 实现 `backend/internal/factor/dsl/`

## 1. 概述

因子 DSL（Domain-Specific Language）是 ant 量化平台的因子表达式语言，允许用户用简洁的数学语法定义技术指标和自定义因子。DSL 引擎同时存在于 Go（生产端）和 Python（研究端），两端的语义严格对齐，误差 < 1e-9。

### 1.1 设计目标

- **安全**：受限文法，禁止任意代码执行。DSL 是纯数据，不是代码。
- **可校验**：表达式合法性可在编译时完全验证。
- **双引擎对齐**：Go 和 Python 实现相同语义，回测与实盘结果一致。
- **增量求值**：算子支持逐 bar 增量计算，无需重复扫描历史窗口。

### 1.2 与 alfq 的关系

本 DSL 规范源自 alfq 的因子 DSL（`alfq/docs/09-因子DSL规范.md`），ant 移植了全套 Lexer/Parser/AST/Compiler/Operators，算子实现精确复刻，数值误差控制在 1e-9 以内。

## 2. EBNF 语法

```
expr       = ternary ;
ternary    = logic_or [ "?" expr ":" expr ] ;
logic_or   = logic_and { "||" logic_and } ;
logic_and  = equality { "&&" equality } ;
equality   = compare { ("==" | "!=") compare } ;
compare    = addsub { ("<" | "<=" | ">" | ">=" ) addsub } ;
addsub     = muldiv { ("+" | "-") muldiv } ;
muldiv     = unary { ("*" | "/" | "%") unary } ;
unary      = [ "-" | "!" ] primary ;
primary    = NUMBER | BOOL | STRING | FIELD | "(" expr ")" | IDENT [ "(" [ expr { "," expr } ] ")" ] ;

NUMBER     = digit { digit | "." } [ ("e" | "E") [ "+" | "-" ] digit { digit } ] ;
BOOL       = "true" | "false" ;
STRING     = '"' { any_char_except_quote } '"' ;
FIELD      = "$" ident ;
IDENT      = letter { letter | digit | "_" } ;
```

### 2.1 运算符优先级（由低到高）

| 优先级 | 运算符 | 结合性 |
|--------|--------|--------|
| 1 | `?:` (三元) | 右结合 |
| 2 | `\|\|` | 左结合 |
| 3 | `&&` | 左结合 |
| 4 | `==` `!=` | 左结合 |
| 5 | `<` `<=` `>` `>=` | 左结合 |
| 6 | `+` `-` | 左结合 |
| 7 | `*` `/` `%` | 左结合 |
| 8 | `-` `!` (一元) | 右结合 |

## 3. 词法元素

### 3.1 字段引用（Field）

以 `$` 开头，后接标识符：`$close`、`$open`、`$high`、`$low`、`$volume`。

字段名必须在编译时声明（`FieldIndex`），未声明的字段引用编译报错。

### 3.2 函数调用

`func_name(arg1, arg2, ...)` —— 函数名大小写不敏感。

### 3.3 因子引用

裸标识符（不以 `$` 开头），解析为已注册的因子定义。

## 4. 内置算子（15 个）

### 4.1 移动平均类

| 函数 | 签名 | 描述 | Warmup |
|------|------|------|--------|
| `sma` | `sma(field, n)` | 简单移动平均 | n |
| `ema` | `ema(field, n)` | 指数移动平均 (α=2/(n+1)) | n |
| `wma` | `wma(field, n)` | 加权移动平均 | n |

### 4.2 统计类

| 函数 | 签名 | 描述 | Warmup |
|------|------|------|--------|
| `std` | `std(field, n)` | 滚动标准差 | n |
| `var` | `var(field, n)` | 滚动方差 | n |
| `min` | `min(field, n)` | 滚动最小值 | n |
| `max` | `max(field, n)` | 滚动最大值 | n |
| `sum` | `sum(field, n)` | 滚动求和 | n |

### 4.3 振荡器类

| 函数 | 签名 | 描述 | Warmup |
|------|------|------|--------|
| `rsi` | `rsi(field, n)` | 相对强弱指数（Wilder 平滑） | n+1 |
| `macd` | `macd(field, fast, slow)` | MACD 线: ema(fast)-ema(slow) | slow |
| `atr` | `atr(field, n)` | 平均真实波幅 | n+1 |

### 4.4 时间序列类

| 函数 | 签名 | 描述 | Warmup |
|------|------|------|--------|
| `ref` | `ref(field, n)` | n 期前的值 | n |
| `delta` | `delta(field, n)` | v - ref(v, n) | n |
| `pct_change` | `pct_change(field, n)` | v/ref(v,n) - 1 | n |

### 4.5 标量类

| 函数 | 签名 | 描述 |
|------|------|------|
| `abs` | `abs(x)` | 绝对值 |
| `sign` | `sign(x)` | 符号 (-1/0/1) |
| `log` | `log(x)` | 自然对数 |
| `exp` | `exp(x)` | 指数函数 |
| `sqrt` | `sqrt(x)` | 平方根 |
| `pow` | `pow(x, n)` | xⁿ (n 为常数) |

### 4.6 布林带

| 函数 | 签名 | 描述 | Warmup |
|------|------|------|--------|
| `bb_upper` | `bb_upper(field, n, k)` | sma + k*std | n |
| `bb_lower` | `bb_lower(field, n, k)` | sma - k*std | n |

### 4.7 其他

| 函数 | 签名 | 描述 | Warmup |
|------|------|------|--------|
| `zscore` | `zscore(field, n)` | (v - sma) / std | n |
| `rank` | `rank(field, n)` | 滚动百分位排名 | n |
| `corr` | `corr(field1, field2, n)` | 皮尔逊相关系数 | n |
| `cov` | `cov(field1, field2, n)` | 协方差 | n |
| `if_` | `if_(cond, a, b)` | 条件分支（编译时展开为三元） | max(cond, a, b) |

## 5. 校验规则

### 5.1 语法校验

- 表达式长度 ≤ 4096 字符
- 解析超时 100ms（防 ReDoS）
- 所有字段引用必须在声明白名单内

### 5.2 安全校验

- 拒绝包含危险 token 的函数名：`exec`、`eval`、`import`、`open`、`os`、`sys`、`subprocess`、`__`
- DSL 不是代码 —— 没有循环、变量赋值、字符串操作（除字面量）

### 5.3 复杂度限制

- AST 深度 ≤ 8（防递归炸弹）
- 函数嵌套 ≤ 3 层

## 6. Go/Python 对齐约定

### 6.1 浮点语义

- 所有中间计算使用 `float64`（IEEE 754 双精度）
- NaN 传播：任何算子输入 NaN → 输出 NaN
- 除零 → NaN
- log/√ 负值 → NaN

### 6.2 Warmup 语义

- 每个有状态算子在缓冲填满前返回 NaN
- `Warmup()` 方法声明所需的最小数据点数
- 回测引擎应在 warmup 期间跳过信号生成

### 6.3 对齐测试

Go 和 Python 引擎需通过 100 个表达式 × 1000 bar 的对齐测试，最大误差 < 1e-9。

## 7. 示例表达式

```dsl
# 双均线交叉信号
ema($close, 20) / ema($close, 60) - 1

# RSI 超卖
rsi($close, 14) < 30 ? 1 : 0

# 布林带突破
$close > bb_upper($close, 20, 2)

# 量价共振
$close > sma($close, 50) && $volume > sma($volume, 20) * 1.5

# MACD 金叉
macd($close, 12, 26) > 0 ? 1 : -1

# 动量因子
pct_change($close, 5)

# 波动率因子
zscore(std($close, 20), 100)
```
