const aiCore = {
  ai: {
    agentPrompts: {
      style: {
        title: 'Trạng thái thị trường / phong cách',
        prompt: `Bạn là nhà phân tích nghiên cứu định lượng cấp cao. Dựa trên thông tin dưới đây, hãy đề xuất một mô hình chiến lược chính (trend / mean-reversion / short-term) và giải thích lý do, điều kiện phù hợp và kịch bản không phù hợp.

Yêu cầu đầu ra: Markdown, bắt buộc gồm:
1) Lập luận: bạn suy ra từ dữ liệu/ràng buộc/mục tiêu như thế nào (gạch đầu dòng)
2) Kết luận: 1 mô hình chính (chỉ 1) + phương án thay thế (tùy chọn) + điều kiện phù hợp/không phù hợp
3) Cảnh báo rủi ro: ít nhất 3 ý

{{baseInfo}}`
      },
      consensus: {
        title: 'Đồng thuận & Trò chuyện',
        actions: {
          refresh: 'Làm mới'
        },
        fields: {
          account: 'Tài khoản',
          symbol: 'Mã',
          timeframe: 'Khung thời gian'
        },
        panel: {
          title: 'Điểm khách quan',
          decision: 'Quyết định',
          overallScore: 'Điểm tổng',
          technicalScore: 'Điểm kỹ thuật'
        },
        signals: {
          rsi: {
            value: 'RSI',
            flag: 'Tín hiệu'
          },
          macd: {
            value: 'MACD',
            signalLine: 'Đường tín hiệu',
            hist: 'Histogram',
            flag: 'Tín hiệu',
            trend: 'Xu hướng'
          },
          ma: {
            trend: 'Xu hướng MA'
          }
        }
      },
      signals: {
        title: 'Tín hiệu & chỉ báo',
        prompt: `Bạn là kỹ sư yếu tố & tín hiệu định lượng. Không phụ thuộc dữ liệu bên ngoài (trừ khi người dùng cung cấp bảng sự kiện vĩ mô), hãy thiết kế các tín hiệu giao dịch có thể triển khai.

Yêu cầu: nêu rõ điều kiện vào/ra/bộ lọc, tham số hóa tối đa có thể, tránh overfitting.

Yêu cầu đầu ra: Markdown, bắt buộc gồm:
1) Lập luận: vì sao chọn chỉ báo/ngưỡng/bộ lọc (gạch đầu dòng)
2) Kết luận: danh sách quy tắc thực thi (vào/ra/lọc) và gợi ý tham số (mặc định/phạm vi)
3) Biên & rủi ro: ít nhất 3 ý (vd: sideway/gap/biến động cao/tin tức)

{{baseInfo}}`
      },
      risk: {
        title: 'Rủi ro & ràng buộc thực thi',
        prompt: `Bạn là chuyên gia quản trị rủi ro & thực thi giao dịch. Dựa trên thông tin dưới đây, hãy thiết kế quản lý vị thế, SL/TP, kiểm soát drawdown tối đa, cooldown/giới hạn tần suất giao dịch, v.v.

Yêu cầu đầu ra: Markdown, bắt buộc gồm:
1) Lập luận: vì sao các quy tắc rủi ro này phù hợp mục tiêu/ràng buộc (gạch đầu dòng)
2) Kết luận: ràng buộc cứng (bắt buộc) + tham số mặc định (gợi ý/phạm vi) + hành động khi kích hoạt
3) Mô hình thất bại: ít nhất 3 ý (vd: thua liên tiếp, slippage tăng, spread bất thường)

{{baseInfo}}`
      },
      code: {
        title: 'Sinh mã',
        prompt: `Bạn là kỹ sư mã chiến lược AntTrader Python. Hãy tạo một chiến lược AntTrader Python có thể chạy, yêu cầu:
- Phải qua validate (không import, không dunder, tuân thủ sandbox)
- Dùng API nền tảng như on_tick / on_kline (không tự truy cập mạng/tệp)
- run chỉ nhận đúng 1 tham số: context (tên tham số phải là context; không dùng run(ctx) hay run(context, data))
- run(context) trả về dict, tối thiểu gồm: signal(buy/sell/hold), symbol, confidence(0~1), risk_level(low/medium/high), reason
- Đọc params từ context["params"] (dict do schedule inject); thiếu thì dùng default trong bảng tham số
- Áp dụng gợi ý tín hiệu & rủi ro phía trên (nếu không có, chọn mặc định hợp lý)
- Xuất toàn bộ code và bọc bằng \`\`\`python
- Nghiêm ngặt: chỉ được xuất 1 \`\`\`python code block\`\`\`, không kèm giải thích khác
- Trong code block: chỉ Python thuần; cấm ký hiệu Markdown ("- ", "* ", "###"), cấm dấu câu full-width không ASCII, cấm hàng rào \`\`\`

[Mẫu entry (chép nguyên văn; không đổi tên hàm/số lượng tham số/tên tham số)]
\`\`\`python
def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or params.get("symbol") or ""
    # TODO: implement signal/risk logic here
    return {
        "signal": "hold",
        "symbol": symbol,
        "confidence": 0.5,
        "risk_level": "low",
        "reason": "",
    }
\`\`\`

{{baseInfo}}

[Phân tích upstream (nếu có)]
Hãy đưa kết luận style/signals/risk vào code (nếu không có, chọn mặc định hợp lý).`
      }
    },
    tabs: {
      settings: 'Cài đặt',
      agentSettings: 'Thiết lập chuyên gia',
      gate: 'Cổng AI'
    },
    strategyCard: {
      status: {
        active: 'Đang chạy',
        inactive: 'Đã dừng',
        paused: 'Đã tạm dừng'
      },
      actionType: {
        buy: 'Mua',
        sell: 'Bán',
        closeLong: 'Đóng long',
        closeShort: 'Đóng short',
        alert: 'Nhắc nhở'
      },
      labels: {
        triggeredCount: 'Kích hoạt {{count}} lần',
        lastTriggeredAt: 'Kích hoạt gần nhất: {{time}}'
      },
      sections: {
        conditions: 'Điều kiện kích hoạt',
        actions: 'Hành động'
      },
      tooltips: {
        createdAt: 'Thời gian tạo',
        lastTriggeredAt: 'Kích hoạt gần nhất'
      },
      actions: {
        start: 'Bắt đầu',
        stop: 'Dừng'
      },
      confirmDelete: {
        title: 'Xóa chiến lược này?',
        description: 'Không thể khôi phục sau khi xóa'
      }
    },
    requireConfig: {
      title: 'Chưa cấu hình AI',
      description: 'Vui lòng cấu hình nhà cung cấp, mô hình và API key trong Cài đặt trước khi dùng trình hướng dẫn hoặc trò chuyện.',
      actions: {
        goSettings: 'Đi tới Cài đặt'
      }
    },
    riskEval: {
      failed: 'Đánh giá rủi ro thất bại'
    },
    workflowRuns: {
      title: 'Lịch sử workflow AI',
      defaultTitle: 'Quy trình AI',
      hints: {
        selectToViewDetail: 'Chọn một lần chạy ở bên trái để xem chi tiết'
      },
      messages: {
        loadListFailed: 'Tải danh sách lần chạy thất bại',
        loadDetailFailed: 'Tải chi tiết thất bại'
      }
    },
    client: {
      errors: {
        requestFailed: 'Yêu cầu thất bại, vui lòng thử lại.',
        insufficientBalance: 'Nhà cung cấp báo hết số dư / chưa thanh toán. Vui lòng nạp tiền trong console rồi thử lại.',
        rateLimited: 'Nhà cung cấp đang giới hạn tần suất (yêu cầu quá nhiều). Vui lòng đợi một chút rồi thử lại.',
        unauthorized: 'Nhà cung cấp trả về 401 (Unauthorized). Hãy kiểm tra API key và quyền truy cập mô hình.',
        forbidden: 'Nhà cung cấp trả về 403 (Forbidden). Hãy kiểm tra quyền key, IP allowlist hoặc trạng thái tài khoản.',
        invalidModelId: 'Mô hình không khả dụng{{model}}: có thể không tồn tại, đã ngừng hỗ trợ hoặc ngoài quyền của bạn. Hãy chọn lại hoặc copy id chính xác từ console nhà cung cấp.',
        contextTooLong: 'Yêu cầu vượt độ dài ngữ cảnh tối đa của mô hình. Hãy rút ngắn lịch sử/đầu vào hoặc chọn mô hình có cửa sổ lớn hơn.',
        contentBlocked: 'Nội dung bị bộ lọc an toàn của nhà cung cấp chặn. Vui lòng diễn đạt lại và thử lại.',
        regionNotSupported: 'Khu vực/quốc gia hiện tại không được nhà cung cấp này hỗ trợ. Hãy chuyển sang nhà cung cấp khác.',
        providerInternalError: 'Nhà cung cấp đang gặp lỗi máy chủ (5xx). Vui lòng đợi hoặc chuyển sang nhà cung cấp khác.',
        edgeGatewayTimeout: 'Lỗi timeout tại lớp CDN / reverse proxy (thường là HTTP 524 của Cloudflare): trình duyệt không nhận được phản hồi từ ứng dụng, hay gặp với bước “tạo mã” lâu. Trên màn hình mã của luận điểm hãy dùng “Thử tạo mã lại”, hoặc lùi một bước rồi vào lại bước tạo mã; nếu vẫn lỗi cần tăng timeout proxy/origin.',
        networkUnreachable: 'Cổng AI bị timeout hoặc không thể truy cập. Hãy kiểm tra Base URL, kết nối mạng, hoặc thử lại sau.',
        gatewayTimeoutOrUnreachable: 'Cổng AI bị timeout hoặc không thể truy cập. Vui lòng kiểm tra Base URL trong AI Settings, kết nối mạng, hoặc thử lại sau.',
        gatewayUnauthorized401: 'Cổng AI trả về 401 (Unauthorized). Vui lòng kiểm tra API key và quyền truy cập mô hình.',
        gatewayForbidden403: 'Cổng AI trả về 403 (Forbidden). Vui lòng kiểm tra quyền key, IP allowlist hoặc trạng thái tài khoản.',
        gatewayRateLimited429: 'Cổng AI bị giới hạn tần suất (429). Vui lòng thử lại sau.'
      }
    },
    chatBox: {
      title: 'Trò chuyện AI',
      conversations: 'Hội thoại',
      empty: 'Chưa có hội thoại. Nhấn "+" để tạo mới.',
      emptyDescription: 'Bắt đầu trò chuyện với trợ lý AI',
      untitled: 'Chưa đặt tên',
      newConversation: 'Cuộc trò chuyện mới',
      messageCount: '{{count}} tin nhắn',
      thinking: 'Đang suy nghĩ...',
      truncated: 'Nội dung quá dài và đã bị cắt bớt',
      expandAll: 'Mở rộng tất cả',
      collapse: 'Thu gọn',
      delete: {
        title: 'Xóa hội thoại',
        content: 'Xóa hội thoại này? Không thể khôi phục.'
      }
    },
    conversation: {
      defaultTitle: 'Hội thoại mới'
    },
    reports: {
      tradeAnalysis: {
        title: 'Báo cáo phân tích giao dịch AI',
        riskAssessmentPrefix: 'Đánh giá rủi ro:'
      }
    },
    signalCard: {
      status: {
        pending: 'Chờ xác nhận',
        confirmed: 'Đã xác nhận',
        executed: 'Đã thực thi',
        cancelled: 'Đã hủy'
      },
      labels: {
        price: 'Giá',
        volume: 'Khối lượng',
        confidence: 'Độ tin cậy',
        stopLoss: 'Cắt lỗ',
        takeProfit: 'Chốt lời',
        analysisReason: 'Phân tích'
      },
      actions: {
        confirm: 'Xác nhận',
        cancel: 'Hủy',
        executeTrade: 'Thực thi giao dịch'
      },
      confirmCancel: {
        title: 'Hủy tín hiệu này?'
      },
      confirmExecute: {
        title: 'Thực thi tín hiệu giao dịch này?',
        description: 'Thao tác này sẽ đặt lệnh ngay lập tức'
      }
    },
    assistant: {
      chat: {
        title: 'Trò chuyện AI',
        conversations: 'Hội thoại',
        empty: 'Chưa có hội thoại. Nhấn "+" để tạo mới.',
        untitled: 'Chưa đặt tên',
        newConversation: 'Cuộc trò chuyện mới',
        messageCount: '{{count}} tin nhắn',
        delete: {
          title: 'Xóa hội thoại',
          content: 'Xóa hội thoại này? Không thể khôi phục.'
        }
      },
      input: {
        placeholder: 'Nhập tin nhắn...'
      },
      actions: {
        generateStrategyAutoValidate: 'Tạo chiến lược (tự xác minh)',
        applyCodeToEditor: 'Áp dụng code vào trình soạn thảo'
      },
      currentModel: {
        label: 'Mô hình hiện tại: ',
        notConfigured: 'Chưa cấu hình'
      },
      configWarning: {
        title: 'Chưa cấu hình mô hình',
        description: 'Vui lòng cấu hình nhà cung cấp, mô hình và API key trước khi trò chuyện.',
        action: 'Cấu hình'
      },
      settingsModal: {
        title: 'Cài đặt AI'
      },
      clearChat: {
        title: 'Xóa hội thoại',
        content: 'Xóa toàn bộ lịch sử hội thoại?'
      },
      autoGeneratePrompt: {
        initial: {
          title: 'Vui lòng tạo mã chiến lược AntTrader Python có thể chạy. Yêu cầu:',
          rules: {
            validate: '- Phải vượt qua validate (không import, không dunder, tuân thủ ràng buộc run(context), v.v.)',
            entry: '- Phải định nghĩa run(context) hoặc signal (khuyến nghị run(context))',
            outputShape: '- run(context) trả về dict, ít nhất gồm: signal(buy/sell/hold), symbol, confidence(0~1), risk_level(low/medium/high), reason',
            outputFence: '- Chỉ xuất toàn bộ mã và bọc bằng \`\`\`python'
          }
        },
        noCodeBlock: {
          title: 'Bạn chưa xuất code block theo yêu cầu. Vui lòng xuất lại toàn bộ mã chiến lược:',
          rules: {
            outputFence: '- Bọc bằng \`\`\`python',
            noImport: '- Không được có import',
            validate: '- Phải vượt qua validate'
          }
        },
        fixByErrors: {
          title: 'Chiến lược bạn tạo không vượt qua validate. Vui lòng sửa theo lỗi và xuất toàn bộ mã:',
          sections: {
            validateErrors: '【validate errors】',
            currentCode: '【Mã hiện tại】'
          },
          outputRequirement: 'Yêu cầu đầu ra: chỉ xuất toàn bộ mã sau khi sửa (bọc bằng \`\`\`python).'
        }
      },
      messages: {
        noCodeBlockFound: 'Không tìm thấy khối code (\`\`\`...\`\`\`)',
        appliedToEditorAutoValidating: 'Đã áp dụng vào trình soạn thảo. Đang tự xác minh...',
        validationPassedWithWarning: 'Xác minh thành công nhưng có cảnh báo: {{warning}}',
        codeValidationPassed: 'Xác minh code thành công',
        codeValidationFailed: 'Xác minh code thất bại',
        autoValidationFailed: 'Tự xác minh thất bại',
        generatedAndApplied: 'Tạo thành công và đã áp dụng (đã xác minh)',
        validateNotPassed: 'Validate không đạt',
        autoGenerateStillFailed: 'Tự tạo nhiều lần vẫn không đạt. Vui lòng chỉnh thủ công.',
        autoGenerateFailed: 'Tự tạo thất bại'
      }
    },
    store: {
      strategyRules: {
        title: 'Khi viết mã chiến lược AntTrader Python, bạn phải tuân thủ nghiêm các quy tắc xác thực sau:',
        rules: {
          noImport: '- Cấm import / from ... import ...',
          noGlobal: '- Cấm global / nonlocal',
          noDunderAccess: '- Cấm truy cập thuộc tính dunder (vd: obj.__xxx__)',
          noDunderName: '- Cấm dùng tên dunder (vd: __xxx__)',
          noDangerousCalls: '- Cấm gọi: open()/eval()/exec()/compile()/__import__()/input()/globals()/locals()/vars()/dir()',
          runSignature: '- Nếu định nghĩa run: chỉ được có một run(context), đúng 1 tham số tên context; cấm *args/**kwargs',
          mustDefineEntry: '- Chiến lược phải có biến signal hoặc run(context) (khuyến nghị run(context))'
        },
        allowedGlobals: 'Được phép dùng global/module: np, math, datetime, calculate_rsi (không import).'
      },
      context: {
        userPrefsTitle: 'Sở thích người dùng (vui lòng cố gắng tuân theo):',
        outputTitle: 'Yêu cầu đầu ra:',
        outputRules: {
          wrapPython: '- Nếu xuất mã chiến lược, hãy xuất toàn bộ mã và bọc bằng \`\`\`python',
          validateFirst: '- Mã phải ưu tiên vượt qua validate',
          noImport: '- Không xuất bất kỳ câu lệnh import nào'
        }
      },
      prefs: {
        rememberPrefix: 'Ghi nhớ sở thích:',
        rememberedToast: 'Đã ghi nhớ sở thích. Sẽ áp dụng cho các cuộc trò chuyện sau.',
        savedReply: 'Đã lưu sở thích'
      },
      conversations: {
        newConversationTitle: 'Cuộc trò chuyện mới'
      },
      messages: {
        sendFailedInline: 'Gửi thất bại, vui lòng thử lại',
        sendFailedToast: 'Gửi thất bại, vui lòng thử lại',
        createConversationFailed: 'Tạo cuộc trò chuyện thất bại',
        loadConversationFailed: 'Tải cuộc trò chuyện thất bại',
        deleteConversationFailed: 'Xóa cuộc trò chuyện thất bại',
        clearedLocalOnly: 'Đã xóa tin nhắn cuộc trò chuyện hiện tại (lịch sử server vẫn giữ)',
        getReportsFailed: 'Không thể lấy báo cáo',
        generateReportSuccess: 'Tạo báo cáo thành công',
        generateReportFailed: 'Tạo báo cáo thất bại'
      }
    },
    backtestScoreCard: {
      title: 'Thẻ điểm backtest',
      stateLabel: 'Trạng thái',
      status: {
        succeeded: 'Thành công',
        running: 'Đang chạy',
        pending: 'Đang chờ',
        failed: 'Thất bại',
        cancelRequested: 'Đang hủy',
        canceled: 'Đã hủy'
      },
      recommendation: {
        loading: 'Đang đánh giá rủi ro. Vui lòng chờ xong trước khi triển khai.',
        recommended: 'Khuyến nghị triển khai: rủi ro trong mức kiểm soát, chỉ số nhìn chung tốt.',
        cautious: 'Triển khai thận trọng: bắt đầu nhỏ / theo dõi thủ công một thời gian.',
        notRecommended: 'Không khuyến nghị triển khai ngay: rủi ro cao hoặc không đáng tin cậy. Hãy tối ưu rồi thử lại.'
      },
      backendRiskScore: {
        title: 'Điểm rủi ro từ backend',
        loading: 'Đang tính...',
        unknown: 'không xác định',
        reliable: 'Đáng tin cậy',
        unreliable: 'Không đáng tin cậy',
        reasons: 'Lý do',
        warnings: 'Cảnh báo',
        empty: 'Chưa có (hãy lưu template trước; sẽ tự tính sau khi backtest xong)'
      },
      score: {
        empty: 'Chưa có điểm (đợi backtest hoàn tất hoặc thiếu metrics)',
        title: 'Điểm tổng hợp (heuristic phía frontend)'
      },
      level: {
        excellent: 'Xuất sắc',
        good: 'Tốt',
        fair: 'Trung bình',
        poor: 'Kém'
      },
      metrics: {
        totalReturn: 'Tổng lợi nhuận',
        annualReturn: 'Lợi nhuận năm',
        maxDrawdown: 'Sụt giảm tối đa',
        sharpe: 'Sharpe',
        winRate: 'Tỷ lệ thắng',
        totalTrades: 'Số lệnh',
        equityPoints: 'Điểm equity'
      },
      chart: {
        title: 'Đường cong equity'
      }
    },
    systemAI: {
      taglines: {
        openai: 'Dòng GPT · Chính thức',
        anthropic: 'Dòng Claude',
        deepseek: 'DeepSeek · Hiệu quả chi phí',
        moonshot: 'Kimi · Ngữ cảnh dài',
        qwen: 'Alibaba Cloud · Tối ưu tiếng Trung',
        zhipu: 'Hệ Thanh Hoa · Đa dụng',
        openai_compatible: 'Bất kỳ endpoint tương thích OpenAI'
      },
      pageTitle: 'Cài đặt trợ lý AI',
      pageSubtitle: 'Cấu hình bộ não AI — chọn nhà cung cấp, quản lý API key và model khả dụng, và chỉ định "Mô hình chính mặc định" dùng cho toàn hệ thống.',
      emptyConfigs: 'Chưa có cấu hình AI Provider (hệ thống sẽ tạo provider mặc định khi khởi động).',
      section1: {
        title: 'Chọn nhà cung cấp model',
        subtitle: 'Mỗi thẻ hiển thị cấu hình và trạng thái sẵn sàng của một nhà cung cấp; nhấp để chọn.'
      },
      status: {
        noProvider: 'Chưa chọn nhà cung cấp',
        noProviderDesc: 'Chọn một nhà cung cấp trong các thẻ bên dưới để bắt đầu',
        error: 'Có lỗi',
        ready: 'Sẵn sàng',
        readyDesc: 'đã bật và kết nối bình thường',
        notEnabled: 'Đã kết nối nhưng chưa bật',
        notEnabledDesc: 'Bật công tắc Enable để đưa vào sử dụng',
        configReady: 'Đã sẵn sàng cấu hình',
        configReadyDesc: 'Thêm model khả dụng để hệ thống tự kiểm tra kết nối',
        checkUrl: 'Kiểm tra Base URL',
        checkUrlDesc: 'Đã có API key nhưng URL có vẻ không hợp lệ',
        needKey: 'Cần cấu hình API key',
        needKeyDesc: 'Nhập API key, hệ thống sẽ tự phát hiện danh sách model',
        connectionFailed: 'Kết nối thất bại — xem cảnh báo phía trên'
      },
      cardState: {
        noKey: 'Chưa cấu hình',
        noModel: 'Chọn model',
        enabled: 'Đã bật',
        readyDisabled: 'Sẵn sàng · chưa bật'
      },
      cardTags: {
        current: 'Hiện tại',
        hasKey: 'Đã có key',
        noKey: 'Chưa có key',
        noModels: 'Chưa cấu hình model khả dụng',
        enabledButUnavailable: 'Đã bật nhưng không khả dụng'
      },
      statusBar: {
        enabled: 'Đã bật',
        disabled: 'Chưa bật',
        keyReady: 'Key sẵn sàng',
        checking: 'Đang kiểm tra kết nối…',
        connected: 'Kết nối bình thường'
      },
      fields: {
        autoFetching: 'Đang tự lấy danh sách…',
        baseUrlCustomHint: 'Nhập endpoint tương thích OpenAI, ví dụ https://model.example.com/v1',
        baseUrlReadonlyHint: 'Địa chỉ chính thức do hệ thống quản lý, không sửa được',
        baseUrlCustomPlaceholder: 'Ví dụ: https://model.example.com/v1',
        baseUrlReadonlyPlaceholder: 'Địa chỉ chính thức (chỉ đọc)',
        httpWarning: 'Đang dùng HTTP — môi trường sản xuất nên dùng HTTPS',
        apiKeyHint: 'Sau khi nhập sẽ được mã hoá và lưu tự động, không cần submit thủ công',
        apiKeyPastePlaceholder: 'Dán API key — sẽ được lưu trước tự động',
        enabledHint: 'Tắt thì nhà cung cấp này sẽ không được hệ thống sử dụng',
        temperatureHint: 'Cao hơn = đa dạng hơn, thấp hơn = ổn định hơn',
        timeoutHint: 'Thời gian chờ tối đa cho mỗi request',
        maxTokensHint: 'Token tối đa cho mỗi phản hồi',
        primaryFor: 'Mục đích chính (Primary For)',
        primaryForHint: 'Chỉ dùng cho định tuyến nội bộ: chat / embedding / summarizer / reasoning'
      },
      messages: {
        loadConfigFailed: 'Tải cấu hình thất bại',
        secretSavedAutoDiscover: 'Đã lưu khóa bí mật, đang tự động khám phá mô hình...',
        secretAutoSaveFailed: 'Tự động lưu khóa bí mật thất bại',
        autoDiscoveredModels: 'Đã tự động khám phá {{count}} mô hình (chỉ để gợi ý)',
        autoValidatedModels: 'Đã tự động xác thực: tìm thấy {{count}} mô hình',
        configSaved: 'Đã lưu cấu hình',
        configSaveFailed: 'Lưu cấu hình thất bại',
        toggleEnabledFailed: 'Thay đổi trạng thái bật/tắt thất bại',
        secretDeletedConfigReset: 'Đã xóa khóa bí mật, cấu hình nhà cung cấp đặt lại về mặc định',
        deleteSecretFailed: 'Xóa khóa bí mật thất bại',
        validationPassedModels: 'Xác thực thành công: tìm thấy {{count}} mô hình',
        validationFailedNeedApiKey: 'Xác thực thất bại: nhà cung cấp này thường yêu cầu khóa API. Vui lòng điền và lưu khóa trước, sau đó thử lại.'
      }
    },
    consensus: {
      title: 'Đồng thuận & Thảo luận',
      actions: {
        refresh: 'Làm mới'
      },
      fields: {
        account: 'Tài khoản',
        symbol: 'Mã chứng khoán',
        timeframe: 'Khung thời gian'
      },
      panel: {
        title: 'Điểm mục tiêu',
        decision: 'Quyết định',
        overallScore: 'Tổng thể',
        technicalScore: 'Kỹ thuật'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: 'Tín hiệu'
        },
        macd: {
          value: 'MACD',
          signalLine: 'Đường tín hiệu',
          hist: 'Biểu đồ cột',
          flag: 'Tín hiệu',
          trend: 'Mô hình'
        },
        ma: {
          trend: 'Xu hướng MA'
        }
      }
    },
    gate: {
      title: 'Tiến trình Cổng AI',
      pipelineDesc: 'Quy trình 6 cổng: Tuân thủ -> Nhìn về phía trước -> Walk-Forward -> Deflated Sharpe -> Giấy -> Tương quan',
      labels: {
        compliance: 'Tuân thủ',
        lookahead: 'Độ chệch nhìn về phía trước',
        walkforward: 'Walk-Forward',
        deflated_sharpe: 'Deflated Sharpe',
        paper: 'Giao dịch giấy',
        correlation: 'Tương quan'
      },
      descriptions: {
        compliance: 'Xác thực biểu thức DSL không rỗng',
        lookahead: 'Quét tham chiếu hàm tương lai (close[t+N], ref offset âm)',
        walkforward: 'Xác thực chéo Walk-Forward đã thanh lọc',
        deflated_sharpe: 'Tỷ lệ Sharpe đã giảm phát của Lopez de Prado',
        paper: 'Xác thực giao dịch giấy >=14 ngày',
        correlation: 'Kiểm tra tương quan tín hiệu với các chiến lược hiện có'
      },
      status: {
        evaluating: 'Đang đánh giá...'
      },
      strategyParams: 'Tham số chiến lược',
      dslExpression: 'Biểu thức DSL',
      dailyReturns: 'Lợi nhuận hàng ngày (phân cách bằng dấu phẩy hoặc xuống dòng)',
      numAttempts: 'Số lần thử chiến lược',
      paperMetrics: 'Chỉ số giao dịch giấy',
      paperDays: 'Ngày giao dịch giấy',
      paperNetPnL: 'Lãi/Lỗ ròng giấy',
      paperNetReturn: 'Lợi nhuận ròng giấy',
      paperTradeCount: 'Số giao dịch giấy',
      backtestNetReturn: 'Lợi nhuận ròng backtest',
      backtestGrossReturn: 'Lợi nhuận gộp backtest',
      runPipeline: 'Chạy quy trình Cổng',
      retry: 'Thử lại',
      gateProgress: 'Tiến trình đánh giá Cổng',
      pipelineResult: 'Kết quả quy trình',
      allPassed: 'Cả 6 cổng đều vượt qua — chiến lược đủ điều kiện đánh giá PromoteToLive',
      failed: 'Thất bại: {{gate}}',
      details: 'Chi tiết',
      skipped: 'SKIPPED',
      noData: 'không có dữ liệu',
      pass: 'PASS',
      fail: 'FAIL',
      unknown: 'không xác định',
      selectRun: 'Chọn lần chạy kiểm thử lùi...'
    }
  }
} as const;

export default aiCore;
