const aiStore = {
  ai: {
    store: {
      conversations: {
        newConversationTitle: 'Hội thoại mới'
      },
      messages: {
        sendFailedToast: 'Gửi tin nhắn thất bại',
        sendFailedInline: '(Gửi thất bại)',
        loadConversationFailed: 'Tải hội thoại thất bại',
        createConversationFailed: 'Tạo hội thoại thất bại',
        deleteConversationFailed: 'Xóa hội thoại thất bại',
        clearedLocalOnly: 'Đã xóa (chỉ cục bộ)',
        getReportsFailed: 'Get reports failed',
        generateReportSuccess: 'Report generated successfully',
        generateReportFailed: 'Report generation failed'
      },
      prefs: {
        rememberPrefix: '/remember ',
        rememberedToast: 'Đã lưu tùy chọn',
        savedReply: 'Đã lưu tùy chọn của bạn.'
      },
      strategyRules: {
        title: 'When writing AntTrader Python strategy code, you must strictly follow these validation rules:',
        rules: {
          noImport: '- No import / from ... import ... allowed',
          noGlobal: '- No global / nonlocal',
          noDunderAccess: '- No access to dunder attributes (obj.__xxx__)',
          noDunderName: '- No dunder names (__xxx__)',
          noDangerousCalls: '- No calls to: open()/eval()/exec()/compile()/__import__()/input()/globals()/locals()/vars()/dir()',
          runSignature: '- If defining run function: only one run(context), must have exactly 1 parameter context, no *args/**kwargs',
          mustDefineEntry: '- Strategy must define signal variable or run(context) function (prefer run(context))'
        },
        allowedGlobals: 'Allowed globals/modules: np, math, datetime, calculate_rsi (do not import).'
      },
      context: {
        userPrefsTitle: 'User preferences (please follow as much as possible):',
        outputTitle: 'Output requirements:',
        outputRules: {
          wrapPython: '- If outputting strategy code, output full code wrapped in \`\`\`python',
          validateFirst: '- Code must pass validate first',
          noImport: '- Do not output any import statements'
        }
      }
    }
  }
} as const;

export default aiStore;
