// Parameter label translations for common strategy parameter names.
// These are the canonical labels displayed in BacktestPanel and StrategyParamsModal.
// Keys are the raw parameter names (as extracted from self.ctx.param() calls).
// Values are locale-specific translations.

const PARAM_LABEL_I18N: Record<string, Record<string, string>> = {
  // ── 马丁 (Martingale) strategy params ──
  '多空方向': { en: 'Trade Direction', 'zh-cn': '多空方向', 'zh-tw': '多空方向', ja: 'トレード方向', vi: 'Hướng giao dịch' },
  '起始下单量': { en: 'Initial Lot', 'zh-cn': '起始下单量', 'zh-tw': '起始下單量', ja: '初期ロット', vi: 'Khối lượng ban đầu' },
  '滑点': { en: 'Slippage', 'zh-cn': '滑点', 'zh-tw': '滑點', ja: 'スリッページ', vi: 'Trượt giá' },
  '显示止盈价格': { en: 'Show TP Price', 'zh-cn': '显示止盈价格', 'zh-tw': '顯示止盈價格', ja: 'TP価格表示', vi: 'Hiển thị giá TP' },
  '显示浮亏单数': { en: 'Show Floating Loss Count', 'zh-cn': '显示浮亏单数', 'zh-tw': '顯示浮虧單數', ja: '浮動損失数表示', vi: 'Hiển thị số lệnh lỗ' },
  '启用虚拟下单': { en: 'Virtual Orders', 'zh-cn': '启用虚拟下单', 'zh-tw': '啟用虛擬下單', ja: '仮想注文', vi: 'Lệnh ảo' },
  '虚拟下单单数': { en: 'Virtual Order Count', 'zh-cn': '虚拟下单单数', 'zh-tw': '虛擬下單數', ja: '仮想注文数', vi: 'Số lệnh ảo' },
  '单向最大单数': { en: 'Max Positions Per Side', 'zh-cn': '单向最大单数', 'zh-tw': '單向最大單數', ja: '片側最大ポジション', vi: 'Vị thế tối đa mỗi bên' },
  '资金2W单笔最大下单量': { en: 'Max Lot per 20K', 'zh-cn': '资金2W单笔最大下单量', 'zh-tw': '資金2W單筆最大下單量', ja: '2万あたり最大ロット', vi: 'Lot tối đa mỗi 20K' },
  '翻倍': { en: 'Multiplier', 'zh-cn': '翻倍', 'zh-tw': '翻倍', ja: '倍率', vi: 'Hệ số nhân' },
  '单K线限制点数': { en: 'Single Bar Point Limit', 'zh-cn': '单K线限制点数', 'zh-tw': '單K線限制點數', ja: 'バーあたりポイント制限', vi: 'Giới hạn điểm mỗi nến' },
  '达到限制暂停小时': { en: 'Cooldown Hours', 'zh-cn': '达到限制暂停小时', 'zh-tw': '達到限制暫停小時', ja: 'クールダウン時間', vi: 'Giờ tạm dừng' },
  '间隔单数': { en: 'Grid Spacing Count', 'zh-cn': '间隔单数', 'zh-tw': '間隔單數', ja: 'グリッド間隔数', vi: 'Số lệnh giãn cách' },
  '单数以下间隔点数': { en: 'Grid Spacing Below', 'zh-cn': '单数以下间隔点数', 'zh-tw': '單數以下間隔點數', ja: '下限グリッド間隔', vi: 'Khoảng cách dưới' },
  '单数以上间隔点数': { en: 'Grid Spacing Above', 'zh-cn': '单数以上间隔点数', 'zh-tw': '單數以上間隔點數', ja: '上限グリッド間隔', vi: 'Khoảng cách trên' },
  '总体盈利点数': { en: 'Global Profit Points', 'zh-cn': '总体盈利点数', 'zh-tw': '總體盈利點數', ja: '全体利益ポイント', vi: 'Điểm lợi nhuận toàn cục' },
  '总亏损金额平仓': { en: 'Max Loss Close', 'zh-cn': '总亏损金额平仓', 'zh-tw': '總虧損金額平倉', ja: '最大損失クローズ', vi: 'Đóng khi lỗ tối đa' },
  '总盈利金额平仓': { en: 'Max Profit Close', 'zh-cn': '总盈利金额平仓', 'zh-tw': '總盈利金額平倉', ja: '最大利益クローズ', vi: 'Đóng khi lời tối đa' },
  '定单识别码': { en: 'Magic Number', 'zh-cn': '定单识别码', 'zh-tw': '定單識別碼', ja: 'マジックナンバー', vi: 'Magic Number' },
  '定单注释': { en: 'Order Comment', 'zh-cn': '定单注释', 'zh-tw': '定單註釋', ja: '注文コメント', vi: 'Ghi chú lệnh' },
  '启用时间控制': { en: 'Time Control', 'zh-cn': '启用时间控制', 'zh-tw': '啟用時間控制', ja: '時間制御', vi: 'Kiểm soát thời gian' },
  '开始小时': { en: 'Start Hour', 'zh-cn': '开始小时', 'zh-tw': '開始小時', ja: '開始時間', vi: 'Giờ bắt đầu' },
  '开始分钟': { en: 'Start Minute', 'zh-cn': '开始分钟', 'zh-tw': '開始分鐘', ja: '開始分', vi: 'Phút bắt đầu' },
  '结束小时': { en: 'End Hour', 'zh-cn': '结束小时', 'zh-tw': '結束小時', ja: '終了時間', vi: 'Giờ kết thúc' },
  '结束分钟': { en: 'End Minute', 'zh-cn': '结束分钟', 'zh-tw': '結束分鐘', ja: '終了分', vi: 'Phút kết thúc' },

  // ── Common params ──
  'fast_period': { en: 'Fast Period', 'zh-cn': '快线周期', 'zh-tw': '快線週期', ja: 'Fast期間', vi: 'Chu kỳ nhanh' },
  'slow_period': { en: 'Slow Period', 'zh-cn': '慢线周期', 'zh-tw': '慢線週期', ja: 'Slow期間', vi: 'Chu kỳ chậm' },
  'rsi_period': { en: 'RSI Period', 'zh-cn': 'RSI 周期', 'zh-tw': 'RSI 週期', ja: 'RSI期間', vi: 'Chu kỳ RSI' },
  'atr_period': { en: 'ATR Period', 'zh-cn': 'ATR 周期', 'zh-tw': 'ATR 週期', ja: 'ATR期間', vi: 'Chu kỳ ATR' },
  'entryPct': { en: 'Entry %', 'zh-cn': '入场比例', 'zh-tw': '入場比例', ja: 'エントリー%', vi: 'Tỷ lệ vào lệnh' },
  'stopLossPct': { en: 'Stop Loss %', 'zh-cn': '止损比例', 'zh-tw': '止損比例', ja: 'SL%', vi: 'Tỷ lệ SL' },
  'takeProfitPct': { en: 'Take Profit %', 'zh-cn': '止盈比例', 'zh-tw': '止盈比例', ja: 'TP%', vi: 'Tỷ lệ TP' },
  'lot_size': { en: 'Lot Size', 'zh-cn': '手数', 'zh-tw': '手數', ja: 'ロットサイズ', vi: 'Khối lượng' },
};

/**
 * Look up a translated label for a strategy parameter name.
 * Falls back to the raw name if no translation is available.
 */
export function paramLabel(name: string, locale: string, fallbackLabel?: string): string {
  const entry = PARAM_LABEL_I18N[name];
  if (!entry) return fallbackLabel || name;

  // Try the exact locale first, then the language prefix, then English, then fallback.
  return entry[locale]
    || entry[locale.split('-')[0]]
    || entry['zh-cn']
    || entry['en']
    || fallbackLabel
    || name;
}
