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
        getReportsFailed: 'Lấy báo cáo thất bại',
        generateReportSuccess: 'Báo cáo đã được tạo thành công',
        generateReportFailed: 'Tạo báo cáo thất bại'
      },
      prefs: {
        rememberPrefix: '/remember ',
        rememberedToast: 'Đã lưu tùy chọn',
        savedReply: 'Đã lưu tùy chọn của bạn.'
      },
      strategyRules: {
        title: 'Khi viết mã chiến lược Python AntTrader, bạn phải tuân thủ nghiêm ngặt các quy tắc xác thực sau:',
        rules: {
          noImport: '- Không được phép import / from ... import ...',
          noGlobal: '- Không được phép global / nonlocal',
          noDunderAccess: '- Không truy cập thuộc tính dunder (obj.__xxx__)',
          noDunderName: '- Không tên dunder (__xxx__)',
          noDangerousCalls: '- Không gọi: open()/eval()/exec()/compile()/__import__()/input()/globals()/locals()/vars()/dir()',
          runSignature: '- Nếu định nghĩa hàm run: chỉ một run(context), phải có chính xác 1 tham số context, không *args/**kwargs',
          mustDefineEntry: '- Chiến lược phải định nghĩa biến signal hoặc hàm run(context) (ưu tiên run(context))'
        },
        allowedGlobals: 'Toàn cục/mô-đun được phép: np, math, datetime, calculate_rsi (không import).'
      },
      context: {
        userPrefsTitle: 'Tùy chọn người dùng (vui lòng tuân thủ càng nhiều càng tốt):',
        outputTitle: 'Yêu cầu đầu ra:',
        outputRules: {
          wrapPython: '- Nếu xuất mã chiến lược, xuất mã đầy đủ trong \`\`\`python',
          validateFirst: '- Mã phải vượt qua xác thực trước',
          noImport: '- Không xuất bất kỳ câu lệnh import nào'
        }
      }
    }
  },
  ai_store: {
    ai: {
      store: {
        strategyRules: {
          title: 'Khi viết mã chiến lược Python AntTrader, bạn phải tuân thủ nghiêm ngặt các quy tắc xác thực sau:',
          rules: {
            noImport: '- Không được phép import / from ... import ...',
            noGlobal: '- Không được dùng global / nonlocal',
            noDunderAccess: '- Không được truy cập thuộc tính dunder (obj.__xxx__)',
            noDunderName: '- Không được dùng tên dunder (__xxx__)',
            noDangerousCalls: '- Không được gọi: open()/eval()/exec()/compile()/__import__()/input()/globals()/locals()/vars()/dir()',
            runSignature: '- Nếu định nghĩa hàm run: chỉ một run(context), phải có đúng 1 tham số context, không *args/**kwargs',
            mustDefineEntry: '- Chiến lược phải định nghĩa biến signal hoặc hàm run(context) (ưu tiên run(context))'
          },
          allowedGlobals: 'Các module/biến toàn cục được phép: np, math, datetime, calculate_rsi (không cần import).'
        },
        context: {
          userPrefsTitle: 'Sở thích người dùng (vui lòng tuân theo càng nhiều càng tốt):',
          outputTitle: 'Yêu cầu đầu ra:',
          outputRules: {
            wrapPython: '- Nếu xuất mã chiến lược, hãy xuất toàn bộ mã được bọc trong \`\`\`python',
            validateFirst: '- Mã phải vượt qua xác thực trước tiên',
            noImport: '- Không xuất bất kỳ câu lệnh import nào'
          }
        },
        messages: {
          getReportsFailed: 'Lấy báo cáo thất bại',
          generateReportSuccess: 'Tạo báo cáo thành công',
          generateReportFailed: 'Tạo báo cáo thất bại'
        }
      }
    }
  }
} as const;

export default aiStore;
