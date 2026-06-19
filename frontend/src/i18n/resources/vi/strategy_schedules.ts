// Auto-generated from proto/ant/v1/i18n/strategy_schedules_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Schedules = {
  "strategy": {
    "schedules": {
      "actions": {
        "create": "Tạo lịch",
        "healthCheck": "Kiểm tra sức khỏe",
        "logs": "Nhật ký chạy",
        "runNow": "Chạy Ngay"
      },
      "createSchedule": "Tạo lịch",
      "deleteConfirm": {
        "title": "Xóa lịch này?"
      },
      "editModal": {
        "advanced": {
          "fixedIntervalSeconds": "Khoảng cố định (giây)",
          "fixedIntervalSecondsExtra": "Tùy chọn. Chạy theo khoảng cố định thay vì theo timeframe. VD: 60 = mỗi 60 giây",
          "hfCooldownMs": "Cooldown cao tần (ms)",
          "hfCooldownMsExtra": "Debounce: khoảng tối thiểu giữa các lần đánh giá/đặt lệnh",
          "parametersJson": "Tham số (JSON object)",
          "parametersJsonExtra": "Tham số JSON cho chiến lược",
          "stableOverrideIntervalSeconds": "Ghi đè khoảng ổn định (giây)",
          "stableOverrideIntervalSecondsExtra": "Tùy chọn. Ghi đè khoảng kích hoạt ở chế độ ổn định",
          "timeframe": "Khung thời gian",
          "timeframeExtra": "Dùng cho tính nến/chỉ báo",
          "title": "Nâng cao",
          "triggerMode": "Chế độ kích hoạt",
          "triggerModeExtra": "Ổn định: theo nến/timeframe; Cao tần: theo báo giá (nhanh hơn, cần debounce)",
          "triggerModeOptions": {
            "hf": "Luồng Tín Hiệu Tần Suất Cao",
            "stable": "Ổn định (nến/timeframe)"
          }
        },
        "autoName": {
          "strategy": "Chiến Lược"
        },
        "fields": {
          "account": "Tài khoản",
          "cronExpression": "Cron (nâng cao)",
          "cronExtra": "Cron 5 phần: phút giờ ngày tháng thứ. VD: */5 * * * *; 0 9 * * 1-5",
          "enableExtra": "Kích hoạt lịch sau khi tạo",
          "intervalSeconds": "Khoảng (giây)",
          "intervalSecondsExtra": "Tự theo timeframe; không cần chỉnh",
          "lot": "Khối lượng (Lot)",
          "lotExtra": "Khối lượng đặt lệnh. Khuyến nghị bắt đầu từ 0.01",
          "name": "Tên",
          "runFrequency": "Tần suất chạy",
          "symbol": "Mã",
          "template": "Mẫu",
          "templateExtra": "Mẫu đã lưu trong “Quản lý chiến lược”"
        },
        "placeholders": {
          "name": "VD: EURUSD M5 chiến lược buổi sáng",
          "selectAccountFirst": "Chọn tài khoản trước",
          "symbol": "Chọn mã"
        },
        "runFrequencyExtra": {
          "byTimeframe": "Chạy Theo Khung TG",
          "cron": "Nâng cao: dùng Cron để điều khiển thời điểm chạy"
        },
        "runFrequencyOptions": {
          "byTimeframe": "Theo timeframe (khuyến nghị)",
          "cron": "Cron"
        },
        "title": {
          "create": "Tạo lịch chạy",
          "edit": "Chỉnh sửa lịch chạy"
        },
        "validation": {
          "accountRequired": "Vui lòng chọn tài khoản",
          "cronRequired": "Vui lòng nhập cron",
          "lotRequired": "Vui lòng nhập khối lượng",
          "nameRequired": "Vui lòng nhập tên",
          "runFrequencyRequired": "Vui lòng chọn tần suất chạy",
          "symbolRequired": "Vui lòng chọn mã",
          "templateRequired": "Vui lòng chọn mẫu",
          "timeframeRequired": "Vui lòng chọn timeframe",
          "triggerModeRequired": "Chế độ kích hoạt là bắt buộc"
        }
      },
      "enableCount": "Số lần bật",
      "format": {
        "cron": "cron: {{expr}}",
        "interval": "mỗi {{s}}s"
      },
      "health": {
        "fields": {
          "configKey": "Khóa cấu hình",
          "failedRuns": "Số lần thất bại",
          "grade": "Mức sức khỏe",
          "lastRunAt": "Lần chạy gần nhất",
          "latestError": "Lỗi Mới Nhất",
          "latestProfit": "Lãi/lỗ gần nhất",
          "latestTicket": "Ticket khớp lệnh gần nhất",
          "rule": "Tiêu chí đánh giá",
          "successOverTotal": "Thành công / Tổng",
          "thresholds": "Ngưỡng hiện tại"
        },
        "grade": {
          "alert": "Cảnh Báo",
          "healthy": "Tốt",
          "noSample": "Thiếu mẫu",
          "pending": "Chưa kiểm tra",
          "watch": "Cần theo dõi"
        },
        "messages": {
          "clickRefresh": "Nhấn làm mới để tải dữ liệu sức khỏe",
          "loadFailed": "Tải dữ liệu sức khỏe thất bại"
        },
        "notes": {
          "alert": "Tỷ lệ thành công thấp. Kiểm tra điều kiện chiến lược/tài khoản ngay.",
          "healthy": "Tỷ lệ thành công cao và số lần thất bại trong ngưỡng.",
          "noSample": "Không đủ mẫu để đánh giá (tối thiểu {{minSampleSize}}).",
          "pending": "Vui lòng chạy kiểm tra sức khỏe trước.",
          "watch": "Tỷ lệ thành công đạt ngưỡng theo dõi (>= {{yellowSuccessRate}}%)."
        },
        "runLogs": {
          "signalType": "Tín hiệu (đặt lệnh)"
        },
        "sections": {
          "orders": "Bản Ghi Lệnh Gần Đây",
          "runLogs": "Nhật ký chạy gần đây"
        },
        "summaryBanner": "Mức sức khỏe: {{grade}}; mẫu gần nhất {{totalRuns}} lần, tỷ lệ thành công {{successRate}}%",
        "thresholdsSummary": "min_sample_size={{minSampleSize}}; xanh: success>={{greenSuccessRate}}% & failed<={{greenMaxFailedRuns}}; vàng: success>={{yellowSuccessRate}}%",
        "title": "Kiểm tra sức khỏe chiến lược {{name}}"
      },
      "messages": {
        "defaultTemplateNotFound": "Không tìm thấy mẫu mặc định. Vui lòng làm mới và thử lại.",
        "executeFailed": "Thực thi thất bại",
        "importDefaultTemplateFailedNoId": "Nhập mẫu mặc định thất bại: thiếu template id",
        "noOrderableSignal": "Không có tín hiệu có thể đặt lệnh",
        "orderFailed": "Đặt lệnh thất bại",
        "orderSubmitted": "Đã gửi lệnh",
        "parametersParseFailed": "Phân tích tham số thất bại",
        "signalHoldCannotOrder": "Tín hiệu là hold/không hành động. Không thể đặt lệnh.",
        "strategyExecuteFailed": "Thực thi chiến lược thất bại",
        "templateCodeEmptyCannotExecute": "Code mẫu trống. Không thể thực thi.",
        "volumeInvalid": "Khối lượng không hợp lệ (phải > 0)"
      },
      "nextRunAt": "Lần chạy kế tiếp",
      "status": {
        "disabled": "Đã Tắt",
        "running": "Đang chạy"
      },
      "table": {
        "account": "Tài khoản",
        "actions": "Thao tác",
        "lastRun": "Lần chạy gần nhất",
        "name": "Tên",
        "schedule": "Lịch",
        "status": "Trạng thái",
        "template": "Mẫu",
        "tradeParams": "Tham số giao dịch"
      },
      "templateVisibility": {
        "private": "Riêng tư",
        "public": "Công khai"
      },
      "title": "Lịch chạy chiến lược",
      "triggerModal": {
        "actions": {
          "confirmOrder": "Xác nhận đặt lệnh",
          "rerun": "Chạy Lại"
        },
        "cards": {
          "logs": "Nhật ký chạy",
          "signal": "Tín hiệu (đặt lệnh)"
        },
        "confirmOrder": {
          "ok": "Xác Nhận",
          "title": "Xác nhận đặt lệnh"
        },
        "emptyLogs": "(không có nhật ký)",
        "emptySignal": "Không Có Tín Hiệu",
        "messages": {
          "signalNotOrderable": "Tín hiệu không thể đặt lệnh"
        },
        "summary": {
          "account": "Tài khoản",
          "scheduleName": "Tên lịch",
          "symbol": "Mã",
          "timeframe": "Khung thời gian"
        },
        "title": "Chạy ngay (đặt lệnh)"
      },
      "validation": {
        "parametersMustBeJsonObject": "Tham số phải là đối tượng JSON"
      }
    }
  }
} as const;
export default Schedules;
