package mdgateway

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// healthMonitor checks gateway health every 30s, monitors memory pressure
// for S-2 auto-degradation, and emits stale/dead account metrics.
func healthMonitor(ctx context.Context, mgr *Manager, chw *CHWriter, log *zap.Logger, onDisconnect func(accountID string)) {
	ticker := Clk.NewTicker(30 * time.Second)
	defer ticker.Stop()

	const (
		highThreshold = 0.80
		lowThreshold  = 0.60
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			var stale, dead int64
			for _, h := range mgr.Health() {
				switch h.State {
				case "stale":
					stale++
					log.Warn("mdgateway: stale account — no ticks for >5 min",
						zap.String("account", h.AccountID),
						zap.String("platform", h.Platform))
				case "dead":
					dead++
					mgr.MarkDisconnecting(h.AccountID)
					if onDisconnect != nil {
						onDisconnect(h.AccountID)
					}
					log.Error("mdgateway: dead account — no ticks for >15 min",
						zap.String("account", h.AccountID),
						zap.String("platform", h.Platform))
					if err := mgr.RemoveGateway(ctx, h.AccountID); err != nil {
						log.Warn("mdgateway: remove dead gateway failed",
							zap.String("account", h.AccountID), zap.Error(err))
					}
					mgr.UnmarkDisconnecting(h.AccountID)
					// Brief pause to let in-flight NATS messages drain
					// before reconnects are allowed for this account.
					time.Sleep(100 * time.Millisecond)
				case "no_data":
					log.Debug("mdgateway: no data yet",
						zap.String("account", h.AccountID))
				default:
					log.Debug("mdgateway: health",
						zap.String("account", h.AccountID),
						zap.String("state", h.State))
				}
			}
			SetStaleAccountCount(stale, dead)

			// S-2: memory pressure → auto buffer bypass.
			memRatio := currentMemoryRatio()
			bufEnabled := chw.BufferEnabled()

			if memRatio > highThreshold && bufEnabled {
				log.Warn("mdgateway: memory pressure — disabling CH Buffer engine",
					zap.Float64("mem_ratio", memRatio),
					zap.Float64("threshold", highThreshold))
				chw.SetBufferEnabled(false)
			} else if memRatio < lowThreshold && !bufEnabled {
				log.Info("mdgateway: memory pressure resolved — re-enabling CH Buffer engine",
					zap.Float64("mem_ratio", memRatio),
					zap.Float64("threshold", lowThreshold))
				chw.SetBufferEnabled(true)
			}
		}
	}
}

// currentMemoryRatio returns the current process RSS as a fraction of the memory limit.
func currentMemoryRatio() float64 {
	limitBytes := cgroupMemoryLimit()
	if limitBytes <= 0 {
		return 0
	}
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return float64(kb*1024) / float64(limitBytes)
				}
			}
		}
	}
	return 0
}

// cgroupMemoryLimit returns the cgroup memory limit in bytes (v1 or v2).
func cgroupMemoryLimit() int64 {
	// cgroup v2
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		val := strings.TrimSpace(string(data))
		if val == "max" {
			// No limit — fall through to system memory.
		} else if limit, err := strconv.ParseInt(val, 10, 64); err == nil && limit > 0 {
			return limit
		}
	}
	// cgroup v1 fallback
	if data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		val := strings.TrimSpace(string(data))
		if limit, err := strconv.ParseInt(val, 10, 64); err == nil && limit > 0 {
			return limit
		}
	}
	// Fallback: system total memory.
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return int64(ms.Sys)
}

// drain collects all pending ticks and bars from the CHWriter queues for flush.
func (w *CHWriter) drain() (ticks []*mdtick.Tick, bars []*mdtick.Bar) {
	// Drain tickQ.
	for {
		select {
		case t := <-w.tickQ:
			ticks = append(ticks, t)
		default:
			goto drainBars
		}
	}
drainBars:
	for {
		select {
		case b := <-w.barQ:
			bars = append(bars, b)
		default:
			return
		}
	}
}

// defaultQuoteSymbols returns a broad set of symbols for mtapi SymbolSubscribe
// when an account has no configured symbols. Kept in sync with frontend COMMON_SYMBOLS.
func defaultQuoteSymbols() []string {
	return []string{
		// Forex majors
		"EURUSDm", "GBPUSDm", "USDJPYm", "AUDUSDm", "NZDUSDm", "USDCADm", "USDCHFm",
		// Forex crosses
		"EURGBPm", "EURJPYm", "GBPJPYm", "AUDJPYm", "NZDJPYm", "CADJPYm", "CHFJPYm",
		"EURCHFm", "EURAUDm", "EURNZDm", "GBPCHFm", "GBPAUDm", "GBPNZDm",
		"GBPCADm", "AUDCADm", "AUDCHFm", "AUDNZDm", "NZDCADm", "NZDCHFm", "CADCHFm",
		// Metals
		"XAUUSDm", "XAGUSDm", "XAUJPYm",
		// Crypto
		"BTCUSDm", "ETHUSDm", "XRPUSDm", "SOLUSDm", "BNBUSDm",
		// Indices
		"US30m", "US100m", "GER40m",
	}
}
