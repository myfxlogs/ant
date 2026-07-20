// cmd/solvency-check is a standalone binary that runs on an admin device or
// cold machine (NOT on the online server) to verify platform solvency.
//
// It independently queries the public TRON blockchain for on-chain custody
// (sum of all deposit address USDT balances + cold wallet USDT), then compares
// against the platform's self-reported total liabilities via admin RPC.
//
// Alerts:
//   - liability > custody + tolerance → INSOLVENCY
//   - ledger tip seq hasn't advanced in N hours → DEAD-MAN SWITCH
//   - entry_hash chain mismatch → TAMPERING
//
// Usage:
//   solvency-check -server http://online-host:8080 \
//                  -api-key <admin_api_key> \
//                  -xpub <deposit_xpub> \
//                  -cold-wallet <cold_wallet_address> \
//                  -trongrid-key <api_key> \
//                  [-tolerance 1.0] [-deadman-hours 6] [-interval 300s] \
//                  [-chain-history ledger_history.txt]
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/chain"
	"alphaforge/internal/hdwallet"
)

func main() {
	serverURL := flag.String("server", "", "online server base URL (e.g. http://host:8080)")
	apiKey := flag.String("api-key", "", "admin API key for authentication")
	xpubStr := flag.String("xpub", "", "deposit account-level xpub for address derivation")
	coldWallet := flag.String("cold-wallet", "", "cold wallet TRC20 address")
	tronGridKey := flag.String("trongrid-key", "", "TronGrid API key")
	tolerance := flag.String("tolerance", "1.0", "solvency tolerance in USDT (liability may exceed custody by this much)")
	deadmanHours := flag.Int("deadman-hours", 6, "hours without ledger advancement before dead-man alert")
	interval := flag.Duration("interval", 5*time.Minute, "check interval")
	chainHistoryFile := flag.String("chain-history", "", "path to local ledger history file for tampering detection")
	flag.Parse()

	if *serverURL == "" || *xpubStr == "" || *coldWallet == "" || *apiKey == "" {
		fmt.Fprintln(os.Stderr, "solvency-check: -server, -api-key, -xpub, -cold-wallet are required")
		os.Exit(1)
	}

	tol, err := decimal.NewFromString(*tolerance)
	if err != nil {
		fmt.Fprintf(os.Stderr, "solvency-check: invalid tolerance: %v\n", err)
		os.Exit(1)
	}

	xpubKey, err := hdwallet.ParseXpub(*xpubStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "solvency-check: invalid xpub: %v\n", err)
		os.Exit(1)
	}

	tronGrid := chain.NewTronGridClient(*tronGridKey)

	// Interceptor that adds X-API-Key header to every RPC call.
	authInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("X-API-Key", *apiKey)
			return next(ctx, req)
		})
	})

	httpClient := &http.Client{Timeout: 30 * time.Second}
	billingClient := antv1c.NewAdminBillingServiceClient(httpClient, *serverURL, connect.WithInterceptors(authInterceptor))
	depositClient := antv1c.NewDepositServiceClient(httpClient, *serverURL, connect.WithInterceptors(authInterceptor))

	checker := &checker{
		billingClient:    billingClient,
		depositClient:    depositClient,
		tronGrid:         tronGrid,
		xpubKey:          xpubKey,
		coldWallet:       *coldWallet,
		tolerance:        tol,
		deadmanHours:     *deadmanHours,
		chainHistoryFile: *chainHistoryFile,
	}

	ctx := context.Background()
	for {
		if err := checker.run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[ALERT] %v\n", err)
		} else {
			fmt.Printf("[OK] solvency check passed at %s\n", time.Now().Format(time.RFC3339))
		}
		select {
		case <-time.After(*interval):
		case <-ctx.Done():
			return
		}
	}
}

type checker struct {
	billingClient    antv1c.AdminBillingServiceClient
	depositClient    antv1c.DepositServiceClient
	tronGrid         *chain.TronGridClient
	xpubKey          *hdkeychain.ExtendedKey
	coldWallet       string
	tolerance        decimal.Decimal
	deadmanHours     int
	chainHistoryFile string
}

