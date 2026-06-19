// Auto-generated from proto/ant/v1/i18n/strategy_code_assist_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyCodeAssist = {
  "strategy": {
    "codeAssist": {
      "paramDescriptions": {
        "confidence": "訊號信心度閾值 (0-1)。低於此值的訊號將被忽略。",
        "emaPeriod": "EMA (指數移動平均) 回溯週期。",
        "fastPeriod": "快週期 (K線數)。用於 MACD / 雙均線，越小越靈敏。",
        "genericPercent": "百分比 / 比例參數 (例如 1 表示 1%)。",
        "genericPeriod": "指標計算的回溯視窗 (K線數)。",
        "lotSize": "訂單手數，手數越大風險越高。",
        "maxLoss": "每筆最大虧損佔淨值比例 (0.01 = 1%)。",
        "riskLevel": "風險等級 (低/中/高)。控制倉位大小及止損/止盈幅度。",
        "rsiPeriod": "RSI 回溯週期 (K線數)，典型值: 14。",
        "signalPeriod": "訊號週期 (K線數)。MACD DIF/DEA 的平滑長度。",
        "slowPeriod": "慢週期 (K線數)。用於 MACD / 雙均線，越大越平滑。",
        "smaPeriod": "SMA (簡單移動平均) 回溯週期。",
        "stopLoss": "止損距離 (%)。價格朝不利方向移動此幅度後平倉。",
        "takeProfit": "止盈距離 (%)。價格朝有利方向移動此幅度後平倉。",
        "threshold": "觸發訊號的閾值，具體含義取決於策略邏輯。"
      },
      "aiReviseTitle": "AI 助手 — 修改程式碼",
      "applyAllSuggestions": "套用建議預設值",
      "codeEmpty": "尚無程式碼可修改。",
      "codeUpdated": "程式碼已更新。儲存前請重新驗證。",
      "defaultLabel": "預設值",
      "enterInstruction": "請說明您要修改的內容。",
      "explain": "解釋程式碼",
      "fillRequiredParams": "請填寫必要參數: {{keys}}",
      "generatePlaceholder": "說明您的策略需求...",
      "noPython": "AI 未返回 Python 程式碼區塊。請嘗試重新說明。",
      "optionalParamsDesc": "這些參數已有程式碼預設值。留空則使用預設值；填入的值僅對本次執行生效，不會修改已儲存的策略。",
      "optionalParamsTitle": "可選參數",
      "required": "必要",
      "requiredParamsDesc": "策略讀取了這些參數但未提供預設值，請在儲存前填寫。",
      "requiredParamsTitle": "必要參數",
      "reviseInputPlaceholder": "例如: 把 SMA(20) 替換為 EMA(50)，並加入 1% 止損。",
      "reviseSend": "發給AI修改",
      "saveBlockedNotValidated": "請先點選「驗證程式碼」。驗證通過後才能儲存。",
      "suggested": "建議",
      "tabAI": "AI 修改",
      "tabExplain": "解釋程式碼"
    }
  }
} as const;
export default StrategyCodeAssist;
