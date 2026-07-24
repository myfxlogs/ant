// Auto-generated from proto/ant/v1/i18n/ai_core_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiCore = {
  "ai": {
    "consensus": {
      "signals": {
        "ma": {
          "trend": "均线趋势"
        },
        "macd": {
          "flag": "Tín hiệu",
          "hist": "Biểu đồ cột",
          "signalLine": "Đường tín hiệu",
          "trend": "形态",
          "value": "MACD"
        },
        "rsi": {
          "flag": "Tín hiệu",
          "value": "RSI"
        }
      },
      "actions": {
        "refresh": "刷新"
      },
      "fields": {
        "account": "Tài khoản",
        "symbol": "Mã chứng khoán",
        "timeframe": "周期"
      },
      "panel": {
        "decision": "Quyết định",
        "overallScore": "Tổng thể",
        "technicalScore": "技术面",
        "title": "Điểm mục tiêu"
      },
      "title": "Đồng thuận & Thảo luận"
    },
    "agentPrompts": {
      "code": {
        "title": "Sinh mã"
      },
      "risk": {
        "title": "Rủi ro & ràng buộc thực thi"
      },
      "signals": {
        "title": "Tín hiệu & chỉ báo"
      },
      "style": {
        "title": "Trạng thái thị trường / phong cách"
      }
    },
    "assistant": {
      "messages": {
        "noCodeBlockFound": "Không tìm thấy khối mã (\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`...\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`)"
      }
    },
    "backtestScoreCard": {
      "backendRiskScore": {
        "empty": "无（先保存模板，回测完成后自动计算）",
        "loading": "Đang tính...",
        "reasons": "Lý do",
        "reliable": "Đáng tin cậy",
        "title": "Điểm rủi ro chiến lược",
        "unknown": "không xác định",
        "unreliable": "Không đáng tin cậy",
        "warnings": "Cảnh báo"
      },
      "chart": {
        "title": "净值曲线"
      },
      "level": {
        "excellent": "Xuất sắc",
        "fair": "Trung bình",
        "good": "Tốt",
        "poor": "差"
      },
      "metrics": {
        "annualReturn": "Lợi nhuận năm",
        "equityPoints": "净值点数",
        "maxDrawdown": "Sụt giảm tối đa",
        "sharpe": "夏普",
        "totalReturn": "Tổng lợi nhuận",
        "totalTrades": "Số lệnh",
        "winRate": "Tỷ lệ thắng"
      },
      "recommendation": {
        "cautious": "Triển khai thận trọng: bắt đầu nhỏ / theo dõi thủ công một thời gian.",
        "loading": "Đang đánh giá rủi ro. Vui lòng chờ xong trước khi triển khai.",
        "notRecommended": "Không khuyến nghị giao dịch trực tiếp: rủi ro cao hoặc không đáng tin, tối ưu trước khi thử.",
        "recommended": "Khuyến nghị triển khai: rủi ro trong mức kiểm soát, chỉ số nhìn chung tốt."
      },
      "score": {
        "empty": "Chưa có điểm (đợi backtest hoàn tất hoặc thiếu metrics)",
        "title": "综合评分（启发式）"
      },
      "status": {
        "cancelRequested": "Đang hủy",
        "canceled": "已取消",
        "failed": "Thất bại",
        "pending": "Đang chờ",
        "running": "Đang chạy",
        "succeeded": "Thành công"
      },
      "stateLabel": "Trạng thái",
      "title": "Thẻ điểm backtest"
    },
    "client": {
      "errors": {
        "contentBlocked": "服务商安全过滤器阻止了响应。请改写提示词后重试。",
        "contextTooLong": "请求超出模型上下文窗口。请缩短对话/输入，或选择上下文更大的模型。",
        "edgeGatewayTimeout": "边缘网关超时 (通常为 Cloudflare HTTP 524)：浏览器未收到应用响应，长时运行操作常见。请重试；如持续出现，请联系运维提高代理/源站超时。",
        "forbidden": "服务商拒绝了请求 (403)。请检查 Key 权限、IP 白名单和账户状态。",
        "gatewayForbidden403": "网关被禁止 (403)。",
        "gatewayRateLimited429": "网关速率受限 (429)。",
        "gatewayTimeoutOrUnreachable": "网关超时或不可达。",
        "gatewayUnauthorized401": "网关未授权 (401)。",
        "insufficientBalance": "服务商报告余额为空/欠费。请在服务商控制台充值后重试。",
        "invalidModelId": "模型不可用{{model}} — 可能输入错误、已弃用或超出您的套餐。请从下拉列表中选择其他模型，或从服务商控制台复制标准 ID。",
        "networkUnreachable": "网关超时或不可达。请检查 Base URL、网络连接，或稍后重试。",
        "providerInternalError": "服务商返回服务器错误 (5xx)。请稍候或切换到其他服务商。",
        "rateLimited": "服务商正在限制您的请求频率，请稍候再试。",
        "regionNotSupported": "所选服务商在您所在地区/国家不可用，请切换到其他服务商。",
        "requestFailed": "请求失败，请重试。",
        "unauthorized": "服务商拒绝了 API Key (401)。请检查 Key 值及其是否有权访问所选模型。"
      }
    },
    "gate": {
      "descriptions": {
        "compliance": "Xác thực biểu thức DSL không rỗng",
        "correlation": "与现有策略的信号相关性检查",
        "deflated_sharpe": "Tỷ lệ Sharpe đã giảm phát của Lopez de Prado",
        "lookahead": "Quét tham chiếu hàm tương lai (close[t+N], ref offset âm)",
        "paper": "Xác thực giao dịch giấy >=14 ngày",
        "walkforward": "Xác thực chéo Walk-Forward đã thanh lọc"
      },
      "labels": {
        "compliance": "Tuân thủ",
        "correlation": "相关性",
        "deflated_sharpe": "通缩夏普比率",
        "lookahead": "Độ chệch nhìn về phía trước",
        "paper": "Giao dịch giấy",
        "walkforward": "前向分析"
      },
      "status": {
        "evaluating": "Đang đánh giá..."
      },
      "allPassed": "Cả 6 cổng đều vượt qua — chiến lược đủ điều kiện đánh giá PromoteToLive",
      "backtestGrossReturn": "Lợi nhuận gộp backtest",
      "backtestNetReturn": "Lợi nhuận ròng backtest",
      "dailyReturns": "Lợi nhuận hàng ngày (phân cách bằng dấu phẩy hoặc xuống dòng)",
      "details": "Chi tiết",
      "dslExpression": "Biểu thức DSL",
      "evaluating": "Đang đánh giá...",
      "fail": "失败",
      "failed": "Thất bại: {{gate}}",
      "gateProgress": "Tiến trình đánh giá Cổng",
      "noData": "không có dữ liệu",
      "numAttempts": "Số lần thử chiến lược",
      "paperDays": "Ngày giao dịch giấy",
      "paperMetrics": "Chỉ số giao dịch giấy",
      "paperNetPnL": "Lãi/Lỗ ròng giấy",
      "paperNetReturn": "Lợi nhuận ròng giấy",
      "paperTradeCount": "Số giao dịch giấy",
      "pass": "通过",
      "pipelineDesc": "Quy trình 6 cổng: Tuân thủ -> Nhìn về phía trước -> Walk-Forward -> Deflated Sharpe -> Giấy -> Tương quan",
      "pipelineResult": "Kết quả quy trình",
      "retry": "Thử lại",
      "runHint": "请先运行回测，然后点击\"运行质量门\"评估策略质量。",
      "runPipeline": "Chạy quy trình Cổng",
      "selectRun": "Chọn lần chạy kiểm thử lùi...",
      "skipped": "已跳过",
      "strategyParams": "Tham số chiến lược",
      "title": "Tiến trình Cổng AI",
      "unknown": "không xác định"
    },
    "reports": {
      "tradeAnalysis": {
        "riskAssessmentPrefix": "风险评估:",
        "title": "Báo cáo phân tích giao dịch AI"
      }
    },
    "requireConfig": {
      "actions": {
        "goSettings": "前往设置"
      },
      "description": "Vui lòng cấu hình nhà cung cấp, mô hình và API key trong Cài đặt trước khi dùng trình hướng dẫn hoặc trò chuyện.",
      "title": "Chưa cấu hình AI"
    },
    "signalCard": {
      "actions": {
        "cancel": "Hủy",
        "confirm": "Xác nhận",
        "executeTrade": "执行交易"
      },
      "confirmCancel": {
        "title": "确定要取消此信号？"
      },
      "confirmExecute": {
        "description": "将立即下单",
        "title": "Thực thi tín hiệu giao dịch này?"
      },
      "labels": {
        "analysisReason": "分析理由",
        "confidence": "Độ tin cậy",
        "price": "Giá",
        "stopLoss": "Cắt lỗ",
        "takeProfit": "Chốt lời",
        "volume": "Khối lượng"
      },
      "status": {
        "cancelled": "已取消",
        "confirmed": "Đã xác nhận",
        "executed": "Đã thực thi",
        "pending": "Chờ xác nhận"
      }
    },
    "strategyCard": {
      "actionType": {
        "alert": "警报",
        "buy": "Mua",
        "closeLong": "Đóng long",
        "closeShort": "Đóng short",
        "sell": "Bán"
      },
      "actions": {
        "start": "Bắt đầu",
        "stop": "停止"
      },
      "confirmDelete": {
        "description": "删除后无法恢复",
        "title": "Xóa chiến lược này?"
      },
      "labels": {
        "lastTriggeredAt": "最近触发: {{time}}",
        "triggeredCount": "Kích hoạt {{count}} lần"
      },
      "sections": {
        "actions": "操作",
        "conditions": "Điều kiện kích hoạt"
      },
      "status": {
        "active": "Đang chạy",
        "inactive": "Đã dừng",
        "paused": "已暂停"
      },
      "tooltips": {
        "createdAt": "Thời gian tạo",
        "lastTriggeredAt": "最近触发"
      }
    },
    "systemAI": {
      "cardState": {
        "enabled": "Đã bật",
        "noKey": "Chưa cấu hình",
        "noModel": "Chọn model",
        "readyDisabled": "就绪 · 已禁用"
      },
      "cardTags": {
        "current": "Hiện tại",
        "enabledButUnavailable": "已启用但不可用",
        "hasKey": "Đã có key",
        "noKey": "Chưa có key",
        "noModels": "Chưa cấu hình model khả dụng"
      },
      "customProvider": {
        "deleted": "Đã xóa nhà cung cấp tùy chỉnh",
        "fillNameFirst": "Vui lòng điền tên trước",
        "nameHint": "用于识别此提供商的唯一名称",
        "nameLabel": "Tên Nhà Cung Cấp",
        "namePlaceholder": "Nhà Cung Cấp Của Tôi",
        "nameRequired": "服务商名称不能为空"
      },
      "fields": {
        "apiKeyHint": "Sau khi nhập sẽ được mã hoá và lưu tự động, không cần submit thủ công",
        "apiKeyPastePlaceholder": "Dán API key — sẽ được lưu trước tự động",
        "autoFetching": "Đang tự lấy danh sách…",
        "baseUrlCustomHint": "Nhập endpoint tương thích OpenAI, ví dụ https://model.example.com/v1",
        "baseUrlCustomPlaceholder": "Ví dụ: https://model.example.com/v1",
        "baseUrlReadonlyHint": "Địa chỉ chính thức do hệ thống quản lý, không sửa được",
        "baseUrlReadonlyPlaceholder": "Địa chỉ chính thức (chỉ đọc)",
        "enabledHint": "Tắt thì nhà cung cấp này sẽ không được hệ thống sử dụng",
        "httpWarning": "Đang dùng HTTP — môi trường sản xuất nên dùng HTTPS",
        "maxTokensHint": "Token tối đa cho mỗi phản hồi",
        "primaryFor": "Mục đích chính (Primary For)",
        "primaryForHint": "用于内部分发：对话/嵌入/摘要/推理",
        "temperatureHint": "Cao hơn = đa dạng hơn, thấp hơn = ổn định hơn",
        "timeoutHint": "Thời gian chờ tối đa cho mỗi request"
      },
      "messages": {
        "autoDiscoveredModels": "Đã tự động khám phá {{count}} mô hình (chỉ để gợi ý)",
        "autoValidatedModels": "Đã tự động xác thực: tìm thấy {{count}} mô hình",
        "configSaveFailed": "Lưu cấu hình thất bại",
        "configSaved": "Đã lưu cấu hình",
        "deleteSecretFailed": "Xóa khóa bí mật thất bại",
        "loadConfigFailed": "Tải cấu hình thất bại",
        "secretAutoSaveFailed": "Tự động lưu khóa bí mật thất bại",
        "secretDeletedConfigReset": "Đã xóa khóa bí mật, cấu hình nhà cung cấp đặt lại về mặc định",
        "secretSavedAutoDiscover": "Đã lưu khóa bí mật, đang tự động khám phá mô hình...",
        "toggleEnabledFailed": "Thay đổi trạng thái bật/tắt thất bại",
        "validationFailedNeedApiKey": "Xác thực thất bại: nhà cung cấp này thường yêu cầu API Key. Vui lòng điền và lưu key trước, rồi thử lại.",
        "validationPassedModels": "Xác thực thành công: tìm thấy {{count}} mô hình"
      },
      "section1": {
        "subtitle": "Thẻ hiển thị cấu hình và trạng thái sẵn sàng của từng nhà cung cấp; nhấn để chọn",
        "title": "Chọn nhà cung cấp model"
      },
      "status": {
        "checkUrl": "Kiểm tra Base URL",
        "checkUrlDesc": "Đã có API key nhưng URL có vẻ không hợp lệ",
        "configReady": "Đã sẵn sàng cấu hình",
        "configReadyDesc": "Thêm model khả dụng để hệ thống tự kiểm tra kết nối",
        "connectionFailed": "连接错误，请检查上方提示",
        "error": "Có lỗi",
        "needKey": "Cần cấu hình API key",
        "needKeyDesc": "Nhập API key, hệ thống sẽ tự phát hiện danh sách model",
        "noProvider": "Chưa chọn nhà cung cấp",
        "noProviderDesc": "Chọn một nhà cung cấp trong các thẻ bên dưới để bắt đầu",
        "notEnabled": "Đã kết nối nhưng chưa bật",
        "notEnabledDesc": "Bật công tắc Enable để đưa vào sử dụng",
        "ready": "Sẵn sàng",
        "readyDesc": "đã bật và kết nối bình thường"
      },
      "statusBar": {
        "checking": "Đang kiểm tra kết nối…",
        "connected": "已连接",
        "disabled": "Chưa bật",
        "enabled": "Đã bật",
        "keyReady": "Key sẵn sàng"
      },
      "taglines": {
        "anthropic": "Dòng Claude",
        "deepseek": "DeepSeek · Hiệu quả chi phí",
        "moonshot": "Kimi · Ngữ cảnh dài",
        "openai": "Dòng GPT · Chính thức",
        "openai_compatible": "任意兼容端点",
        "qwen": "Alibaba Cloud · Tối ưu tiếng Trung",
        "zhipu": "Hệ Thanh Hoa · Đa dụng"
      },
      "emptyConfigs": "Chưa có cấu hình AI Provider (hệ thống sẽ tạo provider mặc định khi khởi động).",
      "pageSubtitle": "Cấu hình bộ não AI — chọn nhà cung cấp, quản lý API key và model khả dụng, và chỉ định \"Mô hình chính mặc định\" dùng cho toàn hệ thống.",
      "pageTitle": "Cài đặt trợ lý AI"
    },
    "workflowRuns": {
      "hints": {
        "selectToViewDetail": "从左侧选择运行记录查看详情"
      },
      "messages": {
        "loadDetailFailed": "加载详情失败",
        "loadListFailed": "Tải danh sách lần chạy thất bại"
      },
      "defaultTitle": "Quy trình AI",
      "title": "Quy trình AI"
    },
    "chatBox": {
      "collapse": "收起",
      "emptyDescription": "Bắt đầu trò chuyện với trợ lý AI",
      "expandAll": "Mở rộng tất cả",
      "thinking": "Đang suy nghĩ...",
      "truncated": "Nội dung quá dài và đã bị cắt bớt"
    },
    "conversation": {
      "defaultTitle": "新对话"
    },
    "gateway": {
      "balance": "Số dư ví",
      "modelPlaceholder": "Chọn mô hình AI",
      "monthlyCost": "Chi phí tháng này",
      "monthlyTokens": "Token tháng này",
      "noModels": "Không có mô hình nào",
      "selectModel": "Chọn mô hình",
      "title": "AI 网关",
      "usageByFeature": "Sử dụng theo tính năng",
      "useGateway": "AI Gateway",
      "useGatewayDesc": "Trừ ví · Tính theo token",
      "useOwnKey": "API Key của tôi",
      "useOwnKeyDesc": "Thanh toán trực tiếp · Tự quản lý",
      "useOwnKeyHint": "Sử dụng API Key của bạn để thanh toán trực tiếp cho nhà cung cấp. Chọn thẻ nhà cung cấp bên dưới để cấu hình.",
      "groupMyKeys": "Khóa API của tôi",
      "groupGateway": "AI Gateway",
      "groupCurrent": "Đang chọn"
    },
    "riskEval": {
      "failed": "风险评估失败"
    },
    "tabs": {
      "agentSettings": "Thiết lập chuyên gia",
      "gate": "AI 质量门",
      "settings": "Cài đặt"
    }
  }
} as const;
export default AiCore;
