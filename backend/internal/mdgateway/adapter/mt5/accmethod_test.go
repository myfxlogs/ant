// accmethod_test.go — MT5-ACCMETHOD-ADAPTER-1 (2026-09-18).
//
// accMethodToString maps mtapi AccMethod to the margin-mode string; unknown
// values (Default/undefined) map to "" so downstream consumers fail closed.
//
// Adversarial (M1): make accMethodToString return "" unconditionally → all
// value assertions RED.
package mt5

import (
	"testing"

	pb "alphaforge/mt5"
)

func TestAccMethodToString(t *testing.T) {
	cases := []struct {
		name string
		in   pb.AccMethod
		want string
	}{
		{"Netting", pb.AccMethod_AccMethod_Netting, "netting"},
		{"Hedging", pb.AccMethod_AccMethod_Hedging, "hedging"},
		{"Default is unknown", pb.AccMethod_AccMethod_Default, ""},
		{"undefined value 99 is unknown", pb.AccMethod(99), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := accMethodToString(tc.in); got != tc.want {
				t.Fatalf("accMethodToString(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
