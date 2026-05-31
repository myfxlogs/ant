const aiSettings = {
  ai: {
    settings: {
      agent: {
        types: {
          strategist: 'Nhà phân tích chiến lược',
          risk_manager: 'Quản lý rủi ro',
          executor: 'Cố vấn thực thi',
          researcher: 'Nghiên cứu thị trường',
        },
        defaults: {
          strategist: {
            identity: 'Nhà phân tích chiến lược định lượng cấp cao — đề xuất mô hình chiến lược dựa trên điều kiện tài khoản/thị trường.',
          },
          risk_manager: {
            identity: 'Chuyên gia kiểm soát rủi ro nghiêm ngặt — thiết kế định cỡ vị thế, cắt lỗ, giới hạn drawdown.',
          },
          executor: {
            identity: 'Chuyên gia tối ưu hóa thực thi giao dịch — giảm thiểu trượt giá và chi phí thực thi.',
          },
          researcher: {
            identity: 'Nhà nghiên cứu kinh tế vĩ mô và ngành — phân tích sự kiện vĩ mô và xu hướng ngành.',
          },
        },
      },
    },
  },
} as const;

export default aiSettings;
