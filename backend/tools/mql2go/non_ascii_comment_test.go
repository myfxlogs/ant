package mql2go

import (
	"testing"
)

// Regression tests for non-ASCII comments in MQL source code.
// MQL source from users in ja/vi/zh locales may contain non-ASCII comments.
// The compiler must handle these without panicking or producing invalid bytecode.
// Per Phase 1.2: code comments should be in English, but we must not crash
// if users submit source with non-ASCII comments.

func TestCompileMQL_NonASCII_Comment_ZH(t *testing.T) {
	source := `
// 这是一个移动平均线交叉策略
extern int FastPeriod = 10;
extern int SlowPeriod = 50;

int OnInit()
{
    // 初始化策略参数
    return 0;
}

void OnBar()
{
    double fastMA = iMA(FastPeriod, 0, MODE_SMA);
    double slowMA = iMA(SlowPeriod, 0, MODE_SMA);
    // 当快线上穿慢线时买入
    if (fastMA > slowMA)
    {
        OrderSend(Symbol(), OP_BUY, 0.1, Ask, 10, 0, 0, "buy", 12345, 0, clrNone);
    }
}

void OnDeinit() {}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL with Chinese comments failed: %v", err)
	}
	if runner == nil {
		t.Fatal("CompileMQL returned nil runner")
	}
}

func TestCompileMQL_NonASCII_Comment_JA(t *testing.T) {
	source := `
// 移動平均線クロスオーバー戦略
extern int FastPeriod = 10;
extern int SlowPeriod = 50;

int OnInit()
{
    // パラメータ初期化
    return 0;
}

void OnBar()
{
    double fastMA = iMA(FastPeriod, 0, MODE_SMA);
    double slowMA = iMA(SlowPeriod, 0, MODE_SMA);
    // ゴールデンクロスで買い
    if (fastMA > slowMA)
    {
        OrderSend(Symbol(), OP_BUY, 0.1, Ask, 10, 0, 0, "buy", 12345, 0, clrNone);
    }
}

void OnDeinit() {}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL with Japanese comments failed: %v", err)
	}
	if runner == nil {
		t.Fatal("CompileMQL returned nil runner")
	}
}

func TestCompileMQL_NonASCII_Comment_VI(t *testing.T) {
	source := `
// Chiến lược giao cắt đường trung bình động
extern int FastPeriod = 10;
extern int SlowPeriod = 50;

int OnInit()
{
    // Khởi tạo tham số chiến lược
    return 0;
}

void OnBar()
{
    double fastMA = iMA(FastPeriod, 0, MODE_SMA);
    double slowMA = iMA(SlowPeriod, 0, MODE_SMA);
    // Khi đường nhanh cắt lên đường chậm thì mua
    if (fastMA > slowMA)
    {
        OrderSend(Symbol(), OP_BUY, 0.1, Ask, 10, 0, 0, "buy", 12345, 0, clrNone);
    }
}

void OnDeinit() {}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL with Vietnamese comments failed: %v", err)
	}
	if runner == nil {
		t.Fatal("CompileMQL returned nil runner")
	}
}

func TestCompileMQL_NonASCII_StringLiteral(t *testing.T) {
	source := `
extern int MagicNumber = 12345;

int OnInit() { return 0; }

void OnBar()
{
    // Non-ASCII in string literal should also work
    OrderSend(Symbol(), OP_BUY, 0.1, Ask, 10, 0, 0, "买入", MagicNumber, 0, clrNone);
}

void OnDeinit() {}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL with non-ASCII string literal failed: %v", err)
	}
	if runner == nil {
		t.Fatal("CompileMQL returned nil runner")
	}
}

func TestCompileMQL_NonASCII_MixedComments(t *testing.T) {
	source := `
// Mixed language comments: English and 中文 and 日本語 and Tiếng Việt
extern int Period = 14;

int OnInit() { return 0; }

void OnBar()
{
    // RSI calculation — RSI計算 — Tính RSI
    double rsi = iRSI(Period, PRICE_CLOSE);
    if (rsi > 70)
    {
        OrderSend(Symbol(), OP_SELL, 0.1, Bid, 10, 0, 0, "sell", 12345, 0, clrNone);
    }
}

void OnDeinit() {}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL with mixed non-ASCII comments failed: %v", err)
	}
	if runner == nil {
		t.Fatal("CompileMQL returned nil runner")
	}
}

func TestCompileToIR_NonASCII_Comment(t *testing.T) {
	source := `// 策略说明
extern int MagicNumber = 12345;
int OnInit() { return 0; }
void OnBar() {}
void OnDeinit() {}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR with non-ASCII comment failed: %v", err)
	}
	if ir == nil {
		t.Fatal("CompileToIR returned nil IR")
	}
	if ir.Version != "mql4" {
		t.Errorf("version = %s, want mql4", ir.Version)
	}
}
