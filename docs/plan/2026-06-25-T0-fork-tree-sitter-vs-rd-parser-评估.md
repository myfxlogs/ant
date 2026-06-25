# T0：Fork tree-sitter-cpp vs 补齐 RD parser — 量化评估

> **状态**：`NEEDS-DECISION`（待人类拍板 parser 路线）
> **日期**：2026-06-25
> **依据**：ADR-0020 决策 D8、实施说明书 C4

---

## 1. 工作量对比（小时）

| 维度 | 补齐 RD parser | Fork tree-sitter-c |
|------|---------------|-------------------|
| **P0 止血**（UnaryOp + Unicode） | <15 分钟 | N/A — 更换 parser |
| **表达式优先级/结合性** | ~50 行（重新排序递归下降层级） | **免费**（tree-sitter-c 自带 21 级优先级） |
| **for 循环子句** | ~50 行（重写 `_parse_for_clause`） | **免费**（C 语法已处理） |
| **#define 内联展开** | ~40 行 | **免费**（C 预处理已处理） |
| **数组声明 `double arr[10]`** | ~30 行（新增 array_declarator 规则） | **免费** |
| **类型修饰符（const, &, static）** | ~30 行 | **免费** |
| **break/continue** | ~15 行 | **免费** |
| **do-while** | ~20 行 | **免费** |
| **错误恢复** | 0（手写 RD 天然无错误恢复） | **免费**（tree-sitter 内建） |
| **表达式 codegen（剥离正则回退）** | ~100 行（最不确定的部分） | ~80 行（映射 tree-sitter CST→Python） |
| **添加 MQL 特有语法** | ~80 行（MQL5 class/交易 API） | ~200 行 grammar.js（MQL 类型/`#property`/交易函数签名） |
| **删除 C/C++ 特有语法** | N/A | ~100 行 diff（指针/联合体/asm/SEH…） |
| **AST→Python 桥接** | 已有（`ast_transpiler.py`，部分工作） | ~200 行（tree-sitter CST→Python codegen） |
| **编译 grammar + Python binding** | 0 | ~2h（CI 集成 `tree-sitter build`） |
| **测试重写** | ~150 行 | ~150 行（同） |
| **总代码量** | ~582 行 | ~530 行 grammar.js + ~430 行桥接 = ~960 行 |
| **总工时** | **2–3 天** | **1–1.5 天** |

## 2. 决定性的对比维度

### 表达式优先级：RD 的最大风险项

手写递归下降 parser 的表达式翻译**是当前最不可靠的部分**：

- 当前 RD parser 把三元/一元/二元全部塞进 `_parse_expression` → `_parse_primary`，**优先级靠 try/except 而非结构保证**。
- `-a * b` 被错误解析为 `-(a * b)`（一元优先级低于二元）—— 这是系统性 bug，不是个案。
- **补齐这意味着重写 `_parse_expression` 为标准的多层递归下降**（`_parse_assignment` → `_parse_ternary` → `_parse_or` → ... → `_parse_unary` → `_parse_primary`），约 80-120 行，且**每次加新运算符都要手工维护层级**。

Tree-sitter-c 已有 21 级优先级、左结合/右结合标记、动态优先级——**这些是 0 成本继承的**。

### 错误恢复

手写 RD parser 的 `_max_iter` / `_parse_depth` 只防止无限循环，不修复错误。一个语法错误 → 整棵树无效。Tree-sitter 有内建错误恢复——语法错误产生 ERROR 节点 + 继续解析后续。

### RD 的根本风险：未完的 20%

Opus 指出的核心问题：现在 RD parser 通过 `_needs_line_fallback` 把表达式甩给正则。**删除正则回退后，那 20% 必须从头实现**——这正是表达式 codegen 的 ~100 行，也是最不确定的。Tree-sitter 路径这条风险不存在：**C 语法天生覆盖所有表达式**。

## 3. 关键反论回应

**"Fork tree-sitter 需要写 2500 行 grammar.js"** → 不成立。
- Tree-sitter-c grammar.js ≈ 1200 行
- MQL4 ≈ C **减去** 指针/联合体/asm/SEH/restrict/_Generic（~40% 删除）**加上** MQL 类型/交易函数/#property（~20% 新增）
- 估计最终 grammar.js ≈ 1400 行，其中改动 ≈ 300 行 diff

**"Tree-sitter 引入新依赖"** → 成立，但可控。
- `tree-sitter` Python binding（pip install，已有生态）
- 编译 grammar → `.so`（build 脚本，CI 可缓存）
- 与 ConnectRPC 方向一致（外部工具链）

**"手写 RD 更可控"** → 理论成立，实证相反。
- 当前 RD parser 有 13 个已知 bug/缺失，其中表达式优先级问题是系统性的
- 每次修一个优先级 bug → 下一个出来（打地鼠）
- Tree-sitter-c 的优先级是 10 年 C 语法工程沉淀——"先质疑再自研"在这里适得其反

## 4. 推荐

```
████████████████████████████████████████████████████████████████
█  RECOMMENDATION: Fork tree-sitter-c（不是 cpp）            █
████████████████████████████████████████████████████████████████
```

**理由**：

| # | 论据 |
|---|------|
| 1 | **表达式优先级免费**：21 级 → RD 要手写 ~100 行 + 持续维护 |
| 2 | **错误恢复免费**：RD 永远做不到（手写 RD 没有这能力） |
| 3 | **工作量更低**：~1.5 天 vs ~3 天 |
| 4 | **长期维护成本更低**：grammar.js 是声明式语法，不是命令式解析代码 |
| 5 | **MQL4/5 都是 C 子集**：fork c（不是 cpp），MQL5 的 class 从 tree-sitter-cpp **再** fork 或直接手写类声明规则 |

**为什么 fork `tree-sitter-c` 而不是 `tree-sitter-cpp`**：
- MQL4 根本不涉及 MQL5 的 class/继承（Venus EA 是 MQL4）
- MQL5 的 class 特性可以直接在 MQL4-fork 上追加 ~80 行规则
- 从 `tree-sitter-cpp` fork → 需要删除更多（概念/模块/协程/lambda/fold/…），改动量更大
- 走 "fork c 做 MQL4 baseline → 后续追加 MQL5 class 规则" 更干净

**RD 路线的唯一优势**：0 外部依赖，0 编译步骤。但代价是表达式优先级这块会持续消耗精力，且错误恢复永远拿不到。

## 5. NEEDS-DECISION

请确认以下二选一：

- **A) Fork tree-sitter-c**（推荐）：立即 clone `tree-sitter/tree-sitter-c`，剥离 C 特有语法，加入 MQL 类型/内建，编译 `.so`，写 bridge 把 CST 喂给 `ast_transpiler.py` 的 codegen。
- **B) 补齐 RD parser**：修 13 个 bug，重写表达式优先级 → 多层递归下降，删除正则 fallback，代码量 ~582 行，工时 ~3 天。

无论选哪个，**T1（质量门）和 C1–C4 不变**——门先建，parser 后改。

---

*评估人：Claude (DeepSeek v4) · 审阅待：人类*
