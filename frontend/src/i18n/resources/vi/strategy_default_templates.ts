// Auto-generated from proto/ant/v1/i18n/strategy_default_templates_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const DefaultTemplates = {
  "strategy": {
    "defaultTemplates": {
      "forceBuy": {
        "description": "Dùng để xác minh pipeline đặt lệnh: luôn trả về mua, đọc khối lượng từ context/params",
        "name": "Kiểm Tra BUY Bắt Buộc"
      },
      "maCross": {
        "description": "Mua khi MA nhanh cắt lên MA chậm, bán khi cắt xuống",
        "name": "Chiến Lược Giao Cắt MA Kép"
      },
      "macd": {
        "description": "Mua khi MACD cắt vàng, bán khi cắt tử thần",
        "name": "Chiến Lược MACD"
      },
      "rsi": {
        "description": "Mua khi RSI < 30 (quá bán), bán khi RSI > 70 (quá mua)",
        "name": "Chiến Lược RSI Quá Mua/Quá Bán"
      }
    }
  }
} as const;
export default DefaultTemplates;
