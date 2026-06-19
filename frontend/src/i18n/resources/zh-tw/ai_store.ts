// Auto-generated from proto/ant/v1/i18n/ai_store_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiStore = {
  "ai": {
    "store": {
      "context": {
        "outputRules": {
          "noImport": "- 不要输出任何 import 语句",
          "validateFirst": "- 程式碼必须优先保证 validate 通过",
          "wrapPython": "- 如果你输出策略程式碼，请输出完整程式碼，并用 ```python 包裹"
        },
        "outputTitle": "输出要求：",
        "userPrefsTitle": "用户偏好（请尽量遵循）："
      },
      "conversations": {
        "newConversationTitle": "新對話"
      },
      "messages": {
        "clearedLocalOnly": "当前對話訊息已清空（服务端记录保留）",
        "createConversationFailed": "建立對話失敗",
        "deleteConversationFailed": "刪除對話失敗",
        "generateReportFailed": "报告生成失敗",
        "generateReportSuccess": "报告生成成功",
        "getReportsFailed": "获取报告失敗",
        "loadConversationFailed": "載入對話失敗",
        "sendFailedInline": "傳送失敗，请重試",
        "sendFailedToast": "傳送失敗，请重試"
      },
      "prefs": {
        "rememberPrefix": "记住偏好：",
        "rememberedToast": "已记住偏好，将在后续對話中生效",
        "savedReply": "偏好已儲存"
      },
      "strategyRules": {
        "allowedGlobals": "允许使用的全局对象/模块：np, math, datetime, calculate_rsi（不要 import）。",
        "rules": {
          "mustDefineEntry": "- 策略必须定义 signal 变量或 run(context) 函数（建议优先 run(context)）",
          "noDunderAccess": "- 禁止访问任何 dunder 属性（形如 obj.__xxx__）",
          "noDunderName": "- 禁止使用 dunder 名称（形如 __xxx__）",
          "noGlobal": "- 禁止 global / nonlocal",
          "noImport": "- 禁止任何 import / from ... import ..."
        },
        "title": "你在编写 AntTrader Python 策略程式碼时，必须严格遵守以下驗證规则："
      }
    }
  }
} as const;
export default AiStore;
