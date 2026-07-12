package indicator

import (
	"testing"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
)

func makeBars(prices []float64) []mdtick.Bar {
	bars := make([]mdtick.Bar, len(prices))
	for i, p := range prices {
		bars[i] = mdtick.Bar{
			Open:  decimal.NewFromFloat(p),
			High:  decimal.NewFromFloat(p * 1.01),
			Low:   decimal.NewFromFloat(p * 0.99),
			Close: decimal.NewFromFloat(p),
			Volume: 1000,
		}
	}
	return bars
}

func TestSMA(t *testing.T) {
	bars := makeBars([]float64{10, 12, 14, 16, 18, 20})
	r, err := Compute("SMA", bars, map[string]float64{"length": 3})
	if err != nil {
		t.Fatal(err)
	}
	sma := r.Lines["sma"]
	// First 2 bars: partial averages. Bar 2: (10+12+14)/3=12, Bar 3: (12+14+16)/3=14
	expected := []float64{10, 11, 12, 14, 16, 18}
	for i, e := range expected {
		f, _ := sma[i].Float64()
		if abs(f-e) > 0.001 {
			t.Errorf("SMA[%d] expected %.4f got %.4f", i, e, f)
		}
	}
}

func TestEMA(t *testing.T) {
	bars := makeBars([]float64{10, 12, 14, 16, 18, 20})
	r, err := Compute("EMA", bars, map[string]float64{"length": 3})
	if err != nil {
		t.Fatal(err)
	}
	emaVals := r.Lines["ema"]
	// Seed = SMA(10,12,14)/3 = 12. α=2/(3+1)=0.5
	// EMA[0]=seed=12, EMA[1]=12*0.5+12*0.5=12, EMA[2]=14*0.5+12*0.5=13
	// EMA[3]=16*0.5+13*0.5=14.5, EMA[4]=18*0.5+14.5*0.5=16.25, EMA[5]=20*0.5+16.25*0.5=18.125
	f0, _ := emaVals[0].Float64()
	if abs(f0-12) > 0.01 {
		t.Errorf("EMA[0] seed expected 12 got %.4f", f0)
	}
	f5, _ := emaVals[5].Float64()
	if abs(f5-18.125) > 0.1 {
		t.Errorf("EMA[5] expected 18.125 got %.4f", f5)
	}
}

func TestRSI(t *testing.T) {
	// Rising prices → RSI should be high (~100)
	prices := make([]float64, 20)
	for i := range prices {
		prices[i] = float64(10 + i) // 10, 11, 12, ..., 29
	}
	bars := makeBars(prices)
	r, err := Compute("RSI", bars, map[string]float64{"length": 14})
	if err != nil {
		t.Fatal(err)
	}
	rsi := r.Lines["rsi"]
	last, _ := rsi[len(rsi)-1].Float64()
	if last < 90 {
		t.Errorf("RSI for strictly rising prices expected > 90, got %.4f", last)
	}
}

func TestBOLL(t *testing.T) {
	bars := makeBars([]float64{10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50})
	r, err := Compute("BOLL", bars, map[string]float64{"length": 20, "mult": 2})
	if err != nil {
		t.Fatal(err)
	}
	middle, _ := r.Lines["middle"][len(bars)-1].Float64()
	upper, _ := r.Lines["upper"][len(bars)-1].Float64()
	lower, _ := r.Lines["lower"][len(bars)-1].Float64()
	if upper <= middle {
		t.Error("upper band should be above middle")
	}
	if middle <= lower {
		t.Error("middle should be above lower band")
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
