package factor

import "sync/atomic"

var chanFullTotal atomic.Int64

func recordChanFull() { chanFullTotal.Add(1) }

// ChanFullTotal returns the bounded-channel-full drop count.
func ChanFullTotal() int64 { return chanFullTotal.Load() }
