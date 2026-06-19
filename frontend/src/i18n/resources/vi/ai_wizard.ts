// Auto-generated from proto/ant/v1/i18n/ai_wizard_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiWizard = {
  "ai": {
    "wizard": {
      "actions": {
        "cancel": "Hủy",
        "next": "Tiếp",
        "prev": "Trước"
      },
      "agents": {
        "codeTitle": "Sinh mã",
        "riskTitle": "Rủi ro & ràng buộc thực thi",
        "signalsTitle": "Tín hiệu & chỉ báo",
        "styleTitle": "Trạng thái thị trường / phong cách"
      },
      "currentModel": "Mô hình hiện tại: {{model}}",
      "generate": {
        "actions": {
          "abort": "Hủy",
          "goValidate": "Đi xác thực",
          "hide": "Ẩn",
          "regenerateSummary": "Tạo lại tóm tắt",
          "rerun": "Chạy lại",
          "runAgents": "Phân tích chuyên gia + sinh mã"
        },
        "cards": {
          "resultsTitle": "Multiple experts\\\\\\\\\\\\\\\\"
        },
        "hints": {
          "afterGenerated": "Sau khi tạo xong, sang bước tiếp theo để xác thực/backtest/triển khai."
        },
        "labels": {
          "elapsed": "Thời gian"
        },
        "modals": {
          "final": {
            "title": "Đã sinh mã. Khuyến nghị nhấn “Xác thực mã” để xác nhận."
          }
        },
        "sections": {
          "output": "Kết quả mô hình",
          "prompt": "Prompt gửi tới mô hình",
          "spec": "Đặc tả"
        },
        "status": {
          "done": "Hoàn tất",
          "error": "Thất bại",
          "idle": "Đang chờ",
          "inProgress": "Đang chạy",
          "running": {
            "code": "Đang sinh mã",
            "generic": "{{title}} đang chạy",
            "risk": "Đang thiết kế rủi ro/ràng buộc thực thi",
            "signals": "Đang thiết kế tín hiệu/chỉ báo",
            "style": "Đang phân tích trạng thái/phong cách thị trường"
          }
        }
      },
      "messages": {
        "agentFailed": "{{title}} thất bại",
        "aiRequestTimeout": "Hết thời gian yêu cầu AI (> {{seconds}}s)",
        "backtestCreated": "Đã tạo backtest",
        "backtestNotDoneWait": "Backtest chưa xong. Hãy chờ đến khi trạng thái thành “Succeeded/Failed/Canceled”",
        "chatAborted": "Đã hủy trò chuyện với mô hình",
        "codeInvalidFixAndContinue": "Xác thực mã thất bại. Hãy sửa trước khi tiếp tục",
        "confirmScoreFirst": "Vui lòng xác nhận kết quả trong popup điểm số trước",
        "createBacktestFailed": "Không thể tạo backtest",
        "createDraftFailed": "Không thể tạo bản nháp",
        "createScheduleFailed": "Không thể tạo lịch",
        "datasetFrozenCreated": "Đã tạo dataset đóng băng",
        "draftNotCreated": "Chưa tạo bản nháp",
        "draftSaved": "Đã lưu bản nháp",
        "fillRequired": "Vui lòng điền các trường bắt buộc",
        "fillRequiredWithFields": "Vui lòng điền các trường bắt buộc: {{fields}}",
        "freezeDatasetFailed": "Không thể đóng băng dataset",
        "generateCodeFirst": "Vui lòng tạo mã chiến lược trước",
        "inputIntentFirst": "Vui lòng nhập mục tiêu/ý tưởng chiến lược trước",
        "loadAccountsFailed": "Không thể tải tài khoản",
        "loadDatasetFailed": "Không thể tải dataset",
        "loadSymbolsFailed": "Không thể tải mã",
        "modelReturnedEmpty": "Mô hình trả về rỗng",
        "noCodeToBacktest": "Không có mã để backtest",
        "noCodeToValidate": "Không có mã để xác thực",
        "noPythonCodeBlock": "代码 Agent 未输出 ```python 代码块```，请在结果中检查",
        "publishFailed": "Triển khai thất bại",
        "publishTemplateFirst": "Vui lòng triển khai template trước",
        "publishedNoId": "Đã triển khai nhưng không nhận được id (vui lòng kiểm tra trong quản lý chiến lược)",
        "saveFailed": "Lưu thất bại",
        "scheduleAlreadyExists": "Đã tồn tại lịch với cùng template+mã+khung thời gian cho tài khoản này. Vui lòng không tạo trùng.",
        "scheduleCreated": "Đã tạo lịch",
        "scheduleCreatedAndEnabled": "Đã tạo và bật lịch",
        "startBacktestFirst": "Vui lòng bắt đầu backtest trước",
        "templatePublished": "Đã triển khai template",
        "userAborted": "Người dùng đã hủy",
        "validateCodeFirst": "Vui lòng nhấn “Xác thực mã” trước",
        "validateError": "Lỗi xác thực",
        "validateFailed": "Xác thực thất bại",
        "validateOk": "Xác thực thành công",
        "watchBacktestRunFailed": "watchBacktestRun thất bại"
      },
      "prompts": {
        "base": {
          "account": "Tài khoản: {{accountId}}",
          "constraints": "Ràng buộc: max drawdown={{maxDrawdownPct}}% rủi ro/lệnh={{riskPerTradePct}}% tối đa lệnh/ngày={{maxTradesPerDay}}",
          "data": "Dữ liệu: {{dataSpec}}",
          "empty": "(trống)",
          "macroDisabled": "Sự kiện vĩ mô: không dùng",
          "macroEnabled": "Macro events (user-provided):\\\\\\\\\\\\\\\\n{{text}}",
          "params": "Parameters (defs+current values; injected into context[\"params\"] at runtime):\\\\\\\\\\\\\\\\n{{params}}",
          "symbol": "Mã: {{symbol}}",
          "timeframe": "Khung thời gian: {{timeframe}}",
          "userIntent": "User strategy goal (natural language):\\\\\\\\\\\\\\\\n{{intent}}"
        },
        "dataSpec": {
          "dataset": "Sử dụng dataset đã đóng băng datasetId={{datasetId}}",
          "klineRange": "Sử dụng phạm vi nến lịch sử from={{from}} to={{to}}"
        },
        "summary": {
          "codeTitle": "Mã:",
          "intro": "Bạn là trợ lý giải thích chiến lược định lượng. Hãy giải thích ý tưởng cốt lõi của đoạn mã chiến lược AntTrader Python dưới đây bằng các gạch đầu dòng ngắn gọn (tối đa 12 dòng) để giúp người dùng đánh giá có đúng kỳ vọng hay không.",
          "mustInclude1": "1) Loại/kiểu chiến lược (trend/mean-reversion/breakout/momentum/grid... nếu không chắc hãy ghi “Không rõ”)",
          "mustInclude2": "2) Điều kiện vào lệnh chính (2-4 ý)",
          "mustInclude3": "3) Điều kiện thoát/SL/TP/ràng buộc rủi ro chính (2-4 ý)",
          "mustInclude4": "4) 1 bối cảnh phù hợp và 1 bối cảnh không phù hợp",
          "mustIncludeTitle": "Bắt buộc gồm:",
          "userIntent": "User expectation (natural language):\\\\\\\\\\\\\\\\n{{intent}}"
        },
        "upstream": {
          "risk": "【Risk control conclusion】\\\\\\\\\\\\\\\\n{{text}}",
          "sectionTitle": "[Kết luận agent phía trên (nguyên văn)]",
          "signals": "【Signal design conclusion】\\\\\\\\\\\\\\\\n{{text}}",
          "style": "【Market condition/style conclusion】\\\\\\\\\\\\\\\\n{{text}}"
        }
      },
      "publish": {
        "actions": {
          "publishTemplate": "Triển khai template",
          "startBacktest": "Backtest (tác vụ bất đồng bộ)",
          "validateCode": "Xác thực mã"
        },
        "cards": {
          "codeTitle": "1) Mã chiến lược (có thể chỉnh sửa)",
          "launchTitle": "3) Triển khai lịch chạy",
          "scoreCardTitle": "2) Thẻ điểm backtest"
        },
        "messages": {
          "validateFailed": "validate thất bại",
          "validateOk": "validate thành công"
        },
        "placeholders": {
          "codeEditable": "Mã do AI tạo sẽ xuất hiện ở đây. Bạn cũng có thể chỉnh sửa thủ công."
        }
      },
      "publishBacktest": {
        "actions": {
          "close": "Đóng",
          "confirm": "Xác nhận",
          "inProgress": "Đang chạy",
          "retry": "Thử lại",
          "runInBackground": "Chạy nền",
          "startBacktest": "Bắt đầu backtest",
          "succeeded": "Thành công"
        },
        "cards": {
          "backtestTitle": "回测",
          "scoreCardTitle": "Thẻ điểm"
        },
        "draftName": "Kiểm thử lùi {{datetime}} {{symbol}} {{timeframe}}",
        "draftNameShort": "Kiểm thử lùi {{symbol}} {{timeframe}}",
        "labels": {
          "confirmed": "Đã xác nhận",
          "elapsed": "Thời gian",
          "overallScore": "Điểm tổng",
          "scoringProgress": "Tiến độ chấm điểm",
          "status": "Trạng thái"
        },
        "modals": {
          "score": {
            "title": "Xác nhận điểm số"
          },
          "status": {
            "title": "Backtest đang chạy"
          }
        }
      },
      "schedule": {
        "defaultName": "Lịch AI {{symbol}} {{timeframe}}"
      },
      "setup": {
        "actions": {
          "deleteCurrentDataset": "Xóa dataset hiện tại",
          "freezeFromCurrentRange": "Đóng băng từ phạm vi hiện tại",
          "refreshDataset": "Làm mới"
        },
        "cards": {
          "constraintsAndGoalTitle": "Ràng buộc & mục tiêu",
          "hardConstraintsTitle": "Ràng buộc cứng",
          "hintsTitle": "Gợi ý",
          "tradeAndDataTitle": "Giao dịch & dữ liệu"
        },
        "dataModes": {
          "dataset": "Dataset đóng băng",
          "klineRange": "Phạm vi nến"
        },
        "hints": {
          "nextWillGenerateCode": "Bước tiếp theo sẽ tạo mã chiến lược.",
          "tradeDataNextStep": "Sau khi điền xong, nhấn “Tiếp” để tiếp tục thiết lập ràng buộc & mục tiêu."
        },
        "labels": {
          "account": "Tài khoản",
          "backtestRange": "Phạm vi backtest",
          "dataset": "Dataset đóng băng",
          "historicalData": "Dữ liệu lịch sử",
          "intent": "Ý định chiến lược",
          "macroEvents": "Sự kiện vĩ mô",
          "macroModule": "Mô-đun vĩ mô",
          "maxDrawdownPct": "Sụt giảm tối đa (%)",
          "maxTradesPerDay": "Số lệnh tối đa mỗi ngày",
          "riskPerTradePct": "Rủi ro mỗi lệnh (%)",
          "symbol": "Mã",
          "timeframe": "Khung thời gian"
        },
        "macro": {
          "off": "Tắt",
          "on": "开"
        },
        "messages": {
          "datasetDeleted": "Đã xóa dataset"
        },
        "modals": {
          "deleteDataset": {
            "content": "Xóa dataset đóng băng đang chọn?",
            "ok": "Xóa",
            "title": "Xóa dataset"
          }
        },
        "placeholders": {
          "intentExample": "Ví dụ: Theo xu hướng khi phá vỡ; tránh biến động cao; ưu tiên tỷ lệ thắng...",
          "macroExample": "Example:\\\\\\\\\\\\\\\\n2024-01-03 21:15 FOMC minutes\\\\\\\\\\\\\\\\n2024-01-05 20:30 NFP",
          "selectAccount": "Chọn tài khoản",
          "selectFrozenDataset": "Chọn dataset đóng băng",
          "selectSymbol": "Chọn mã",
          "selectTimeframe": "Chọn khung thời gian"
        },
        "validations": {
          "enterIntent": "Vui lòng nhập ý định chiến lược",
          "selectAccount": "Vui lòng chọn tài khoản",
          "selectDataset": "Vui lòng chọn dataset",
          "selectSymbol": "Vui lòng chọn mã",
          "selectTimeframe": "Vui lòng chọn khung thời gian"
        }
      },
      "steps": {
        "generate": "Tạo chiến lược",
        "publishBacktest": "Triển khai - Backtest",
        "publishCode": "Triển khai - Mã",
        "publishLaunch": "Triển khai - Khởi chạy",
        "setup": "Thiết lập"
      },
      "strategyParams": {
        "actions": {
          "addParam": "Thêm tham số",
          "delete": "Xóa",
          "exportJson": "Xuất JSON",
          "importJson": "Nhập JSON"
        },
        "empty": "Chưa có tham số. Bạn có thể thêm fast/slow/risk_per_trade... để chiến lược dễ tái sử dụng.",
        "hints": {
          "intro": "Các tham số này sẽ:",
          "line1": "1) được lưu vào template.parameters",
          "line2": "2) được ghi vào schedule.parameters (map<string,string>) khi tạo lịch",
          "line3Prefix": "3) được tiêm vào chiến lược Python khi chạy dưới dạng"
        },
        "labels": {
          "default": "mặc định",
          "description": "mô tả",
          "label": "nhãn",
          "max": "tối đa",
          "min": "tối thiểu",
          "name": "tên",
          "options": "options (dùng cho select, phân tách bằng dấu phẩy)",
          "step": "bước",
          "type": "kiểu",
          "value": "value (giá trị hiện tại của lịch)"
        },
        "messages": {
          "copied": "Đã sao chép",
          "copyFailed": "Sao chép thất bại",
          "importFormatInvalid": "Định dạng nhập không hợp lệ: cần mảng hoặc {\"paramDefs\": [...] }",
          "importMissingName": "Nhập thất bại: có mục thiếu name",
          "imported": "Đã nhập {{count}} tham số",
          "jsonParseFailed": "Phân tích JSON thất bại"
        },
        "modals": {
          "copyAndClose": "Sao chép và đóng",
          "exportTitle": "Xuất JSON tham số",
          "importOk": "Nhập",
          "importTitle": "Nhập JSON tham số"
        },
        "paramCardTitle": "Tham số #{{index}}",
        "placeholders": {
          "defaultExample": "vd: 10",
          "description": "Mô tả",
          "importJson": "Dán JSON tham số (mảng hoặc {\"paramDefs\": [...]})",
          "label": "Tên hiển thị",
          "nameExample": "vd: fast",
          "optionsExample": "vd: low,medium,high",
          "value": "Để trống sẽ dùng default"
        },
        "title": "Tham số chiến lược (tùy chọn)",
        "types": {
          "bool": "bool",
          "number": "số",
          "select": "chọn",
          "string": "chuỗi"
        },
        "validations": {
          "nameRequired": "name là bắt buộc",
          "typeRequired": "type là bắt buộc"
        }
      },
      "subtitle": "Mỗi bước một trang, bạn có thể tiến/lùi",
      "template": {
        "defaultDescription": "Tạo bởi trình hướng dẫn AI",
        "defaultName": "Chiến lược AI {{title}}"
      },
      "title": "Trình hướng dẫn chiến lược AI"
    }
  }
} as const;
export default AiWizard;
