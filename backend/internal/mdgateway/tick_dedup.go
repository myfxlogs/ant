package mdgateway

import (
	"encoding/binary"
	"sync"

	"github.com/cespare/xxhash/v2"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// TickDedup detects duplicate ticks within a sliding window to protect
// against broker re-sending historical quotes on reconnect (quirk Q-010).
// Hash key: ts || bid || ask || bid_volume || ask_volume (decision D-2).
type TickDedup struct {
	mu   sync.Mutex
	seen map[string]*ringHash // broker:canonical -> ring buffer
	size int                  // window size, default 100
}

type ringHash struct {
	hashes []uint64
	cur    int
	set    map[uint64]struct{}
}

// NewTickDedup creates a dedup with the given window size.
func NewTickDedup(size int) *TickDedup {
	if size <= 0 {
		size = 100
	}
	return &TickDedup{seen: make(map[string]*ringHash), size: size}
}

// Seen returns true if this tick appears to be a duplicate within the window.
func (d *TickDedup) Seen(t *mdtick.Tick) bool {
	key := t.Broker + ":" + t.Canonical

	d.mu.Lock()
	defer d.mu.Unlock()

	rh, ok := d.seen[key]
	if !ok {
		rh = &ringHash{
			hashes: make([]uint64, d.size),
			set:    make(map[uint64]struct{}),
		}
		d.seen[key] = rh
	}

	hash := tickHash(t)
	if _, exists := rh.set[hash]; exists {
		return true
	}

	// Evict oldest
	old := rh.hashes[rh.cur]
	delete(rh.set, old)

	rh.hashes[rh.cur] = hash
	rh.set[hash] = struct{}{}
	rh.cur = (rh.cur + 1) % d.size
	return false
}

func tickHash(t *mdtick.Tick) uint64 {
	var buf [40]byte
	binary.LittleEndian.PutUint64(buf[0:8], uint64(t.TsUnixMs))
	binary.LittleEndian.PutUint64(buf[8:16], uint64(t.ArrivedUnixMs))
	binary.LittleEndian.PutUint64(buf[16:24], uint64(t.BidVolume))
	binary.LittleEndian.PutUint64(buf[24:32], uint64(t.AskVolume))

	// Stringify bid/ask for hash stability
	bid := t.Bid.String()
	ask := t.Ask.String()
	h := xxhash.Sum64String(bid)
	h ^= xxhash.Sum64String(ask)
	h ^= xxhash.Sum64(buf[:])
	return h
}
