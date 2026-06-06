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
	dir        string
	publisher  *Publisher
	ch         *CHWriter
	aggregator *BarAggregator // finality check for bar dedup (ADR-0009 §2.2)
	log        *zap.Logger
}

// NewSpillReplay creates a replay engine.
func NewSpillReplay(dir string, pub *Publisher, ch *CHWriter, aggregator *BarAggregator, log *zap.Logger) *SpillReplay {
	return &SpillReplay{dir: dir, publisher: pub, ch: ch, aggregator: aggregator, log: log}
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
	os.MkdirAll(processedDir, 0750)
	os.MkdirAll(failedDir, 0750)

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
	f, err := os.Open(path) // #nosec G304 — path from fixed spill directory
	if err != nil { return 0, err }
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e spillEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil { continue }
		if r.replayEntry(ctx, &e) { count++ }
	}
	return count, scanner.Err()
}

func (r *SpillReplay) replayEntry(ctx context.Context, e *spillEntry) bool {
	switch e.Kind {
	case "tick":
		return r.replayTick(ctx, e)
	case "bar":
		return r.replayBar(ctx, e)
	}
	return false
}

func (r *SpillReplay) replayTick(ctx context.Context, e *spillEntry) bool {
	bidDec, errBid := decimal.NewFromString(e.Bid)
	askDec, errAsk := decimal.NewFromString(e.Ask)
	if errBid != nil || errAsk != nil { return false }
	tick := &mdtick.Tick{
		Broker: e.Broker, Canonical: e.Canonical, TsUnixMs: e.Ts, ArrivedUnixMs: e.Ts,
		Bid: bidDec, Ask: askDec, BidVolume: e.BidVol, AskVolume: e.AskVol, IsReplay: true,
	}
	if r.publisher != nil { _ = r.publisher.PublishTick(ctx, tick) }
	if r.ch != nil { r.ch.EnqueueTick(tick) }
	return true
}

func (r *SpillReplay) replayBar(ctx context.Context, e *spillEntry) bool {
	openTs := e.Ts
	for _, p := range Periods {
		if p.Name == e.Period { openTs = e.Ts - p.Ms; break }
	}
	openDec, errO := decimal.NewFromString(e.Open)
	highDec, errH := decimal.NewFromString(e.High)
	lowDec, errL := decimal.NewFromString(e.Low)
	closeDec, errC := decimal.NewFromString(e.Close)
	if errO != nil || errH != nil || errL != nil || errC != nil { return false }
	bar := &mdtick.Bar{
		Broker: e.Broker, Canonical: e.Canonical, Period: e.Period,
		OpenTsUnixMs: openTs, CloseTsUnixMs: e.Ts,
		Open: openDec, High: highDec, Low: lowDec, Close: closeDec,
		Volume: e.Volume, TickCount: e.Count, IsClosed: true, IsReplay: true,
	}
	if r.aggregator != nil && !r.aggregator.IngestExternalBar(bar) { return false }
	if r.publisher != nil { _ = r.publisher.PublishBar(ctx, bar) }
	if r.ch != nil { r.ch.EnqueueBar(bar) }
	return true
}
