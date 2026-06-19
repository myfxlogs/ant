// Auto-generated from proto/ant/v1/i18n/strategy_tuning_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTuning = {
  "strategy": {
    "tuning": {
      "apply": "Áp Dụng",
      "degradation": "Suy Giảm",
      "enabledCombinations": "{{enabled}} bật · {{combos}} tổ hợp",
      "grade": "Xếp Hạng",
      "gridWarning": "Tìm Kiếm Lưới sẽ kiểm tra <b>{{count}}</b> tổ hợp (giới hạn: 48). Cân nhắc chuyển sang <b>Tiến Hóa Vi Phân</b> để xử lý không gian tham số lớn hiệu quả.",
      "hide": "Ẩn",
      "oosFootnote": "Xác thực OOS trên 5 ứng viên hàng đầu (theo điểm IS). Suy giảm xanh <20%, cam 20-40%, đỏ >40%.",
      "oosScore": "Điểm OOS",
      "optimizer": {
        "ags": "Gaussian Ủ",
        "agsDesc": "Nhiễu Gaussian với ủ sigma. Thay thế nhẹ cho TPE.",
        "ai": "Tối Ưu AI",
        "aiDesc": "Đề xuất đa vòng LLM. Học từ kết quả trước qua 3 vòng.",
        "de": "Tiến Hóa Vi Phân",
        "deDesc": "Đột biến rand/1/bin. Hội tụ nhanh trên không gian mượt.",
        "grid": "Tìm Kiếm Lưới",
        "gridDesc": "Tích Descartes đầy đủ. Tốt nhất cho ≤3 tham số.",
        "random": "Tìm Kiếm Ngẫu Nhiên",
        "randomDesc": "Lấy mẫu ngẫu nhiên đều. Tốt cho khám phá.",
        "tpe": "TPE (核密度估计)",
        "tpeDesc": "Công cụ Ước lượng Parzen Cấu Trúc Cây. KDE mô hình hóa phân phối tốt/xấu."
      },
      "optimizerMethod": "Phương Pháp Tối Ưu",
      "overfit": "Quá Khớp",
      "overfitWarning": "⚠ QUÁ KHỚP",
      "parameterDimensions": "Chiều Tham Số",
      "parameters": "Tham Số",
      "preview": "Xem trước tín hiệu",
      "previewTitle": "Xem Trước ({{shown}} / {{total}})",
      "rank": "#",
      "requiresAI": "Cần cấu hình nhà cung cấp AI",
      "results": "Kết Quả ({{count}})",
      "run": "Chạy ({{count}})",
      "score": "Điểm",
      "started": "Tinh Chỉnh Thông Minh đã bắt đầu",
      "summary": "Tóm Tắt",
      "switchToDE": "Chuyển sang DE",
      "truncated": "ĐÃ CẮT",
      "tuning": "Đang tinh chỉnh…",
      "waiting": "Đợi thử nghiệm... (SSE tự động làm mới)"
    }
  }
} as const;
export default StrategyTuning;
