package risksvc

import (
	"testing"
)

func TestPlatformAggregator_NetExposure(t *testing.T) {
	a := NewPlatformAggregator()

	// Account 1: long 0.1 EURUSD
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: 0.1, Notional: 108500})
	// Account 2: short 0.1 EURUSD
	a.UpdatePosition("acc-2", &AggregatorPosition{Canonical: "EURUSD", NetVolume: -0.1, Notional: -108500})

	exposure := a.Recalculate(nil)

	net := exposure.NetExposureBySymbol["EURUSD"]
	if net != 0 {
		t.Fatalf("long 0.1 + short 0.1 = net 0, got %.4f", net)
	}
	if exposure.TotalNetExposure != 0 {
		t.Fatalf("total net exposure should be 0, got %.4f", exposure.TotalNetExposure)
	}
	if exposure.TotalGrossExposure != 217000 {
		t.Fatalf("total gross should be 217000, got %.4f", exposure.TotalGrossExposure)
	}
	t.Logf("NetExposure: EURUSD=%.4f gross=%.0f net=%.0f accounts=%d",
		net, exposure.TotalGrossExposure, exposure.TotalNetExposure, exposure.AccountCount)
}

func TestPlatformAggregator_MultipleSymbols(t *testing.T) {
	a := NewPlatformAggregator()

	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: 0.2, Notional: 217000, Margin: 2170})
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "GBPUSD", NetVolume: -0.1, Notional: -126500, Margin: 1265})
	a.UpdatePosition("acc-2", &AggregatorPosition{Canonical: "EURUSD", NetVolume: -0.1, Notional: -108500, Margin: 1085})

	exposure := a.Recalculate(nil)

	if exposure.AccountCount != 2 {
		t.Fatalf("want 2 accounts, got %d", exposure.AccountCount)
	}
	if exposure.TotalMarginUsed != 4520 {
		t.Fatalf("want 4520 margin, got %.0f", exposure.TotalMarginUsed)
	}
	eurNet := exposure.NetExposureBySymbol["EURUSD"]
	if eurNet != 0.1 {
		t.Fatalf("EURUSD net should be 0.1, got %.4f", eurNet)
	}
	gbpNet := exposure.NetExposureBySymbol["GBPUSD"]
	if gbpNet != -0.1 {
		t.Fatalf("GBPUSD net should be -0.1, got %.4f", gbpNet)
	}
}

func TestPlatformAggregator_ClearAccount(t *testing.T) {
	a := NewPlatformAggregator()
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: 0.1, Notional: 108500})
	a.ClearAccount("acc-1")

	exposure := a.Recalculate(nil)
	if exposure.AccountCount != 0 {
		t.Fatalf("want 0 accounts after clear, got %d", exposure.AccountCount)
	}
	if exposure.TotalGrossExposure != 0 {
		t.Fatalf("exposure should be 0 after clear")
	}
}

func TestPlatformAggregator_BrokerLimits(t *testing.T) {
	a := NewPlatformAggregator()
	a.UpdatePosition("acc-1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: 0.1, Notional: 108500, Margin: 1085})

	limits := map[string]float64{"mt5": 10000}
	exposure := a.Recalculate(limits)

	t.Logf("BrokerLimitUsage: %v", exposure.BrokerLimitUsage)
}
