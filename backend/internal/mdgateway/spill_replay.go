package mdgateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// SpillReplay reads spill JSONL files and replays them through the publisher and CH writer.
type SpillReplay struct {
	dir       string
	publisher *Publisher
	ch        *CHWriter
	log       *zap.Logger
}

// NewSpillReplay creates a replay engine.
func NewSpillReplay(dir string, pub *Publisher, ch *CHWriter, log *zap.Logger) *SpillReplay {
	return &SpillReplay{dir: dir, publisher: pub, ch: ch, log: log}
}

// Run scans the spill directory for *.jsonl files and replays them in
// filename order. Successful files are moved to processed/; failed rows
// go to failed/.
func (r *SpillReplay) Run(ctx context.Context) (int, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("spill_replay: read dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".jsonl" {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return 0, nil
	}
	sort.Strings(files)

	processedDir := filepath.Join(r.dir, "processed")
	failedDir := filepath.Join(r.dir, "failed")
	os.MkdirAll(processedDir, 0755)
	os.MkdirAll(failedDir, 0755)

	var total int
	for _, fname := range files {
		path := filepath.Join(r.dir, fname)
		n, err := r.replayFile(ctx, path)
		total += n
		if err != nil {
			r.log.Warn("spill_replay: file failed", zap.String("file", fname), zap.Error(err))
			os.Rename(path, filepath.Join(failedDir, fname))
			continue
		}
		os.Rename(path, filepath.Join(processedDir, fname))
		r.log.Info("spill_replay: replayed", zap.String("file", fname), zap.Int("rows", n))
	}
	return total, nil
}

func (r *SpillReplay) replayFile(ctx context.Context, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var e spillEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip malformed lines
		}

		// ADR-0009 §2.1: dual-write — publisher first, then CHWriter.
		// Construct minimal DTO from spillEntry (sparse format; missing
		// UserID/AccountID/SymbolRaw are empty — fine for replay).
		switch e.Kind {
		case "tick":
			tick := &mdtick.Tick{
				Broker:        e.Broker,
				Canonical:     e.Canonical,
				TsUnixMs:      e.Ts,
				ArrivedUnixMs: e.Ts,
				Bid:           decimal.RequireFromString(e.Bid),
				Ask:           decimal.RequireFromString(e.Ask),
				BidVolume:     e.BidVol,
				AskVolume:     e.AskVol,
				IsReplay:      true,
			}
			_ = r.publisher.PublishTick(tick)
			r.ch.EnqueueTick(tick)
		case "bar":
			bar := &mdtick.Bar{
				Broker:        e.Broker,
				Canonical:     e.Canonical,
				Period:        e.Period,
				OpenTsUnixMs:  e.Ts,
				CloseTsUnixMs: e.Ts,
				Open:          decimal.RequireFromString(e.Open),
				High:          decimal.RequireFromString(e.High),
				Low:           decimal.RequireFromString(e.Low),
				Close:         decimal.RequireFromString(e.Close),
				Volume:        e.Volume,
				TickCount:     e.Count,
				IsReplay:      true,
			}
			_ = r.publisher.PublishBar(bar)
			r.ch.EnqueueBar(bar)
		}
		count++
	}
	return count, scanner.Err()
}
