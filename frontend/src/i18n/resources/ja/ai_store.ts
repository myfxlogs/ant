const aiStore = {
  ai: {
    store: {
      strategyRules: { title: 'AntTrader Pythonコード作成時の検証ルール：', rules: { noImport: '- import / from ... import ... 禁止', noGlobal: '- global / nonlocal 禁止', noDunderAccess: '- dunder属性アクセス禁止（obj.__xxx__）', noDunderName: '- dunder名禁止（__xxx__）', noDangerousCalls: '- 禁止関数呼出: open()/eval()/exec()/compile()/__import__()/input()/globals()/locals()/vars()/dir()', runSignature: '- run関数: run(context) のみ、1パラメータ、*args/**kwargs禁止', mustDefineEntry: '- signal変数またはrun(context)関数必須（run推奨）' }, allowedGlobals: '使用可能: np, math, datetime, calculate_rsi（import不要）。' },
      context: { userPrefsTitle: 'ユーザー優先（可能な限り従ってください）：', outputTitle: '出力要求：', outputRules: { wrapPython: '- コード出力時は ```python で囲む', validateFirst: '- コードはvalidate通過必須', noImport: '- import文出力禁止' } },
      prefs: { rememberPrefix: '優先を記憶：', rememberedToast: '優先を記憶しました。以降の会話に適用。', savedReply: '優先保存済' },
      conversations: { newConversationTitle: '新規会話' },
      messages: { sendFailedInline: '送信失敗、再試行', sendFailedToast: '送信失敗、再試行', createConversationFailed: '会話作成失敗', loadConversationFailed: '会話読込失敗', deleteConversationFailed: '会話削除失敗', clearedLocalOnly: '現在会話メッセージ消去（サーバー記録保持）', getReportsFailed: 'レポート取得失敗', generateReportSuccess: 'レポート生成成功', generateReportFailed: 'レポート生成失敗' },
    },
  },
} as const;

export default aiStore;
