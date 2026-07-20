// Package hdwallet — sign_tron.go provides TRON transaction signing using
// a derived private key. OFFLINE USE ONLY (cmd/coldsign, cmd/coldsign-gui).
//
// This function is in the hdwallet package because it inherently handles
// private key material — the CI zero-private-key assertion excludes this
// package from scanning.
package hdwallet

import (
	"crypto/sha256"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// SignTronTransaction signs a raw TRON transaction (serialized core.Transaction)
// with the given private key. Returns the signed transaction bytes and txid.
//
// This is the core air-gapped signing operation (ADR-0026 §10.2, R1/R2).
// No network access is needed — signing is purely local.
// The txid = hex(sha256(raw_data_bytes)) is deterministic before signing.
func SignTronTransaction(rawTxData []byte, privKey *btcec.PrivateKey) (signedData []byte, txid string, err error) {
	tx := &core.Transaction{}
	if err := proto.Unmarshal(rawTxData, tx); err != nil {
		return nil, "", fmt.Errorf("hdwallet: sign: unmarshal raw tx: %w", err)
	}
	if tx.RawData == nil {
		return nil, "", fmt.Errorf("hdwallet: sign: missing raw data")
	}

	signedTx, err := transaction.SignTransaction(tx, privKey)
	if err != nil {
		return nil, "", fmt.Errorf("hdwallet: sign: sign transaction: %w", err)
	}

	signedData, err = proto.Marshal(signedTx)
	if err != nil {
		return nil, "", fmt.Errorf("hdwallet: sign: marshal signed tx: %w", err)
	}

	txid = computeTronTxID(signedTx)
	return signedData, txid, nil
}

// computeTronTxID computes the TRON transaction ID = hex(sha256(raw_data_bytes)).
func computeTronTxID(tx *core.Transaction) string {
	rawData, err := proto.Marshal(tx.RawData)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(rawData)
	return common.BytesToHexString(h[:])
}
