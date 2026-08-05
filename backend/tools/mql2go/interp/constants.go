package interp

// MQL predefined constants — single source of truth.
//
// All MQL4 and MQL5 predefined constants are defined here. The VM compiler
// (compile_expr.go) consumes this map via LookupMQLConstant().
//
// Sources:
//   - MQL4: https://docs.mql4.com/constants (all sub-pages)
//   - MQL5: https://www.mql5.com/en/docs/constants (all sub-pages)
//
// Constants are organized by enumeration group, matching the official docs.
// Values are int32 for all numeric constants; booleans use BoolVal.

// MQLConstants is the complete map of predefined MQL4/MQL5 constants.
var MQLConstants = map[string]Value{
	// ── ENUM_ORDER_TYPE (MQL4 OP_* and MQL5 ORDER_TYPE_*) ──────────────
	// MQL4: https://docs.mql4.com/constants/tradingconstants/orderproperties
	"OP_BUY":           IntVal(0),
	"OP_SELL":          IntVal(1),
	"OP_BUYLIMIT":      IntVal(2),
	"OP_SELLLIMIT":     IntVal(3),
	"OP_BUYSTOP":       IntVal(4),
	"OP_SELLSTOP":      IntVal(5),
	// MQL5: https://www.mql5.com/en/docs/constants/tradingconstants/orderproperties
	"ORDER_TYPE_BUY":           IntVal(0),
	"ORDER_TYPE_SELL":          IntVal(1),
	"ORDER_TYPE_BUY_LIMIT":     IntVal(2),
	"ORDER_TYPE_SELL_LIMIT":    IntVal(3),
	"ORDER_TYPE_BUY_STOP":      IntVal(4),
	"ORDER_TYPE_SELL_STOP":     IntVal(5),
	"ORDER_TYPE_BUY_STOP_LIMIT":  IntVal(6),
	"ORDER_TYPE_SELL_STOP_LIMIT": IntVal(7),
	"ORDER_TYPE_CLOSE_BY":        IntVal(8),

	// ── Selection / pool modes (MQL4) ──────────────────────────────────
	"SELECT_BY_POS":    IntVal(0),
	"SELECT_BY_TICKET": IntVal(1),
	"MODE_TRADES":      IntVal(0),
	"MODE_HISTORY":     IntVal(1),

	// ── MarketInfo mode constants (MQL4) ───────────────────────────────
	// https://docs.mql4.com/constants/environment_state/marketinfoconstants
	"MODE_LOW":               IntVal(1),
	"MODE_HIGH":              IntVal(2),
	"MODE_TIME":              IntVal(5),
	"MODE_VOLUME":            IntVal(4),
	"MODE_BID":               IntVal(9),
	"MODE_ASK":               IntVal(10),
	"MODE_POINT":             IntVal(11),
	"MODE_DIGITS":            IntVal(12),
	"MODE_SPREAD":            IntVal(13),
	"MODE_STOPLEVEL":         IntVal(14),
	"MODE_LOTSIZE":           IntVal(15),
	"MODE_FREEZELEVEL":       IntVal(16),
	"MODE_TICKVALUE":         IntVal(17),
	"MODE_TICKSIZE":          IntVal(18),
	"MODE_SWAPTYPE":          IntVal(24),  // MQL4 doc says MODE_SWAPTYPE; some code uses MODE_SWAPMODE
	"MODE_SWAPMODE":          IntVal(24),  // alias
	"MODE_STARTING":          IntVal(25),
	"MODE_EXPIRATION":        IntVal(26),
	"MODE_TRADEALLOWED":      IntVal(27),
	"MODE_MINLOT":            IntVal(20),
	"MODE_MAXLOT":            IntVal(21),
	"MODE_LOTSTEP":           IntVal(22),
	"MODE_PROFITCALCMODE":    IntVal(33),
	"MODE_MARGINCALCMODE":    IntVal(34),
	"MODE_MARGININIT":        IntVal(30),
	"MODE_MARGINMAINTENANCE": IntVal(31),
	"MODE_MARGINHEDGED":      IntVal(32),
	"MODE_MARGINREQUIRED":    IntVal(37),
	"MODE_SWAPLONG":          IntVal(35),
	"MODE_SWAPSHORT":         IntVal(36),
	"MODE_CLOSEBY_ALLOWED":   IntVal(28),

	// ── Indicator line selection modes (ENUM_INDEXBUFFER) ──────────────
	// Used by iMACD, iStochastic, iADX, iBands, iEnvelopes, iFractals,
	// iAlligator, iGator, iIchimoku to select which indicator line to return.
	// https://docs.mql4.com/constants/indicatorconstants/lines
	"MODE_MAIN":    IntVal(0), // base/main line (iMACD, iStochastic, iADX, iAlligator, iGator, iIchimoku)
	"MODE_SIGNAL":  IntVal(1), // signal line (iMACD, iStochastic)
	"MODE_PLUSDI":  IntVal(1), // +DI line (iADX)
	"MODE_MINUSDI": IntVal(2), // -DI line (iADX)
	"MODE_UPPER":   IntVal(1), // upper band (iBands, iEnvelopes)
	"MODE_LOWER":   IntVal(2), // lower band (iBands, iEnvelopes)
	"MODE_BASE":    IntVal(0), // base line (iAlligator jaw, iGator)
	"MODE_TENKAN":  IntVal(1), // Tenkan-sen (iIchimoku)
	"MODE_KIJUN":   IntVal(2), // Kijun-sen (iIchimoku)
	"MODE_SENKOUA": IntVal(3), // Senkou Span A (iIchimoku)
	"MODE_SENKOUB": IntVal(4), // Senkou Span B (iIchimoku)
	"MODE_CHIKOU":  IntVal(5), // Chikou Span (iIchimoku)
	// Alligator jaw/teeth/lips (MQL5 names, MQL4 uses MODE_BASE/MODE_UPPER/MODE_LOWER)
	"MODE_GATORJAW":  IntVal(0),
	"MODE_GATORTEETH": IntVal(1),
	"MODE_GATORLIPS":  IntVal(2),

	// ── Stochastic price field constants ────────────────────────────────
	// Used as the price_field argument of iStochastic.
	"STO_LOWHIGH":    IntVal(0),
	"STO_CLOSECLOSE": IntVal(1),

	// ── Moving average methods (ENUM_MA_METHOD) ────────────────────────
	"MODE_SMA":  IntVal(0),
	"MODE_EMA":  IntVal(1),
	"MODE_SMMA": IntVal(2),
	"MODE_LWMA": IntVal(3),

	// ── Applied price (ENUM_APPLIED_PRICE) ─────────────────────────────
	"PRICE_CLOSE":    IntVal(1),
	"PRICE_OPEN":     IntVal(2),
	"PRICE_HIGH":     IntVal(3),
	"PRICE_LOW":      IntVal(4),
	"PRICE_MEDIAN":   IntVal(5),
	"PRICE_TYPICAL":  IntVal(6),
	"PRICE_WEIGHTED": IntVal(7),

	// ── Chart timeframes (ENUM_TIMEFRAMES) ─────────────────────────────
	// https://docs.mql4.com/constants/chartconstants/enum_timeframes
	"PERIOD_CURRENT": IntVal(0),
	"PERIOD_M1":      IntVal(1),
	"PERIOD_M2":      IntVal(2),
	"PERIOD_M3":      IntVal(3),
	"PERIOD_M4":      IntVal(4),
	"PERIOD_M5":      IntVal(5),
	"PERIOD_M6":      IntVal(6),
	"PERIOD_M10":     IntVal(10),
	"PERIOD_M12":     IntVal(12),
	"PERIOD_M15":     IntVal(15),
	"PERIOD_M20":     IntVal(20),
	"PERIOD_M30":     IntVal(30),
	"PERIOD_H1":      IntVal(60),
	"PERIOD_H2":      IntVal(120),
	"PERIOD_H3":      IntVal(180),
	"PERIOD_H4":      IntVal(240),
	"PERIOD_H6":      IntVal(360),
	"PERIOD_H8":      IntVal(480),
	"PERIOD_H12":     IntVal(720),
	"PERIOD_D1":      IntVal(1440),
	"PERIOD_W1":      IntVal(10080),
	"PERIOD_MN1":     IntVal(43200),

	// ── MQL5 trade request actions (ENUM_TRADE_REQUEST_ACTIONS) ────────
	// https://www.mql5.com/en/docs/constants/tradingconstants/enum_trade_request_actions
	"TRADE_ACTION_DEAL":       IntVal(0),
	"TRADE_ACTION_PENDING":    IntVal(1),
	"TRADE_ACTION_SLTP":       IntVal(2),
	"TRADE_ACTION_PEND_CLOSE": IntVal(3),
	"TRADE_ACTION_MODIFY":     IntVal(4),
	"TRADE_ACTION_REMOVE":     IntVal(5),
	"TRADE_ACTION_CLOSE_BY":   IntVal(6),

	// ── MQL5 position types (ENUM_POSITION_TYPE) ───────────────────────
	"POSITION_TYPE_BUY":  IntVal(0),
	"POSITION_TYPE_SELL": IntVal(1),

	// ── MQL5 order states (ENUM_ORDER_STATE) ───────────────────────────
	"ORDER_STATE_STARTED":       IntVal(0),
	"ORDER_STATE_PLACED":        IntVal(1),
	"ORDER_STATE_CANCELED":      IntVal(2),
	"ORDER_STATE_PARTIAL":       IntVal(3),
	"ORDER_STATE_FILLED":        IntVal(4),
	"ORDER_STATE_REJECTED":      IntVal(5),
	"ORDER_STATE_EXPIRED":       IntVal(6),
	"ORDER_STATE_REQUEST_ADD":   IntVal(7),
	"ORDER_STATE_REQUEST_MODIFY": IntVal(8),
	"ORDER_STATE_REQUEST_CANCEL": IntVal(9),

	// ── MQL5 order filling types (ENUM_ORDER_TYPE_FILLING) ─────────────
	"ORDER_FILLING_FOK":  IntVal(0),
	"ORDER_FILLING_IOC":  IntVal(1),
	"ORDER_FILLING_BOC":  IntVal(2),
	"ORDER_FILLING_RETURN": IntVal(3),

	// ── MQL5 order time types (ENUM_ORDER_TYPE_TIME) ───────────────────
	"ORDER_TIME_GTC":     IntVal(0),
	"ORDER_TIME_DAY":     IntVal(1),
	"ORDER_TIME_SPECIFIED": IntVal(2),
	"ORDER_TIME_SPECIFIED_DAY": IntVal(3),

	// ── MQL5 order reasons (ENUM_ORDER_REASON) ─────────────────────────
	"ORDER_REASON_CLIENT": IntVal(0),
	"ORDER_REASON_EXPERT": IntVal(1),
	"ORDER_REASON_SL":     IntVal(2),
	"ORDER_REASON_TP":     IntVal(3),

	// ── MQL5 deal properties (ENUM_DEAL_PROPERTY) ──────────────────────
	"DEAL_TICKET":          IntVal(0),
	"DEAL_ORDER":           IntVal(1),
	"DEAL_TIME":            IntVal(2),
	"DEAL_TYPE":            IntVal(3),
	"DEAL_ENTRY":           IntVal(4),
	"DEAL_MAGIC":           IntVal(5),
	"DEAL_POSITION_ID":     IntVal(6),
	"DEAL_VOLUME":          IntVal(7),
	"DEAL_PRICE":           IntVal(8),
	"DEAL_SL":              IntVal(9),
	"DEAL_TP":              IntVal(10),
	"DEAL_COMMISSION":      IntVal(11),
	"DEAL_SWAP":            IntVal(12),
	"DEAL_PROFIT":          IntVal(13),
	"DEAL_COMMENT":         IntVal(14),

	// ── MQL5 deal types (ENUM_DEAL_TYPE) ───────────────────────────────
	"DEAL_TYPE_BUY":         IntVal(0),
	"DEAL_TYPE_SELL":        IntVal(1),
	"DEAL_TYPE_BALANCE":     IntVal(2),
	"DEAL_TYPE_CREDIT":      IntVal(3),
	"DEAL_TYPE_COMMISSION":  IntVal(4),
	"DEAL_TYPE_CORRECTION":  IntVal(5),
	"DEAL_TYPE_BONUS":       IntVal(6),
	"DEAL_TYPE_CHARGE":      IntVal(7),

	// ── MQL5 position properties ───────────────────────────────────────
	"POSITION_TICKET":          IntVal(0),
	"POSITION_TIME":            IntVal(1),
	"POSITION_TYPE":            IntVal(2),
	"POSITION_MAGIC":           IntVal(3),
	"POSITION_VOLUME":          IntVal(4),
	"POSITION_PRICE_OPEN":      IntVal(5),
	"POSITION_SL":              IntVal(6),
	"POSITION_TP":              IntVal(7),
	"POSITION_PRICE_CURRENT":   IntVal(8),
	"POSITION_SWAP":            IntVal(9),
	"POSITION_PROFIT":          IntVal(10),
	"POSITION_SYMBOL":          IntVal(11),
	"POSITION_COMMENT":         IntVal(12),
	"POSITION_IDENTIFIER":      IntVal(13),

	// ── Special values ─────────────────────────────────────────────────
	"NULL":           IntVal(0),
	"EMPTY":          IntVal(-1),
	"EMPTY_VALUE":    IntVal(-1),
	"WHOLE_ARRAY":    IntVal(-1),
	"INVALID_HANDLE": IntVal(-1),

	// ── Colors (MQL4/MQL5 WebColors) ───────────────────────────────────
	// https://docs.mql4.com/constants/namedconstants/otherconstants
	"Black":           IntVal(0),
	"White":           IntVal(16777215),
	"Red":             IntVal(255),
	"Green":           IntVal(32768),
	"Blue":            IntVal(16711680),
	"Yellow":          IntVal(65535),
	"CLR_NONE":        IntVal(-1),
	"clrNONE":         IntVal(-1),
	"Aqua":            IntVal(16776960),
	"Orange":          IntVal(42495),
	"Gold":            IntVal(55295),
	"Gray":            IntVal(8421504),
	"Silver":          IntVal(12632256),
	"Lime":            IntVal(65280),
	"Olive":           IntVal(32896),
	"Purple":          IntVal(8388736),
	"Teal":            IntVal(8421376),
	"NavajoWhite":     IntVal(1056964608),
	"WhiteSmoke":      IntVal(16119285),
	"DarkGray":        IntVal(9109504),
	"LightGray":       IntVal(13882323),
	"MidnightBlue":    IntVal(1644825),
	"Navy":            IntVal(128),
	"CornflowerBlue":  IntVal(15570276),
	"DodgerBlue":      IntVal(16748574),
	"RoyalBlue":       IntVal(16749338),
	"SteelBlue":       IntVal(11829830),
	"DeepSkyBlue":     IntVal(16760576),
	"SkyBlue":         IntVal(15453815),
	"LightSkyBlue":    IntVal(16436871),
	"LightBlue":       IntVal(15128749),
	"PaleTurquoise":   IntVal(15658671),
	"DarkTurquoise":   IntVal(13749760),
	"Turquoise":       IntVal(13688896),
	"Aquamarine":      IntVal(16753920),
	"Khaki":           IntVal(13434828),
	"DarkKhaki":       IntVal(9054196),
	"YellowGreen":     IntVal(6730598),
	"DarkOrange":      IntVal(36095),
	"OrangeRed":       IntVal(17919),
	"Tomato":          IntVal(4678655),
	"Coral":           IntVal(44279),
	"LightCoral":      IntVal(8421614),
	"Salmon":          IntVal(7504122),
	"DarkSalmon":      IntVal(8034025),
	"LightSalmon":     IntVal(12180223),
	"RosyBrown":       IntVal(7724975),
	"IndianRed":       IntVal(32832),
	"SandyBrown":      IntVal(6333684),
	"Chocolate":       IntVal(1993170),
	"SaddleBrown":     IntVal(1262987),
	"Sienna":          IntVal(6700286),
	"Brown":           IntVal(2763429),
	"Maroon":          IntVal(128),
	"FireBrick":       IntVal(22637),
	"DarkRed":         IntVal(139),
	"GreenYellow":     IntVal(3145645),
	"Chartreuse":      IntVal(65407),
	"LawnGreen":       IntVal(64636),
	"DarkGreen":       IntVal(25600),
	"SpringGreen":     IntVal(8388608),
	"MediumSpringGreen": IntVal(8206336),
	"SeaGreen":        IntVal(5737262),
	"ForestGreen":     IntVal(2263842),
	"LightGreen":      IntVal(9498256),
	"PaleGreen":       IntVal(10025880),
	"DarkSeaGreen":    IntVal(9419919),
	"MediumSeaGreen":  IntVal(7359752),
	"MediumAquamarine": IntVal(6737322),
	"DarkOliveGreen":  IntVal(3107669),
	"OliveDrab":       IntVal(2330179),
	"DarkCyan":        IntVal(9145088),
	"MediumTurquoise": IntVal(13422912),
	"Transparent":     IntVal(-1),

	// ── Time format flags ──────────────────────────────────────────────
	"TIME_DATE":    IntVal(1),
	"TIME_MINUTES": IntVal(2),
	"TIME_SECONDS": IntVal(3),

	// ── Init return values (MQL4/MQL5) ─────────────────────────────────
	"INIT_SUCCEEDED":            IntVal(0),
	"INIT_FAILED":               IntVal(1),
	"INIT_PARAMETERS_INCORRECT": IntVal(2),

	// ── Uninit reasons (ENUM_UNINIT_REASON) ────────────────────────────
	"REASON_REMOVE":     IntVal(1),
	"REASON_RECOMPILE":  IntVal(2),
	"REASON_CHARTCLOSE": IntVal(3),
	"REASON_CHARTCHANGE": IntVal(4),
	"REASON_ACCOUNT":    IntVal(5),
	"REASON_TEMPLATE":   IntVal(6),
	"REASON_INITFAILED": IntVal(7),
	"REASON_CLOSE":      IntVal(8),

	// ── Object types (ENUM_OBJECT) ─────────────────────────────────────
	"OBJ_HLINE":         IntVal(0),
	"OBJ_VLINE":         IntVal(1),
	"OBJ_TREND":         IntVal(2),
	"OBJ_TRENDANGLE":    IntVal(3),
	"OBJ_REGRESSION":    IntVal(4),
	"OBJ_CHANNEL":       IntVal(5),
	"OBJ_STDDEVCHANNEL": IntVal(6),
	"OBJ_GANNLINE":      IntVal(7),
	"OBJ_GANNFAN":       IntVal(8),
	"OBJ_GANNGRID":      IntVal(9),
	"OBJ_FIBO":          IntVal(10),
	"OBJ_ELLIOTWAVE":    IntVal(11),
	"OBJ_RECTANGLE":     IntVal(12),
	"OBJ_TRIANGLE":      IntVal(13),
	"OBJ_ELLIPSE":       IntVal(14),
	"OBJ_PITCHFORK":     IntVal(15),
	"OBJ_CYCLES":        IntVal(16),
	"OBJ_TEXT":          IntVal(17),
	"OBJ_ARROW":         IntVal(18),
	"OBJ_LABEL":         IntVal(19),
	"OBJ_BUTTON":        IntVal(20),
	"OBJ_BITMAP":        IntVal(21),
	"OBJ_EDIT":          IntVal(22),
	"OBJ_EVENT":         IntVal(23),
	"OBJ_RECTANGLE_LABEL": IntVal(24),
	"OBJ_BITMAP_LABEL":    IntVal(25),

	// ── Error codes (MQL4 common) ──────────────────────────────────────
	"ERR_NO_ERROR":              IntVal(0),
	"ERR_NO_RESULT":             IntVal(1),
	"ERR_COMMON_ERROR":          IntVal(2),
	"ERR_INVALID_TRADE_PARAMETERS": IntVal(3),
	"ERR_SERVER_BUSY":           IntVal(4),
	"ERR_OLD_VERSION":           IntVal(5),
	"ERR_NO_CONNECTION":         IntVal(6),
	"ERR_NOT_ENOUGH_RIGHTS":     IntVal(7),
	"ERR_TOO_FREQUENT_REQUESTS": IntVal(8),
	"ERR_MALFUNCTIONAL_TRADE":   IntVal(9),
	"ERR_ACCOUNT_DISABLED":      IntVal(10),
	"ERR_INVALID_ACCOUNT":       IntVal(11),
	"ERR_TRADE_TIMEOUT":         IntVal(12),
	"ERR_INVALID_PRICE":         IntVal(13),
	"ERR_INVALID_STOPS":         IntVal(130),
	"ERR_INVALID_TRADE_VOLUME":  IntVal(131),
	"ERR_NOT_ENOUGH_MONEY":      IntVal(134),
	"ERR_PRICE_CHANGED":         IntVal(135),
	"ERR_OFF_QUOTES":            IntVal(136),
	"ERR_BROKER_BUSY":           IntVal(137),
	"ERR_REQUOTE":               IntVal(138),
	"ERR_ORDER_LOCKED":          IntVal(139),
	"ERR_LONG_POSITIONS_ONLY":   IntVal(140),
	"ERR_TOO_MANY_REQUESTS":     IntVal(141),
	"ERR_TRADE_MODIFY_DENIED":   IntVal(145),
	"ERR_TRADE_CONTEXT_BUSY":    IntVal(146),
	"ERR_TRADE_EXPIRATION_DENIED": IntVal(147),

	// ── MQL5 symbol info constants ─────────────────────────────────────
	"SYMBOL_SELECT":              IntVal(0),
	"SYMBOL_VISIBLE":             IntVal(1),
	"SYMBOL_TIME":                IntVal(5),
	"SYMBOL_DIGITS":              IntVal(12),
	"SYMBOL_SPREAD_FLOAT":        IntVal(14),
	"SYMBOL_SPREAD":              IntVal(13),
	"SYMBOL_TRADE_CALC_MODE":     IntVal(15),
	"SYMBOL_TRADE_MODE":          IntVal(16),
	"SYMBOL_START_TIME":          IntVal(17),
	"SYMBOL_EXPIRATION_TIME":     IntVal(18),
	"SYMBOL_TRADE_STOPS_LEVEL":   IntVal(19),
	"SYMBOL_TRADE_FREEZE_LEVEL":  IntVal(20),
	"SYMBOL_TRADE_EXEMODE":       IntVal(21),
	"SYMBOL_SWAP_MODE":           IntVal(22),
	"SYMBOL_SWAP_ROLLOVER3DAYS":  IntVal(23),
	"SYMBOL_BID":                 IntVal(0),
	"SYMBOL_ASK":                 IntVal(1),
	"SYMBOL_POINT":               IntVal(2),
	"SYMBOL_TRADE_TICK_VALUE":    IntVal(3),
	"SYMBOL_TRADE_TICK_SIZE":     IntVal(4),
	"SYMBOL_TRADE_CONTRACT_SIZE": IntVal(5),
	"SYMBOL_VOLUME_MIN":          IntVal(6),
	"SYMBOL_VOLUME_MAX":          IntVal(7),
	"SYMBOL_VOLUME_STEP":         IntVal(8),
	"SYMBOL_SWAP_LONG":           IntVal(9),
	"SYMBOL_SWAP_SHORT":          IntVal(10),
	"SYMBOL_MARGIN_INITIAL":      IntVal(11),
	"SYMBOL_MARGIN_MAINTENANCE":  IntVal(12),

	// ── MQL5 symbol trade modes (ENUM_SYMBOL_TRADE_MODE) ───────────────
	"SYMBOL_TRADE_MODE_DISABLED":  IntVal(0),
	"SYMBOL_TRADE_MODE_LONGONLY":  IntVal(1),
	"SYMBOL_TRADE_MODE_SHORTONLY": IntVal(2),
	"SYMBOL_TRADE_MODE_CLOSEONLY": IntVal(3),
	"SYMBOL_TRADE_MODE_FULL":      IntVal(4),

	// ── MQL5 symbol trade execution (ENUM_SYMBOL_TRADE_EXECUTION) ──────
	"SYMBOL_TRADE_EXECUTION_REQUEST":  IntVal(0),
	"SYMBOL_TRADE_EXECUTION_INSTANT":   IntVal(1),
	"SYMBOL_TRADE_EXECUTION_MARKET":    IntVal(2),
	"SYMBOL_TRADE_EXECUTION_EXCHANGE":  IntVal(3),

	// ── Account stopout modes ──────────────────────────────────────────
	"ACCOUNT_STOPOUT_MODE_NONE":    IntVal(0),
	"ACCOUNT_STOPOUT_MODE_PERCENT": IntVal(1),
	"ACCOUNT_STOPOUT_MODE_MONEY":   IntVal(2),

	// ── Account info double properties (MQL5) ──────────────────────────
	"ACCOUNT_BALANCE":           IntVal(0),
	"ACCOUNT_CREDIT":            IntVal(1),
	"ACCOUNT_PROFIT":            IntVal(2),
	"ACCOUNT_EQUITY":            IntVal(3),
	"ACCOUNT_MARGIN":            IntVal(4),
	"ACCOUNT_MARGIN_FREE":       IntVal(5),
	"ACCOUNT_MARGIN_LEVEL":      IntVal(6),
	"ACCOUNT_MARGIN_INITIAL":    IntVal(7),
	"ACCOUNT_MARGIN_MAINTENANCE": IntVal(8),
	"ACCOUNT_MARGIN_SO_CALL":    IntVal(9),
	"ACCOUNT_MARGIN_SO_SO":      IntVal(10),

	// ── Booleans ───────────────────────────────────────────────────────
	"true":  BoolVal(true),
	"false": BoolVal(false),
}

// LookupMQLConstant resolves a predefined MQL constant by name.
// Returns the value and true if found, or NoneVal and false if not.
// This is the single entry point used by the VM compiler (compile_expr.go)
// to resolve ExprConst nodes.
func LookupMQLConstant(name string) (Value, bool) {
	v, ok := MQLConstants[name]
	return v, ok
}

// IsMQLConstant checks if a name is a predefined MQL constant.
func IsMQLConstant(name string) bool {
	_, ok := MQLConstants[name]
	return ok
}
