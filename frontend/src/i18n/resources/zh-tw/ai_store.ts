const aiStore = {
  ai: {
    store: {
      strategyRules: { title: '編寫 AntTrader Python 策略程式碼時的驗證規則：', rules: { noImport: '- 禁止任何 import / from ... import ...', noGlobal: '- 禁止 global / nonlocal', noDunderAccess: '- 禁止存取任何 dunder 屬性（形如 obj.__xxx__）', noDunderName: '- 禁止使用 dunder 名稱（形如 __xxx__）', noDangerousCalls: '- 禁止呼叫：open()/eval()/exec()/compile()/__import__()/input()/globals()/locals()/vars()/dir()', runSignature: '- 若定義 run 函數：只能定義一個 run(context)，必須且只能有 1 個參數 context，禁止 *args/**kwargs', mustDefineEntry: '- 策略必須定義 signal 變數或 run(context) 函數（建議優先 run(context)）' }, allowedGlobals: '允許使用的全域物件/模組：np, math, datetime, calculate_rsi（不要 import）。' },
      context: { userPrefsTitle: '使用者偏好（請盡量遵循）：', outputTitle: '輸出要求：', outputRules: { wrapPython: '- 若輸出策略程式碼，請輸出完整程式碼，並用 ```python 包裹', validateFirst: '- 程式碼必須優先保證 validate 通過', noImport: '- 不要輸出任何 import 語句' } },
      prefs: { rememberPrefix: '記住偏好：', rememberedToast: '已記住偏好，將在後續對話中生效', savedReply: '偏好已儲存' },
      conversations: { newConversationTitle: '新對話' },
      messages: { sendFailedInline: '傳送失敗，請重試', sendFailedToast: '傳送失敗，請重試', createConversationFailed: '建立對話失敗', loadConversationFailed: '載入對話失敗', deleteConversationFailed: '刪除對話失敗', clearedLocalOnly: '目前對話訊息已清空（服務端記錄保留）', getReportsFailed: '獲取報告失敗', generateReportSuccess: '報告生成成功', generateReportFailed: '報告生成失敗' },
    },
  },
} as const;

export default aiStore;
