// Auto-generated from proto/ant/v1/i18n/strategy_backtest_params_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyBacktestParams = {
  "strategy": {
    "backtestParams": {
      "presets": {
        "exploration": "探索模式",
        "liveAligned": "实盘对齐"
      },
      "backtestFailed": "回测失败",
      "both": "双向",
      "capital": "本金",
      "commission": "手续费",
      "currentDraft": "📝 当前草稿",
      "dateRange": "日期范围",
      "defaultsLoaded": "默认值已加载",
      "defaultsReset": "已恢复出厂默认值",
      "defaultsSaved": "默认值已保存",
      "direction": "方向",
      "endDate": "结束日期",
      "enterCodeAndSymbol": "请输入策略代码并选择品种",
      "eventDrivenMode": "Run(context) 事件驱动",
      "execution": "执行参数",
      "history": "回测历史",
      "leverage": "杠杆",
      "long": "↑ 做多",
      "lotSize": "手数",
      "run": "▶ 运行",
      "runtimeMode": "运行模式",
      "settingsLoad": "加载我的默认值",
      "settingsReset": "恢复出厂默认",
      "settingsSave": "保存为我的默认值",
      "short": "↓ 做空",
      "strategy": "策略",
      "strategyParams": "策略参数",
      "slippage": "滑点",
      "startDate": "开始日期",
      "strictMode": "严格模式",
      "strictModeOff": "关闭",
      "strictModeOffDesc": "同根K线收盘 + 1分钟子分辨率。精度更高。",
      "strictModeOffTooltip": "关闭: 同根K线收盘执行，1分钟子分辨率",
      "strictModeOn": "开启",
      "strictModeOnDesc": "次根K线开盘执行。标准保守模式。",
      "strictModeOnTooltip": "开启: 信号在K线收盘确认，次根开盘执行",
      "title": "回测",
      "trade": "交易",
      "vectorizedMode": "向量化",
      "mtLive": "MT 实时",
      "mtDataset": "数据集",
      "nextBarOpen": "次根K线开盘",
      "sameBarClose": "同根K线收盘"
    }
  }
} as const;
export default StrategyBacktestParams;
