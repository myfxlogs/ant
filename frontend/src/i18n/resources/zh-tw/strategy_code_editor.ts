// Auto-generated from proto/ant/v1/i18n/strategy_code_editor_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyCodeEditor = {
  "strategy": {
    "codeEditor": {
      "actions": {
        "copy": "複製",
        "preview": "預覽訊號",
        "saveAsTemplate": "儲存為模板",
        "sendToAI": "發給AI修改",
        "sendToAIFixTitlePreview": "修復預覽問題",
        "sendToAIFixTitleValidate": "驗證未透過/有警告",
        "validate": "驗證程式碼"
      },
      "aiPrompt": {
        "currentCodeTitle": "【當前程式碼】",
        "fenceEnd": "```",
        "intro": "請根據以下資訊修改策略程式碼，使其透過驗證並且預覽訊號執行成功。",
        "outputTitle": "【輸出資訊】",
        "outro": "僅返回用 ```python``` 包裹的修復後程式碼。",
        "problem": "【問題】{{title}}",
        "pythonFenceStart": "```python"
      },
      "cards": {
        "previewResult": "預覽結果",
        "validationResult": "驗證結果"
      },
      "hints": {
        "previewInfo": "預覽將使用示例市場資料執行。"
      },
      "labels": {
        "account": "賬號",
        "code": "策略程式碼",
        "disabledSuffix": "（已禁用）",
        "symbol": "品種",
        "timeframe": "週期"
      },
      "messages": {
        "copied": "程式碼已複製",
        "copyFailed": "複製失敗，請手動複製",
        "enterCode": "請輸入策略程式碼",
        "execFailed": "執行失敗",
        "previewFailed": "預覽失敗",
        "previewOk": "預覽完成",
        "previewSuccess": "預覽成功",
        "savedAsTemplate": "已儲存為模板",
        "selectAccount": "請選擇賬號",
        "validateError": "驗證失敗",
        "validateFailed": "程式碼驗證失敗",
        "validateOk": "程式碼驗證透過"
      },
      "placeholders": {
        "code": "輸入Python策略程式碼...",
        "loadingSymbols": "可用品種載入中…",
        "noSymbols": "暫無可用品種",
        "selectAccount": "選擇賬號",
        "selectAccountFirst": "先選賬號",
        "selectSymbol": "選擇品種"
      },
      "title": "策略編輯器"
    }
  }
} as const;
export default StrategyCodeEditor;
