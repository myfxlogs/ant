// Auto-generated from proto/ant/v1/i18n/strategy_code_assist_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const CodeAssist = {
  "strategy": {
    "codeAssist": {
      "aiReviseTitle": "AI assistant — revise code",
      "applyAllSuggestions": "Apply suggested defaults",
      "codeEmpty": "There is no code to revise yet.",
      "codeUpdated": "Code updated. Please re-run validation before saving.",
      "defaultLabel": "default",
      "enterInstruction": "Please describe what you want to change.",
      "explain": "Explain code",
      "fillRequiredParams": "Please fill the required parameters: {{keys}}",
      "generatePlaceholder": "Describe your strategy requirements...",
      "noPython": "AI did not return a Python block. Try rephrasing.",
      "optionalParamsDesc": "These parameters already have defaults from the code. Leave a field blank to use the default; entering a value only applies to this run and does not modify the saved strategy.",
      "optionalParamsTitle": "Optional parameters",
      "paramDescriptions": {
        "confidence": "Signal confidence threshold (0-1). Signals below this value are ignored.",
        "emaPeriod": "EMA (exponential moving average) lookback in bars.",
        "fastPeriod": "Fast period (number of bars). Used by MACD / dual-MA; smaller is more reactive.",
        "genericPercent": "Percentage / ratio parameter (e.g. 1 means 1%).",
        "genericPeriod": "Lookback window in bars used for indicator calculation.",
        "lotSize": "Order size (lots / volume). Larger size means more risk.",
        "maxLoss": "Max loss per trade as a fraction of equity (0.01 = 1%).",
        "riskLevel": "Risk level (low / medium / high). Controls position size and stop/take-profit width.",
        "rsiPeriod": "RSI lookback (number of bars). Typical value: 14.",
        "signalPeriod": "Signal period (number of bars). Smoothing length for MACD DIF/DEA.",
        "slowPeriod": "Slow period (number of bars). Used by MACD / dual-MA; larger is smoother.",
        "smaPeriod": "SMA (simple moving average) lookback in bars.",
        "stopLoss": "Stop-loss distance (%). Close the position once price moves this far against you.",
        "takeProfit": "Take-profit distance (%). Close the position once price moves this far in your favour.",
        "threshold": "Threshold that triggers a signal. Exact meaning depends on the strategy logic."
      },
      "required": "required",
      "requiredParamsDesc": "The strategy reads these parameters but no default was provided. Fill them in before saving.",
      "requiredParamsTitle": "Required parameters",
      "reviseInputPlaceholder": "e.g. Replace SMA(20) with EMA(50) and add a 1% stop-loss.",
      "reviseSend": "Send to AI",
      "saveBlockedNotValidated": "Please click \"Validate code\" first. Save is disabled until validation passes.",
      "suggested": "suggested",
      "tabAI": "AI revise",
      "tabExplain": "Explain code"
    }
  }
} as const;
export default CodeAssist;
