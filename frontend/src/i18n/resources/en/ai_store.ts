// Auto-generated from proto/ant/v1/i18n/ai_store_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiStore = {
  "ai": {
    "store": {
      "context": {
        "outputRules": {
          "noImport": "- Do not output any import statements",
          "validateFirst": "- Code must pass validate first",
          "wrapPython": "- If outputting strategy code, output full code wrapped in ```python"
        },
        "outputTitle": "Output requirements:",
        "userPrefsTitle": "User preferences (please follow as much as possible):"
      },
      "strategyRules": {
        "rules": {
          "mustDefineEntry": "- Strategy must define signal variable or run(context) function (prefer run(context))",
          "noDunderAccess": "- No access to dunder attributes (obj.__xxx__)",
          "noDunderName": "- No dunder names (__xxx__)",
          "noGlobal": "- No global / nonlocal",
          "noImport": "- No import / from ... import ... allowed"
        },
        "allowedGlobals": "Allowed globals/modules: np, math, datetime, calculate_rsi (do not import).",
        "title": "When writing AntTrader Python strategy code, you must strictly follow these validation rules:"
      },
      "conversations": {
        "newConversationTitle": "New conversation"
      },
      "messages": {
        "clearedLocalOnly": "Current conversation messages cleared (server records retained)",
        "createConversationFailed": "Create conversation failed",
        "deleteConversationFailed": "Delete conversation failed",
        "generateReportFailed": "Report generation failed",
        "generateReportSuccess": "Report generated successfully",
        "getReportsFailed": "Get reports failed",
        "loadConversationFailed": "Load conversation failed",
        "sendFailedInline": "Send failed, please retry",
        "sendFailedToast": "Send failed, please retry"
      },
      "prefs": {
        "rememberPrefix": "Remember preference: ",
        "rememberedToast": "Preference remembered, will apply to subsequent conversations",
        "savedReply": "Preference saved"
      }
    }
  }
} as const;
export default AiStore;
