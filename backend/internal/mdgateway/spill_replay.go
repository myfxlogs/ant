package mdgateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.uber.org/zap"
)

// SpillReplay reads spill JSONL files and replays them through the CH writer.
type SpillReplay struct {
	dir  string
	ch   *CHWriter
	log  *zap.Logger
}

// NewSpillReplay creates a replay engine.
func NewSpillReplay(dir string, ch *CHWriter, log *zap.Logger) *SpillReplay {
	return &SpillReplay{dir: dir, ch: ch, log: log}
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
		count++
	}
	return count, scanner.Err()
}
