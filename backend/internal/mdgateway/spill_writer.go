package mdgateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

type SpillWriterConfig struct {
	Dir          string        // default /var/lib/ant/spill
	MaxFileBytes int64         // default 100 MB
	MaxFileAge   time.Duration // default 1h
}

func DefaultSpillConfig() SpillWriterConfig {
	return SpillWriterConfig{
		Dir:          "/var/lib/ant/spill",
		MaxFileBytes: 100 * 1024 * 1024,
		MaxFileAge:   time.Hour,
	}
}

type SpillWriter struct {
	cfg SpillWriterConfig
	log *zap.Logger

	mu       sync.Mutex
	cur      *os.File
	curBytes int64
	curStart time.Time
}

func NewSpillWriter(cfg SpillWriterConfig, log *zap.Logger) (*SpillWriter, error) {
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("spill: mkdir %s: %w", cfg.Dir, err)
	}
	return &SpillWriter{cfg: cfg, log: log}, nil
}

type spillEntry struct {
	Kind     string  `json:"_kind"` // "tick" or "bar"
	Ts       int64   `json:"ts"`
	Broker   string  `json:"broker"`
	Canonical string `json:"canonical"`
	Bid      string  `json:"bid,omitempty"`
	Ask      string  `json:"ask,omitempty"`
	BidVol   float64 `json:"bv,omitempty"`
	AskVol   float64 `json:"av,omitempty"`
	Period   string  `json:"period,omitempty"`
	Open     string  `json:"O,omitempty"`
	High     string  `json:"H,omitempty"`
	Low      string  `json:"L,omitempty"`
	Close    string  `json:"C,omitempty"`
	Volume   float64 `json:"V,omitempty"`
	Count    uint32  `json:"N,omitempty"`
}

func (s *SpillWriter) WriteTick(t *mdtick.Tick) error {
	return s.write(spillEntry{
		Kind: "tick", Ts: t.ArrivedUnixMs, Broker: t.Broker, Canonical: t.Canonical,
		Bid: t.Bid.String(), Ask: t.Ask.String(), BidVol: t.BidVolume, AskVol: t.AskVolume,
	})
}

func (s *SpillWriter) WriteBar(b *mdtick.Bar) error {
	return s.write(spillEntry{
		Kind: "bar", Ts: b.CloseTsUnixMs, Broker: b.Broker, Canonical: b.Canonical,
		Period: b.Period, Open: b.Open.String(), High: b.High.String(),
		Low: b.Low.String(), Close: b.Close.String(), Volume: b.Volume, Count: b.TickCount,
	})
}

func (s *SpillWriter) write(e spillEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cur == nil || s.shouldRotate() {
		if err := s.rotate(); err != nil {
			return fmt.Errorf("rotate spill file: %w", err)
		}
	}

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal spill entry: %w", err)
	}
	data = append(data, '\n')
	n, err := s.cur.Write(data)
	if err != nil {
		return fmt.Errorf("write spill entry to file: %w", err)
	}
	s.curBytes += int64(n)
	return nil
}

func (s *SpillWriter) shouldRotate() bool {
	return s.curBytes >= s.cfg.MaxFileBytes ||
		time.Since(s.curStart) >= s.cfg.MaxFileAge
}

func (s *SpillWriter) rotate() error {
	if s.cur != nil {
		s.cur.Close()
	}
	fname := filepath.Join(s.cfg.Dir, fmt.Sprintf("spill-%d.jsonl", Clk.Now().UnixMilli()))
	f, err := os.Create(fname)
	if err != nil {
		return fmt.Errorf("spill: create %s: %w", fname, err)
	}
	s.cur = f
	s.curBytes = 0
	s.curStart = Clk.Now()
	s.log.Info("spill: rotated", zap.String("file", fname))
	return nil
}

func (s *SpillWriter) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur != nil {
		if err := s.cur.Close(); err != nil {
			return fmt.Errorf("close spill file: %w", err)
		}
		s.cur = nil
	}
	return nil
}