func (c *checker) run(ctx context.Context) error {
	// 1. Get platform self-reported ledger summary.
	ledgerResp, err := c.billingClient.GetLedgerSummary(ctx, connect.NewRequest(&antv1.GetLedgerSummaryRequest{}))
	if err != nil {
		return fmt.Errorf("GetLedgerSummary: %w", err)
	}
	liability, err := decimal.NewFromString(ledgerResp.Msg.TotalLiabilities)
	if err != nil {
		return fmt.Errorf("parse total_liabilities: %w", err)
	}

	// 2. Tampering detection: compare against local chain history.
	if c.chainHistoryFile != "" {
		if err := c.verifyChainHistory(ledgerResp.Msg); err != nil {
			return err
		}
		if err := c.appendChainHistory(ledgerResp.Msg); err != nil {
			fmt.Fprintf(os.Stderr, "  WARN: failed to append chain history: %v\n", err)
		}
	}

	// 3. Dead-man switch: check if ledger tip is stale.
	if ledgerResp.Msg.LatestEntryTime != nil {
		tipTime := ledgerResp.Msg.LatestEntryTime.AsTime()
		staleDuration := time.Since(tipTime)
		if staleDuration > time.Duration(c.deadmanHours)*time.Hour {
			return fmt.Errorf("DEAD-MAN SWITCH: ledger tip stale for %s (seq=%d, last=%s)",
				staleDuration.Round(time.Minute), ledgerResp.Msg.LatestSeq, tipTime.Format(time.RFC3339))
		}
	}

	// 4. Independently compute on-chain custody.
	custody, err := c.computeCustody(ctx)
	if err != nil {
		return fmt.Errorf("compute custody: %w", err)
	}

	// 5. Solvency check: liability must not exceed custody + tolerance.
	excess := liability.Sub(custody).Sub(c.tolerance)
	if excess.GreaterThan(decimal.Zero) {
		return fmt.Errorf("INSOLVENCY: liability=%s USDT, custody=%s USDT, tolerance=%s USDT — shortfall=%s USDT",
			liability.String(), custody.String(), c.tolerance.String(), excess.String())
	}

	fmt.Printf("  liability: %s USDT\n", liability.String())
	fmt.Printf("  custody:   %s USDT\n", custody.String())
	fmt.Printf("  margin:    %s USDT\n", custody.Sub(liability).String())
	return nil
}

func (c *checker) computeCustody(ctx context.Context) (decimal.Decimal, error) {
	total := decimal.Zero

	// Sum all deposit address balances.
	page := int32(1)
	pageSize := int32(100)
	for {
		resp, err := c.depositClient.ListDepositAddresses(ctx, connect.NewRequest(&antv1.ListDepositAddressesRequest{
			Page:     page,
			PageSize: pageSize,
		}))
		if err != nil {
			return decimal.Zero, fmt.Errorf("ListDepositAddresses page %d: %w", page, err)
		}

		for _, addr := range resp.Msg.Addresses {
			// Verify address matches xpub derivation (detect xpub substitution).
			derived, err := hdwallet.DeriveAddressFromExtKey(c.xpubKey, uint32(addr.DerivationIndex))
			if err != nil {
				return decimal.Zero, fmt.Errorf("derive index %d: %w", addr.DerivationIndex, err)
			}
			if derived != addr.Address {
				return decimal.Zero, fmt.Errorf("TAMPERING: address mismatch at index %d: expected=%s got=%s",
					addr.DerivationIndex, derived, addr.Address)
			}

			balStr, err := c.tronGrid.GetTRC20Balance(ctx, addr.Address)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  WARN: GetTRC20Balance(%s): %v\n", addr.Address, err)
				continue
			}
			bal, err := decimal.NewFromString(balStr)
			if err != nil {
				return decimal.Zero, fmt.Errorf("parse balance for %s: %q: %w", addr.Address, balStr, err)
			}
			total = total.Add(bal)
		}

		if int32(len(resp.Msg.Addresses)) < pageSize {
			break
		}
		page++
	}

	// Add cold wallet balance.
	coldBal, err := c.tronGrid.GetTRC20Balance(ctx, c.coldWallet)
	if err != nil {
		return decimal.Zero, fmt.Errorf("cold wallet balance: %w", err)
	}
	cold, err := decimal.NewFromString(coldBal)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse cold wallet balance: %q: %w", coldBal, err)
	}
	total = total.Add(cold)

	return total, nil
}

// verifyChainHistory reads the local chain history file and checks that the
// latest (seq, entry_hash) from the server is consistent with what we recorded
// previously. If the server reports a seq we've already seen with a different
// hash, or a seq lower than our last recorded seq, that indicates tampering.
func (c *checker) verifyChainHistory(resp *antv1.GetLedgerSummaryResponse) error {
	f, err := os.Open(c.chainHistoryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // first run — no history to compare
		}
		return fmt.Errorf("open chain history: %w", err)
	}
	defer f.Close()

	var lastSeq int64
	var lastHash string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			var s int64
			fmt.Sscanf(parts[0], "%d", &s)
			lastSeq = s
			lastHash = parts[1]
		}
	}

	if resp.LatestSeq < lastSeq {
		return fmt.Errorf("TAMPERING: server seq=%d < local last seq=%d (chain rollback detected)",
			resp.LatestSeq, lastSeq)
	}
	if resp.LatestSeq == lastSeq && resp.LatestEntryHash != lastHash {
		return fmt.Errorf("TAMPERING: same seq=%d but hash changed (local=%s server=%s)",
			lastSeq, lastHash, resp.LatestEntryHash)
	}
	return nil
}

// appendChainHistory records the current (seq, hash, timestamp) to the local
// history file. Each check appends one line, building an append-only local
// record of the chain tip over time.
func (c *checker) appendChainHistory(resp *antv1.GetLedgerSummaryResponse) error {
	f, err := os.OpenFile(c.chainHistoryFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open chain history for append: %w", err)
	}
	defer f.Close()

	timestamp := time.Now().Format(time.RFC3339)
	_, err = fmt.Fprintf(f, "%d %s %s\n", resp.LatestSeq, resp.LatestEntryHash, timestamp)
	return err
}
