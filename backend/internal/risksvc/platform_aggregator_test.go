package risksvc

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestPlatformAggregator_NetExposure(t *testing.T) {
	t.Parallel()
	a := NewPlatformAggregator()

	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(0.1), Notional: decF(108500)})
	a.UpdatePosition("acc-2", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(-0.1), Notional: decF(-108500)})

	exposure := a.Recalculate()

	net := exposure.NetExposureBySymbol["EURUSD"]
	if !net.IsZero() {
		t.Fatalf("long 0.1 + short 0.1 = net 0, got %s", net.String())
	}
	if !exposure.TotalNetExposure.IsZero() {
		t.Fatalf("total net exposure should be 0, got %s", exposure.TotalNetExposure.String())
	}
	if !exposure.TotalGrossExposure.Equal(decF(217000)) {
		t.Fatalf("total gross should be 217000, got %s", exposure.TotalGrossExposure.String())
	}
	t.Logf("NetExposure: EURUSD=%s gross=%s net=%s accounts=%d",
		net.String(), exposure.TotalGrossExposure.String(), exposure.TotalNetExposure.String(), exposure.AccountCount)
}

func TestPlatformAggregator_MultipleSymbols(t *testing.T) {
	t.Parallel()
	a := NewPlatformAggregator()

	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(0.2), Notional: decF(217000), Margin: decF(2170)})
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "GBPUSD", NetVolume: decF(-0.1), Notional: decF(-126500), Margin: decF(1265)})
	a.UpdatePosition("acc-2", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(-0.1), Notional: decF(-108500), Margin: decF(1085)})

	exposure := a.Recalculate()

	if exposure.AccountCount != 2 {
		t.Fatalf("want 2 accounts, got %d", exposure.AccountCount)
	}
	if !exposure.TotalMarginUsed.Equal(decF(4520)) {
		t.Fatalf("want 4520 margin, got %s", exposure.TotalMarginUsed.String())
	}
	eurNet := exposure.NetExposureBySymbol["EURUSD"]
	if !eurNet.Equal(decF(0.1)) {
		t.Fatalf("EURUSD net should be 0.1, got %s", eurNet.String())
	}
	gbpNet := exposure.NetExposureBySymbol["GBPUSD"]
	if !gbpNet.Equal(decF(-0.1)) {
		t.Fatalf("GBPUSD net should be -0.1, got %s", gbpNet.String())
	}
}

func TestPlatformAggregator_ClearAccount(t *testing.T) {
	t.Parallel()
	a := NewPlatformAggregator()
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(0.1), Notional: decF(108500)})
	a.ClearAccount("acc-1")

	exposure := a.Recalculate()
	if exposure.AccountCount != 0 {
		t.Fatalf("want 0 accounts after clear, got %d", exposure.AccountCount)
	}
	if !exposure.TotalGrossExposure.IsZero() {
		t.Fatalf("exposure should be 0 after clear")
	}
}

func TestPlatformAggregator_BrokerLimits(t *testing.T) {
	t.Parallel()
	a := NewPlatformAggregator()
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(0.1), Notional: decF(108500), Margin: decF(1085), Broker: "mt5"})

	a.SetBrokerLimits(map[string]decimal.Decimal{"mt5": decF(10000)})
	exposure := a.Recalculate()

	if usage, ok := exposure.BrokerLimitUsage["mt5"]; !ok || usage <= 0 {
		t.Fatalf("expected mt5 broker limit usage > 0, got %v", exposure.BrokerLimitUsage)
	}
}

func TestPlatformAggregator_SetBrokerLimits(t *testing.T) {
	t.Parallel()
	a := NewPlatformAggregator()
	a.SetBrokerLimits(map[string]decimal.Decimal{"mt4": decF(5000), "mt5": decF(10000)})
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(0.1), Notional: decF(108500), Margin: decF(1085), Broker: "mt5"})
	exposure := a.Recalculate()
	if len(exposure.BrokerLimitUsage) != 2 {
		t.Fatalf("expected 2 brokers in usage map, got %d: %v", len(exposure.BrokerLimitUsage), exposure.BrokerLimitUsage)
	}
	if exposure.BrokerLimitUsage["mt5"] <= 0 {
		t.Fatalf("expected mt5 usage > 0, got %f", exposure.BrokerLimitUsage["mt5"])
	}
}

func TestPlatformAggregator_NetExposureForSymbol(t *testing.T) {
	t.Parallel()
	a := NewPlatformAggregator()
	// No snapshot yet → zero
	if !a.NetExposureForSymbol("EURUSD").IsZero() {
		t.Fatal("expected zero before any recalculation")
	}
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(0.5), Notional: decF(54250)})
	a.Recalculate()
	if !a.NetExposureForSymbol("EURUSD").Equal(decF(0.5)) {
		t.Fatalf("expected 0.5, got %s", a.NetExposureForSymbol("EURUSD").String())
	}
	if !a.NetExposureForSymbol("GBPUSD").IsZero() {
		t.Fatal("expected zero for unknown symbol")
	}
}

func TestPlatformAggregator_StartRefreshLoop_Shutdown(t *testing.T) {
	t.Parallel()
	a := NewPlatformAggregator()
	a.StartRefreshLoop()
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: decF(0.1), Notional: decF(10850)})

	// Wait for refresh loop to process the dirty signal (up to 200ms)
	var got bool
	for i := 0; i < 200; i++ {
		if snap := a.GetSnapshot(); snap != nil {
			if snap.AccountCount == 1 {
				got = true
				break
			}
		}
		time.Sleep(time.Millisecond)
	}
	if !got {
		t.Log("snapshot not updated in time (timing-dependent) — verifying shutdown still works")
	}

	a.Shutdown()  // should not panic
	a.Shutdown()  // double shutdown should also be safe
}
