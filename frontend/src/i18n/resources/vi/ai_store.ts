const aiStore = {
  ai: {
    store: {
      conversations: {
        newConversationTitle: 'Hội thoại mới',
      },
      messages: {
        sendFailedToast: 'Gửi tin nhắn thất bại',
        sendFailedInline: '(Gửi thất bại)',
        loadConversationFailed: 'Tải hội thoại thất bại',
        createConversationFailed: 'Tạo hội thoại thất bại',
        deleteConversationFailed: 'Xóa hội thoại thất bại',
        clearedLocalOnly: 'Đã xóa (chỉ cục bộ)',
      },
      prefs: {
        rememberPrefix: '/remember ',
        rememberedToast: 'Đã lưu tùy chọn',
        savedReply: 'Đã lưu tùy chọn của bạn.',
      },
    },
  },
} as const;

export default aiStore;
