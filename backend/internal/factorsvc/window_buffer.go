package factorsvc

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// WindowBuffer maintains a fixed-size rolling window of bars per key.
// Key format: "user_id/symbol/period"
type WindowBuffer struct {
	mu      sync.RWMutex
	buffers map[string]*ringBuffer
	maxSize int
	log     *zap.Logger
}

type ringBuffer struct {
	bars []*barRecord
	head int
	size int
	cap  int
}

// BarRecord is a flattened bar stored in the ring buffer.
type BarRecord struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
	TsMs   int64
}

type barRecord = BarRecord

// NewWindowBuffer creates a new window buffer with the given maximum window size.
func NewWindowBuffer(maxSize int, log *zap.Logger) *WindowBuffer {
	return &WindowBuffer{
		buffers: make(map[string]*ringBuffer),
		maxSize: maxSize,
		log:     log,
	}
}

// Push adds a bar to the window and evicts the oldest if at capacity.
func (w *WindowBuffer) Push(userID, symbol, period string, bar *Bar) {
	if bar == nil {
		return
	}
	key := fmt.Sprintf("%s/%s/%s", userID, symbol, period)

	open, _ := parseFloat(bar.Open)
	high, _ := parseFloat(bar.High)
	low, _ := parseFloat(bar.Low)
	closeV, _ := parseFloat(bar.Close)

	w.mu.Lock()
	rb, ok := w.buffers[key]
	if !ok {
		rb = &ringBuffer{bars: make([]*barRecord, w.maxSize), cap: w.maxSize}
		w.buffers[key] = rb
	}
	rb.add(&barRecord{
		Open:   open,
		High:   high,
		Low:    low,
		Close:  closeV,
		Volume: bar.Volume,
		TsMs:   bar.CloseTsUnixMs,
	})
	w.mu.Unlock()
}

// PushRaw adds a bar from raw values (used for CH bootstrap).
func (w *WindowBuffer) PushRaw(userID, symbol, period string, open, high, low, closeV, volume float64, tsMs int64) {
	key := fmt.Sprintf("%s/%s/%s", userID, symbol, period)

	w.mu.Lock()
	rb, ok := w.buffers[key]
	if !ok {
		rb = &ringBuffer{bars: make([]*barRecord, w.maxSize), cap: w.maxSize}
		w.buffers[key] = rb
	}
	rb.add(&barRecord{
		Open:   open,
		High:   high,
		Low:    low,
		Close:  closeV,
		Volume: volume,
		TsMs:   tsMs,
	})
	w.mu.Unlock()
}

// Snapshot returns a copy of the current window for a key, oldest first.
func (w *WindowBuffer) Snapshot(userID, symbol, period string, limit int) []*barRecord {
	key := fmt.Sprintf("%s/%s/%s", userID, symbol, period)

	w.mu.RLock()
	rb, ok := w.buffers[key]
	w.mu.RUnlock()
	if !ok {
		return nil
	}
	return rb.snapshot(limit)
}

// BootstrapSpec defines a bootstrap target.
type BootstrapSpec struct {
	UserID string
	Symbol   string
	Period   string
	Limit    int
}

// Bootstrap pre-allocates buffer entries for the given specs.
func (w *WindowBuffer) Bootstrap(ctx context.Context, specs []BootstrapSpec) {
	for _, spec := range specs {
		key := fmt.Sprintf("%s/%s/%s", spec.UserID, spec.Symbol, spec.Period)
		w.mu.Lock()
		if _, ok := w.buffers[key]; !ok {
			w.buffers[key] = &ringBuffer{bars: make([]*barRecord, w.maxSize), cap: w.maxSize}
		}
		w.mu.Unlock()
	}
	w.log.Info("bootstrap complete", zap.Int("specs", len(specs)))
}

func (rb *ringBuffer) add(b *barRecord) {
	rb.bars[rb.head] = b
	rb.head = (rb.head + 1) % rb.cap
	if rb.size < rb.cap {
		rb.size++
	}
}

func (rb *ringBuffer) snapshot(limit int) []*barRecord {
	if limit <= 0 || limit > rb.size {
		limit = rb.size
	}
	out := make([]*barRecord, 0, limit)
	for i := 0; i < limit; i++ {
		idx := (rb.head - rb.size + i) % rb.cap
		if idx < 0 {
			idx += rb.cap
		}
		out = append(out, rb.bars[idx])
	}
	return out
}
