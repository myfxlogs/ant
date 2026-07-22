// cmd/coldsign is an air-gapped cold signing tool for HD wallet sweep bundles.
//
// It reads an UnsignedSweepBundle (proto binary), derives private keys from
// a BIP39 mnemonic, verifies xpub fingerprint, enforces whitelist (R4),
// and produces a SignedSweepBundle.
//
// Derivation paths (ADR-0026 §11 G):
//   TransferTx  → m/44'/195'/0'/0/{derivation_index}
//   DelegateTx  → m/44'/195'/0'/1/0 (energy account fixed path)
//   UndelegateTx → m/44'/195'/0'/1/0 (energy account fixed path)
//
// Whitelist (R4):
//   TransferTx without auth (sweep) → to_address must == cold_wallet_address
//   TransferTx with auth (withdrawal) → verify WebAuthn (Phase E)
//
// Usage: coldsign -i bundle.bin [-o signed.bin] [--cold-wallet T...]
// Mnemonic is read from stdin (never via flag to avoid ps/history leakage).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/go-bip39"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/hdwallet"
)

func main() {
	inPath := flag.String("i", "", "input: UnsignedSweepBundle proto binary (required)")
	outPath := flag.String("o", "", "output: SignedSweepBundle proto binary (default: stdout hex)")
	coldWallet := flag.String("cold-wallet", "", "cold wallet TRC20 address for R4 whitelist (required for sweep)")
	credDBPath := flag.String("cred-db", "", "path to coldsign self-held credential database JSON (required for withdrawals)")
	whitelistPath := flag.String("whitelist", "", "path to withdrawal whitelist JSON (optional, for withdrawal verification)")
	maxWithdrawal := flag.String("max-withdrawal", "", "max per-withdrawal amount in USDT (optional, for withdrawal verification)")
	rpIDFlag := flag.String("rp-id", "", "WebAuthn relying party ID (required for withdrawals)")
	rpOriginFlag := flag.String("rp-origin", "", "WebAuthn relying party origin (required for withdrawals)")
	flag.Parse()

	if *inPath == "" {
		fmt.Fprintln(os.Stderr, "coldsign: -i is required")
		flag.Usage()
		os.Exit(1)
	}

	// Read mnemonic from stdin — never via -m flag to avoid ps/history leakage (ADR-0026 §8.1 R2).
	fmt.Fprint(os.Stderr, "Enter mnemonic: ")
	reader := bufio.NewReader(os.Stdin)
	mnemonicLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "coldsign: read mnemonic: %v\n", err)
		os.Exit(1)
	}
	mnemonicStr := strings.TrimSpace(mnemonicLine)

	if err := run(*inPath, mnemonicStr, *outPath, *coldWallet, *credDBPath, *whitelistPath, *maxWithdrawal, *rpIDFlag, *rpOriginFlag); err != nil {
		fmt.Fprintf(os.Stderr, "coldsign: %v\n", err)
		os.Exit(1)
	}
}

func run(inPath, mnemonicStr, outPath, coldWalletAddr, credDBPath, whitelistPath, maxWithdrawal, rpIDFlag, rpOriginFlag string) error {
	data, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	bundle := &antv1.UnsignedSweepBundle{}
	if err := proto.Unmarshal(data, bundle); err != nil {
		return fmt.Errorf("unmarshal unsigned bundle: %w", err)
	}

	if len(bundle.Txs) == 0 {
		return fmt.Errorf("bundle contains no transactions")
	}

	if !bip39.IsMnemonicValid(mnemonicStr) {
		return fmt.Errorf("invalid mnemonic")
	}

	// Derive seed ONCE — reuse across all txs in the bundle.
	seed := bip39.NewSeed(mnemonicStr, "")
	defer hdwallet.ZeroSeed(seed)

	_, fingerprint, err := hdwallet.DeriveAccountXpubAndFingerprint(seed)
	if err != nil {
		return fmt.Errorf("derive xpub: %w", err)
	}

	if bundle.XpubFingerprint != "" && bundle.XpubFingerprint != fingerprint {
		return fmt.Errorf("xpub fingerprint mismatch: bundle=%s actual=%s — refusing to sign",
			bundle.XpubFingerprint, fingerprint)
	}
	fmt.Fprintf(os.Stderr, "xpub fingerprint verified: %s\n", fingerprint)

	// Load coldsign self-held credential database for withdrawal verification (R11).
	var credDB *CredentialDB
	if credDBPath != "" {
		credDB, err = LoadCredentialDB(credDBPath)
		if err != nil {
			return fmt.Errorf("load credential database: %w", err)
		}
		fmt.Fprintf(os.Stderr, "credential database loaded: %d credentials\n", len(credDB.Credentials))
	}
	rpID = rpIDFlag
	rpOrigin = rpOriginFlag

	signedTxs := make([]*antv1.SignedTx, 0, len(bundle.Txs))
	for i, tx := range bundle.Txs {
		signed, err := signTx(tx, seed, coldWalletAddr, credDB, whitelistPath, maxWithdrawal)
		if err != nil {
			return fmt.Errorf("sign tx %d (kind=%s from=%s): %w",
				i, tx.Kind.String(), tx.FromAddress, err)
		}
		signedTxs = append(signedTxs, signed)
		fmt.Fprintf(os.Stderr, "signed tx %d: kind=%s from=%s to=%s amount=%s\n",
			i, tx.Kind.String(), tx.FromAddress, tx.ToAddress, tx.Amount)
	}

	signedBundle := &antv1.SignedSweepBundle{
		Txs:             signedTxs,
		BundleId:        bundle.BundleId,
		SignedAt:        time.Now().UnixMilli(),
		XpubFingerprint: fingerprint,
	}

	out, err := proto.Marshal(signedBundle)
	if err != nil {
		return fmt.Errorf("marshal signed bundle: %w", err)
	}

	if outPath != "" {
		if err := os.WriteFile(outPath, out, 0600); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "signed bundle written to %s\n", outPath)
	} else {
		fmt.Printf("%x\n", out)
	}
	return nil
}

