// Auto-generated from proto/ant/v1/i18n/strategy_code_editor_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const CodeEditor = {
  "strategy": {
    "codeEditor": {
      "actions": {
        "copy": "複製",
        "preview": "預覽訊號",
        "saveAsTemplate": "保存為模板",
        "sendToAI": "發給AI修改",
        "sendToAIFixTitlePreview": "修复預覽问题",
        "sendToAIFixTitleValidate": "驗證未通過/有警告",
        "validate": "驗證代碼"
      },
      "aiPrompt": {
        "currentCodeTitle": "【當前代碼】",
        "fenceEnd": "```",
        "intro": "請根據以下資訊修改策略代碼，使其通過驗證並且預覽訊號執行成功。",
        "outputTitle": "【輸出資訊】",
        "outro": "仅返回用 ```python``` 包裹的修复后程式碼。",
        "problem": "【問題】{{title}}",
        "pythonFenceStart": "```python"
      },
      "cards": {
        "previewResult": "預覽结果",
        "validationResult": "驗證結果"
      },
      "hints": {
        "previewInfo": "預覽将使用示例市场資料执行。"
      },
      "labels": {
        "account": "帳號",
        "code": "策略代碼",
        "disabledSuffix": "（已禁用）",
        "symbol": "品種",
        "timeframe": "週期"
      },
      "messages": {
        "copied": "代碼已複製",
        "copyFailed": "複製失敗，請手動複製",
        "enterCode": "請輸入策略代碼",
        "execFailed": "執行失敗",
        "previewFailed": "預覽失敗",
        "previewOk": "預覽完成",
        "previewSuccess": "預覽成功",
        "savedAsTemplate": "已保存為模板",
        "selectAccount": "請選擇帳號",
        "validateError": "驗證失敗",
        "validateFailed": "代碼驗證失敗",
        "validateOk": "代碼驗證通過"
      },
      "placeholders": {
        "code": "輸入Python策略代碼...",
        "loadingSymbols": "可用品種載入中…",
        "noSymbols": "暫無可用品種",
        "selectAccount": "選擇帳號",
        "selectAccountFirst": "先選帳號",
        "selectSymbol": "選擇品種"
      },
      "title": "策略編輯器"
    }
  }
} as const;
export default CodeEditor;
