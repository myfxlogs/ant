// Auto-generated from proto/ant/v1/i18n/ai_settings_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiSettings = {
  "ai": {
    "settings": {
      "actions": {
        "saveConfig": "Lưu cấu hình",
        "validateApiKey": "Xác minh API key"
      },
      "agent": {
        "actions": {
          "add": "Thêm",
          "loadDefaults": "Tải 8 agent mặc định",
          "remove": "Xoá",
          "restoreDefaults": "Khôi phục mặc định",
          "restoreDefaultsConfirmContent": "Thao tác này sẽ đặt lại 8 agent hệ thống (style/signals/risk/macro/sentiment/portfolio/execution/code) về nhân dạng mặc định. Các agent tự thêm sẽ được giữ. Chỉ chỉnh sửa bản náp, phải bấm Lưu mới được lưu vào CSDL.",
          "restoreDefaultsConfirmTitle": "Khôi phục nhân dạng mặc định?",
          "save": "Lưu"
        },
        "defaultName": "Agent tùy chỉnh",
        "defaults": {
          "code": {
            "inputHint": "Ví dụ: trend-following EMA(fast)/EMA(slow) với bộ lọc ATR; params = fast, slow, atr_period, risk_per_trade."
          },
          "execution": {
            "inputHint": "Ví dụ: mua 10 lot EURUSD; spread = 0,6 pip; mục tiêu 5 phút; trượt giá tối đa = 0,8 pip."
          },
          "executor": {
            "identity": "Chuyên gia tối ưu hóa thực thi giao dịch — giảm thiểu trượt giá và chi phí thực thi."
          },
          "macro": {
            "inputHint": "Ví dụ: sự kiện chính = CPI Mỹ và biên bản FOMC; mã mục tiêu = XAUUSD."
          },
          "portfolio": {
            "inputHint": "Ví dụ: chiến lược = trend-EURUSD và mean-reversion-XAUUSD; vốn = 50.000; vol mục tiêu = 12% năm."
          },
          "researcher": {
            "identity": "Nhà nghiên cứu kinh tế vĩ mô và ngành — phân tích sự kiện vĩ mô và xu hướng ngành."
          },
          "risk": {
            "inputHint": "Ví dụ: vốn = 10.000; giới hạn drawdown tháng = 5%; rủi ro mỗi giao dịch = 0,5%; giao dịch trong ngày <= 5; cắt lỗ = 1,5×ATR."
          },
          "risk_manager": {
            "identity": "Chuyên gia kiểm soát rủi ro nghiêm ngặt — thiết kế quy mô vị thế, cắt lỗ, giới hạn sụt giảm."
          },
          "sentiment": {
            "inputHint": "Ví dụ: VIX từ 14 lên 22; vị thế long ròng phi thương mại -18%; tin tức chủ đạo về suy thoái / cắt giảm lãi suất."
          },
          "signals": {
            "inputHint": "Ví dụ: mô hình = trend-following; khung thời gian = H1; chỉ báo = EMA/ATR/ADX; fast = 20, slow = 60."
          },
          "strategist": {
            "identity": "Chuyên viên phân tích chiến lược định lượng cấp cao — đề xuất mô hình chiến lược dựa trên điều kiện tài khoản/thị trường."
          },
          "style": {
            "inputHint": "Ví dụ: tài khoản = EURUSD cá nhân; khung thời gian = H1; mục tiêu = lợi nhuận 3%/tháng, drawdown tối đa <10%; ưu tiên = tỷ lệ thắng hơn tỷ lệ lời/lỗ."
          }
        },
        "fields": {
          "historicalBinding": "{{value}} (lịch sử)",
          "identityPlaceholder": "Nhân dạng / persona (ghép vào system prompt)",
          "inputHintPlaceholder": "Gợi ý nhập (tuỳ chọn)",
          "modelProfileEmpty": "Vui lòng bật ít nhất một provider/model trong \"Cài đặt AI\" trước",
          "modelProfilePlaceholder": "Mặc định (dùng cấu hình hiện tại)",
          "namePlaceholder": "Tên agent"
        },
        "messages": {
          "defaultsLoaded": "Đã tải mẫu agent mặc định. Bấm Lưu để lưu vào CSDL.",
          "empty": "Chưa có agent tuỳ chỉnh, bấm \"Thêm\" để bắt đầu",
          "loading": "Đang tải...",
          "saveFailed": "Lưu agents thất bại",
          "saveSuccess": "Đã lưu agents",
          "selectProfileFirst": "Vui lòng chọn một cấu hình ở bên trái trước"
        },
        "removeConfirmContent": "Bạn chắc chắn muốn xoá agent này?",
        "removeConfirmTitle": "Xoá Agent",
        "title": "Định nghĩa Agent",
        "types": {
          "code": "Mã",
          "execution": "Thực thi",
          "executor": "Cố vấn thực thi",
          "macro": "Vĩ mô",
          "portfolio": "Danh mục",
          "researcher": "Nhà nghiên cứu thị trường",
          "risk": "Kiểm soát rủi ro",
          "risk_manager": "Quản lý rủi ro",
          "sentiment": "Tâm lý",
          "signals": "Tín hiệu",
          "strategist": "Chuyên viên phân tích chiến lược",
          "style": "Phong cách"
        }
      },
      "apiKeyGuide": {
        "deepseek": {
          "step1": "Mở nền tảng DeepSeek: ",
          "step2": "Đăng nhập/đăng ký, sau đó tạo và sao chép API key trong trang API Keys",
          "title": "Lấy DeepSeek API key"
        },
        "default": "Current provider: {{provider}}. Go to the provider\\\\\\\\\\\\\\\\",
        "modelSuggestionDeepSeek": "模型建议: 在\"模型\"下拉中选择 `deepseek-chat`",
        "modelSuggestionZhipu": "模型建议: 在\"模型\"下拉中选择 `glm-4-flash` / `glm-4`",
        "selectProviderHint": "Chọn nhà cung cấp để xem hướng dẫn lấy API key.",
        "title": "Hướng dẫn lấy API key",
        "zhipu": {
          "step1": "Mở nền tảng Zhipu: ",
          "step2": "Đăng nhập/đăng ký, sau đó tạo và sao chép API key",
          "title": "Lấy Zhipu API key"
        }
      },
      "apiKeySavedAs": "Đã lưu: {{masked}}",
      "defaultProfileName": "Mặc định",
      "discoverErrors": {
        "baseUrlInvalid": "Base URL không hợp lệ: dùng URL đầy đủ, ví dụ https://model.example.com hoặc https://model.example.com/v1",
        "baseUrlRequired": "Vui lòng nhập Base URL (địa chỉ dịch vụ model).",
        "endpoint404": "Không tìm thấy endpoint model: kiểm tra Base URL có khớp API tương thích OpenAI (một số dịch vụ cần /v1).",
        "freeTierExhausted": "Đã hết miễn phí: tắt chế độ chỉ free tier trên console hoặc đổi sang key trả phí.",
        "generic": "Không tải được danh sách model. Kiểm tra Base URL và API key.",
        "genericDetail": "Không tải được danh sách model: {{detail}}",
        "invalidModelsResponse": "Phản hồi không tương thích giao thức /models.",
        "noModelsReturned": "Không có model khả dụng: kiểm tra quyền tài khoản hoặc cấu hình.",
        "quotaForbidden403": "Bị từ chối (quota): kiểm tra thanh toán/quota trên console.",
        "quotaOrRateLimit": "Hết quota hoặc bị giới hạn tốc độ: nhà cung cấp từ chối. Kiểm tra thanh toán/giới hạn hoặc thử lại sau.",
        "timeout": "Hết thời gian chờ: kiểm tra mạng hoặc thử lại sau.",
        "unauthorized": "Xác thực thất bại: kiểm tra API key/secret.",
        "unreachable": "Không kết nối được dịch vụ model: kiểm tra Base URL, mạng hoặc gateway."
      },
      "errors": {
        "arrearage": "Phản hồi từ nhà cung cấp: tài khoản nợ phí/thiếu số dư hoặc trạng thái bất thường. Vui lòng kiểm tra hóa đơn và trạng thái tài khoản.",
        "forbidden": "Phản hồi từ nhà cung cấp: bị từ chối (403). Vui lòng kiểm tra quyền, IP allowlist hoặc trạng thái tài khoản.",
        "invalidModelId": "Phản hồi từ nhà cung cấp: mô hình không khả dụng{{model}}. Vui lòng chọn từ danh sách hoặc dùng đúng model id.",
        "timeout": "Hết thời gian chờ. Vui lòng kiểm tra Base URL/mạng và thử lại.",
        "unauthorized": "Phản hồi từ nhà cung cấp: không được ủy quyền (401). Vui lòng kiểm tra API key và quyền."
      },
      "fields": {
        "apiKey": "Khóa API",
        "apiKeyConfigured": "Đã cấu hình",
        "apiKeyReplaceHint": "Để thay key, nhập lại tại đây",
        "availableModels": "Model khả dụng",
        "availableModelsEmpty": "Gõ model id rồi nhấn Enter để thêm",
        "availableModelsHint": "Có thể bật nhiều model dùng chung một API key. Danh sách này hiện trong dropdown của /ai/agents. Mặc định trống — chọn từ dropdown hoặc gõ model id rồi Enter để thêm; chỉ giữ những model bạn chọn rõ ràng.",
        "availableModelsPlaceholder": "Chọn từ dropdown hoặc gõ model id rồi Enter (mặc định trống)",
        "availableModelsTip": "Lưu ý: xoá một model không tự huỷ các Agent đã liên kết model đó tại /ai/agents, nhưng nó sẽ biến mất khỏi gợi ý dropdown.",
        "baseUrl": "URL Cơ sở",
        "baseUrlHint": " (địa chỉ dịch vụ model)",
        "clear": "Xoá hết",
        "defaultModel": "Model mặc định",
        "deleteApiKey": "Xoá key",
        "enabledOff": "Đang tắt → nhấp để bật",
        "enabledOn": "Đang bật → nhấp để tắt",
        "enabledStatus": "Đã bật",
        "maxTokens": "Số token tối đa",
        "model": "Mô hình",
        "name": "Tên",
        "provider": "Nhà cung cấp AI",
        "temperature": "Nhiệt độ (Temperature)",
        "timeoutSeconds": "Thời gian chờ (giây)"
      },
      "inferenceParams": {
        "title": "Tham số suy luận"
      },
      "messages": {
        "apiKeyValidated": "API key hợp lệ",
        "deleted": "Đã xóa",
        "disabled": "Đã tắt",
        "enabled": "Đã bật",
        "loadConfigFailed": "Tải cấu hình AI thất bại",
        "probeFailed": "Kết nối thất bại",
        "probeSuccess": "Kết nối thành công",
        "saveSuccess": "Lưu thành công",
        "selectSavedProfileOrEnterKey": "Vui lòng chọn cấu hình đã lưu hoặc nhập API key",
        "setCurrentSuccess": "Đã chuyển cấu hình hiện tại",
        "validateBeforeSave": "Vui lòng xác minh API key trước khi lưu",
        "validateFailed": "Xác minh thất bại",
        "validateSuccess": "Xác minh thành công"
      },
      "pageTitle": "Cài đặt trợ lý AI",
      "placeholders": {
        "apiKey": "Nhập API key",
        "baseUrl": "VD: https://api.example.com/v1",
        "modelManual": "Nhập tên mô hình (khuyến nghị copy model id từ trang quản lý)",
        "modelSelect": "Chọn mô hình",
        "modelSelectOrType": "Chọn từ danh sách hoặc nhập ID mô hình",
        "name": "VD: DeepSeek - chi phí thấp",
        "provider": "Chọn nhà cung cấp AI",
        "providerFirst": "Vui lòng chọn nhà cung cấp trước"
      },
      "primary": {
        "hint": "Dùng cho bước \"Làm rõ ý định\", sinh mã, panel \"Trợ lý AI — sửa mã\" trong trình soạn template, và bất kỳ Agent nào chưa chọn model riêng.",
        "placeholder": "Chọn một provider · model làm bộ não mặc định",
        "title": "Mô hình chính mặc định"
      },
      "profiles": {
        "actions": {
          "setCurrent": "Đặt hiện tại"
        },
        "current": "Hiện tại",
        "delete": {
          "content": "Xóa cấu hình này?",
          "title": "Xóa cấu hình"
        }
      },
      "providers": {
        "anthropic": "Anthropic Claude",
        "custom": "Tùy chỉnh (tương thích OpenAI)",
        "deepseek": "DeepSeek",
        "doubao": "Doubao",
        "emptyHint": "Cấu hình API key và model khả dụng tại ",
        "emptyHintTail": "。",
        "emptyTitle": "Chưa có nhà cung cấp nào được bật",
        "enabledTitle": "Nhà cung cấp đã bật",
        "groq": "Groq",
        "mistral": "Mistral",
        "modelsUnit": "model",
        "moonshot": "月之暗面 Moonshot (Kimi)",
        "noModels": "Chưa có model khả dụng",
        "openai": "OpenAI",
        "openai_compatible": "Tùy chỉnh (tương thích OpenAI)",
        "openrouter": "OpenRouter",
        "qwen": "Qwen / DashScope",
        "siliconflow": "硅基流动 SiliconFlow",
        "zhipu": "智谱 AI"
      },
      "sections": {
        "advanced": "Tham số nâng cao",
        "advancedHint": "Chỉ chỉnh khi bạn hiểu rõ ý nghĩa; giá trị mặc định phù hợp đa số kịch bản",
        "basic": "Thông tin cơ bản",
        "connection": "Cấu hình kết nối",
        "connectionApiKeyLink": "Đến trang đăng ký / quản lý API key của nhà cung cấp"
      },
      "tabs": {
        "agents": "Cấu hình Tác nhân",
        "config": "Cấu hình Mô hình"
      },
      "validation": {
        "apiKeyRequired": "API key là bắt buộc",
        "baseUrlNoChatCompletionsSuffix": "Base URL không nên kết thúc bằng /chat/completions",
        "baseUrlProtocol": "Base URL phải bắt đầu bằng http:// hoặc https://",
        "baseUrlRequired": "Base URL là bắt buộc",
        "modelFormat": "Định dạng mô hình không hợp lệ",
        "modelRequired": "Mô hình là bắt buộc",
        "nameRequired": "Tên là bắt buộc"
      }
    }
  }
} as const;
export default AiSettings;
