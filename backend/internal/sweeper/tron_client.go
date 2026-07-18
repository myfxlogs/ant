package sweeper

import (
	"context"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TronClient wraps gotron-sdk GrpcClient for TRC20 transfer + energy delegation.
// It connects to a TRON gRPC node (e.g. api.trongrid.io:50051) to sign and broadcast transactions.
type TronClient struct {
	conn     *client.GrpcClient
	apiKey   string
	nodeAddr string
}

// NewTronClient creates a new Tron gRPC client. Call Start() before use.
func NewTronClient(nodeAddr, apiKey string) *TronClient {
	return &TronClient{
		apiKey:   apiKey,
		nodeAddr: nodeAddr,
	}
}

// Start establishes the gRPC connection.
func (t *TronClient) Start() error {
	t.conn = client.NewGrpcClient(t.nodeAddr)
	if err := t.conn.Start(grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		return fmt.Errorf("tron client: start: %w", err)
	}
	if t.apiKey != "" {
		if err := t.conn.SetAPIKey(t.apiKey); err != nil {
			return fmt.Errorf("tron client: set api key: %w", err)
		}
	}
	return nil
}

// Close terminates the gRPC connection.
func (t *TronClient) Close() {
	if t.conn != nil {
		t.conn.Stop()
	}
}

// TransferTRC20 transfers USDT from fromAddr to toAddr.
// privKeyBytes is the raw 32-byte private key of the sender.
// contractAddress is the USDT TRC20 contract address.
// amountSun is the amount in smallest unit (10^-6 for USDT, e.g. 1000000 = 1 USDT).
// Returns the broadcast tx hash on success.
func (t *TronClient) TransferTRC20(ctx context.Context, fromAddr, toAddr, contractAddress string, privKeyBytes []byte, amountSun int64) (string, error) {
	if t.conn == nil {
		return "", fmt.Errorf("tron client: not started")
	}
	sk, _ := btcec.PrivKeyFromBytes(privKeyBytes)
	if sk == nil {
		return "", fmt.Errorf("tron client: invalid private key")
	}

	signerInst, err := signer.NewPrivateKeySignerFromBTCEC(sk)
	if err != nil {
		return "", fmt.Errorf("tron client: create signer: %w", err)
	}

	// Build TRC20 transfer(to, amount) call data.
	// transfer(address,uint256) selector = 0xa9059cbb
	// Parameters are padded to 32 bytes each.
	data, err := buildTRC20TransferData(toAddr, amountSun)
	if err != nil {
		return "", fmt.Errorf("tron client: build transfer data: %w", err)
	}

	// feeLimit: 30 TRX (in sun) — enough for energy + bandwidth.
	const feeLimit = 30_000_000

	txExt, err := t.conn.TriggerContractWithDataCtx(ctx, fromAddr, contractAddress, data, feeLimit, 0, "", 0)
	if err != nil {
		return "", fmt.Errorf("tron client: trigger contract: %w", err)
	}
	if txExt.GetResult() == nil || !txExt.GetResult().GetResult() {
		msg := ""
		if txExt.GetResult() != nil {
			msg = string(txExt.GetResult().GetMessage())
		}
		return "", fmt.Errorf("tron client: trigger result: %s", msg)
	}

	txHash := common.BytesToHexString(txExt.GetTxid())
	tx := txExt.GetTransaction()

	// Sign the transaction.
	tx, err = signerInst.Sign(tx)
	if err != nil {
		return "", fmt.Errorf("tron client: sign: %w", err)
	}

	// Broadcast.
	result, err := t.conn.BroadcastCtx(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("tron client: broadcast: %w", err)
	}
	if result.GetCode() != api.Return_SUCCESS {
		return "", fmt.Errorf("tron client: broadcast code: %s, msg: %s", result.GetCode(), string(result.GetMessage()))
	}

	return txHash, nil
}

// DelegateEnergy delegates energy from fromAddr to receiverAddr.
// privKeyBytes is the raw 32-byte private key of the delegator (hot wallet).
// energyAmount is the amount of energy to delegate (in units, not TRX).
// Returns the broadcast tx hash on success.
func (t *TronClient) DelegateEnergy(ctx context.Context, fromAddr, receiverAddr string, privKeyBytes []byte, energyAmount int64) (string, error) {
	if t.conn == nil {
		return "", fmt.Errorf("tron client: not started")
	}
	sk, _ := btcec.PrivKeyFromBytes(privKeyBytes)
	if sk == nil {
		return "", fmt.Errorf("tron client: invalid delegator key")
	}

	signerInst, err := signer.NewPrivateKeySignerFromBTCEC(sk)
	if err != nil {
		return "", fmt.Errorf("tron client: create signer: %w", err)
	}

	// RESOURCE_ENERGY = 0x01 in TRON ResourceCode.
	txExt, err := t.conn.DelegateResourceCtx(ctx, fromAddr, receiverAddr, core.ResourceCode_ENERGY, energyAmount, false, 0)
	if err != nil {
		return "", fmt.Errorf("tron client: delegate resource: %w", err)
	}
	if txExt.GetResult() == nil || !txExt.GetResult().GetResult() {
		msg := ""
		if txExt.GetResult() != nil {
			msg = string(txExt.GetResult().GetMessage())
		}
		return "", fmt.Errorf("tron client: delegate result: %s", msg)
	}

	txHash := common.BytesToHexString(txExt.GetTxid())
	tx := txExt.GetTransaction()
	tx, err = signerInst.Sign(tx)
	if err != nil {
		return "", fmt.Errorf("tron client: sign delegate: %w", err)
	}

	result, err := t.conn.BroadcastCtx(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("tron client: broadcast delegate: %w", err)
	}
	if result.GetCode() != api.Return_SUCCESS {
		return "", fmt.Errorf("tron client: delegate broadcast code: %s", result.GetCode())
	}

	return txHash, nil
}

// UndelegateEnergy reclaims delegated energy from receiverAddr back to ownerAddr.
// privKeyBytes is the raw 32-byte private key of the owner (hot wallet).
func (t *TronClient) UndelegateEnergy(ctx context.Context, ownerAddr, receiverAddr string, privKeyBytes []byte, energyAmount int64) (string, error) {
	if t.conn == nil {
		return "", fmt.Errorf("tron client: not started")
	}
	sk, _ := btcec.PrivKeyFromBytes(privKeyBytes)
	if sk == nil {
		return "", fmt.Errorf("tron client: invalid owner key")
	}

	signerInst, err := signer.NewPrivateKeySignerFromBTCEC(sk)
	if err != nil {
		return "", fmt.Errorf("tron client: create signer: %w", err)
	}

	txExt, err := t.conn.UnDelegateResourceCtx(ctx, ownerAddr, receiverAddr, core.ResourceCode_ENERGY, energyAmount)
	if err != nil {
		return "", fmt.Errorf("tron client: undelegate resource: %w", err)
	}
	if txExt.GetResult() == nil || !txExt.GetResult().GetResult() {
		msg := ""
		if txExt.GetResult() != nil {
			msg = string(txExt.GetResult().GetMessage())
		}
		return "", fmt.Errorf("tron client: undelegate result: %s", msg)
	}

	txHash := common.BytesToHexString(txExt.GetTxid())
	tx := txExt.GetTransaction()
	tx, err = signerInst.Sign(tx)
	if err != nil {
		return "", fmt.Errorf("tron client: sign undelegate: %w", err)
	}

	result, err := t.conn.BroadcastCtx(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("tron client: broadcast undelegate: %w", err)
	}
	if result.GetCode() != api.Return_SUCCESS {
		return "", fmt.Errorf("tron client: undelegate broadcast code: %s", result.GetCode())
	}

	return txHash, nil
}

// WaitForConfirmation polls the TRON node until the transaction is confirmed or timeout.
func (t *TronClient) WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) error {
	if t.conn == nil {
		return fmt.Errorf("tron client: not started")
	}
	deadline := time.Now().Add(timeout)
	pollInterval := 3 * time.Second

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		txi, err := t.conn.GetTransactionInfoByIDCtx(ctx, txHash)
		if err == nil && txi != nil && txi.GetReceipt() != nil {
			if txi.GetReceipt().GetResult() == core.Transaction_Result_SUCCESS {
				return nil
			}
			return fmt.Errorf("tron client: tx %s failed with result: %s", txHash, txi.GetReceipt().GetResult())
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("tron client: tx %s not confirmed within %s", txHash, timeout)
}

// buildTRC20TransferData builds the calldata for transfer(address,uint256).
// Selector: 0xa9059cbb (first 4 bytes of keccak256("transfer(address,uint256)")).
// Parameters: address (padded to 32 bytes, right-aligned), amount (padded to 32 bytes, big-endian).
func buildTRC20TransferData(toAddr string, amountSun int64) ([]byte, error) {
	// Parse the Base58 TRON address to 20-byte hex.
	addrBytes, err := parseBase58ToBytes(toAddr)
	if err != nil {
		return nil, err
	}

	// Build 4 + 32 + 32 = 68 bytes.
	data := make([]byte, 0, 68)

	// Selector: transfer(address,uint256) = 0xa9059cbb
	data = append(data, 0xa9, 0x05, 0x9c, 0xbb)

	// Address: pad 20 bytes to 32 bytes (left-pad with zeros).
	addrParam := make([]byte, 32)
	copy(addrParam[12:], addrBytes) // right-align in 32 bytes
	data = append(data, addrParam...)

	// Amount: big-endian int64 in 32 bytes.
	amountParam := make([]byte, 32)
	// Convert amountSun to big-endian bytes.
	for i := 0; i < 8; i++ {
		amountParam[31-i] = byte(amountSun >> (i * 8))
	}
	data = append(data, amountParam...)

	return data, nil
}

// parseBase58ToBytes converts a TRON Base58 address (T...) to 20-byte raw address.
// Base58ToAddress returns 21 bytes (0x41 version prefix + 20 address bytes);
// we strip the prefix for ABI-encoded smart contract calldata.
func parseBase58ToBytes(addr string) ([]byte, error) {
	tronAddr, err := address.Base58ToAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("parse tron address: %w", err)
	}
	raw := tronAddr.Bytes()
	if len(raw) == 21 && raw[0] == 0x41 {
		return raw[1:], nil
	}
	return raw, nil
}
