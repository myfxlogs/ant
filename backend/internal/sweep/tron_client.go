// Package sweep implements the fund consolidation (sweep) pipeline:
// build → cold-sign → broadcast → confirm, with crash recovery and double-spend prevention.
// ADR-0026 §2.7, §10.4, Q3.
package sweep

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// TronClient wraps gotron-sdk GrpcClient for sweep operations.
// It provides transaction building (no signing), broadcast, confirmation checking,
// and outgoing TRC20 transfer detection for double-spend prevention.
type TronClient struct {
	grpc *client.GrpcClient
}

// NewTronClient creates a TronClient connected to the given TRON gRPC endpoint.
// apiKey is optional (for TronGrid paid tier).
func NewTronClient(endpoint, apiKey string) (*TronClient, error) {
	c := client.NewGrpcClient(endpoint)
	if err := c.Start(grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		return nil, fmt.Errorf("sweep tron client: start: %w", err)
	}
	if apiKey != "" {
		_ = c.SetAPIKey(apiKey)
	}
	return &TronClient{grpc: c}, nil
}

// Close stops the underlying gRPC connection.
func (c *TronClient) Close() {
	c.grpc.Stop()
}

// BuildTRC20Transfer builds an unsigned TRC20 transfer transaction.
// Returns the raw transaction bytes and the expected txid (hex).
// No signing is performed — the online server is watch-only (R1).
func (c *TronClient) BuildTRC20Transfer(ctx context.Context, from, to, contract string, amount *big.Int, feeLimit int64) ([]byte, string, error) {
	ext, err := c.grpc.TRC20SendCtx(ctx, from, to, contract, amount, feeLimit)
	if err != nil {
		return nil, "", fmt.Errorf("sweep tron client: build trc20 transfer: %w", err)
	}
	if ext.Transaction == nil || ext.Transaction.RawData == nil {
		return nil, "", fmt.Errorf("sweep tron client: build trc20 transfer: missing raw data")
	}
	return marshalTx(ext.Transaction), txIDFromExt(ext), nil
}

// BuildDelegateResource builds an unsigned DelegateResource transaction.
// energyAccount delegates ENERGY to the deposit address for the sweep transfer.
func (c *TronClient) BuildDelegateResource(ctx context.Context, from, to string, energyAmount int64) ([]byte, string, error) {
	ext, err := c.grpc.DelegateResourceCtx(ctx, from, to, core.ResourceCode_ENERGY, energyAmount, false, 0)
	if err != nil {
		return nil, "", fmt.Errorf("sweep tron client: build delegate: %w", err)
	}
	if ext.Transaction == nil || ext.Transaction.RawData == nil {
		return nil, "", fmt.Errorf("sweep tron client: build delegate: missing raw data")
	}
	return marshalTx(ext.Transaction), txIDFromExt(ext), nil
}

// BuildUndelegateResource builds an unsigned UnDelegateResource transaction.
// Energy is reclaimed from the deposit address back to the energy account.
func (c *TronClient) BuildUndelegateResource(ctx context.Context, from, to string, energyAmount int64) ([]byte, string, error) {
	ext, err := c.grpc.UnDelegateResourceCtx(ctx, from, to, core.ResourceCode_ENERGY, energyAmount)
	if err != nil {
		return nil, "", fmt.Errorf("sweep tron client: build undelegate: %w", err)
	}
	if ext.Transaction == nil || ext.Transaction.RawData == nil {
		return nil, "", fmt.Errorf("sweep tron client: build undelegate: missing raw data")
	}
	return marshalTx(ext.Transaction), txIDFromExt(ext), nil
}

// BroadcastSignedTx broadcasts a pre-signed transaction.
// The signedTxData is a serialized core.Transaction.
// Returns the txid and any broadcast error.
func (c *TronClient) BroadcastSignedTx(ctx context.Context, signedTxData []byte) (string, error) {
	tx := &core.Transaction{}
	if err := proto.Unmarshal(signedTxData, tx); err != nil {
		return "", fmt.Errorf("sweep tron client: unmarshal signed tx: %w", err)
	}

	result, err := c.grpc.BroadcastCtx(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("sweep tron client: broadcast: %w", err)
	}
	if result == nil {
		return "", fmt.Errorf("sweep tron client: broadcast: empty response")
	}
	if result.Code != 0 {
		return "", fmt.Errorf("sweep tron client: broadcast rejected: %s", string(result.GetMessage()))
	}

	txid, err := computeTxID(tx)
	if err != nil {
		return "", fmt.Errorf("sweep tron client: compute txid: %w", err)
	}
	return txid, nil
}

// GetTransactionInfo queries the chain for transaction confirmation status.
// Returns (confirmed, success, energyUsed, error).
// If the transaction is not found on chain, confirmed=false with nil error.
func (c *TronClient) GetTransactionInfo(ctx context.Context, txid string) (confirmed, success bool, energyUsed int64, err error) {
	info, err := c.grpc.GetTransactionInfoByIDCtx(ctx, txid)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, false, 0, nil
		}
		return false, false, 0, fmt.Errorf("sweep tron client: get tx info: %w", err)
	}
	if info == nil || info.GetBlockNumber() == 0 {
		return false, false, 0, nil
	}
	energyUsed = 0
	if info.GetReceipt() != nil {
		energyUsed = info.GetReceipt().GetEnergyUsageTotal()
	}
	// 0 is the success code from the generated TransactionInfoCode enum.
	success = info.GetResult() == 0
	return true, success, energyUsed, nil
}

// SetTxExpiry sets the expiration timestamp on a transaction's raw data.
// TRON transactions expire after a certain time; we set ~24h for crash recovery window (Q3).
func SetTxExpiry(rawTxData []byte, expiryMs int64) ([]byte, error) {
	tx := &core.Transaction{}
	if err := proto.Unmarshal(rawTxData, tx); err != nil {
		return nil, fmt.Errorf("sweep tron client: set expiry: unmarshal: %w", err)
	}
	if tx.RawData == nil {
		return nil, fmt.Errorf("sweep tron client: set expiry: missing raw data")
	}
	tx.RawData.Expiration = expiryMs
	out, err := proto.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("sweep tron client: set expiry: marshal: %w", err)
	}
	return out, nil
}

// WaitForConfirmation polls GetTransactionInfoByID until the transaction is confirmed
// or the context is cancelled. Poll interval defaults to 3s.
func (c *TronClient) WaitForConfirmation(ctx context.Context, txid string, pollInterval time.Duration) (bool, int64, error) {
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, 0, ctx.Err()
		case <-ticker.C:
			confirmed, success, energyUsed, err := c.GetTransactionInfo(ctx, txid)
			if err != nil {
				return false, 0, err
			}
			if confirmed {
				return success, energyUsed, nil
			}
		}
	}
}

func marshalTx(tx *core.Transaction) []byte {
	data, err := proto.Marshal(tx)
	if err != nil {
		return nil
	}
	return data
}

func txIDFromExt(ext *api.TransactionExtention) string {
	if len(ext.Txid) > 0 {
		return common.BytesToHexString(ext.Txid)
	}
	id, _ := computeTxID(ext.Transaction)
	return id
}

func computeTxID(tx *core.Transaction) (string, error) {
	if tx == nil || tx.RawData == nil {
		return "", fmt.Errorf("missing raw data")
	}
	rawData, err := proto.Marshal(tx.RawData)
	if err != nil {
		return "", fmt.Errorf("marshal raw data: %w", err)
	}
	h := sha256Sum(rawData)
	return common.BytesToHexString(h), nil
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
