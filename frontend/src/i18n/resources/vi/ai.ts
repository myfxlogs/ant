const ai = {
  ai: {
    agentPrompts: {
      style: {
        title: 'Trạng thái thị trường / phong cách',
        prompt: 'Bạn là nhà phân tích nghiên cứu định lượng cấp cao. Dựa trên thông tin dưới đây, hãy đề xuất một mô hình chiến lược chính (trend / mean-reversion / short-term) và giải thích lý do, điều kiện phù hợp và kịch bản không phù hợp.

Yêu cầu đầu ra: Markdown, bắt buộc gồm:
1) Lập luận: bạn suy ra từ dữ liệu/ràng buộc/mục tiêu như thế nào (gạch đầu dòng)
2) Kết luận: 1 mô hình chính (chỉ 1) + phương án thay thế (tùy chọn) + điều kiện phù hợp/không phù hợp
3) Cảnh báo rủi ro: ít nhất 3 ý

{{baseInfo}}'
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
        prompt: 'Bạn là kỹ sư yếu tố & tín hiệu định lượng. Không phụ thuộc dữ liệu bên ngoài (trừ khi người dùng cung cấp bảng sự kiện vĩ mô), hãy thiết kế các tín hiệu giao dịch có thể triển khai.

Yêu cầu: nêu rõ điều kiện vào/ra/bộ lọc, tham số hóa tối đa có thể, tránh overfitting.

Yêu cầu đầu ra: Markdown, bắt buộc gồm:
1) Lập luận: vì sao chọn chỉ báo/ngưỡng/bộ lọc (gạch đầu dòng)
2) Kết luận: danh sách quy tắc thực thi (vào/ra/lọc) và gợi ý tham số (mặc định/phạm vi)
3) Biên & rủi ro: ít nhất 3 ý (vd: sideway/gap/biến động cao/tin tức)

{{baseInfo}}'
      },
      risk: {
        title: 'Rủi ro & ràng buộc thực thi',
        prompt: 'Bạn là chuyên gia quản trị rủi ro & thực thi giao dịch. Dựa trên thông tin dưới đây, hãy thiết kế quản lý vị thế, SL/TP, kiểm soát drawdown tối đa, cooldown/giới hạn tần suất giao dịch, v.v.

Yêu cầu đầu ra: Markdown, bắt buộc gồm:
1) Lập luận: vì sao các quy tắc rủi ro này phù hợp mục tiêu/ràng buộc (gạch đầu dòng)
2) Kết luận: ràng buộc cứng (bắt buộc) + tham số mặc định (gợi ý/phạm vi) + hành động khi kích hoạt
3) Mô hình thất bại: ít nhất 3 ý (vd: thua liên tiếp, slippage tăng, spread bất thường)

