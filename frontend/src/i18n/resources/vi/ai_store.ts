const aiStore = {
  ai: {
    store: {
      strategyRules: {
        title: 'Khi viết mã chiến lược Python AntTrader, bạn phải tuân thủ nghiêm ngặt các quy tắc xác thực sau:',
        rules: {
          noImport: '- Không được phép import / from ... import ...',
          noGlobal: '- Không được dùng global / nonlocal',
          noDunderAccess: '- Không được truy cập thuộc tính dunder (obj.__xxx__)',
          noDunderName: '- Không được dùng tên dunder (__xxx__)',
          noDangerousCalls:
            '- No calls to: open()/eval()/exec()/compile()/__import__()/input()/globals()/locals()/vars()/dir()',
          runSignature:
            '- If defining run function: only one run(context), must have exactly 1 parameter context, no *args/**kwargs',
          mustDefineEntry: '- Chiến lược phải định nghĩa biến signal hoặc hàm run(context) (ưu tiên run(context))',
        },
        allowedGlobals: 'Toàn cục/mô-đun được phép: np, math, datetime, calculate_rsi (không import).',
      },
      context: {
        userPrefsTitle: 'Tùy chọn người dùng (vui lòng tuân thủ càng nhiều càng tốt):',
        outputTitle: 'Yêu cầu đầu ra:',
        outputRules: {
          wrapPython: '- If outputting strategy code, output full code wrapped in ```python',
          validateFirst: '- Mã phải vượt qua xác thực trước tiên',
          noImport: '- Không xuất bất kỳ câu lệnh import nào',
        },
      },
      prefs: {
        rememberPrefix: '/remember ',
        rememberedToast: 'Đã lưu tùy chọn',
        savedReply: 'Đã lưu tùy chọn của bạn.',
      },
      conversations: {
        newConversationTitle: 'Hội thoại mới',
      },
      messages: {
        sendFailedInline: 'Gửi tin nhắn thất bại',
        sendFailedToast: 'Gửi tin nhắn thất bại',
        createConversationFailed: 'Tạo hội thoại thất bại',
        loadConversationFailed: 'Tải hội thoại thất bại',
        deleteConversationFailed: 'Xóa hội thoại thất bại',
        clearedLocalOnly: 'Đã xóa (chỉ cục bộ)',
        getReportsFailed: 'Lấy báo cáo thất bại',
        generateReportSuccess: 'Báo cáo đã được tạo thành công',
        generateReportFailed: 'Tạo báo cáo thất bại',
      },
    },
  },
} as const;

export default aiStore;
