// Package mdgateway — spill writer for CH write failures.
// When ClickHouse is unavailable, ticks are written to JSONL spill files
// and replayed on next startup via SpillReplay.
package mdgateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SpillConfig holds spill writer settings.
type SpillConfig struct {
	Dir         string        // spill directory (default "/tmp/mdgateway/spill")
	MaxFileSize int64         // max bytes per spill file before rotation (default 100MB)
	MaxAge      time.Duration // max age of spill files before cleanup (default 24h)
	SyncEvery   int           // fsync after N writes (default 100, 0 = every write)
}

// DefaultSpillConfig returns sensible defaults.
func DefaultSpillConfig() SpillConfig {
	return SpillConfig{
		Dir:         "/tmp/mdgateway/spill",
		MaxFileSize: 100 * 1024 * 1024,
		MaxAge:      24 * time.Hour,
		SyncEvery:   100,
	}
}

// spillRecord is the JSONL format written to spill files.
type spillRecord struct {
	UserID        string  `json:"user_id"`
	Broker        string  `json:"broker"`
	Symbol        string  `json:"symbol"`
	TsUnixMs      int64   `json:"ts_unix_ms"`
	ArrivedUnixMs int64   `json:"arrived_unix_ms"`
	Bid           float64 `json:"bid"`
	Ask           float64 `json:"ask"`
	BidVolume     float64 `json:"bid_volume"`
	AskVolume     float64 `json:"ask_volume"`
}

// SpillWriter writes ticks to JSONL files when CH is unavailable.
type SpillWriter struct {
	cfg   SpillConfig
	log   *zap.Logger
	mu    sync.Mutex
	file  *os.File
	count int
	size  int64
}

// NewSpillWriter creates a spill writer.
func NewSpillWriter(cfg SpillConfig, log *zap.Logger) (*SpillWriter, error) {
	if cfg.Dir == "" {
		cfg.Dir = "/tmp/mdgateway/spill"
	}
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = 100 * 1024 * 1024
	}
	if cfg.SyncEvery <= 0 {
		cfg.SyncEvery = 100
	}

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("spill: mkdir: %w", err)
	}

	return &SpillWriter{cfg: cfg, log: log}, nil
}

// Write writes a Tick to the current spill file.
func (w *SpillWriter) Write(tick *Tick) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.ensureFile(); err != nil {
		return err
	}

	rec := spillRecord{
		UserID:        tick.UserID,
		Broker:        tick.Broker,
		Symbol:        tick.Symbol,
		TsUnixMs:      tick.TsUnixMs,
		ArrivedUnixMs: tick.ArrivedUnixMs,
		BidVolume:     tick.BidVolume,
		AskVolume:     tick.AskVolume,
	}
	if tick.GetBid() != nil {
		rec.Bid, _ = parseFloat(tick.GetBid().GetValue())
	}
	if tick.GetAsk() != nil {
		rec.Ask, _ = parseFloat(tick.GetAsk().GetValue())
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("spill: marshal: %w", err)
	}
	data = append(data, '\n')

	n, err := w.file.Write(data)
	if err != nil {
		return fmt.Errorf("spill: write: %w", err)
	}

	w.count++
	w.size += int64(n)

	if w.cfg.SyncEvery > 0 && w.count%w.cfg.SyncEvery == 0 {
		if err := w.file.Sync(); err != nil {
			w.log.Warn("spill: fsync failed", zap.Error(err))
		}
	}

	if w.size >= w.cfg.MaxFileSize {
		if err := w.rotate(); err != nil {
			return err
		}
	}

	return nil
}

// WriteBatch writes multiple ticks to the spill file.
func (w *SpillWriter) WriteBatch(ticks []*Tick) error {
	for _, t := range ticks {
		if err := w.Write(t); err != nil {
			return err
		}
	}
	return nil
}

// Count returns the number of ticks written since last rotation.
func (w *SpillWriter) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.count
}

// Close flushes and closes the current spill file.
func (w *SpillWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			w.log.Warn("spill: final fsync failed", zap.Error(err))
		}
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}
	return nil
}

func (w *SpillWriter) ensureFile() error {
	if w.file != nil {
		return nil
	}

	fname := fmt.Sprintf("spill-%s.jsonl", time.Now().UTC().Format("20060102T150405"))
	path := filepath.Join(w.cfg.Dir, fname)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("spill: open file: %w", err)
	}

	w.file = f
	w.count = 0
	w.size = 0

	if w.log != nil {
		w.log.Info("spill: new file", zap.String("path", path))
	}
	return nil
}

func (w *SpillWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			w.log.Warn("spill: rotate fsync failed", zap.Error(err))
		}
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}
	return w.ensureFile()
}
