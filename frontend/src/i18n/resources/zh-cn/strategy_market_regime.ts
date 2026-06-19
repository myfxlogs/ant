// Auto-generated from proto/ant/v1/i18n/strategy_market_regime_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const MarketRegime = {
  "strategy": {
    "marketRegime": {
      "detectFailed": "市场环境检测失败",
      "detectSuccess": "市场环境检测完成",
      "form": {
        "accountId": "账户 ID",
        "accountIdPlaceholder": "输入 MT 账户 UUID",
        "accountIdRequired": "请输入账户 ID",
        "klineCount": "K 线数量",
        "submit": "开始检测",
        "symbol": "品种",
        "symbolPlaceholder": "例如 EURUSD",
        "symbolRequired": "请选择品种",
        "timeframe": "周期",
        "title": "检测参数"
      },
      "result": {
        "confidence": "置信度",
        "features": "特征",
        "modelVersion": "模型版本",
        "recordId": "记录 ID",
        "status": "状态",
        "strategyFamilies": "策略族类",
        "title": "检测结果"
      },
      "ruleVersionAlert": "当前使用基于规则的检测模型 rule-v1，由实时 K 线市场数据驱动。",
      "subtitle": "从历史 K 线数据分析趋势方向、波动率状态和价格效率，分类当前市场环境。",
      "title": "市场环境检测"
    }
  }
} as const;
export default MarketRegime;
