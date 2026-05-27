const errors = {
  errors: {
    // General 0-9
    success: 'Thành công',
    unknown: 'Đã xảy ra lỗi không xác định',
    invalid_parameter: 'Tham số không hợp lệ',
    unauthorized: 'Vui lòng đăng nhập để tiếp tục',
    forbidden: 'Truy cập bị từ chối',
    not_found: 'Không tìm thấy tài nguyên',
    internal: 'Lỗi máy chủ nội bộ',
    rate_limited: 'Quá nhiều yêu cầu. Vui lòng thử lại sau.',
    service_unavailable: 'Dịch vụ tạm thời không khả dụng',
    request_timeout: 'Yêu cầu đã hết thời gian',

    // User 1001-1010
    user_not_found: 'Không tìm thấy người dùng',
    user_already_exists: 'Email này đã được đăng ký',
    invalid_password: 'Email hoặc mật khẩu không hợp lệ',
    token_expired: 'Phiên đã hết hạn. Vui lòng đăng nhập lại.',
    token_invalid: 'Token phiên không hợp lệ',
    token_missing: 'Yêu cầu xác thực',
    user_disabled: 'Tài khoản đã bị vô hiệu hóa',
    email_not_verified: 'Email chưa được xác minh',
    password_too_weak: 'Mật khẩu phải có ít nhất 8 ký tự',
    old_password_incorrect: 'Mật khẩu hiện tại không chính xác',

    // Account 2001-2010
    account_not_found: 'Không tìm thấy tài khoản giao dịch',
    account_already_bound: 'Tài khoản giao dịch này đã được liên kết',
    account_connection_failed: 'Không thể kết nối đến máy chủ giao dịch',
    account_disconnected: 'Tài khoản giao dịch đã ngắt kết nối',
    account_auth_failed: 'Xác thực tài khoản giao dịch thất bại',
    account_timeout: 'Kết nối đến máy chủ giao dịch đã hết thời gian',
    account_limit_exceeded: 'Đã đạt số lượng tài khoản giao dịch tối đa',
    invalid_account_type: 'Loại tài khoản không được hỗ trợ',
    account_not_connected: 'Tài khoản giao dịch chưa được kết nối',
    platform_not_supported: 'Nền tảng giao dịch này không được hỗ trợ',

    // Order 3001-3015
    order_not_found: 'Không tìm thấy lệnh',
    order_rejected: 'Lệnh bị từ chối',
    insufficient_margin: 'Ký quỹ không đủ',
    market_closed: 'Thị trường hiện đã đóng cửa',
    invalid_order_type: 'Loại lệnh không hợp lệ',
    invalid_volume: 'Khối lượng giao dịch không hợp lệ',
    invalid_price: 'Giá lệnh không hợp lệ',
    order_timeout: 'Thực hiện lệnh đã hết thời gian',
    position_not_found: 'Không tìm thấy vị thế',
    cannot_close_position: 'Không thể đóng vị thế này',
    cannot_modify_order: 'Không thể sửa lệnh này',
    order_already_filled: 'Lệnh đã được khớp',
    order_already_cancelled: 'Lệnh đã bị hủy',
    slippage_exceeded: 'Trượt giá vượt quá dung sai',
    symbol_not_subscribed: 'Chưa đăng ký mã giao dịch',

    // Market Data 4001-4008
    symbol_not_found: 'Không tìm thấy mã giao dịch',
    no_market_data: 'Không có dữ liệu thị trường',
    subscription_failed: 'Đăng ký dữ liệu thị trường thất bại',
    unsubscription_failed: 'Hủy đăng ký dữ liệu thị trường thất bại',
    quote_not_available: 'Báo giá không khả dụng',
    history_not_available: 'Dữ liệu lịch sử không khả dụng',
    invalid_timeframe: 'Khung thời gian không hợp lệ',
    invalid_time_range: 'Khoảng thời gian không hợp lệ',

    // Analytics 5001-5004
    analytics_not_available: 'Dữ liệu phân tích không khả dụng',
    report_generation_failed: 'Tạo báo cáo thất bại',
    invalid_date_range: 'Khoảng ngày không hợp lệ',
    insufficient_data: 'Không đủ dữ liệu để phân tích',

    // Admin 6001-6003
    admin_access_denied: 'Yêu cầu quyền quản trị viên',
    operation_forbidden: 'Không được phép thực hiện thao tác này',
    audit_log_not_found: 'Không tìm thấy mục nhật ký kiểm toán',

    // Broker 7001-7003
    broker_search_failed: 'Tìm kiếm nhà môi giới thất bại',
    broker_not_found: 'Không tìm thấy nhà môi giới',
    broker_server_unavailable: 'Máy chủ nhà môi giới hiện không khả dụng',
  },
} as const;

export default errors;
