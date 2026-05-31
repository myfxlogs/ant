const aiDebate = {
  ai: {
    debate: {
      v2: {
        steps: {
          agentSelection: 'Chọn chuyên gia',
          intent: 'Làm rõ ý định',
          code: 'Tạo mã',
        },
        agentSelection: {
          title: 'Chọn chuyên gia tham gia thảo luận',
          desc: 'Có thể bỏ qua: nếu không chọn, hệ thống sẽ tạo mã trực tiếp sau khi làm rõ ý định.',
          selectedCount: 'Đã chọn {{count}} chuyên gia',
          next: 'Tiếp theo',
          nextNoAgents: 'Không cần chuyên gia, tiếp theo',
        },
        chat: {
          intentHint: 'Mô tả chiến lược bạn muốn bằng ngôn ngữ tự nhiên.',
          agentHint: 'Trò chuyện với chuyên gia hiện tại bằng ngôn ngữ tự nhiên.',
          placeholder: 'Nói với chuyên gia...',
          send: 'Gửi',
          back: 'Quay lại',
          nextHint: 'Gửi "tiếp theo" hoặc "next" để chuyển bước.',
          autoAdvance: 'Phát hiện ý định "tiếp theo", tự động chuyển bước.',
        },
        code: {
          title: 'Đề xuất mã',
          hint: 'Mã chiến lược được tạo dựa trên sự đồng thuận của tất cả các bước trước.',
          generating: 'Đang tạo mã...',
          regenerating: 'Đang viết lại mã theo phản hồi...',
          elapsed: 'Đã chờ {{time}}',
        },
        modelWait: {
          banner: 'Mô hình đang xử lý, đã chờ {{time}}',
          bubble: 'Đã chờ {{time}}',
        },
        validation: {
          inputFirst: 'Vui lòng nhập yêu cầu của bạn trước',
          selectAgentsFirst: 'Vui lòng chọn ít nhất một chuyên gia',
          emptyOutput: '(trống)',
          needUpstreamBeforeCode: 'Vui lòng chạy các chuyên gia trước khi tạo mã',
          templateSaved: 'Đã lưu làm mẫu riêng',
          backtestFailed: 'Backtest thất bại',
          feedbackRequired: 'Vui lòng nhập phản hồi để chuyên gia sửa đổi',
          loadingAgents: 'Đang tải chuyên gia...',
          noAgentsHint: 'Vui lòng cấu hình chuyên gia trong "Cài đặt AI" trước.',
          enableAgentFirst: 'Chuyên gia chưa được bật, vui lòng bật trong Cài đặt AI.',
        },
        status: {
          currentModel: 'Mô hình hiện tại',
          modelUnknown: 'Chưa cài đặt',
          tokenUsage: 'Token: nhắc {{p}} / trả lời {{c}} / tổng {{n}}',
        },
        roles: 'Vai trò chuyên gia',
        outputs: 'Đầu ra',
        consensus: {
          decision: 'Quyết định',
          constraints: 'Ràng buộc',
          params: 'Tham số',
          rationale: 'Lý do',
        },
      },
    },
  },
} as const;

export default aiDebate;