// signTx derives the private key for the transaction and signs it.
// Derivation path depends on the oneof tx type (ADR-0026 §11 G):
//   TransferTx  → m/44'/195'/0'/0/{derivation_index}
//   DelegateTx  → m/44'/195'/0'/1/0 (energy account, change=1)
//   UndelegateTx → m/44'/195'/0'/1/0 (energy account, change=1)
func signTx(tx *antv1.UnsignedTx, seed []byte, coldWalletAddr string, credDB *CredentialDB, whitelistPath, maxWithdrawal string) (*antv1.SignedTx, error) {
	// Derive private key once — used for both address verification and signing.
	var sk *btcec.PrivateKey
	var err error
	switch tx.Tx.(type) {
	case *antv1.UnsignedTx_Transfer:
		sk, err = hdwallet.DeriveDepositPrivKey(seed, tx.DerivationIndex)
		if err != nil {
			return nil, fmt.Errorf("derive private key at index %d: %w", tx.DerivationIndex, err)
		}
		derivedAddr := address.BTCECPrivkeyToAddress(sk).String()
		if derivedAddr != tx.FromAddress {
			return nil, fmt.Errorf("address mismatch: derived=%s expected=%s — refusing to sign",
				derivedAddr, tx.FromAddress)
		}

		// R4 whitelist: TransferTx without auth (sweep) → to must == cold_wallet_address
		transferTx := tx.GetTransfer()
		if transferTx.GetAuth() == nil {
			if coldWalletAddr == "" {
				return nil, fmt.Errorf("R4: cold wallet address not provided — cannot verify sweep whitelist")
			}
			if tx.ToAddress != coldWalletAddr {
				return nil, fmt.Errorf("R4 whitelist violation: sweep destination %s != cold wallet %s — refusing to sign",
					tx.ToAddress, coldWalletAddr)
			}
		} else {
			// R11: Withdrawal transfer — verify WebAuthn assertion with self-held pubkey.
			if credDB == nil {
				return nil, fmt.Errorf("R11: withdrawal transfer requires -cred-db — refusing to sign")
			}
			if err := verifyWithdrawalAuth(tx, credDB, whitelistPath, maxWithdrawal); err != nil {
				return nil, err
			}
		}

	case *antv1.UnsignedTx_Delegate, *antv1.UnsignedTx_Undelegate:
		// Energy account: m/44'/195'/0'/1/0 (change=1, index=0) — same path for both
		sk, err = hdwallet.DeriveEnergyAccountPrivKey(seed)
		if err != nil {
			return nil, fmt.Errorf("derive energy account private key: %w", err)
		}
		derivedAddr := address.BTCECPrivkeyToAddress(sk).String()
		if derivedAddr != tx.FromAddress {
			return nil, fmt.Errorf("energy account address mismatch: derived=%s expected=%s",
				derivedAddr, tx.FromAddress)
		}

	default:
		return nil, fmt.Errorf("unknown tx type or nil oneof")
	}

	// Sign the raw transaction with the derived private key (ADR-0026 §10.2).
	// This is the core air-gapped operation — no network access needed.
	signedData, txid, err := hdwallet.SignTronTransaction(tx.RawTx, sk)
	if err != nil {
		return nil, fmt.Errorf("sign raw tx: %w", err)
	}

	return &antv1.SignedTx{
		Kind:         tx.Kind,
		FromAddress:  tx.FromAddress,
		ToAddress:    tx.ToAddress,
		Amount:       tx.Amount,
		SignedTxData: signedData,
		TxHash:       txid,
	}, nil
}
