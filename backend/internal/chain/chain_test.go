package chain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTronGridClient_GetBlockEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/contracts/TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t/events" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("event_name") != "Transfer" {
			t.Errorf("expected event_name=Transfer, got %s", q.Get("event_name"))
		}
		if q.Get("block_number") != "12345" {
			t.Errorf("expected block_number=12345, got %s", q.Get("block_number"))
		}

		resp := map[string]any{
			"data": []map[string]any{
				{
					"transaction_id": "abc123",
					"block_number":   12345,
					"event_name":     "Transfer",
					"result": map[string]string{
						"0": "4152d6c2c1e6e6b8a3b1a5c7d8e9f0a1b2c3d4e5f6",
						"1": "4158e1b0a2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8",
						"2": "1000000",
					},
					"_unconfirmed": false,
				},
			},
			"success": true,
			"meta":    map[string]any{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewTronGridClient("test-key")
	client.baseURL = srv.URL

	events, err := client.GetBlockEvents(t.Context(), "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", 12345)
	if err != nil {
		t.Fatalf("GetBlockEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].TxHash != "abc123" {
		t.Errorf("expected tx_hash=abc123, got %s", events[0].TxHash)
	}
	if events[0].AmountString != "1.000000" {
		t.Errorf("expected amount=1.000000, got %s", events[0].AmountString)
	}
	if !events[0].Confirmed {
		t.Error("expected confirmed=true")
	}
}

func TestTronGridClient_GetBlockEvents_Pagination(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		q := r.URL.Query()
		if page == 1 {
			resp := map[string]any{
				"data": []map[string]any{
					{
						"transaction_id": "tx1",
						"block_number":   100,
						"event_name":     "Transfer",
						"result":         map[string]string{"0": "", "1": "", "2": "1000000"},
						"_unconfirmed":   false,
					},
				},
				"meta":    map[string]any{"fingerprint": "fp123"},
				"success": true,
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			if q.Get("fingerprint") != "fp123" {
				t.Errorf("expected fingerprint=fp123, got %s", q.Get("fingerprint"))
			}
			resp := map[string]any{
				"data": []map[string]any{
					{
						"transaction_id": "tx2",
						"block_number":   100,
						"event_name":     "Transfer",
						"result":         map[string]string{"0": "", "1": "", "2": "2000000"},
						"_unconfirmed":   false,
					},
				},
				"meta":    map[string]any{},
				"success": true,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	client := NewTronGridClient("")
	client.baseURL = srv.URL

	events, err := client.GetBlockEvents(t.Context(), "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", 100)
	if err != nil {
		t.Fatalf("GetBlockEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events across pages, got %d", len(events))
	}
	if events[0].TxHash != "tx1" || events[1].TxHash != "tx2" {
		t.Errorf("unexpected tx hashes: %s, %s", events[0].TxHash, events[1].TxHash)
	}
}

func TestTronGridClient_GetLatestBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wallet/getnowblock" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]any{
			"blockID": "0001",
			"block_header": map[string]any{
				"raw_data": map[string]any{
					"number":    99999,
					"timestamp": 1700000000000,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewTronGridClient("")
	client.baseURL = srv.URL

	block, err := client.GetLatestBlock(t.Context())
	if err != nil {
		t.Fatalf("GetLatestBlock: %v", err)
	}
	if block != 99999 {
		t.Errorf("expected block 99999, got %d", block)
	}
}

func TestTronScanClient_VerifyTransaction(t *testing.T) {
	// TRC20 transfer: top-level toAddress is the USDT contract, NOT the recipient.
	// The actual recipient is in contractData.to_address.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"hash":         "abc123",
			"block":        100,
			"confirmed":    true,
			"toAddress":    "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", // USDT contract
			"ownerAddress": "TFromAddr",
			"contractType": 1,
			"contractData": map[string]any{
				"to_address":    "TUserDepositAddr", // actual recipient
				"owner_address": "TFromAddr",
				"amount":        1000000,
			},
			"revert": false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewTronScanClient("")
	client.baseURL = srv.URL

	// Should verify against contractData.to_address, not top-level toAddress.
	verified, err := client.VerifyTransaction(t.Context(), "abc123", "TUserDepositAddr")
	if err != nil {
		t.Fatalf("VerifyTransaction: %v", err)
	}
	if !verified {
		t.Error("expected verified=true for recipient in contractData.to_address")
	}

	// Should NOT verify against the contract address (top-level toAddress).
	verified, err = client.VerifyTransaction(t.Context(), "abc123", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		t.Fatalf("VerifyTransaction: %v", err)
	}
	if verified {
		t.Error("expected verified=false when checking against contract address, not recipient")
	}

	// Wrong address should fail.
	verified, err = client.VerifyTransaction(t.Context(), "abc123", "WrongAddr")
	if err != nil {
		t.Fatalf("VerifyTransaction: %v", err)
	}
	if verified {
		t.Error("expected verified=false for wrong address")
	}
}

func TestTronScanClient_VerifyTransaction_NonContract(t *testing.T) {
	// Non-TRC20 transfer (contractType != 1): top-level toAddress is the recipient.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"hash":         "def456",
			"block":        200,
			"confirmed":    true,
			"toAddress":    "TDirectRecipient",
			"ownerAddress": "TFromAddr",
			"contractType": 0,
			"contractData": map[string]any{
				"to_address":    "",
				"owner_address": "TFromAddr",
				"amount":        500000,
			},
			"revert": false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewTronScanClient("")
	client.baseURL = srv.URL

	verified, err := client.VerifyTransaction(t.Context(), "def456", "TDirectRecipient")
	if err != nil {
		t.Fatalf("VerifyTransaction: %v", err)
	}
	if !verified {
		t.Error("expected verified=true for non-contract tx using top-level toAddress")
	}
}

func TestConvertSunToUSDT(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"1000000", "1.000000"},
		{"1500000", "1.500000"},
		{"0", "0.000000"},
		{"100", "0.000100"},
		{"", "0"},
	}
	for _, tt := range tests {
		got := convertSunToUSDT(tt.input)
		if got != tt.expected {
			t.Errorf("convertSunToUSDT(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestTronGridClient_GetTRC20Balance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/accounts/TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// TronGrid returns trc20 as array of {contract_address: balance_string} maps.
		resp := map[string]any{
			"data": []map[string]any{
				{
					"balance": 3577033,
					"trc20": []map[string]any{
						{"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t": "1000000"},
						{"TLa2f6VPqDgRE67v1736s7bJ8Ray5wYjU7": "619313"},
					},
				},
			},
			"success": true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewTronGridClient("test-key")
	client.baseURL = srv.URL

	balance, err := client.GetTRC20Balance(t.Context(), "TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh")
	if err != nil {
		t.Fatalf("GetTRC20Balance: %v", err)
	}
	if balance != "1.000000" {
		t.Errorf("expected balance=1.000000, got %s", balance)
	}
}

func TestTronGridClient_GetTRC20Balance_NoUSDT(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{
					"trc20": []map[string]any{
						{"TLa2f6VPqDgRE67v1736s7bJ8Ray5wYjU7": "619313"},
					},
				},
			},
			"success": true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewTronGridClient("")
	client.baseURL = srv.URL

	balance, err := client.GetTRC20Balance(t.Context(), "TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh")
	if err != nil {
		t.Fatalf("GetTRC20Balance: %v", err)
	}
	if balance != "0" {
		t.Errorf("expected balance=0 for no USDT, got %s", balance)
	}
}

func TestTronGridClient_GetTRC20Balance_EmptyAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":    []map[string]any{},
			"success": true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewTronGridClient("")
	client.baseURL = srv.URL

	balance, err := client.GetTRC20Balance(t.Context(), "TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh")
	if err != nil {
		t.Fatalf("GetTRC20Balance: %v", err)
	}
	if balance != "0" {
		t.Errorf("expected balance=0 for empty account, got %s", balance)
	}
}

func TestTronGridClient_GetTRC20Balance_APIFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":    []map[string]any{},
			"success": false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewTronGridClient("")
	client.baseURL = srv.URL

	_, err := client.GetTRC20Balance(t.Context(), "TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh")
	if err == nil {
		t.Fatal("expected error for success=false, got nil")
	}
}
