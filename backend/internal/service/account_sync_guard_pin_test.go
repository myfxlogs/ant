package service

// TRADE-RECORDS-DUP-1 T2: both ledger sync loops must call the single-source
// guard. SyncableClosedTrade itself is pinned by the mthub unit matrix; this
// pin kills the regression of someone re-adding an unguarded append loop.
// (Both services depend on concrete types, so the loop wiring is what a
// source pin can lock without a live broker.)

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestTradeLedgerGuardWiredAtBothSyncSites(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	svcDir := thisFile[:strings.LastIndex(thisFile, "/")]
	sites := map[string]string{
		"account_sync_service.go": svcDir + "/account_sync_service.go",
		"mthub_service_orders.go": svcDir + "/../connect/system/mthub_service_orders.go",
	}
	for name, path := range sites {
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			code := string(src)
			if !strings.Contains(code, "!r.SyncableClosedTrade()") {
				t.Errorf("%s: ledger write loop lost the SyncableClosedTrade guard — phantom rows bleed again", name)
			}
		})
	}
}

// TRADE-RECORDS-DUP-1 third-path pin: the reconciliation ghost-import writes
// trade_records through ImportBrokerOrder, whose own IsZero() gate never
// fired for epoch close times (Unix(0,0) is not the Go zero time). The write
// must route through the single-source guard.
func TestGhostImportLedgerWriteUsesSharedGuard(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	svcDir := thisFile[:strings.LastIndex(thisFile, "/")]
	src, err := os.ReadFile(svcDir + "/../mthub/service_orders_import.go")
	if err != nil {
		t.Fatalf("read service_orders_import.go: %v", err)
	}
	if !strings.Contains(string(src), "br.SyncableClosedTrade()") {
		t.Error("ImportBrokerOrder trade_records write lost the SyncableClosedTrade guard — epoch ghost rows bleed via reconciliation")
	}
}
