// Auto-generated from proto/ant/v1/i18n/ai_store_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiStore = {
  "ai": {
    "store": {
      "context": {
        "outputRules": {
          "noImport": "- 不要輸出任何 import 語句",
          "validateFirst": "- 程式碼必須優先保證 validate 透過",
          "wrapPython": "- 如果你輸出策略程式碼，請輸出完整程式碼，並用 ```python 包裹"
        },
        "outputTitle": "輸出要求：",
        "userPrefsTitle": "使用者偏好（請儘量遵循）："
      },
      "strategyRules": {
        "rules": {
          "mustDefineEntry": "- 策略必須定義 signal 變數或 run(context) 函式（建議優先 run(context)）",
          "noDunderAccess": "- 禁止訪問任何 dunder 屬性（形如 obj.__xxx__）",
          "noDunderName": "- 禁止使用 dunder 名稱（形如 __xxx__）",
          "noGlobal": "- 禁止 global / nonlocal",
          "noImport": "- 禁止任何 import / from ... import ..."
        },
        "allowedGlobals": "允許使用的全域性物件/模組：np, math, datetime, calculate_rsi（不要 import）。",
        "title": "你在編寫 AlphaForge Python 策略程式碼時，必須嚴格遵守以下驗證規則："
      },
      "conversations": {
        "newConversationTitle": "新對話"
      },
      "messages": {
        "clearedLocalOnly": "當前對話訊息已清空（服務端記錄保留）",
        "createConversationFailed": "建立對話失敗",
        "deleteConversationFailed": "刪除對話失敗",
        "generateReportFailed": "報告生成失敗",
        "generateReportSuccess": "報告生成成功",
        "getReportsFailed": "獲取報告失敗",
        "loadConversationFailed": "載入對話失敗",
        "sendFailedInline": "傳送失敗，請重試",
        "sendFailedToast": "傳送失敗，請重試"
      },
      "prefs": {
        "rememberPrefix": "記住偏好：",
        "rememberedToast": "已記住偏好，將在後續對話中生效",
        "savedReply": "偏好已儲存"
      }
    }
  }
} as const;
export default AiStore;
