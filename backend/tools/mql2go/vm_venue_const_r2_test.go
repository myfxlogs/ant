package mql2go

// VM-LIVE-VENUE-R2 T4: the trade-enum builtins read the broker SymbolInfo
// verbatim (including -1 = unknown), and MODE_TRADEALLOWED is two-axis —
// symbol "may open" (1=long_only/2=short_only/4=full) ∧ account
// IsTradeAllowed; 0=disabled / 3=close_only / -1=unknown fail closed.
// Mutations: M2 (case 26 → constant 0) → T4a/T4c RED;
// M3 (TRADEALLOWED → pure account flag) → T4b (T∧-1) cell RED.

import (
	"testing"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// venueConstBroker serves a caller-set sdk.SymbolInfo so trade-enum
// passthrough can be pinned with values distinct from the old hardcodes.
type venueConstBroker struct {
	sdk.Broker
	info sdk.SymbolInfo
}

func (b *venueConstBroker) SymbolInfo(string) (sdk.SymbolInfo, error) { return b.info, nil }

func newVenueConstVM(t *testing.T, info sdk.SymbolInfo, isTradeAllowed bool) *VM {
	t.Helper()
	vm := newSymbolEnumVM(t)
	vm.ctx.(*symbolEnumTestContext).broker = &venueConstBroker{info: info}
	vm.ctx.(*symbolEnumTestContext).isTradeAllowed = isTradeAllowed
	return vm
}

// T4a: SymbolInfoInteger 22/26/27 read info.TradeMode/FreezeLevel/
// TradeExemode verbatim — broker facts, model values, and the -1 unknown
// sentinel all surface honestly.
func TestVMVenueConstR2_SymbolInfoIntegerTradeEnumsVerbatim(t *testing.T) {
	cases := []struct {
		name                       string
		tradeMode, freeze, exemode int32
	}{
		{"brokerFacts", 1, 5, 3},
		{"unknownSentinel", -1, -1, -1},
		{"disabledTrueZero", 0, 0, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := newVenueConstVM(t, sdk.SymbolInfo{
				TradeMode: tc.tradeMode, FreezeLevel: tc.freeze, TradeExemode: tc.exemode,
			}, true)
			for _, c := range []struct {
				prop int32
				want int32
			}{
				{22, tc.tradeMode},
				{26, tc.freeze},
				{27, tc.exemode},
			} {
				v, err := builtinSymbolInfoInteger(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(c.prop)})
				if err != nil {
					t.Fatalf("prop %d: err = %v, want nil", c.prop, err)
				}
				if v.ToInt() != c.want {
					t.Errorf("prop %d = %d, want %d (verbatim passthrough)", c.prop, v.ToInt(), c.want)
				}
			}
		})
	}
}

// T4b: MODE_TRADEALLOWED two-axis truth table (VM-LIVE-VENUE-R2).
func TestVMVenueConstR2_ModeTradeAllowedTwoAxis(t *testing.T) {
	cases := []struct {
		name    string
		allowed bool
		mode    int32
		want    string
	}{
		{"allowed_full", true, 4, "1"},
		{"allowed_longonly", true, 1, "1"},
		{"allowed_unknown", true, -1, "0"},
		{"allowed_disabled", true, 0, "0"},
		{"allowed_closeonly", true, 3, "0"},
		{"denied_full", false, 4, "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := newVenueConstVM(t, sdk.SymbolInfo{TradeMode: tc.mode}, tc.allowed)
			v, err := builtinMarketInfo(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(22)})
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got := v.ToDecimal().String(); got != tc.want {
				t.Errorf("TRADEALLOWED(allowed=%v, mode=%d) = %s, want %s", tc.allowed, tc.mode, got, tc.want)
			}
		})
	}
}

// T4c: MarketInfo MODE_FREEZELEVEL reads info.FreezeLevel verbatim —
// backtest model 0 stays 0, broker fact and -1 sentinel surface as-is.
func TestVMVenueConstR2_MarketInfoFreezeLevelVerbatim(t *testing.T) {
	cases := []struct {
		name   string
		freeze int32
		want   string
	}{
		{"modelZero", 0, "0"},
		{"brokerFact", 20, "20"},
		{"unknownSentinel", -1, "-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := newVenueConstVM(t, sdk.SymbolInfo{FreezeLevel: tc.freeze}, true)
			v, err := builtinMarketInfo(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(32)})
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got := v.ToDecimal().String(); got != tc.want {
				t.Errorf("MODE_FREEZELEVEL = %s, want %s (verbatim passthrough)", got, tc.want)
			}
		})
	}
}
