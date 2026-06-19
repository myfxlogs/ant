// Auto-generated from proto/ant/v1/i18n/strategy_code_assist_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyCodeAssist = {
  "strategy": {
    "codeAssist": {
      "paramDescriptions": {
        "confidence": "信号置信度阈值 (0-1)。低于此值的信号将被忽略。",
        "emaPeriod": "EMA (指数移动平均) 回溯周期。",
        "fastPeriod": "快周期 (K线数)。用于 MACD / 双均线，越小越灵敏。",
        "genericPercent": "百分比 / 比例参数 (例如 1 表示 1%)。",
        "genericPeriod": "指标计算的回溯窗口 (K线数)。",
        "lotSize": "订单手数，手数越大风险越高。",
        "maxLoss": "每笔最大亏损占净值比例 (0.01 = 1%)。",
        "riskLevel": "风险等级 (低/中/高)。控制仓位大小及止损/止盈幅度。",
        "rsiPeriod": "RSI 回溯周期 (K线数)，典型值: 14。",
        "signalPeriod": "信号周期 (K线数)。MACD DIF/DEA 的平滑长度。",
        "slowPeriod": "慢周期 (K线数)。用于 MACD / 双均线，越大越平滑。",
        "smaPeriod": "SMA (简单移动平均) 回溯周期。",
        "stopLoss": "止损距离 (%)。价格朝不利方向移动此幅度后平仓。",
        "takeProfit": "止盈距离 (%)。价格朝有利方向移动此幅度后平仓。",
        "threshold": "触发信号的阈值，具体含义取决于策略逻辑。"
      },
      "aiReviseTitle": "AI 助手 — 修改代码",
      "applyAllSuggestions": "应用建议默认值",
      "codeEmpty": "尚无代码可修改。",
      "codeUpdated": "代码已更新。保存前请重新验证。",
      "defaultLabel": "默认值",
      "enterInstruction": "请描述您要修改的内容。",
      "explain": "解释代码",
      "fillRequiredParams": "请填写必填参数: {{keys}}",
      "generatePlaceholder": "描述您的策略需求...",
      "noPython": "AI 未返回 Python 代码块。请尝试重新描述。",
      "optionalParamsDesc": "这些参数已有代码默认值。留空则使用默认值；填入的值仅对本次运行生效，不会修改已保存的策略。",
      "optionalParamsTitle": "可选参数",
      "required": "必填",
      "requiredParamsDesc": "策略读取了这些参数但未提供默认值，请在保存前填写。",
      "requiredParamsTitle": "必填参数",
      "reviseInputPlaceholder": "例如: 把 SMA(20) 替换为 EMA(50)，并添加 1% 止损。",
      "reviseSend": "发给AI修改",
      "saveBlockedNotValidated": "请先点击\"验证代码\"。验证通过后才能保存。",
      "suggested": "建议",
      "tabAI": "AI 修改",
      "tabExplain": "解释代码"
    }
  }
} as const;
export default StrategyCodeAssist;
