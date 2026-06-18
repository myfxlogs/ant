const aiStore = {
  ai: {
    store: {
      strategyRules: {
        title: 'When writing AntTrader Python strategy code, you must strictly follow these validation rules:',
        rules: {
          noImport: '- No import / from ... import ... allowed',
          noGlobal: '- No global / nonlocal',
          noDunderAccess: '- No access to dunder attributes (obj.__xxx__)',
          noDunderName: '- No dunder names (__xxx__)',
          noDangerousCalls:
            '- No calls to: open()/eval()/exec()/compile()/__import__()/input()/globals()/locals()/vars()/dir()',
          runSignature:
            '- If defining run function: only one run(context), must have exactly 1 parameter context, no *args/**kwargs',
          mustDefineEntry: '- Strategy must define signal variable or run(context) function (prefer run(context))',
        },
        allowedGlobals: 'Allowed globals/modules: np, math, datetime, calculate_rsi (do not import).',
      },
      context: {
        userPrefsTitle: 'User preferences (please follow as much as possible):',
        outputTitle: 'Output requirements:',
        outputRules: {
          wrapPython: '- If outputting strategy code, output full code wrapped in ```python',
          validateFirst: '- Code must pass validate first',
          noImport: '- Do not output any import statements',
        },
      },
      prefs: {
        rememberPrefix: 'Remember preference: ',
        rememberedToast: 'Preference remembered, will apply to subsequent conversations',
        savedReply: 'Preference saved',
      },
      conversations: {
        newConversationTitle: 'New conversation',
      },
      messages: {
        sendFailedInline: 'Send failed, please retry',
        sendFailedToast: 'Send failed, please retry',
        createConversationFailed: 'Create conversation failed',
        loadConversationFailed: 'Load conversation failed',
        deleteConversationFailed: 'Delete conversation failed',
        clearedLocalOnly: 'Current conversation messages cleared (server records retained)',
        getReportsFailed: 'Get reports failed',
        generateReportSuccess: 'Report generated successfully',
        generateReportFailed: 'Report generation failed',
      },
    },
  },
} as const;

export default aiStore;