{{baseInfo}}'
      },
      code: {
        title: 'Sinh mã',
        prompt: 'Bạn là kỹ sư mã chiến lược AntTrader Python. Hãy tạo một chiến lược AntTrader Python có thể chạy, yêu cầu:
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
Hãy đưa kết luận style/signals/risk vào code (nếu không có, chọn mặc định hợp lý).'
      }
    },
    tabs: {
      settings: 'Cài đặt',
      agentSettings: 'Thiết lập chuyên gia',
      gate: 'AI Gate'
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
        unknown: 'unknown',
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
        loadConfigFailed: 'Failed to load configs',
        secretSavedAutoDiscover: 'Secret saved, auto-discovering models...',
        secretAutoSaveFailed: 'Secret auto-save failed',
        autoDiscoveredModels: 'Auto-discovered {{count}} model(s) (for suggestion only)',
        autoValidatedModels: 'Auto-validated: {{count}} model(s) found',
        configSaved: 'Config saved',
        configSaveFailed: 'Config save failed',
        toggleEnabledFailed: 'Toggle enabled status failed',
        secretDeletedConfigReset: 'Secret deleted, provider config reset to defaults',
        deleteSecretFailed: 'Delete secret failed',
        validationPassedModels: 'Validation passed: {{count}} model(s) found',
        validationFailedNeedApiKey: 'Validation failed: this provider typically requires an API Key. Please fill and save the key first, then retry.'
      }
    },
    wizard: {
      title: 'Trình hướng dẫn chiến lược AI',
      subtitle: 'Mỗi bước một trang, bạn có thể tiến/lùi',
      currentModel: 'Mô hình hiện tại: {{model}}',
      rangePresets: {
        '1d': '1 ngày gần nhất',
        '3d': '3 ngày gần nhất',
        '7d': '1 tuần gần nhất',
        '30d': '1 tháng gần nhất',
        '90d': '3 tháng gần nhất'
      },
      steps: {
        setup: 'Thiết lập',
        generate: 'Tạo chiến lược',
        publishCode: 'Triển khai - Mã',
        publishBacktest: 'Triển khai - Backtest',
        publishLaunch: 'Triển khai - Khởi chạy'
      },
      actions: {
        prev: 'Trước',
        next: 'Tiếp',
        cancel: 'Hủy'
      },
      generate: {
        cards: {
          resultsTitle: 'Kết quả chuyên gia'
        },
        actions: {
          runAgents: 'Phân tích chuyên gia + sinh mã',
          hide: 'Ẩn',
          abort: 'Hủy',
          rerun: 'Chạy lại',
          regenerateSummary: 'Tạo lại tóm tắt',
          goValidate: 'Đi xác thực'
        },
        hints: {
          afterGenerated: 'Sau khi tạo xong, sang bước tiếp theo để xác thực/backtest/triển khai.'
        },
        labels: {
          elapsed: 'Thời gian'
        },
        status: {
          inProgress: 'Đang chạy',
          done: 'Hoàn tất',
          error: 'Thất bại',
          idle: 'Đang chờ',
          running: {
            style: 'Đang phân tích trạng thái/phong cách thị trường',
            signals: 'Đang thiết kế tín hiệu/chỉ báo',
            risk: 'Đang thiết kế rủi ro/ràng buộc thực thi',
            code: 'Đang sinh mã',
            generic: '{{title}} đang chạy'
          }
        },
        sections: {
          prompt: 'Prompt gửi tới mô hình',
          output: 'Kết quả mô hình',
          spec: 'Đặc tả'
        },
        modals: {
          final: {
            title: 'Đã sinh mã. Khuyến nghị nhấn “Xác thực mã” để xác nhận.'
          }
        }
      },
      publish: {
        cards: {
          codeTitle: '1) Mã chiến lược (có thể chỉnh sửa)',
          scoreCardTitle: '2) Thẻ điểm backtest',
          launchTitle: '3) Triển khai lịch chạy'
        },
        placeholders: {
          codeEditable: 'Mã do AI tạo sẽ xuất hiện ở đây. Bạn cũng có thể chỉnh sửa thủ công.'
        },
        actions: {
          validateCode: 'Xác thực mã',
          startBacktest: 'Backtest (tác vụ bất đồng bộ)',
          publishTemplate: 'Triển khai template'
        },
        messages: {
          validateOk: 'validate thành công',
          validateFailed: 'validate thất bại'
        }
      },
      agents: {
        styleTitle: 'Trạng thái thị trường / phong cách',
        signalsTitle: 'Tín hiệu & chỉ báo',
        riskTitle: 'Rủi ro & ràng buộc thực thi',
        codeTitle: 'Sinh mã'
      },
      template: {
        defaultName: 'Chiến lược AI {{title}}',
        defaultDescription: 'Tạo bởi trình hướng dẫn AI'
      },
      schedule: {
        defaultName: 'Lịch AI {{symbol}} {{timeframe}}'
      },
      setup: {
        cards: {
          tradeAndDataTitle: 'Giao dịch & dữ liệu',
          constraintsAndGoalTitle: 'Ràng buộc & mục tiêu',
          hardConstraintsTitle: 'Ràng buộc cứng',
          hintsTitle: 'Gợi ý'
        },
        labels: {
          account: 'Tài khoản',
          symbol: 'Mã',
          timeframe: 'Khung thời gian',
          historicalData: 'Dữ liệu lịch sử',
          backtestRange: 'Phạm vi backtest',
          dataset: 'Dataset đóng băng',
          maxDrawdownPct: 'Sụt giảm tối đa (%)',
          riskPerTradePct: 'Rủi ro mỗi lệnh (%)',
          maxTradesPerDay: 'Số lệnh tối đa mỗi ngày',
          macroModule: 'Mô-đun vĩ mô',
          macroEvents: 'Sự kiện vĩ mô',
          intent: 'Ý định chiến lược'
        },
        placeholders: {
          selectAccount: 'Chọn tài khoản',
          selectSymbol: 'Chọn mã',
          selectTimeframe: 'Chọn khung thời gian',
          selectFrozenDataset: 'Chọn dataset đóng băng',
          macroExample: 'Ví dụ:
2024-01-03 21:15 FOMC Minutes
2024-01-05 20:30 NFP',
          intentExample: 'Ví dụ: Theo xu hướng khi phá vỡ; tránh biến động cao; ưu tiên tỷ lệ thắng...'
        },
        validations: {
          selectAccount: 'Vui lòng chọn tài khoản',
          selectSymbol: 'Vui lòng chọn mã',
          selectTimeframe: 'Vui lòng chọn khung thời gian',
          selectDataset: 'Vui lòng chọn dataset',
          enterIntent: 'Vui lòng nhập ý định chiến lược'
        },
        dataModes: {
          klineRange: 'Phạm vi nến',
          dataset: 'Dataset đóng băng'
        },
        actions: {
          refreshDataset: 'Làm mới',
          freezeFromCurrentRange: 'Đóng băng từ phạm vi hiện tại',
          deleteCurrentDataset: 'Xóa dataset hiện tại'
        },
        modals: {
          deleteDataset: {
            title: 'Xóa dataset',
            content: 'Xóa dataset đóng băng đang chọn?',
            ok: 'Xóa'
          }
        },
        messages: {
          datasetDeleted: 'Đã xóa dataset'
        },
        macro: {
          off: 'Tắt',
          on: 'Bật'
        },
        hints: {
          nextWillGenerateCode: 'Bước tiếp theo sẽ tạo mã chiến lược.',
          tradeDataNextStep: 'Sau khi điền xong, nhấn “Tiếp” để tiếp tục thiết lập ràng buộc & mục tiêu.'
        }
      },
      publishBacktest: {
        cards: {
          backtestTitle: 'Backtest',
          scoreCardTitle: 'Thẻ điểm'
        },
        actions: {
          startBacktest: 'Bắt đầu backtest',
          close: 'Đóng',
          retry: 'Thử lại',
          succeeded: 'Thành công',
          inProgress: 'Đang chạy',
          runInBackground: 'Chạy nền',
          confirm: 'Xác nhận'
        },
        labels: {
          status: 'Trạng thái',
          elapsed: 'Thời gian',
          scoringProgress: 'Tiến độ chấm điểm',
          overallScore: 'Điểm tổng',
          confirmed: 'Đã xác nhận'
        },
        modals: {
          status: {
            title: 'Backtest đang chạy'
          },
          score: {
            title: 'Xác nhận điểm số'
          }
        },
        draftName: 'Backtest {{datetime}} {{symbol}} {{timeframe}}',
        draftNameShort: 'Backtest {{symbol}} {{timeframe}}'
      },
      strategyParams: {
        title: 'Tham số chiến lược (tùy chọn)',
        hints: {
          intro: 'Các tham số này sẽ:',
          line1: '1) được lưu vào template.parameters',
          line2: '2) được ghi vào schedule.parameters (map<string,string>) khi tạo lịch',
          line3Prefix: '3) được tiêm vào chiến lược Python khi chạy dưới dạng'
        },
        actions: {
          addParam: 'Thêm tham số',
          exportJson: 'Xuất JSON',
          importJson: 'Nhập JSON',
          delete: 'Xóa'
        },
        empty: 'Chưa có tham số. Bạn có thể thêm fast/slow/risk_per_trade... để chiến lược dễ tái sử dụng.',
        paramCardTitle: 'Tham số #{{index}}',
        labels: {
          name: 'name',
          type: 'type',
          value: 'value (giá trị hiện tại của lịch)',
          default: 'default',
          min: 'min',
          max: 'max',
          step: 'step',
          label: 'label',
          description: 'description',
          options: 'options (dùng cho select, phân tách bằng dấu phẩy)'
        },
        validations: {
          nameRequired: 'name là bắt buộc',
          typeRequired: 'type là bắt buộc'
        },
        placeholders: {
          nameExample: 'vd: fast',
          value: 'Để trống sẽ dùng default',
          defaultExample: 'vd: 10',
          label: 'Tên hiển thị',
          description: 'Mô tả',
          optionsExample: 'vd: low,medium,high',
          importJson: 'Dán JSON tham số (mảng hoặc {"paramDefs": [...]})'
        },
        modals: {
          exportTitle: 'Xuất JSON tham số',
          importTitle: 'Nhập JSON tham số',
          copyAndClose: 'Sao chép và đóng',
          importOk: 'Nhập'
        },
        messages: {
          jsonParseFailed: 'Phân tích JSON thất bại',
          importFormatInvalid: 'Định dạng nhập không hợp lệ: cần mảng hoặc {"paramDefs": [...] }',
          importMissingName: 'Nhập thất bại: có mục thiếu name',
          imported: 'Đã nhập {{count}} tham số',
          copied: 'Đã sao chép',
          copyFailed: 'Sao chép thất bại'
        },
        types: {
          number: 'số',
          string: 'chuỗi',
          bool: 'bool',
          select: 'chọn'
        }
      },
      prompts: {
        dataSpec: {
          dataset: 'Sử dụng dataset đã đóng băng datasetId={{datasetId}}',
          klineRange: 'Sử dụng phạm vi nến lịch sử from={{from}} to={{to}}'
        },
        base: {
          account: 'Tài khoản: {{accountId}}',
          symbol: 'Mã: {{symbol}}',
          timeframe: 'Khung thời gian: {{timeframe}}',
          data: 'Dữ liệu: {{dataSpec}}',
          constraints: 'Ràng buộc: max drawdown={{maxDrawdownPct}}% rủi ro/lệnh={{riskPerTradePct}}% tối đa lệnh/ngày={{maxTradesPerDay}}',
          params: 'Tham số (định nghĩa + giá trị hiện tại; có trong context["params"] khi chạy):
{{params}}',
          empty: '(trống)',
          macroEnabled: 'Sự kiện vĩ mô (người dùng cung cấp):
{{text}}',
          macroDisabled: 'Sự kiện vĩ mô: không dùng',
          userIntent: 'Mục tiêu (ngôn ngữ tự nhiên):
{{intent}}'
        },
        upstream: {
          style: '[Kết luận trạng thái thị trường / phong cách]
{{text}}',
          signals: '[Kết luận tín hiệu & chỉ báo]
{{text}}',
          risk: '[Kết luận rủi ro & ràng buộc]
{{text}}',
          sectionTitle: '[Kết luận agent phía trên (nguyên văn)]'
        },
        summary: {
          intro: 'Bạn là trợ lý giải thích chiến lược định lượng. Hãy giải thích ý tưởng cốt lõi của đoạn mã chiến lược AntTrader Python dưới đây bằng các gạch đầu dòng ngắn gọn (tối đa 12 dòng) để giúp người dùng đánh giá có đúng kỳ vọng hay không.',
          mustIncludeTitle: 'Bắt buộc gồm:',
          mustInclude1: '1) Loại/kiểu chiến lược (trend/mean-reversion/breakout/momentum/grid... nếu không chắc hãy ghi “Không rõ”)',
          mustInclude2: '2) Điều kiện vào lệnh chính (2-4 ý)',
          mustInclude3: '3) Điều kiện thoát/SL/TP/ràng buộc rủi ro chính (2-4 ý)',
          mustInclude4: '4) 1 bối cảnh phù hợp và 1 bối cảnh không phù hợp',
          userIntent: 'Kỳ vọng người dùng (ngôn ngữ tự nhiên):
{{intent}}',
          codeTitle: 'Mã:'
        }
      },
      messages: {
        generateCodeFirst: 'Vui lòng tạo mã chiến lược trước',
        validateCodeFirst: 'Vui lòng nhấn “Xác thực mã” trước',
        codeInvalidFixAndContinue: 'Xác thực mã thất bại. Hãy sửa trước khi tiếp tục',
        startBacktestFirst: 'Vui lòng bắt đầu backtest trước',
        backtestNotDoneWait: 'Backtest chưa xong. Hãy chờ đến khi trạng thái thành “Succeeded/Failed/Canceled”',
        confirmScoreFirst: 'Vui lòng xác nhận kết quả trong popup điểm số trước',
        fillRequiredWithFields: 'Vui lòng điền các trường bắt buộc: {{fields}}',
        fillRequired: 'Vui lòng điền các trường bắt buộc',
        watchBacktestRunFailed: 'watchBacktestRun thất bại',
        createDraftFailed: 'Không thể tạo bản nháp',
        loadAccountsFailed: 'Không thể tải tài khoản',
        loadSymbolsFailed: 'Không thể tải mã',
        loadDatasetFailed: 'Không thể tải dataset',
        datasetFrozenCreated: 'Đã tạo dataset đóng băng',
        freezeDatasetFailed: 'Không thể đóng băng dataset',
        inputIntentFirst: 'Vui lòng nhập mục tiêu/ý tưởng chiến lược trước',
        aiRequestTimeout: 'Hết thời gian yêu cầu AI (> {{seconds}}s)',
        modelReturnedEmpty: 'Mô hình trả về rỗng',
        noPythonCodeBlock: 'Agent code không xuất \`\`\`python code block\`\`\`. Vui lòng kiểm tra kết quả',
        agentFailed: '{{title}} thất bại',
        userAborted: 'Người dùng đã hủy',
        chatAborted: 'Đã hủy trò chuyện với mô hình',
        noCodeToValidate: 'Không có mã để xác thực',
        validateOk: 'Xác thực thành công',
        validateFailed: 'Xác thực thất bại',
        validateError: 'Lỗi xác thực',
        noCodeToBacktest: 'Không có mã để backtest',
        backtestCreated: 'Đã tạo backtest',
        createBacktestFailed: 'Không thể tạo backtest',
        draftNotCreated: 'Chưa tạo bản nháp',
        draftSaved: 'Đã lưu bản nháp',
        saveFailed: 'Lưu thất bại',
        publishedNoId: 'Đã triển khai nhưng không nhận được id (vui lòng kiểm tra trong quản lý chiến lược)',
        templatePublished: 'Đã triển khai template',
        publishFailed: 'Triển khai thất bại',
        publishTemplateFirst: 'Vui lòng triển khai template trước',
        scheduleCreatedAndEnabled: 'Đã tạo và bật lịch',
        scheduleCreated: 'Đã tạo lịch',
        createScheduleFailed: 'Không thể tạo lịch',
        scheduleAlreadyExists: 'Đã tồn tại lịch với cùng template+mã+khung thời gian cho tài khoản này. Vui lòng không tạo trùng.'
      }
    },
    settings: {
      pageTitle: 'Cài đặt trợ lý AI',
      defaultProfileName: 'Mặc định',
      primary: {
        title: 'Mô hình chính mặc định',
        hint: 'Dùng cho bước "Làm rõ ý định", sinh mã, panel "Trợ lý AI — sửa mã" trong trình soạn template, và bất kỳ Agent nào chưa chọn model riêng.',
        placeholder: 'Chọn một provider · model làm bộ não mặc định'
      },
      fields: {
        name: 'Tên',
        provider: 'Nhà cung cấp AI',
        baseUrl: 'Base URL',
        baseUrlHint: ' (địa chỉ dịch vụ model)',
        apiKey: 'API Key',
        apiKeyConfigured: 'Đã cấu hình',
        apiKeyReplaceHint: 'Để thay key, nhập lại tại đây',
        deleteApiKey: 'Xoá key',
        model: 'Mô hình',
        defaultModel: 'Model mặc định',
        availableModels: 'Model khả dụng',
        availableModelsHint: 'Có thể bật nhiều model dùng chung một API key. Danh sách này hiện trong dropdown của /ai/agents. Mặc định trống — chọn từ dropdown hoặc gõ model id rồi Enter để thêm; chỉ giữ những model bạn chọn rõ ràng.',
        availableModelsPlaceholder: 'Chọn từ dropdown hoặc gõ model id rồi Enter (mặc định trống)',
        availableModelsEmpty: 'Gõ model id rồi nhấn Enter để thêm',
        availableModelsTip: 'Lưu ý: xoá một model không tự huỷ các Agent đã liên kết model đó tại /ai/agents, nhưng nó sẽ biến mất khỏi gợi ý dropdown.',
        clear: 'Xoá hết',
        temperature: 'Nhiệt độ (Temperature)',
        timeoutSeconds: 'Thời gian chờ (giây)',
        maxTokens: 'Số token tối đa',
        enabledStatus: 'Trạng thái bật',
        enabledOn: 'Đang bật → nhấp để tắt',
        enabledOff: 'Đang tắt → nhấp để bật'
      },
      inferenceParams: {
        title: 'Tham số suy luận'
      },
      sections: {
        basic: 'Thông tin cơ bản',
        connection: 'Cấu hình kết nối',
        advanced: 'Tham số nâng cao',
        advancedHint: 'Chỉ chỉnh khi bạn hiểu rõ ý nghĩa; giá trị mặc định phù hợp đa số kịch bản',
        connectionApiKeyLink: 'Đến trang đăng ký / quản lý API key của nhà cung cấp'
      },
      providers: {
        enabledTitle: 'Nhà cung cấp đã bật',
        emptyTitle: 'Chưa có nhà cung cấp nào được bật',
        emptyHint: 'Cấu hình API key và model khả dụng tại ',
        emptyHintTail: ' trước.',
        modelsUnit: 'model',
        noModels: 'Chưa có model khả dụng',
        openai: 'OpenAI',
        anthropic: 'Anthropic Claude',
        deepseek: 'DeepSeek',
        zhipu: 'Zhipu AI',
        qwen: 'Qwen / DashScope',
        moonshot: 'Moonshot (Kimi)',
        doubao: 'Doubao',
        siliconflow: 'SiliconFlow',
        openrouter: 'OpenRouter',
        mistral: 'Mistral',
        groq: 'Groq',
        custom: 'Tùy chỉnh (tương thích OpenAI)',
        openai_compatible: 'Tùy chỉnh (tương thích OpenAI)'
      },
      placeholders: {
        name: 'VD: DeepSeek - chi phí thấp',
        provider: 'Chọn nhà cung cấp AI',
        baseUrl: 'VD: https://api.example.com/v1',
        apiKey: 'Nhập API key',
        providerFirst: 'Vui lòng chọn nhà cung cấp trước',
        modelManual: 'Nhập tên mô hình (khuyến nghị copy model id từ trang quản lý)',
        modelSelect: 'Chọn mô hình'
      },
      apiKeySavedAs: 'Đã lưu: {{masked}}',
      apiKeyGuide: {
        title: 'Hướng dẫn lấy API key',
        selectProviderHint: 'Chọn nhà cung cấp để xem hướng dẫn lấy API key.',
        modelSuggestionZhipu: 'Gợi ý: chọn \`glm-4-flash\` / \`glm-4\`',
        modelSuggestionDeepSeek: 'Gợi ý: chọn \`deepseek-chat\`',
        default: 'Nhà cung cấp: {{provider}}. Tạo API key trong trang quản lý và dán vào ô phía trên.',
        zhipu: {
          title: 'Lấy Zhipu API key',
          step1: 'Mở nền tảng Zhipu: ',
          step2: 'Đăng nhập/đăng ký, sau đó tạo và sao chép API key'
        },
        deepseek: {
          title: 'Lấy DeepSeek API key',
          step1: 'Mở nền tảng DeepSeek: ',
          step2: 'Đăng nhập/đăng ký, sau đó tạo và sao chép API key trong trang API Keys'
        }
      },
      actions: {
        validateApiKey: 'Xác minh API key',
        saveConfig: 'Lưu cấu hình'
      },
      profiles: {
        current: 'Hiện tại',
        actions: {
          setCurrent: 'Đặt hiện tại'
        },
        delete: {
          title: 'Xóa cấu hình',
          content: 'Xóa cấu hình này?'
        }
      },
      messages: {
        loadConfigFailed: 'Tải cấu hình AI thất bại',
        probeSuccess: 'Kết nối thành công',
        probeFailed: 'Kết nối thất bại',
        selectSavedProfileOrEnterKey: 'Vui lòng chọn cấu hình đã lưu hoặc nhập API key',
        validateSuccess: 'Xác minh thành công',
        validateFailed: 'Xác minh thất bại',
        apiKeyValidated: 'API key hợp lệ',
        validateBeforeSave: 'Vui lòng xác minh API key trước khi lưu',
        saveSuccess: 'Lưu thành công',
        deleted: 'Đã xóa',
        setCurrentSuccess: 'Đã chuyển cấu hình hiện tại',
        enabled: 'Đã bật',
        disabled: 'Đã tắt'
      },
      errors: {
        arrearage: 'Phản hồi từ nhà cung cấp: tài khoản nợ phí/thiếu số dư hoặc trạng thái bất thường. Vui lòng kiểm tra hóa đơn và trạng thái tài khoản.',
        invalidModelId: 'Phản hồi từ nhà cung cấp: mô hình không khả dụng{{model}}. Vui lòng chọn từ danh sách hoặc dùng đúng model id.',
        unauthorized: 'Phản hồi từ nhà cung cấp: không được ủy quyền (401). Vui lòng kiểm tra API key và quyền.',
        forbidden: 'Phản hồi từ nhà cung cấp: bị từ chối (403). Vui lòng kiểm tra quyền, IP allowlist hoặc trạng thái tài khoản.',
        timeout: 'Hết thời gian chờ. Vui lòng kiểm tra Base URL/mạng và thử lại.'
      },
      discoverErrors: {
        baseUrlRequired: 'Vui lòng nhập Base URL (địa chỉ dịch vụ model).',
        baseUrlInvalid: 'Base URL không hợp lệ: dùng URL đầy đủ, ví dụ https://model.example.com hoặc https://model.example.com/v1',
        freeTierExhausted: 'Đã hết miễn phí: tắt chế độ chỉ free tier trên console hoặc đổi sang key trả phí.',
        quotaOrRateLimit: 'Hết quota hoặc bị giới hạn tốc độ: nhà cung cấp từ chối. Kiểm tra thanh toán/giới hạn hoặc thử lại sau.',
        quotaForbidden403: 'Bị từ chối (quota): kiểm tra thanh toán/quota trên console.',
        unauthorized: 'Xác thực thất bại: kiểm tra API key/secret.',
        endpoint404: 'Không tìm thấy endpoint model: kiểm tra Base URL có khớp API tương thích OpenAI (một số dịch vụ cần /v1).',
        timeout: 'Hết thời gian chờ: kiểm tra mạng hoặc thử lại sau.',
        unreachable: 'Không kết nối được dịch vụ model: kiểm tra Base URL, mạng hoặc gateway.',
        invalidModelsResponse: 'Phản hồi không tương thích giao thức /models.',
        noModelsReturned: 'Không có model khả dụng: kiểm tra quyền tài khoản hoặc cấu hình.',
        providerRegionBlocked: 'Hạn chế khu vực: nhà cung cấp model từ chối theo vùng phát hiện được (IP egress có thể khác vị trí máy chủ). Hãy đổi vùng egress/proxy HTTP(S) hợp lệ hoặc dùng nhà cung cấp khác.',
        generic: 'Không tải được danh sách model. Kiểm tra Base URL và API key.',
        genericDetail: 'Không tải được danh sách model: {{detail}}'
      },
      validation: {
        nameRequired: 'Tên là bắt buộc',
        apiKeyRequired: 'API key là bắt buộc',
        baseUrlRequired: 'Base URL là bắt buộc',
        baseUrlProtocol: 'Base URL phải bắt đầu bằng http:// hoặc https://',
        baseUrlNoChatCompletionsSuffix: 'Base URL không nên kết thúc bằng /chat/completions',
        modelRequired: 'Mô hình là bắt buộc',
        modelFormat: 'Định dạng mô hình không hợp lệ'
      },
      agent: {
        title: 'Định nghĩa Agent',
        defaultName: 'Agent tùy chỉnh',
        removeConfirmTitle: 'Xoá Agent',
        removeConfirmContent: 'Bạn chắc chắn muốn xoá agent này?',
        actions: {
          add: 'Thêm',
          save: 'Lưu',
          remove: 'Xoá',
          loadDefaults: 'Tải 8 agent mặc định',
          restoreDefaults: 'Khôi phục mặc định',
          restoreDefaultsConfirmTitle: 'Khôi phục nhân dạng mặc định?',
          restoreDefaultsConfirmContent: 'Thao tác này sẽ đặt lại 8 agent hệ thống (style/signals/risk/macro/sentiment/portfolio/execution/code) về nhân dạng mặc định. Các agent tự thêm sẽ được giữ. Chỉ chỉnh sửa bản náp, phải bấm Lưu mới được lưu vào CSDL.'
        },
        messages: {
          selectProfileFirst: 'Vui lòng chọn một cấu hình ở bên trái trước',
          loading: 'Đang tải...',
          empty: 'Chưa có agent tuỳ chỉnh, bấm "Thêm" để bắt đầu',
          saveSuccess: 'Đã lưu agents',
          saveFailed: 'Lưu agents thất bại',
          defaultsLoaded: 'Đã tải mẫu agent mặc định. Bấm Lưu để lưu vào CSDL.'
        },
        fields: {
          namePlaceholder: 'Tên agent',
          identityPlaceholder: 'Nhân dạng / persona (ghép vào system prompt)',
          inputHintPlaceholder: 'Gợi ý nhập (tuỳ chọn)',
          modelProfilePlaceholder: 'Mặc định (dùng cấu hình hiện tại)',
          modelProfileEmpty: 'Vui lòng bật ít nhất một provider/model trong "Cài đặt AI" trước',
          historicalBinding: '{{value}} (lịch sử)'
        },
        types: {
          style: 'Phong cách',
          signals: 'Tín hiệu',
          risk: 'Kiểm soát rủi ro',
          macro: 'Vĩ mô',
          sentiment: 'Tâm lý',
          portfolio: 'Danh mục',
          execution: 'Thực thi',
          code: 'Mã'
        },
        defaults: {
          style: {
            identity: 'Bạn là chiến lược gia định lượng cấp cao, tập trung vào chọn mô hình giao dịch phù hợp. Dựa trên loại tài khoản, công cụ, khung thời gian và thống kê lịch sử cùng mục tiêu và ràng buộc của người dùng, đề xuất một mô hình chính và một mô hình thay thế (trend, mean-reversion, breakout, momentum, arbitrage, grid, event-driven). Giải thích điều kiện phù hợp và không phù hợp, kèm ít nhất ba cảnh báo rủi ro.',
            inputHint: 'Ví dụ: tài khoản = EURUSD cá nhân; khung thời gian = H1; mục tiêu = lợi nhuận 3%/tháng, drawdown tối đa <10%; ưu tiên = tỷ lệ thắng hơn tỷ lệ lời/lỗ.'
          },
          signals: {
            identity: 'Bạn là kỹ sư yếu tố và tín hiệu, sử dụng MA/EMA, RSI, MACD, ADX, ATR, Bollinger Bands, VWAP, pivot, khối lượng và biến động. Không dùng dữ liệu bên ngoài, thiết kế các quy tắc vào/ra/lọc có thể tái tạo và tham số hóa, kèm lập luận và ít nhất ba kịch bản thất bại.',
            inputHint: 'Ví dụ: mô hình = trend-following; khung thời gian = H1; chỉ báo = EMA/ATR/ADX; fast = 20, slow = 60.'
          },
          risk: {
            identity: 'Bạn là chuyên gia rủi ro, thiết kế định cỡ vị thế, cắt lỗ, giới hạn rủi ro, drawdown tối đa, quy tắc tạm dừng, giới hạn tần suất giao dịch và bảo vệ bất thường. Đầu ra gồm các ràng buộc cứng với tham số đề xuất và hành động kích hoạt, kèm các chế độ thất bại phổ biến.',
            inputHint: 'Ví dụ: vốn = 10.000; giới hạn drawdown tháng = 5%; rủi ro mỗi giao dịch = 0,5%; giao dịch trong ngày <= 5; cắt lỗ = 1,5×ATR.'
          },
          macro: {
            identity: 'Bạn là nhà nghiên cứu vĩ mô, tập trung vào quyết định ngân hàng trung ương, CPI/PPI, NFP, PMI, GDP và các sự kiện quan trọng. Dùng lịch sự kiện, xác định cửa sổ sự kiện và đề xuất vị thế (tránh/giảm/theo sự kiện) cho công cụ mục tiêu.',
            inputHint: 'Ví dụ: sự kiện chính = CPI Mỹ và biên bản FOMC; mã mục tiêu = XAUUSD.'
          },
          sentiment: {
            identity: 'Bạn là nhà phân tích tâm lý và dòng vốn, sử dụng COT, VIX, funding, dòng ETF và tin tức/tâm lý mạng xã hội. Đầu ra là điểm tâm lý từ -1 đến 1 với động lực và thay đổi, cùng cách điều chỉnh hoặc ngược dòng.',
            inputHint: 'Ví dụ: VIX từ 14 lên 22; vị thế long ròng phi thương mại -18%; tin tức chủ đạo về suy thoái / cắt giảm lãi suất.'
          },
          portfolio: {
            identity: 'Bạn là nhà quản lý danh mục, phân bổ vốn giữa các chiến lược và công cụ bằng cách dùng tương quan, co rút hiệp phương sai, risk parity, vol-targeting và đa dạng hóa. Cung cấp tỷ trọng, đóng góp rủi ro và quy tắc tái cân bằng.',
            inputHint: 'Ví dụ: chiến lược = trend-EURUSD và mean-reversion-XAUUSD; vốn = 50.000; vol mục tiêu = 12% năm.'
          },
          execution: {
            identity: 'Bạn là chuyên gia thực thi, chọn phong cách thực thi, phiên giao dịch và chia lệnh, ước tính tác động và trượt giá, xác định hành vi hạ cấp khi thanh khoản kém.',
            inputHint: 'Ví dụ: mua 10 lot EURUSD; spread = 0,6 pip; mục tiêu 5 phút; trượt giá tối đa = 0,8 pip.'
          },
          code: {
            identity: 'Bạn là kỹ sư Python AntTrader, tạo mã chiến lược an toàn sandbox với run(context) trả về signal, symbol, confidence, risk_level và reason từ context["params"], đầu ra là một khối \`\`\`python\`\`\` duy nhất không có Markdown thêm.',
            inputHint: 'Ví dụ: trend-following EMA(fast)/EMA(slow) với bộ lọc ATR; params = fast, slow, atr_period, risk_per_trade.'
          }
        }
      }
    },
    consensus: {
      title: 'Consensus & Discussion',
      actions: {
        refresh: 'Refresh'
      },
      fields: {
        account: 'Account',
        symbol: 'Symbol',
        timeframe: 'Timeframe'
      },
      panel: {
        title: 'Objective Score',
        decision: 'Decision',
        overallScore: 'Overall',
        technicalScore: 'Technical'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: 'Signal'
        },
        macd: {
          value: 'MACD',
          signalLine: 'Signal Line',
          hist: 'Histogram',
          flag: 'Signal',
          trend: 'Pattern'
        },
        ma: {
          trend: 'MA Trend'
        }
      }
    },
    gate: {
      title: 'AI Gate Progress',
      pipelineDesc: '6-stage Gate pipeline: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation',
      labels: {
        compliance: 'Compliance',
        lookahead: 'Look-Ahead Bias',
        walkforward: 'Walk-Forward',
        deflated_sharpe: 'Deflated Sharpe',
        paper: 'Paper Trading',
        correlation: 'Correlation'
      },
      descriptions: {
        compliance: 'DSL expression non-empty validation',
        lookahead: 'Future function reference scan (close[t+N], ref negative offset)',
        walkforward: 'Purged Walk-Forward cross-validation',
        deflated_sharpe: 'Lopez de Prado Deflated Sharpe Ratio',
        paper: '≥14 days paper trading validation',
        correlation: 'Signal correlation check with existing strategies'
      },
      status: {
        evaluating: 'Evaluating...'
      },
      strategyParams: 'Strategy Parameters',
      dslExpression: 'DSL Expression',
      dailyReturns: 'Daily Returns (comma or newline separated)',
      numAttempts: 'Strategy Attempts',
      paperMetrics: 'Paper Trading Metrics',
      paperDays: 'Paper Days',
      paperNetPnL: 'Paper Net P&L',
      paperNetReturn: 'Paper Net Return',
      paperTradeCount: 'Paper Trade Count',
      backtestNetReturn: 'Backtest Net Return',
      backtestGrossReturn: 'Backtest Gross Return',
      runPipeline: 'Run Gate Pipeline',
      retry: 'Retry',
      gateProgress: 'Gate Evaluation Progress',
      pipelineResult: 'Pipeline Result',
      allPassed: 'All 6 gates passed — strategy eligible for PromoteToLive evaluation',
      failed: 'Failed: {{gate}}',
      details: 'Details'
    }
  }
} as const;

export default ai;
