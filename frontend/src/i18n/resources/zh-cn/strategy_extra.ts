// Auto-generated supplementary keys for strategy
const StrategyExtra = {
  "strategy": {
    "backtest": {
      "canceled": "回测已取消",
      "lotSize": "手数",
      "strategyParameters": "策略参数"
    },
    "workspace": {
      "chartIndicators": {
        "overlay": "主图叠加",
        "subPane": "副图指标"
      },
      "tour": {
        "code": "代码编辑器",
        "codeDesc": "在此编写或粘贴 MQL 策略代码，也可导入 .mq4/.mq5 文件。",
        "ai": "AI 助手",
        "aiDesc": "让 AI 生成、优化或调试策略，生成的代码会即时显示在编辑器中。",
        "backtestDesc": "使用可配置参数运行回测，查看资金曲线、交易统计和风险指标。",
        "save": "保存并发布",
        "saveDesc": "将策略保存为模板、发布到市场或部署到实盘调度。"
      }
    },
    "codeAssist": {
      "aiHint": "描述你想要的修改，例如"
    },
    "chat": {
      "executionPlan": "执行计划",
      "codeGenerated": "代码已生成，使用下方按钮运行策略审查和回测。"
    },
    "templates": {
      "title": "策略模板",
      "saveCurrent": "保存当前策略",
      "chatEdit": "对话编辑",
      "confirmDelete": "删除此策略？",
      "noTemplates": "暂无保存的策略模板",
      "sourceCode": "策略源码",
      "gallery": {
        "unpublishFailed": "取消发布失败",
        "fork": "复刻并编辑",
        "aiGenerate": "AI 生成",
        "searchPlaceholder": "搜索策略...",
        "empty": "未找到策略",
        "deleteFailed": "删除失败"
      },
      "scheduleLaunch": {
        "metrics": {
          "winRate": "胜率",
          "maxDrawdown": "最大回撤",
          "sharpe": "夏普比率"
        }
      },
      "detail": {
        "profitFactor": "盈利因子",
        "notFound": "未找到策略",
        "noDescription": "暂无描述",
        "equityCurve": "资金曲线",
        "tradeStats": "交易统计"
      },
      "table": {
        "useCount": "使用次数"
      },
      "messages": {
        "fetchTemplateListFailed": "复刻失败",
        "publishFailed": "发布失败"
      },
      "actions": {
        "create": "新建策略"
      },
      "deleteConfirm": "删除此策略？"
    },
    "live": {
      "stopSuccess": "策略已停止",
      "stopFailed": "停止失败",
      "runId": "运行 ID",
      "account": "账号",
      "symbol": "品种",
      "timeframe": "周期",
      "mode": "模式",
      "signals": "信号",
      "errors": "错误",
      "startedAt": "开始时间",
      "watchSignals": "查看信号",
      "confirmStop": "确认停止此策略？",
      "status": "状态",
      "totalSignals": "总信号数",
      "stoppedAt": "停止时间",
      "error": "错误",
      "title": "实盘策略监控",
      "activeTab": "活跃运行",
      "noActive": "无活跃策略",
      "historyTab": "运行历史",
      "noRuns": "无策略运行记录",
      "schedulesTab": "调度",
      "signalLog": "信号日志",
      "waitingSignals": "等待信号...",
      "time": "时间",
      "signalType": "类型",
      "volume": "手数",
      "price": "价格",
      "sl": "止损",
      "tp": "止盈",
      "reason": "原因"
    },
    "ai": {
      "reviseHint": "先编写代码，再让 AI 改进。",
      "explainHint": "编写代码后查看 AI 解释。",
      "settingsHint": "配置 AI 提供商和模型"
    },
    "validate": {
      "running": "正在验证...",
      "fixWithAI": "发送错误到 AI 修订",
      "allClear": "所有检查通过 — 未发现问题。",
      "passed": "验证通过 — 现在可以保存。"
    },
    "importEA": {
      "writeTab": "策略代码",
      "importTab": "导入 EA",
      "codeTooShort": "请粘贴完整的 EA/指标源码。",
      "pastePlaceholder": "粘贴 MQL4/MQL5 EA 代码...",
      "migration": "策略导入",
      "aiTranslate": "AI 翻译",
      "bridge": "盲区桥接",
      "analyze": "分析策略结构",
      "confirmImport": "确认导入",
      "tryAI": "AI 翻译补充",
      "apply": "应用到编辑器",
      "importSuccess": "MQL 源码已导入，点击「应用到编辑器」写入编辑器",
      "hint": "粘贴 MQL4/MQL5 代码并点击分析",
      "translate": "翻译为 Go",
      "translating": "粘贴 MQL4/MQL5 代码并点击翻译",
      "bridgeBtn": "盲区桥接翻译",
      "bridging": "AI 正在桥接盲区...",
      "bridgeFailedMsg": "Agent 无法自动桥接所有盲区",
      "noBridgeNeeded": "覆盖率 100%，无需桥接",
      "bridgeHint": "粘贴 MQL4/MQL5 EA 代码，AI 将桥接盲区到平台字节码",
      "tooltip": "导入 MQL4/MQL5 源码",
      "button": "导入 MQL",
      "title": "导入 MQL 策略"
    },
    "version": {
      "loadFailed": "加载版本失败",
      "rollbackSuccess": "已回滚到版本 {{n}}",
      "rollbackFailed": "回滚失败",
      "loadVersionFailed": "加载版本失败",
      "loadDiffFailed": "加载差异失败",
      "colSummary": "变更摘要",
      "rollbackConfirm": "回滚到 v{{n}}？",
      "title": "版本历史",
      "empty": "暂无版本历史",
      "history": "版本历史"
    }
  }
} as const;
export default StrategyExtra;
