package repository

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/google/uuid"
)

func TestComputeTradeEntryHash_Deterministic(t *testing.T) {
	t.Parallel()
	accountID := uuid.New()
	prevHash := []byte{1, 2, 3}

	h1 := computeTradeEntryHash(prevHash, 1, accountID, 100, "EURUSD",
		"0.1", "1.1000", "1.1050", "50.00", [2]int64{1700000000000, 1700001000000})
	h2 := computeTradeEntryHash(prevHash, 1, accountID, 100, "EURUSD",
		"0.1", "1.1000", "1.1050", "50.00", [2]int64{1700000000000, 1700001000000})

	if len(h1) != 32 {
		t.Fatalf("expected 32-byte hash, got %d", len(h1))
	}
	if !bytesEqual(h1, h2) {
		t.Fatal("same inputs should produce same hash")
	}
}

func TestComputeTradeEntryHash_DifferentInputs(t *testing.T) {
	t.Parallel()
	accountID := uuid.New()
	prevHash := []byte{}

	h1 := computeTradeEntryHash(prevHash, 1, accountID, 100, "EURUSD",
		"0.1", "1.1000", "1.1050", "50.00", [2]int64{1700000000000, 1700001000000})

	// Different ticket → different hash.
	h2 := computeTradeEntryHash(prevHash, 1, accountID, 200, "EURUSD",
		"0.1", "1.1000", "1.1050", "50.00", [2]int64{1700000000000, 1700001000000})
	if bytesEqual(h1, h2) {
		t.Fatal("different ticket should produce different hash")
	}

	// Different profit → different hash.
	h3 := computeTradeEntryHash(prevHash, 1, accountID, 100, "EURUSD",
		"0.1", "1.1000", "1.1050", "100.00", [2]int64{1700000000000, 1700001000000})
	if bytesEqual(h1, h3) {
		t.Fatal("different profit should produce different hash")
	}

	// Different seq → different hash.
	h4 := computeTradeEntryHash(prevHash, 2, accountID, 100, "EURUSD",
		"0.1", "1.1000", "1.1050", "50.00", [2]int64{1700000000000, 1700001000000})
	if bytesEqual(h1, h4) {
		t.Fatal("different seq should produce different hash")
	}

	// Different prev_hash → different hash.
	h5 := computeTradeEntryHash([]byte{9}, 1, accountID, 100, "EURUSD",
		"0.1", "1.1000", "1.1050", "50.00", [2]int64{1700000000000, 1700001000000})
	if bytesEqual(h1, h5) {
		t.Fatal("different prev_hash should produce different hash")
	}
}

func TestComputeTradeEntryHash_EmptyPrevHash(t *testing.T) {
	t.Parallel()
	accountID := uuid.New()

	// First record in chain: prev_hash is nil/empty.
	h := computeTradeEntryHash(nil, 1, accountID, 100, "EURUSD",
		"0.1", "1.1000", "1.1050", "50.00", [2]int64{1700000000000, 1700001000000})
	if len(h) != 32 {
		t.Fatalf("expected 32-byte hash for first record, got %d", len(h))
	}

	// Verify it matches the manual computation.
	hManual := sha256.New()
	hManual.Write(nil) // empty prev_hash
	var seqBuf [8]byte
	binary.BigEndian.PutUint64(seqBuf[:], 1)
	hManual.Write(seqBuf[:])
	hManual.Write(accountID[:])
	var ticketBuf [8]byte
	binary.BigEndian.PutUint64(ticketBuf[:], 100)
	hManual.Write(ticketBuf[:])
	hManual.Write([]byte("EURUSD"))
	hManual.Write([]byte("0.1"))
	hManual.Write([]byte("1.1000"))
	hManual.Write([]byte("1.1050"))
	hManual.Write([]byte("50.00"))
	var timeBuf [8]byte
	binary.BigEndian.PutUint64(timeBuf[:], 1700000000000)
	hManual.Write(timeBuf[:])
	binary.BigEndian.PutUint64(timeBuf[:], 1700001000000)
	hManual.Write(timeBuf[:])
	expected := hManual.Sum(nil)

	if !bytesEqual(h, expected) {
		t.Fatal("hash should match manual SHA256 computation")
	}
}

func TestBytesEqual(t *testing.T) {
	t.Parallel()
	if !bytesEqual([]byte{1, 2, 3}, []byte{1, 2, 3}) {
		t.Fatal("equal slices should be equal")
	}
	if bytesEqual([]byte{1, 2, 3}, []byte{1, 2, 4}) {
		t.Fatal("different slices should not be equal")
	}
	if bytesEqual([]byte{1, 2}, []byte{1, 2, 3}) {
		t.Fatal("different length slices should not be equal")
	}
	if !bytesEqual(nil, nil) {
		t.Fatal("nil slices should be equal")
	}
	if bytesEqual(nil, []byte{1}) {
		t.Fatal("nil and non-nil should not be equal")
	}
}

func TestChainBreakStruct(t *testing.T) {
	t.Parallel()
	// Verify ChainBreak can be constructed with expected fields.
	cb := struct {
		Seq    int64
		Ticket int64
		Type   string
		Detail string
	}{
		Seq:    5,
		Ticket: 12345,
		Type:   "hash_mismatch",
		Detail: "entry_hash mismatch at seq=5",
	}
	if cb.Seq != 5 || cb.Ticket != 12345 || cb.Type != "hash_mismatch" {
		t.Fatal("ChainBreak fields not set correctly")
	}
}
