// Auto-generated from proto/ant/v1/i18n/strategy_experiment_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyExperiment = {
  "strategy": {
    "experiment": {
      "candidates": {
        "column": {
          "actions": "Thao tác",
          "generateDraft": "Tạo Bản Nháp",
          "grade": "Xếp Hạng",
          "parameters": "Tham Số",
          "rank": "Hạng",
          "recommendation": "Khuyến Nghị",
          "score": "Điểm",
          "summary": "Tóm Tắt",
          "viewCandidates": "Xem Ứng Viên"
        },
        "title": "Ứng Viên",
        "titleWithId": "Ứng Viên: {{id}}"
      },
      "list": {
        "column": {
          "actions": "Thao tác",
          "maxCandidates": "Số Ứng Viên Tối Đa",
          "objective": "Mục Tiêu",
          "searchMethod": "Phương Pháp Tìm Kiếm",
          "status": "Trạng thái",
          "viewCandidates": "Xem Ứng Viên"
        },
        "title": "Danh Sách Thử Nghiệm"
      },
      "messages": {
        "candidatesGenerated": "Ứng viên thử nghiệm chiến lược đã được tạo",
        "draftGenerated": "Mẫu nháp đã tạo: {{templateId}}",
        "loadCandidatesFailed": "Tải ứng viên thất bại",
        "loadExperimentsFailed": "Tải danh sách thử nghiệm thất bại",
        "loadTemplatesFailed": "Tải mẫu chiến lược thất bại",
        "promoteFailed": "Thăng cấp ứng viên thành bản nháp thất bại",
        "submitFailed": "Nộp thử nghiệm thất bại. Vui lòng xác minh không gian tham số là JSON hợp lệ.",
        "subscribeJobFailed": "Đăng ký sự kiện Job thử nghiệm thất bại"
      },
      "submitForm": {
        "baseTemplate": "Mẫu Chiến Lược Cơ Sở",
        "baseTemplatePlaceholder": "Chọn Mẫu",
        "baseTemplateRequired": "Vui lòng chọn mẫu chiến lược cơ sở",
        "maxCandidates": "Số Ứng Viên Tối Đa",
        "objective": "Mục Tiêu",
        "parameterSpace": "JSON Không Gian Tham Số",
        "parameterSpaceRequired": "Vui lòng nhập JSON không gian tham số",
        "searchMethod": "Phương Pháp Tìm Kiếm",
        "submit": "Nộp Thử Nghiệm",
        "title": "Nộp Thử Nghiệm"
      },
      "jobEventStream": "Luồng Sự Kiện Job",
      "noEvents": "Không có sự kiện",
      "ruleVersionAlert": "Vòng lặp tối thiểu hiện tại: thử nghiệm tham số xác định. Ứng viên chỉ tạo bản nháp, không tự động xuất bản, lên lịch hoặc giao dịch.",
      "selectJobToView": "Chọn thử nghiệm có Job để xem sự kiện.",
      "subtitle": "Nộp tổ hợp tham số để tự động chạy thử nghiệm, chấm điểm ứng viên và tạo bản nháp.",
      "title": "Thử Nghiệm Chiến Lược"
    }
  }
} as const;
export default StrategyExperiment;
